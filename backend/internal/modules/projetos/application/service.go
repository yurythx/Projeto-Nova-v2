package application

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/outbox"
)

const (
	EventProjetoCreated      = "projetos.projeto.created"
	EventProjetoUpdated      = "projetos.projeto.updated"
	EventProjetoTransitioned = "projetos.projeto.transitioned"
)

type Service struct {
	pool      *pgxpool.Pool
	repo      domain.Repository
	outbox    *outbox.Writer
	logger    *slog.Logger
	passagens Passagens
	portas    domain.Portas
	produtos  domain.Produtos
	horas     domain.HorasVinculadas
	avisos    *notificacoes.Service
	arquivos  *Arquivos
}

// Passagens lê os caminhos acrescentados no quadro. Nil mantém só o fluxo de fábrica.
type Passagens interface {
	Mapa(ctx context.Context, db database.DBTX, quadro string) (map[string][]string, error)
}

func NewService(pool *pgxpool.Pool, repo domain.Repository, ob *outbox.Writer, logger *slog.Logger) *Service {
	return &Service{pool: pool, repo: repo, outbox: ob, logger: logger}
}

// ComPassagens liga os caminhos acrescentados nos quadros de projetos e de tarefas.
func (s *Service) ComPassagens(p Passagens) { s.passagens = p }

// ComHoras liga a soma das horas lançadas nos chamados vinculados.
func (s *Service) ComHoras(h domain.HorasVinculadas) { s.horas = h }

// ComPortas liga as portas de mudanças, problemas, ativos, solicitações e produtos.
func (s *Service) ComPortas(portas domain.Portas, produtos domain.Produtos) {
	s.portas = portas
	s.produtos = produtos
}

// ComAvisos liga a caixa de notificações. Nil não avisa.
func (s *Service) ComAvisos(n *notificacoes.Service) { s.avisos = n }

func (s *Service) mapa(ctx context.Context, db database.DBTX, quadro string) (map[string][]string, error) {
	if s.passagens == nil {
		return map[string][]string{}, nil
	}
	return s.passagens.Mapa(ctx, db, quadro)
}

func MapError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrProjetoNaoEncontrado):
		return apperrors.NotFound("projeto não encontrado")
	case errors.Is(err, domain.ErrTarefaNaoEncontrada):
		return apperrors.NotFound("tarefa não encontrada")
	case errors.Is(err, domain.ErrTituloObrigatorio):
		return apperrors.Validation("informe um título válido (até 160 caracteres)")
	case errors.Is(err, domain.ErrTransicaoInvalida):
		return apperrors.Conflict("passagem de estado não permitida para o estado atual")
	case errors.Is(err, domain.ErrEstadoInvalido):
		return apperrors.Validation("esse estado não está entre as colunas ativas da tarefa")
	case errors.Is(err, domain.ErrProjetoEncerrado):
		return apperrors.Conflict("projeto cancelado não aceita alterações")
	case errors.Is(err, domain.ErrProjetoConcluido):
		return apperrors.Conflict("projeto concluído só muda depois da reabertura")
	case errors.Is(err, domain.ErrTarefasAbertas):
		return apperrors.Conflict("conclua as tarefas em aberto antes de concluir o projeto")
	case errors.Is(err, domain.ErrPredecessoraInvalida):
		return apperrors.Validation("a tarefa anterior precisa ser outra tarefa deste projeto, sem ciclo")
	case errors.Is(err, domain.ErrPredecessoraAberta):
		return apperrors.Conflict("conclua a tarefa anterior antes de concluir esta")
	case errors.Is(err, domain.ErrResponsavelObrigatorio):
		return apperrors.Conflict("atribua um responsável antes de mover a tarefa")
	case errors.Is(err, domain.ErrRelatoInvalido):
		return apperrors.Validation("o relato da tarefa tem no máximo 4000 caracteres")
	case errors.Is(err, domain.ErrAnexoNaoEncontrado):
		return apperrors.NotFound("anexo não encontrado")
	case errors.Is(err, domain.ErrPrioridadeInvalida):
		return apperrors.Validation("prioridade inválida")
	case errors.Is(err, domain.ErrApontamentoInvalido):
		return apperrors.Validation("horas devem ser maiores que zero e até 24h, com descrição")
	case errors.Is(err, domain.ErrApontamentoNaoEncontrado):
		return apperrors.NotFound("apontamento não encontrado")
	case errors.Is(err, domain.ErrForaDoEscopo):
		return apperrors.Forbidden("fora do seu escopo de gestão")
	case errors.Is(err, domain.ErrVinculoInvalido):
		return apperrors.Validation("o vínculo precisa apontar para uma mudança, problema, ativo ou solicitação")
	case errors.Is(err, domain.ErrVinculoInexistente):
		return apperrors.Validation("item do vínculo ou produto não encontrado")
	case errors.Is(err, domain.ErrDeOutraUnidade):
		return apperrors.Validation("o vínculo ou produto precisa ser compatível com a unidade do projeto")
	case errors.Is(err, domain.ErrIntegracaoDesligada):
		return apperrors.Validation("o módulo deste vínculo está desligado")
	case errors.Is(err, domain.ErrPeriodoInvalido):
		return apperrors.Validation("a data de término precisa ser igual ou posterior à de início")
	case errors.Is(err, domain.ErrOrcamentoInvalido):
		return apperrors.Validation("o orçamento não pode ser negativo")
	case errors.Is(err, domain.ErrEtapaInvalida):
		return apperrors.Validation("cada etapa tem até 200 caracteres, no máximo 30")
	case errors.Is(err, domain.ErrCartaInvalida):
		return apperrors.Validation("texto longo demais na carta do projeto")
	case errors.Is(err, domain.ErrPessoaInvalida):
		return apperrors.Validation("escolha uma pessoa ativa lotada na unidade do projeto")
	case errors.Is(err, domain.ErrReaberturaExigeAprovacao):
		return apperrors.Conflict("projeto concluído não muda por esta passagem; a reabertura pede um superior ou administrador e o motivo")
	case errors.Is(err, domain.ErrReaberturaNaoAutorizada):
		return apperrors.Forbidden("só um superior da unidade ou um administrador reabre um projeto concluído")
	case errors.Is(err, domain.ErrMotivoObrigatorio):
		return apperrors.Validation("descreva o motivo da reabertura, em até 2000 caracteres")
	}
	return err
}

