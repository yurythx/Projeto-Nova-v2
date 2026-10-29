package app

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ativos"
	"github.com/yurythx/projeto-nova-v2/internal/modules/auditoria"
	"github.com/yurythx/projeto-nova-v2/internal/modules/blog"
	"github.com/yurythx/projeto-nova-v2/internal/modules/busca"
	"github.com/yurythx/projeto-nova-v2/internal/modules/calendar"
	"github.com/yurythx/projeto-nova-v2/internal/modules/catalog"
	"github.com/yurythx/projeto-nova-v2/internal/modules/cofre"
	"github.com/yurythx/projeto-nova-v2/internal/modules/contact"
	"github.com/yurythx/projeto-nova-v2/internal/modules/contratos"
	contratosapp "github.com/yurythx/projeto-nova-v2/internal/modules/contratos/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/directory"
	"github.com/yurythx/projeto-nova-v2/internal/modules/egress"
	"github.com/yurythx/projeto-nova-v2/internal/modules/example"
	"github.com/yurythx/projeto-nova-v2/internal/modules/files"
	"github.com/yurythx/projeto-nova-v2/internal/modules/iam"
	iamTransport "github.com/yurythx/projeto-nova-v2/internal/modules/iam/transport"
	"github.com/yurythx/projeto-nova-v2/internal/modules/integracoes"
	"github.com/yurythx/projeto-nova-v2/internal/modules/mercurio"
	"github.com/yurythx/projeto-nova-v2/internal/modules/mudancas"
	"github.com/yurythx/projeto-nova-v2/internal/modules/ocorrencias"
	"github.com/yurythx/projeto-nova-v2/internal/modules/problemas"
	"github.com/yurythx/projeto-nova-v2/internal/modules/produtos"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos"
	"github.com/yurythx/projeto-nova-v2/internal/modules/quadros"
	"github.com/yurythx/projeto-nova-v2/internal/modules/signum"
	signumApp "github.com/yurythx/projeto-nova-v2/internal/modules/signum/application"
	"github.com/yurythx/projeto-nova-v2/internal/modules/tramite"
	tramiteDomain "github.com/yurythx/projeto-nova-v2/internal/modules/tramite/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/wiki"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/iaconfig"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/keycloakadmin"
	"github.com/yurythx/projeto-nova-v2/internal/platform/lgpd"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

