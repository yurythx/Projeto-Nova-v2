package application

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/domain/pagination"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
)

const (
	EventoSolicitacaoAberta    = "atendimento.solicitacao.aberta"
	EventoSolicitacaoEstado    = "atendimento.solicitacao.estado_alterado"
	EventoSolicitacaoAtribuida = "atendimento.solicitacao.atribuida"
	EventoSolicitacaoComentada = "atendimento.solicitacao.comentada"
)

// Recortes de responsável da fila.
const (
	ResponsavelEu     = "eu"
	ResponsavelNenhum = "nenhum"
)

// rotulos é o nome do estado nos avisos.
var rotulos = map[string]string{
	domain.EstadoAberto:             "aberta",
	domain.EstadoEmAtendimento:      "em atendimento",
	domain.EstadoAguardandoServidor: "aguardando a sua resposta",
	domain.EstadoAguardandoTerceiro: "aguardando terceiro",
	domain.EstadoResolvido:          "resolvida",
	domain.EstadoAtendido:           "atendida",
	domain.EstadoEncerrado:          "encerrada",
	domain.EstadoCancelado:          "cancelada",
}

// AberturaEntrada é o pedido do servidor.
type AberturaEntrada struct {
	OfertaID          uuid.UUID
	Titulo, Descricao string
}

// FiltroFila são os recortes da lista. Responsavel aceita "eu", "nenhum"
// ou o id de alguém.
type FiltroFila struct {
	Tipo, Estado, Responsavel string
	Finalizadas               bool
}

// Detalhe é a solicitação vista por quem pergunta: o que ela pode fazer
// agora e a conversa que lhe cabe ver.
type Detalhe struct {
	Solicitacao domain.Solicitacao  `json:"solicitacao"`
	Papeis      []string            `json:"papeis"`
	Proximos    []string            `json:"proximos"`
	Passagens   []domain.Passagem   `json:"passagens"`
	Comentarios []domain.Comentario `json:"comentarios"`
	// Visitas vêm sem coordenadas: a posição só aparece na supervisão de
	// visitas em andamento.
	Visitas []domain.Visita `json:"visitas"`
}

// papeis devolve como identity age sobre a solicitação. Vazio = não vê.
func papeis(identity auth.Identity, sol domain.Solicitacao) []string {
	out := []string{}
	if auth.Can(identity, auth.PermAtendimentoAtender, domain.Dona(sol.UnidadeID)) {
		out = append(out, domain.PapelTecnico)
	}
	if identity.UserID != uuid.Nil && identity.UserID == sol.SolicitanteID {
		out = append(out, domain.PapelSolicitante)
	}
	return out
}

// Catalogo devolve as ofertas que o servidor pode pedir.
func (s *Service) Catalogo(ctx context.Context) ([]domain.Oferta, error) {
	return s.repo.ListOfertasAtivas(ctx, s.pool)
}

// Abrir registra a solicitação pela oferta, a primeira passagem, o evento
// e a auditoria na mesma transação, e avisa a equipe da unidade.
func (s *Service) Abrir(ctx context.Context, identity auth.Identity, in AberturaEntrada) (domain.Solicitacao, error) {
	o, err := s.repo.GetOferta(ctx, s.pool, in.OfertaID)
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return domain.Solicitacao{}, MapError(domain.ErrOfertaInexistente)
	}
	if err != nil {
		return domain.Solicitacao{}, MapError(err)
	}
	sol, err := domain.NovaSolicitacao(o, in.Titulo, in.Descricao, identity.UserID)
	if err != nil {
		return domain.Solicitacao{}, MapError(err)
	}
	var enviadas []notificacoes.Enviada
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		// O prazo começa na abertura, com impacto e urgência médios; a
		// equipe reclassifica na triagem.
		p, err := s.politicaDe(ctx, tx, sol.UnidadeID)
		if err != nil {
			return err
		}
		sol = sol.AplicarPolitica(p)
		if sol, err = s.repo.InsertSolicitacao(ctx, tx, sol); err != nil {
			return err
		}
		if _, err = s.passar(ctx, tx, sol, nil, identity.UserID); err != nil {
			return err
		}
		if err = s.registrar(ctx, tx, EventoSolicitacaoAberta, "solicitacao", sol.ID, payloadDe(sol), sol, nil); err != nil {
			return err
		}
		equipe, err := s.repo.Responsaveis(ctx, tx, sol.UnidadeID)
		if err != nil {
			return err
		}
		ids := make([]uuid.UUID, 0, len(equipe))
		for _, c := range equipe {
			ids = append(ids, c.ID)
		}
		enviadas, err = s.avisar(ctx, tx, sol, ids, identity.UserID, "aberta",
			fmt.Sprintf("Nova solicitação #%d", sol.Numero), sol.OfertaNome+": "+sol.Titulo)
		return err
	})
	if err != nil {
		return domain.Solicitacao{}, MapError(err)
	}
	s.entregar(ctx, enviadas)
	return sol, nil
}