func (s *Service) conferirProduto(ctx context.Context, db database.DBTX, produto, unidade *uuid.UUID) error {
	if produto == nil || s.produtos == nil {
		return nil
	}
	dona, err := s.produtos.Consultar(ctx, db, *produto)
	if err != nil {
		return err
	}
	if !domain.MesmaUnidadeCompativel(unidade, dona) {
		return domain.ErrDeOutraUnidade
	}
	return nil
}

// fixarPessoa troca o nome livre por uma pessoa ativa. Sem id e sem nome,
// o campo fica vazio. O mesmo nome já gravado (sem id) é preservado.
// Um id é conferido de novo e o nome gravado é o da conta.
func (s *Service) fixarPessoa(ctx context.Context, db database.DBTX, unidade, id *uuid.UUID, nome, nomeGuardado string, idGuardado *uuid.UUID) (*uuid.UUID, string, error) {
	nome = strings.TrimSpace(nome)
	if id != nil && *id == uuid.Nil {
		id = nil
	}
	if id != nil {
		canon, err := s.repo.NomeDaPessoa(ctx, db, *id, unidade)
		if err != nil {
			return nil, "", err
		}
		return id, canon, nil
	}
	if nome == "" {
		return nil, "", nil
	}
	if nome == strings.TrimSpace(nomeGuardado) {
		return idGuardado, nomeGuardado, nil
	}
	return nil, "", domain.ErrPessoaInvalida
}

func alvoUnidade(u *uuid.UUID) auth.Target {
	if u == nil {
		return auth.Target{}
	}
	return auth.InUnidade(*u)
}

func (s *Service) Criar(ctx context.Context, idt auth.Identity, e domain.EntradaProjeto) (*domain.Projeto, error) {
	return s.criar(ctx, idt, e, domain.EntradaCarta{}, nil, false)
}

// CriarGuiado grava o projeto, a carta e as tarefas do Como na mesma transação.
func (s *Service) CriarGuiado(ctx context.Context, idt auth.Identity, e domain.EntradaProjeto, carta domain.EntradaCarta, etapas []string) (*domain.Projeto, error) {
	carta, etapas, err := domain.ValidarAbertura(e, carta, etapas)
	if err != nil {
		return nil, err
	}
	return s.criar(ctx, idt, e, carta, etapas, true)
}

