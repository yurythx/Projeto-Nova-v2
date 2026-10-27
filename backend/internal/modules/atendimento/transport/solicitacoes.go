package transport

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/application"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// maxPagina limita a página da fila e da lista de quem abriu.
const maxPagina = 100

// AberturaRequest é o pedido do servidor.
type AberturaRequest struct {
	OfertaID  uuid.UUID         `json:"oferta_id" validate:"required"`
	Titulo    string            `json:"titulo" validate:"required,max=160"`
	Descricao string            `json:"descricao" validate:"max=4000"`
	Respostas map[string]string `json:"respostas"`
}

// EstadoRequest é a passagem pedida.
type EstadoRequest struct {
	Estado string `json:"estado" validate:"required,max=40"`
}

// ResponsavelRequest é quem assume a solicitação.
type ResponsavelRequest struct {
	ResponsavelID uuid.UUID `json:"responsavel_id" validate:"required"`
}

// TopicosResponse são os tópicos de WebSocket que a fila assina.
type TopicosResponse struct {
	Topicos []string `json:"topicos"`
}

// SegredoVinculoRequest aponta o chamado para um segredo. Nulo remove o vínculo.
type SegredoVinculoRequest struct {
	SegredoID *uuid.UUID `json:"segredo_id"`
}

// ComentarioRequest é a resposta ou, com interno, a nota da equipe.
type ComentarioRequest struct {
	Texto   string `json:"texto" validate:"required,max=4000"`
	Interno bool   `json:"interno"`
}

// filtroFila lê os recortes. situacao=todas inclui encerradas e
// canceladas; situacao=abertas as tira; sem ela vale o padrão da lista.
func filtroFila(r *http.Request, finalizadas bool) application.FiltroFila {
	switch httputil.Query(r, "situacao", 20) {
	case "todas":
		finalizadas = true
	case "abertas":
		finalizadas = false
	}
	return application.FiltroFila{
		Tipo:        httputil.Query(r, "tipo", 20),
		Estado:      httputil.Query(r, "estado", 40),
		Responsavel: httputil.Query(r, "responsavel", 36),
		Finalizadas: finalizadas,
	}
}

func (h *Handlers) Catalogo(w http.ResponseWriter, r *http.Request) {
	lista, err := h.svc.Catalogo(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) Abrir(w http.ResponseWriter, r *http.Request) {
	var req AberturaRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	sol, err := h.svc.Abrir(r.Context(), identity, application.AberturaEntrada{
		OfertaID: req.OfertaID, Titulo: req.Titulo, Descricao: req.Descricao, Respostas: req.Respostas,
	})
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, sol)
}

// ListarMinhas mostra, por padrão, também as encerradas: é o histórico de
// quem abriu.
func (h *Handlers) ListarMinhas(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	p := httputil.Page(r, maxPagina)
	lista, total, err := h.svc.ListarMinhas(r.Context(), identity, filtroFila(r, true), p)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WritePage(w, lista, p, total)
}

// ListarFila mostra, por padrão, só o que ainda está em andamento.
func (h *Handlers) ListarFila(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	p := httputil.Page(r, maxPagina)
	lista, total, err := h.svc.ListarFila(r.Context(), identity, filtroFila(r, false), p)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WritePage(w, lista, p, total)
}

// TopicosDaFila diz quais tópicos a fila assina: a cobertura expandida
// (subunidades, entidade) só o servidor conhece.
func (h *Handlers) TopicosDaFila(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	httputil.WriteOK(w, TopicosResponse{Topicos: h.svc.TopicosDaFila(identity)})
}

func (h *Handlers) ObterSolicitacao(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	d, err := h.svc.Obter(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, d)
}

func (h *Handlers) Transicionar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req EstadoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	d, err := h.svc.Transicionar(r.Context(), identity, id, req.Estado)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, d)
}

func (h *Handlers) Atribuir(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req ResponsavelRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	d, err := h.svc.Atribuir(r.Context(), identity, id, req.ResponsavelID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, d)
}

func (h *Handlers) VincularSegredo(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req SegredoVinculoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	d, err := h.svc.VincularSegredo(r.Context(), identity, id, req.SegredoID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, d)
}

func (h *Handlers) Comentar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req ComentarioRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	c, err := h.svc.Comentar(r.Context(), identity, id, req.Texto, req.Interno)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, c)
}

func (h *Handlers) Responsaveis(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	lista, err := h.svc.Responsaveis(r.Context(), identity, id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

// AvaliacaoRequest é a nota de quem abriu a solicitação.
type AvaliacaoRequest struct {
	Nota int `json:"nota" validate:"required,min=1,max=5"`
	// Facilidade responde "foi fácil conseguir o que precisava?" (opcional).
	Facilidade *int   `json:"facilidade" validate:"omitempty,min=1,max=5"`
	Comentario string `json:"comentario" validate:"max=4000"`
}

func (h *Handlers) Avaliar(w http.ResponseWriter, r *http.Request) {
	id, err := httputil.UUIDParam(r, "id")
	if err != nil {
		h.fail(w, r, err)
		return
	}
	var req AvaliacaoRequest
	if err := httputil.Bind(w, r, &req); err != nil {
		h.fail(w, r, err)
		return
	}
	identity, _ := auth.IdentityFromContext(r.Context())
	a, err := h.svc.AvaliarCom(r.Context(), identity, id, req.Nota, req.Facilidade, req.Comentario)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteCreated(w, a)
}

func (h *Handlers) Serie(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	dias := 90
	if q := r.URL.Query().Get("dias"); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			dias = n
		}
	}
	lista, err := h.svc.Serie(r.Context(), identity, dias)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, lista)
}

func (h *Handlers) Governanca(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	g, err := h.svc.Governanca(r.Context(), identity)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, g)
}
