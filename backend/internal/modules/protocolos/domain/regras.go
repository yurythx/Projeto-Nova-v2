// Package domain guarda a carta de serviços e o pedido do cidadão.
// Secretaria é a unidade do IAM. O módulo não importa outro plug-in.
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrInvalido       = errors.New("protocolos: dados inválidos")
	ErrNaoEncontrado  = errors.New("protocolos: não encontrado")
	ErrEstadoFinal    = errors.New("protocolos: pedido encerrado")
	ErrComplemento    = errors.New("protocolos: pedido em complementação")
	ErrNumeroRepetido = errors.New("protocolos: número já existe")
	ErrCPFAberto      = errors.New("protocolos: já há pedido deste CPF neste serviço")
	ErrResponsavel    = errors.New("protocolos: serviço exclusivo de responsável técnico")
	ErrCPFCadastrado  = errors.New("protocolos: CPF já cadastrado como responsável técnico")
	ErrBloqueio       = errors.New("protocolos: agendamento bloqueado por faltas")
	ErrSemVaga        = errors.New("protocolos: sem horário nos próximos 30 dias")
)

// Cuiaba é o fuso do recibo, do prazo e do agendamento (UTC−4, sem horário de verão).
var Cuiaba = time.FixedZone("Cuiaba", -4*60*60)

// SomarDiasUteis conta dias em Cuiabá e pula sábado e domingo. O feriado
// municipal não entra neste plug-in: ele mora no calendário de quem o cadastra.
func SomarDiasUteis(inicio time.Time, dias int) time.Time {
	atual := inicio.In(Cuiaba)
	if dias <= 0 {
		return atual
	}
	for restantes := dias; restantes > 0; {
		atual = atual.AddDate(0, 0, 1)
		if atual.Weekday() == time.Saturday || atual.Weekday() == time.Sunday {
			continue
		}
		restantes--
	}
	return atual
}

// RotuloStatus é o nome que a consulta mostra.
func RotuloStatus(status string) string {
	switch status {
	case StatusNovo:
		return "Novo"
	case StatusAndamento:
		return "Em andamento"
	case StatusPendente:
		return "Pendente"
	case StatusAnalise:
		return "Em análise"
	case StatusComplementacao:
		return "Em complementação"
	case StatusAguardando:
		return "Aguardando aprovação"
	case StatusAprovado:
		return "Aprovado"
	case StatusRejeitado:
		return "Rejeitado"
	case StatusConcluido:
		return "Concluído"
	case StatusCancelado:
		return "Cancelado"
	default:
		return status
	}
}

const (
	PrazoNoPrazo    = "no_prazo"
	PrazoAtencao    = "atencao"
	PrazoAtrasado   = "atrasado"
	PrazoFinalizado = "finalizado"
)

// SituacaoPrazo classifica a mesa. Estado final encerra a conta. Sem data,
// o pedido fica no prazo. Passou o dia do prazo, está atrasado. A partir de
// três quartos dos dias úteis entre a abertura e o prazo, fica em atenção.
func SituacaoPrazo(status string, criado time.Time, prazo *time.Time, agora time.Time) string {
	if EstadoFinal(status) {
		return PrazoFinalizado
	}
	if prazo == nil || prazo.IsZero() {
		return PrazoNoPrazo
	}
	limite := prazo.In(Cuiaba)
	hoje := agora.In(Cuiaba)
	if diaCivil(hoje).After(diaCivil(limite)) {
		return PrazoAtrasado
	}
	total := diasUteisFechado(criado.In(Cuiaba), limite)
	if total <= 0 {
		return PrazoNoPrazo
	}
	if diasUteisFechado(criado.In(Cuiaba), hoje)*4 > total*3 {
		return PrazoAtencao
	}
	return PrazoNoPrazo
}

func diaCivil(t time.Time) time.Time {
	ano, mes, dia := t.In(Cuiaba).Date()
	return time.Date(ano, mes, dia, 0, 0, 0, 0, Cuiaba)
}

func diasUteisFechado(inicio, fim time.Time) int {
	a, b := diaCivil(inicio), diaCivil(fim)
	if b.Before(a) {
		return 0
	}
	n := 0
	for !a.After(b) {
		if a.Weekday() != time.Saturday && a.Weekday() != time.Sunday {
			n++
		}
		a = a.AddDate(0, 0, 1)
	}
	return n
}

