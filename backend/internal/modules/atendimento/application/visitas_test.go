package application_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	apperrors "github.com/yurythx/sismarq/internal/domain/errors"
	"github.com/yurythx/sismarq/internal/modules/atendimento/application"
	"github.com/yurythx/sismarq/internal/modules/atendimento/domain"
	"github.com/yurythx/sismarq/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/sismarq/internal/platform/auth"
	"github.com/yurythx/sismarq/internal/platform/storage/storagetest"
)

type identidade = auth.Identity

var praca = domain.Ponto{Lat: -23.5503, Lng: -46.6339, PrecisaoM: 8}

func armazenamento() (*storagetest.Memory, application.Armazenamento) {
	mem := storagetest.New()
	return mem, application.Armazenamento{Store: mem, Bucket: "evidencias", MaxBytes: 1 << 20, Expiry: time.Minute}
}

func checkin(id uuid.UUID) application.CheckinEntrada {
	return application.CheckinEntrada{
		ID: id, Local: "Escola Municipal", Destino: &domain.Destino{Lat: -23.5512, Lng: -46.634, RaioM: 50},
		Ponto: praca, CapturadaEm: time.Now(),
	}
}

func checkout() application.CheckoutEntrada {
	return application.CheckoutEntrada{Ponto: praca, CapturadaEm: time.Now(), AssinanteNome: "Maria", Observacao: "ok"}
}

// enviar faz o papel do navegador: pede a URL e põe a foto no armazenamento.
func enviar(t *testing.T, svc *application.Service, mem *storagetest.Memory, quem identidade, visita uuid.UUID) string {
	t.Helper()
	ticket, err := svc.UploadEvidencia(context.Background(), quem, visita, "foto.jpg", "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if err := mem.Put(context.Background(), "evidencias", ticket.ObjectKey, bytes.NewReader([]byte("jpeg")), 4, "image/jpeg"); err != nil {
		t.Fatal(err)
	}
	return ticket.ObjectKey
}

