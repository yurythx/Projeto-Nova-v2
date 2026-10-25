package main

import (
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

type disco struct {
	Nome  string `json:"nome"`
	Bytes int64  `json:"bytes"`
}

type programa struct {
	Nome   string `json:"nome"`
	Versao string `json:"versao,omitempty"`
}

// maquina é só esta máquina: sistema, processadores, memória e discos
// locais. Não inclui rede, compartilhamento nem arquivo de outra pessoa.
type maquina struct {
	Sistema           string
	Processadores     int
	MemoriaBytes      int64
	Discos            []disco
	Programas         []programa
	ProgramasTruncado bool
}

func nomeDoSistema(texto string) string {
	for _, linha := range strings.Split(texto, "\n") {
		linha = strings.TrimSpace(linha)
		if valor, ok := strings.CutPrefix(linha, "PRETTY_NAME="); ok {
			return strings.Trim(valor, `"`)
		}
	}
	return ""
}

func memoriaDe(texto string) int64 {
	for _, linha := range strings.Split(texto, "\n") {
		if !strings.HasPrefix(linha, "MemTotal:") {
			continue
		}
		campos := strings.Fields(linha)
		if len(campos) < 2 {
			return 0
		}
		kb, err := strconv.ParseInt(campos[1], 10, 64)
		if err != nil || kb <= 0 {
			return 0
		}
		return kb * 1024
	}
	return 0
}

func desmontar(s string) string {
	s = strings.ReplaceAll(s, `\040`, " ")
	s = strings.ReplaceAll(s, `\011`, "\t")
	s = strings.ReplaceAll(s, `\012`, "\n")
	s = strings.ReplaceAll(s, `\134`, `\`)
	return s
}

type volume struct{ nome, ponto string }

func volumesLocais(mounts string) []volume {
	tipos := map[string]bool{
		"ext2": true, "ext3": true, "ext4": true, "xfs": true, "btrfs": true,
		"vfat": true, "ntfs": true, "ntfs3": true, "exfat": true, "zfs": true, "f2fs": true,
	}
	vistos := map[string]bool{}
	var out []volume
	for _, linha := range strings.Split(mounts, "\n") {
		campos := strings.Fields(linha)
		if len(campos) < 3 || !tipos[campos[2]] || strings.HasPrefix(campos[0], "//") || vistos[campos[0]] {
			continue
		}
		vistos[campos[0]] = true
		nome := ultimo(desmontar(campos[0]))
		if nome == "" {
			nome = ultimo(desmontar(campos[1]))
		}
		out = append(out, volume{nome: nome, ponto: desmontar(campos[1])})
	}
	return out
}

func ultimo(caminho string) string {
	caminho = strings.TrimRight(caminho, `/\`)
	if i := strings.LastIndexAny(caminho, `/\`); i >= 0 {
		caminho = caminho[i+1:]
	}
	return caminho
}

func maiores(discos []disco, limite int) []disco {
	porNome := map[string]int64{}
	var nomes []string
	for _, d := range discos {
		if d.Nome == "" || d.Bytes < 0 {
			continue
		}
		if atual, ok := porNome[d.Nome]; ok {
			if d.Bytes > atual {
				porNome[d.Nome] = d.Bytes
			}
			continue
		}
		porNome[d.Nome] = d.Bytes
		nomes = append(nomes, d.Nome)
	}
	out := make([]disco, 0, len(nomes))
	for _, nome := range nomes {
		out = append(out, disco{Nome: nome, Bytes: porNome[nome]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Bytes > out[j].Bytes })
	if len(out) > limite {
		out = out[:limite]
	}
	return out
}

func lerProgramas(texto string, limite int) ([]programa, bool) {
	vistos := map[string]bool{}
	var todos []programa
	for _, linha := range strings.Split(texto, "\n") {
		nome, versao, _ := strings.Cut(linha, "\t")
		nome, versao = strings.TrimSpace(nome), strings.TrimSpace(versao)
		if nome == "" || strings.ContainsAny(nome, `/\`) || strings.ContainsAny(versao, `/\`) {
			continue
		}
		if utf8.RuneCountInString(nome) > 80 {
			nome = string([]rune(nome)[:80])
		}
		if utf8.RuneCountInString(versao) > 40 {
			versao = string([]rune(versao)[:40])
		}
		chave := nome + "\t" + versao
		if vistos[chave] {
			continue
		}
		vistos[chave] = true
		todos = append(todos, programa{Nome: nome, Versao: versao})
	}
	sort.Slice(todos, func(i, j int) bool {
		if todos[i].Nome == todos[j].Nome {
			return todos[i].Versao < todos[j].Versao
		}
		return todos[i].Nome < todos[j].Nome
	})
	if len(todos) > limite {
		return todos[:limite], true
	}
	return todos, false
}

func sistemaCurto(nome string) string {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		nome = runtime.GOOS
	}
	if !strings.HasSuffix(nome, " "+runtime.GOARCH) {
		nome += " " + runtime.GOARCH
	}
	if utf8.RuneCountInString(nome) <= 80 {
		return nome
	}
	return string([]rune(nome)[:80])
}
