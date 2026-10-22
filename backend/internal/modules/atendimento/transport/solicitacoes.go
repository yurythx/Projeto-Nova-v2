package transport

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/yurythx/sismarq/internal/modules/atendimento/application"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/pkg/httputil"
)

// maxPagina limita a página da fila e da lista de quem abriu.
const maxPagina = 100

// AberturaRequest é o pedido do servidor.
type AberturaRequest struct {
	OfertaID  uuid.UUID `json:"oferta_id" validate:"required"`
	Titulo    string    `json:"titulo" validate:"required,max=160"`
	Descricao string    `json:"descricao" validate:"max=4000"`
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
		OfertaID: req.OfertaID, Titulo: req.Titulo, Descricao: req.Descricao,
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
