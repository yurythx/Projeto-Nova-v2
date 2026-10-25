"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { type FormEvent, Suspense, useState } from "react";

import { Congelamentos } from "@/components/servicos/Congelamentos";
import { MudancaForm, type MudancaCorpo } from "@/components/servicos/MudancaForm";
import { COLUNAS_MUDANCA, MudancasKanban, faixaDaMudanca } from "@/components/servicos/MudancasKanban";
import { DataState } from "@/components/plataforma/DataState";
import { ListaFaixas, NomeClicavel } from "@/components/plataforma/Faixas";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
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
  const vista: Vista = params.get("vista") === "lista" ? "lista" : "kanban";
  const kanban = vista === "kanban";

  const gere = can("mudancas:manage");
  const ve = gere || can("mudancas:aprovar");
  const [estado, setEstado] = useState("");
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
    router.replace(v === "lista" ? "/mudancas?vista=lista" : "/mudancas?vista=quadro");
  }

  async function criar(corpo: MudancaCorpo) {
    const criada = await run(
      () => apiClient.post<Mudanca>("v1/mudancas", corpo),
      "Mudança registrada",
    );
    if (criada) router.push(`/mudancas/${criada.data.id}`);
  }

  async function criarCard(nome: string) {
    const ok = await run(
      () =>
        apiClient.post<Mudanca>("v1/mudancas", {
          titulo: nome.slice(0, 160),
          descricao: "",
          justificativa: "",
          tipo: "normal",
          risco: "medio",
          risco_score: null,
          plano_reversao: "",
          inicio_planejado: null,
          fim_planejado: null,
          produto_id: null,
          unidade_id: null,
        }),
      "Mudança registrada",
    );
    if (ok) void mudancas.mutate();
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

  const lista = mudancas.data ?? [];

  function novaMudanca(colunaId: string) {
    if (colunaId !== "rascunho" || !gere) return null;
    return (
      <Button
        type="button"
        size="sm"
        variant="secondary"
        onClick={() => document.getElementById("mudanca-titulo")?.focus()}
      >
        Nova mudança
      </Button>
    );
  }

  const reqTexto = pedindoTexto ? TEXTO_DA_PASSAGEM[pedindoTexto.para] : undefined;

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Assistência"
        title="Mudanças"
        description="A mudança padrão é pré-aprovada. A normal e a emergencial passam pela aprovação de outra pessoa, com a janela e o plano de reversão definidos."
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
        {lista.length === 0 && (
          <p className="text-sm text-muted">Nenhuma mudança neste recorte</p>
        )}
        {kanban ? (
          <MudancasKanban
            itens={lista}
            onMover={mover}
            pending={pending}
            extra={novaMudanca}
            aoCriar={gere ? (_coluna, nome) => void criarCard(nome) : undefined}
            aoRenomear={gere ? (m, nome) => void renomearMudanca(m, nome) : undefined}
          />
        ) : (
          <ListaFaixas
            rotulo="Lista de mudanças"
            memoria="nova-faixas:mudancas-lista"
            aparencia="nova-faixas:mudancas"
            colunas={COLUNAS_MUDANCA}
            itens={lista}
            colunaDe={(m) => faixaDaMudanca(m.estado)}
            idDe={(m) => m.id}
            nomeDe={(m) => m.titulo}
            extra={novaMudanca}
            aoCriar={gere ? (_coluna, nome) => void criarCard(nome) : undefined}
            aoRenomear={gere ? (m, nome) => void renomearMudanca(m, nome) : undefined}
            aoAbrir={(m) => router.push(`/mudancas/${m.id}`)}
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
                  <Link
                    href={`/mudancas/${m.id}`}
                    className="text-xs font-medium text-primary hover:underline"
                  >
                    Editar
                  </Link>
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

      {gere && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Nova mudança</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <MudancaForm
              rotulo="Registrar rascunho"
              pending={pending}
              onSalvar={(c) => void criar(c)}
            />
          </CardContent>
        </Card>
      )}

      {gere ? <Congelamentos /> : null}

      {/* Modal para parecer ou resultado de transição de mudança */}
      {pedindoTexto && reqTexto && (
        <Dialog
          open={true}
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
              rows={3}
              value={textoDecisao}
              onChange={(e) => setTextoDecisao(e.target.value)}
              placeholder="Digite suas observações..."
            />
          </form>
        </Dialog>
      )}
    </div>
  );
}
