package transport

import (
	"log/slog"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
)

// RegisterPublic monta a carta, a abertura e a consulta pelo número.
func RegisterPublic(r chi.Router, h *Handlers) {
	r.Get("/protocolos/carta", h.Carta)
	r.Get("/protocolos/carta/{id}", h.CartaItem)
	r.Post("/protocolos/carta/{id}/arquivo", h.PedirArquivo)
	// A abertura fica sob /carta. POST /protocolos/pedidos é a coleção
	// autenticada; no chi o prefixo montado engole o POST anônimo.
	r.Post("/protocolos/carta/{id}/pedido", h.AbrirPublico)
	r.Get("/protocolos/consulta/{numero}", h.Consultar)
	r.Post("/protocolos/consulta/{numero}/arquivo", h.PedirArquivoComplemento)
	r.Post("/protocolos/consulta/{numero}/resposta", h.ResponderPublico)
}

// RegisterAuthed monta a gestão e a mesa. A permissão da rota é conferida de novo na unidade.
func RegisterAuthed(r chi.Router, h *Handlers, logger *slog.Logger) {
	gerir := auth.RequirePermission(logger, auth.PermProtocolosGerir)
	tratar := auth.RequirePermission(logger, auth.PermProtocolosTratar)
	abrir := auth.RequirePermission(logger, auth.PermProtocolosAbrir)
	r.With(gerir).Post("/protocolos/formularios", h.SalvarFormulario)
	r.With(gerir).Get("/protocolos/formularios", h.ListarFormularios)
	r.With(gerir).Post("/protocolos/servicos", h.SalvarServico)
	r.With(gerir).Get("/protocolos/servicos", h.ListarServicos)
	r.With(gerir).Get("/protocolos/responsaveis", h.ListarResponsaveis)
	r.With(gerir).Post("/protocolos/responsaveis", h.Responsavel)
	r.With(tratar).Post("/protocolos/faltas", h.Falta)
	r.With(abrir).Post("/protocolos/balcao", h.AbrirBalcao)
	r.Route("/protocolos/pedidos", func(r chi.Router) {
		r.Get("/", h.ListarPedidos)
		r.Get("/{id}", h.BuscarPedido)
		r.Get("/{id}/arquivo", h.LinkArquivo)
		r.With(tratar).Post("/{id}/complemento", h.Complementar)
		r.With(tratar).Post("/{id}/resposta", h.Responder)
		r.With(tratar).Post("/{id}/nota", h.Anotar)
		r.With(tratar).Post("/{id}/editar", h.Editar)
		r.With(tratar).Post("/{id}/avancar", h.Avancar)
		r.With(tratar).Post("/{id}/concluir", h.Concluir)
		r.With(tratar).Post("/{id}/cancelar", h.Cancelar)
		r.Get("/{id}/pessoas", h.Pessoas)
		r.With(tratar).Post("/{id}/aceitar", h.Aceitar)
		r.With(tratar).Post("/{id}/transferir", h.Transferir)
		r.With(tratar).Post("/{id}/encaminhar", h.Encaminhar)
		r.With(tratar).Post("/{id}/encaminhamento/aceitar", h.AceitarEncaminhamento)
		r.With(tratar).Post("/{id}/encaminhamento/recusar", h.RecusarEncaminhamento)
		r.With(gerir).Post("/{id}/aprovar", h.Aprovar)
		r.With(gerir).Post("/{id}/devolver", h.Devolver)
	})
}
