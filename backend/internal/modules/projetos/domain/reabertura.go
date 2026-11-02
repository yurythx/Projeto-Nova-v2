package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

const MotivoReaberturaMax = 2000

// Reabertura é a aprovação que tira o projeto de concluído.
type Reabertura struct {
	ID            uuid.UUID `json:"id"`
	ProjetoID     uuid.UUID `json:"projeto_id"`
	Motivo        string    `json:"motivo"`
	AprovadorID   uuid.UUID `json:"aprovador_id"`
	AprovadorNome string    `json:"aprovador_nome"`
	CreatedAt     time.Time `json:"created_at"`
}

// NormalizarMotivo exige o texto da reabertura, até 2000 caracteres.
func NormalizarMotivo(motivo string) (string, error) {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" || utf8.RuneCountInString(motivo) > MotivoReaberturaMax {
		return "", ErrMotivoObrigatorio
	}
	return motivo, nil
}

// PodeReabrir é administrador (papel ou gestão global) ou superior da
// unidade: gestão na entidade dela ou numa unidade acima, não nela mesma.
// Projeto sem unidade só o administrador reabre.
func PodeReabrir(idt auth.Identity, unidade, entidade *uuid.UUID, acima []uuid.UUID) bool {
	if idt.IsAdmin() || auth.CanGlobal(idt, auth.PermProjetosManage) {
		return true
	}
	if unidade == nil {
		return false
	}
	for _, sc := range idt.Scopes {
		if !auth.MatchPermission(sc.Permissions, string(auth.PermProjetosManage)) {
			continue
		}
		if sc.UnidadeID != nil {
			for _, id := range acima {
				if id == *sc.UnidadeID {
					return true
				}
			}
			continue
		}
		if sc.EntidadeID != nil && entidade != nil && *sc.EntidadeID == *entidade {
			return true
		}
	}
	return false
}
