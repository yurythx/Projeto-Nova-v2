// Command importarglpi traz o histórico do GLPI para o atendimento
// (ADR 072). Sem --gravar, só simula: lê tudo, casa as pessoas e diz o
// que faria. Rodar de novo pula o que já entrou.
//
// Os tokens vêm do ambiente, não da linha de comando, para não ficarem no
// histórico do shell:
//
//	GLPI_URL=https://glpi.prefeitura/apirest.php GLPI_APP_TOKEN=... GLPI_USER_TOKEN=... \
//	DB_HOST=... DB_PORT=5432 DB_NAME=nova DB_USER=nova DB_PASSWORD=... \
//	  go run ./cmd/importarglpi --unidade <uuid> [--gravar]
//
// Sem token de usuário, GLPI_LOGIN e GLPI_PASSWORD abrem a sessão.
// GLPI_PERFIL escolhe o perfil (padrão: Super-Admin, se a conta tiver):
// o perfil decide quais entidades e categorias a API mostra.
package main

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/glpi"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/config"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
)

// opcoes é o que a linha de comando e o ambiente dizem.
type opcoes struct {
	url, appToken, userToken string
	login, senha, perfil     string
	mapa                     string
	unidade                  *uuid.UUID
	gravar                   bool
	fuso                     *time.Location
	pagina                   int
}

func main() {
	op, err := ler(os.Args[1:], os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "importarglpi:", err)
		os.Exit(2)
	}
	if err := rodar(context.Background(), op, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "importarglpi:", err)
		os.Exit(1)
	}
}

func ler(args []string, env func(string) string) (opcoes, error) {
	fs := flag.NewFlagSet("importarglpi", flag.ContinueOnError)
	unidade := fs.String("unidade", "", "unidade dona das ofertas e dos chamados importados (vazio = institucional)")
	gravar := fs.Bool("gravar", false, "grava; sem ele, só simula")
	fuso := fs.String("fuso", "America/Sao_Paulo", "fuso das datas do GLPI")
	pagina := fs.Int("pagina", glpi.PaginaPadrao, "chamados por página da API")
	mapa := fs.String("mapa", "", "tabela de-para revisada (CSV com ;): categoria -> oferta, unidade e prioridade (ADR 075)")
	if err := fs.Parse(args); err != nil {
		return opcoes{}, err
	}
	op := opcoes{url: env("GLPI_URL"), appToken: env("GLPI_APP_TOKEN"), userToken: env("GLPI_USER_TOKEN"),
		login: env("GLPI_LOGIN"), senha: env("GLPI_PASSWORD"), perfil: env("GLPI_PERFIL"), mapa: *mapa, gravar: *gravar, pagina: *pagina}
	if op.url == "" || (op.userToken == "" && (op.login == "" || op.senha == "")) {
		return opcoes{}, errors.New("defina GLPI_URL e GLPI_USER_TOKEN, ou GLPI_LOGIN e GLPI_PASSWORD (e GLPI_APP_TOKEN, se o GLPI pedir)")
	}
	if *unidade != "" {
		id, err := uuid.Parse(*unidade)
		if err != nil {
			return opcoes{}, fmt.Errorf("--unidade: %w", err)
		}
		op.unidade = &id
	}
	loc, err := time.LoadLocation(*fuso)
	if err != nil {
		return opcoes{}, fmt.Errorf("--fuso: %w", err)
	}
	op.fuso = loc
	return op, nil
}

func rodar(ctx context.Context, op opcoes, saida io.Writer) error {
	dbCfg, err := config.LoadDatabase()
	if err != nil {
		return fmt.Errorf("configuração do banco (DB_HOST/DB_PORT/DB_NAME/DB_USER/DB_PASSWORD): %w", err)
	}
	pool, err := database.New(ctx, dbCfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	return importar(ctx, op, pool, saida)
}

// lerMapa lê a tabela, casando o nome de cada unidade com as daqui.
func lerMapa(ctx context.Context, pool *pgxpool.Pool, caminho string) (*glpi.Mapa, error) {
	f, err := os.Open(caminho)
	if err != nil {
		return nil, fmt.Errorf("--mapa: %w", err)
	}
	defer f.Close()
	rows, err := pool.Query(ctx, `SELECT u.id, u.nome, u.sigla, COALESCE(e.nome, ''), COALESCE(e.sigla, ''), u.parent_id IS NULL
		FROM unidades u LEFT JOIN entidades e ON e.id = u.entidade_id
		WHERE u.ativo
		ORDER BY u.parent_id IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("unidades: %w", err)
	}
	defer rows.Close()
	unidades := map[string]uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		var nome, sigla, entNome, entSigla string
		var raiz bool
		if err := rows.Scan(&id, &nome, &sigla, &entNome, &entSigla, &raiz); err != nil {
			return nil, err
		}
		unidades[nome] = id
		if raiz {
			for _, chave := range []string{sigla, entNome, entSigla} {
				if chave != "" {
					if _, existe := unidades[chave]; !existe {
						unidades[chave] = id
					}
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return glpi.LerMapa(f, unidades)
}

func importar(ctx context.Context, op opcoes, pool *pgxpool.Pool, saida io.Writer) (err error) {
	cliente := glpi.NovoCliente(op.url, op.appToken, op.userToken)
	cliente.Login, cliente.Senha, cliente.Perfil = op.login, op.senha, op.perfil
	if err := cliente.Abrir(ctx); err != nil {
		return fmt.Errorf("sessão no GLPI: %w", err)
	}
	// O perfil decide o que a API mostra: outro perfil, outras entidades.
	fmt.Fprintf(os.Stderr, "importarglpi: perfil no GLPI: %s\n", cmp.Or(cliente.PerfilAtivo, "o padrão da conta"))
	defer func() { _ = cliente.Fechar(context.WithoutCancel(ctx)) }()
	im := &glpi.Importador{Fonte: cliente, Pool: pool, Repo: infrastructure.NewRepository(), Unidade: op.unidade,
		Gravar: op.gravar, Fuso: op.fuso, Pagina: op.pagina}
	if op.mapa != "" {
		if im.Mapa, err = lerMapa(ctx, pool, op.mapa); err != nil {
			return err
		}
	}
	rel, err := im.Executar(ctx)
	enc := json.NewEncoder(saida)
	enc.SetIndent("", "  ")
	if errEnc := enc.Encode(rel); errEnc != nil && err == nil {
		err = errEnc
	}
	if err == nil && !op.gravar {
		_, err = fmt.Fprintln(saida, "Simulação: nada foi gravado. Rode de novo com --gravar.")
	}
	return err
}
