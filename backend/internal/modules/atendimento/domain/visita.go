package domain

import (
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Atendimento externo (fase 5). A visita é um ato do técnico: check-in ao
// chegar e check-out ao sair. A localização é lida só nesses dois
// momentos, nunca em segundo plano.

const (
	VisitaEmAndamento = "em_andamento"
	VisitaConcluida   = "concluida"
	VisitaCancelada   = "cancelada"

	EvidenciaFoto       = "foto"
	EvidenciaAssinatura = "assinatura"

	// BaseLegalLocalizacao é a finalidade gravada em cada visita e
	// mostrada ao técnico no check-in.
	BaseLegalLocalizacao = "LGPD, art. 7º, III, e art. 23 — execução de política pública: comprovar a prestação do serviço na visita. A localização é lida só no check-in e no check-out e apagada após um ano."

	// RetencaoLocalizacao é por quanto tempo as coordenadas ficam.
	RetencaoLocalizacao = 365 * 24 * time.Hour
	// AtrasoMaximo é o quanto uma ação feita sem rede pode chegar atrasada.
	AtrasoMaximo = 7 * 24 * time.Hour
	// Adiantamento tolera o relógio do aparelho um pouco à frente.
	Adiantamento = 5 * time.Minute

	LocalMax     = 300
	RaioMinM     = 10
	RaioMaxM     = 50000
	PrecisaoMaxM = 100000
	raioTerraM   = 6371008.8
)

var (
	ErrVisitaInvalida    = errors.New("atendimento: visita inválida")
	ErrLocalizacao       = errors.New("atendimento: localização inválida")
	ErrCapturaForaPrazo  = errors.New("atendimento: horário da captura fora do aceito")
	ErrVisitaAberta      = errors.New("atendimento: já há visita sua em andamento nesta solicitação")
	ErrVisitaEncerrada   = errors.New("atendimento: a visita já foi encerrada")
	ErrVisitaDeOutro     = errors.New("atendimento: só o técnico da visita a encerra")
	ErrEvidenciaInvalida = errors.New("atendimento: evidência inválida")
)

// Ponto é uma leitura do GPS do aparelho.
type Ponto struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	PrecisaoM float64 `json:"precisao_m"`
}

// Destino é onde a visita deveria acontecer, com o raio aceito.
type Destino struct {
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
	RaioM int     `json:"raio_m"`
}

// Evidencia é uma foto ou a assinatura, já conferida no armazenamento. A
// chave do objeto não sai da API: o acesso é por URL temporária.
type Evidencia struct {
	ID          uuid.UUID  `json:"id"`
	VisitaID    uuid.UUID  `json:"visita_id"`
	Tipo        string     `json:"tipo"`
	ObjectKey   string     `json:"-"`
	ContentType string     `json:"content_type"`
	Tamanho     int64      `json:"tamanho"`
	AutorID     *uuid.UUID `json:"autor_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Visita é o atendimento no local. Inicio, Fim e Destino são dados de
// localização: saem da API só na supervisão de visitas em andamento.
type Visita struct {
	ID                  uuid.UUID  `json:"id"`
	SolicitacaoID       uuid.UUID  `json:"solicitacao_id"`
	TecnicoID           uuid.UUID  `json:"tecnico_id"`
	TecnicoNome         string     `json:"tecnico_nome"`
	Estado              string     `json:"estado"`
	Local               string     `json:"local"`
	BaseLegal           string     `json:"base_legal"`
	Destino             *Destino   `json:"destino,omitempty"`
	InicioEm            time.Time  `json:"inicio_em"`
	Inicio              *Ponto     `json:"inicio,omitempty"`
	DistanciaM          *float64   `json:"distancia_m,omitempty"`
	ForaDaArea          bool       `json:"fora_da_area"`
	FimEm               *time.Time `json:"fim_em,omitempty"`
	Fim                 *Ponto     `json:"fim,omitempty"`
	AssinanteNome       string     `json:"assinante_nome"`
	Observacao          string     `json:"observacao"`
	LocalizacaoExpiraEm time.Time  `json:"localizacao_expira_em"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	Evidencias []Evidencia `json:"evidencias"`
}

// SemCoordenadas é a visita sem os dados de localização: o que a
// auditoria, os eventos e as telas comuns recebem.
func (v Visita) SemCoordenadas() Visita {
	v.Destino, v.Inicio, v.Fim = nil, nil, nil
	return v
}

// valido confere as faixas (NaN falha em toda comparação, logo também).
func (p Ponto) valido() bool {
	return p.Lat >= -90 && p.Lat <= 90 && p.Lng >= -180 && p.Lng <= 180 &&
		p.PrecisaoM >= 0 && p.PrecisaoM <= PrecisaoMaxM
}

