package application

import (
	"context"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	apperrors "github.com/yurythx/projeto-nova-v2/internal/domain/errors"
	"github.com/yurythx/projeto-nova-v2/internal/modules/projetos/domain"
	"github.com/yurythx/projeto-nova-v2/internal/platform/audit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/auth"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
	"github.com/yurythx/projeto-nova-v2/internal/platform/notificacoes"
	"github.com/yurythx/projeto-nova-v2/internal/platform/storage"
)

const prefixoAnexos = "projetos/tarefas"

// Arquivos é o armazenamento das fotos e dos documentos da tarefa.
type Arquivos struct {
	Store    storage.Provider
	Bucket   string
	MaxBytes int64
	Expiry   time.Duration
}

// ComArquivos liga o envio de arquivos. Sem ele, o pedido de upload avisa
// que o armazenamento está desligado.
func (s *Service) ComArquivos(a Arquivos) *Service {
	s.arquivos = &a
	return s
}

func (s *Service) prefixoDaTarefa(tarefa uuid.UUID) string {
	return prefixoAnexos + "/" + tarefa.String()
}

func (s *Service) tarefaNoProjeto(ctx context.Context, db database.DBTX, projetoID, tarefaID uuid.UUID) (*domain.Tarefa, *domain.Projeto, error) {
	tarefa, err := s.repo.ObterTarefa(ctx, db, tarefaID)
	if err != nil {
		return nil, nil, err
	}
	if tarefa.ProjetoID != projetoID {
		return nil, nil, domain.ErrTarefaNaoEncontrada
	}
	projeto, err := s.repo.Obter(ctx, db, projetoID)
	if err != nil {
		return nil, nil, err
	}
	return tarefa, projeto, nil
}

// PedirAnexo emite a URL para o navegador enviar a foto ou o documento
// direto ao armazenamento.
func (s *Service) PedirAnexo(ctx context.Context, idt auth.Identity, projetoID, tarefaID uuid.UUID, nome, contentType string) (modkit.UploadTicket, error) {
	if s.arquivos == nil {
		return modkit.UploadTicket{}, apperrors.DependencyUnavailable("armazenamento de arquivos desligado")
	}
	nome = strings.TrimSpace(nome)
	if nome == "" || utf8.RuneCountInString(nome) > 255 {
		return modkit.UploadTicket{}, apperrors.Validation("informe o nome do arquivo")
	}
	err := database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, projeto, err := s.tarefaNoProjeto(ctx, tx, projetoID, tarefaID)
		if err != nil {
			return err
		}
		if !s.podeGerir(idt, projeto.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		return domain.RecusarAlteracao(projeto.Estado)
	})
	if err != nil {
		return modkit.UploadTicket{}, err
	}
	return modkit.NewUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, s.prefixoDaTarefa(tarefaID), nome, contentType, modkit.DocumentTypes, s.arquivos.Expiry)
}

// RegistrarAnexo confere o arquivo enviado e o liga à tarefa.
func (s *Service) RegistrarAnexo(ctx context.Context, idt auth.Identity, projetoID, tarefaID uuid.UUID, nome, objectKey string) (*domain.Anexo, error) {
	if s.arquivos == nil {
		return nil, apperrors.DependencyUnavailable("armazenamento de arquivos desligado")
	}
	nome = strings.TrimSpace(nome)
	if nome == "" || utf8.RuneCountInString(nome) > 255 {
		return nil, apperrors.Validation("informe o nome do arquivo")
	}
	info, err := modkit.ConfirmUpload(ctx, s.arquivos.Store, s.arquivos.Bucket, objectKey, s.prefixoDaTarefa(tarefaID), s.arquivos.MaxBytes, modkit.DocumentTypes)
	if err != nil {
		return nil, err
	}
	var gravado *domain.Anexo
	var enviadas []notificacoes.Enviada
	err = database.WithTx(ctx, s.pool, func(ctx context.Context, tx pgx.Tx) error {
		tarefa, projeto, err := s.tarefaNoProjeto(ctx, tx, projetoID, tarefaID)
		if err != nil {
			return err
		}
		if !s.podeGerir(idt, projeto.UnidadeID) {
			return domain.ErrForaDoEscopo
		}
		if err := domain.RecusarAlteracao(projeto.Estado); err != nil {
			return err
		}
		autorNome := strings.TrimSpace(idt.Name)
		if autorNome == "" {
			autorNome = idt.Username
		}
		var autorID *uuid.UUID
		if idt.UserID != uuid.Nil {
			id := idt.UserID
			autorID = &id
		}
		a := &domain.Anexo{
			ID: uuid.New(), TarefaID: tarefaID, Nome: nome, ObjectKey: objectKey,
			ContentType: info.ContentType, Tamanho: info.Size, AutorID: autorID, AutorNome: autorNome,
			CreatedAt: time.Now().UTC(),
		}
		gravado, err = s.repo.CriarAnexo(ctx, tx, a)
		if err != nil {
			return err
		}
		if err := audit.NewWriter(tx).Record(ctx, audit.Meta(ctx, "projetos.tarefa.anexo", "projeto_tarefa", tarefaID.String(), nil, gravado)); err != nil {
			return err
		}
		destinos := append(idDe(projeto.GerenteID), idDe(tarefa.ResponsavelID)...)
		parte, err := s.registrar(ctx, tx, idt.UserID, destinos, notificacoes.Nova{
			Modulo:   "projetos",
			Titulo:   "Arquivo na tarefa do projeto #" + strconv.FormatInt(projeto.Numero, 10),
			Mensagem: cortar(tarefa.Titulo+" · "+nome, 1000),
			Link:     linkProjeto(projeto.ID),
			Chave:    chaveAviso("tarefa", tarefaID.String(), "anexo"),
		})
		if err != nil {
			return err
		}
		enviadas = append(enviadas, parte...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.entregar(ctx, enviadas)
	return gravado, nil
}

// ListarAnexos devolve os arquivos de quem pode ler a tarefa.
func (s *Service) ListarAnexos(ctx context.Context, idt auth.Identity, projetoID, tarefaID uuid.UUID) ([]domain.Anexo, error) {
	_, projeto, err := s.tarefaNoProjeto(ctx, s.pool, projetoID, tarefaID)
	if err != nil {
		return nil, err
	}
	if !s.podeLer(idt, projeto.UnidadeID) {
		return nil, domain.ErrForaDoEscopo
	}
	return s.repo.ListarAnexos(ctx, s.pool, tarefaID)
}

// URLAnexo devolve o endereço temporário do arquivo.
func (s *Service) URLAnexo(ctx context.Context, idt auth.Identity, projetoID, tarefaID, anexoID uuid.UUID) (string, error) {
	if s.arquivos == nil {
		return "", apperrors.DependencyUnavailable("armazenamento de arquivos desligado")
	}
	_, projeto, err := s.tarefaNoProjeto(ctx, s.pool, projetoID, tarefaID)
	if err != nil {
		return "", err
	}
	if !s.podeLer(idt, projeto.UnidadeID) {
		return "", domain.ErrForaDoEscopo
	}
	anexo, err := s.repo.ObterAnexo(ctx, s.pool, anexoID)
	if err != nil {
		return "", err
	}
	if anexo.TarefaID != tarefaID {
		return "", domain.ErrAnexoNaoEncontrado
	}
	url, err := s.arquivos.Store.PresignedGetURL(ctx, s.arquivos.Bucket, anexo.ObjectKey, s.arquivos.Expiry)
	if err != nil {
		return "", apperrors.DependencyUnavailable("armazenamento indisponível").WithCause(err)
	}
	return url, nil
}
