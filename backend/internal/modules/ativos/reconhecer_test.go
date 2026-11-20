package ativos

import (
	"context"
	"testing"
)

func TestReconhecerSemModulo(t *testing.T) {
	id, ok, err := (*Module)(nil).Reconhecer(context.Background(), "nova_agt_x")
	if ok || err != nil || id.Subject != "" {
		t.Fatalf("%+v %v %v", id, ok, err)
	}
}
