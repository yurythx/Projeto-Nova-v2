package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/fazenda/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage"
)

const prefixoPecas = "fazenda/pecas"

// Arquivos é o armazenamento das peças. Sem ele, o envio avisa que está desligado.
type Arquivos struct {
	Store    storage.Provider
	Bucket   string
	MaxBytes int64
	Expiry   time.Duration
}

// ComArquivos liga o PDF de cada peça do Anexo II.
func (s *Service) ComArquivos(a Arquivos) *Service {
	s.arquivos = &a
	return s
}

func (s *Service) links(ctx context.Context, pecas []domain.PecaArquivo) []domain.PecaArquivo {
	if pecas == nil {
		pecas = []domain.PecaArquivo{}
	}
	for i := range pecas {
		chave := pecas[i].Chave
		pecas[i].Chave = ""
		if s.arquivos == nil || chave == "" {
			continue
		}
		url, err := s.arquivos.Store.PresignedGetURL(ctx, s.arquivos.Bucket, chave, s.arquivos.Expiry)
		if err == nil {
			pecas[i].URL = url
		}
	}
	return pecas
}

func (s *Service) apresentar(ctx context.Context, p domain.Processo) (domain.Processo, error) {
	pecas, err := s.repo.ListarPecas(ctx, s.leitor, p.ID)
	if err != nil {
		return domain.Processo{}, err
	}
	p.Pendencia = domain.Pendencia(p, s.agora())
	p = p.ComCiencia().ComAtos(s.agora())
	p.Pecas = s.links(ctx, pecas)
	return p, nil
}

// EmitirCertidao grava o Anexo V com o número e o código deste processo.
func (s *Service) EmitirCertidao(ctx context.Context, quem auth.Identity, id uuid.UUID, dados domain.DadosCertidao) (domain.Processo, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoDecidido, func(p domain.Processo) (domain.Processo, string, error) {
		p, err := p.EmitirAnexoV(dados)
		return p, "", err
	})
}

// AssinarTQC registra a data em que o termo foi assinado.
func (s *Service) AssinarTQC(ctx context.Context, quem auth.Identity, id uuid.UUID, assinado time.Time) (domain.Processo, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoDecidido, func(p domain.Processo) (domain.Processo, string, error) {
		p, err := p.RegistrarTQC(assinado)
		return p, "", err
	})
}

// GuardarSelo anota os critérios que continuam funcionando.
func (s *Service) GuardarSelo(ctx context.Context, quem auth.Identity, id uuid.UUID, criterios int) (domain.Processo, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoDecidido, func(p domain.Processo) (domain.Processo, string, error) {
		p, err := p.RegistrarSelo(criterios)
		return p, "", err
	})
}

// GuardarIsencao anota o último exercício da isenção.
func (s *Service) GuardarIsencao(ctx context.Context, quem auth.Identity, id uuid.UUID, fim int) (domain.Processo, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoDecidido, func(p domain.Processo) (domain.Processo, string, error) {
		p, err := p.RegistrarIsencao(fim)
		return p, "", err
	})
}

// AbrirCassacao conta vinte dias a partir de agora.
func (s *Service) AbrirCassacao(ctx context.Context, quem auth.Identity, id uuid.UUID) (domain.Processo, error) {
	if err := s.daEquipe(quem); err != nil {
		return domain.Processo{}, err
	}
	return s.alterar(ctx, id, EventoDecidido, func(p domain.Processo) (domain.Processo, string, error) {
		p, err := p.RegistrarCassacao(s.agora())
		return p, "", err
	})
}

func (s *Service) prefixoPeca(id uuid.UUID) string {
	return prefixoPecas + "/" + id.String()
}

// PedirPeca autoriza um PDF para um tipo do Anexo II.
func (s *Service) PedirPeca(ctx context.Context, quem auth.Identity, id uuid.UUID, tipo, nome, contentType string) (modkit.UploadTicket, error) {
	if s.arquivos == nil {
		return modkit.UploadTicket{}, apperrors.DependencyUnavailable("armazenamento de peças desligado")
	}
	if err := domain.TipoDePeca(tipo); err != nil {
		return modkit.UploadTicket{}, err
	}
	if _, err := s.Buscar(ctx, quem, id); err != nil {
		return modkit.UploadTicket{}, err
	}
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return modkit.UploadTicket{}, apperrors.Validation("informe o nome do arquivo")
	}
	return modkit.NewUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, s.prefixoPeca(id), nome, contentType, []string{"application/pdf"}, s.arquivos.Expiry)
}

// GuardarPeca confere o PDF enviado e substitui a peça daquele tipo.
func (s *Service) GuardarPeca(ctx context.Context, quem auth.Identity, id uuid.UUID, tipo, objectKey string) (domain.Processo, error) {
	if s.arquivos == nil {
		return domain.Processo{}, apperrors.DependencyUnavailable("armazenamento de peças desligado")
	}
	if err := domain.TipoDePeca(tipo); err != nil {
		return domain.Processo{}, err
	}
	if _, err := s.Buscar(ctx, quem, id); err != nil {
		return domain.Processo{}, err
	}
	if _, err := modkit.ConfirmUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, objectKey, s.prefixoPeca(id), s.arquivos.MaxBytes, []string{"application/pdf"}); err != nil {
		return domain.Processo{}, err
	}
	err := s.transacao(ctx, func(ctx context.Context, tx pgx.Tx) error {
		if err := s.repo.GuardarPeca(ctx, tx, id, tipo, objectKey); err != nil {
			return err
		}
		return s.trilha.Registrar(ctx, tx, "fazenda.peca.anexada", id.String(), map[string]string{"tipo": tipo}, "")
	})
	if err != nil {
		return domain.Processo{}, err
	}
	return s.Buscar(ctx, quem, id)
}