func (s *Service) criar(ctx context.Context, idt auth.Identity, e domain.EntradaProjeto, carta domain.EntradaCarta, etapas []string, guiado bool) (*domain.Projeto, error) {
	if !auth.Can(idt, auth.PermProjetosManage, alvoUnidade(e.UnidadeID)) {
		return nil, domain.ErrForaDoEscopo
	}
	if err := e.Validar(); err != nil {
		return nil, err
	}

	p := &domain.Projeto{
		ID:                  uuid.New(),
		Titulo:              e.Titulo,
		Descricao:           e.Descricao,
		Objetivo:            e.Objetivo,
		Estado:              domain.EstadoPlanejado,
		Prioridade:          e.Prioridade,
		DataInicioPlanejada: e.DataInicioPlanejada,
		DataFimPlanejada:    e.DataFimPlanejada,
		DataInicioReal:      e.DataInicioReal,
		DataFimReal:         e.DataFimReal,
		Orcamento:           e.Orcamento,
		UnidadeID:           e.UnidadeID,
		ProdutoID:           e.ProdutoID,
		GerenteID:           e.GerenteID,
		GerenteNome:         e.GerenteNome,
	}

	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.conferirProduto(ctx, tx, p.ProdutoID, p.UnidadeID); err != nil {
			return err
		}
		gid, gnome, err := s.fixarPessoa(ctx, tx, e.UnidadeID, e.GerenteID, e.GerenteNome, "", nil)
		if err != nil {
			return err
		}
		p.GerenteID, p.GerenteNome = gid, gnome
		if err := s.repo.Criar(ctx, tx, p); err != nil {
			return err
		}
		if guiado {
			if _, err := s.repo.SalvarCarta(ctx, tx, p.ID, carta); err != nil {
				return err
			}
			for i, titulo := range etapas {
				t := &domain.Tarefa{
					ID: uuid.New(), ProjetoID: p.ID, Titulo: titulo,
					ResponsavelID: p.GerenteID, ResponsavelNome: p.GerenteNome,
					Estado: domain.EstadoTarefaAFazer, Ordem: i + 1, Origem: domain.OrigemHow,
				}
				if err := s.repo.CriarTarefa(ctx, tx, t); err != nil {
					return err
				}
			}
			if s.outbox != nil {
				payload := map[string]any{"id": p.ID, "etapas": len(etapas)}
				if err := s.outbox.Write(ctx, tx, "projetos.carta.created", "projeto", p.ID.String(), uuid.Nil, payload); err != nil {
					return err
				}
			}
			if err := audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.carta.created", "projeto", p.ID.String(), nil, carta)); err != nil {
				return err
			}
		}
		if s.outbox != nil {
			payload := map[string]any{"id": p.ID, "numero": p.Numero, "titulo": p.Titulo}
			if err := s.outbox.Write(ctx, tx, EventProjetoCreated, "projeto", p.ID.String(), uuid.Nil, payload); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, EventProjetoCreated, "projeto", p.ID.String(), nil, p))
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Service) montarVisao(ctx context.Context, db database.DBTX, p *domain.Projeto, carta *domain.Carta) (*domain.VisaoCarta, error) {
	total, concluidas, err := s.repo.ContarEtapasHow(ctx, db, p.ID)
	if err != nil {
		return nil, err
	}
	custo, err := s.repo.SomarCusto(ctx, db, p.ID)
	if err != nil {
		return nil, err
	}
	horasChamados, err := s.horasDosChamados(ctx, db, p.ID)
	if err != nil {
		return nil, err
	}
	v := &domain.VisaoCarta{
		ProjetoID:           p.ID,
		Titulo:              p.Titulo,
		Descricao:           p.Descricao,
		Objetivo:            p.Objetivo,
		DataInicioPlanejada: p.DataInicioPlanejada,
		DataFimPlanejada:    p.DataFimPlanejada,
		UnidadeID:           p.UnidadeID,
		GerenteID:           p.GerenteID,
		GerenteNome:         p.GerenteNome,
		Orcamento:           p.Orcamento,
		EtapasHow:           total,
		EtapasHowConcluidas: concluidas,
		CustoRealizado:      custo,
		HorasChamados:       horasChamados,
	}
	if carta != nil {
		v.Existe = true
		v.Onde = carta.Onde
		v.Patrocinador = carta.Patrocinador
		v.Metodologia = carta.Metodologia
		v.CentroCusto = carta.CentroCusto
		atualizado := carta.UpdatedAt
		v.UpdatedAt = &atualizado
	}
	return v, nil
}

func (s *Service) ObterCarta(ctx context.Context, idt auth.Identity, id uuid.UUID) (*domain.VisaoCarta, error) {
	p, err := s.repo.Obter(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if !auth.Can(idt, auth.PermProjetosRead, alvoUnidade(p.UnidadeID)) && !auth.Can(idt, auth.PermProjetosManage, alvoUnidade(p.UnidadeID)) {
		return nil, domain.ErrForaDoEscopo
	}
	carta, ok, err := s.repo.ObterCarta(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	var cp *domain.Carta
	if ok {
		cp = &carta
	}
	return s.montarVisao(ctx, s.pool, p, cp)
}

// SalvarCarta grava o plano de novo. A segunda chamada com o mesmo corpo
// não cria tarefa: as etapas do Como só nascem na abertura guiada.
func (s *Service) SalvarCarta(ctx context.Context, idt auth.Identity, id uuid.UUID, plano domain.Plano5W2H) (*domain.VisaoCarta, error) {
	plano, carta, err := plano.Normalizar()
	if err != nil {
		return nil, err
	}
	var visao *domain.VisaoCarta
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		p, err := s.repo.Obter(ctx, tx, id)
		if err != nil {
			return err
		}
		if !auth.Can(idt, auth.PermProjetosManage, alvoUnidade(p.UnidadeID)) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(p.Estado); err != nil {
			return err
		}
		gid, gnome, err := s.fixarPessoa(ctx, tx, p.UnidadeID, plano.GerenteID, plano.GerenteNome, p.GerenteNome, p.GerenteID)
		if err != nil {
			return err
		}
		plano.GerenteID, plano.GerenteNome = gid, gnome
		atualizado, err := s.repo.AtualizarPlano(ctx, tx, id, plano)
		if err != nil {
			return err
		}
		salva, err := s.repo.SalvarCarta(ctx, tx, id, carta)
		if err != nil {
			return err
		}
		if s.outbox != nil {
			payload := map[string]any{"id": id, "titulo": plano.Titulo}
			if err := s.outbox.Write(ctx, tx, "projetos.carta.saved", "projeto", id.String(), uuid.Nil, payload); err != nil {
				return err
			}
		}
		if err := audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.carta.saved", "projeto", id.String(), p, atualizado)); err != nil {
			return err
		}
		visao, err = s.montarVisao(ctx, tx, atualizado, &salva)
		return err
	})
	if err != nil {
		return nil, err
	}
	return visao, nil
}

