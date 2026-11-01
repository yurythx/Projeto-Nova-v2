package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yurythx/projeto-nova-v2/internal/modules/riscos/domain"
)

func TestNovoRisco(t *testing.T) {
	uID := uuid.New()
	r, err := domain.NovoRisco(
		"RSK-100",
		"Falha de refrigeração no Datacenter",
		"Ar-condicionado de precisão desligado",
		domain.CategoriaInfra,
		uID,
		4,
		4,
		domain.EstrategiaMitigar,
		"Instalar segundo ar-condicionado redundante",
		nil,
		"Equipe de Infra",
	)
	require.NoError(t, err)
	assert.Equal(t, "RSK-100", r.Codigo)
	assert.Equal(t, 16, r.Severidade)
	assert.Equal(t, "alto", r.NivelSeveridade())
	assert.Equal(t, domain.StatusIdentificado, r.Status)

	t.Run("calculo de severidade critica", func(t *testing.T) {
		rCritico, err := domain.NovoRisco("RSK-200", "Ataque Ransomware", "Sequestro", domain.CategoriaSeguranca, uID, 5, 5, domain.EstrategiaMitigar, "Backup", nil, "Sec")
		require.NoError(t, err)
		assert.Equal(t, 25, rCritico.Severidade)
		assert.Equal(t, "critico", rCritico.NivelSeveridade())
	})

	t.Run("validacao de limites", func(t *testing.T) {
		_, err := domain.NovoRisco("", "A", "desc", domain.CategoriaInfra, uID, 4, 4, domain.EstrategiaMitigar, "", nil, "")
		assert.ErrorIs(t, err, domain.ErrRiscoInvalido)

		_, err = domain.NovoRisco("RSK-999", "Titulo valido", "desc", domain.CategoriaInfra, uID, 6, 2, domain.EstrategiaMitigar, "", nil, "")
		assert.ErrorIs(t, err, domain.ErrRiscoInvalido)
	})
}

