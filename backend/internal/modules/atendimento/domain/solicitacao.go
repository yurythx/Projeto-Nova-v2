package domain

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	TipoIncidente  = "incidente"
	TipoRequisicao = "requisicao"

	// Papel de quem age sobre a solicitação. A mesma pessoa pode ter os
	// dois: o técnico que abre um pedido para si.
	PapelTecnico     = "tecnico"
	PapelSolicitante = "solicitante"

	// EstadoAguardandoAprovacao: requisição de oferta com etapas, antes de
	// chegar à fila (ADR 068). A equipe vê, mas não atende.
	EstadoAguardandoAprovacao = "aguardando_aprovacao"
	EstadoAberto              = "aberto"
	EstadoEmAtendimento       = "em_atendimento"
	EstadoAguardandoServidor  = "aguardando_servidor"
	EstadoAguardandoTerceiro  = "aguardando_terceiro"
	EstadoResolvido           = "resolvido" // incidente: o serviço voltou
	EstadoAtendido            = "atendido"  // requisição: o pedido foi cumprido
	EstadoEncerrado           = "encerrado"
	EstadoCancelado           = "cancelado"
)

// Solicitação é o pedido do servidor. Incidente e requisição não
// compartilham a máquina de estado: a conclusão de um é "resolvido", a do
// outro é "atendido". Os nomes de leitura (oferta, pessoas) vêm da
// consulta e não são gravados.
type Solicitacao struct {
	ID            uuid.UUID  `json:"id"`
	Numero        int64      `json:"numero"`
	Tipo          string     `json:"tipo"`
	Estado        string     `json:"estado"`
	Titulo        string     `json:"titulo"`
	Descricao     string     `json:"descricao"`
	OfertaID      uuid.UUID  `json:"oferta_id"`
	ProdutoID     *uuid.UUID `json:"produto_id,omitempty"`
	UnidadeID     *uuid.UUID `json:"unidade_id,omitempty"`
	SolicitanteID uuid.UUID  `json:"solicitante_id"`
	ResponsavelID *uuid.UUID `json:"responsavel_id,omitempty"`
	ResolvidaEm   *time.Time `json:"resolvida_em,omitempty"`
	// Classificação e o compromisso de prazo do momento em que foi feita.
	Impacto         int               `json:"impacto"`
	Urgencia        int               `json:"urgencia"`
	Prioridade      *string           `json:"prioridade,omitempty"`
	MetaRespostaMin *int              `json:"meta_resposta_min,omitempty"`
	MetaSolucaoMin  *int              `json:"meta_solucao_min,omitempty"`
	EncerradaEm     *time.Time        `json:"encerrada_em,omitempty"`
	Respostas       map[string]string `json:"respostas,omitempty"`
	// CofreSegredoID aponta para um segredo. A senha não fica nesta linha.
	CofreSegredoID *uuid.UUID `json:"cofre_segredo_id,omitempty"`
	// EtapaAtual é a etapa de aprovação em curso; só existe aguardando
	// aprovação (ADR 068).
	EtapaAtual *int      `json:"etapa_atual,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	OfertaNome      string `json:"oferta_nome"`
	SolicitanteNome string `json:"solicitante_nome"`
	ResponsavelNome string `json:"responsavel_nome,omitempty"`

	// PodeIrPara são as passagens de quem consulta a fila; vazio fora dela.
	PodeIrPara []string `json:"proximos,omitempty"`
	// passagens são caminhos acrescentados na fila. Quem atende conduz.
	passagens []string
	// Prazo é medido na leitura, só para a equipe.
	Prazo *Prazo `json:"prazo,omitempty"`
}

// Passagem é uma mudança de estado com o horário. De vazio = abertura.
type Passagem struct {
	ID            int64      `json:"id"`
	SolicitacaoID uuid.UUID  `json:"solicitacao_id"`
	De            *string    `json:"de,omitempty"`
	Para          string     `json:"para"`
	AutorID       *uuid.UUID `json:"autor_id,omitempty"`
	AutorNome     string     `json:"autor_nome"`
	Em            time.Time  `json:"em"`
}

// Comentario é a resposta ao servidor ou, com Interno, a nota da equipe.
type Comentario struct {
	ID            uuid.UUID  `json:"id"`
	SolicitacaoID uuid.UUID  `json:"solicitacao_id"`
	AutorID       *uuid.UUID `json:"autor_id,omitempty"`
	AutorNome     string     `json:"autor_nome"`
	Texto         string     `json:"texto"`
	Interno       bool       `json:"interno"`
	CreatedAt     time.Time  `json:"created_at"`
}

// Conclusao é o estado em que a prática considera o trabalho feito.
func Conclusao(tipo string) string {
	if tipo == TipoIncidente {
		return EstadoResolvido
	}
	return EstadoAtendido
}

// transicoes devolve, por papel e estado atual, os destinos possíveis.
// O solicitante cancela enquanto ninguém começou, e confirma ou reabre o
// que a equipe concluiu. Sair de aguardando_servidor é efeito da resposta
// dele (RetomarPorResposta), não um botão.
func transicoes(tipo, papel string) map[string][]string {
	if !TipoValido(tipo) {
		return nil
	}
	fim := Conclusao(tipo)
	switch papel {
	case PapelTecnico:
		return map[string][]string{
			EstadoAguardandoAprovacao: {EstadoCancelado},
			EstadoAberto:              {EstadoEmAtendimento, EstadoCancelado},
			EstadoEmAtendimento:       {EstadoAguardandoServidor, EstadoAguardandoTerceiro, fim, EstadoCancelado},
			EstadoAguardandoServidor:  {EstadoEmAtendimento, EstadoCancelado},
			EstadoAguardandoTerceiro:  {EstadoEmAtendimento, EstadoCancelado},
			fim:                       {EstadoEncerrado, EstadoEmAtendimento},
		}
	case PapelSolicitante:
		return map[string][]string{
			EstadoAguardandoAprovacao: {EstadoCancelado},
			EstadoAberto:              {EstadoCancelado},
			fim:                       {EstadoEncerrado, EstadoEmAtendimento},
		}
	default:
		return nil
	}
}

// NovaSolicitacao abre o pedido pela oferta: tipo, unidade que atende e
// produto vêm dela.
func NovaSolicitacao(o Oferta, titulo, descricao string, solicitante uuid.UUID) (Solicitacao, error) {
	if !o.Ativa {
		return Solicitacao{}, ErrOfertaInativa
	}
	if !TipoValido(o.Tipo) {
		return Solicitacao{}, ErrTipoInvalido
	}
	titulo, descricao, err := campos(titulo, descricao)
	if err != nil {
		return Solicitacao{}, ErrSolicitacaoInvalida
	}
	if solicitante == uuid.Nil {
		return Solicitacao{}, ErrSolicitanteInvalido
	}
	now := time.Now().UTC()
	return Solicitacao{
		ID: uuid.New(), Tipo: o.Tipo, Estado: EstadoAberto, Titulo: titulo, Descricao: descricao,
		OfertaID: o.ID, ProdutoID: o.ProdutoID, UnidadeID: o.UnidadeID, SolicitanteID: solicitante,
		Impacto: NivelPadrao, Urgencia: NivelPadrao,
		CreatedAt: now, UpdatedAt: now, OfertaNome: o.Nome,
	}, nil
}

// Finalizada reporta se a solicitação saiu da fila de vez.
func (s Solicitacao) Finalizada() bool {
	return s.Estado == EstadoEncerrado || s.Estado == EstadoCancelado
}

// Transicionar aplica uma passagem permitida a algum dos papéis de quem
// age. O mesmo estado não é mudança. Atender exige responsável.
func (s Solicitacao) Transicionar(para string, papeis ...string) (Solicitacao, bool, error) {
	para = strings.TrimSpace(para)
	if para == s.Estado {
		return s, false, nil
	}
	if s.Finalizada() {
		return Solicitacao{}, false, ErrSolicitacaoFinalizada
	}
	if !slices.Contains(s.Proximos(papeis...), para) {
		return Solicitacao{}, false, ErrTransicaoInvalida
	}
	if para == EstadoEmAtendimento && s.ResponsavelID == nil {
		return Solicitacao{}, false, ErrSemResponsavel
	}
	now := time.Now().UTC()
	switch para {
	case Conclusao(s.Tipo):
		s.ResolvidaEm = &now
	case EstadoEmAtendimento:
		s.ResolvidaEm = nil // reaberta: a conclusão anterior não vale
	case EstadoEncerrado, EstadoCancelado:
		s.EncerradaEm = &now
	}
	if para != EstadoAguardandoAprovacao {
		s.EtapaAtual = nil
	}
	s.Estado = para
	s.UpdatedAt = now
	return s, true, nil
}

// Atribuir troca o responsável. Atribuir uma solicitação aberta já a põe
// em atendimento: alguém assumiu. Quem pode ser responsável (membro de
// uma equipe da unidade) a aplicação confere.
func (s Solicitacao) Atribuir(responsavel uuid.UUID) (Solicitacao, bool, error) {
	if s.Finalizada() {
		return Solicitacao{}, false, ErrSolicitacaoFinalizada
	}
	if responsavel == uuid.Nil {
		return Solicitacao{}, false, ErrResponsavelInvalido
	}
	if s.Estado == EstadoAguardandoAprovacao {
		return Solicitacao{}, false, ErrAguardandoAprovacao
	}
	if s.ResponsavelID != nil && *s.ResponsavelID == responsavel {
		return s, false, nil
	}
	s.ResponsavelID = &responsavel
	if s.Estado == EstadoAberto {
		s.Estado = EstadoEmAtendimento
	}
	s.UpdatedAt = time.Now().UTC()
	return s, true, nil
}

// RetomarPorResposta devolve à equipe a solicitação que esperava o
// servidor, quando ele responde.
func (s Solicitacao) RetomarPorResposta() (Solicitacao, bool) {
	if s.Estado != EstadoAguardandoServidor || s.ResponsavelID == nil {
		return s, false
	}
	s.Estado = EstadoEmAtendimento
	s.UpdatedAt = time.Now().UTC()
	return s, true
}

// Proximos devolve as passagens que os papéis podem fazer agora, sem
// repetição, na ordem da máquina.
func (s Solicitacao) Proximos(papeis ...string) []string {
	out := []string{}
	for _, papel := range papeis {
		for _, destino := range transicoes(s.Tipo, papel)[s.Estado] {
			if !slices.Contains(out, destino) {
				out = append(out, destino)
			}
		}
	}
	// Esperando aprovação, só o cancelamento: as colunas acrescentadas no
	// quadro não tiram o pedido da aprovação.
	if slices.Contains(papeis, PapelTecnico) && !s.Finalizada() && s.Estado != EstadoAguardandoAprovacao {
		for _, destino := range s.passagens {
			if destino != "" && destino != s.Estado && !slices.Contains(out, destino) {
				out = append(out, destino)
			}
		}
	}
	return out
}

// ComPassagens devolve a cópia com os caminhos acrescentados na fila.
func (s Solicitacao) ComPassagens(extras []string) Solicitacao {
	s.passagens = append([]string{}, extras...)
	return s
}

// NovoComentario valida a resposta ou a nota. Solicitação finalizada não
// recebe comentário; nota interna é só da equipe.
func NovoComentario(s Solicitacao, autor uuid.UUID, texto string, interno, tecnico bool) (Comentario, error) {
	if s.Finalizada() {
		return Comentario{}, ErrSolicitacaoFinalizada
	}
	if interno && !tecnico {
		return Comentario{}, ErrNotaInternaNegada
	}
	texto = strings.TrimSpace(texto)
	if texto == "" || utf8.RuneCountInString(texto) > TextoMax {
		return Comentario{}, ErrComentarioInvalido
	}
	return Comentario{
		ID: uuid.New(), SolicitacaoID: s.ID, AutorID: &autor, Texto: texto, Interno: interno,
		CreatedAt: time.Now().UTC(),
	}, nil
}