func (s *Service) Obter(ctx context.Context, idt auth.Identity, id uuid.UUID) (*domain.Projeto, error) {
	p, err := s.repo.Obter(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if !auth.Can(idt, auth.PermProjetosRead, alvoUnidade(p.UnidadeID)) && !auth.Can(idt, auth.PermProjetosManage, alvoUnidade(p.UnidadeID)) {
		return nil, domain.ErrForaDoEscopo
	}
	p, err = s.comProximos(ctx, s.pool, p)
	if err != nil {
		return nil, err
	}
	return s.comReabertura(ctx, s.pool, idt, p)
}

func (s *Service) Listar(ctx context.Context, idt auth.Identity, estado, busca string) ([]domain.Projeto, error) {
	return s.ListarPorFiltro(ctx, idt, domain.FiltroProjetos{Estado: estado, Busca: busca})
}

func (s *Service) ListarPorFiltro(ctx context.Context, idt auth.Identity, f domain.FiltroProjetos) ([]domain.Projeto, error) {
	cob := auth.CoverageOf(idt, auth.PermProjetosRead)
	if !cob.All && len(cob.Unidades) == 0 {
		cob = auth.CoverageOf(idt, auth.PermProjetosManage)
	}
	f.Restrito = !cob.All
	f.Unidades = cob.Unidades
	lista, err := s.repo.Listar(ctx, s.pool, f)
	if err != nil {
		return nil, err
	}
	mapa, err := s.mapa(ctx, s.pool, "projetos")
	if err != nil {
		return nil, err
	}
	for i := range lista {
		lista[i].Proximos = domain.ProximosEstados(lista[i].Estado, mapa[lista[i].Estado]...)
	}
	return lista, nil
}

func (s *Service) comProximos(ctx context.Context, db database.DBTX, p *domain.Projeto) (*domain.Projeto, error) {
	if p == nil {
		return nil, nil
	}
	mapa, err := s.mapa(ctx, db, "projetos")
	if err != nil {
		return nil, err
	}
	p.Proximos = domain.ProximosEstados(p.Estado, mapa[p.Estado]...)
	return p, nil
}

func (s *Service) Atualizar(ctx context.Context, idt auth.Identity, id uuid.UUID, e domain.EntradaProjeto) (*domain.Projeto, error) {
	if err := e.Validar(); err != nil {
		return nil, err
	}

	var atualizado *domain.Projeto
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		p, err := s.repo.Obter(ctx, tx, id)
		if err != nil {
			return err
		}
		if !auth.Can(idt, auth.PermProjetosManage, alvoUnidade(p.UnidadeID)) || !auth.Can(idt, auth.PermProjetosManage, alvoUnidade(e.UnidadeID)) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(p.Estado); err != nil {
			return err
		}
		if err := s.conferirProduto(ctx, tx, e.ProdutoID, e.UnidadeID); err != nil {
			return err
		}
		gid, gnome, err := s.fixarPessoa(ctx, tx, e.UnidadeID, e.GerenteID, e.GerenteNome, p.GerenteNome, p.GerenteID)
		if err != nil {
			return err
		}
		e.GerenteID, e.GerenteNome = gid, gnome

		res, err := s.repo.Atualizar(ctx, tx, id, e)
		if err != nil {
			return err
		}
		atualizado = res

		if s.outbox != nil {
			payload := map[string]any{"id": id, "titulo": e.Titulo}
			if err := s.outbox.Write(ctx, tx, EventProjetoUpdated, "projeto", id.String(), uuid.Nil, payload); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, EventProjetoUpdated, "projeto", id.String(), p, atualizado))
	})
	if err != nil {
		return nil, err
	}
	return atualizado, nil
}

func (s *Service) Transicionar(ctx context.Context, idt auth.Identity, id uuid.UUID, novoEstado string) (*domain.Projeto, error) {
	var atualizado *domain.Projeto
	var enviadas []notificacoes.Enviada
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		p, err := s.repo.Obter(ctx, tx, id)
		if err != nil {
			return err
		}
		if !auth.Can(idt, auth.PermProjetosManage, alvoUnidade(p.UnidadeID)) {
			return domain.ErrForaDoEscopo
		}
		if p.Estado == domain.EstadoConcluido {
			return domain.ErrReaberturaExigeAprovacao
		}

		mapa, err := s.mapa(ctx, tx, "projetos")
		if err != nil {
			return err
		}
		if !domain.ValidarTransicao(p.Estado, novoEstado, mapa[p.Estado]...) {
			return domain.ErrTransicaoInvalida
		}
		if novoEstado == domain.EstadoConcluido {
			tarefas, err := s.repo.ListarTarefas(ctx, tx, id)
			if err != nil {
				return err
			}
			if domain.TarefasEmAberto(tarefas) > 0 {
				return domain.ErrTarefasAbertas
			}
		}

		res, err := s.repo.Transicionar(ctx, tx, id, novoEstado)
		if err != nil {
			return err
		}
		atualizado = res

		if s.outbox != nil {
			payload := map[string]any{"id": id, "estado_anterior": p.Estado, "novo_estado": novoEstado}
			if err := s.outbox.Write(ctx, tx, EventProjetoTransitioned, "projeto", id.String(), uuid.Nil, payload); err != nil {
				return err
			}
		}
		if novoEstado == domain.EstadoConcluido {
			var err error
			enviadas, err = s.avisarAprovadores(ctx, tx, res, idt.UserID)
			if err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, EventProjetoTransitioned, "projeto", id.String(), p, atualizado))
	})
	if err != nil {
		return nil, err
	}
	if s.avisos != nil {
		s.avisos.Entregar(ctx, enviadas)
	}
	return atualizado, nil
}

