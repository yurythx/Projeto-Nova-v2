//go:build !linux && !windows

package main

import "runtime"

func coletar() maquina {
	return maquina{Sistema: sistemaCurto(""), Processadores: runtime.NumCPU()}
}
