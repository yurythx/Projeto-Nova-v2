package application

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/protocolos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage"
)

// Arquivos é o MinIO da plataforma. Sem Store, o campo de arquivo continua um texto.
type Arquivos struct {
	Store    storage.Provider
	Bucket   string
	MaxBytes int64
	Expiry   time.Duration
}

// ComArquivos liga o armazenamento. O pedido não depende dele para nascer.
func (s *Service) ComArquivos(a Arquivos) *Service {
	if a.Expiry <= 0 {
		a.Expiry = 15 * time.Minute
	}
	if a.MaxBytes <= 0 {
		a.MaxBytes = 10 << 20
	}
	s.arquivos = a
	return s
}

// PedirArquivo entrega o ticket para o cidadão enviar o arquivo direto ao MinIO.
func (s *Service) PedirArquivo(ctx context.Context, servico uuid.UUID, nome, contentType string) (modkit.UploadTicket, error) {
	if s.arquivos.Store == nil {
		return modkit.UploadTicket{}, apperrors.DependencyUnavailable("armazenamento de arquivos desligado")
	}
	serv, err := s.repo.BuscarServico(ctx, s.db, servico)
	if err != nil {
		return modkit.UploadTicket{}, err
	}
	if !serv.Ativo {
		return modkit.UploadTicket{}, domain.ErrNaoEncontrado
	}
	form, err := s.repo.BuscarFormulario(ctx, s.db, serv.FormularioID)
	if err != nil {
		return modkit.UploadTicket{}, err
	}
	if !temArquivo(form) {
		return modkit.UploadTicket{}, apperrors.Validation("este serviço não pede arquivo")
	}
	return modkit.NewUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, prefixoArquivo(servico), nome, contentType, modkit.DocumentTypes, s.arquivos.Expiry)
}

// LinkArquivo devolve a URL de leitura para quem já pode ver a ficha.
func (s *Service) LinkArquivo(ctx context.Context, pedido uuid.UUID, campo string) (string, error) {
	if s.arquivos.Store == nil {
		return "", apperrors.DependencyUnavailable("armazenamento de arquivos desligado")
	}
	ficha, err := s.BuscarPedido(ctx, pedido)
	if err != nil {
		return "", err
	}
	chave, ok := textoResposta(ficha.Pedido.Respostas, campo)
	if ok && strings.HasPrefix(chave, prefixoArquivo(ficha.Pedido.ServicoID)+"/") {
		return s.assinar(ctx, chave)
	}
	if anexoDaFicha(ficha, campo) {
		return s.assinar(ctx, campo)
	}
	return "", domain.ErrInvalido
}

// PedirArquivoComplemento entrega o ticket da resposta do cidadão. Só vale em complementação e com o CPF do pedido.
func (s *Service) PedirArquivoComplemento(ctx context.Context, numero, cpf, nome, contentType string) (modkit.UploadTicket, error) {
	if s.arquivos.Store == nil {
		return modkit.UploadTicket{}, apperrors.DependencyUnavailable("armazenamento de arquivos desligado")
	}
	p, err := s.pedidoDoCidadao(ctx, numero, cpf)
	if err != nil {
		return modkit.UploadTicket{}, err
	}
	if p.Status != domain.StatusComplementacao {
		return modkit.UploadTicket{}, domain.ErrInvalido
	}
	return modkit.NewUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, prefixoComplemento(p.ID), nome, contentType, modkit.DocumentTypes, s.arquivos.Expiry)
}

func (s *Service) assinar(ctx context.Context, chave string) (string, error) {
	url, err := s.arquivos.Store.PresignedGetURL(ctx, s.arquivos.Bucket, chave, s.arquivos.Expiry)
	if err != nil {
		return "", apperrors.DependencyUnavailable("armazenamento indisponível").WithCause(err)
	}
	return url, nil
}

func (s *Service) confirmarComplemento(ctx context.Context, pedido uuid.UUID, arquivos []string) ([]string, error) {
	if len(arquivos) > 5 {
		return nil, domain.ErrInvalido
	}
	out := make([]string, 0, len(arquivos))
	prefixo := prefixoComplemento(pedido)
	for _, chave := range arquivos {
		chave = strings.TrimSpace(chave)
		if chave == "" {
			continue
		}
		if s.arquivos.Store == nil || !strings.HasPrefix(chave, prefixo+"/") {
			return nil, domain.ErrInvalido
		}
		if _, err := modkit.ConfirmUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, chave, prefixo, s.arquivos.MaxBytes, modkit.DocumentTypes); err != nil {
			return nil, err
		}
		out = append(out, chave)
	}
	return out, nil
}

func (s *Service) confirmarArquivos(ctx context.Context, servico uuid.UUID, form domain.Formulario, respostas map[string]any) error {
	if s.arquivos.Store == nil {
		return nil
	}
	for _, campo := range form.Campos {
		if campo.Tipo != "file" {
			continue
		}
		chave, ok := textoResposta(respostas, campo.ID)
		if !ok {
			continue
		}
		if _, err := modkit.ConfirmUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, chave, prefixoArquivo(servico), s.arquivos.MaxBytes, modkit.DocumentTypes); err != nil {
			return err
		}
	}
	return nil
}

func prefixoArquivo(servico uuid.UUID) string {
	return "protocolos/carta/" + servico.String()
}

func prefixoComplemento(pedido uuid.UUID) string {
	return "protocolos/complemento/" + pedido.String()
}

func anexoDaFicha(ficha Ficha, chave string) bool {
	prefixo := prefixoComplemento(ficha.Pedido.ID) + "/"
	if !strings.HasPrefix(chave, prefixo) {
		return false
	}
	for _, item := range ficha.Historico {
		for _, anexo := range item.Anexos {
			if anexo == chave {
				return true
			}
		}
	}
	return false
}

func temArquivo(form domain.Formulario) bool {
	for _, campo := range form.Campos {
		if campo.Tipo == "file" {
			return true
		}
	}
	return false
}

func textoResposta(respostas map[string]any, campo string) (string, bool) {
	if respostas == nil {
		return "", false
	}
	texto, ok := respostas[campo].(string)
	texto = strings.TrimSpace(texto)
	return texto, ok && texto != ""
}
