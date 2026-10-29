"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { type FormEvent, Suspense, useState } from "react";

import { Congelamentos } from "@/components/servicos/Congelamentos";
import { MudancaForm, type MudancaCorpo } from "@/components/servicos/MudancaForm";
import {
  COLUNAS_MUDANCA,
  MudancasKanban,
  faixaDaMudanca,
} from "@/components/servicos/MudancasKanban";
import { DataState } from "@/components/plataforma/DataState";
import { ListaFaixas, NomeClicavel, useAparencia } from "@/components/plataforma/Faixas";
import { colunasVisiveis, rotuloQuadro, useQuadro } from "@/components/plataforma/useQuadro";
import { PainelDoQuadro, usePainelNaUrl } from "@/components/plataforma/PainelDoQuadro";
import { MudancaDetalhe } from "@/app/(protected)/mudancas/[id]/page";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { FiltroPorUnidade, filtrarPorUnidade } from "@/components/servicos/FiltroPorUnidade";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Dialog } from "@/components/ui/Dialog";
import { EmptyState } from "@/components/ui/EmptyState";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery, withQuery } from "@/lib/api/swr";
import {
  ESTADO_MUDANCA,
  RISCO,
  TEXTO_DA_PASSAGEM,
  type EstadoMudanca,
  type Mudanca,
} from "@/lib/servicos/servicos";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

type Vista = "lista" | "kanban";

export default function MudancasPage() {
  return (
    <Suspense fallback={null}>
      <MudancasConteudo />
    </Suspense>
  );
}

/** Mudanças: o que vai mudar no que está em operação, quando, com que
 * risco e como desfazer — e quem aprovou. Exibe em lista ou em quadro kanban. */