// registerPlugins monta o catálogo de módulos (skill §6) e o registra no
// Kernel via RegisterModule. A ordem é a do menu.
func registerPlugins(d *Dependencies) {
	deps := modkit.Deps{
		Config:                d.Config,
		Logger:                d.Logger,
		Pool:                  d.DB,
		Outbox:                d.Outbox,
		Storage:               d.Storage,
		Hub:                   d.Hub,
		Notificacoes:          d.Notificacoes,
		Cipher:                d.Cipher,
		PublicLimiter:         d.RateLimiters.Public,
		ContactLimiter:        d.RateLimiters.Contact,
		InvalidatePermissions: d.IAM.Invalidate,
		ResetLoginLockout: func(ctx context.Context, username string) error {
			return d.RateLimiters.Lockout.Reset(ctx, "user:"+strings.ToLower(username))
		},
	}

	signumModule := signum.New(deps, d.keycloakCredentials, d.RateLimiters.Lockout)
	buscaModule := busca.New(deps, d.Kernel.SearchProviders)
	if d.SearchIndex != nil {
		buscaModule.WithIndex(d.SearchIndex)
	}
	produtosModule := produtos.New(deps)
	ligado := d.Kernel.Enabled
	atendimentoModule := atendimento.New(deps, produtoPort{repo: produtosModule.Repositorio(), ativo: func() bool { return d.Kernel.Enabled(produtos.Key) }})
	ativosModule := ativos.New(deps, portaProdutoAtivos(produtosModule.Repositorio(), ligado))
	iamModule := iam.New(deps, d.permissionCatalog)
	iamModule.Servico().WithDiretorio(portaSSO{cli: keycloakadmin.Novo(), credenciais: d.keycloakCredentials}, d.Logger)
	integracoesModule := integracoes.New(deps)
	integracoesModule.Servico().WithExtrator(extratorGLPI{abrir: abrirGLPI}).
		WithCarregador(carregadorGLPI{pool: d.DB, ativo: func() bool { return d.Kernel.Enabled(atendimento.Key) }})
	d.Credencial = credenciais(ativosModule.Reconhecer, integracoesModule.Reconhecer)
	// Sincronizando com o GLPI, o importado muda lá até a virada (ADR 081).
	atendimentoModule.ComEspelho(espelhoGLPI{svc: integracoesModule.Servico(), ativo: func() bool { return d.Kernel.Enabled(integracoes.Key) }}.Ativo)
	problemasModule := problemas.New(deps, portasProblemas(produtosModule.Repositorio(), atendimentoModule.Repositorio(),
		ativosModule.Repositorio(), ligado))
	alvos, produtoDaMudanca := portasMudancas(produtosModule.Repositorio(), atendimentoModule.Repositorio(),
		ativosModule.Repositorio(), problemasModule.Repositorio(), ligado)
	mudancasModule := mudancas.New(deps, alvos, produtoDaMudanca)
	mudancasModule.ComEfeitos(efeitosCascata{atend: atendimentoModule, prob: problemasModule, ligado: ligado})
	projetosModule := projetos.New(deps)
	alvosProj, produtoDoProjeto := portasProjetos(produtosModule.Repositorio(), atendimentoModule.Repositorio(),
		ativosModule.Repositorio(), problemasModule.Repositorio(), mudancasModule.Repositorio(), ligado)
	projetosModule.ComPortas(alvosProj, produtoDoProjeto)
	quadrosModule := quadros.New(deps)
	problemasModule.ComPassagens(quadrosModule)
	mudancasModule.ComPassagens(quadrosModule)
	projetosModule.ComPassagens(quadrosModule)
	ativosModule.ComPassagens(quadrosModule)
	atendimentoModule.ComPassagens(quadrosModule)
	cofreModule := cofre.New(deps)
	cofreModule.ComConfirmacao(confirmaSenha{reauth: signumModule.Reautenticador()}, d.RateLimiters.Lockout)
	atendimentoModule.ComSegredo(portaSegredo{svc: cofreModule.Servico(), ativo: func() bool { return d.Kernel.Enabled(cofre.Key) }})
	atendimentoModule.ComAtivos(ativoAtendimento{repo: ativosModule.Repositorio(), ativo: func() bool { return d.Kernel.Enabled(ativos.Key) }})
	contratosModule := contratos.New(deps, programasDoAtivo{mod: ativosModule, ativo: func() bool { return d.Kernel.Enabled(ativos.Key) }})
	atendimentoModule.ComContratos(portaContratos{svc: contratosModule.Servico(), ativo: func() bool { return d.Kernel.Enabled(contratos.Key) }})
	ia := d.Config.IA
	// A resposta pelo WhatsApp vira comentário no atendimento (ADR 078).
	d.Canais.WithRecebedor(portaWhatsApp{mod: atendimentoModule, ativo: func() bool { return d.Kernel.Enabled(atendimento.Key) }})
	atendimentoModule.ComAssistente(portaIA{roteador: iaconfig.NovoRoteador(iaconfig.NewPostgresStore(d.DB, d.Cipher),
		iaconfig.NovoCliente(), iaconfig.ConexaoDoAmbiente(ia.Endpoint, ia.APIKey, ia.Modelo, int(ia.Timeout.Seconds())), d.Logger)})

	d.Kernel.MustRegister(
		// Core imutável (nunca desativável).
		iamModule,
		auditoria.New(deps),
		// Plugins (ativação em runtime).
		mercurio.New(deps),
		egress.New(deps),
		blog.New(deps),
		catalog.New(deps),
		contact.New(deps),
		directory.New(deps),
		calendar.New(deps),
		files.New(deps),
		wiki.New(deps),
		buscaModule,
		signumModule,
		tramite.New(deps, &signaturePort{svc: signumModule.Service(), enabled: func() bool { return d.Kernel.Enabled(signum.Key) }}),
		produtosModule,
		atendimentoModule,
		ocorrencias.New(deps),
		ativosModule,
		problemasModule,
		mudancasModule,
		projetosModule,
		quadrosModule,
		contratosModule,
		cofreModule,
		integracoesModule,
		example.New(deps),
	)
	d.LGPD.SetPersonalDataSource(d.personalData)
}

