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
	// Categorias do catálogo em blocos (ADR 078), em ordem; soAtivas para
	// a central do solicitante.
	ListCategorias(ctx context.Context, db database.DBTX, soAtivas bool) ([]Categoria, error)
	GetCategoria(ctx context.Context, db database.DBTX, id uuid.UUID) (Categoria, error)
	InsertCategoria(ctx context.Context, db database.DBTX, c Categoria) (Categoria, error)
	UpdateCategoria(ctx context.Context, db database.DBTX, c Categoria) (Categoria, error)
	// DeleteCategoria apaga; as ofertas dela ficam sem categoria.
	DeleteCategoria(ctx context.Context, db database.DBTX, id uuid.UUID) error

	ListSolicitacoes(ctx context.Context, db database.DBTX, f FiltroSolicitacao) ([]Solicitacao, int64, error)
	GetSolicitacao(ctx context.Context, db database.DBTX, id uuid.UUID) (Solicitacao, error)
	// TravarSolicitacao bloqueia a linha até o fim da transação: duas
	// passagens simultâneas sobre a mesma solicitação são serializadas.
	TravarSolicitacao(ctx context.Context, db database.DBTX, id uuid.UUID) error
	InsertSolicitacao(ctx context.Context, db database.DBTX, s Solicitacao) (Solicitacao, error)
	UpdateSolicitacao(ctx context.Context, db database.DBTX, s Solicitacao) error
	// SetSegredo grava ou limpa o id do cofre. Não grava senha.
	SetSegredo(ctx context.Context, db database.DBTX, id uuid.UUID, segredo *uuid.UUID, agora time.Time) error
	// SetContratoApoio grava ou limpa (nil) o contrato de apoio (ADR 071).
	SetContratoApoio(ctx context.Context, db database.DBTX, id uuid.UUID, c *ContratoDeApoio, agora time.Time) error
	// AbrirAtribuicao abre o trecho do responsável, com a equipe dele na
	// unidade; FecharAtribuicao fecha o trecho aberto, se houver (ADR 073).
	AbrirAtribuicao(ctx context.Context, db database.DBTX, solicitacao, responsavel uuid.UUID, unidade *uuid.UUID, em time.Time) error
	FecharAtribuicao(ctx context.Context, db database.DBTX, solicitacao uuid.UUID, em time.Time) error

	InsertPassagem(ctx context.Context, db database.DBTX, p Passagem) (Passagem, error)
	ListPassagens(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]Passagem, error)
	InsertComentario(ctx context.Context, db database.DBTX, c Comentario) (Comentario, error)
	ListComentarios(ctx context.Context, db database.DBTX, solicitacao uuid.UUID, internos bool) ([]Comentario, error)
	// GetAvaliacao devolve ErrNaoEncontrado quando ninguém avaliou.
	// InsertAvaliacao recusa a segunda avaliação da mesma solicitação.
	GetAvaliacao(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) (Avaliacao, error)
	InsertAvaliacao(ctx context.Context, db database.DBTX, a Avaliacao) (Avaliacao, error)
	// ResumoAvaliacoes agrega as notas do escopo e devolve as mais recentes.
	ResumoAvaliacoes(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, recentes int) (ResumoAvaliacoes, error)
	// ListarParaGovernanca devolve até limite solicitações do escopo, da
	// mais antiga para a mais nova, para os indicadores. Sem paginação da fila.
	ListarParaGovernanca(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, limite int) ([]Solicitacao, error)
	// UnidadesComSolicitacao lista as unidades que já tiveram chamado.
	UnidadesComSolicitacao(ctx context.Context, db database.DBTX) ([]uuid.UUID, error)
	// GravarPonto regrava o dia de hoje. Dias anteriores não são chamados.
	GravarRetrato(ctx context.Context, db database.DBTX, p Retrato) error
	// ListarRetratos devolve o retrato diário. restrito false lê só a linha
	// institucional (unidade nula), que já reúne tudo.
	ListarRetratos(ctx context.Context, db database.DBTX, desde time.Time, restrito bool, unidades []uuid.UUID) ([]Retrato, error)
	// IncidentesRecorrentes agrupa por oferta os incidentes (não
	// cancelados) desde a data, com pelo menos minimo: a matéria-prima da
	// sugestão de problema (fase 6).
	IncidentesRecorrentes(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time, minimo int) ([]Recorrencia, error)

	// SolicitacaoPorNumero acha a solicitação pelo número público.
	SolicitacaoPorNumero(ctx context.Context, db database.DBTX, numero int64) (Solicitacao, error)
	EmailDe(ctx context.Context, db database.DBTX, user uuid.UUID) (string, error)
	// ReservarCorreio marca a mensagem. false significa que ela já entrou.
	ReservarCorreio(ctx context.Context, db database.DBTX, messageID string, solicitacao uuid.UUID) (bool, error)
	LiberarCorreio(ctx context.Context, db database.DBTX, messageID string) error

	// Responsaveis lista os membros ativos das equipes da unidade (nil =
	// equipes institucionais); ResponsavelValido confere um deles.
	Responsaveis(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]Candidato, error)
	ResponsavelValido(ctx context.Context, db database.DBTX, user uuid.UUID, unidade *uuid.UUID) (bool, error)

	ListarRegras(ctx context.Context, db database.DBTX, f Filtro) ([]Regra, error)
	ListarRegrasDaUnidade(ctx context.Context, db database.DBTX, unidade *uuid.UUID) ([]Regra, error)
	ContarRegras(ctx context.Context, db database.DBTX, unidade *uuid.UUID) (int, error)
	InserirRegra(ctx context.Context, db database.DBTX, r Regra) error
	ExcluirRegra(ctx context.Context, db database.DBTX, id uuid.UUID) error
	ObterRegra(ctx context.Context, db database.DBTX, id uuid.UUID) (Regra, error)

	ObterResposta(ctx context.Context, db database.DBTX, oferta uuid.UUID) (RespostaPronta, error)
	SalvarResposta(ctx context.Context, db database.DBTX, r RespostaPronta) error
	ExcluirResposta(ctx context.Context, db database.DBTX, oferta uuid.UUID) error

	ListarRecorrencias(ctx context.Context, db database.DBTX, f Filtro) ([]ChamadoMensal, error)
	ListarRecorrenciasAtivas(ctx context.Context, db database.DBTX) ([]ChamadoMensal, error)
	InserirRecorrencia(ctx context.Context, db database.DBTX, r ChamadoMensal) error
	ExcluirRecorrencia(ctx context.Context, db database.DBTX, id uuid.UUID) error
	ObterRecorrencia(ctx context.Context, db database.DBTX, id uuid.UUID) (ChamadoMensal, error)
	// ReivindicarAbertura reserva o mês. false significa que ele já foi aberto.
	ReivindicarAbertura(ctx context.Context, db database.DBTX, id uuid.UUID, ano, mes int) (bool, error)
	LigarAbertura(ctx context.Context, db database.DBTX, id uuid.UUID, ano, mes int, solicitacao uuid.UUID) error
	SolicitacaoDoMes(ctx context.Context, db database.DBTX, oferta, solicitante uuid.UUID, titulo string, inicio time.Time) (uuid.UUID, error)

	RegistrarDeflexao(ctx context.Context, db database.DBTX, usuario uuid.UUID, artigo string, dia string, unidade *uuid.UUID, resolveu bool) error
	MedirDeflexao(ctx context.Context, db database.DBTX, desde string, restrito bool, unidades []uuid.UUID) (int, int, error)

	ListApontamentos(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]Apontamento, error)
	InsertApontamento(ctx context.Context, db database.DBTX, a Apontamento) (Apontamento, error)
	DeleteApontamento(ctx context.Context, db database.DBTX, solicitacaoID, apontamentoID uuid.UUID) error

	PrazoRepository
	IARepository
	EventosRepository
	AprovacaoRepository
	TarefaRepository

	RelatorioRepository

	// MetasXLA devolve as metas de experiência (vazias quando ninguém definiu).
	MetasXLA(ctx context.Context, db database.DBTX) (MetasXLA, error)
	SalvarMetasXLA(ctx context.Context, db database.DBTX, m MetasXLA, por uuid.UUID, agora time.Time) error
}

