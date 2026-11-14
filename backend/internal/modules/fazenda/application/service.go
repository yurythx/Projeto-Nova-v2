// Package application executa o protocolo da Fazenda. A regra fica no
// domínio; aqui entram a transação, o evento e a auditoria.
package application

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/govbr"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

// IntervaloAlertas é o ritmo do processo que publica o aviso de prazo.
const IntervaloAlertas = time.Hour

var registrarCaixa = notificacoes.Registrar

const (
	EventoProtocolado = "fazenda.processo.protocolado"
	EventoExigencia   = "fazenda.processo.exigencia"
	EventoDecidido    = "fazenda.processo.decidido"
	EventoAviso       = "fazenda.aviso.externo"
	EventoConsulta    = "fazenda.processo.consulta"
)

// Trilha grava o evento de saída e a auditoria na mesma transação.
type Trilha interface {
	Registrar(ctx context.Context, tx pgx.Tx, acao, id string, depois any, aviso string) error
}

// Nucleo é a trilha do processo: outbox e auditoria no mesmo tx.
type Nucleo struct{ Out *outbox.Writer }

func (n Nucleo) Registrar(ctx context.Context, tx pgx.Tx, acao, id string, depois any, aviso string) error {
	if n.Out == nil || tx == nil {
		return errors.New("fazenda: trilha indisponível")
	}
	payload := map[string]string{"id": id}
	if aviso != "" {
		payload["aviso"] = aviso
	}
	if err := n.Out.Write(ctx, tx, acao, "fazenda_processo", id, uuid.Nil, payload); err != nil {
		return err
	}
	return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, acao, "fazenda_processo", id, nil, depois))
}

var avisoDeProtocolo = func(numero string) (string, error) {
	return domain.ModeloAviso(numero, "recibo")
}

type trilhaVazia struct{}

func (trilhaVazia) Registrar(context.Context, pgx.Tx, string, string, any, string) error { return nil }

// Service é o caso de uso do protocolo.
type Service struct {
	leitor    database.DBTX
	repo      domain.Repositorio
	trilha    Trilha
	gov       govbr.Config
	logger    *slog.Logger
	transacao func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error
	agora     func() time.Time
	http      govHTTP
	arquivos  *Arquivos
	sino      *notificacoes.Service
	intervalo time.Duration
}

// ComSino liga a caixa da plataforma. O e-mail e o WhatsApp saem dela quando o canal está ligado.
func (s *Service) ComSino(n *notificacoes.Service) *Service {
	if n != nil {
		s.sino = n
	}
	return s
}

type govHTTP interface {
	Trocar(cfg govbr.Config, code, verifier string) (govbr.Identidade, error)
}

type govPadrao struct{}

func (govPadrao) Trocar(cfg govbr.Config, code, verifier string) (govbr.Identidade, error) {
	return cfg.Trocar(nil, code, verifier)
}

// NewService cria o serviço. O relógio e a transação têm padrão de produção.
func NewService(pool *pgxpool.Pool, repo domain.Repositorio, trilha Trilha, gov govbr.Config, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	if trilha == nil {
		trilha = trilhaVazia{}
	}
	s := &Service{
		leitor: pool, repo: repo, trilha: trilha, gov: gov, logger: logger,
		agora: time.Now, http: govPadrao{},
	}
	s.transacao = func(ctx context.Context, fn func(context.Context, pgx.Tx) error) error {
		return database.WithTx(ctx, pool, fn)
	}
	return s
}

// MapError traduz a regra para o erro que o HTTP conhece.
func MapError(err error) error {
	if err == nil {
		return nil
	}
	var rec domain.Recusa
	switch {
	case errors.Is(err, domain.ErrNaoEncontrado):
		return apperrors.NotFound("processo não encontrado")
	case errors.As(err, &rec):
		return apperrors.Validation(rec.Msg)
	case errors.Is(err, govbr.ErrSemCredencial):
		return apperrors.DependencyUnavailable("credencial do Login Único ainda não configurada")
	}
	if _, ok := apperrors.As(err); ok {
		return err
	}
	return apperrors.Internal(err)
}

