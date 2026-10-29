package app

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/glpi"
	atendinfra "github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	intapp "github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/application"
	intdomain "github.com/yurythx/projeto-nova-v2/internal/modules/integracoes/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// As portas das integrações (ADR 076): o módulo não conhece o GLPI, e o
// atendimento não conhece a área de preparo. Aqui as duas pontas se
// encontram.

// paraGLPI e paraItem copiam o item da área de preparo entre os dois
// lados: os campos são os mesmos.
func paraGLPI(it intdomain.Item) glpi.ItemPreparo {
	return glpi.ItemPreparo{Tipo: it.Tipo, ID: it.ID, Versao: it.Versao, Nome: it.Nome, Entidade: it.Entidade, Categoria: it.Categoria,
		SLA: it.SLA, Minutos: it.Minutos, Classe: it.Classe, Data: it.Data, Dados: it.Dados}
}

func paraItem(it glpi.ItemPreparo) intdomain.Item {
	return intdomain.Item{Tipo: it.Tipo, ID: it.ID, Versao: it.Versao, Nome: it.Nome, Entidade: it.Entidade, Categoria: it.Categoria,
		SLA: it.SLA, Minutos: it.Minutos, Classe: it.Classe, Data: it.Data, Dados: it.Dados}
}

// preparoGLPI é a área de preparo vista pela extração do GLPI.
type preparoGLPI struct{ d intapp.Destino }

func (p preparoGLPI) Versoes(ctx context.Context) (map[int]string, error) { return p.d.Versoes(ctx) }
func (p preparoGLPI) Pessoas(ctx context.Context) (map[int]bool, error)   { return p.d.Pessoas(ctx) }
func (p preparoGLPI) Gravar(ctx context.Context, itens []glpi.ItemPreparo) error {
	out := make([]intdomain.Item, len(itens))
	for i, it := range itens {
		out[i] = paraItem(it)
	}
	return p.d.Gravar(ctx, out)
}

// leitorGLPI é a área de preparo vista pelo importador do GLPI.
type leitorGLPI struct{ l intapp.Leitor }

func (l leitorGLPI) Itens(ctx context.Context, tipo string, de, limite int) ([]glpi.ItemPreparo, int, error) {
	itens, total, err := l.l.Itens(ctx, tipo, de, limite)
	out := make([]glpi.ItemPreparo, len(itens))
	for i, it := range itens {
		out[i] = paraGLPI(it)
	}
	return out, total, err
}

func (l leitorGLPI) Item(ctx context.Context, tipo string, id int) (glpi.ItemPreparo, bool, error) {
	it, ok, err := l.l.Item(ctx, tipo, id)
	return paraGLPI(it), ok, err
}

// sessaoGLPI abre a sessão da API do GLPI; os testes trocam por um GLPI
// falso.
type sessaoGLPI func(ctx context.Context, c intdomain.Conexao, cred intdomain.Credenciais) (glpi.Extraivel, func(), error)

func abrirGLPI(ctx context.Context, c intdomain.Conexao, cred intdomain.Credenciais) (glpi.Extraivel, func(), error) {
	cli := glpi.NovoCliente(c.URL, cred.AppToken, cred.UserToken)
	cli.Login, cli.Senha, cli.Perfil = cred.Login, cred.Senha, c.Perfil
	if err := cli.Abrir(ctx); err != nil {
		return nil, nil, fmt.Errorf("sessão no GLPI: %w", err)
	}
	return cli, func() { _ = cli.Fechar(context.WithoutCancel(ctx)) }, nil
}

// extratorGLPI lê o GLPI a partir do servidor (modo direto).
type extratorGLPI struct{ abrir sessaoGLPI }

func (e extratorGLPI) Extrair(ctx context.Context, c intdomain.Conexao, cred intdomain.Credenciais, destino intapp.Destino, progresso intapp.Progresso) (any, error) {
	if c.Tipo != intdomain.TipoGLPI {
		return nil, fmt.Errorf("origem %q sem extrator", c.Tipo)
	}
	origem, fechar, err := e.abrir(ctx, c, cred)
	if err != nil {
		return nil, err
	}
	defer fechar()
	return glpi.Extrair(ctx, origem, preparoGLPI{destino}, 0, progresso)
}

// carregadorGLPI roda o importador do atendimento sobre a área de preparo.
type carregadorGLPI struct {
	pool  *pgxpool.Pool
	ativo func() bool
}

func (c carregadorGLPI) Carregar(ctx context.Context, conexao intdomain.Conexao, tabela []intdomain.Linha, gravar bool, leitor intapp.Leitor, progresso intapp.Progresso) (any, error) {
	if !c.ativo() {
		return nil, errors.New("o atendimento está desligado: ligue o módulo antes de carregar")
	}
	fuso, err := time.LoadLocation(conexao.Fuso)
	if err != nil {
		return nil, err
	}
	linhas := make([]glpi.LinhaMapa, len(tabela))
	for i, l := range tabela {
		nome := l.UnidadeNome
		if nome == "" {
			nome = "(sem unidade) " + l.Secretaria
		}
		linhas[i] = glpi.LinhaMapa{Secretaria: l.Secretaria, Categoria: l.Categoria,
			Destino: glpi.Destino{Unidade: l.UnidadeID, UnidadeNome: nome, Oferta: l.Oferta, Catalogo: l.Catalogo, Tipo: l.Tipo,
				Prioridade: l.Prioridade}}
	}
	im := &glpi.Importador{Fonte: &glpi.FontePreparo{Leitor: leitorGLPI{leitor}}, Pool: c.pool, Repo: atendinfra.NewRepository(),
		Gravar: gravar, Fuso: fuso, Mapa: glpi.NovoMapa(linhas), Progresso: progresso, Sincronizar: conexao.Sincronizar}
	return im.Executar(ctx)
}

// Desfazer remove o que a carga do GLPI trouxe (ADR 079).
func (c carregadorGLPI) Desfazer(ctx context.Context, gravar bool, progresso intapp.Progresso) (any, error) {
	if !c.ativo() {
		return nil, errors.New("o atendimento está desligado: ligue o módulo antes de desfazer")
	}
	return glpi.Desfazer(ctx, c.pool, gravar, progresso)
}

// espelhoGLPI diz ao atendimento se a sincronização está ligada. Com as
// integrações desligadas, não há sincronização.
type espelhoGLPI struct {
	svc   *intapp.Service
	ativo func() bool
}

func (e espelhoGLPI) Ativo(ctx context.Context) (bool, error) {
	if !e.ativo() {
		return false, nil
	}
	return e.svc.SincronizacaoAtiva(ctx)
}

// credenciais encadeia os reconhecedores de token de máquina: o primeiro
// que reconhece o formato decide.
func credenciais(reconhecedores ...auth.Credencial) auth.Credencial {
	return func(ctx context.Context, token string) (auth.Identity, bool, error) {
		for _, r := range reconhecedores {
			if id, ok, err := r(ctx, token); ok || err != nil {
				return id, ok, err
			}
		}
		return auth.Identity{}, false, nil
	}
}
