package application

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage"
)

const (
	EventoVisitaIniciada   = "atendimento.visita.iniciada"
	EventoVisitaConcluida  = "atendimento.visita.concluida"
	EventoVisitaCancelada  = "atendimento.visita.cancelada"
	EventoEvidencia        = "atendimento.visita.evidencia"
	AcaoLocalizacaoApagada = "atendimento.localizacao.apagada"

	prefixoEvidencias = "atendimento/visitas"
)

// IntervaloRetencao é de quanto em quanto tempo o worker apaga as
// coordenadas vencidas (variável só para os testes encurtarem).
var IntervaloRetencao = time.Hour

// Armazenamento é onde ficam as fotos e as assinaturas.
type Armazenamento struct {
	Store    storage.Provider
	Bucket   string
	MaxBytes int64
	Expiry   time.Duration
}

// WithArmazenamento liga as evidências. Sem ele, pedir upload devolve
// indisponível.
func (s *Service) WithArmazenamento(a Armazenamento) *Service {
	s.arquivos = &a
	return s
}

// CheckinEntrada é o check-in, como o aparelho registrou.
type CheckinEntrada struct {
	ID          uuid.UUID
	Local       string
	Destino     *domain.Destino
	Ponto       domain.Ponto
	CapturadaEm time.Time
}

// CheckoutEntrada é o check-out, como o aparelho registrou.
type CheckoutEntrada struct {
	Ponto                     domain.Ponto
	CapturadaEm               time.Time
	AssinanteNome, Observacao string
}

func payloadVisita(v domain.Visita) map[string]any {
	return map[string]any{"id": v.ID.String(), "solicitacao_id": v.SolicitacaoID.String(), "estado": v.Estado}
}

// registrarVisita grava o evento e a auditoria sem coordenadas: a trilha
// é imutável e a localização precisa poder ser apagada.
func (s *Service) registrarVisita(ctx context.Context, tx pgx.Tx, evento string, v domain.Visita, antes *domain.Visita) error {
	var prev any
	if antes != nil {
		prev = antes.SemCoordenadas()
	}
	return s.registrar(ctx, tx, evento, "visita", v.ID, payloadVisita(v), v.SemCoordenadas(), prev)
}

// IniciarVisita faz o check-in. Repetir o mesmo check-in (mesmo id) é a
// fila do aparelho reenviando: devolve a visita já gravada.
func (s *Service) IniciarVisita(ctx context.Context, identity auth.Identity, solicitacao uuid.UUID, in CheckinEntrada) (domain.Visita, error) {
	var out domain.Visita
	err := s.alterar(ctx, identity, solicitacao, func(ctx context.Context, tx pgx.Tx, sol domain.Solicitacao, ps []string) ([]notificacoes.Enviada, error) {
		if !slices.Contains(ps, domain.PapelTecnico) {
			return nil, domain.ErrForaDoEscopo
		}
		if prev, err := s.repo.GetVisita(ctx, tx, in.ID); err == nil {
			if prev.SolicitacaoID != solicitacao || prev.TecnicoID != identity.UserID {
				return nil, domain.ErrVisitaInvalida
			}
			out = prev
			return nil, nil
		} else if !errors.Is(err, domain.ErrNaoEncontrado) {
			return nil, err
		}
		v, err := domain.NovaVisita(sol, in.ID, identity.UserID, in.Local, in.Destino, in.Ponto, in.CapturadaEm, s.agora())
		if err != nil {
			return nil, err
		}
		if err := s.repo.InsertVisita(ctx, tx, v); err != nil {
			return nil, err
		}
		out = v
		return nil, s.registrarVisita(ctx, tx, EventoVisitaIniciada, v, nil)
	})
	return out.SemCoordenadas(), err
}