// Coordenada aceita "latitude,longitude" dentro dos limites do mapa.
func Coordenada(s string) bool {
	partes := strings.Split(strings.TrimSpace(s), ",")
	if len(partes) != 2 {
		return false
	}
	lat, errLat := strconv.ParseFloat(strings.TrimSpace(partes[0]), 64)
	lng, errLng := strconv.ParseFloat(strings.TrimSpace(partes[1]), 64)
	if errLat != nil || errLng != nil {
		return false
	}
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

const (
	StatusNovo             = "novo"
	StatusAndamento        = "em_andamento"
	StatusPendente         = "pendente"
	StatusAnalise          = "em_analise"
	StatusComplementacao   = "em_complementacao"
	StatusAguardando       = "pending_approval"
	StatusAprovado         = "aprovado"
	StatusRejeitado        = "rejeitado"
	StatusConcluido        = "concluido"
	StatusCancelado        = "cancelado"
	PrioridadeNormal       = "normal"
	DecisaoFavoravel       = "favoravel"
	DecisaoDesfavoravel    = "desfavoravel"
	LimiteBuscaAgendamento = 30
)

var (
	tiposCampo = []string{
		"text", "number", "email", "phone", "date", "time", "password", "url",
		"cpf", "cnpj", "cep", "textarea", "select", "checkbox", "checkbox_group",
		"radio", "boolean", "file", "map", "map_area", "ambulante", "page",
	}
	statusConhecidos = []string{
		StatusNovo, StatusAndamento, StatusPendente, StatusAnalise, StatusComplementacao,
		StatusAguardando, StatusAprovado, StatusRejeitado, StatusConcluido, StatusCancelado,
	}
	finais      = []string{StatusConcluido, StatusAprovado, StatusRejeitado, StatusCancelado}
	prioridades = []string{"baixa", "normal", "alta", "urgente"}
	idCampo     = regexp.MustCompile(`^[A-Za-z0-9_-]{1,40}$`)
	horaMinuto  = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)
	digitosSo   = regexp.MustCompile(`\D`)
)

// EstadoFinal recusa mutação.
func EstadoFinal(status string) bool { return slices.Contains(finais, status) }

// StatusConhecido confere a lista fechada.
func StatusConhecido(status string) bool { return slices.Contains(statusConhecidos, status) }

// Campo é um item do formulário desenhado pela secretaria.
type Campo struct {
	ID          string   `json:"id"`
	Tipo        string   `json:"tipo"`
	Rotulo      string   `json:"rotulo"`
	Obrigatorio bool     `json:"obrigatorio"`
	Placeholder string   `json:"placeholder,omitempty"`
	Opcoes      []string `json:"opcoes,omitempty"`
	Revela      []string `json:"revela,omitempty"`
}

