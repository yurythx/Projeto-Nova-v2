package application

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

const tetoRegras = 30

func fusoBrasilia() *time.Location {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		return time.UTC
	}
	return loc
}

func unidadesDo(identity auth.Identity) []uuid.UUID {
	vistos := map[uuid.UUID]bool{}
	var ids []uuid.UUID
	for _, sc := range identity.Scopes {
		if sc.UnidadeID == nil || vistos[*sc.UnidadeID] {
			continue
		}
		vistos[*sc.UnidadeID] = true
		ids = append(ids, *sc.UnidadeID)
	}
	return ids
}

func (s *Service) regraDaAbertura(ctx context.Context, db database.DBTX, sol domain.Solicitacao, identity auth.Identity) (*domain.Regra, error) {
	regras, err := s.repo.ListarRegrasDaUnidade(ctx, db, sol.UnidadeID)
	if err != nil {
		return nil, err
	}
	return domain.Primeira(regras, sol.OfertaID, sol.Titulo, unidadesDo(identity)), nil
}

func (s *Service) destinosDaAbertura(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, regra *domain.Regra) ([]uuid.UUID, error) {
	if regra != nil && regra.EquipeID != nil {
		eq, err := s.repo.GetEquipe(ctx, tx, *regra.EquipeID)
		if err != nil {
			return nil, err
		}
		if len(eq.Membros) > 0 {
			return eq.Membros, nil
		}
	}
	equipe, err := s.repo.Responsaveis(ctx, tx, sol.UnidadeID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(equipe))
	for _, c := range equipe {
		ids = append(ids, c.ID)
	}
	return ids, nil
}

func (s *Service) conferirAlvos(ctx context.Context, unidade *uuid.UUID, ofertaID, equipeID *uuid.UUID) error {
	if ofertaID != nil {
		oferta, err := s.repo.GetOferta(ctx, s.pool, *ofertaID)
		if err != nil {
			return err
		}
		if !domain.MesmaUnidade(oferta.UnidadeID, unidade) {
			return domain.Recusa{"a oferta precisa ser da mesma unidade da regra"}
		}
	}
	if equipeID != nil {
		equipe, err := s.repo.GetEquipe(ctx, s.pool, *equipeID)
		if err != nil {
			return err
		}
		if !domain.MesmaUnidade(equipe.UnidadeID, unidade) {
			return domain.Recusa{"a equipe precisa ser da mesma unidade da regra"}
		}
	}
	return nil
}

// ListarRegras devolve a triagem do escopo de quem gerencia.
func (s *Service) ListarRegras(ctx context.Context, identity auth.Identity) ([]domain.Regra, error) {
	f, ok := filtro(identity)
	if !ok {
		return []domain.Regra{}, nil
	}
	lista, err := s.repo.ListarRegras(ctx, s.pool, f)
	return lista, MapError(err)
}

// CriarRegra grava a triagem. Trinta regras por unidade.
func (s *Service) CriarRegra(ctx context.Context, identity auth.Identity, in domain.Regra) (domain.Regra, error) {
	if !ver(identity, in.UnidadeID) {
		return domain.Regra{}, MapError(domain.ErrForaDoEscopo)
	}
	regra, err := domain.NovaRegra(in)
	if err != nil {
		return domain.Regra{}, MapError(err)
	}
	regra.UnidadeID = in.UnidadeID
	if err := s.conferirAlvos(ctx, regra.UnidadeID, regra.OfertaID, regra.EquipeID); err != nil {
		return domain.Regra{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		n, err := s.repo.ContarRegras(ctx, tx, regra.UnidadeID)
		if err != nil {
			return err
		}
		if n >= tetoRegras {
			return domain.Recusa{"esta unidade já tem 30 regras"}
		}
		if err := s.repo.InserirRegra(ctx, tx, regra); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "atendimento.regra.criada", "atendimento_regra", regra.ID, map[string]any{"id": regra.ID.String()}, regra, nil)
	})
	return regra, MapError(err)
}

// ExcluirRegra tira a regra do escopo.
func (s *Service) ExcluirRegra(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		atual, err := s.repo.ObterRegra(ctx, tx, id)
		if err != nil {
			return err
		}
		if !ver(identity, atual.UnidadeID) {
			return domain.ErrNaoEncontrado
		}
		if err := s.repo.ExcluirRegra(ctx, tx, id); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "atendimento.regra.excluida", "atendimento_regra", id, map[string]any{"id": id.String()}, nil, atual)
	}))
}