// ListarMinhas devolve as solicitações que identity abriu.
func (s *Service) ListarMinhas(ctx context.Context, identity auth.Identity, f FiltroFila, p pagination.Params) ([]domain.Solicitacao, int64, error) {
	quem := identity.UserID
	return s.repo.ListSolicitacoes(ctx, s.pool, domain.FiltroSolicitacao{
		Solicitante: &quem, Tipo: f.Tipo, Estado: f.Estado, Finalizadas: f.Finalizadas,
		Limit: p.Limit(), Offset: p.Offset(),
	})
}

// ListarFila devolve as solicitações das unidades que identity atende.
func (s *Service) ListarFila(ctx context.Context, identity auth.Identity, f FiltroFila, p pagination.Params) ([]domain.Solicitacao, int64, error) {
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	if !cob.All && len(cob.Unidades) == 0 {
		return []domain.Solicitacao{}, 0, nil
	}
	filtro := domain.FiltroSolicitacao{
		Restrito: !cob.All, Unidades: cob.Unidades, Tipo: f.Tipo, Estado: f.Estado, Finalizadas: f.Finalizadas,
		Limit: p.Limit(), Offset: p.Offset(),
	}
	switch f.Responsavel {
	case "":
	case ResponsavelEu:
		quem := identity.UserID
		filtro.Responsavel = &quem
	case ResponsavelNenhum:
		filtro.SemDono = true
	default:
		id, err := uuid.Parse(f.Responsavel)
		if err != nil {
			return nil, 0, MapError(domain.ErrEntradaInvalida)
		}
		filtro.Responsavel = &id
	}
	lista, total, err := s.repo.ListSolicitacoes(ctx, s.pool, filtro)
	if err != nil {
		return nil, 0, err
	}
	// O Kanban move o cartão só para onde a máquina deixa: as passagens
	// vêm daqui, não de uma cópia da regra no navegador.
	for i := range lista {
		lista[i].PodeIrPara = lista[i].Proximos(papeis(identity, lista[i])...)
	}
	if err := s.medir(ctx, lista); err != nil {
		return nil, 0, err
	}
	return lista, total, nil
}

// Obter devolve a solicitação para quem a abriu ou a atende. Para o
// solicitante, as notas internas não existem.
func (s *Service) Obter(ctx context.Context, identity auth.Identity, id uuid.UUID) (Detalhe, error) {
	sol, err := s.repo.GetSolicitacao(ctx, s.pool, id)
	if err != nil {
		return Detalhe{}, MapError(err)
	}
	ps := papeis(identity, sol)
	if len(ps) == 0 {
		return Detalhe{}, MapError(domain.ErrNaoEncontrado)
	}
	passagens, err := s.repo.ListPassagens(ctx, s.pool, id)
	if err != nil {
		return Detalhe{}, MapError(err)
	}
	comentarios, err := s.repo.ListComentarios(ctx, s.pool, id, slices.Contains(ps, domain.PapelTecnico))
	if err != nil {
		return Detalhe{}, MapError(err)
	}
	// O prazo é instrumento da equipe; quem abriu acompanha pelo estado.
	if slices.Contains(ps, domain.PapelTecnico) {
		um := []domain.Solicitacao{sol}
		if err := s.medir(ctx, um); err != nil {
			return Detalhe{}, MapError(err)
		}
		sol = um[0]
	}
	visitas, err := s.repo.ListVisitas(ctx, s.pool, id)
	if err != nil {
		return Detalhe{}, MapError(err)
	}
	for i := range visitas {
		visitas[i] = visitas[i].SemCoordenadas()
	}
	return Detalhe{Solicitacao: sol, Papeis: ps, Proximos: sol.Proximos(ps...), Passagens: passagens, Comentarios: comentarios, Visitas: visitas}, nil
}

