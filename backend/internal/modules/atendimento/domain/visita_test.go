package domain

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func solicitacaoAberta(t *testing.T, agora time.Time) Solicitacao {
	t.Helper()
	s, err := NovaSolicitacao(Oferta{ID: uuid.New(), Nome: "Rede", Tipo: TipoIncidente, Ativa: true}, "Sem rede", "", uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	s.CreatedAt = agora.Add(-48 * time.Hour)
	return s
}

// Praça da Sé (SP) e a Catedral, ~100 m.
var (
	se       = Ponto{Lat: -23.5503, Lng: -46.6339, PrecisaoM: 8}
	catedral = Destino{Lat: -23.5512, Lng: -46.6340, RaioM: 50}
)

func TestDistancia(t *testing.T) {
	if d := Distancia(-23.5503, -46.6339, -23.5503, -46.6339); d != 0 {
		t.Fatalf("mesmo ponto: %v", d)
	}
	// Um grau de latitude ≈ 111,2 km.
	if d := Distancia(0, 0, 1, 0); math.Abs(d-111195) > 50 {
		t.Fatalf("um grau: %v", d)
	}
}

func TestNovaVisita(t *testing.T) {
	agora := time.Now()
	sol := solicitacaoAberta(t, agora)
	id, tec := uuid.New(), uuid.New()
	v, err := NovaVisita(sol, id, tec, "  Prédio da Educação ", &catedral, se, agora.Add(-time.Hour), agora)
	if err != nil {
		t.Fatal(err)
	}
	if v.Estado != VisitaEmAndamento || v.Local != "Prédio da Educação" || v.BaseLegal != BaseLegalLocalizacao {
		t.Fatalf("visita: %+v", v)
	}
	if v.DistanciaM == nil || *v.DistanciaM < 90 || *v.DistanciaM > 110 || !v.ForaDaArea {
		t.Fatalf("~100 m de um raio de 50 m (+8 de precisão): fora da área, sem recusar: %v %v", v.DistanciaM, v.ForaDaArea)
	}
	if !v.LocalizacaoExpiraEm.Equal(v.InicioEm.Add(RetencaoLocalizacao)) || len(v.Evidencias) != 0 {
		t.Fatalf("retenção de um ano: %v", v.LocalizacaoExpiraEm)
	}
	perto := catedral
	perto.RaioM = 200
	if v, _ := NovaVisita(sol, uuid.New(), tec, "", &perto, se, agora, agora); v.ForaDaArea {
		t.Fatal("dentro do raio")
	}
	if v, _ := NovaVisita(sol, uuid.New(), tec, "", nil, se, agora, agora); v.DistanciaM != nil || v.ForaDaArea {
		t.Fatal("sem destino, sem distância")
	}
	semCoord := v.SemCoordenadas()
	if semCoord.Inicio != nil || semCoord.Destino != nil || semCoord.Fim != nil || v.Inicio == nil {
		t.Fatal("a versão sem coordenadas não altera a original")
	}

	finalizada := sol
	finalizada.Estado = EstadoEncerrado
	raioCurto, longe := catedral, se
	raioCurto.RaioM = 5
	longe.Lat = 91
	casos := []struct {
		nome      string
		sol       Solicitacao
		id, tec   uuid.UUID
		local     string
		destino   *Destino
		ponto     Ponto
		capturada time.Time
		want      error
	}{
		{"solicitação encerrada", finalizada, id, tec, "", nil, se, agora, ErrSolicitacaoFinalizada},
		{"sem id", sol, uuid.Nil, tec, "", nil, se, agora, ErrVisitaInvalida},
		{"sem técnico", sol, id, uuid.Nil, "", nil, se, agora, ErrVisitaInvalida},
		{"local longo", sol, id, tec, strings.Repeat("a", LocalMax+1), nil, se, agora, ErrVisitaInvalida},
		{"latitude inválida", sol, id, tec, "", nil, longe, agora, ErrLocalizacao},
		{"precisão negativa", sol, id, tec, "", nil, Ponto{Lat: 1, Lng: 1, PrecisaoM: -1}, agora, ErrLocalizacao},
		{"NaN", sol, id, tec, "", nil, Ponto{Lat: math.NaN(), Lng: 1}, agora, ErrLocalizacao},
		{"raio curto", sol, id, tec, "", &raioCurto, se, agora, ErrLocalizacao},
		{"destino inválido", sol, id, tec, "", &Destino{Lat: 0, Lng: 200, RaioM: 50}, se, agora, ErrLocalizacao},
		{"no futuro", sol, id, tec, "", nil, se, agora.Add(time.Hour), ErrCapturaForaPrazo},
		{"velha demais", sol, id, tec, "", nil, se, agora.Add(-8 * 24 * time.Hour), ErrCapturaForaPrazo},
		{"antes da solicitação", sol, id, tec, "", nil, se, sol.CreatedAt.Add(-time.Minute), ErrCapturaForaPrazo},
	}
	for _, c := range casos {
		if _, err := NovaVisita(c.sol, c.id, c.tec, c.local, c.destino, c.ponto, c.capturada, agora); err != c.want {
			t.Errorf("%s: %v", c.nome, err)
		}
	}
	// Relógio do aparelho um pouco adiantado é aceito.
	if _, err := NovaVisita(sol, id, tec, "", nil, se, agora.Add(2*time.Minute), agora); err != nil {
		t.Fatalf("adiantamento tolerado: %v", err)
	}
}

func TestConcluirECancelar(t *testing.T) {
	agora := time.Now()
	v, _ := NovaVisita(solicitacaoAberta(t, agora), uuid.New(), uuid.New(), "", nil, se, agora.Add(-2*time.Hour), agora)
	fim := agora.Add(-time.Hour)
	c, mudou, err := v.Concluir(se, fim, agora, " Maria ", " troquei o cabo ")
	if err != nil || !mudou || c.Estado != VisitaConcluida || c.AssinanteNome != "Maria" || c.Observacao != "troquei o cabo" {
		t.Fatalf("check-out: %+v %v", c, err)
	}
	if !c.LocalizacaoExpiraEm.Equal(instante(fim).Add(RetencaoLocalizacao)) || c.FimEm.Nanosecond()%1000 != 0 {
		t.Fatal("a retenção conta do check-out")
	}
	if again, mudou, err := c.Concluir(se, fim, agora, "", ""); err != nil || mudou || again.AssinanteNome != "Maria" {
		t.Fatalf("o mesmo check-out reenviado não muda: %v %v", mudou, err)
	}
	if _, _, err := c.Concluir(se, fim.Add(time.Minute), agora, "", ""); err != ErrVisitaEncerrada {
		t.Fatalf("outro check-out: %v", err)
	}
	if _, _, err := c.Cancelar(agora); err != ErrVisitaEncerrada {
		t.Fatalf("cancelar concluída: %v", err)
	}
	for nome, caso := range map[string]struct {
		ponto     Ponto
		capturada time.Time
		assinante string
		obs       string
		want      error
	}{
		"assinante longo":  {se, fim, strings.Repeat("a", NomeMax+1), "", ErrVisitaInvalida},
		"observação longa": {se, fim, "", strings.Repeat("a", TextoMax+1), ErrVisitaInvalida},
		"ponto inválido":   {Ponto{Lat: 100}, fim, "", "", ErrLocalizacao},
		"antes do início":  {se, v.InicioEm.Add(-time.Minute), "", "", ErrCapturaForaPrazo},
	} {
		if _, _, err := v.Concluir(caso.ponto, caso.capturada, agora, caso.assinante, caso.obs); err != caso.want {
			t.Errorf("%s: %v", nome, err)
		}
	}
	cancelada, mudou, err := v.Cancelar(agora)
	if err != nil || !mudou || cancelada.Estado != VisitaCancelada || cancelada.FimEm == nil {
		t.Fatalf("cancelar: %+v %v", cancelada, err)
	}
	if _, mudou, err := cancelada.Cancelar(agora); err != nil || mudou {
		t.Fatal("cancelar de novo não muda")
	}
	if !TipoEvidenciaValido(EvidenciaFoto) || !TipoEvidenciaValido(EvidenciaAssinatura) || TipoEvidenciaValido("video") {
		t.Fatal("tipos de evidência")
	}
}
