//go:build linux

package main

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/sys/unix"
)

func coletar() maquina {
	m := maquina{Sistema: sistemaCurto(""), Processadores: runtime.NumCPU()}
	if b, err := os.ReadFile("/etc/os-release"); err == nil {
		if nome := nomeDoSistema(string(b)); nome != "" {
			m.Sistema = sistemaCurto(nome)
		}
	}
	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		m.MemoriaBytes = memoriaDe(string(b))
	}
	if b, err := os.ReadFile("/proc/mounts"); err == nil {
		for _, v := range volumesLocais(string(b)) {
			if v.nome == "" {
				continue
			}
			bytes := bytesDoPonto(v.ponto)
			if bytes <= 0 {
				continue
			}
			m.Discos = append(m.Discos, disco{Nome: v.nome, Bytes: bytes})
		}
	}
	m.Discos = maiores(m.Discos, 16)
	m.Programas, m.ProgramasTruncado = programasLinux()
	return m
}

func programasLinux() ([]programa, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "dpkg-query", "-W", "-f", "${Package}\t${Version}\n").Output()
	if err != nil {
		out, err = exec.CommandContext(ctx, "rpm", "-qa", "--qf", "%{NAME}\t%{VERSION}\n").Output()
		if err != nil {
			return nil, false
		}
	}
	return lerProgramas(string(out), 100)
}

func bytesDoPonto(ponto string) int64 {
	var st unix.Statfs_t
	if unix.Statfs(ponto, &st) != nil || st.Blocks == 0 || st.Frsize <= 0 {
		return 0
	}
	return int64(st.Frsize) * int64(st.Blocks)
}