// Transicionar muda o estado com a regra do tipo e o papel de quem age.
// Repetir o estado atual não grava nada.
func (s *Service) Transicionar(ctx context.Context, identity auth.Identity, id uuid.UUID, para string) (Detalhe, error) {
	err := s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, ps []string) ([]notificacoes.Enviada, error) {
		next, mudou, err := sol.Transicionar(para, ps...)
		if err != nil || !mudou {
			return nil, err
		}
		return s.gravarPassagem(ctx, tx, identity, sol, next, true)
	})
	if err != nil {
		return Detalhe{}, err
	}
	return s.Obter(ctx, identity, id)
}

// Atribuir define o responsável, que precisa ser membro de uma equipe da
// unidade. A passagem de aberta para em atendimento, quando houver, fica
// registrada como as outras.
func (s *Service) Atribuir(ctx context.Context, identity auth.Identity, id, responsavel uuid.UUID) (Detalhe, error) {
	err := s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, ps []string) ([]notificacoes.Enviada, error) {
		if !slices.Contains(ps, domain.PapelTecnico) {
			return nil, domain.ErrForaDoEscopo
		}
		next, mudou, err := sol.Atribuir(responsavel)
		if err != nil || !mudou {
			return nil, err
		}
		ok, err := s.repo.ResponsavelValido(ctx, tx, responsavel, sol.UnidadeID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, domain.ErrResponsavelInvalido
		}
		if err := s.repo.UpdateSolicitacao(ctx, tx, next); err != nil {
			return nil, err
		}
		if next.Estado != sol.Estado {
			de := sol.Estado
			if _, err := s.passar(ctx, tx, next, &de, identity.UserID); err != nil {
				return nil, err
			}
		}
		if err := s.registrar(ctx, tx, EventoSolicitacaoAtribuida, "solicitacao", next.ID, payloadDe(next), next, sol); err != nil {
			return nil, err
		}
		return s.avisar(ctx, tx, next, []uuid.UUID{responsavel}, identity.UserID, "atribuida:"+uuid.NewString(),
			fmt.Sprintf("Solicitação #%d atribuída a você", next.Numero), next.OfertaNome+": "+next.Titulo)
	})
	if err != nil {
		return Detalhe{}, err
	}
	return s.Obter(ctx, identity, id)
}

// Comentar grava a resposta ou a nota interna. A resposta de quem abriu,
// quando a equipe esperava por ela, devolve a solicitação à equipe.
func (s *Service) Comentar(ctx context.Context, identity auth.Identity, id uuid.UUID, texto string, interno bool) (domain.Comentario, error) {
	var c domain.Comentario
	err := s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, ps []string) ([]notificacoes.Enviada, error) {
		novo, err := domain.NovoComentario(sol, identity.UserID, texto, interno, slices.Contains(ps, domain.PapelTecnico))
		if err != nil {
			return nil, err
		}
		if c, err = s.repo.InsertComentario(ctx, tx, novo); err != nil {
			return nil, err
		}
		payload := payloadDe(sol)
		payload["comentario_id"], payload["interno"] = c.ID.String(), c.Interno
		if err := s.registrar(ctx, tx, EventoSolicitacaoComentada, "solicitacao", sol.ID, payload, c, nil); err != nil {
			return nil, err
		}
		// A volta à equipe é avisada pela própria resposta, não à parte.
		if !interno && slices.Contains(ps, domain.PapelSolicitante) {
			if next, mudou := sol.RetomarPorResposta(); mudou {
				if _, err = s.gravarPassagem(ctx, tx, identity, sol, next, false); err != nil {
					return nil, err
				}
				sol = next
			}
		}
		destinos := []uuid.UUID{sol.SolicitanteID}
		if interno {
			destinos = nil
		}
		if sol.ResponsavelID != nil {
			destinos = append(destinos, *sol.ResponsavelID)
		}
		titulo := fmt.Sprintf("Nova resposta na solicitação #%d", sol.Numero)
		if interno {
			titulo = fmt.Sprintf("Nova nota interna na solicitação #%d", sol.Numero)
		}
		return s.avisar(ctx, tx, sol, destinos, identity.UserID, "comentario:"+c.ID.String(), titulo, sol.Titulo)
	})
	if err != nil {
		return domain.Comentario{}, err
	}
	return c, nil
}

// Responsaveis lista quem pode assumir a solicitação.
func (s *Service) Responsaveis(ctx context.Context, identity auth.Identity, id uuid.UUID) ([]domain.Candidato, error) {
	sol, err := s.repo.GetSolicitacao(ctx, s.pool, id)
	if err != nil {
		return nil, MapError(err)
	}
	ps := papeis(identity, sol)
	if len(ps) == 0 {
		return nil, MapError(domain.ErrNaoEncontrado)
	}
	if !slices.Contains(ps, domain.PapelTecnico) {
		return nil, MapError(domain.ErrForaDoEscopo)
	}
	lista, err := s.repo.Responsaveis(ctx, s.pool, sol.UnidadeID)
	return lista, MapError(err)
}