// ObterResposta devolve o texto da oferta. Vazio quando ninguém escreveu.
func (s *Service) ObterResposta(ctx context.Context, identity auth.Identity, oferta uuid.UUID) (domain.RespostaPronta, error) {
	o, err := s.repo.GetOferta(ctx, s.pool, oferta)
	if err != nil {
		return domain.RespostaPronta{}, MapError(err)
	}
	alvo := domain.Dona(o.UnidadeID)
	if !auth.Can(identity, auth.PermAtendimentoAtender, alvo) && !auth.Can(identity, auth.PermAtendimentoManage, alvo) {
		return domain.RespostaPronta{}, MapError(domain.ErrNaoEncontrado)
	}
	item, err := s.repo.ObterResposta(ctx, s.pool, oferta)
	return item, MapError(err)
}

// SalvarResposta grava o texto que a equipe cola. Os dois vazios apagam.
func (s *Service) SalvarResposta(ctx context.Context, identity auth.Identity, oferta uuid.UUID, publico, interna string) (domain.RespostaPronta, error) {
	o, err := s.repo.GetOferta(ctx, s.pool, oferta)
	if err != nil {
		return domain.RespostaPronta{}, MapError(err)
	}
	if !ver(identity, o.UnidadeID) {
		return domain.RespostaPronta{}, MapError(domain.ErrForaDoEscopo)
	}
	publico, interna = strings.TrimSpace(publico), strings.TrimSpace(interna)
	if publico == "" && interna == "" {
		err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
			if err := s.repo.ExcluirResposta(ctx, tx, oferta); err != nil {
				return err
			}
			return s.registrar(ctx, tx, "atendimento.resposta.excluida", "atendimento_resposta", oferta, map[string]any{"oferta_id": oferta.String()}, nil, nil)
		})
		return domain.RespostaPronta{OfertaID: oferta}, MapError(err)
	}
	item, err := domain.NovaResposta(oferta, publico, interna)
	if err != nil {
		return domain.RespostaPronta{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.SalvarResposta(ctx, tx, item); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "atendimento.resposta.salva", "atendimento_resposta", oferta, map[string]any{"oferta_id": oferta.String()}, item, nil)
	})
	return item, MapError(err)
}

// ListarRecorrencias devolve os chamados mensais do escopo.
func (s *Service) ListarRecorrencias(ctx context.Context, identity auth.Identity) ([]domain.ChamadoMensal, error) {
	f, ok := filtro(identity)
	if !ok {
		return []domain.ChamadoMensal{}, nil
	}
	lista, err := s.repo.ListarRecorrencias(ctx, s.pool, f)
	return lista, MapError(err)
}

// CriarRecorrencia marca o chamado que abre todo mês, no dia pedido.
func (s *Service) CriarRecorrencia(ctx context.Context, identity auth.Identity, in domain.ChamadoMensal) (domain.ChamadoMensal, error) {
	oferta, err := s.repo.GetOferta(ctx, s.pool, in.OfertaID)
	if err != nil {
		return domain.ChamadoMensal{}, MapError(err)
	}
	in.UnidadeID = oferta.UnidadeID
	in.CriadoPor = identity.UserID
	if !ver(identity, in.UnidadeID) {
		return domain.ChamadoMensal{}, MapError(domain.ErrForaDoEscopo)
	}
	item, err := domain.NovoChamadoMensal(in)
	if err != nil {
		return domain.ChamadoMensal{}, MapError(err)
	}
	item.UnidadeID = in.UnidadeID
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.InserirRecorrencia(ctx, tx, item); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "atendimento.recorrencia.criada", "atendimento_recorrencia", item.ID, map[string]any{"id": item.ID.String()}, item, nil)
	})
	return item, MapError(err)
}

// ExcluirRecorrencia para os próximos meses. O chamado já aberto fica.
func (s *Service) ExcluirRecorrencia(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		atual, err := s.repo.ObterRecorrencia(ctx, tx, id)
		if err != nil {
			return err
		}
		if !ver(identity, atual.UnidadeID) {
			return domain.ErrNaoEncontrado
		}
		if err := s.repo.ExcluirRecorrencia(ctx, tx, id); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "atendimento.recorrencia.excluida", "atendimento_recorrencia", id, map[string]any{"id": id.String()}, nil, atual)
	}))
}