// Distancia é a distância em metros sobre a superfície (haversine).
func Distancia(aLat, aLng, bLat, bLng float64) float64 {
	rad := math.Pi / 180
	dLat, dLng := (bLat-aLat)*rad, (bLng-aLng)*rad
	h := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(aLat*rad)*math.Cos(bLat*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * raioTerraM * math.Asin(math.Min(1, math.Sqrt(h)))
}

// instante leva o horário do aparelho à precisão do banco (microssegundo):
// o mesmo horário reenviado pela fila sem rede compara igual ao gravado.
func instante(t time.Time) time.Time {
	return t.UTC().Truncate(time.Microsecond)
}

// capturaAceita confere o horário vindo do aparelho: não antes do mínimo,
// nem muito no futuro, nem mais velho que o atraso máximo.
func capturaAceita(capturada, minimo, agora time.Time) bool {
	return !capturada.Before(minimo) && !capturada.After(agora.Add(Adiantamento)) && !capturada.Before(agora.Add(-AtrasoMaximo))
}

// NovaVisita abre a visita (check-in). Fora do raio do destino não é
// recusado — o GPS erra —, só marcado.
func NovaVisita(sol Solicitacao, id, tecnico uuid.UUID, local string, destino *Destino, inicio Ponto, capturada, agora time.Time) (Visita, error) {
	if sol.Finalizada() {
		return Visita{}, ErrSolicitacaoFinalizada
	}
	local = strings.TrimSpace(local)
	if id == uuid.Nil || tecnico == uuid.Nil || utf8.RuneCountInString(local) > LocalMax {
		return Visita{}, ErrVisitaInvalida
	}
	if !inicio.valido() {
		return Visita{}, ErrLocalizacao
	}
	if destino != nil && (!(Ponto{Lat: destino.Lat, Lng: destino.Lng}).valido() || destino.RaioM < RaioMinM || destino.RaioM > RaioMaxM) {
		return Visita{}, ErrLocalizacao
	}
	if !capturaAceita(capturada, sol.CreatedAt, agora) {
		return Visita{}, ErrCapturaForaPrazo
	}
	v := Visita{
		ID: id, SolicitacaoID: sol.ID, TecnicoID: tecnico, Estado: VisitaEmAndamento, Local: local,
		BaseLegal: BaseLegalLocalizacao, Destino: destino, InicioEm: instante(capturada), Inicio: &inicio,
		LocalizacaoExpiraEm: instante(capturada).Add(RetencaoLocalizacao),
		CreatedAt:           agora.UTC(), UpdatedAt: agora.UTC(), Evidencias: []Evidencia{},
	}
	if destino != nil {
		d := Distancia(inicio.Lat, inicio.Lng, destino.Lat, destino.Lng)
		v.DistanciaM = &d
		v.ForaDaArea = d > float64(destino.RaioM)+inicio.PrecisaoM
	}
	return v, nil
}

// Concluir encerra a visita (check-out). O mesmo check-out reenviado
// (mesmo horário) não é mudança: é a fila do aparelho repetindo.
func (v Visita) Concluir(fim Ponto, capturada, agora time.Time, assinante, observacao string) (Visita, bool, error) {
	if v.Estado == VisitaConcluida && v.FimEm != nil && v.FimEm.Equal(instante(capturada)) {
		return v, false, nil
	}
	if v.Estado != VisitaEmAndamento {
		return Visita{}, false, ErrVisitaEncerrada
	}
	assinante, observacao = strings.TrimSpace(assinante), strings.TrimSpace(observacao)
	if utf8.RuneCountInString(assinante) > NomeMax || utf8.RuneCountInString(observacao) > TextoMax {
		return Visita{}, false, ErrVisitaInvalida
	}
	if !fim.valido() {
		return Visita{}, false, ErrLocalizacao
	}
	if !capturaAceita(capturada, v.InicioEm, agora) {
		return Visita{}, false, ErrCapturaForaPrazo
	}
	fimEm := instante(capturada)
	v.Estado, v.FimEm, v.Fim, v.AssinanteNome, v.Observacao = VisitaConcluida, &fimEm, &fim, assinante, observacao
	// A retenção conta da última leitura.
	v.LocalizacaoExpiraEm = fimEm.Add(RetencaoLocalizacao)
	v.UpdatedAt = agora.UTC()
	return v, true, nil
}

// Cancelar desfaz uma visita aberta por engano.
func (v Visita) Cancelar(agora time.Time) (Visita, bool, error) {
	if v.Estado == VisitaCancelada {
		return v, false, nil
	}
	if v.Estado != VisitaEmAndamento {
		return Visita{}, false, ErrVisitaEncerrada
	}
	fim := agora.UTC()
	v.Estado, v.FimEm, v.UpdatedAt = VisitaCancelada, &fim, fim
	return v, true, nil
}

// TipoEvidenciaValido reporta se o tipo é foto ou assinatura.
func TipoEvidenciaValido(tipo string) bool {
	return tipo == EvidenciaFoto || tipo == EvidenciaAssinatura
}
