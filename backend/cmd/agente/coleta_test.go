package main

import "testing"

func TestLeituraDaPropriaMaquina(t *testing.T) {
	if nomeDoSistema("NAME=Debian\nPRETTY_NAME=\"Debian GNU/Linux 12\"\n") != "Debian GNU/Linux 12" {
		t.Fatal("pretty name")
	}
	if memoriaDe("MemTotal:       1048576 kB\n") != 1048576*1024 {
		t.Fatal("memória")
	}
	if memoriaDe("MemTotal:       no kB\n") != 0 {
		t.Fatal("memória ilegível fica sem amostra")
	}
	vols := volumesLocais("/dev/sda1 / ext4 rw 0 0\n//servidor/pasta /mnt/rede cifs rw 0 0\ntmpfs /run tmpfs rw 0 0\n/dev/sda1 /boot ext4 rw 0 0\n")
	if len(vols) != 1 || vols[0].nome != "sda1" || vols[0].ponto != "/" {
		t.Fatalf("%#v", vols)
	}
	comEspaco := volumesLocais(`/dev/sdb1 /dados\040locais xfs rw 0 0` + "\n")
	if len(comEspaco) != 1 || comEspaco[0].ponto != "/dados locais" {
		t.Fatalf("%#v", comEspaco)
	}
	cortados := maiores([]disco{{Nome: "a", Bytes: 1}, {Nome: "b", Bytes: 3}, {Nome: "a", Bytes: 2}, {Nome: "", Bytes: 9}}, 1)
	if len(cortados) != 1 || cortados[0].Nome != "b" {
		t.Fatalf("%#v", cortados)
	}
	if sistemaCurto("") == "" {
		t.Fatal("sistema")
	}
	lista, truncado := lerProgramas("zsh\t1\nbash\t5\nC:\\Windows\\notepad.exe\t1\nbash\t5\n", 1)
	if !truncado || len(lista) != 1 || lista[0].Nome != "bash" {
		t.Fatalf("%v %#v", truncado, lista)
	}
}