// RelatorioRepository lê o que o relatório por secretaria e o OLA das
// equipes precisam (ADR 070).
type RelatorioRepository interface {
	// ListarDoPeriodo lê as solicitações do escopo abertas desde a data, das
	// mais novas para as mais antigas, até o limite.
	ListarDoPeriodo(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time, limite int) ([]Solicitacao, error)
	// SecretariasDe devolve, para cada unidade, a unidade de topo da árvore.
	SecretariasDe(ctx context.Context, db database.DBTX, unidades []uuid.UUID) (map[uuid.UUID]Secretaria, error)
	// AvaliacoesDe devolve a avaliação de cada solicitação que tem uma.
	AvaliacoesDe(ctx context.Context, db database.DBTX, solicitacoes []uuid.UUID) (map[uuid.UUID]Avaliacao, error)
	// TarefasDoPeriodo lê as tarefas dos pedidos do escopo abertos desde a
	// data, com a equipe e o prazo dela.
	TarefasDoPeriodo(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time) ([]TarefaMedida, error)
	// ComContratoNoPeriodo lê os chamados do escopo, abertos desde a data,
	// que apontam para um contrato de apoio.
	ComContratoNoPeriodo(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time) ([]Solicitacao, error)
	// AtribuicoesDoPeriodo lê os trechos com equipe dos chamados do escopo
	// que começaram desde a data ou que seguem abertos (ADR 073).
	AtribuicoesDoPeriodo(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, desde time.Time) ([]Atribuicao, error)
}

