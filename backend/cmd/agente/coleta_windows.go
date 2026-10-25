//go:build windows

package main

import (
	"fmt"
	"runtime"
	"strings"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func coletar() maquina {
	m := maquina{Sistema: sistemaCurto(sistemaWindows()), Processadores: runtime.NumCPU(), MemoriaBytes: memoriaWindows()}
	m.Discos = maiores(discosFixos(), 16)
	m.Programas, m.ProgramasTruncado = programasWindows()
	return m
}

func programasWindows() ([]programa, bool) {
	var texto string
	for _, caminho := range []string{
		`SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall`,
		`SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`,
	} {
		texto += lerChave(caminho)
	}
	return lerProgramas(texto, 100)
}

func lerChave(caminho string) string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, caminho, registry.ENUMERATE_SUB_KEYS)
	if err != nil {
		return ""
	}
	defer k.Close()
	nomes, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, nome := range nomes {
		sk, err := registry.OpenKey(k, nome, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		if v, _, err := sk.GetIntegerValue("SystemComponent"); err == nil && v == 1 {
			sk.Close()
			continue
		}
		nome, _, nomeErr := sk.GetStringValue("DisplayName")
		versao, _, _ := sk.GetStringValue("DisplayVersion")
		sk.Close()
		if nomeErr != nil || strings.TrimSpace(nome) == "" {
			continue
		}
		b.WriteString(nome)
		b.WriteByte('\t')
		b.WriteString(versao)
		b.WriteByte('\n')
	}
	return b.String()
}

func sistemaWindows() string {
	v := windows.RtlGetVersion()
	nome := fmt.Sprintf("Windows %d", v.MajorVersion)
	if v.MajorVersion == 10 && v.BuildNumber >= 22000 {
		nome = "Windows 11"
	} else if v.MajorVersion == 10 {
		nome = "Windows 10"
	}
	return fmt.Sprintf("%s %d", nome, v.BuildNumber)
}

type memoryStatusEx struct {
	Length               uint32
	MemoryLoad           uint32
	TotalPhys            uint64
	AvailPhys            uint64
	TotalPageFile        uint64
	AvailPageFile        uint64
	TotalVirtual         uint64
	AvailVirtual         uint64
	AvailExtendedVirtual uint64
}

func memoriaWindows() int64 {
	proc := windows.NewLazySystemDLL("kernel32.dll").NewProc("GlobalMemoryStatusEx")
	var mem memoryStatusEx
	mem.Length = uint32(unsafe.Sizeof(mem))
	r, _, _ := proc.Call(uintptr(unsafe.Pointer(&mem)))
	if r == 0 {
		return 0
	}
	return int64(mem.TotalPhys)
}

func discosFixos() []disco {
	buf := make([]uint16, 254)
	n, err := windows.GetLogicalDriveStrings(uint32(len(buf)), &buf[0])
	if err != nil || n == 0 || int(n) > len(buf) {
		return nil
	}
	var out []disco
	for _, drive := range strings.Split(string(utf16.Decode(buf[:n])), "\x00") {
		drive = strings.TrimSpace(drive)
		if drive == "" {
			continue
		}
		ptr, err := windows.UTF16PtrFromString(drive)
		if err != nil || windows.GetDriveType(ptr) != windows.DRIVE_FIXED {
			continue
		}
		var total uint64
		if windows.GetDiskFreeSpaceEx(ptr, nil, &total, nil) != nil || total == 0 {
			continue
		}
		out = append(out, disco{Nome: strings.TrimRight(drive, `\`), Bytes: int64(total)})
	}
	return out
}
