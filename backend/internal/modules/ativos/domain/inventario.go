package domain

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	sistemaMax   = 80
	discoNomeMax = 40
	discosMax    = 16
	cpuMax       = 1024
	programaMax  = 80
	versaoMax    = 40
	programasMax = 100
)

// Disco é um volume local da máquina. Não descreve compartilhamento de rede.
type Disco struct {
	Nome  string `json:"nome"`
	Bytes int64  `json:"bytes"`
}

// Programa é um pacote da própria máquina: nome e versão, sem caminho.
type Programa struct {
	Nome   string `json:"nome"`
	Versao string `json:"versao,omitempty"`
}

// Inventario é o que a própria máquina mede de si: sistema, processadores,
// memória, discos locais e programas instalados. Sem amostra de memória,
// o campo fica de fora.
type Inventario struct {
	Sistema           string     `json:"sistema"`
	Processadores     int        `json:"processadores"`
	MemoriaBytes      int64      `json:"memoria_bytes,omitempty"`
	Discos            []Disco    `json:"discos,omitempty"`
	Programas         []Programa `json:"programas,omitempty"`
	ProgramasTruncado bool       `json:"programas_truncado,omitempty"`
}

// ValidarInventario normaliza o relato. nil passa: o agente antigo, ou
// um cadastro manual, não inventa medida.
func ValidarInventario(in *Inventario) (*Inventario, error) {
	if in == nil {
		return nil, nil
	}
	out := Inventario{
		Sistema:       strings.TrimSpace(in.Sistema),
		Processadores: in.Processadores,
		MemoriaBytes:  in.MemoriaBytes,
	}
	if out.Sistema == "" || utf8.RuneCountInString(out.Sistema) > sistemaMax ||
		out.Processadores < 1 || out.Processadores > cpuMax || out.MemoriaBytes < 0 {
		return nil, ErrEntradaInvalida
	}
	if len(in.Discos) > discosMax {
		return nil, ErrEntradaInvalida
	}
	vistos := map[string]bool{}
	for _, d := range in.Discos {
		nome := strings.TrimSpace(d.Nome)
		if nome == "" || utf8.RuneCountInString(nome) > discoNomeMax || d.Bytes < 0 || vistos[nome] {
			return nil, ErrEntradaInvalida
		}
		vistos[nome] = true
		out.Discos = append(out.Discos, Disco{Nome: nome, Bytes: d.Bytes})
	}
	slices.SortFunc(out.Discos, func(a, b Disco) int { return strings.Compare(a.Nome, b.Nome) })
	programas, err := validarProgramas(in.Programas)
	if err != nil {
		return nil, err
	}
	out.Programas = programas
	out.ProgramasTruncado = in.ProgramasTruncado
	return &out, nil
}

func validarProgramas(in []Programa) ([]Programa, error) {
	if len(in) > programasMax {
		return nil, ErrEntradaInvalida
	}
	vistos := map[string]bool{}
	out := make([]Programa, 0, len(in))
	for _, p := range in {
		nome := strings.TrimSpace(p.Nome)
		versao := strings.TrimSpace(p.Versao)
		if nome == "" || strings.ContainsAny(nome, `/\`) || utf8.RuneCountInString(nome) > programaMax ||
			utf8.RuneCountInString(versao) > versaoMax || strings.ContainsAny(versao, `/\`) || vistos[nome+"\t"+versao] {
			return nil, ErrEntradaInvalida
		}
		vistos[nome+"\t"+versao] = true
		out = append(out, Programa{Nome: nome, Versao: versao})
	}
	slices.SortFunc(out, func(a, b Programa) int {
		if a.Nome == b.Nome {
			return strings.Compare(a.Versao, b.Versao)
		}
		return strings.Compare(a.Nome, b.Nome)
	})
	return out, nil
}

// Resumo é a linha estável que a lista de ativos mostra.
func (i Inventario) Resumo() string {
	partes := []string{i.Sistema, fmt.Sprintf("%d processadores", i.Processadores)}
	if i.MemoriaBytes > 0 {
		partes = append(partes, bytesHumanos(i.MemoriaBytes))
	}
	if n := len(i.Discos); n == 1 {
		partes = append(partes, "1 disco")
	} else if n > 1 {
		partes = append(partes, fmt.Sprintf("%d discos", n))
	}
	if n := len(i.Programas); n == 1 {
		partes = append(partes, "1 programa")
	} else if n > 1 {
		texto := fmt.Sprintf("%d programas", n)
		if i.ProgramasTruncado {
			texto += ", lista parcial"
		}
		partes = append(partes, texto)
	}
	return strings.Join(partes, " · ")
}

func bytesHumanos(n int64) string {
	unidades := []string{"B", "KB", "MB", "GB", "TB"}
	v := float64(n)
	i := 0
	for v >= 1024 && i < len(unidades)-1 {
		v /= 1024
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%d B", n)
	}
	s := strconv.FormatFloat(math.Round(v*10)/10, 'f', -1, 64)
	return strings.Replace(s, ".", ",", 1) + " " + unidades[i]
}

func inventarioIgual(a, b *Inventario) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if a.Sistema != b.Sistema || a.Processadores != b.Processadores || a.MemoriaBytes != b.MemoriaBytes ||
		a.ProgramasTruncado != b.ProgramasTruncado || len(a.Discos) != len(b.Discos) || len(a.Programas) != len(b.Programas) {
		return false
	}
	for i := range a.Discos {
		if a.Discos[i] != b.Discos[i] {
			return false
		}
	}
	for i := range a.Programas {
		if a.Programas[i] != b.Programas[i] {
			return false
		}
	}
	return true
}