// TarefaRepository guarda o modelo de tarefas da oferta e as tarefas de
// cada pedido (ADR 068).
type TarefaRepository interface {
	// ModeloDaOferta devolve o modelo em ordem, com o nome da equipe.
	ModeloDaOferta(ctx context.Context, db database.DBTX, oferta uuid.UUID) ([]ModeloTarefa, error)
	// SalvarModelo troca o modelo inteiro (vazio tira as tarefas).
	SalvarModelo(ctx context.Context, db database.DBTX, oferta uuid.UUID, modelo []ModeloTarefa) error
	InsertTarefas(ctx context.Context, db database.DBTX, tarefas []Tarefa) error
	ListTarefas(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]Tarefa, error)
	// GetTarefa busca a tarefa do pedido (ErrNaoEncontrado se for de outro).
	GetTarefa(ctx context.Context, db database.DBTX, solicitacao, id uuid.UUID) (Tarefa, error)
	UpdateTarefa(ctx context.Context, db database.DBTX, t Tarefa) error
	// MembroDe diz de quais das equipes a pessoa é membro.
	MembroDe(ctx context.Context, db database.DBTX, user uuid.UUID, equipes []uuid.UUID) ([]uuid.UUID, error)
}

// AprovacaoRepository guarda as etapas das ofertas e as decisões (ADR 068).
type AprovacaoRepository interface {
	// EtapasDaOferta devolve as etapas em ordem (vazio = sem aprovação).
	EtapasDaOferta(ctx context.Context, db database.DBTX, oferta uuid.UUID) ([]EtapaAprovacao, error)
	// SalvarEtapas troca as etapas da oferta (vazio tira a aprovação).
	SalvarEtapas(ctx context.Context, db database.DBTX, oferta uuid.UUID, etapas []EtapaAprovacao) error
	InsertAprovacao(ctx context.Context, db database.DBTX, a Aprovacao) error
	ListAprovacoes(ctx context.Context, db database.DBTX, solicitacao uuid.UUID) ([]Aprovacao, error)
	// AguardandoAprovacaoDe lista os pedidos cuja etapa atual a pessoa
	// aprova, dos mais antigos para os mais novos.
	AguardandoAprovacaoDe(ctx context.Context, db database.DBTX, aprovador uuid.UUID, limite int) ([]Solicitacao, error)
}

// EventosRepository liga os alertas do monitoramento aos chamados (ADR 067).
type EventosRepository interface {
	// OfertaDosEventos devolve a oferta escolhida (nil = nenhuma).
	OfertaDosEventos(ctx context.Context, db database.DBTX) (*uuid.UUID, error)
	SalvarOfertaDosEventos(ctx context.Context, db database.DBTX, oferta *uuid.UUID, por uuid.UUID, agora time.Time) error
	// TravarEvento serializa a mesma fingerprint até o fim da transação e
	// devolve o vínculo (ErrNaoEncontrado quando o alerta é novo).
	TravarEvento(ctx context.Context, db database.DBTX, fingerprint string) (EventoVinculado, error)
	SalvarEvento(ctx context.Context, db database.DBTX, e EventoVinculado) error
}

// IARepository guarda as sugestões da IA e o que quem atende decidiu
// (ADR 065).
type IARepository interface {
	InsertSugestao(ctx context.Context, db database.DBTX, s Sugestao) error
	// TravarSugestao lê a sugestão e bloqueia a linha até o fim da
	// transação: duas decisões simultâneas são serializadas.
	TravarSugestao(ctx context.Context, db database.DBTX, id uuid.UUID) (Sugestao, error)
	DecidirSugestao(ctx context.Context, db database.DBTX, s Sugestao) error
	// IndicadoresIA conta, por tarefa, as sugestões desde a data no escopo.
	IndicadoresIA(ctx context.Context, db database.DBTX, desde time.Time, restrito bool, unidades []uuid.UUID) ([]IndicadorIA, error)
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
	Avaliacoes   []Avaliacao   `json:"avaliacoes,omitempty"`
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
	// Texto, quando preenchido, casa com título, descrição ou número.
	// Vazio não filtra.
	Texto  string
	Limit  int
	Offset int
}
