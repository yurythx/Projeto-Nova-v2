package application

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

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
	EventoSolicitacaoAvaliada  = "atendimento.solicitacao.avaliada"
	EventoSolicitacaoSegredo   = "atendimento.solicitacao.segredo_vinculado"
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
	Respostas         map[string]string
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
	// Avaliacao é a nota de quem abriu, quando já existe.
	Avaliacao *domain.Avaliacao `json:"avaliacao,omitempty"`
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
	sol.Respostas, err = domain.ValidarRespostas(o.Campos, in.Respostas)
	if err != nil {
		return domain.Solicitacao{}, MapError(err)
	}
	var enviadas []notificacoes.Enviada
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		sol, enviadas, err = s.abrirNa(ctx, tx, sol, identity, nil)
		return err
	})
	if err != nil {
		return domain.Solicitacao{}, MapError(err)
	}
	s.entregar(ctx, enviadas)
	return sol, nil
}

// abrirNa grava a abertura na transação de quem chama: política de prazo,
// regra da triagem, primeira passagem, evento, auditoria e aviso à
// equipe. ajustar (opcional) mexe na solicitação depois da regra e antes
// da política: a entrada de eventos usa para a severidade do alerta.
func (s *Service) abrirNa(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, identity auth.Identity, ajustar func(*domain.Solicitacao)) (domain.Solicitacao, []notificacoes.Enviada, error) {
	// O prazo começa na abertura. Sem regra, impacto e urgência ficam
	// médios. A primeira regra que casa ajusta os dois pela matriz.
	p, err := s.politicaDe(ctx, tx, sol.UnidadeID)
	if err != nil {
		return domain.Solicitacao{}, nil, err
	}
	regra, err := s.regraDaAbertura(ctx, tx, sol, identity)
	if err != nil {
		return domain.Solicitacao{}, nil, err
	}
	if regra != nil && regra.Prioridade != "" && p != nil {
		if impacto, urgencia, ok := p.Niveis(regra.Prioridade); ok {
			sol.Impacto, sol.Urgencia = impacto, urgencia
		}
	}
	if ajustar != nil {
		ajustar(&sol)
	}
	sol = sol.AplicarPolitica(p)
	if sol, err = s.repo.InsertSolicitacao(ctx, tx, sol); err != nil {
		return domain.Solicitacao{}, nil, err
	}
	if _, err = s.passar(ctx, tx, sol, nil, identity.UserID); err != nil {
		return domain.Solicitacao{}, nil, err
	}
	corpo := payloadDe(sol)
	if regra != nil {
		corpo["regra_id"] = regra.ID.String()
	}
	if err = s.registrar(ctx, tx, EventoSolicitacaoAberta, "solicitacao", sol.ID, corpo, sol, nil); err != nil {
		return domain.Solicitacao{}, nil, err
	}
	ids, err := s.destinosDaAbertura(ctx, tx, sol, regra)
	if err != nil {
		return domain.Solicitacao{}, nil, err
	}
	enviadas, err := s.avisar(ctx, tx, sol, ids, identity.UserID, "aberta",
		fmt.Sprintf("Nova solicitação #%d", sol.Numero), sol.OfertaNome+": "+sol.Titulo)
	return sol, enviadas, err
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
	mapa, err := s.mapaFila(ctx, s.pool)
	if err != nil {
		return nil, 0, err
	}
	// O Kanban move o cartão só para onde a máquina deixa: as passagens
	// vêm daqui, não de uma cópia da regra no navegador.
	for i := range lista {
		ps := papeis(identity, lista[i])
		lista[i] = lista[i].ComPassagens(extrasDaFila(lista[i], mapa))
		lista[i].PodeIrPara = lista[i].Proximos(ps...)
		ocultarSegredo(&lista[i], ps)
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
	ocultarSegredo(&sol, ps)
	visitas, err := s.repo.ListVisitas(ctx, s.pool, id)
	if err != nil {
		return Detalhe{}, MapError(err)
	}
	for i := range visitas {
		visitas[i] = visitas[i].SemCoordenadas()
	}
	mapa, err := s.mapaFila(ctx, s.pool)
	if err != nil {
		return Detalhe{}, MapError(err)
	}
	sol = sol.ComPassagens(extrasDaFila(sol, mapa))
	detalhe := Detalhe{Solicitacao: sol, Papeis: ps, Proximos: sol.Proximos(ps...), Passagens: passagens, Comentarios: comentarios, Visitas: visitas}
	avaliacao, err := s.repo.GetAvaliacao(ctx, s.pool, id)
	if err != nil && !errors.Is(err, domain.ErrNaoEncontrado) {
		return Detalhe{}, MapError(err)
	}
	if err == nil {
		detalhe.Avaliacao = &avaliacao
	}
	return detalhe, nil
}

// VincularSegredo grava o id de um segredo do cofre, ou limpa quando ele vem nulo.
// A senha não entra na solicitação nem na auditoria.
func (s *Service) VincularSegredo(ctx context.Context, identity auth.Identity, id uuid.UUID, segredo *uuid.UUID) (Detalhe, error) {
	err := s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, ps []string) ([]notificacoes.Enviada, error) {
		if !slices.Contains(ps, domain.PapelTecnico) {
			return nil, domain.ErrForaDoEscopo
		}
		if sol.Estado == domain.EstadoEncerrado || sol.Estado == domain.EstadoCancelado {
			return nil, domain.ErrSolicitacaoFinalizada
		}
		if segredo != nil {
			if s.segredos == nil {
				return nil, domain.ErrCofreDesligado
			}
			if _, err := s.segredos.Nome(ctx, identity, *segredo); err != nil {
				if errors.Is(err, domain.ErrCofreDesligado) {
					return nil, err
				}
				return nil, domain.ErrSegredoAusente
			}
		}
		agora := time.Now()
		if s.relogio != nil {
			agora = s.relogio()
		}
		if err := s.repo.SetSegredo(ctx, tx, sol.ID, segredo, agora); err != nil {
			return nil, err
		}
		payload := payloadDe(sol)
		if segredo != nil {
			payload["cofre_segredo_id"] = segredo.String()
		}
		if err := s.registrar(ctx, tx, EventoSolicitacaoSegredo, "solicitacao", sol.ID, payload, payload, nil); err != nil {
			return nil, err
		}
		return nil, nil
	})
	if err != nil {
		return Detalhe{}, err
	}
	return s.Obter(ctx, identity, id)
}