// visitaDoTecnico trava a visita e confere que quem age é o técnico dela
// e ainda atende a unidade.
func (s *Service) visitaDoTecnico(ctx context.Context, tx pgx.Tx, identity auth.Identity, id uuid.UUID) (domain.Visita, error) {
	if err := s.repo.TravarVisita(ctx, tx, id); err != nil {
		return domain.Visita{}, err
	}
	v, err := s.repo.GetVisita(ctx, tx, id)
	if err != nil {
		return domain.Visita{}, err
	}
	sol, err := s.repo.GetSolicitacao(ctx, tx, v.SolicitacaoID)
	if err != nil {
		return domain.Visita{}, err
	}
	if !slices.Contains(papeis(identity, sol), domain.PapelTecnico) {
		return domain.Visita{}, domain.ErrNaoEncontrado
	}
	if v.TecnicoID != identity.UserID {
		return domain.Visita{}, domain.ErrVisitaDeOutro
	}
	return v, nil
}

// ConcluirVisita faz o check-out.
func (s *Service) ConcluirVisita(ctx context.Context, identity auth.Identity, id uuid.UUID, in CheckoutEntrada) (domain.Visita, error) {
	var out domain.Visita
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		prev, err := s.visitaDoTecnico(ctx, tx, identity, id)
		if err != nil {
			return err
		}
		next, mudou, err := prev.Concluir(in.Ponto, in.CapturadaEm, s.agora(), in.AssinanteNome, in.Observacao)
		if err != nil || !mudou {
			out = prev
			return err
		}
		if err := s.repo.UpdateVisita(ctx, tx, next); err != nil {
			return err
		}
		out = next
		return s.registrarVisita(ctx, tx, EventoVisitaConcluida, next, &prev)
	})
	return out.SemCoordenadas(), MapError(err)
}

// CancelarVisita desfaz a visita aberta por engano.
func (s *Service) CancelarVisita(ctx context.Context, identity auth.Identity, id uuid.UUID) (domain.Visita, error) {
	var out domain.Visita
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		prev, err := s.visitaDoTecnico(ctx, tx, identity, id)
		if err != nil {
			return err
		}
		next, mudou, err := prev.Cancelar(s.agora())
		if err != nil || !mudou {
			out = prev
			return err
		}
		if err := s.repo.UpdateVisita(ctx, tx, next); err != nil {
			return err
		}
		out = next
		return s.registrarVisita(ctx, tx, EventoVisitaCancelada, next, &prev)
	})
	return out.SemCoordenadas(), MapError(err)
}

func (s *Service) prefixoDa(visita uuid.UUID) string {
	return prefixoEvidencias + "/" + visita.String()
}

// UploadEvidencia emite a URL de envio direto de uma foto ou assinatura.
func (s *Service) UploadEvidencia(ctx context.Context, identity auth.Identity, id uuid.UUID, nome, contentType string) (modkit.UploadTicket, error) {
	if s.arquivos == nil {
		return modkit.UploadTicket{}, apperrors.DependencyUnavailable("armazenamento de evidências desligado")
	}
	if err := s.conferirEvidencia(ctx, identity, id); err != nil {
		return modkit.UploadTicket{}, MapError(err)
	}
	return modkit.NewUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, s.prefixoDa(id), nome, contentType, modkit.ImageTypes, s.arquivos.Expiry)
}

// conferirEvidencia: só o técnico da visita, e não em visita cancelada.
// Concluída aceita — a foto tirada sem rede pode subir depois do check-out.
func (s *Service) conferirEvidencia(ctx context.Context, identity auth.Identity, id uuid.UUID) error {
	return database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		v, err := s.visitaDoTecnico(ctx, tx, identity, id)
		if err != nil {
			return err
		}
		if v.Estado == domain.VisitaCancelada {
			return domain.ErrVisitaEncerrada
		}
		return nil
	})
}