// Formulario é o desenho guardado e a cópia que viaja com o pedido.
type Formulario struct {
	ID        uuid.UUID `json:"id"`
	UnidadeID uuid.UUID `json:"unidade_id"`
	Titulo    string    `json:"titulo"`
	Descricao string    `json:"descricao"`
	Instrucao string    `json:"instrucao"`
	Campos    []Campo   `json:"campos"`
	Ativo     bool      `json:"ativo"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Orientacao é requisito ou documento esperado: texto, não o arquivo.
type Orientacao struct {
	Titulo      string `json:"titulo"`
	Texto       string `json:"texto"`
	Obrigatorio bool   `json:"obrigatorio"`
}

// Etapa é um passo do workflow do serviço.
type Etapa struct {
	ID              string `json:"id"`
	Ordem           int    `json:"ordem"`
	Status          string `json:"status"`
	PrazoDias       int    `json:"prazo_dias"`
	ResponsavelTipo string `json:"responsavel_tipo"`
	ResponsavelID   string `json:"responsavel_id"`
}

// Workflow só corre quando Ligado é verdadeiro.
type Workflow struct {
	Ligado bool    `json:"ligado"`
	Etapas []Etapa `json:"etapas"`
}

// Agenda é a grade do serviço que pede hora.
type Agenda struct {
	Dias       []int  `json:"dias"`
	Inicio     string `json:"inicio"`
	Fim        string `json:"fim"`
	DuracaoMin int    `json:"duracao_min"`
	VagasDia   int    `json:"vagas_dia"`
}

// Servico é o que o cidadão escolhe na carta.
type Servico struct {
	ID                   uuid.UUID    `json:"id"`
	UnidadeID            uuid.UUID    `json:"unidade_id"`
	UnidadeNome          string       `json:"unidade_nome,omitempty"`
	FormularioID         uuid.UUID    `json:"formulario_id"`
	Nome                 string       `json:"nome"`
	Descricao            string       `json:"descricao"`
	Categoria            string       `json:"categoria"`
	PrazoDias            int          `json:"prazo_dias"`
	TaxaCentavos         int          `json:"taxa_centavos"`
	Destaque             bool         `json:"destaque"`
	Ativo                bool         `json:"ativo"`
	AprovaSozinho        bool         `json:"aprova_sozinho"`
	ExigeAgendamento     bool         `json:"exige_agendamento"`
	SoResponsavelTecnico bool         `json:"so_responsavel_tecnico"`
	ImpedirCPFAberto     bool         `json:"impedir_cpf_aberto"`
	CampoAnonimo         string       `json:"campo_anonimo"`
	Workflow             Workflow     `json:"workflow"`
	Agenda               Agenda       `json:"agenda"`
	Requisitos           []Orientacao `json:"requisitos"`
	Documentos           []Orientacao `json:"documentos"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// Pedido é o protocolo preenchido.
type Pedido struct {
	ID             uuid.UUID      `json:"id"`
	Numero         string         `json:"numero"`
	ServicoID      uuid.UUID      `json:"servico_id"`
	ServicoNome    string         `json:"servico_nome"`
	FormularioID   uuid.UUID      `json:"formulario_id"`
	UnidadeID      uuid.UUID      `json:"unidade_id"`
	UnidadeNome    string         `json:"unidade_nome,omitempty"`
	DepartamentoID *uuid.UUID     `json:"departamento_id,omitempty"`
	Nome           string         `json:"nome"`
	CPF            string         `json:"cpf,omitempty"`
	Email          string         `json:"email,omitempty"`
	Telefone       string         `json:"telefone,omitempty"`
	Respostas      map[string]any `json:"respostas,omitempty"`
	Campos         []Campo        `json:"campos,omitempty"`
	Status         string         `json:"status"`
	SituacaoPrazo  string         `json:"situacao_prazo,omitempty"`
	EtapaID        string         `json:"etapa_id"`
	Prioridade     string         `json:"prioridade"`
	PrazoEm        *time.Time     `json:"prazo_em,omitempty"`
	Decisao        string         `json:"decisao"`
	Justificativa  string         `json:"justificativa"`
	Complemento    string         `json:"complemento"`
	UsuarioID      *uuid.UUID     `json:"usuario_id,omitempty"`
	ResponsavelID  *uuid.UUID     `json:"responsavel_id,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Historico é uma linha da trilha do pedido.
type Historico struct {
	ID        uuid.UUID  `json:"id"`
	PedidoID  uuid.UUID  `json:"pedido_id"`
	Acao      string     `json:"acao"`
	Descricao string     `json:"descricao"`
	Status    string     `json:"status"`
	Anexos    []string   `json:"anexos,omitempty"`
	UsuarioID *uuid.UUID `json:"usuario_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Agendamento é o horário marcado junto do pedido.
type Agendamento struct {
	ID        uuid.UUID `json:"id"`
	PedidoID  uuid.UUID `json:"pedido_id"`
	ServicoID uuid.UUID `json:"servico_id"`
	Data      string    `json:"data"`
	Hora      string    `json:"hora"`
	Status    string    `json:"status"`
}

// ResponsavelTecnico é o cadastro exigido por alguns serviços.
type ResponsavelTecnico struct {
	ID    uuid.UUID `json:"id"`
	CPF   string    `json:"cpf"`
	Nome  string    `json:"nome"`
	Ativo bool      `json:"ativo"`
}

// Consulta é o recorte público pelo número.
type Consulta struct {
	Numero       string       `json:"numero"`
	Status       string       `json:"status"`
	StatusRotulo string       `json:"status_rotulo"`
	Prioridade   string       `json:"prioridade"`
	Nome         string       `json:"nome"`
	Servico      string       `json:"servico"`
	Unidade      string       `json:"unidade"`
	CriadoEm     time.Time    `json:"criado_em"`
	PrazoEm      *time.Time   `json:"prazo_em,omitempty"`
	Complemento  string       `json:"complemento,omitempty"`
	Historico    []Historico  `json:"historico"`
	Agendamento  *Agendamento `json:"agendamento,omitempty"`
}

// Visao é o recorte da listagem da equipe.
type Visao struct {
	Tudo      bool
	Unidades  []uuid.UUID
	Entidades []uuid.UUID
}

// NovoFormulario valida o desenho.
func NovoFormulario(unidade uuid.UUID, titulo, descricao, instrucao string, campos []Campo, agora time.Time) (Formulario, error) {
	if unidade == uuid.Nil {
		return Formulario{}, ErrInvalido
	}
	titulo = strings.TrimSpace(titulo)
	if utf8.RuneCountInString(titulo) < 2 || utf8.RuneCountInString(titulo) > 200 {
		return Formulario{}, ErrInvalido
	}
	if err := ValidarCampos(campos); err != nil {
		return Formulario{}, err
	}
	if campos == nil {
		campos = []Campo{}
	}
	return Formulario{
		ID: uuid.New(), UnidadeID: unidade, Titulo: titulo,
		Descricao: strings.TrimSpace(descricao), Instrucao: strings.TrimSpace(instrucao),
		Campos: campos, Ativo: true, CreatedAt: agora, UpdatedAt: agora,
	}, nil
}

// ValidarCampos confere tipos, identificadores e opções.
func ValidarCampos(campos []Campo) error {
	vistos := map[string]bool{}
	for _, c := range campos {
		if !idCampo.MatchString(c.ID) || !slices.Contains(tiposCampo, c.Tipo) {
			return ErrInvalido
		}
		if vistos[c.ID] || strings.TrimSpace(c.Rotulo) == "" || utf8.RuneCountInString(c.Rotulo) > 200 {
			return ErrInvalido
		}
		vistos[c.ID] = true
		if c.Tipo == "select" || c.Tipo == "radio" || c.Tipo == "checkbox_group" {
			if len(c.Opcoes) == 0 {
				return ErrInvalido
			}
			for _, op := range c.Opcoes {
				if strings.TrimSpace(op) == "" {
					return ErrInvalido
				}
			}
		}
	}
	for _, c := range campos {
		for _, id := range c.Revela {
			if !vistos[id] || id == c.ID || c.Tipo != "boolean" {
				return ErrInvalido
			}
		}
	}
	return nil
}

// RespostasValidas confere o preenchimento contra o desenho.
func RespostasValidas(campos []Campo, dados map[string]any) error {
	if dados == nil {
		dados = map[string]any{}
	}
	for _, c := range campos {
		if c.Tipo == "page" || oculto(campos, dados, c.ID) {
			continue
		}
		valor, tem := dados[c.ID]
		if !tem || vazio(valor) {
			if c.Obrigatorio {
				return ErrInvalido
			}
			continue
		}
		if err := valorAceito(c, valor); err != nil {
			return err
		}
	}
	return nil
}

func oculto(campos []Campo, dados map[string]any, id string) bool {
	for _, c := range campos {
		if c.Tipo != "boolean" || !slices.Contains(c.Revela, id) {
			continue
		}
		return !verdadeiro(dados[c.ID])
	}
	return false
}

func verdadeiro(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func vazio(v any) bool {
	if v == nil {
		return true
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t) == ""
	case []any:
		return len(t) == 0
	default:
		return false
	}
}

func valorAceito(c Campo, v any) error {
	switch c.Tipo {
	case "number":
		if _, ok := numero(v); !ok {
			return ErrInvalido
		}
	case "email":
		s, ok := v.(string)
		if !ok || !strings.Contains(s, "@") || !strings.Contains(s, ".") {
			return ErrInvalido
		}
	case "cpf":
		if digitos(v) != 11 {
			return ErrInvalido
		}
	case "cnpj":
		if digitos(v) != 14 {
			return ErrInvalido
		}
	case "checkbox", "boolean":
		if _, ok := v.(bool); !ok {
			return ErrInvalido
		}
	case "checkbox_group":
		lista, ok := v.([]any)
		if !ok {
			return ErrInvalido
		}
		for _, item := range lista {
			s, ok := item.(string)
			if !ok || !slices.Contains(c.Opcoes, s) {
				return ErrInvalido
			}
		}
	case "select", "radio":
		s, ok := v.(string)
		if !ok || !slices.Contains(c.Opcoes, s) {
			return ErrInvalido
		}
	case "map", "ambulante":
		s, ok := v.(string)
		if !ok || !Coordenada(s) {
			return ErrInvalido
		}
	case "map_area":
		s, ok := v.(string)
		partes := strings.Split(s, ";")
		if !ok || len(partes) != 2 || !Coordenada(partes[0]) || !Coordenada(partes[1]) {
			return ErrInvalido
		}
	case "file", "text", "phone", "date", "time", "password", "url", "cep", "textarea":
		s, ok := v.(string)
		if !ok || strings.TrimSpace(s) == "" {
			return ErrInvalido
		}
	default:
		return ErrInvalido
	}
	return nil
}

func numero(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func digitos(v any) int {
	s, ok := v.(string)
	if !ok {
		return 0
	}
	return len(digitosSo.ReplaceAllString(s, ""))
}

// NovoServico valida a carta.
func NovoServico(s Servico, agora time.Time) (Servico, error) {
	s.Nome = strings.TrimSpace(s.Nome)
	s.Descricao = strings.TrimSpace(s.Descricao)
	s.Categoria = strings.TrimSpace(s.Categoria)
	if s.UnidadeID == uuid.Nil || s.FormularioID == uuid.Nil {
		return Servico{}, ErrInvalido
	}
	if utf8.RuneCountInString(s.Nome) < 2 || utf8.RuneCountInString(s.Nome) > 200 || utf8.RuneCountInString(s.Categoria) > 80 {
		return Servico{}, ErrInvalido
	}
	if s.PrazoDias < 0 || s.TaxaCentavos < 0 {
		return Servico{}, ErrInvalido
	}
	if err := s.Workflow.Validar(); err != nil {
		return Servico{}, err
	}
	if s.ExigeAgendamento {
		if err := s.Agenda.Validar(); err != nil {
			return Servico{}, err
		}
	}
	if err := validarOrientacao(s.Requisitos); err != nil || validarOrientacao(s.Documentos) != nil {
		return Servico{}, ErrInvalido
	}
	if s.Requisitos == nil {
		s.Requisitos = []Orientacao{}
	}
	if s.Documentos == nil {
		s.Documentos = []Orientacao{}
	}
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = agora
	}
	s.Ativo = true
	s.UpdatedAt = agora
	return s, nil
}

func validarOrientacao(lista []Orientacao) error {
	for _, o := range lista {
		if utf8.RuneCountInString(strings.TrimSpace(o.Titulo)) < 2 || utf8.RuneCountInString(o.Titulo) > 200 {
			return ErrInvalido
		}
	}
	return nil
}

// Validar confere o workflow. Desligado, aceita o que houver.
func (w Workflow) Validar() error {
	if !w.Ligado {
		return nil
	}
	if len(w.Etapas) == 0 {
		return ErrInvalido
	}
	vistos := map[string]bool{}
	for _, e := range w.Etapas {
		if !idCampo.MatchString(e.ID) || vistos[e.ID] || e.Ordem < 0 || e.PrazoDias < 0 {
			return ErrInvalido
		}
		if e.Status != "" && !StatusConhecido(e.Status) {
			return ErrInvalido
		}
		switch e.ResponsavelTipo {
		case "", "unidade", "departamento", "usuario":
		default:
			return ErrInvalido
		}
		if e.ResponsavelTipo != "" {
			if _, err := uuid.Parse(e.ResponsavelID); err != nil {
				return ErrInvalido
			}
		}
		vistos[e.ID] = true
	}
	return nil
}

// Primeira devolve a etapa de menor ordem.
func (w Workflow) Primeira() (Etapa, bool) {
	if !w.Ligado || len(w.Etapas) == 0 {
		return Etapa{}, false
	}
	primeira := w.Etapas[0]
	for _, e := range w.Etapas[1:] {
		if e.Ordem < primeira.Ordem {
			primeira = e
		}
	}
	return primeira, true
}

// Seguinte devolve a etapa depois da atual.
func (w Workflow) Seguinte(atual string) (Etapa, error) {
	if !w.Ligado || len(w.Etapas) == 0 {
		return Etapa{}, ErrInvalido
	}
	ordenadas := append([]Etapa(nil), w.Etapas...)
	slices.SortFunc(ordenadas, func(a, b Etapa) int { return a.Ordem - b.Ordem })
	if atual == "" {
		return ordenadas[0], nil
	}
	for i, e := range ordenadas {
		if e.ID != atual {
			continue
		}
		if i == len(ordenadas)-1 {
			return Etapa{}, ErrInvalido
		}
		return ordenadas[i+1], nil
	}
	return ordenadas[0], nil
}

// Validar confere a grade.
func (a Agenda) Validar() error {
	if a.DuracaoMin < 1 || a.VagasDia < 1 || !horaMinuto.MatchString(a.Inicio) || !horaMinuto.MatchString(a.Fim) || a.Inicio >= a.Fim {
		return ErrInvalido
	}
	if len(a.Dias) == 0 {
		return ErrInvalido
	}
	for _, d := range a.Dias {
		if d < 0 || d > 6 {
			return ErrInvalido
		}
	}
	return nil
}

// Cabe diz se o dia da semana está na grade.
func (a Agenda) Cabe(dia time.Time) bool {
	return slices.Contains(a.Dias, int(dia.In(Cuiaba).Weekday()))
}

// HorarioLivre calcula o próximo intervalo do dia a partir de quantos já foram marcados.
func (a Agenda) HorarioLivre(ocupados int) (string, bool) {
	if ocupados < 0 || ocupados >= a.VagasDia {
		return "", false
	}
	inicio := minutos(a.Inicio) + ocupados*a.DuracaoMin
	if inicio+a.DuracaoMin > minutos(a.Fim) {
		return "", false
	}
	return fmt.Sprintf("%02d:%02d", inicio/60, inicio%60), true
}

func minutos(hhmm string) int {
	h, _ := strconv.Atoi(hhmm[:2])
	m, _ := strconv.Atoi(hhmm[3:])
	return h*60 + m
}

// ProcurarVaga anda até 30 dias a partir de amanhã em Cuiabá.
func ProcurarVaga(a Agenda, agora time.Time, ocupados func(dia string) (int, error)) (string, string, error) {
	if err := a.Validar(); err != nil {
		return "", "", err
	}
	base := agora.In(Cuiaba)
	inicio := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, Cuiaba).AddDate(0, 0, 1)
	for i := 0; i < LimiteBuscaAgendamento; i++ {
		dia := inicio.AddDate(0, 0, i)
		if !a.Cabe(dia) {
			continue
		}
		chave := dia.Format("2006-01-02")
		n, err := ocupados(chave)
		if err != nil {
			return "", "", err
		}
		if hora, ok := a.HorarioLivre(n); ok {
			return chave, hora, nil
		}
	}
	return "", "", ErrSemVaga
}

// StatusDoHorario: no dia, confirmado; depois, agendado.
func StatusDoHorario(data, hoje string) (string, error) {
	if !dataValida(data) || !dataValida(hoje) || data < hoje {
		return "", ErrInvalido
	}
	if data == hoje {
		return "confirmado", nil
	}
	return "agendado", nil
}

func dataValida(s string) bool { return regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(s) }

// BloqueioPorFaltas: duas faltas travam novos horários por 30 dias.
func BloqueioPorFaltas(faltas int, ultima, agora time.Time) bool {
	if faltas < 2 || ultima.IsZero() {
		return false
	}
	return agora.Before(ultima.Add(30 * 24 * time.Hour))
}

// Iniciais prefere a sigla. Sem ela, junta a primeira letra de cada palavra, até quatro.
func Iniciais(sigla, nome string) string {
	sigla = letras(strings.ToUpper(strings.TrimSpace(sigla)))
	if sigla != "" {
		r := []rune(sigla)
		if len(r) > 4 {
			return string(r[:4])
		}
		return sigla
	}
	var b strings.Builder
	for _, palavra := range strings.Fields(nome) {
		for _, r := range palavra {
			if unicode.IsLetter(r) {
				b.WriteRune(unicode.ToUpper(r))
				break
			}
		}
	}
	out := b.String()
	if out == "" {
		return "PROT"
	}
	r := []rune(out)
	if len(r) > 4 {
		return string(r[:4])
	}
	return out
}

func letras(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// MontarNumero junta sigla, ano e cinco dígitos.
func MontarNumero(iniciais string, ano, sufixo int) (string, error) {
	if iniciais == "" || ano < 2000 || sufixo < 0 || sufixo > 99999 {
		return "", ErrInvalido
	}
	return fmt.Sprintf("%s%d%05d", iniciais, ano, sufixo), nil
}

// DigitosCPF devolve só os números.
func DigitosCPF(s string) string { return digitosSo.ReplaceAllString(s, "") }

// CPFMascarado deixa só os três dígitos do meio e os dois do fim. A lista da mesa não devolve o número inteiro.
func CPFMascarado(s string) string {
	d := DigitosCPF(s)
	if len(d) != 11 {
		return ""
	}
	return "***." + d[3:6] + "." + d[6:9] + "-" + d[9:]
}

// NovoResponsavel valida o cadastro técnico.
func NovoResponsavel(nome, cpf string) (ResponsavelTecnico, error) {
	nome = strings.TrimSpace(nome)
	cpf = DigitosCPF(cpf)
	if utf8.RuneCountInString(nome) < 2 || len(cpf) != 11 {
		return ResponsavelTecnico{}, ErrInvalido
	}
	return ResponsavelTecnico{ID: uuid.New(), CPF: cpf, Nome: nome, Ativo: true}, nil
}

// Prioridade aceita a lista fechada e cai em normal.
func Prioridade(s string) string {
	if slices.Contains(prioridades, s) {
		return s
	}
	return PrioridadeNormal
}

// Abrir monta o pedido a partir do serviço ativo e das respostas já validadas.
func Abrir(s Servico, f Formulario, numero, nome, cpf, email, telefone, prioridade string, respostas map[string]any, usuario *uuid.UUID, agora time.Time) (Pedido, error) {
	if !s.Ativo || !f.Ativo || s.FormularioID != f.ID {
		return Pedido{}, ErrInvalido
	}
	nome = strings.TrimSpace(nome)
	cpf = DigitosCPF(cpf)
	if utf8.RuneCountInString(nome) < 2 || len(cpf) != 11 || numero == "" {
		return Pedido{}, ErrInvalido
	}
	if err := RespostasValidas(f.Campos, respostas); err != nil {
		return Pedido{}, err
	}
	if respostas == nil {
		respostas = map[string]any{}
	}
	p := Pedido{
		ID: uuid.New(), Numero: numero, ServicoID: s.ID, ServicoNome: s.Nome,
		FormularioID: f.ID, UnidadeID: s.UnidadeID, Nome: nome, CPF: cpf,
		Email: strings.TrimSpace(email), Telefone: strings.TrimSpace(telefone),
		Respostas: respostas, Campos: f.Campos, Status: StatusNovo, Prioridade: Prioridade(prioridade),
		UsuarioID: usuario, CreatedAt: agora, UpdatedAt: agora,
	}
	if etapa, ok := s.Workflow.Primeira(); ok {
		var err error
		p, err = p.aplicar(etapa, agora)
		if err != nil {
			return Pedido{}, err
		}
	} else if s.AprovaSozinho {
		p.Status = StatusAprovado
	}
	if p.PrazoEm == nil && s.PrazoDias > 0 {
		limite := SomarDiasUteis(agora, s.PrazoDias)
		p.PrazoEm = &limite
	}
	return p, nil
}

func (p Pedido) aplicar(e Etapa, agora time.Time) (Pedido, error) {
	if e.Status != "" {
		if !StatusConhecido(e.Status) {
			return Pedido{}, ErrInvalido
		}
		p.Status = e.Status
	}
	p.EtapaID = e.ID
	if e.PrazoDias > 0 {
		limite := SomarDiasUteis(agora, e.PrazoDias)
		p.PrazoEm = &limite
	}
	switch e.ResponsavelTipo {
	case "":
	case "unidade":
		id, err := uuid.Parse(e.ResponsavelID)
		if err != nil {
			return Pedido{}, ErrInvalido
		}
		p.UnidadeID = id
		p.DepartamentoID = nil
		p.ResponsavelID = nil
	case "departamento":
		id, err := uuid.Parse(e.ResponsavelID)
		if err != nil {
			return Pedido{}, ErrInvalido
		}
		p.DepartamentoID = &id
		p.ResponsavelID = nil
	case "usuario":
		id, err := uuid.Parse(e.ResponsavelID)
		if err != nil {
			return Pedido{}, ErrInvalido
		}
		p.ResponsavelID = &id
	default:
		return Pedido{}, ErrInvalido
	}
	p.UpdatedAt = agora
	return p, nil
}

func (p Pedido) aberto() error {
	if EstadoFinal(p.Status) {
		return ErrEstadoFinal
	}
	return nil
}

// Complementar pede informação e trava o avanço.
func (p Pedido) Complementar(mensagem string, agora time.Time) (Pedido, error) {
	if err := p.aberto(); err != nil {
		return Pedido{}, err
	}
	if p.Status == StatusComplementacao {
		return Pedido{}, ErrComplemento
	}
	mensagem = strings.TrimSpace(mensagem)
	if mensagem == "" || utf8.RuneCountInString(mensagem) > 2000 {
		return Pedido{}, ErrInvalido
	}
	p.Status = StatusComplementacao
	p.Complemento = mensagem
	p.UpdatedAt = agora
	return p, nil
}

// ResponderComplemento devolve o pedido ao andamento.
func (p Pedido) ResponderComplemento(agora time.Time) (Pedido, error) {
	if err := p.aberto(); err != nil {
		return Pedido{}, err
	}
	if p.Status != StatusComplementacao {
		return Pedido{}, ErrInvalido
	}
	p.Status = StatusAndamento
	p.Complemento = ""
	p.UpdatedAt = agora
	return p, nil
}

// Avancar passa à etapa seguinte. Complemento aberto recusa.
func (p Pedido) Avancar(w Workflow, agora time.Time) (Pedido, error) {
	if err := p.aberto(); err != nil {
		return Pedido{}, err
	}
	if p.Status == StatusComplementacao {
		return Pedido{}, ErrComplemento
	}
	etapa, err := w.Seguinte(p.EtapaID)
	if err != nil {
		return Pedido{}, err
	}
	return p.aplicar(etapa, agora)
}

// Concluir grava a decisão.
func (p Pedido) Concluir(decisao, justificativa string, agora time.Time) (Pedido, error) {
	if err := p.aberto(); err != nil {
		return Pedido{}, err
	}
	if p.Status == StatusComplementacao {
		return Pedido{}, ErrComplemento
	}
	if decisao != DecisaoFavoravel && decisao != DecisaoDesfavoravel {
		return Pedido{}, ErrInvalido
	}
	justificativa = strings.TrimSpace(justificativa)
	if justificativa == "" || utf8.RuneCountInString(justificativa) > 4000 {
		return Pedido{}, ErrInvalido
	}
	p.Status = StatusConcluido
	p.Decisao = decisao
	p.Justificativa = justificativa
	p.UpdatedAt = agora
	return p, nil
}

// Cancelar encerra o pedido.
func (p Pedido) Cancelar(agora time.Time) (Pedido, error) {
	if err := p.aberto(); err != nil {
		return Pedido{}, err
	}
	p.Status = StatusCancelado
	p.UpdatedAt = agora
	return p, nil
}

// Publico retira CPF, telefone e respostas. O anonimato troca o nome.
func Publico(p Pedido, campoAnonimo string, historico []Historico, agenda *Agendamento) Consulta {
	nome := p.Nome
	if campoAnonimo != "" && verdadeiro(p.Respostas[campoAnonimo]) {
		nome = "Denúncia anônima"
	}
	if historico == nil {
		historico = []Historico{}
	}
	publico := make([]Historico, len(historico))
	for i, h := range historico {
		h.Anexos = nil
		publico[i] = h
	}
	historico = publico
	var comp string
	if p.Status == StatusComplementacao {
		comp = p.Complemento
	}
	return Consulta{
		Numero: p.Numero, Status: p.Status, StatusRotulo: RotuloStatus(p.Status), Prioridade: p.Prioridade, Nome: nome,
		Servico: p.ServicoNome, Unidade: p.UnidadeNome, CriadoEm: p.CreatedAt,
		PrazoEm: p.PrazoEm, Complemento: comp, Historico: historico, Agendamento: agenda,
	}
}

// Aviso é o texto que pode sair do sistema: número, fato e orientação.
func Aviso(numero, fato, orientacao string) (string, error) {
	numero, fato, orientacao = strings.TrimSpace(numero), strings.TrimSpace(fato), strings.TrimSpace(orientacao)
	if numero == "" || fato == "" || orientacao == "" {
		return "", ErrInvalido
	}
	return numero + ": " + fato + ". " + orientacao, nil
}