func (s *Service) avisarAprovadores(ctx context.Context, tx pgx.Tx, p *domain.Projeto, autor uuid.UUID) ([]notificacoes.Enviada, error) {
	if s.avisos == nil || p == nil {
		return nil, nil
	}
	ids, err := s.repo.Aprovadores(ctx, tx, p.UnidadeID)
	if err != nil {
		return nil, err
	}
	destinos := make([]uuid.UUID, 0, len(ids)+1)
	for _, id := range ids {
		if id != autor && id != uuid.Nil {
			destinos = append(destinos, id)
		}
	}
	if p.GerenteID != nil && *p.GerenteID != uuid.Nil && *p.GerenteID != autor {
		ja := false
		for _, id := range destinos {
			if id == *p.GerenteID {
				ja = true
				break
			}
		}
		if !ja {
			destinos = append(destinos, *p.GerenteID)
		}
	}
	titulo := p.Titulo
	if utf8.RuneCountInString(titulo) > 160 {
		titulo = string([]rune(titulo)[:160])
	}
	mensagem := titulo + ". A reabertura, se for necessária, pede o motivo na ficha."
	vinculos, err := s.repo.ListarVinculos(ctx, tx, p.ID)
	if err != nil {
		return nil, err
	}
	ligados := 0
	for _, v := range vinculos {
		if v.Tipo == domain.VinculoSolicitacao || v.Tipo == domain.VinculoProblema || v.Tipo == domain.VinculoMudanca {
			ligados++
		}
	}
	if ligados == 1 {
		mensagem += " 1 registro ligado continua no fluxo dele."
	} else if ligados > 1 {
		mensagem += " " + strconv.Itoa(ligados) + " registros ligados continuam no fluxo deles."
	}
	return notificacoes.Registrar(ctx, tx, destinos, notificacoes.Nova{
		Modulo:   "projetos",
		Titulo:   "Projeto #" + strconv.FormatInt(p.Numero, 10) + " concluído",
		Mensagem: mensagem,
		Link:     "/projetos/" + p.ID.String(),
		Chave:    "projetos:" + p.ID.String() + ":concluido:" + strconv.FormatInt(p.UpdatedAt.UnixNano(), 10),
	})
}

func (s *Service) comReabertura(ctx context.Context, db database.DBTX, idt auth.Identity, p *domain.Projeto) (*domain.Projeto, error) {
	if p == nil {
		return nil, nil
	}
	if p.Estado == domain.EstadoConcluido {
		ok, err := s.podeReabrir(ctx, db, idt, p.UnidadeID)
		if err != nil {
			return nil, err
		}
		p.PodeReabrir = ok
	}
	reab, achou, err := s.repo.UltimaReabertura(ctx, db, p.ID)
	if err != nil {
		return nil, err
	}
	if achou {
		p.Reabertura = &reab
	}
	return p, nil
}

func (s *Service) podeReabrir(ctx context.Context, db database.DBTX, idt auth.Identity, unidade *uuid.UUID) (bool, error) {
	if unidade == nil {
		return domain.PodeReabrir(idt, nil, nil, nil), nil
	}
	entidade, acima, err := s.repo.PosicaoUnidade(ctx, db, *unidade)
	if err != nil {
		return false, err
	}
	return domain.PodeReabrir(idt, unidade, entidade, acima), nil
}

func nomeDe(idt auth.Identity) string {
	switch {
	case idt.Name != "":
		return idt.Name
	case idt.Username != "":
		return idt.Username
	case idt.Email != "":
		return idt.Email
	default:
		return "Usuário"
	}
}

// Reabrir tira o projeto de concluído quando um superior ou administrador
// descreve o motivo. A passagem comum de estado não faz isso.
func (s *Service) Reabrir(ctx context.Context, idt auth.Identity, id uuid.UUID, motivo string) (*domain.Projeto, error) {
	motivo, err := domain.NormalizarMotivo(motivo)
	if err != nil {
		return nil, err
	}
	var atualizado *domain.Projeto
	var enviadas []notificacoes.Enviada
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		p, err := s.repo.Obter(ctx, tx, id)
		if err != nil {
			return err
		}
		if p.Estado != domain.EstadoConcluido {
			return domain.ErrReaberturaExigeAprovacao
		}
		ok, err := s.podeReabrir(ctx, tx, idt, p.UnidadeID)
		if err != nil {
			return err
		}
		if !ok {
			return domain.ErrReaberturaNaoAutorizada
		}
		res, err := s.repo.Transicionar(ctx, tx, id, domain.EstadoEmAndamento)
		if err != nil {
			return err
		}
		aprovador := idt.UserID
		if aprovador == uuid.Nil {
			if parsed, parseErr := uuid.Parse(idt.Subject); parseErr == nil {
				aprovador = parsed
			}
		}
		reab := &domain.Reabertura{
			ID: uuid.New(), ProjetoID: id, Motivo: motivo,
			AprovadorID: aprovador, AprovadorNome: nomeDe(idt),
		}
		if err := s.repo.SalvarReabertura(ctx, tx, reab); err != nil {
			return err
		}
		res.Reabertura = reab
		atualizado = res
		enviadas, err = s.avisarReabertura(ctx, tx, res, motivo, aprovador, reab.AprovadorNome)
		if err != nil {
			return err
		}
		if s.outbox != nil {
			payload := map[string]any{"id": id, "motivo": motivo, "aprovador_id": aprovador}
			if err := s.outbox.Write(ctx, tx, "projetos.projeto.reopened", "projeto", id.String(), uuid.Nil, payload); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.projeto.reopened", "projeto", id.String(), p, atualizado))
	})
	if err != nil {
		return nil, err
	}
	if s.avisos != nil {
		s.avisos.Entregar(ctx, enviadas)
	}
	return s.comProximos(ctx, s.pool, atualizado)
}

