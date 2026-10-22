package keycloakconfig

import (
	"testing"

	"github.com/yurythx/projeto-nova-v2/internal/platform/config"
)

func TestSettings_ToKeycloakConfig(t *testing.T) {
	s := Settings{
		IssuerURL:        "https://sso.orgao.gov.br/realms/nova",
		Realm:            "nova",
		ClientID:         "nova-backend",
		ClientSecret:     "segredo",
		Audience:         "nova-backend",
		FrontendClientID: "nova-frontend", // não faz parte de config.KeycloakConfig — deve ser ignorado
	}

	want := config.KeycloakConfig{
		IssuerURL:    "https://sso.orgao.gov.br/realms/nova",
		Realm:        "nova",
		ClientID:     "nova-backend",
		ClientSecret: "segredo",
		Audience:     "nova-backend",
	}

	if got := s.ToKeycloakConfig(); got != want {
		t.Errorf("ToKeycloakConfig() = %+v, want %+v", got, want)
	}
}