type mudanca func(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, papeis []string) ([]notificacoes.Enviada, error)

// alterar trava a solicitação, confere quem a vê e aplica a mudança na
// mesma transação; os avisos saem depois do commit.
func (s *Service) alterar(ctx context.Context, identity auth.Identity, id uuid.UUID, fn mudanca) error {
	var enviadas []notificacoes.Enviada
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.TravarSolicitacao(ctx, tx, id); err != nil {
			return err
		}
		sol, err := s.repo.GetSolicitacao(ctx, tx, id)
		if err != nil {
			return err
		}
		ps := papeis(identity, sol)
		if len(ps) == 0 {
			return domain.ErrNaoEncontrado
		}
		enviadas, err = fn(ctx, tx, sol, ps)
		return err
	})
	if err != nil {
		return MapError(err)
	}
	s.entregar(ctx, enviadas)
	return nil
}

// gravarPassagem persiste a mudança de estado de prev para next, com a
// passagem, o evento, a auditoria e, com aviso, o aviso às duas pontas.
func (s *Service) gravarPassagem(ctx context.Context, tx pgx.Tx, identity auth.Identity, prev, next domain.Solicitacao, aviso bool) ([]notificacoes.Enviada, error) {
	if err := s.repo.UpdateSolicitacao(ctx, tx, next); err != nil {
		return nil, err
	}
	de := prev.Estado
	p, err := s.passar(ctx, tx, next, &de, identity.UserID)
	if err != nil {
		return nil, err
	}
	payload := payloadDe(next)
	payload["de"] = de
	if err := s.registrar(ctx, tx, EventoSolicitacaoEstado, "solicitacao", next.ID, payload, next, prev); err != nil {
		return nil, err
	}
	if !aviso {
		return nil, nil
	}
	destinos := []uuid.UUID{next.SolicitanteID}
	if next.ResponsavelID != nil {
		destinos = append(destinos, *next.ResponsavelID)
	}
	return s.avisar(ctx, tx, next, destinos, identity.UserID, fmt.Sprintf("passagem:%d", p.ID),
		fmt.Sprintf("Solicitação #%d %s", next.Numero, rotulos[next.Estado]), next.Titulo)
}

func (s *Service) passar(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, de *string, autor uuid.UUID) (domain.Passagem, error) {
	return s.repo.InsertPassagem(ctx, tx, domain.Passagem{
		SolicitacaoID: sol.ID, De: de, Para: sol.Estado, AutorID: &autor, Em: sol.UpdatedAt,
	})
}

// avisar grava o aviso na transação para os destinos, menos quem agiu.
// A chave inclui a solicitação e o fato: a mesma mudança não avisa duas
// vezes.
func (s *Service) avisar(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, destinos []uuid.UUID, autor uuid.UUID, fato, titulo, mensagem string) ([]notificacoes.Enviada, error) {
	if s.avisos == nil {
		return nil, nil
	}
	alvo := make([]uuid.UUID, 0, len(destinos))
	for _, id := range destinos {
		if id != autor && id != uuid.Nil && !slices.Contains(alvo, id) {
			alvo = append(alvo, id)
		}
	}
	// Título e mensagem cabem nos limites do aviso: o maior é o nome da
	// oferta mais o título da solicitação (2 × 160 caracteres).
	return notificacoes.Registrar(ctx, tx, alvo, notificacoes.Nova{
		Modulo: "atendimento", Titulo: titulo, Mensagem: mensagem,
		Link: "/atendimento/solicitacoes/" + sol.ID.String(), Chave: "atendimento:" + sol.ID.String() + ":" + fato,
	})
}

func (s *Service) entregar(ctx context.Context, enviadas []notificacoes.Enviada) {
	if s.avisos != nil && len(enviadas) > 0 {
		s.avisos.Entregar(ctx, enviadas)
	}
}

// payloadDe é o que o barramento precisa para avisar as telas abertas sem
// reler a solicitação (HandleEvento).
func payloadDe(sol domain.Solicitacao) map[string]any {
	p := map[string]any{
		"id": sol.ID.String(), "numero": sol.Numero, "tipo": sol.Tipo, "estado": sol.Estado,
		"solicitante_id": sol.SolicitanteID.String(),
	}
	if sol.UnidadeID != nil {
		p["unidade_id"] = sol.UnidadeID.String()
	}
	return p
}