func (s *Service) avisarReabertura(ctx context.Context, tx pgx.Tx, p *domain.Projeto, motivo string, autor uuid.UUID, autorNome string) ([]notificacoes.Enviada, error) {
	if s.avisos == nil || p == nil || p.GerenteID == nil || *p.GerenteID == uuid.Nil || *p.GerenteID == autor {
		return nil, nil
	}
	quem := autorNome
	if quem == "" {
		quem = "um superior"
	}
	mensagem := cortar("Reaberto por "+quem+". "+motivo, 1000)
	return notificacoes.Registrar(ctx, tx, []uuid.UUID{*p.GerenteID}, notificacoes.Nova{
		Modulo:   "projetos",
		Titulo:   "Projeto #" + strconv.FormatInt(p.Numero, 10) + " reaberto",
		Mensagem: mensagem,
		Link:     "/projetos/" + p.ID.String(),
		Chave:    "projetos:" + p.ID.String() + ":reaberto:" + strconv.FormatInt(p.UpdatedAt.UnixNano(), 10),
	})
}

func cortar(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

func (s *Service) podeGerir(idt auth.Identity, unidade *uuid.UUID) bool {
	return auth.Can(idt, auth.PermProjetosManage, alvoUnidade(unidade))
}

func (s *Service) podeLer(idt auth.Identity, unidade *uuid.UUID) bool {
	alvo := alvoUnidade(unidade)
	return auth.Can(idt, auth.PermProjetosRead, alvo) || auth.Can(idt, auth.PermProjetosManage, alvo)
}

func (s *Service) horasDosChamados(ctx context.Context, db database.DBTX, projetoID uuid.UUID) (float64, error) {
	if s.horas == nil {
		return 0, nil
	}
	vinculos, err := s.repo.ListarVinculos(ctx, db, projetoID)
	if err != nil {
		return 0, err
	}
	ids := make([]uuid.UUID, 0, len(vinculos))
	for _, v := range vinculos {
		if v.Tipo == domain.VinculoSolicitacao {
			ids = append(ids, v.AlvoID)
		}
	}
	if len(ids) == 0 {
		return 0, nil
	}
	return s.horas.Somar(ctx, db, ids)
}

func normalizarPredecessora(id *uuid.UUID) *uuid.UUID {
	if id != nil && *id == uuid.Nil {
		return nil
	}
	return id
}

func estadoEfetivo(estado string) string {
	if estado == "" {
		return domain.EstadoTarefaAFazer
	}
	return estado
}

func (s *Service) conferirPredecessora(ctx context.Context, db database.DBTX, projetoID, tarefaID uuid.UUID, pred *uuid.UUID, estado string) error {
	pred = normalizarPredecessora(pred)
	tarefas, err := s.repo.ListarTarefas(ctx, db, projetoID)
	if err != nil {
		return err
	}
	if pred != nil {
		if err := domain.ValidarCadeia(tarefaID, *pred, tarefas); err != nil {
			return err
		}
	}
	var anterior *domain.Tarefa
	if pred != nil {
		for i := range tarefas {
			if tarefas[i].ID == *pred {
				anterior = &tarefas[i]
				break
			}
		}
	}
	return domain.ConferirPredecessora(estadoEfetivo(estado), anterior)
}

func (s *Service) CriarTarefa(ctx context.Context, idt auth.Identity, projetoID uuid.UUID, titulo, descricao string, responsavelID *uuid.UUID, responsavel, estado string, ordem int, dataLimite *time.Time, predecessoraID *uuid.UUID) (*domain.Tarefa, error) {
	var criada *domain.Tarefa
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		projeto, err := s.repo.Obter(ctx, tx, projetoID)
		if err != nil {
			return err
		}
		if !s.podeGerir(idt, projeto.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(projeto.Estado); err != nil {
			return err
		}
		mapa, err := s.mapa(ctx, tx, "tarefas")
		if err != nil {
			return err
		}
		if !domain.EstadoDeTarefaAceito("", estado, mapa) {
			return domain.ErrEstadoInvalido
		}
		predecessoraID = normalizarPredecessora(predecessoraID)
		t := &domain.Tarefa{
			ID:             uuid.New(),
			ProjetoID:      projetoID,
			PredecessoraID: predecessoraID,
			Estado:         estadoEfetivo(estado),
		}
		if err := s.conferirPredecessora(ctx, tx, projetoID, t.ID, predecessoraID, t.Estado); err != nil {
			return err
		}
		gid, gnome, err := s.fixarPessoa(ctx, tx, projeto.UnidadeID, responsavelID, responsavel, "", nil)
		if err != nil {
			return err
		}
		if err := domain.MovimentoExigeResponsavel("", t.Estado, gid); err != nil {
			return err
		}
		t.Titulo = titulo
		t.Descricao = descricao
		t.ResponsavelID = gid
		t.ResponsavelNome = gnome
		t.Ordem = ordem
		t.DataLimite = dataLimite
		if err := s.repo.CriarTarefa(ctx, tx, t); err != nil {
			return err
		}
		criada = t
		return s.anotarMovimento(ctx, tx, idt, domain.Tarefa{}, *t)
	})
	if err != nil {
		return nil, err
	}
	return criada, nil
}

