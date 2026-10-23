package domain

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// Filtro restringe a listagem ao escopo de atendimento:manage.
type Filtro struct {
	Restrito bool
	Unidades []uuid.UUID
}

// Repository é a porta de persistência do atendimento.
type Repository interface {
	ListOfertas(ctx context.Context, db database.DBTX, f Filtro) ([]Oferta, error)
	GetOferta(ctx context.Context, db database.DBTX, id uuid.UUID) (Oferta, error)
	InsertOferta(ctx context.Context, db database.DBTX, o Oferta) (Oferta, error)
	UpdateOferta(ctx context.Context, db database.DBTX, o Oferta) (Oferta, error)

	ListEquipes(ctx context.Context, db database.DBTX, f Filtro) ([]Equipe, error)
	GetEquipe(ctx context.Context, db database.DBTX, id uuid.UUID) (Equipe, error)
	InsertEquipe(ctx context.Context, db database.DBTX, e Equipe) (Equipe, error)
	UpdateEquipe(ctx context.Context, db database.DBTX, e Equipe) (Equipe, error)
	SubstituirMembros(ctx context.Context, db database.DBTX, equipe uuid.UUID, membros []uuid.UUID) error

	ContarAtivos(ctx context.Context, db database.DBTX, ids []uuid.UUID) (int, error)
	Candidatos(ctx context.Context, db database.DBTX, unidade uuid.UUID) ([]Candidato, error)

	SolicitacaoRepository
}

// SolicitacaoRepository persiste a solicitação, as passagens e a conversa.
type SolicitacaoRepository interface {
	ListOfertasAtivas(ctx context.Context, db database.DBTX) ([]Oferta, error)

	ListSolicitacoes(ctx context.Context, db database.DBTX, f FiltroSolicitacao) ([]Solicitacao, int64, error)
	GetSolicitacao(ctx context.Context, db database.DBTX, id uuid.UUID) (Solicitacao, error)
	// TravarSolicitacao bloqueia a linha até o fim da transação: duas
	// passagens simultâneas sobre a mesma solicitação são serializadas.
	TravarSolicitacao(ctx context.Context, db database.DBTX, id uuid.UUID) error
	InsertSolicitacao(ctx context.Context, db database.DBTX, s Solicitacao) (Solicitacao, error)
	UpdateSolicitacao(ctx context.Context, db database.DBTX, s Solicitacao) error

	InsertPassagem(ctx context.Context, db database.DBTX, p Passagem) (Passagem, error)
	ListPassagens(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]Passagem, error)
	InsertComentario(ctx context.Context, db database.DBTX, c Comentario) (Comentario, error)
	ListComentarios(ctx context.Context, db database.DBTX, solicitacao uuid.UUID, internos bool) ([]Comentario, error)
	// IncidentesRecorrentes agrupa por oferta os incidentes (não
	// cancelados) desde a data, com pelo menos minimo: a matéria-prima da
	// sugestão de problema (fase 6).
	IncidentesRecorrentes(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time, minimo int) ([]Recorrencia, error)

	// Responsaveis lista os membros ativos das equipes da unidade (nil =
	// equipes institucionais); ResponsavelValido confere um deles.
	Responsaveis(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]Candidato, error)
	ResponsavelValido(ctx context.Context, db database.DBTX, user uuid.UUID, unidade *uuid.UUID) (bool, error)

	ListApontamentos(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]Apontamento, error)
	InsertApontamento(ctx context.Context, db database.DBTX, a Apontamento) (Apontamento, error)
	DeleteApontamento(ctx context.Context, db database.DBTX, solicitacaoID, apontamentoID uuid.UUID) error

	PrazoRepository
}

