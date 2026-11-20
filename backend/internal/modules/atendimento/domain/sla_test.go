package domain_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
)

func TestNovaSLA(t *testing.T) {
	t.Run("valida com sucesso", func(t *testing.T) {
		uID := uuid.New()
		sla, err := domain.NovaSLA("SLA Padrão", "Descrição de teste", 30, 240, true, domain.CalendarioExpediente, &uID)
		require.NoError(t, err)
		assert.Equal(t, "SLA Padrão", sla.Nome)
		assert.Equal(t, 30, sla.TempoRespostaMinutos)
		assert.Equal(t, 240, sla.TempoSolucaoMinutos)
		assert.True(t, sla.PausarEmEspera)
		assert.Equal(t, domain.CalendarioExpediente, sla.CalendarioTipo)
		assert.Equal(t, &uID, sla.UnidadeID)
		assert.True(t, sla.Ativa)
	})

	t.Run("falha com nome vazio", func(t *testing.T) {
		_, err := domain.NovaSLA("", "Desc", 30, 240, true, domain.CalendarioExpediente, nil)
		assert.ErrorIs(t, err, domain.ErrSLAInvalida)
	})

	t.Run("falha com tempos invalidos", func(t *testing.T) {
		_, err := domain.NovaSLA("SLA", "Desc", 0, 240, true, domain.CalendarioExpediente, nil)
		assert.ErrorIs(t, err, domain.ErrSLAInvalida)

		_, err = domain.NovaSLA("SLA", "Desc", 300, 240, true, domain.CalendarioExpediente, nil)
		assert.ErrorIs(t, err, domain.ErrSLAInvalida)
	})

	t.Run("aplicar SLA em solicitacao", func(t *testing.T) {
		sla, err := domain.NovaSLA("SLA Crítica", "Resolvido rápido", 15, 60, true, domain.CalendarioCorrido, nil)
		require.NoError(t, err)

		sol := domain.Solicitacao{}
		sol = sol.AplicarSLA(&sla)
		require.NotNil(t, sol.MetaRespostaMin)
		require.NotNil(t, sol.MetaSolucaoMin)
		require.NotNil(t, sol.Prioridade)
		assert.Equal(t, 15, *sol.MetaRespostaMin)
		assert.Equal(t, 60, *sol.MetaSolucaoMin)
		assert.Equal(t, domain.PrioridadeMedia, *sol.Prioridade)

		alta := domain.PrioridadeAlta
		sol.Prioridade = &alta
		sol = sol.AplicarSLA(&sla)
		assert.Equal(t, domain.PrioridadeAlta, *sol.Prioridade)
	})

	t.Run("SLA nula ou inativa não aplica", func(t *testing.T) {
		sol := domain.Solicitacao{}
		assert.Nil(t, sol.AplicarSLA(nil).MetaRespostaMin)
		inativa := domain.SLA{Ativa: false, TempoRespostaMinutos: 5, TempoSolucaoMinutos: 10}
		assert.Nil(t, sol.AplicarSLA(&inativa).Prioridade)
	})

	t.Run("calendário desconhecido vira expediente", func(t *testing.T) {
		sla, err := domain.NovaSLA("Outro", "", 10, 20, false, "marte", nil)
		require.NoError(t, err)
		assert.Equal(t, domain.CalendarioExpediente, sla.CalendarioTipo)
	})
}