// RegistrarEvidencia confere o arquivo enviado e o junta à visita.
// Reenviar a mesma chave devolve a evidência já gravada.
func (s *Service) RegistrarEvidencia(ctx context.Context, identity auth.Identity, id uuid.UUID, tipo, objectKey string) (domain.Evidencia, error) {
	if s.arquivos == nil {
		return domain.Evidencia{}, apperrors.DependencyUnavailable("armazenamento de evidências desligado")
	}
	if !domain.TipoEvidenciaValido(tipo) {
		return domain.Evidencia{}, MapError(domain.ErrEvidenciaInvalida)
	}
	if err := s.conferirEvidencia(ctx, identity, id); err != nil {
		return domain.Evidencia{}, MapError(err)
	}
	info, err := modkit.ConfirmUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, objectKey, s.prefixoDa(id), s.arquivos.MaxBytes, modkit.ImageTypes)
	if err != nil {
		return domain.Evidencia{}, err
	}
	autor := identity.UserID
	e := domain.Evidencia{ID: uuid.New(), VisitaID: id, Tipo: tipo, ObjectKey: objectKey, ContentType: info.ContentType,
		Tamanho: info.Size, AutorID: &autor, CreatedAt: s.agora().UTC()}
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		gravada, err := s.repo.InsertEvidencia(ctx, tx, e)
		if err != nil {
			return err
		}
		if gravada.ID != e.ID {
			e = gravada
			return nil
		}
		e = gravada
		return s.registrar(ctx, tx, EventoEvidencia, "visita", id, map[string]any{"id": id.String(), "evidencia_id": e.ID.String(), "tipo": tipo}, e, nil)
	})
	return e, MapError(err)
}

// URLEvidencia devolve o endereço temporário da foto ou assinatura para
// quem vê a solicitação.
func (s *Service) URLEvidencia(ctx context.Context, identity auth.Identity, visita, evidencia uuid.UUID) (string, error) {
	if s.arquivos == nil {
		return "", apperrors.DependencyUnavailable("armazenamento de evidências desligado")
	}
	e, err := s.repo.GetEvidencia(ctx, s.pool, evidencia)
	if err != nil {
		return "", MapError(err)
	}
	v, err := s.repo.GetVisita(ctx, s.pool, visita)
	if err != nil || e.VisitaID != v.ID {
		return "", MapError(domain.ErrNaoEncontrado)
	}
	sol, err := s.repo.GetSolicitacao(ctx, s.pool, v.SolicitacaoID)
	if err != nil {
		return "", MapError(err)
	}
	if len(papeis(identity, sol)) == 0 {
		return "", MapError(domain.ErrNaoEncontrado)
	}
	url, err := s.arquivos.Store.PresignedGetURL(ctx, s.arquivos.Bucket, e.ObjectKey, s.arquivos.Expiry)
	if err != nil {
		return "", apperrors.DependencyUnavailable("armazenamento indisponível").WithCause(err)
	}
	return url, nil
}

// VisitasEmCampo é a supervisão: só as visitas em andamento, com a
// posição, nas unidades que identity atende.
func (s *Service) VisitasEmCampo(ctx context.Context, identity auth.Identity) ([]domain.VisitaEmCampo, error) {
	cob := auth.CoverageOf(identity, auth.PermAtendimentoAtender)
	if !cob.All && len(cob.Unidades) == 0 {
		return []domain.VisitaEmCampo{}, nil
	}
	lista, err := s.repo.VisitasEmAndamento(ctx, s.pool, domain.Filtro{Restrito: !cob.All, Unidades: cob.Unidades})
	return lista, MapError(err)
}

// RunRetencao apaga as coordenadas vencidas a cada IntervaloRetencao.
func (s *Service) RunRetencao(ctx context.Context) error {
	t := time.NewTicker(IntervaloRetencao)
	defer t.Stop()
	for {
		if err := s.ApagarLocalizacoesVencidas(ctx); err != nil && ctx.Err() == nil {
			s.logger.WarnContext(ctx, "atendimento: retenção da localização falhou", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
	}
}

// ApagarLocalizacoesVencidas anula as coordenadas com mais de um ano e
// deixa na auditoria quantas visitas foram tocadas (LGPD art. 16).
func (s *Service) ApagarLocalizacoesVencidas(ctx context.Context) error {
	return database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		n, err := s.repo.ApagarLocalizacoes(ctx, tx, s.agora())
		if err != nil || n == 0 {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, AcaoLocalizacaoApagada, "visita", "", nil,
			map[string]any{"visitas": n, "retencao_dias": int(domain.RetencaoLocalizacao.Hours() / 24)}))
	})
}
