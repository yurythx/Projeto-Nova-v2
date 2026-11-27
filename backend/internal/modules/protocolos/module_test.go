package protocolos

import (
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/yurythx/projeto-nova-v2/internal/platform/config"
	"github.com/yurythx/projeto-nova-v2/internal/platform/kernel"
	"github.com/yurythx/projeto-nova-v2/internal/platform/modkit"
)

func TestModulo(t *testing.T) {
	m := New(modkit.Deps{})
	man := m.Manifest()
	if man.Key != Key || man.DefaultEnabled || man.Route != "/protocolos" || len(man.Permissions) != 3 {
		t.Fatalf("%+v", man)
	}
	m.RegisterRoutes(kernel.Routes{Public: chi.NewRouter(), Authed: chi.NewRouter()})
	com := New(modkit.Deps{Config: &config.Config{
		MinIO:  config.MinIOConfig{Bucket: "nova"},
		Upload: config.UploadConfig{MaxFileBytes: 1024, URLExpiry: time.Minute},
	}})
	if com.Manifest().Key != Key {
		t.Fatal("config")
	}
	if w := m.Workers(); len(w) != 1 || w[0].Name != "prazos-de-protocolos" || w[0].Process != kernel.ProcessWorker {
		t.Fatalf("%+v", w)
	}
}
