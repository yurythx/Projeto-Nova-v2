package auth

import "strings"

// Roles de realm reconhecidos diretamente no token. O Keycloak é a fonte
// da verdade para roles; o controle fino é feito por Perfis (permissões
// "recurso:ação") resolvidos pelo IAM a partir das lotações e do
// mapeamento de grupos do AD. nova-admin é o único role com efeito
// próprio (equivale à permissão "*"), para que um administrador nunca
// fique trancado fora da plataforma por um mapeamento mal configurado.
const (
	RoleAdmin RoleName = "nova-admin"
	RoleUser  RoleName = "nova-user"
)

type RoleName = string

// Permission é uma capacidade granular no formato "recurso:ação",
// verificada por RequirePermission ANTES da camada de aplicação (A01).
type Permission string

// Catálogo de permissões da plataforma. Cada plugin declara as suas no
// próprio Manifest (ver internal/platform/kernel); estas constantes
// existem para que as checagens no código não espalhem strings literais.
const (
	PermUsersRead      Permission = "users:read"
	PermUsersManage    Permission = "users:manage"
	PermIAMManage      Permission = "iam:manage"
	PermAuditRead      Permission = "audit:read"
	PermAuditVerify    Permission = "audit:verify"
	PermModulesManage  Permission = "modules:manage"
	PermKeycloakManage Permission = "keycloak:manage"
	PermBrandingManage Permission = "branding:manage"
	PermMonitoringRead Permission = "monitoring:read"
	// PermMonitoringManage: ações operacionais do painel (reprocessar o outbox).
	PermMonitoringManage Permission = "monitoring:manage"

	PermMercurioManage    Permission = "mercurio:manage"
	PermEgressManage      Permission = "egress:manage"
	PermBlogManage        Permission = "blog:manage"
	PermCatalogManage     Permission = "catalog:manage"
	PermContactRead       Permission = "contact:read"
	PermContactManage     Permission = "contact:manage"
	PermDirectoryManage   Permission = "directory:manage"
	PermCalendarManage    Permission = "calendar:manage"
	PermFilesManage       Permission = "files:manage"
	PermWikiManage        Permission = "wiki:manage"
	PermSignumManage      Permission = "signum:manage"
	PermTramiteCreate     Permission = "tramite:create"
	PermTramiteRoute      Permission = "tramite:route"
	PermTramiteManage     Permission = "tramite:manage"
	PermProdutosManage    Permission = "produtos:manage"
	PermAtendimentoManage Permission = "atendimento:manage"
	// PermAtendimentoAbrir: o servidor abre e acompanha as próprias solicitações.
	PermAtendimentoAbrir Permission = "atendimento:abrir"
	// PermAtendimentoAtender: a equipe atende a fila da unidade (assume,
	// responde, muda o estado). Cadastrar oferta e equipe é manage.
	PermAtendimentoAtender Permission = "atendimento:atender"
	// Ativos, problemas e mudanças (fases 6 e 7 da gestão de serviços).
	PermAtivosManage    Permission = "ativos:manage"
	PermAtivosReportar  Permission = "ativos:reportar"
	PermProblemasRead   Permission = "problemas:read"
	PermProblemasManage Permission = "problemas:manage"
	PermMudancasManage  Permission = "mudancas:manage"
	PermMudancasAprovar Permission = "mudancas:aprovar"
	PermProjetosRead    Permission = "projetos:read"
	PermProjetosManage  Permission = "projetos:manage"
	PermContratosManage Permission = "contratos:manage"
	// Cofre de senhas. A lista não devolve o segredo; revelar é outra ação.
	PermCofreRead     Permission = "cofre:read"
	PermCofreReveal   Permission = "cofre:reveal"
	PermCofreManage   Permission = "cofre:manage"
	PermCofreGrant    Permission = "cofre:grant"
	PermExampleManage Permission = "example:manage"
	// PermIAManage: conexões de inteligência artificial (Configurações).
	PermIAManage Permission = "ia:manage"
	// PermCanaisManage: e-mail e WhatsApp dos avisos (ADR 078).
	PermCanaisManage Permission = "canais:manage"
	// Integrações (ADR 076): migrar de outra ferramenta. A do conector é
	// só do token de máquina, nunca de um perfil.
	PermIntegracoesManage   Permission = "integracoes:manage"
	PermIntegracoesConector Permission = "integracoes:conector"
)

// HasPermission reporta se identity possui permission, considerando os
// curingas "*" (tudo) e "recurso:*" (toda ação do recurso). O role
// nova-admin equivale a "*".
func HasPermission(identity Identity, permission Permission) bool {
	if identity.HasRole(RoleAdmin) {
		return true
	}
	return MatchPermission(identity.Permissions, string(permission))
}

// MatchPermission aplica a regra de curingas sobre uma lista de
// permissões concedidas.
func MatchPermission(granted []string, want string) bool {
	resource, _, _ := strings.Cut(want, ":")
	for _, g := range granted {
		switch {
		case g == "*":
			return true
		case g == want:
			return true
		case strings.HasSuffix(g, ":*") && strings.TrimSuffix(g, ":*") == resource:
			return true
		}
	}
	return false
}
