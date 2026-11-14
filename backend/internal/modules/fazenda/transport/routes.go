package transport

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// RegisterPublic monta o simulador, a autenticidade e a porta do gov.br.
func RegisterPublic(r chi.Router, h *Handlers) {
	r.Post("/fazenda/simulador", h.Simular)
	r.Get("/fazenda/govbr", h.IniciarGovBR)
	r.Post("/fazenda/govbr/sessao", h.TrocarGovBR)
	r.Get("/fazenda/autenticidade/{codigo}", h.Autenticidade)
}

// RegisterAuthed monta o protocolo. A permissão é da rota; o serviço ainda
// confere se o contribuinte está no próprio processo.
func RegisterAuthed(r chi.Router, h *Handlers, logger *slog.Logger) {
	r.Route("/fazenda/processos", func(r chi.Router) {
		r.Post("/", h.Abrir)
		r.Post("/dua", h.DUA)
		r.Get("/painel", h.Painel)
		r.Post("/alertas", h.Alertar)
		r.Get("/", h.Listar)
		r.Get("/{id}", h.Buscar)
		r.Post("/{id}/consulta", h.Consultar)
		r.With(auth.RequirePermission(logger, auth.PermFazendaAnalisar)).Post("/{id}/exigencia", h.Exigir)
		r.Post("/{id}/resposta", h.Responder)
		r.With(auth.RequirePermission(logger, auth.PermFazendaDecidir)).Post("/{id}/decisao", h.Decidir)
		r.With(auth.RequirePermission(logger, auth.PermFazendaDecidir)).Post("/{id}/certidao", h.Certidao)
		r.With(auth.RequirePermission(logger, auth.PermFazendaDecidir)).Post("/{id}/tqc", h.TQC)
		r.With(auth.RequirePermission(logger, auth.PermFazendaDecidir)).Post("/{id}/selo", h.Selo)
		r.With(auth.RequirePermission(logger, auth.PermFazendaDecidir)).Post("/{id}/isencao", h.Isencao)
		r.With(auth.RequirePermission(logger, auth.PermFazendaDecidir)).Post("/{id}/cassacao", h.Cassacao)
		r.Post("/{id}/pecas/upload", h.PedirPeca)
		r.Post("/{id}/pecas", h.GuardarPeca)
		r.With(auth.RequirePermission(logger, auth.PermFazendaDecidir)).Post("/{id}/recurso", h.Recorrer)
		r.With(auth.RequirePermission(logger, auth.PermFazendaDecidir)).Post("/{id}/reducao", h.Reducao)
		r.With(auth.RequirePermission(logger, auth.PermFazendaAnalisar)).Post("/{id}/parcelamento", h.Parcelar)
		r.With(auth.RequirePermission(logger, auth.PermFazendaAnalisar)).Post("/{id}/baixa", h.Baixar)
		r.With(auth.RequirePermission(logger, auth.PermFazendaAnalisar)).Post("/{id}/execucao", h.Execucao)
		r.With(auth.RequirePermission(logger, auth.PermFazendaAnalisar)).Post("/{id}/aviso", h.Avisar)
	})
}
