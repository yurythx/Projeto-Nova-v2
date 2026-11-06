package application

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/directory/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage"
)

const (
	prefixoAvatar = "avatares"
	maxAvatar     = 2 << 20
)

// Arquivos é o armazenamento da foto do perfil.
type Arquivos struct {
	Store  storage.Provider
	Bucket string
}

// ComArquivos liga o envio da foto. Sem ele, o perfil continua sem imagem.
func (s *Service) ComArquivos(a Arquivos) *Service {
	s.arquivos = &a
	return s
}

type fotos interface {
	LerAvatar(ctx context.Context, db database.DBTX, userID uuid.UUID) (string, error)
	GravarAvatar(ctx context.Context, db database.DBTX, userID uuid.UUID, key string) (string, error)
}

func (s *Service) fotos() (fotos, error) {
	repo, ok := s.repo.(fotos)
	if !ok {
		return nil, domain.ErrNotFound
	}
	return repo, nil
}

func (s *Service) prefixoDaPessoa(userID uuid.UUID) string {
	return prefixoAvatar + "/" + userID.String()
}

// PedirAvatar emite a URL para o navegador enviar a foto direto ao armazenamento.
func (s *Service) PedirAvatar(ctx context.Context, idt auth.Identity, nome, contentType string) (modkit.UploadTicket, error) {
	if idt.UserID == uuid.Nil {
		return modkit.UploadTicket{}, domain.ErrNotFound
	}
	if s.arquivos == nil {
		return modkit.UploadTicket{}, apperrors.DependencyUnavailable("armazenamento de fotos desligado")
	}
	return modkit.NewUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, s.prefixoDaPessoa(idt.UserID), nome, contentType, modkit.ImageTypes, 15*time.Minute)
}

// SalvarAvatar confirma a foto enviada e passa a usá-la no lugar da anterior.
func (s *Service) SalvarAvatar(ctx context.Context, idt auth.Identity, objectKey string) error {
	if s.arquivos == nil || idt.UserID == uuid.Nil {
		return domain.ErrNotFound
	}
	repo, err := s.fotos()
	if err != nil {
		return err
	}
	objectKey = strings.TrimSpace(objectKey)
	info, err := modkit.ConfirmUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, objectKey, s.prefixoDaPessoa(idt.UserID), maxAvatar, modkit.ImageTypes)
	if err != nil {
		return err
	}
	var anterior string
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		anterior, err = repo.GravarAvatar(ctx, tx, idt.UserID, objectKey)
		if err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "directory.avatar.updated", "user", idt.UserID.String(),
			map[string]bool{"foto": anterior != ""}, map[string]any{"foto": true, "bytes": info.Size}))
	})
	if err != nil {
		return err
	}
	if anterior != "" && anterior != objectKey {
		_ = s.arquivos.Store.Delete(ctx, s.arquivos.Bucket, anterior)
	}
	return nil
}

// RemoverAvatar tira a foto. As listas voltam a mostrar as iniciais.
func (s *Service) RemoverAvatar(ctx context.Context, idt auth.Identity) error {
	if idt.UserID == uuid.Nil {
		return domain.ErrNotFound
	}
	repo, err := s.fotos()
	if err != nil {
		return err
	}
	var anterior string
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		var err error
		anterior, err = repo.GravarAvatar(ctx, tx, idt.UserID, "")
		if err != nil {
			return err
		}
		return audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "directory.avatar.updated", "user", idt.UserID.String(),
			map[string]bool{"foto": anterior != ""}, map[string]bool{"foto": false}))
	})
	if err != nil {
		return err
	}
	if anterior != "" && s.arquivos != nil {
		_ = s.arquivos.Store.Delete(ctx, s.arquivos.Bucket, anterior)
	}
	return nil
}

// AbrirAvatar devolve a foto para as listas e os quadros.
func (s *Service) AbrirAvatar(ctx context.Context, userID uuid.UUID) (io.ReadCloser, string, error) {
	if s.arquivos == nil {
		return nil, "", domain.ErrNotFound
	}
	repo, err := s.fotos()
	if err != nil {
		return nil, "", err
	}
	key, err := repo.LerAvatar(ctx, s.pool, userID)
	if err != nil || key == "" {
		if err == nil {
			err = domain.ErrNotFound
		}
		return nil, "", err
	}
	info, err := s.arquivos.Store.Stat(ctx, s.arquivos.Bucket, key)
	if err != nil {
		return nil, "", domain.ErrNotFound
	}
	rc, err := s.arquivos.Store.Get(ctx, s.arquivos.Bucket, key)
	if err != nil {
		return nil, "", err
	}
	tipo := info.ContentType
	if tipo == "" {
		tipo = "application/octet-stream"
	}
	return rc, tipo, nil
}