func TestVisitaNaPratica(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.emAtendimento(t)
	mem, arm := armazenamento()
	svc := c.com(infrastructure.NewRepository(), eventosFalham{}).WithArmazenamento(arm)

	if _, err := svc.IniciarVisita(ctx, c.servidor, sol.ID, checkin(uuid.New())); !errors.Is(err, domain.ErrForaDoEscopo) {
		t.Fatalf("quem abriu não faz visita: %v", err)
	}
	id := uuid.New()
	v, err := svc.IniciarVisita(ctx, c.tecnico, sol.ID, checkin(id))
	if err != nil || v.Inicio != nil || v.Destino != nil || !v.ForaDaArea || v.DistanciaM == nil {
		t.Fatalf("check-in devolvido sem coordenadas, marcado fora da área: %+v %v", v, err)
	}
	if again, err := svc.IniciarVisita(ctx, c.tecnico, sol.ID, checkin(id)); err != nil || again.ID != id {
		t.Fatalf("o mesmo check-in reenviado: %v", err)
	}
	if _, err := svc.IniciarVisita(ctx, c.tecnico, c.emAtendimento(t).ID, checkin(id)); !errors.Is(err, domain.ErrVisitaInvalida) {
		t.Fatalf("o mesmo id em outra solicitação: %v", err)
	}
	if _, err := svc.IniciarVisita(ctx, c.tecnico, sol.ID, checkin(uuid.New())); !errors.Is(err, domain.ErrVisitaAberta) {
		t.Fatalf("segunda visita aberta: %v", err)
	}
	invalido := checkin(uuid.New())
	invalido.Ponto.Lat = 99
	if _, err := svc.IniciarVisita(ctx, c.tecnico, sol.ID, invalido); !errors.Is(err, domain.ErrLocalizacao) {
		t.Fatalf("localização inválida: %v", err)
	}

	campo, err := svc.VisitasEmCampo(ctx, c.tecnico)
	if err != nil || len(campo) != 1 || campo[0].Inicio == nil || campo[0].Numero != sol.Numero {
		t.Fatalf("a supervisão vê a posição da visita em andamento: %+v %v", campo, err)
	}
	if vazio, err := svc.VisitasEmCampo(ctx, c.servidor); err != nil || len(vazio) != 0 {
		t.Fatalf("sem atender, sem supervisão: %v %v", vazio, err)
	}
	d, err := svc.Obter(ctx, c.servidor, sol.ID)
	if err != nil || len(d.Visitas) != 1 || d.Visitas[0].Inicio != nil {
		t.Fatalf("o detalhe mostra a visita sem coordenadas: %+v %v", d.Visitas, err)
	}

	// Evidências.
	colega := atendente(c.unidade)
	colega.UserID = c.servidor.UserID // outra pessoa que atende a unidade
	if _, err := svc.UploadEvidencia(ctx, colega, id, "f.jpg", "image/jpeg"); !errors.Is(err, domain.ErrVisitaDeOutro) {
		t.Fatalf("só o técnico da visita envia: %v", err)
	}
	if _, err := svc.UploadEvidencia(ctx, atendente(uuid.New()), id, "f.jpg", "image/jpeg"); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("de outra unidade: %v", err)
	}
	chave := enviar(t, svc, mem, c.tecnico, id)
	if _, err := svc.RegistrarEvidencia(ctx, c.tecnico, id, "video", chave); !errors.Is(err, domain.ErrEvidenciaInvalida) {
		t.Fatalf("tipo: %v", err)
	}
	if _, err := svc.RegistrarEvidencia(ctx, colega, id, domain.EvidenciaFoto, chave); !errors.Is(err, domain.ErrVisitaDeOutro) {
		t.Fatalf("registrar de outro técnico: %v", err)
	}
	if _, err := svc.RegistrarEvidencia(ctx, c.tecnico, id, domain.EvidenciaFoto, "atendimento/visitas/"+id.String()+"/nada.jpg"); err == nil {
		t.Fatal("arquivo não enviado")
	}
	e, err := svc.RegistrarEvidencia(ctx, c.tecnico, id, domain.EvidenciaFoto, chave)
	if err != nil {
		t.Fatal(err)
	}
	if again, err := svc.RegistrarEvidencia(ctx, c.tecnico, id, domain.EvidenciaFoto, chave); err != nil || again.ID != e.ID {
		t.Fatalf("a mesma foto reenviada: %v", err)
	}
	for nome, quem := range map[string]identidade{"técnico": c.tecnico, "quem abriu": c.servidor} {
		if url, err := svc.URLEvidencia(ctx, quem, id, e.ID); err != nil || url == "" {
			t.Errorf("%s vê a evidência: %v", nome, err)
		}
	}
	if _, err := svc.URLEvidencia(ctx, atendente(uuid.New()), id, e.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("estranho: %v", err)
	}
	if _, err := svc.URLEvidencia(ctx, c.tecnico, uuid.New(), e.ID); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("visita trocada: %v", err)
	}
	if _, err := svc.URLEvidencia(ctx, c.tecnico, id, uuid.New()); !errors.Is(err, domain.ErrNaoEncontrado) {
		t.Fatalf("evidência ausente: %v", err)
	}
	mem.FailPresign = true
	if _, err := svc.URLEvidencia(ctx, c.tecnico, id, e.ID); err == nil {
		t.Fatal("armazenamento fora")
	}
	mem.FailPresign = false

	// Check-out, reenvio, foto depois do check-out; cancelamento.
	if _, err := svc.ConcluirVisita(ctx, colega, id, checkout()); !errors.Is(err, domain.ErrVisitaDeOutro) {
		t.Fatalf("check-out de outro: %v", err)
	}
	saida := checkout()
	fim, err := svc.ConcluirVisita(ctx, c.tecnico, id, saida)
	if err != nil || fim.Estado != domain.VisitaConcluida || fim.Fim != nil || fim.AssinanteNome != "Maria" {
		t.Fatalf("check-out: %+v %v", fim, err)
	}
	if _, err := svc.ConcluirVisita(ctx, c.tecnico, id, saida); err != nil {
		t.Fatalf("o mesmo check-out reenviado: %v", err)
	}
	if _, err := svc.CancelarVisita(ctx, c.tecnico, id); !errors.Is(err, domain.ErrVisitaEncerrada) {
		t.Fatalf("cancelar concluída: %v", err)
	}
	enviar(t, svc, mem, c.tecnico, id)
	segunda := uuid.New()
	if _, err := svc.IniciarVisita(ctx, c.tecnico, sol.ID, checkin(segunda)); err != nil {
		t.Fatalf("nova visita depois de concluir a anterior: %v", err)
	}
	if v, err := svc.CancelarVisita(ctx, c.tecnico, segunda); err != nil || v.Estado != domain.VisitaCancelada {
		t.Fatalf("cancelar: %+v %v", v, err)
	}
	if _, err := svc.CancelarVisita(ctx, c.tecnico, segunda); err != nil {
		t.Fatalf("cancelar de novo: %v", err)
	}
	if _, err := svc.UploadEvidencia(ctx, c.tecnico, segunda, "f.jpg", "image/jpeg"); !errors.Is(err, domain.ErrVisitaEncerrada) {
		t.Fatalf("foto em visita cancelada: %v", err)
	}

	// Um ano depois: as coordenadas somem, a auditoria registra quantas.
	daquiUmAno := svc.WithRelogio(func() time.Time { return time.Now().Add(domain.RetencaoLocalizacao + 24*time.Hour) })
	if err := daquiUmAno.ApagarLocalizacoesVencidas(ctx); err != nil {
		t.Fatal(err)
	}
	if err := daquiUmAno.ApagarLocalizacoesVencidas(ctx); err != nil {
		t.Fatalf("nada a apagar não é erro: %v", err)
	}
	var lat *float64
	if err := c.pool.QueryRow(ctx, `SELECT inicio_lat FROM atendimento_visitas WHERE id = $1`, id).Scan(&lat); err != nil || lat != nil {
		t.Fatalf("coordenada apagada: %v %v", lat, err)
	}

	// Direitos do titular: o técnico exporta as visitas; a eliminação
	// apaga as coordenadas e mantém o registro.
	dados, err := svc.ExportPersonalData(ctx, c.pool, c.tecnico.UserID)
	if err != nil || dados == nil {
		t.Fatalf("exportação do técnico: %v %v", dados, err)
	}
	if err := svc.ErasePersonalData(ctx, c.pool, c.tecnico.UserID); err != nil {
		t.Fatal(err)
	}
	if nada, err := svc.ExportPersonalData(ctx, c.pool, uuid.New()); err != nil || nada != nil {
		t.Fatalf("quem nada tem aqui: %v %v", nada, err)
	}
	if _, err := c.com(falhando("DadosDoTitular"), eventosFalham{}).ExportPersonalData(ctx, c.pool, c.tecnico.UserID); err == nil {
		t.Fatal("exportação com o banco fora")
	}

	// Sem armazenamento ligado.
	sem := c.com(infrastructure.NewRepository(), eventosFalham{})
	if _, err := sem.UploadEvidencia(ctx, c.tecnico, id, "f.jpg", "image/jpeg"); !indisponivel(err) {
		t.Fatalf("upload sem armazenamento: %v", err)
	}
	if _, err := sem.RegistrarEvidencia(ctx, c.tecnico, id, domain.EvidenciaFoto, chave); !indisponivel(err) {
		t.Fatalf("registro sem armazenamento: %v", err)
	}
	if _, err := sem.URLEvidencia(ctx, c.tecnico, id, e.ID); !indisponivel(err) {
		t.Fatalf("url sem armazenamento: %v", err)
	}
}

