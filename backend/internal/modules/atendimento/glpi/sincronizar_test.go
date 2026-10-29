package glpi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/domain"
	"github.com/yurythx/projeto-nova-v2/internal/modules/atendimento/infrastructure"
	"github.com/yurythx/projeto-nova-v2/internal/platform/database/dbtest"
)

// A sincronização até a virada (ADR 081): o que mudou no GLPI volta, o que
// não mudou fica, nada se repete, e o desfazer continua valendo.
func TestSincronizarComOGLPI(t *testing.T) {
	ctx := context.Background()
	pool := poolDaCarga(t)
	repo := infrastructure.NewRepository()
	u := dbtest.Unidade(t, pool)
	ana := dbtest.User(t, pool)
	var login string
	if err := pool.QueryRow(ctx, `SELECT username FROM users WHERE id = $1`, ana).Scan(&login); err != nil {
		t.Fatal(err)
	}
	feriado := "Feriado da sincronização " + uuid.NewString()[:6]
	limpar := func() {
		ctx := context.Background()
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_solicitacoes WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_ofertas WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_equipes WHERE unidade_id = $1`, u)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_politicas WHERE unidade_id IS NULL`)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_feriados WHERE unidade_id IS NULL AND nome = $1`, feriado)
		_, _ = pool.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi'`)
	}
	limpar()
	t.Cleanup(limpar)

	origem := expedienteDeTeste(t, login, feriado)
	guardar := func(ch Chamado, acomps []Acompanhamento, sat *Satisfacao) {
		ch.Titulo, ch.Tipo, ch.EntidadeID, ch.Abertura = "Chamado", 1, 1, "2025-03-10 09:00:00"
		origem.guardar(ItemPreparo{Tipo: ItemChamado, ID: ch.ID, Versao: ch.Modificado,
			Dados: dados(t, Pacote{Chamado: ch, Acompanhamentos: acomps, Satisfacao: sat,
				Participantes: []Participante{{PessoaID: 5, Papel: 2}}})})
	}
	primeiro := []Acompanhamento{{ID: 1, Texto: "Vou ver", Em: "2025-03-10 10:00:00", AutorID: 5}}
	guardar(Chamado{ID: 301, Status: 2, Modificado: "2025-03-10 10:00:00"}, primeiro, nil)
	guardar(Chamado{ID: 302, Status: 1, Modificado: "2025-03-10 09:00:00"}, nil, nil)
	mapa := NovoMapa([]LinhaMapa{{Secretaria: "SMS", Categoria: -1, Destino: Destino{Unidade: &u, UnidadeNome: "U", Oferta: "Geral",
		Tipo: "incidente", Prioridade: "media"}}})
	fuso, _ := time.LoadLocation("America/Cuiaba")
	executar := func(gravar, sincronizar bool) Relatorio {
		t.Helper()
		im := &Importador{Fonte: &FontePreparo{Leitor: origem}, Pool: pool, Repo: repo, Gravar: gravar, Fuso: fuso, Mapa: mapa,
			Sincronizar: sincronizar}
		rel, err := im.Executar(ctx)
		if err != nil || len(rel.Erros) > 0 {
			t.Fatalf("execução: %+v %v", rel, err)
		}
		return rel
	}
	if rel := executar(true, false); rel.Importados != 2 {
		t.Fatalf("carga: %+v", rel)
	}
	var id uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT destino FROM atendimento_importacao WHERE origem = 'glpi' AND tipo = 'solicitacao' AND id_origem = '301'`).Scan(&id); err != nil {
		t.Fatal(err)
	}

	// No GLPI, o 301 foi solucionado e fechado, com resposta e avaliação.
	cinco := 5
	guardar(Chamado{ID: 301, Status: 6, Solucao: "2025-03-11 08:00:00", Fechamento: "2025-03-12 08:00:00", Modificado: "2025-03-12 08:00:00"},
		append(primeiro, Acompanhamento{ID: 2, Texto: "Resolvido", Em: "2025-03-11 08:00:00", AutorID: 5}),
		&Satisfacao{Nota: &cinco, Comentario: "Ótimo", Em: "2025-03-12 09:00:00"})

	// Sem a chave, o já importado não se toca; simulando, só conta.
	if rel := executar(true, false); rel.JaImportados != 2 || rel.Atualizados != 0 {
		t.Fatalf("sem sincronizar: %+v", rel)
	}
	if rel := executar(false, true); rel.Atualizados != 1 || rel.JaImportados != 1 {
		t.Fatalf("simulação: %+v", rel)
	}
	if s, _ := repo.GetSolicitacao(ctx, pool, id); s.Estado != domain.EstadoEmAtendimento {
		t.Fatalf("a simulação não grava: %s", s.Estado)
	}

	if rel := executar(true, true); rel.Atualizados != 1 || rel.JaImportados != 1 {
		t.Fatalf("sincronização: %+v", rel)
	}
	conferir := func() {
		t.Helper()
		s, err := repo.GetSolicitacao(ctx, pool, id)
		fechado := time.Date(2025, 3, 12, 12, 0, 0, 0, time.UTC)
		if err != nil || s.Estado != domain.EstadoEncerrado || s.EncerradaEm == nil || !s.EncerradaEm.Equal(fechado) ||
			s.ResponsavelID == nil || *s.ResponsavelID != ana || !s.UpdatedAt.Equal(fechado) {
			t.Fatalf("o 301 fechado: %+v %v", s, err)
		}
		if cs, _ := repo.ListComentarios(ctx, pool, id, true); len(cs) != 2 {
			t.Fatalf("comentários: %+v", cs)
		}
		if ps, _ := repo.ListPassagens(ctx, pool, id); len(ps) != 3 || ps[2].Para != domain.EstadoEncerrado {
			t.Fatalf("passagens: %+v", ps)
		}
		if a, err := repo.GetAvaliacao(ctx, pool, id); err != nil || a.Nota != 5 {
			t.Fatalf("avaliação: %+v %v", a, err)
		}
	}
	conferir()

	// De novo: a versão é a mesma, nada se repete.
	if rel := executar(true, true); rel.Atualizados != 0 || rel.JaImportados != 2 {
		t.Fatalf("de novo: %+v", rel)
	}
	// Importado antes do ADR 081 (sem versão nem acompanhamentos no
	// registro): a primeira sincronização o traz sem repetir nada.
	if _, err := pool.Exec(ctx, `UPDATE atendimento_importacao SET versao = NULL WHERE origem = 'glpi' AND tipo = 'solicitacao'`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM atendimento_importacao WHERE origem = 'glpi' AND tipo = 'acompanhamento'`); err != nil {
		t.Fatal(err)
	}
	if rel := executar(true, true); rel.Atualizados != 2 {
		t.Fatalf("legado: %+v", rel)
	}
	conferir()

	// O que a sincronização trouxe não conta como mexido aqui.
	r, err := Desfazer(ctx, pool, true, nil)
	if err != nil || r.Chamados != 2 || r.Mantidos != 0 {
		t.Fatalf("desfazer depois de sincronizar: %+v %v", r, err)
	}
}
