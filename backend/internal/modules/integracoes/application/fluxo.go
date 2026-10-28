package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// preparo é a área de preparo de uma conexão: Destino da extração e
// Leitor da carga.
type preparo struct {
	s       *Service
	conexao uuid.UUID
}

func (p preparo) Versoes(ctx context.Context) (map[int]string, error) {
	return p.s.repo.Versoes(ctx, p.s.pool, p.conexao)
}

func (p preparo) Pessoas(ctx context.Context) (map[int]bool, error) {
	return p.s.repo.IDs(ctx, p.s.pool, p.conexao, domain.ItemPessoa)
}

func (p preparo) Gravar(ctx context.Context, itens []domain.Item) error {
	return database.WithTx(ctx, p.s.pool, func(ctx context.Context, tx pgx.Tx) error {
		return p.s.repo.GravarItens(ctx, tx, p.conexao, itens)
	})
}

func (p preparo) Itens(ctx context.Context, tipo string, de, limite int) ([]domain.Item, int, error) {
	return p.s.repo.Itens(ctx, p.s.pool, p.conexao, tipo, de, limite)
}

func (p preparo) Item(ctx context.Context, tipo string, id int) (domain.Item, bool, error) {
	return p.s.repo.Item(ctx, p.s.pool, p.conexao, tipo, id)
}

// ConectorVersoes diz ao conector o que já está na área de preparo: os
// chamados (com a versão) e as pessoas.
func (s *Service) ConectorVersoes(ctx context.Context, identity auth.Identity) (map[int]string, map[int]bool, error) {
	id, err := conexaoDoConector(identity)
	if err != nil {
		return nil, nil, MapError(err)
	}
	p := preparo{s, id}
	v, err := p.Versoes(ctx)
	if err != nil {
		return nil, nil, MapError(err)
	}
	pessoas, err := p.Pessoas(ctx)
	return v, pessoas, MapError(err)
}

// ConectorGravar recebe um lote do conector.
func (s *Service) ConectorGravar(ctx context.Context, identity auth.Identity, itens []domain.Item) error {
	id, err := conexaoDoConector(identity)
	if err != nil {
		return MapError(err)
	}
	if err := domain.ValidarItens(itens); err != nil {
		return MapError(err)
	}
	return MapError(preparo{s, id}.Gravar(ctx, itens))
}

// ConectorPegar entrega ao conector a extração pedida na tela, se houver.
func (s *Service) ConectorPegar(ctx context.Context, identity auth.Identity) (domain.Tarefa, bool, error) {
	id, err := conexaoDoConector(identity)
	if err != nil {
		return domain.Tarefa{}, false, MapError(err)
	}
	t, ok, err := s.repo.Pegar(ctx, s.pool, &id, []string{domain.TarefaExtracao}, domain.ModoConector)
	return t, ok, MapError(err)
}

func (s *Service) tarefaDoConector(ctx context.Context, identity auth.Identity, tarefa uuid.UUID) (domain.Tarefa, error) {
	id, err := conexaoDoConector(identity)
	if err != nil {
		return domain.Tarefa{}, err
	}
	t, err := s.repo.GetTarefa(ctx, s.pool, tarefa)
	if err == nil && (t.ConexaoID != id || t.Tipo != domain.TarefaExtracao) {
		err = domain.ErrNaoEncontrado
	}
	return t, err
}

// ConectorProgresso registra o andamento e devolve o estado: cancelada
// diz ao conector para parar.
func (s *Service) ConectorProgresso(ctx context.Context, identity auth.Identity, tarefa uuid.UUID, feitos, total int) (string, error) {
	if _, err := s.tarefaDoConector(ctx, identity, tarefa); err != nil {
		return "", MapError(err)
	}
	estado, err := s.repo.Progresso(ctx, s.pool, tarefa, feitos, total)
	return estado, MapError(err)
}