func equipe(id auth.Identity) bool {
	return auth.HasPermission(id, auth.PermFazendaAnalisar) || auth.HasPermission(id, auth.PermFazendaDecidir)
}

// Simular não abre processo.
func (s *Service) Simular(in domain.Simulacao) (domain.Simulado, error) {
	return domain.Simular(in, s.agora())
}

// IniciarGovBR devolve a URL do Login Único, ou a falta da credencial.
func (s *Service) IniciarGovBR() (govbr.Desafio, error) {
	return s.gov.Iniciar()
}

// SessaoGovBR é o que a área restrita pode mostrar. O CPF não sai.
type SessaoGovBR struct {
	Sub   string `json:"sub"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
}

// TrocarGovBR conclui o login e esconde o CPF.
func (s *Service) TrocarGovBR(code, verifier string) (SessaoGovBR, error) {
	id, err := s.http.Trocar(s.gov, code, verifier)
	if err != nil {
		return SessaoGovBR{}, err
	}
	return SessaoGovBR{Sub: id.Sub, Nome: id.Nome, Email: id.Email}, nil
}

// Autenticidade confirma o código. Não achou é resposta pública, não erro.
func (s *Service) Autenticidade(ctx context.Context, codigo string) (domain.Autenticidade, error) {
	p, err := s.repo.PorCodigo(ctx, s.leitor, codigo)
	if errors.Is(err, domain.ErrNaoEncontrado) {
		return domain.Autenticidade{Codigo: codigo, Valido: false, Mensagem: "código não confere"}, nil
	}
	if err != nil {
		return domain.Autenticidade{}, err
	}
	return domain.ConfirmarAutenticidade(p, codigo), nil
}

// Abrir protocola em nome do titular. Titular vazio é o próprio chamador.
func (s *Service) Abrir(ctx context.Context, quem auth.Identity, a domain.Abertura) (domain.Processo, error) {
	if a.Titular == "" {
		a.Titular = quem.Subject
	}
	if a.Titular != quem.Subject && !auth.HasPermission(quem, auth.PermFazendaProtocolar) {
		return domain.Processo{}, apperrors.Forbidden("protocolar em nome de outra pessoa exige a permissão da unidade")
	}
	a.Agora = s.agora()
	a.Ano = a.Agora.In(domain.Cuiaba).Year()
	var out domain.Processo
	var enviadas []notificacoes.Enviada
	err := s.transacao(ctx, func(ctx context.Context, tx pgx.Tx) error {
		seq, err := s.repo.Proximo(ctx, tx, a.Ano)
		if err != nil {
			return err
		}
		a.Seq = seq
		p, err := domain.Abrir(a)
		if err != nil {
			return err
		}
		if quem.UserID != uuid.Nil && p.Titular == quem.Subject {
			p.Destinatario = quem.UserID
		}
		if err := s.repo.Inserir(ctx, tx, p); err != nil {
			return err
		}
		aviso, err := avisoDeProtocolo(p.Numero)
		if err != nil {
			return err
		}
		enviadas, err = s.caixa(ctx, tx, p, EventoProtocolado, aviso)
		if err != nil {
			return err
		}
		out = p
		return s.trilha.Registrar(ctx, tx, EventoProtocolado, p.ID.String(), p, aviso)
	})
	if err != nil {
		return domain.Processo{}, err
	}
	s.entregarCaixa(ctx, enviadas)
	s.logger.Info("fazenda: processo protocolado", slog.String("numero", out.Numero))
	return out, nil
}

// Listar devolve a fila para a equipe e só os próprios processos para o contribuinte.
func (s *Service) Listar(ctx context.Context, quem auth.Identity) ([]domain.Processo, error) {
	titular := ""
	if !equipe(quem) {
		titular = quem.Subject
	}
	lista, err := s.repo.Listar(ctx, s.leitor, titular)
	if err != nil {
		return nil, err
	}
	if lista == nil {
		return []domain.Processo{}, nil
	}
	for i := range lista {
		lista[i].Pendencia = domain.Pendencia(lista[i], s.agora())
	}
	return lista, nil
}

// Painel mostra o relatório institucional e os marcos do despacho. É da equipe.
func (s *Service) Painel(ctx context.Context, quem auth.Identity) ([]domain.PrazoPainel, error) {
	if !equipe(quem) {
		return nil, apperrors.Forbidden("o painel de prazos é da equipe")
	}
	out := domain.PrazosDoPainel(s.agora())
	lista, err := s.repo.Listar(ctx, s.leitor, "")
	if err != nil {
		return nil, err
	}
	for _, p := range lista {
		out = append(out, domain.MarcosDoProcesso(p, s.agora())...)
	}
	return out, nil
}

// Alertar publica o aviso de prazo uma vez por marco, no mesmo evento do canal.
func (s *Service) Alertar(ctx context.Context, quem auth.Identity) (int, error) {
	if !equipe(quem) {
		return 0, apperrors.Forbidden("o alerta de prazo é da equipe")
	}
	return s.publicarAlertas(ctx)
}

// RunAlertas repete a publicação até o processo encerrar. O marco já avisado não sai de novo.
func (s *Service) RunAlertas(ctx context.Context) error {
	intervalo := s.intervalo
	if intervalo <= 0 {
		intervalo = IntervaloAlertas
	}
	t := time.NewTicker(intervalo)
	defer t.Stop()
	for {
		if _, err := s.publicarAlertas(ctx); err != nil && ctx.Err() == nil && s.logger != nil {
			s.logger.WarnContext(ctx, "fazenda: alerta de prazo falhou", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

func (s *Service) publicarAlertas(ctx context.Context) (int, error) {
	lista, err := s.repo.Listar(ctx, s.leitor, "")
	if err != nil {
		return 0, err
	}
	publicadas := 0
	for _, p := range lista {
		limite, ok := domain.MarcoEmAlerta(p, s.agora())
		if !ok {
			continue
		}
		if _, err := s.alterar(ctx, p.ID, EventoAviso, func(atual domain.Processo) (domain.Processo, string, error) {
			aviso, err := domain.ModeloAviso(atual.Numero, "alerta")
			if err != nil {
				return atual, "", err
			}
			atual.AlertaLimite = limite
			return atual, aviso, nil
		}); err != nil {
			return publicadas, err
		}
		publicadas++
	}
	return publicadas, nil
}

func (s *Service) visivel(quem auth.Identity, p domain.Processo) error {
	if equipe(quem) || p.Titular == quem.Subject {
		return nil
	}
	return apperrors.Forbidden("o processo é de outra pessoa")
}

// Buscar lê um processo se for da pessoa ou da equipe.
func (s *Service) Buscar(ctx context.Context, quem auth.Identity, id uuid.UUID) (domain.Processo, error) {
	p, err := s.repo.Buscar(ctx, s.leitor, id)
	if err != nil {
		return domain.Processo{}, err
	}
	if err := s.visivel(quem, p); err != nil {
		return domain.Processo{}, err
	}
	p.Pendencia = domain.Pendencia(p, s.agora())
	pecas, err := s.repo.ListarPecas(ctx, s.leitor, id)
	if err != nil {
		return domain.Processo{}, err
	}
	p = p.ComCiencia().ComAtos(s.agora())
	p.Pecas = s.links(ctx, pecas)
	return p, nil
}

// RegistrarConsulta grava a ciência quando o titular abre o processo.
// A equipe vê as datas e não altera a consulta.
func (s *Service) RegistrarConsulta(ctx context.Context, quem auth.Identity, id uuid.UUID) (domain.Processo, error) {
	p, err := s.Buscar(ctx, quem, id)
	if err != nil {
		return domain.Processo{}, err
	}
	if quem.UserID == uuid.Nil || quem.UserID != p.Destinatario {
		return p, nil
	}
	out, err := s.alterar(ctx, id, EventoConsulta, func(atual domain.Processo) (domain.Processo, string, error) {
		atual.ConsultaEm = s.agora()
		return atual, "", nil
	})
	if err != nil {
		return domain.Processo{}, err
	}
	out.Pendencia = domain.Pendencia(out, s.agora())
	return out.ComCiencia(), nil
}

func (s *Service) alterar(ctx context.Context, id uuid.UUID, acao string, fn func(domain.Processo) (domain.Processo, string, error)) (domain.Processo, error) {
	var out domain.Processo
	var enviadas []notificacoes.Enviada
	err := s.transacao(ctx, func(ctx context.Context, tx pgx.Tx) error {
		p, err := s.repo.Buscar(ctx, tx, id)
		if err != nil {
			return err
		}
		p, aviso, err := fn(p)
		if err != nil {
			return err
		}
		if err := s.repo.Atualizar(ctx, tx, p); err != nil {
			return err
		}
		enviadas, err = s.caixa(ctx, tx, p, acao, aviso)
		if err != nil {
			return err
		}
		out = p
		return s.trilha.Registrar(ctx, tx, acao, p.ID.String(), p, aviso)
	})
	if err != nil {
		return domain.Processo{}, err
	}
	s.entregarCaixa(ctx, enviadas)
	return out, nil
}

func (s *Service) caixa(ctx context.Context, tx pgx.Tx, p domain.Processo, acao, aviso string) ([]notificacoes.Enviada, error) {
	if s.sino == nil || tx == nil || aviso == "" || p.Destinatario == uuid.Nil {
		return nil, nil
	}
	msg := aviso
	if utf8.RuneCountInString(msg) > 1000 {
		msg = string([]rune(msg)[:1000])
	}
	sum := sha256.Sum256([]byte(acao + aviso))
	return registrarCaixa(ctx, tx, []uuid.UUID{p.Destinatario}, notificacoes.Nova{
		Modulo:   "fazenda",
		Titulo:   "Protocolo da Fazenda",
		Mensagem: msg,
		Link:     "/fazenda/" + p.ID.String(),
		Chave:    fmt.Sprintf("fazenda:%s:%x", p.ID, sum[:8]),
	})
}

func (s *Service) entregarCaixa(ctx context.Context, enviadas []notificacoes.Enviada) {
	if s.sino != nil && len(enviadas) > 0 {
		s.sino.Entregar(ctx, enviadas)
	}
}

func (s *Service) daEquipe(quem auth.Identity) error {
	if !equipe(quem) {
		return apperrors.Forbidden("a análise é da equipe da Fazenda")
	}
	return nil
}

// Exigir abre a exigência única e suspende a análise.
func (s *Service) Exigir(ctx context.Context, quem auth.Identity, id uuid.UUID, fatoNovo bool) (domain.Processo, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoExigencia, func(p domain.Processo) (domain.Processo, string, error) {
		ja := p.Situacao == domain.SituacaoExigencia || p.Situacao == domain.SituacaoArquivado
		e, err := domain.NovaExigencia(p.Programa, s.agora(), ja, fatoNovo, nil)
		if err != nil {
			return p, "", err
		}
		aviso, err := domain.ModeloAviso(p.Numero, "exigencia")
		if err != nil {
			return p, "", err
		}
		return p.AplicarExigencia(e), aviso, nil
	})
}

// Responder encerra a exigência ou arquiva pelo prazo.
func (s *Service) Responder(ctx context.Context, quem auth.Identity, id uuid.UUID, respondeu bool) (domain.Processo, error) {
	if _, err := s.Buscar(ctx, quem, id); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoExigencia, func(p domain.Processo) (domain.Processo, string, error) {
		e, err := p.ExigenciaAbertaNoProcesso()
		if err != nil {
			return p, "", err
		}
		return p.ResponderExigencia(e, s.agora(), respondeu), "", nil
	})
}

// Decidir fecha a análise.
func (s *Service) Decidir(ctx context.Context, quem auth.Identity, id uuid.UUID) (domain.Processo, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoDecidido, func(p domain.Processo) (domain.Processo, string, error) {
		p, err := p.EmitirDecisao()
		if err != nil {
			return p, "", err
		}
		p = p.RegistrarDespacho(s.agora())
		aviso, err := domain.ModeloAviso(p.Numero, "conclusao")
		return p, aviso, err
	})
}

// ReceberDUA valida a planilha na área restrita. O recibo não leva o documento fiscal.
func (s *Service) ReceberDUA(linhas []domain.LinhaDUA, omitidas int) (domain.ReciboDUA, error) {
	return domain.ReceberDUA(linhas, omitidas)
}

// Recorrer registra o recurso quando o rito da Lei 1.800/1990 já foi entregue.
func (s *Service) Recorrer(ctx context.Context, quem auth.Identity, id uuid.UUID, rito bool) (domain.Processo, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoDecidido, func(p domain.Processo) (domain.Processo, string, error) {
		p, err := p.Recorrer(rito)
		return p, "", err
	})
}

// GuardarReducao anota a alíquota. A guia é outra fase.
func (s *Service) GuardarReducao(ctx context.Context, quem auth.Identity, id uuid.UUID, milessimos int64, condicoes int) (domain.Processo, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoDecidido, func(p domain.Processo) (domain.Processo, string, error) {
		p, err := p.GuardarReducao(milessimos, condicoes)
		return p, "", err
	})
}

// Parcelar calcula o plano e deixa o estado em aguardando pagamento.
func (s *Service) Parcelar(ctx context.Context, quem auth.Identity, id uuid.UUID, total int64, parcelas int) (domain.Plano, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Plano{}, err
	}
	plano, err := domain.Parcelar(total, parcelas, s.agora())
	if err != nil {
		return domain.Plano{}, err
	}
	_, err = s.alterar(ctx, id, EventoAviso, func(p domain.Processo) (domain.Processo, string, error) {
		p.Situacao = plano.Estado
		return p, "", nil
	})
	if err != nil {
		return domain.Plano{}, err
	}
	return plano, nil
}

// Baixar aplica o arquivo de retorno. Sem a primeira parcela, continua aguardando.
func (s *Service) Baixar(ctx context.Context, quem auth.Identity, id uuid.UUID, primeira bool) (domain.Processo, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoAviso, func(p domain.Processo) (domain.Processo, string, error) {
		p.Situacao = domain.BaixarRetorno(primeira)
		return p, "", nil
	})
}

// MarcarExecucao sinaliza a Procuradoria.
func (s *Service) MarcarExecucao(ctx context.Context, quem auth.Identity, id uuid.UUID, emExecucao bool) (domain.Processo, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoAviso, func(p domain.Processo) (domain.Processo, string, error) {
		return p.MarcarExecucao(emExecucao), "", nil
	})
}

// Avisar publica um texto sem débito e sem CPF.
func (s *Service) Avisar(ctx context.Context, quem auth.Identity, id uuid.UUID, fato, orientacao string) (string, error) {
	if err := s.daEquipe(quem); err != nil {
		return "", err
	}
	var texto string
	_, err := s.alterar(ctx, id, EventoAviso, func(p domain.Processo) (domain.Processo, string, error) {
		aviso, err := domain.AvisoExterno(p.Numero, fato, orientacao)
		if err != nil {
			return p, "", err
		}
		texto = aviso
		return p, aviso, nil
	})
	return texto, err
}