// PrazoRepository persiste a política de prazo e os feriados, e lê o que
// o relógio precisa.
type PrazoRepository interface {
	// GetPolitica busca a política da própria unidade (nil = institucional).
	GetPolitica(ctx context.Context, db database.DBTX, unidade *uuid.UUID) (Politica, error)
	// ResolverPolitica busca a da unidade ou da mais próxima acima; no
	// topo, a institucional. ErrNaoEncontrado quando não há nenhuma.
	ResolverPolitica(ctx context.Context, db database.DBTX, unidade *uuid.UUID) (Politica, error)
	SalvarPolitica(ctx context.Context, db database.DBTX, p Politica) (Politica, error)

	ListFeriados(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]Feriado, error)
	// FeriadosVigentes são os da unidade, os das unidades acima e os
	// institucionais, entre as datas (inclusive).
	FeriadosVigentes(ctx context.Context, db database.DBTX, unidade *uuid.UUID, de, ate string) ([]Feriado, error)
	GetFeriado(ctx context.Context, db database.DBTX, id uuid.UUID) (Feriado, error)
	InsertFeriado(ctx context.Context, db database.DBTX, f Feriado) (Feriado, error)
	DeleteFeriado(ctx context.Context, db database.DBTX, id uuid.UUID) error

	// PassagensDe devolve as passagens de várias solicitações, em ordem.
	PassagensDe(ctx context.Context, db database.DBTX, ids []uuid.UUID) (map[uuid.UUID][]Passagem, error)
	// ComPrazoCorrendo são as solicitações classificadas e em andamento:
	// as que o alerta de prazo acompanha.
	ComPrazoCorrendo(ctx context.Context, db database.DBTX) ([]Solicitacao, error)

	VisitaRepository
}

// VisitaRepository persiste as visitas e as evidências.
type VisitaRepository interface {
	GetVisita(ctx context.Context, db database.DBTX, id uuid.UUID) (Visita, error)
	TravarVisita(ctx context.Context, db database.DBTX, id uuid.UUID) error
	// InsertVisita recusa com ErrVisitaAberta a segunda visita aberta do
	// mesmo técnico na mesma solicitação.
	InsertVisita(ctx context.Context, db database.DBTX, v Visita) error
	UpdateVisita(ctx context.Context, db database.DBTX, v Visita) error
	// ListVisitas devolve as visitas da solicitação com as evidências.
	ListVisitas(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]Visita, error)
	// VisitasEmAndamento são as abertas nas unidades do filtro.
	VisitasEmAndamento(ctx context.Context, db database.DBTX, f Filtro) ([]VisitaEmCampo, error)
	// InsertEvidencia é idempotente pela chave do objeto.
	InsertEvidencia(ctx context.Context, db database.DBTX, e Evidencia) (Evidencia, error)
	GetEvidencia(ctx context.Context, db database.DBTX, id uuid.UUID) (Evidencia, error)
	// ApagarLocalizacoes anula as coordenadas vencidas e diz quantas
	// visitas foram tocadas.
	ApagarLocalizacoes(ctx context.Context, db database.DBTX, agora time.Time) (int64, error)

	// DadosDoTitular e ApagarLocalizacoesDe atendem os direitos do titular
	// (LGPD art. 18): exportação e eliminação.
	DadosDoTitular(ctx context.Context, db database.DBTX, user uuid.UUID) (DadosDoTitular, error)
	ApagarLocalizacoesDe(ctx context.Context, db database.DBTX, user uuid.UUID) error
}

// DadosDoTitular é o que o atendimento guarda sobre uma pessoa: o que ela
// pediu, o que escreveu e as visitas que fez como técnica.
type DadosDoTitular struct {
	Solicitacoes []Solicitacao `json:"solicitacoes"`
	Comentarios  []Comentario  `json:"comentarios"`
	Visitas      []Visita      `json:"visitas"`
}

// Recorrencia são incidentes da mesma oferta num período.
type Recorrencia struct {
	OfertaID     uuid.UUID
	OfertaNome   string
	UnidadeID    *uuid.UUID
	Total        int
	UltimoEm     time.Time
	Solicitacoes []uuid.UUID
}

// VisitaEmCampo é a visita aberta com a solicitação, para a supervisão.
type VisitaEmCampo struct {
	Visita
	Numero    int64      `json:"numero"`
	Titulo    string     `json:"titulo"`
	UnidadeID *uuid.UUID `json:"unidade_id,omitempty"`
}

// FiltroSolicitacao recorta a listagem. Solicitante preenchido devolve só
// as solicitações daquela pessoa e ignora o recorte de unidade: é a lista
// de quem abriu. Sem Estado, Finalizadas decide se encerradas e
// canceladas entram.
type FiltroSolicitacao struct {
	Restrito    bool
	Unidades    []uuid.UUID
	Solicitante *uuid.UUID
	Tipo        string
	Estado      string
	Finalizadas bool
	Responsavel *uuid.UUID
	SemDono     bool
	Limit       int
	Offset      int
}
