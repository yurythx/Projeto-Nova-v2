package application

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// ListarHistorico devolve a trilha do projeto para quem pode lê-lo.
func (s *Service) ListarHistorico(ctx context.Context, idt auth.Identity, id uuid.UUID) ([]domain.Marco, error) {
	if _, err := s.Obter(ctx, idt, id); err != nil {
		return nil, err
	}
	lista, err := s.repo.ListarHistorico(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	for i := range lista {
		lista[i].Texto = textoDoHistorico(lista[i].Acao, lista[i].Antes, lista[i].Depois)
	}
	return lista, nil
}

func textoDoHistorico(acao string, antes, depois json.RawMessage) string {
	switch acao {
	case EventProjetoCreated:
		return "Projeto aberto"
	case "projetos.projeto.reopened":
		if m := motivoDe(depois, "reabertura"); m != "" {
			return "Reaberto. Motivo: " + m
		}
		return "Reaberto"
	case "projetos.projeto.cancelled":
		if m := motivoDe(depois, "cancelamento"); m != "" {
			return "Cancelado. Motivo: " + m
		}
		return "Cancelado"
	case EventProjetoTransitioned:
		if e := rotuloEstado(campo(depois, "estado")); e != "" {
			return "Passou para " + e
		}
		return "Estado alterado"
	case EventProjetoUpdated:
		de, para := campo(antes, "gerente_nome"), campo(depois, "gerente_nome")
		if para != "" && !strings.EqualFold(de, para) {
			return "Responsável passou a ser " + para
		}
		return "Ficha alterada"
	default:
		return "Registro do projeto"
	}
}

func motivoDe(raw json.RawMessage, chave string) string {
	var doc map[string]json.RawMessage
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	var bloco struct {
		Motivo string `json:"motivo"`
	}
	if json.Unmarshal(doc[chave], &bloco) != nil {
		return ""
	}
	return strings.TrimSpace(bloco.Motivo)
}

func campo(raw json.RawMessage, chave string) string {
	var doc map[string]json.RawMessage
	if json.Unmarshal(raw, &doc) != nil {
		return ""
	}
	var s string
	if json.Unmarshal(doc[chave], &s) != nil {
		return ""
	}
	return strings.TrimSpace(s)
}

func rotuloEstado(estado string) string {
	switch estado {
	case domain.EstadoPlanejado:
		return "planejado"
	case domain.EstadoEmAndamento:
		return "em andamento"
	case domain.EstadoSuspenso:
		return "suspenso"
	case domain.EstadoConcluido:
		return "concluído"
	case domain.EstadoCancelado:
		return "cancelado"
	default:
		return estado
	}
}