func indisponivel(err error) bool {
	var app *apperrors.Error
	return errors.As(err, &app) && app.Status == 503
}

func TestVisitaCaminhosDeErro(t *testing.T) {
	ctx := context.Background()
	c := novoCenario(t)
	sol := c.emAtendimento(t)
	mem, arm := armazenamento()
	com := func(metodos ...string) *application.Service {
		return c.com(falhando(metodos...), eventosFalham{}).WithArmazenamento(arm)
	}
	comEvento := func(tipo string) *application.Service {
		return c.com(falhando(), eventosFalham{tipo}).WithArmazenamento(arm)
	}
	base := c.com(infrastructure.NewRepository(), eventosFalham{}).WithArmazenamento(arm)

	for nome, svc := range map[string]*application.Service{
		"leitura": com("GetVisita"), "gravação": com("InsertVisita"), "evento": comEvento(application.EventoVisitaIniciada),
	} {
		if _, err := svc.IniciarVisita(ctx, c.tecnico, sol.ID, checkin(uuid.New())); err == nil {
			t.Errorf("check-in com falha em %s", nome)
		}
	}
	id := uuid.New()
	if _, err := base.IniciarVisita(ctx, c.tecnico, sol.ID, checkin(id)); err != nil {
		t.Fatal(err)
	}
	for nome, svc := range map[string]*application.Service{
		"trava": com("TravarVisita"), "leitura": com("GetVisita"), "solicitação": com("GetSolicitacao"),
		"gravação": com("UpdateVisita"), "evento": comEvento(application.EventoVisitaConcluida),
	} {
		if _, err := svc.ConcluirVisita(ctx, c.tecnico, id, checkout()); err == nil {
			t.Errorf("check-out com falha em %s", nome)
		}
	}
	for nome, svc := range map[string]*application.Service{
		"trava": com("TravarVisita"), "gravação": com("UpdateVisita"), "evento": comEvento(application.EventoVisitaCancelada),
	} {
		if _, err := svc.CancelarVisita(ctx, c.tecnico, id); err == nil {
			t.Errorf("cancelamento com falha em %s", nome)
		}
	}
	chave := enviar(t, base, mem, c.tecnico, id)
	for nome, svc := range map[string]*application.Service{
		"gravação": com("InsertEvidencia"), "evento": comEvento(application.EventoEvidencia),
	} {
		if _, err := svc.RegistrarEvidencia(ctx, c.tecnico, id, domain.EvidenciaFoto, chave); err == nil {
			t.Errorf("evidência com falha em %s", nome)
		}
	}
	e, err := base.RegistrarEvidencia(ctx, c.tecnico, id, domain.EvidenciaFoto, chave)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := com("GetSolicitacao").URLEvidencia(ctx, c.tecnico, id, e.ID); err == nil {
		t.Error("url com a solicitação ilegível")
	}
	if _, err := com("VisitasEmAndamento").VisitasEmCampo(ctx, c.tecnico); err == nil {
		t.Error("supervisão")
	}
	if _, err := com("ListVisitas").Obter(ctx, c.tecnico, sol.ID); err == nil {
		t.Error("detalhe sem as visitas")
	}
	if err := com("ApagarLocalizacoes").ApagarLocalizacoesVencidas(ctx); err == nil {
		t.Error("retenção")
	}

	antes := application.IntervaloRetencao
	application.IntervaloRetencao = 5 * time.Millisecond
	t.Cleanup(func() { application.IntervaloRetencao = antes })
	curto, pare := context.WithTimeout(ctx, 30*time.Millisecond)
	defer pare()
	if err := com("ApagarLocalizacoes").WithLogger(silencio).RunRetencao(curto); err != nil {
		t.Fatalf("o laço só termina com o contexto: %v", err)
	}
}