func (s *Service) AtualizarTarefa(ctx context.Context, idt auth.Identity, id uuid.UUID, titulo, descricao, relato string, responsavelID *uuid.UUID, responsavel, estado string, ordem int, dataLimite *time.Time, predecessoraID *uuid.UUID) (*domain.Tarefa, error) {
	if utf8.RuneCountInString(relato) > domain.RelatoMax {
		return nil, domain.ErrRelatoInvalido
	}
	var atualizada *domain.Tarefa
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		atual, err := s.repo.ObterTarefa(ctx, tx, id)
		if err != nil {
			return err
		}
		projeto, err := s.repo.Obter(ctx, tx, atual.ProjetoID)
		if err != nil {
			return err
		}
		if !s.podeGerir(idt, projeto.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(projeto.Estado); err != nil {
			return err
		}
		mapa, err := s.mapa(ctx, tx, "tarefas")
		if err != nil {
			return err
		}
		if !domain.EstadoDeTarefaAceito(atual.Estado, estado, mapa) {
			return domain.ErrEstadoInvalido
		}
		predecessoraID = normalizarPredecessora(predecessoraID)
		if err := s.conferirPredecessora(ctx, tx, atual.ProjetoID, id, predecessoraID, estadoEfetivo(estado)); err != nil {
			return err
		}
		gid, gnome, err := s.fixarPessoa(ctx, tx, projeto.UnidadeID, responsavelID, responsavel, atual.ResponsavelNome, atual.ResponsavelID)
		if err != nil {
			return err
		}
		if err := domain.MovimentoExigeResponsavel(atual.Estado, estadoEfetivo(estado), gid); err != nil {
			return err
		}
		atualizada, err = s.repo.AtualizarTarefa(ctx, tx, id, titulo, descricao, strings.TrimSpace(relato), gid, gnome, estadoEfetivo(estado), ordem, dataLimite, predecessoraID)
		if err != nil {
			return err
		}
		return s.anotarMovimento(ctx, tx, idt, *atual, *atualizada)
	})
	if err != nil {
		return nil, err
	}
	return atualizada, nil
}

func (s *Service) ListarTarefas(ctx context.Context, idt auth.Identity, projetoID uuid.UUID) ([]domain.Tarefa, error) {
	projeto, err := s.repo.Obter(ctx, s.pool, projetoID)
	if err != nil {
		return nil, err
	}
	if !s.podeLer(idt, projeto.UnidadeID) {
		return nil, domain.ErrForaDoEscopo
	}
	return s.repo.ListarTarefas(ctx, s.pool, projetoID)
}

func (s *Service) ExcluirTarefa(ctx context.Context, idt auth.Identity, id uuid.UUID) error {
	return database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		atual, err := s.repo.ObterTarefa(ctx, tx, id)
		if err != nil {
			return err
		}
		projeto, err := s.repo.Obter(ctx, tx, atual.ProjetoID)
		if err != nil {
			return err
		}
		if !s.podeGerir(idt, projeto.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(projeto.Estado); err != nil {
			return err
		}
		if err := s.repo.ExcluirTarefa(ctx, tx, id); err != nil {
			return err
		}
		if s.outbox != nil {
			if err := s.outbox.Write(ctx, tx, "projetos.tarefa.deleted", "projeto_tarefa", id.String(), uuid.Nil, map[string]any{"id": id}); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.tarefa.deleted", "projeto_tarefa", id.String(), nil, nil))
	})
}

func (s *Service) ListarVinculos(ctx context.Context, idt auth.Identity, projetoID uuid.UUID) ([]domain.Vinculo, error) {
	projeto, err := s.repo.Obter(ctx, s.pool, projetoID)
	if err != nil {
		return nil, err
	}
	if !s.podeLer(idt, projeto.UnidadeID) {
		return nil, domain.ErrForaDoEscopo
	}
	return s.repo.ListarVinculos(ctx, s.pool, projetoID)
}

func (s *Service) Vincular(ctx context.Context, idt auth.Identity, projetoID uuid.UUID, tipo string, alvoID uuid.UUID, rotulo string) (*domain.Vinculo, error) {
	if err := domain.NovoVinculo(tipo, alvoID); err != nil {
		return nil, err
	}
	v := &domain.Vinculo{
		ID:           uuid.New(),
		ProjetoID:    projetoID,
		Tipo:         tipo,
		AlvoID:       alvoID,
		Rotulo:       strings.TrimSpace(rotulo),
		VinculadoPor: idt.Email,
	}

	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		p, err := s.repo.Obter(ctx, tx, projetoID)
		if err != nil {
			return err
		}
		if !auth.Can(idt, auth.PermProjetosManage, alvoUnidade(p.UnidadeID)) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(p.Estado); err != nil {
			return err
		}
		if s.portas != nil {
			if !s.portas.Disponivel(tipo) {
				return domain.ErrIntegracaoDesligada
			}
			item, err := s.portas.Consultar(ctx, tx, tipo, alvoID)
			if err != nil {
				return err
			}
			if !domain.MesmaUnidadeCompativel(p.UnidadeID, item.UnidadeID) {
				return domain.ErrDeOutraUnidade
			}
			if v.Rotulo == "" {
				v.Rotulo = item.Rotulo
			}
		}
		if err := s.repo.CriarVinculo(ctx, tx, v); err != nil {
			return err
		}
		if s.outbox != nil {
			payload := map[string]any{"projeto_id": projetoID, "tipo": tipo, "alvo_id": alvoID, "rotulo": v.Rotulo}
			if err := s.outbox.Write(ctx, tx, "projetos.vinculo.created", "projeto", projetoID.String(), uuid.Nil, payload); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.vinculo.created", "projeto", projetoID.String(), nil, v))
	})
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (s *Service) Desvincular(ctx context.Context, idt auth.Identity, vinculoID uuid.UUID) error {
	return database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		projetoID, ok, err := s.repo.ProjetoDoVinculo(ctx, tx, vinculoID)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		p, err := s.repo.Obter(ctx, tx, projetoID)
		if err != nil {
			return err
		}
		if !s.podeGerir(idt, p.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(p.Estado); err != nil {
			return err
		}
		if err := s.repo.ExcluirVinculo(ctx, tx, vinculoID); err != nil {
			return err
		}
		if s.outbox != nil {
			if err := s.outbox.Write(ctx, tx, "projetos.vinculo.deleted", "projeto_vinculo", vinculoID.String(), uuid.Nil, map[string]any{"id": vinculoID}); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.vinculo.deleted", "projeto_vinculo", vinculoID.String(), nil, nil))
	})
}