// ConectorTerminar fecha a extração com o resumo (ou o erro).
func (s *Service) ConectorTerminar(ctx context.Context, identity auth.Identity, tarefa uuid.UUID, relatorio json.RawMessage, falha string) error {
	t, err := s.tarefaDoConector(ctx, identity, tarefa)
	if err != nil {
		return MapError(err)
	}
	return MapError(s.terminar(ctx, t, relatorio, falha))
}

// terminar grava o fim da tarefa e, na extração que deu certo, a data.
func (s *Service) terminar(ctx context.Context, t domain.Tarefa, relatorio []byte, falha string) error {
	estado := domain.EstadoConcluida
	if falha != "" {
		estado = domain.EstadoFalhou
	}
	if len(relatorio) == 0 || !json.Valid(relatorio) {
		relatorio = nil
	}
	return database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.Terminar(ctx, tx, t.ID, estado, relatorio, truncar(falha, 2000)); err != nil {
			return err
		}
		if estado == domain.EstadoConcluida && t.Tipo == domain.TarefaExtracao {
			if err := s.repo.MarcarExtracao(ctx, tx, t.ConexaoID, s.agora().UTC()); err != nil {
				return err
			}
		}
		return s.registrar(ctx, tx, "integracoes.tarefa."+estado, "integracoes_tarefa", t.ID,
			map[string]any{"tipo": t.Tipo, "conexao_id": t.ConexaoID.String(), "erro": falha != ""})
	})
}

func truncar(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// Inventario é o retrato da área de preparo.
func (s *Service) Inventario(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Inventario, error) {
	c, err := s.ObterConexao(ctx, identity, id)
	if err != nil {
		return domain.Inventario{}, err
	}
	inv, err := s.inventario(ctx, id)
	inv.ExtraidoEm = c.ExtraidoEm
	return inv, MapError(err)
}

func (s *Service) inventario(ctx context.Context, id uuid.UUID) (domain.Inventario, error) {
	itens, err := s.repo.Contagens(ctx, s.pool, id)
	if err != nil {
		return domain.Inventario{}, err
	}
	grupos, err := s.repo.Grupos(ctx, s.pool, id)
	if err != nil {
		return domain.Inventario{}, err
	}
	entidades, err := s.repo.Nomes(ctx, s.pool, id, domain.ItemEntidade)
	if err != nil {
		return domain.Inventario{}, err
	}
	categorias, err := s.repo.Nomes(ctx, s.pool, id, domain.ItemCategoria)
	if err != nil {
		return domain.Inventario{}, err
	}
	slas, err := s.repo.SLAs(ctx, s.pool, id)
	if err != nil {
		return domain.Inventario{}, err
	}
	return domain.MontarInventario(grupos, entidades, categorias, slas, itens), nil
}

// Tabela devolve a tabela de-para da conexão.
func (s *Service) Tabela(ctx context.Context, identity auth.Identity, id uuid.UUID) ([]domain.Linha, error) {
	if _, err := s.ObterConexao(ctx, identity, id); err != nil {
		return nil, err
	}
	linhas, err := s.repo.ListarTabela(ctx, s.pool, id)
	return linhas, MapError(err)
}

// SugerirTabela troca a tabela pela sugerida a partir do inventário.
// unidades leva cada secretaria a uma unidade daqui.
func (s *Service) SugerirTabela(ctx context.Context, identity auth.Identity, id uuid.UUID, unidades map[string]uuid.UUID) ([]domain.Linha, error) {
	if _, err := s.ObterConexao(ctx, identity, id); err != nil {
		return nil, err
	}
	inv, err := s.inventario(ctx, id)
	if err != nil {
		return nil, MapError(err)
	}
	if len(inv.Grupos) == 0 {
		return nil, MapError(domain.Recusa{Msg: "extraia os dados da origem antes de montar a tabela"})
	}
	return s.SalvarTabela(ctx, identity, id, domain.Sugerir(inv, unidades))
}

// SalvarTabela troca a tabela inteira pela revisada.
func (s *Service) SalvarTabela(ctx context.Context, identity auth.Identity, id uuid.UUID, linhas []domain.Linha) ([]domain.Linha, error) {
	if _, err := s.ObterConexao(ctx, identity, id); err != nil {
		return nil, err
	}
	if err := domain.ValidarTabela(linhas); err != nil {
		return nil, MapError(err)
	}
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.TrocarTabela(ctx, tx, id, linhas); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "integracoes.tabela.salva", "integracoes_conexao", id, map[string]any{"linhas": len(linhas)})
	})
	if err != nil {
		return nil, MapError(err)
	}
	return s.Tabela(ctx, identity, id)
}

