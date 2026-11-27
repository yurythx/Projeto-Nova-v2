package domain

import (
	"errors"
	"slices"
	"time"

	"github.com/google/uuid"
)

// Erros do aceite, do encaminhamento e da aprovação.
var (
	ErrSemResponsavel = errors.New("protocolos: aceite o pedido antes de tratá-lo")
	ErrEncaminhamento = errors.New("protocolos: encaminhamento pendente")
	ErrAprovacao      = errors.New("protocolos: conclusão aguardando aprovação")
)

// Ações da trilha. A consulta pública troca a descrição pelo texto de AcaoPublica.
const (
	AcaoAceite                 = "aceite"
	AcaoTransferencia          = "transferencia"
	AcaoEncaminhamento         = "encaminhamento"
	AcaoEncaminhamentoAceito   = "encaminhamento_aceito"
	AcaoEncaminhamentoRecusado = "encaminhamento_recusado"
	AcaoAprovacao              = "aprovacao"
	AcaoDevolucao              = "devolucao"
	AcaoAguardaAprovacao       = "conclusao_enviada"
	// PrazoMaximoEncaminhamento limita os dias úteis dados à unidade de destino.
	PrazoMaximoEncaminhamento = 365
)

// textoPublico esconde da consulta o nome de quem trata e a justificativa interna.
var textoPublico = map[string]string{
	AcaoEdicao:                 "pedido atualizado",
	AcaoAceite:                 "pedido em análise",
	AcaoTransferencia:          "responsável alterado",
	AcaoEncaminhamento:         "encaminhado a outra unidade",
	AcaoEncaminhamentoAceito:   "recebido pela nova unidade",
	AcaoEncaminhamentoRecusado: "encaminhamento não aceito",
	AcaoAguardaAprovacao:       "conclusão em aprovação",
	AcaoAprovacao:              "conclusão aprovada",
	AcaoDevolucao:              "conclusão em revisão",
}

// AcaoPublica devolve o texto que a consulta mostra no lugar da descrição interna.
func AcaoPublica(acao string) (string, bool) {
	texto, ok := textoPublico[acao]
	return texto, ok
}

// Encaminhamento é o pedido de mudança de unidade, à espera do destino.
type Encaminhamento struct {
	UnidadeID     uuid.UUID  `json:"unidade_id"`
	UnidadeNome   string     `json:"unidade_nome,omitempty"`
	Justificativa string     `json:"justificativa"`
	PrazoDias     int        `json:"prazo_dias"`
	Por           *uuid.UUID `json:"por,omitempty"`
	Em            time.Time  `json:"em"`
}

// Pessoa é quem pode assumir o pedido.
type Pessoa struct {
	ID   uuid.UUID `json:"id"`
	Nome string    `json:"nome"`
}

// livre é o pedido aberto e sem encaminhamento pendente.
func (p Pedido) livre() error {
	if err := p.aberto(); err != nil {
		return err
	}
	if p.Encaminhamento != nil {
		return ErrEncaminhamento
	}
	return nil
}

// Aceitar dá dono ao pedido sem responsável. O pedido novo passa a andamento.
func (p Pedido) Aceitar(responsavel uuid.UUID, agora time.Time) (Pedido, error) {
	if err := p.livre(); err != nil {
		return Pedido{}, err
	}
	if p.Status == StatusComplementacao {
		return Pedido{}, ErrComplemento
	}
	if responsavel == uuid.Nil || p.ResponsavelID != nil {
		return Pedido{}, ErrInvalido
	}
	p.ResponsavelID = &responsavel
	if p.Status == StatusNovo {
		p.Status = StatusAndamento
	}
	p.UpdatedAt = agora
	return p, nil
}

// Transferir troca o responsável na hora.
func (p Pedido) Transferir(novo uuid.UUID, agora time.Time) (Pedido, error) {
	if err := p.livre(); err != nil {
		return Pedido{}, err
	}
	if p.ResponsavelID == nil || novo == uuid.Nil || *p.ResponsavelID == novo {
		return Pedido{}, ErrInvalido
	}
	p.ResponsavelID = &novo
	p.UpdatedAt = agora
	return p, nil
}

