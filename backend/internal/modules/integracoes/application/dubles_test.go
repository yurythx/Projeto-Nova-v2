package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
	"github.com/yurythx/projeto-nova-v2/internal/platform/secretcrypto"
)

var errFalha = errors.New("falha injetada")

func admin() auth.Identity {
	return auth.Identity{Roles: []string{auth.RoleAdmin}, UserID: uuid.New(), Username: "admin"}
}

// eventos é o outbox do teste; err faz toda gravação falhar.
type eventos struct{ err error }

func (e eventos) Write(context.Context, pgx.Tx, string, string, string, uuid.UUID, any) error {
	return e.err
}

func cifra(t *testing.T) *secretcrypto.Cipher {
	t.Helper()
	c, err := secretcrypto.NewFromBase64Key(base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// falhaRepo delega ao repositório real e falha só nos métodos marcados.
type falhaRepo struct {
	domain.Repository
	falha map[string]bool
	// alvo é a conexão que ParaSincronizar:todas devolve.
	alvo uuid.UUID
}

func falhando(metodos ...string) falhaRepo {
	f := falhaRepo{Repository: infrastructure.NewRepository(), falha: map[string]bool{}}
	for _, m := range metodos {
		f.falha[m] = true
	}
	return f
}

func (f falhaRepo) ListConexoes(ctx context.Context, db database.DBTX) ([]domain.Conexao, error) {
	if f.falha["ListConexoes"] {
		return nil, errFalha
	}
	return f.Repository.ListConexoes(ctx, db)
}
func (f falhaRepo) GetConexao(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Conexao, error) {
	if f.falha["GetConexao"] {
		return domain.Conexao{}, errFalha
	}
	return f.Repository.GetConexao(ctx, db, id)
}
func (f falhaRepo) InsertConexao(ctx context.Context, db database.DBTX, c domain.Conexao) (domain.Conexao, error) {
	if f.falha["InsertConexao"] {
		return domain.Conexao{}, errFalha
	}
	return f.Repository.InsertConexao(ctx, db, c)
}
func (f falhaRepo) UpdateConexao(ctx context.Context, db database.DBTX, c domain.Conexao) error {
	if f.falha["UpdateConexao"] {
		return errFalha
	}
	return f.Repository.UpdateConexao(ctx, db, c)
}
func (f falhaRepo) DeleteConexao(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if f.falha["DeleteConexao"] {
		return errFalha
	}
	return f.Repository.DeleteConexao(ctx, db, id)
}
func (f falhaRepo) SetCredenciais(ctx context.Context, db database.DBTX, id uuid.UUID, c string) error {
	if f.falha["SetCredenciais"] {
		return errFalha
	}
	return f.Repository.SetCredenciais(ctx, db, id, c)
}
func (f falhaRepo) SetToken(ctx context.Context, db database.DBTX, id uuid.UUID, h *string) error {
	if f.falha["SetToken"] {
		return errFalha
	}
	return f.Repository.SetToken(ctx, db, id, h)
}
func (f falhaRepo) PorToken(ctx context.Context, db database.DBTX, h string) (domain.Conexao, error) {
	if f.falha["PorToken"] {
		return domain.Conexao{}, errFalha
	}
	return f.Repository.PorToken(ctx, db, h)
}
func (f falhaRepo) MarcarExtracao(ctx context.Context, db database.DBTX, id uuid.UUID, q time.Time) error {
	if f.falha["MarcarExtracao"] {
		return errFalha
	}
	return f.Repository.MarcarExtracao(ctx, db, id, q)
}
func (f falhaRepo) Versoes(ctx context.Context, db database.DBTX, id uuid.UUID) (map[int]string, error) {
	if f.falha["Versoes"] {
		return nil, errFalha
	}
	return f.Repository.Versoes(ctx, db, id)
}
func (f falhaRepo) IDs(ctx context.Context, db database.DBTX, id uuid.UUID, tipo string) (map[int]bool, error) {
	if f.falha["IDs"] {
		return nil, errFalha
	}
	return f.Repository.IDs(ctx, db, id, tipo)
}
func (f falhaRepo) GravarItens(ctx context.Context, db database.DBTX, id uuid.UUID, itens []domain.Item) error {
	if f.falha["GravarItens"] {
		return errFalha
	}
	return f.Repository.GravarItens(ctx, db, id, itens)
}
func (f falhaRepo) Contagens(ctx context.Context, db database.DBTX, id uuid.UUID) (map[string]int, error) {
	if f.falha["Contagens"] {
		return nil, errFalha
	}
	return f.Repository.Contagens(ctx, db, id)
}
func (f falhaRepo) Grupos(ctx context.Context, db database.DBTX, id uuid.UUID) ([]domain.Grupo, error) {
	if f.falha["Grupos"] {
		return nil, errFalha
	}
	return f.Repository.Grupos(ctx, db, id)
}
func (f falhaRepo) Nomes(ctx context.Context, db database.DBTX, id uuid.UUID, tipo string) (map[int]string, error) {
	if f.falha["Nomes:"+tipo] {
		return nil, errFalha
	}
	return f.Repository.Nomes(ctx, db, id, tipo)
}
func (f falhaRepo) SLAs(ctx context.Context, db database.DBTX, id uuid.UUID) ([]domain.NivelSLA, error) {
	if f.falha["SLAs"] {
		return nil, errFalha
	}
	return f.Repository.SLAs(ctx, db, id)
}
func (f falhaRepo) ListarTabela(ctx context.Context, db database.DBTX, id uuid.UUID) ([]domain.Linha, error) {
	if f.falha["ListarTabela"] {
		return nil, errFalha
	}
	return f.Repository.ListarTabela(ctx, db, id)
}
func (f falhaRepo) TrocarTabela(ctx context.Context, db database.DBTX, id uuid.UUID, l []domain.Linha) error {
	if f.falha["TrocarTabela"] {
		return errFalha
	}
	return f.Repository.TrocarTabela(ctx, db, id, l)
}
func (f falhaRepo) InserirTarefa(ctx context.Context, db database.DBTX, t domain.Tarefa) error {
	if f.falha["InserirTarefa"] {
		return errFalha
	}
	return f.Repository.InserirTarefa(ctx, db, t)
}
func (f falhaRepo) GetTarefa(ctx context.Context, db database.DBTX, id uuid.UUID) (domain.Tarefa, error) {
	if f.falha["GetTarefa"] {
		return domain.Tarefa{}, errFalha
	}
	return f.Repository.GetTarefa(ctx, db, id)
}
func (f falhaRepo) ListarTarefas(ctx context.Context, db database.DBTX, id uuid.UUID, n int) ([]domain.Tarefa, error) {
	if f.falha["ListarTarefas"] {
		return nil, errFalha
	}
	return f.Repository.ListarTarefas(ctx, db, id, n)
}
func (f falhaRepo) Pegar(ctx context.Context, db database.DBTX, c *uuid.UUID, tipos []string, modo string) (domain.Tarefa, bool, error) {
	if f.falha["Pegar"] {
		return domain.Tarefa{}, false, errFalha
	}
	return f.Repository.Pegar(ctx, db, c, tipos, modo)
}
func (f falhaRepo) Progresso(ctx context.Context, db database.DBTX, id uuid.UUID, a, b int) (string, error) {
	if f.falha["Progresso"] {
		return "", errFalha
	}
	return f.Repository.Progresso(ctx, db, id, a, b)
}
func (f falhaRepo) Terminar(ctx context.Context, db database.DBTX, id uuid.UUID, e string, r []byte, m string) error {
	if f.falha["Terminar"] {
		return errFalha
	}
	return f.Repository.Terminar(ctx, db, id, e, r, m)
}
func (f falhaRepo) Cancelar(ctx context.Context, db database.DBTX, id uuid.UUID) error {
	if f.falha["Cancelar"] {
		return errFalha
	}
	return f.Repository.Cancelar(ctx, db, id)
}
func (f falhaRepo) Retomar(ctx context.Context, db database.DBTX, antes time.Time) (int, error) {
	if f.falha["Retomar"] {
		return 0, errFalha
	}
	return f.Repository.Retomar(ctx, db, antes)
}

// extrator grava dois chamados e uma pessoa, avisa o progresso e devolve
// o resumo; err falha; esperar segura até o contexto acabar.
type extrator struct {
	err     error
	esperar bool
	chamou  int
}

func (e *extrator) Extrair(ctx context.Context, c domain.Conexao, cred domain.Credenciais, destino Destino, progresso Progresso) (any, error) {
	e.chamou++
	if e.err != nil {
		return nil, e.err
	}
	if _, err := destino.Versoes(ctx); err != nil {
		return nil, err
	}
	if _, err := destino.Pessoas(ctx); err != nil {
		return nil, err
	}
	d := json.RawMessage(`{}`)
	if err := destino.Gravar(ctx, []domain.Item{
		{Tipo: domain.ItemChamado, ID: 1, Versao: "v", Dados: d}, {Tipo: domain.ItemChamado, ID: 2, Versao: "v", Dados: d},
		{Tipo: domain.ItemPessoa, ID: 9, Dados: d},
	}); err != nil {
		return nil, err
	}
	progresso(2, 2)
	if e.esperar {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return map[string]any{"chamados": 2, "login": cred.Login}, nil
}

// carregador lê a área de preparo e devolve o que viu.
type carregador struct {
	err      error
	rel      any
	esperar  bool
	gravou   bool
	tabela   []domain.Linha
	chamados int
	desfez   bool
	// antes roda antes do progresso: é onde o teste cancela a tarefa.
	antes func()
}

func (c *carregador) Carregar(ctx context.Context, conexao domain.Conexao, tabela []domain.Linha, gravar bool, leitor Leitor, progresso Progresso) (any, error) {
	c.gravou, c.tabela = gravar, tabela
	itens, total, err := leitor.Itens(ctx, domain.ItemChamado, 0, 100)
	if err != nil {
		return nil, err
	}
	if _, _, err := leitor.Item(ctx, domain.ItemPessoa, 9); err != nil {
		return nil, err
	}
	c.chamados = len(itens)
	if c.antes != nil {
		c.antes()
	}
	progresso(len(itens), total)
	if c.esperar {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	if c.err != nil {
		return nil, c.err
	}
	if c.rel != nil {
		return c.rel, nil
	}
	return map[string]int{"importados": total}, nil
}

func (c *carregador) Desfazer(ctx context.Context, gravar bool, progresso Progresso) (any, error) {
	c.desfez, c.gravou = true, gravar
	progresso(1, 1)
	return map[string]int{"chamados": 3}, c.err
}

// novoServico é o serviço sobre o banco de teste, com os dublês.
func novoServico(t *testing.T, repo domain.Repository) (*Service, *extrator, *carregador) {
	t.Helper()
	ext, car := &extrator{}, &carregador{}
	s := NewService(dbtest.Pool(t), repo, eventos{}, cifra(t)).WithExtrator(ext).WithCarregador(car)
	return s, ext, car
}

// conexao cria uma conexão e a apaga no fim do teste.
func conexao(t *testing.T, s *Service, modo string) domain.Conexao {
	t.Helper()
	url := "http://glpi/apirest.php"
	c, err := s.CriarConexao(context.Background(), admin(), ConexaoEntrada{Tipo: "glpi", Nome: "GLPI " + uuid.NewString()[:6], Modo: modo, URL: url})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = infrastructure.NewRepository().DeleteConexao(context.Background(), s.pool, c.ID) })
	return c
}

func (f falhaRepo) SetSincronizar(ctx context.Context, db database.DBTX, id uuid.UUID, ligar bool) error {
	if f.falha["SetSincronizar"] {
		return errFalha
	}
	return f.Repository.SetSincronizar(ctx, db, id, ligar)
}
func (f falhaRepo) ParaSincronizar(ctx context.Context, db database.DBTX, antes time.Time) ([]uuid.UUID, error) {
	if f.falha["ParaSincronizar"] {
		return nil, errFalha
	}
	if f.falha["ParaSincronizar:todas"] {
		// Como se a conexão estivesse livre: a inserção esbarra na ativa.
		return []uuid.UUID{f.alvo}, nil
	}
	return f.Repository.ParaSincronizar(ctx, db, antes)
}