func extrasDaFila(sol domain.Solicitacao, mapa map[string][]string) []string {
	chaves := []string{sol.Estado}
	if sol.Estado == domain.EstadoResolvido || sol.Estado == domain.EstadoAtendido {
		chaves = append(chaves, "concluida")
	}
	var out []string
	for _, de := range chaves {
		for _, para := range mapa[de] {
			if para == "concluida" {
				out = append(out, domain.Conclusao(sol.Tipo))
				continue
			}
			out = append(out, para)
		}
	}
	return out
}

func ocultarSegredo(sol *domain.Solicitacao, ps []string) {
	if !slices.Contains(ps, domain.PapelTecnico) {
		sol.CofreSegredoID = nil
	}
}

// Transicionar muda o estado com a regra do tipo e o papel de quem age.
// Repetir o estado atual não grava nada.
func (s *Service) Transicionar(ctx context.Context, identity auth.Identity, id uuid.UUID, para string) (Detalhe, error) {
	err := s.alterar(ctx, identity, id, func(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, ps []string) ([]notificacoes.Enviada, error) {
		mapa, err := s.mapaFila(ctx, tx)
		if err != nil {
			return nil, err
		}
		sol = sol.ComPassagens(extrasDaFila(sol, mapa))
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
	enviadas, err := s.avisar(ctx, tx, next, destinos, identity.UserID, fmt.Sprintf("passagem:%d", p.ID),
		fmt.Sprintf("Solicitação #%d %s", next.Numero, rotulos[next.Estado]), next.Titulo)
	if err != nil || (next.Estado != domain.EstadoResolvido && next.Estado != domain.EstadoAtendido) {
		return enviadas, err
	}
	pesquisa, err := s.avisar(ctx, tx, next, []uuid.UUID{next.SolicitanteID}, uuid.Nil, "avaliar",
		fmt.Sprintf("Avalie o atendimento da solicitação #%d", next.Numero),
		"A nota fica registrada uma vez, na própria solicitação.")
	if err != nil {
		return nil, err
	}
	return append(enviadas, pesquisa...), nil
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

// Search alimenta a Busca Global. Devolve o que identity abriu e, se
// atende a fila, o que está no escopo dela. O texto livre da descrição
// não vai no resultado: a busca mostra número, título, oferta e estado.
// Chamado não é conteúdo público, então este módulo não publica no índice.
func (s *Service) Search(ctx context.Context, identity auth.Identity, q string, limit int) ([]domain.Solicitacao, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	if utf8.RuneCountInString(q) > 80 {
		q = string([]rune(q)[:80])
	}
	if limit < 1 || limit > 20 {
		limit = 8
	}
	quem := identity.UserID
	visto := map[uuid.UUID]struct{}{}
	var out []domain.Solicitacao
	juntar := func(lista []domain.Solicitacao) {
		for _, item := range lista {
			if _, ok := visto[item.ID]; ok {
				continue
			}
			visto[item.ID] = struct{}{}
			out = append(out, item)
			if len(out) == limit {
				return
			}
		}
	}
	minhas, _, err := s.repo.ListSolicitacoes(ctx, s.pool, domain.FiltroSolicitacao{
		Solicitante: &quem, Texto: q, Finalizadas: true, Limit: limit,
	})
	if err != nil {
		return nil, err
	}
	juntar(minhas)
	if len(out) == limit {
		return out, nil
	}
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	if !cob.All && len(cob.Unidades) == 0 {
		return out, nil
	}
	fila, _, err := s.repo.ListSolicitacoes(ctx, s.pool, domain.FiltroSolicitacao{
		Restrito: !cob.All, Unidades: cob.Unidades, Texto: q, Finalizadas: true, Limit: limit,
	})
	if err != nil {
		return nil, err
	}
	juntar(fila)
	return out, nil
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