// personalData lista os plugins que guardam dados pessoais do titular
// (LGPD art. 18) — todos os registrados, ativos ou não.
func (d *Dependencies) personalData() map[string]lgpd.PersonalData {
	out := map[string]lgpd.PersonalData{}
	for _, p := range d.Kernel.Plugins() {
		if pd, ok := p.(lgpd.PersonalData); ok {
			out[p.Manifest().Key] = pd
		}
	}
	return out
}

// permissionCatalog agrupa as permissões declaradas nos manifestos (tela
// de Perfis).
func (d *Dependencies) permissionCatalog() []iamTransport.PermissionGroup {
	out := []iamTransport.PermissionGroup{}
	for _, s := range d.Kernel.Status() {
		if len(s.Permissions) == 0 {
			continue
		}
		g := iamTransport.PermissionGroup{Module: s.Key, Name: s.Name}
		for _, p := range s.Permissions {
			g.Permissions = append(g.Permissions, iamTransport.PermissionItem{Key: p.Key, Description: p.Description})
		}
		out = append(out, g)
	}
	return out
}

// keycloakCredentials fornece ao Signum o client usado na reautenticação
// (configuração dinâmica salva pela tela; fallback no .env).
func (d *Dependencies) keycloakCredentials(ctx context.Context) (string, string, string) {
	if s, err := d.KeycloakCfg.Get(ctx); err == nil && s.Configured {
		return s.IssuerURL, s.ClientID, s.ClientSecret
	}
	k := d.Config.Keycloak
	return k.IssuerURL, k.ClientID, k.ClientSecret
}

// signaturePort liga a porta do Trâmite ao Signum — o único ponto em que
// os dois plugins se encontram. O envelope nasce na MESMA transação que
// muda o status do documento no Trâmite.
type signaturePort struct {
	svc     *signumApp.Service
	enabled func() bool
}

func (p *signaturePort) Available() bool { return p.enabled() }

func (p *signaturePort) OpenEnvelope(ctx context.Context, tx pgx.Tx, req tramiteDomain.SignatureRequest) (uuid.UUID, error) {
	env, err := p.svc.OpenTx(ctx, tx, signumApp.OpenRequest{
		Title: req.Title, Description: req.Description, DocumentSHA256: req.DocumentSHA256,
		SignerIDs: req.SignerIDs, Sequential: req.Sequential, SourceModule: tramite.Key, SourceRef: req.SourceRef,
	})
	if err != nil {
		return uuid.Nil, err
	}
	return env.ID, nil
}

// programasDoAtivo liga a licença ao inventário sem os plugins se importarem.
type programasDoAtivo struct {
	mod   *ativos.Module
	ativo func() bool
}

func (p programasDoAtivo) Contar(ctx context.Context, db database.DBTX, restrito bool, unidades []uuid.UUID, nome string) (contratosapp.UsoDePrograma, error) {
	if p.mod == nil || (p.ativo != nil && !p.ativo()) {
		return contratosapp.UsoDePrograma{}, contratosapp.ErrProgramasIndisponiveis
	}
	c, err := p.mod.ContarPrograma(ctx, db, restrito, unidades, nome)
	if err != nil {
		return contratosapp.UsoDePrograma{}, err
	}
	return contratosapp.UsoDePrograma{Instalados: c.Instalados, ComInventario: c.ComInventario, Parcial: c.Parcial}, nil
}

// pluginsOnInTests liga os plug-ins quando APP_ENV=test. No produto eles
// nascem desligados; a suíte precisa deles ativos para exercitar as rotas.
func pluginsOnInTests(ctx context.Context, k *kernel.Kernel) error {
	for _, s := range k.Status() {
		if s.Core || s.Enabled {
			continue
		}
		if err := k.SetEnabled(ctx, s.Key, true, "test"); err != nil {
			return err
		}
	}
	return nil
}
