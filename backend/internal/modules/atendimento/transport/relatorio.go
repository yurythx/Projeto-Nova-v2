package transport

import (
	"net/http"
	"strconv"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/pkg/httputil"
)

// RelatorioPorSecretaria devolve o relatório do período (?dias=, padrão 90).
func (h *Handlers) RelatorioPorSecretaria(w http.ResponseWriter, r *http.Request) {
	identity, _ := auth.IdentityFromContext(r.Context())
	dias := 0
	if q := r.URL.Query().Get("dias"); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			dias = n
		}
	}
	rel, err := h.svc.RelatorioPorSecretaria(r.Context(), identity, dias)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httputil.WriteOK(w, rel)
}