// PedirTarefa enfileira a extração, a simulação, a carga ou o desfazer.
// O que falta para ela rodar é recusado agora, e não na fila. O desfazer
// que remove de fato pede confirmacao igual ao nome da conexão.
func (s *Service) PedirTarefa(ctx context.Context, identity auth.Identity, id uuid.UUID, tipo, confirmacao string) (domain.Tarefa, error) {
	c, err := s.ObterConexao(ctx, identity, id)
	if err != nil {
		return domain.Tarefa{}, err
	}
	t, err := domain.NovaTarefa(id, tipo, identity.Username)
	if err != nil {
		return domain.Tarefa{}, MapError(err)
	}
	if tipo == domain.TarefaDesfazer && strings.TrimSpace(confirmacao) != c.Nome {
		return domain.Tarefa{}, MapError(domain.Recusa{Msg: "para desfazer a carga, digite o nome da conexão"})
	}
	if err := s.conferir(ctx, c, tipo); err != nil {
		return domain.Tarefa{}, MapError(err)
	}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.InserirTarefa(ctx, tx, t); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "integracoes.tarefa.pedida", "integracoes_tarefa", t.ID, map[string]any{"tipo": tipo, "conexao_id": id.String()})
	})
	return t, MapError(err)
}

func (s *Service) conferir(ctx context.Context, c domain.Conexao, tipo string) error {
	if tipo == domain.TarefaExtracao {
		switch {
		case c.Modo == domain.ModoConector && !c.TemToken:
			return domain.Recusa{Msg: "emita o token e instale o conector antes de extrair"}
		case c.Modo == domain.ModoDireto && (!c.TemCredenciais || s.extrator == nil):
			return domain.Recusa{Msg: "defina as credenciais da origem antes de extrair"}
		}
		return nil
	}
	if s.carregador == nil {
		return domain.Recusa{Msg: "a carga está indisponível: o atendimento precisa estar ligado"}
	}
	if tipo == domain.TarefaDesfazerSimulacao || tipo == domain.TarefaDesfazer {
		return nil
	}
	n, err := s.repo.Contagens(ctx, s.pool, c.ID)
	if err != nil {
		return err
	}
	if n[domain.ItemChamado] == 0 {
		return domain.Recusa{Msg: "extraia os dados da origem antes"}
	}
	tabela, err := s.repo.ListarTabela(ctx, s.pool, c.ID)
	if err != nil {
		return err
	}
	if len(tabela) == 0 {
		return domain.Recusa{Msg: "monte a tabela de-para antes"}
	}
	if tipo == domain.TarefaCarga {
		for _, l := range tabela {
			if l.UnidadeID == nil {
				return domain.Recusa{Msg: fmt.Sprintf("escolha a unidade de %s, %s, antes de carregar", l.Secretaria, l.CategoriaNome)}
			}
		}
	}
	return nil
}

// ListarTarefas devolve as últimas tarefas da conexão.
func (s *Service) ListarTarefas(ctx context.Context, identity auth.Identity, id uuid.UUID) ([]domain.Tarefa, error) {
	if _, err := s.ObterConexao(ctx, identity, id); err != nil {
		return nil, err
	}
	lista, err := s.repo.ListarTarefas(ctx, s.pool, id, 30)
	return lista, MapError(err)
}