func (s *Service) RegistrarApontamento(
	ctx context.Context,
	idt auth.Identity,
	projetoID uuid.UUID,
	tarefaID *uuid.UUID,
	dataApontamento string,
	horas float64,
	descricao string,
	custoHora float64,
) (*domain.Apontamento, error) {
	nomeUsuario := idt.Name
	if nomeUsuario == "" {
		nomeUsuario = idt.Username
	}
	if nomeUsuario == "" {
		nomeUsuario = idt.Email
	}
	if nomeUsuario == "" {
		nomeUsuario = "Usuário"
	}

	usuarioID := idt.UserID
	if usuarioID == uuid.Nil {
		if parsed, err := uuid.Parse(idt.Subject); err == nil {
			usuarioID = parsed
		} else {
			usuarioID = uuid.New()
		}
	}

	a := &domain.Apontamento{
		ID:              uuid.New(),
		ProjetoID:       projetoID,
		TarefaID:        tarefaID,
		UsuarioID:       usuarioID,
		UsuarioNome:     nomeUsuario,
		DataApontamento: dataApontamento,
		Horas:           horas,
		Descricao:       descricao,
		CustoHora:       custoHora,
		CustoTotal:      horas * custoHora,
	}
	if err := a.Validar(); err != nil {
		return nil, MapError(err)
	}

	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		p, err := s.repo.Obter(ctx, tx, projetoID)
		if err != nil {
			return err
		}
		if !s.podeGerir(idt, p.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(p.Estado); err != nil {
			return err
		}
		if err := s.repo.CriarApontamento(ctx, tx, a); err != nil {
			return err
		}
		if s.outbox != nil {
			payload := map[string]any{
				"projeto_id":  projetoID,
				"tarefa_id":   tarefaID,
				"usuario_id":  usuarioID,
				"horas":       horas,
				"custo_total": a.CustoTotal,
			}
			if err := s.outbox.Write(ctx, tx, "projetos.apontamento.created", "projeto", projetoID.String(), uuid.Nil, payload); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.apontamento.created", "projeto", projetoID.String(), nil, a))
	})
	if err != nil {
		return nil, MapError(err)
	}
	return a, nil
}

func (s *Service) ListarApontamentos(
	ctx context.Context,
	idt auth.Identity,
	projetoID uuid.UUID,
) ([]domain.Apontamento, domain.ResumoHorasProjeto, error) {
	projeto, err := s.repo.Obter(ctx, s.pool, projetoID)
	if err != nil {
		return nil, domain.ResumoHorasProjeto{}, MapError(err)
	}
	if !s.podeLer(idt, projeto.UnidadeID) {
		return nil, domain.ResumoHorasProjeto{}, MapError(domain.ErrForaDoEscopo)
	}
	lista, err := s.repo.ListarApontamentos(ctx, s.pool, projetoID)
	if err != nil {
		return nil, domain.ResumoHorasProjeto{}, MapError(err)
	}

	resumo := domain.ResumoHorasProjeto{
		PorUsuario: make(map[string]float64),
	}
	for _, a := range lista {
		resumo.TotalHoras += a.Horas
		resumo.TotalCusto += a.CustoTotal
		resumo.PorUsuario[a.UsuarioNome] += a.Horas
	}

	return lista, resumo, nil
}

func (s *Service) ExcluirApontamento(ctx context.Context, idt auth.Identity, apontamentoID uuid.UUID) error {
	return database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		projetoID, ok, err := s.repo.ProjetoDoApontamento(ctx, tx, apontamentoID)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		p, err := s.repo.Obter(ctx, tx, projetoID)
		if err != nil {
			return err
		}
		if !s.podeGerir(idt, p.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(p.Estado); err != nil {
			return err
		}
		if err := s.repo.ExcluirApontamento(ctx, tx, apontamentoID); err != nil {
			return err
		}
		if s.outbox != nil {
			if err := s.outbox.Write(ctx, tx, "projetos.apontamento.deleted", "projeto_apontamento", apontamentoID.String(), uuid.Nil, map[string]any{"id": apontamentoID}); err != nil {
				return err
			}
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.apontamento.deleted", "projeto_apontamento", apontamentoID.String(), nil, nil))
	})
}
