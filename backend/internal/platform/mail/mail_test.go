package mail

import (
	"context"
	"testing"
)

func TestEnvioDesligado(t *testing.T) {
	if err := (SMTP{}).Send(context.Background(), Mensagem{Para: "a@b.co", Assunto: "oi", Corpo: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := (SMTP{Host: "localhost"}).Send(context.Background(), Mensagem{}); err != nil {
		t.Fatal(err)
	}
}