// Encaminhar pede a mudança para outra unidade. O pedido fica parado até o destino decidir.
func (p Pedido) Encaminhar(destino uuid.UUID, justificativa string, prazoDias int, por uuid.UUID, agora time.Time) (Pedido, error) {
	if err := p.livre(); err != nil {
		return Pedido{}, err
	}
	if p.Status == StatusComplementacao {
		return Pedido{}, ErrComplemento
	}
	if destino == uuid.Nil || destino == p.UnidadeID || prazoDias < 0 || prazoDias > PrazoMaximoEncaminhamento {
		return Pedido{}, ErrInvalido
	}
	texto, err := TextoInterno(justificativa, 2000)
	if err != nil {
		return Pedido{}, err
	}
	p.Encaminhamento = &Encaminhamento{UnidadeID: destino, Justificativa: texto, PrazoDias: prazoDias, Por: &por, Em: agora}
	p.UpdatedAt = agora
	return p, nil
}

// ReceberEncaminhamento muda o pedido de unidade. O responsável sai: o destino aceita de novo.
func (p Pedido) ReceberEncaminhamento(agora time.Time) (Pedido, error) {
	if err := p.aberto(); err != nil {
		return Pedido{}, err
	}
	if p.Encaminhamento == nil {
		return Pedido{}, ErrInvalido
	}
	e := *p.Encaminhamento
	p.UnidadeID = e.UnidadeID
	p.UnidadeNome = e.UnidadeNome
	p.DepartamentoID = nil
	p.ResponsavelID = nil
	p.ResponsavelNome = ""
	if e.PrazoDias > 0 {
		limite := SomarDiasUteis(agora, e.PrazoDias)
		p.PrazoEm = &limite
	}
	p.Encaminhamento = nil
	p.UpdatedAt = agora
	return p, nil
}

// RecusarEncaminhamento devolve o pedido à unidade de origem como estava.
func (p Pedido) RecusarEncaminhamento(agora time.Time) (Pedido, error) {
	if err := p.aberto(); err != nil {
		return Pedido{}, err
	}
	if p.Encaminhamento == nil {
		return Pedido{}, ErrInvalido
	}
	p.Encaminhamento = nil
	p.UpdatedAt = agora
	return p, nil
}

// Ajustar troca a prioridade e o prazo. Vazio mantém o valor atual.
// O prazo é o dia (AAAA-MM-DD), até o fim do expediente em Cuiabá.
func (p Pedido) Ajustar(prioridade, prazo string, agora time.Time) (Pedido, error) {
	if err := p.aberto(); err != nil {
		return Pedido{}, err
	}
	if prioridade != "" {
		if !slices.Contains(prioridades, prioridade) {
			return Pedido{}, ErrInvalido
		}
		p.Prioridade = prioridade
	}
	if prazo != "" {
		dia, err := time.ParseInLocation("2006-01-02", prazo, Cuiaba)
		if err != nil || dia.Before(diaCivil(p.CreatedAt.In(Cuiaba))) {
			return Pedido{}, ErrInvalido
		}
		limite := time.Date(dia.Year(), dia.Month(), dia.Day(), 23, 59, 0, 0, Cuiaba)
		p.PrazoEm = &limite
	}
	p.UpdatedAt = agora
	return p, nil
}

// PedirAprovacao grava a decisão e espera quem gere a unidade aprovar.
func (p Pedido) PedirAprovacao(decisao, justificativa string, agora time.Time) (Pedido, error) {
	p, err := p.Concluir(decisao, justificativa, agora)
	if err != nil {
		return Pedido{}, err
	}
	p.Status = StatusAguardando
	return p, nil
}

func (p Pedido) aguardandoAprovacao() error {
	if err := p.aberto(); err != nil {
		return err
	}
	if p.Status != StatusAguardando || p.Decisao == "" {
		return ErrInvalido
	}
	return nil
}

// Aprovar confirma a conclusão com a decisão já gravada.
func (p Pedido) Aprovar(agora time.Time) (Pedido, error) {
	if err := p.aguardandoAprovacao(); err != nil {
		return Pedido{}, err
	}
	p.Status = StatusConcluido
	p.UpdatedAt = agora
	return p, nil
}

// Devolver recusa a conclusão: o pedido volta ao andamento sem a decisão.
func (p Pedido) Devolver(agora time.Time) (Pedido, error) {
	if err := p.aguardandoAprovacao(); err != nil {
		return Pedido{}, err
	}
	p.Status = StatusAndamento
	p.Decisao = ""
	p.Justificativa = ""
	p.UpdatedAt = agora
	return p, nil
}