// CancelarTarefa para a tarefa. Quem executa percebe no próximo progresso.
func (s *Service) CancelarTarefa(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	if err := soGlobal(identity); err != nil {
		return MapError(err)
	}
	return MapError(database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.Cancelar(ctx, tx, id); err != nil {
			return err
		}
		return s.registrar(ctx, tx, "integracoes.tarefa.cancelada", "integracoes_tarefa", id, map[string]any{})
	}))
}

// RunTarefas é o worker: devolve a pendente a tarefa parada e executa,
// uma de cada vez, as extrações diretas, as simulações e as cargas.
func (s *Service) RunTarefas(ctx context.Context) error {
	t := time.NewTicker(s.Intervalo)
	defer t.Stop()
	for {
		if err := s.Passo(ctx); err != nil && ctx.Err() == nil {
			s.logger.Warn("integracoes: tarefas", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// Passo retoma as paradas e executa o que estiver na fila.
func (s *Service) Passo(ctx context.Context) error {
	if n, err := s.repo.Retomar(ctx, s.pool, s.agora().Add(-s.Parada)); err != nil {
		return err
	} else if n > 0 {
		s.logger.Info("integracoes: tarefas retomadas", slog.Int("tarefas", n))
	}
	for ctx.Err() == nil {
		t, ok, err := s.repo.Pegar(ctx, s.pool, nil, []string{domain.TarefaExtracao}, domain.ModoDireto)
		if err == nil && !ok {
			t, ok, err = s.repo.Pegar(ctx, s.pool, nil, []string{domain.TarefaSimulacao, domain.TarefaCarga,
				domain.TarefaDesfazerSimulacao, domain.TarefaDesfazer}, "")
		}
		if err != nil || !ok {
			return err
		}
		s.executar(ctx, t)
	}
	return nil
}

// executar roda a tarefa. Cancelada na tela, ela para no próximo
// progresso; o serviço desligando a deixa em andamento, e ela volta a
// pendente quando parar de dar sinal de vida.
func (s *Service) executar(ctx context.Context, t domain.Tarefa) {
	trabalho, parar := context.WithCancel(ctx)
	defer parar()
	cancelada := false
	progresso := func(feitos, total int) {
		estado, err := s.repo.Progresso(ctx, s.pool, t.ID, feitos, total)
		if err == nil && estado == domain.EstadoCancelada {
			cancelada = true
			parar()
		}
	}
	rel, err := s.rodar(trabalho, t, progresso)
	if ctx.Err() != nil || cancelada {
		return
	}
	var b []byte
	falha := ""
	if err != nil {
		falha = err.Error()
	} else if b, err = json.Marshal(rel); err != nil {
		falha = err.Error()
	}
	if err := s.terminar(context.WithoutCancel(ctx), t, b, falha); err != nil {
		s.logger.Error("integracoes: fim da tarefa", slog.String("tarefa", t.ID.String()), slog.Any("error", err))
	}
}

func (s *Service) rodar(ctx context.Context, t domain.Tarefa, progresso Progresso) (any, error) {
	c, err := s.repo.GetConexao(ctx, s.pool, t.ConexaoID)
	if err != nil {
		return nil, err
	}
	p := preparo{s, c.ID}
	if t.Tipo == domain.TarefaExtracao {
		cred, err := s.credenciais(c)
		if err != nil {
			return nil, err
		}
		if s.extrator == nil {
			return nil, errors.New("extração direta indisponível")
		}
		return s.extrator.Extrair(ctx, c, cred, p, progresso)
	}
	if s.carregador == nil {
		return nil, errors.New("carga indisponível")
	}
	if t.Tipo == domain.TarefaDesfazerSimulacao || t.Tipo == domain.TarefaDesfazer {
		return s.carregador.Desfazer(ctx, t.Tipo == domain.TarefaDesfazer, progresso)
	}
	tabela, err := s.repo.ListarTabela(ctx, s.pool, c.ID)
	if err != nil {
		return nil, err
	}
	return s.carregador.Carregar(ctx, c, tabela, t.Tipo == domain.TarefaCarga, p, progresso)
}
