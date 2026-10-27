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
package main

import (
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
	if err := fs.Parse(args); err != nil {
		return opcoes{}, err
	}
	op := opcoes{url: env("GLPI_URL"), appToken: env("GLPI_APP_TOKEN"), userToken: env("GLPI_USER_TOKEN"), gravar: *gravar, pagina: *pagina}
	if op.url == "" || op.userToken == "" {
		return opcoes{}, errors.New("defina GLPI_URL e GLPI_USER_TOKEN (e GLPI_APP_TOKEN, se o GLPI pedir)")
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

func importar(ctx context.Context, op opcoes, pool *pgxpool.Pool, saida io.Writer) error {
	cliente := glpi.NovoCliente(op.url, op.appToken, op.userToken)
	if err := cliente.Abrir(ctx); err != nil {
		return fmt.Errorf("sessão no GLPI: %w", err)
	}
	defer func() { _ = cliente.Fechar(context.WithoutCancel(ctx)) }()
	im := &glpi.Importador{Fonte: cliente, Pool: pool, Repo: infrastructure.NewRepository(), Unidade: op.unidade,
		Gravar: op.gravar, Fuso: op.fuso, Pagina: op.pagina}
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