function MudancasConteudo() {
  const { can } = usePlataforma();
  const router = useRouter();
  const params = useSearchParams();
  const painel = usePainelNaUrl();
  const vista: Vista = params.get("vista") === "lista" ? "lista" : "kanban";
  const kanban = vista === "kanban";

  const gere = can("mudancas:manage");
  const ve = gere || can("mudancas:aprovar");
  const [estado, setEstado] = useState("");
  const [unidadeFiltro, setUnidadeFiltro] = useState("todas");
  const [formAberto, setFormAberto] = useState(false);
  const [quadro, setQuadro] = useState<"rascunho" | "em_avaliacao" | "cancelada">("rascunho");
  const look = useAparencia("nova-faixas:mudancas");
  const catalogo = useQuadro("mudancas", COLUNAS_MUDANCA);
  const controles = { oficial: catalogo.oficial };
  const mudancas = useApiQuery<Mudanca[]>(
    ve
      ? withQuery("v1/mudancas", {
          estado: kanban ? undefined : estado,
        })
      : null,
  );
  const { run, pending } = useAction();

  // Estado para transição que pede texto (parecer ou resultado)
  const [pedindoTexto, setPedindoTexto] = useState<{
    mudanca: Mudanca;
    para: EstadoMudanca;
  } | null>(null);
  const [textoDecisao, setTextoDecisao] = useState("");

  if (!ve) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Assistência" title="Mudanças" />
        <EmptyState
          title="Seu perfil não gere nem aprova mudanças"
          description="Peça o acesso à gestão da sua unidade."
        />
      </div>
    );
  }

  function trocarVista(v: Vista) {
    router.replace(painel.href({ vista: v === "lista" ? "lista" : "quadro" }));
  }

  async function criar(corpo: MudancaCorpo) {
    const alvo = quadro;
    const criada = await run(async () => {
      const res = await apiClient.post<Mudanca>("v1/mudancas", corpo);
      if (alvo !== "rascunho") {
        await apiClient.post(`v1/mudancas/${res.data.id}/estado`, { estado: alvo });
      }
      return res.data;
    }, "Mudança registrada");
    if (criada) {
      setFormAberto(false);
      router.push(`/mudancas/${criada.id}`);
    }
  }

  async function renomearMudanca(m: Mudanca, nome: string) {
    const ok = await run(
      () =>
        apiClient.put(`v1/mudancas/${m.id}`, {
          titulo: nome.slice(0, 160),
          descricao: m.descricao,
          justificativa: m.justificativa,
          tipo: m.tipo,
          risco: m.risco,
          risco_score: m.risco_score ?? null,
          plano_reversao: m.plano_reversao,
          inicio_planejado: m.inicio_planejado ?? null,
          fim_planejado: m.fim_planejado ?? null,
          produto_id: m.produto_id ?? null,
          unidade_id: m.unidade_id ?? null,
        }),
      "Nome atualizado",
    );
    if (ok) void mudancas.mutate();
  }

  async function mover(m: Mudanca, para: EstadoMudanca) {
    if (TEXTO_DA_PASSAGEM[para]) {
      setPedindoTexto({ mudanca: m, para });
      setTextoDecisao("");
      return;
    }
    const ok = await run(
      () => apiClient.post(`v1/mudancas/${m.id}/estado`, { estado: para }),
      `Mudança #${m.numero}: ${ESTADO_MUDANCA[para].label.toLowerCase()}`,
    );
    if (ok) void mudancas.mutate();
  }

  async function confirmarComTexto(e: FormEvent) {
    e.preventDefault();
    if (!pedindoTexto) return;
    const { mudanca, para } = pedindoTexto;
    const ok = await run(
      () =>
        apiClient.post(`v1/mudancas/${mudanca.id}/estado`, {
          estado: para,
          texto: textoDecisao.trim(),
        }),
      `Mudança #${mudanca.numero}: ${ESTADO_MUDANCA[para].label.toLowerCase()}`,
    );
    setPedindoTexto(null);
    if (ok) void mudancas.mutate();
  }

  const todasMudancas = mudancas.data ?? [];
  const lista = filtrarPorUnidade(todasMudancas, unidadeFiltro);
  const colunas = colunasVisiveis(catalogo.colunas, lista, (m) => faixaDaMudanca(m.estado));
  const quadrosMudanca = (
    [
      ["rascunho", "rascunho"],
      ["em_avaliacao", "em_avaliacao"],
      ["cancelada", "fechada"],
    ] as const
  )
    .filter(
      ([, colunaId]) => catalogo.colunas.find((coluna) => coluna.id === colunaId)?.ativa !== false,
    )
    .map(([estadoId, colunaId]) => ({
      value: estadoId,
      label:
        estadoId === "cancelada"
          ? `Cancelada (${rotuloQuadro(catalogo.oficial, catalogo.colunas, look.gravado, colunaId)})`
          : rotuloQuadro(catalogo.oficial, catalogo.colunas, look.gravado, colunaId),
    }));

  const reqTexto = pedindoTexto ? TEXTO_DA_PASSAGEM[pedindoTexto.para] : undefined;

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Assistência"
        title="Mudanças"
        description="A mudança padrão é pré-aprovada. A normal e a emergencial passam pela aprovação de outra pessoa, com a janela e o plano de reversão definidos."
        actions={
          gere ? (
            <>
              <Link href="/mudancas/colunas" className={buttonClass("secondary")}>
                Colunas
              </Link>
              <Button
                type="button"
                onClick={() => {
                  setQuadro("rascunho");
                  setFormAberto(true);
                }}
              >
                Nova mudança
              </Button>
            </>
          ) : undefined
        }
      />

      <SectionTabsInline
        label="Forma de ver as mudanças"
        value={vista}
        onChange={trocarVista}
        tabs={[
          { value: "kanban", label: "Quadro" },
          { value: "lista", label: "Lista" },
        ]}
      />

      <FiltroPorUnidade
        rotulo="Filtrar mudanças por unidade"
        itens={todasMudancas}
        value={unidadeFiltro}
        onChange={setUnidadeFiltro}
      />

      {!kanban && (
        <div className="max-w-xs">
          <Select
            id="mudancas-estado"
            label="Estado"
            value={estado}
            onChange={(e) => setEstado(e.target.value)}
            options={[
              { value: "", label: "Todos" },
              ...(Object.keys(ESTADO_MUDANCA) as EstadoMudanca[]).map((e) => ({
                value: e,
                label: ESTADO_MUDANCA[e].label,
              })),
            ]}
          />
        </div>
      )}

      <DataState
        loading={mudancas.isLoading}
        error={mudancas.error}
        onRetry={() => void mudancas.mutate()}
        empty={false}
      >
        {lista.length === 0 && <p className="text-sm text-muted">Nenhuma mudança neste recorte</p>}
        {kanban ? (
          <MudancasKanban
            itens={lista}
            onMover={mover}
            pending={pending}
            aoRenomear={gere ? (m, nome) => void renomearMudanca(m, nome) : undefined}
            aoAbrir={(m) => painel.abrir(m.id)}
            colunas={colunas}
            controles={controles}
          />
        ) : (
          <ListaFaixas
            rotulo="Lista de mudanças"
            memoria="nova-faixas:mudancas-lista"
            aparencia="nova-faixas:mudancas"
            colunas={colunas}
            {...controles}
            itens={lista}
            colunaDe={(m) => faixaDaMudanca(m.estado)}
            idDe={(m) => m.id}
            nomeDe={(m) => m.titulo}
            aoRenomear={gere ? (m, nome) => void renomearMudanca(m, nome) : undefined}
            aoAbrir={(m) => painel.abrir(m.id)}
            render={(m, vista) => (
              <div className="flex flex-col gap-1 sm:flex-row sm:items-center sm:justify-between">
                <div>
                  <NomeClicavel
                    nome={vista.titulo || m.titulo}
                    rotulo={`#${m.numero} · ${vista.titulo || m.titulo}`}
                    href={`/mudancas/${m.id}`}
                    onSalvar={vista.salvarNome}
                    className="font-medium text-primary hover:underline"
                  />
                  <p className="text-sm text-muted">
                    {m.inicio_planejado
                      ? `Janela ${fmtDateTime(m.inicio_planejado)}`
                      : "Sem janela"}{" "}
                    · pedida por {m.solicitante_nome}
                  </p>
                  <button
                    type="button"
                    className="text-left text-xs font-medium text-primary hover:underline"
                    onClick={() => painel.abrir(m.id)}
                  >
                    Editar
                  </button>
                </div>
                <div className="flex flex-wrap items-center gap-2">
                  <Badge tone={RISCO[m.risco].tone}>{RISCO[m.risco].label}</Badge>
                  <Badge tone={ESTADO_MUDANCA[m.estado].tone}>
                    {ESTADO_MUDANCA[m.estado].label}
                  </Badge>
                </div>
              </div>
            )}
          />
        )}
      </DataState>

      {gere && formAberto && (
        <Dialog
          open
          size="full"
          onClose={() => setFormAberto(false)}
          title="Nova mudança"
          description="A mudança entra no quadro escolhido. Aprovação e execução continuam no fluxo, e o nome da coluna não vira estado."
        >
          <div className="mb-3 grid gap-3">
            <Select
              id="mudanca-quadro"
              label="Quadro"
              value={quadro}
              onChange={(e) => setQuadro(e.target.value as typeof quadro)}
              options={quadrosMudanca}
            />
            {quadro === "em_avaliacao" && (
              <p className="text-xs text-muted">
                Para ir à avaliação, informe a janela (início antes do fim) e, na mudança normal ou
                emergencial, o plano de reversão.
              </p>
            )}
          </div>
          <MudancaForm
            rotulo="Registrar mudança"
            pending={pending}
            onSalvar={(c) => void criar(c)}
          />
        </Dialog>
      )}

      {gere ? <Congelamentos /> : null}

      {/* Modal para parecer ou resultado de transição de mudança */}
      {pedindoTexto && reqTexto && (
        <Dialog
          open={true}
          size="full"
          onClose={() => setPedindoTexto(null)}
          title={`Mudança #${pedindoTexto.mudanca.numero}: ${ESTADO_MUDANCA[pedindoTexto.para].acao}`}
          description={`Informe ${reqTexto.rotulo.toLowerCase()} para confirmar esta passagem de estado:`}
          footer={
            <>
              <Button type="button" variant="ghost" onClick={() => setPedindoTexto(null)}>
                Cancelar
              </Button>
              <Button
                type="submit"
                form="modal-mudanca-texto-form"
                loading={pending}
                disabled={reqTexto.obrigatorio && !textoDecisao.trim()}
              >
                Confirmar
              </Button>
            </>
          }
        >
          <form id="modal-mudanca-texto-form" onSubmit={confirmarComTexto} className="mt-2">
            <Textarea
              id="modal-mudanca-parecer"
              label={reqTexto.obrigatorio ? `${reqTexto.rotulo} *` : reqTexto.rotulo}
              required={reqTexto.obrigatorio}
              rows={12}
              className="min-h-[50vh]"
              value={textoDecisao}
              onChange={(e) => setTextoDecisao(e.target.value)}
              placeholder="Digite suas observações..."
            />
          </form>
        </Dialog>
      )}
      <PainelDoQuadro
        aberto={painel.id !== null}
        id={painel.id}
        titulo="Detalhe da mudança"
        href={painel.id ? `/mudancas/${painel.id}` : "/mudancas"}
        onClose={painel.fechar}
      >
        {painel.id && <MudancaDetalhe />}
      </PainelDoQuadro>
    </div>
  );
}
