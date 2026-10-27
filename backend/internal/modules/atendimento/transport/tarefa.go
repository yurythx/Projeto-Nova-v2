package transport

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// ModeloTarefaRequest é uma linha do modelo de tarefas da oferta.
type ModeloTarefaRequest struct {
	Titulo   string    `json:"titulo" validate:"required,max=120"`
	EquipeID uuid.UUID `json:"equipe_id" validate:"required"`
}

// ModeloRequest troca o modelo de tarefas (lista vazia tira as tarefas).
type ModeloRequest struct {
	Tarefas []ModeloTarefaRequest `json:"tarefas" validate:"max=20,dive"`
}

// TarefaRequest conclui ou dispensa a tarefa; dispensar pede observação.
type TarefaRequest struct {
	Concluir   *bool  `json:"concluir" validate:"required"`
	Observacao string `json:"observacao" validate:"max=1000"`
}

func (h *Handlers) ModeloDaOferta(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	modelo, err := h.svc.ModeloDaOferta(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, modelo)
}

func (h *Handlers) SalvarModelo(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req ModeloRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	modelo := make([]domain.ModeloTarefa, 0, len(req.Tarefas))
	for _, t := range req.Tarefas {
		modelo = append(modelo, domain.ModeloTarefa{Titulo: t.Titulo, EquipeID: t.EquipeID})
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	out, err := h.svc.SalvarModelo(r.Context(), identity, id, modelo)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, out)
}

func (h *Handlers) FazerTarefa(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	tarefa, err := httputil.UUIDParam(r, "tarefaId")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req TarefaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	d, err := h.svc.FazerTarefa(r.Context(), identity, id, tarefa, *req.Concluir, req.Observacao)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, d)
}
