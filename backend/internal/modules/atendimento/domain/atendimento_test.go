package domain

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestOfertaEEquipe(t *testing.T) {
	u := uuid.New()
	p := uuid.New()
	o, err := NovaOferta("  E-mail  ", " não entra ", TipoIncidente, &p, &u, true)
	if err != nil || o.Nome != "E-mail" || !o.Ativa || o.ID == uuid.Nil {
		t.Fatalf("nova: %+v %v", o, err)
	}
	if _, err := NovaOferta(" ", "", TipoRequisicao, nil, nil, true); err != ErrEntradaInvalida {
		t.Fatal(err)
	}
	if _, err := NovaOferta(strings.Repeat("a", NomeMax+1), "", TipoRequisicao, nil, nil, true); err != ErrEntradaInvalida {
		t.Fatal(err)
	}
	if _, err := NovaOferta("Ok", strings.Repeat("á", TextoMax+1), TipoRequisicao, nil, nil, false); err != ErrEntradaInvalida {
		t.Fatal(err)
	}
	edit, err := o.Aplicar("Acesso", "", TipoRequisicao, nil, nil, false)
	if err != nil || edit.Ativa || edit.ProdutoID != nil || edit.Igual(o) {
		t.Fatalf("aplicar: %+v %v", edit, err)
	}
	if _, err := o.Aplicar(" ", "", TipoRequisicao, nil, nil, true); err != ErrEntradaInvalida {
		t.Fatal(err)
	}
	if _, err := NovaOferta("Ok", "", "problema", nil, nil, true); err != ErrTipoInvalido {
		t.Fatal(err)
	}
	if _, err := o.Aplicar("Ok", "", "", nil, nil, true); err != ErrTipoInvalido {
		t.Fatal(err)
	}
	if o.Tipo != TipoIncidente || edit.Tipo != TipoRequisicao {
		t.Fatalf("tipo: %q %q", o.Tipo, edit.Tipo)
	}
	if !o.Igual(o) || !MesmaUnidade(nil, nil) || MesmaUnidade(&u, nil) || !MesmaUnidade(&u, &u) {
		t.Fatal("comparação de unidade")
	}
	if Dona(nil).UnidadeID != nil || Dona(&u).UnidadeID == nil {
		t.Fatal("dona")
	}

	a, b := uuid.New(), uuid.New()
	eq, err := NovaEquipe(" N1 ", &u, []uuid.UUID{b, a, b})
	if err != nil || eq.Nome != "N1" || len(eq.Membros) != 2 || eq.Membros[0].String() > eq.Membros[1].String() {
		t.Fatalf("equipe: %+v %v", eq, err)
	}
	if _, err := NovaEquipe("", nil, nil); err != ErrEntradaInvalida {
		t.Fatal(err)
	}
	if _, err := NovaEquipe("Ok", nil, []uuid.UUID{uuid.Nil}); err != ErrMembroInvalido {
		t.Fatal(err)
	}
	muitos := make([]uuid.UUID, MaxMembros+1)
	for i := range muitos {
		muitos[i] = uuid.New()
	}
	if _, err := NovaEquipe("Ok", nil, muitos); err != ErrEntradaInvalida {
		t.Fatal(err)
	}
	if _, err := eq.Aplicar(" ", nil, nil); err != ErrEntradaInvalida {
		t.Fatal(err)
	}
	if _, err := eq.Aplicar("Ok", nil, []uuid.UUID{uuid.Nil}); err != ErrMembroInvalido {
		t.Fatal(err)
	}
	mesmo, err := eq.Aplicar("N1", &u, []uuid.UUID{a, b})
	if err != nil || !mesmo.Igual(eq) {
		t.Fatalf("mesmo conjunto: %+v %v", mesmo, err)
	}
	outro, err := eq.Aplicar("N2", &u, []uuid.UUID{a})
	if err != nil || outro.Igual(eq) {
		t.Fatal(outro)
	}
	trocado := eq
	trocado.Membros = []uuid.UUID{a, uuid.New()}
	if trocado.Igual(eq) {
		t.Fatal("membro diferente no mesmo tamanho")
	}
	menor := eq
	menor.Membros = []uuid.UUID{a}
	if menor.Igual(eq) {
		t.Fatal("conjunto menor")
	}
	vazia, err := NovaEquipe("Plantão", nil, nil)
	if err != nil || len(vazia.Membros) != 0 {
		t.Fatalf("sem membros: %+v %v", vazia, err)
	}
}
