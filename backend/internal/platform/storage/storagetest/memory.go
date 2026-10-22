// Package storagetest oferece um storage.Provider em memória para testes,
// com falhas injetáveis por operação.
package storagetest

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/yurythx/sismarq/internal/platform/storage"
)

// ErrInjected é o erro devolvido pelas operações marcadas para falhar.
var ErrInjected = errors.New("storagetest: falha injetada")

// Memory guarda objetos em memória. Os campos Fail* fazem a operação
// correspondente falhar (armazenamento fora do ar); FailRead faz a leitura
// do conteúdo falhar depois de aberta.
type Memory struct {
	FailGet, FailRead, FailStat, FailPresign, FailPut, FailDelete bool

	mu      sync.Mutex
	objects map[string]object
}

type object struct {
	data        []byte
	contentType string
}

// New cria um armazenamento vazio.
func New() *Memory { return &Memory{objects: map[string]object{}} }

func (m *Memory) Ping(context.Context) error { return nil }

func (m *Memory) Put(_ context.Context, bucket, name string, r io.Reader, _ int64, ct string) error {
	if m.FailPut {
		return ErrInjected
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.objects[bucket+"/"+name] = object{data: b, contentType: ct}
	m.mu.Unlock()
	return nil
}

func (m *Memory) Get(_ context.Context, bucket, name string) (io.ReadCloser, error) {
	if m.FailGet {
		return nil, ErrInjected
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objects[bucket+"/"+name]
	if !ok {
		return nil, storage.ErrObjectNotFound
	}
	if m.FailRead {
		return io.NopCloser(failingReader{}), nil
	}
	return io.NopCloser(bytes.NewReader(o.data)), nil
}

func (m *Memory) Delete(_ context.Context, bucket, name string) error {
	if m.FailDelete {
		return ErrInjected
	}
	m.mu.Lock()
	delete(m.objects, bucket+"/"+name)
	m.mu.Unlock()
	return nil
}

func (m *Memory) PresignedPutURL(_ context.Context, bucket, name string, _ time.Duration) (string, error) {
	if m.FailPresign {
		return "", ErrInjected
	}
	return "https://minio.test/" + bucket + "/" + name + "?put", nil
}

func (m *Memory) PresignedGetURL(_ context.Context, bucket, name string, _ time.Duration) (string, error) {
	if m.FailPresign {
		return "", ErrInjected
	}
	return "https://minio.test/" + bucket + "/" + name + "?get", nil
}

func (m *Memory) Stat(_ context.Context, bucket, name string) (storage.ObjectInfo, error) {
	if m.FailStat {
		return storage.ObjectInfo{}, ErrInjected
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.objects[bucket+"/"+name]
	if !ok {
		return storage.ObjectInfo{}, storage.ErrObjectNotFound
	}
	return storage.ObjectInfo{Size: int64(len(o.data)), ContentType: o.contentType}, nil
}

// GetRange devolve o trecho [offset, offset+length) do objeto (Range).
func (m *Memory) GetRange(ctx context.Context, bucket, name string, offset, length int64) (io.ReadCloser, error) {
	rc, err := m.Get(ctx, bucket, name)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, err
	}
	fim := min(offset+length, int64(len(b)))
	if offset > fim {
		offset = fim
	}
	return io.NopCloser(bytes.NewReader(b[offset:fim])), nil
}

// Corromper troca o conteúdo de um objeto (teste de integridade: o hash
// gravado deixa de bater).
func (m *Memory) Corromper(bucket, name string, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	o := m.objects[bucket+"/"+name]
	o.data = data
	m.objects[bucket+"/"+name] = o
}

// Has reporta se o objeto existe.
func (m *Memory) Has(bucket, name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.objects[bucket+"/"+name]
	return ok
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, ErrInjected }