// RegistrarDeflexao anota que a pessoa viu o artigo. Resolver marca que ela dispensou o pedido.
func (s *Service) RegistrarDeflexao(ctx context.Context, identity auth.Identity, artigos []string, resolveu bool) error {
	if identity.UserID == uuid.Nil {
		return MapError(domain.ErrSolicitanteInvalido)
	}
	if len(artigos) == 0 || len(artigos) > 4 {
		return MapError(domain.Recusa{"informe até 4 artigos"})
	}
	dia := s.agora().In(fusoBrasilia()).Format(time.DateOnly)
	var unidade *uuid.UUID
	if ids := unidadesDo(identity); len(ids) > 0 {
		unidade = &ids[0]
	}
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		vistos := map[string]bool{}
		for _, artigo := range artigos {
			artigo = strings.TrimSpace(artigo)
			if artigo == "" || utf8.RuneCountInString(artigo) > 80 || vistos[artigo] {
				if utf8.RuneCountInString(artigo) > 80 {
					return domain.Recusa{"o artigo cabe em 80 caracteres"}
				}
				continue
			}
			vistos[artigo] = true
			if err := s.repo.RegistrarDeflexao(ctx, tx, identity.UserID, artigo, dia, unidade, resolveu); err != nil {
				return err
			}
		}
		if len(vistos) == 0 {
			return domain.Recusa{"informe o artigo"}
		}
		return nil
	}))
}

// RunRecorrencias abre, uma vez por mês, os chamados marcados.
func (s *Service) RunRecorrencias(ctx context.Context) error {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		if err := s.AbrirRecorrencias(ctx); err != nil && ctx.Err() == nil {
			s.logger.WarnContext(ctx, "atendimento: recorrência falhou", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// AbrirRecorrencias percorre as marcas ativas. O mês já aberto não abre de novo.
func (s *Service) AbrirRecorrencias(ctx context.Context) error {
	lista, err := s.repo.ListarRecorrenciasAtivas(ctx, s.pool)
	if err != nil {
		return err
	}
	for _, item := range lista {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := s.abrirRecorrencia(ctx, item); err != nil && ctx.Err() == nil {
			s.logger.WarnContext(ctx, "atendimento: recorrência não abriu", slog.String("id", item.ID.String()), slog.Any("error", err))
		}
	}
	return nil
}

func (s *Service) abrirRecorrencia(ctx context.Context, item domain.ChamadoMensal) error {
	loc := s.fusoDa(ctx, item.UnidadeID)
	agora := s.agora().In(loc)
	if agora.Day() < item.Dia {
		return nil
	}
	ok, err := s.repo.ReivindicarAbertura(ctx, s.pool, item.ID, agora.Year(), int(agora.Month()))
	if err != nil || !ok {
		return err
	}
	inicio := time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, loc)
	existente, err := s.repo.SolicitacaoDoMes(ctx, s.pool, item.OfertaID, item.CriadoPor, item.Titulo, inicio)
	if err != nil {
		return err
	}
	if existente != uuid.Nil {
		return s.repo.LigarAbertura(ctx, s.pool, item.ID, agora.Year(), int(agora.Month()), existente)
	}
	sol, err := s.Abrir(ctx, auth.Identity{UserID: item.CriadoPor, Source: auth.SourceLocal}, AberturaEntrada{
		OfertaID: item.OfertaID, Titulo: item.Titulo, Descricao: item.Descricao,
	})
	if err != nil {
		return err
	}
	return s.repo.LigarAbertura(ctx, s.pool, item.ID, agora.Year(), int(agora.Month()), sol.ID)
}

func (s *Service) fusoDa(ctx context.Context, unidade *uuid.UUID) *time.Location {
	nome := "America/Sao_Paulo"
	p, err := s.repo.ResolverPolitica(ctx, s.pool, unidade)
	if err == nil && p.Fuso != "" {
		nome = p.Fuso
	}
	loc, err := time.LoadLocation(nome)
	if err != nil {
		return fusoBrasilia()
	}
	return loc
}
