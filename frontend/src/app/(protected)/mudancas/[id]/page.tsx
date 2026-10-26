"use client";

import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { useIdDaRota, usePainelEmbutido } from "@/components/plataforma/PainelDoQuadro";
import { type FormEvent, type ReactNode, useState } from "react";

import { MudancaForm, type MudancaCorpo } from "@/components/servicos/MudancaForm";
import { RfcDocumento } from "@/components/servicos/RfcDocumento";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import type { Solicitacao } from "@/lib/atendimento/atendimento";
import {
  ESTADO_MUDANCA,
  RISCO,
  TEXTO_DA_PASSAGEM,
  TIPO_MUDANCA,
  type Ativo,
  type EstadoMudanca,
  type Mudanca,
  type Problema,
} from "@/lib/servicos/servicos";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

type TipoVinculo = "problema" | "incidente" | "ativo";

const DECISAO: Record<Mudanca["aprovacoes"][number]["decisao"], string> = {
  aprovada: "Aprovada",
  rejeitada: "Rejeitada",
  automatica: "Pré-aprovada (padrão)",
};

function Campo({ rotulo, children }: { rotulo: string; children: ReactNode }) {
  return (
    <div>
      <dt className="text-xs text-muted">{rotulo}</dt>
      <dd className="whitespace-pre-wrap text-sm">{children}</dd>
    </div>
  );
}

export function MudancaDetalhe() {
  const id = useIdDaRota();
  const embutido = usePainelEmbutido();
  const { can } = usePlataforma();
  const mudanca = useApiQuery<Mudanca>(id ? `v1/mudancas/${id}` : null);
  const m = mudanca.data;
  const gere = can("mudancas:manage");
  const editavel = gere && (m?.estado === "rascunho" || m?.estado === "rejeitada");
  const vincula = gere && m !== undefined && m.estado !== "cancelada";
  const [tipo, setTipo] = useState<TipoVinculo>("problema");
  const [alvo, setAlvo] = useState("");
  const [pedindo, setPedindo] = useState<EstadoMudanca | null>(null);
  const [folha, setFolha] = useState(false);
  const problemas = useApiQuery<Problema[]>(
    vincula && tipo === "problema" && can("problemas:manage") ? "v1/problemas" : null,
  );
  const incidentes = useApiPage<Solicitacao>(
    vincula && tipo === "incidente" && can("atendimento:atender")
      ? withQuery("v1/atendimento/fila", { tipo: "incidente", situacao: "todas", page_size: 100 })
      : null,
  );
  const ativos = useApiQuery<Ativo[]>(
    vincula && tipo === "ativo" && can("ativos:manage") ? "v1/ativos" : null,
  );
  const { run, pending } = useAction();

  async function agir(fn: () => Promise<unknown>, ok: string) {
    const feito = await run(fn, ok);
    if (feito) void mudanca.mutate();
    return feito;
  }

  function salvar(corpo: MudancaCorpo) {
    void agir(
      () => apiClient.put(`v1/mudancas/${id}`, { ...corpo, produto_id: m?.produto_id ?? null }),
      "Mudança atualizada",
    );
  }

  async function passar(para: EstadoMudanca, texto = "", resolverCascata = false) {
    if (
      await agir(
        () =>
          apiClient.post(`v1/mudancas/${id}/estado`, {
            estado: para,
            texto,
            resolver_vinculos: resolverCascata,
          }),
        ESTADO_MUDANCA[para].label,
      )
    ) {
      setPedindo(null);
    }
  }

  function confirmarComTexto(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const resolverCascata = fd.get("resolver_em_cascata") === "on";
    void passar(pedindo!, String(fd.get("texto") ?? "").trim(), resolverCascata);
  }

  const ligados = new Set(m?.vinculos.filter((v) => v.tipo === tipo).map((v) => v.alvo_id));
  const opcoes =
    tipo === "problema"
      ? (problemas.data ?? []).map((p) => ({ value: p.id, label: `#${p.numero} · ${p.titulo}` }))
      : tipo === "incidente"
        ? (incidentes.data?.items ?? []).map((s) => ({
            value: s.id,
            label: `#${s.numero} · ${s.titulo}`,
          }))
        : (ativos.data ?? []).map((a) => ({ value: a.id, label: a.nome }));
  const textoDe = pedindo ? TEXTO_DA_PASSAGEM[pedindo] : undefined;

  return (
    <div className="flex flex-col gap-6">
      {!embutido && (
        <Link
          href="/mudancas"
          className="inline-flex w-fit items-center gap-1 text-sm text-primary hover:underline"
        >
          <ArrowLeft aria-hidden className="h-4 w-4" />
          Voltar para as mudanças
        </Link>
      )}
      <DataState
        loading={mudanca.isLoading}
        error={mudanca.error}
        onRetry={() => void mudanca.mutate()}
        empty={!m}
      >
        {m && (
          <>
            <PageHeader
              eyebrow={`Mudança #${m.numero}`}
              title={m.titulo}
              description={`${TIPO_MUDANCA[m.tipo]} · pedida por ${m.solicitante_nome}`}
              actions={
                <>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={() => setFolha(true)}
                  >
                    Emitir RFC
                  </Button>
                  <Badge tone={RISCO[m.risco].tone}>{RISCO[m.risco].label}</Badge>
                  <Badge tone={ESTADO_MUDANCA[m.estado].tone}>
                    {ESTADO_MUDANCA[m.estado].label}
                  </Badge>
                </>
              }
            />

            {folha && (
              <Card>
                <CardHeader>
                  <CardTitle as="h2">Folha da RFC</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-4 pb-5">
                  <RfcDocumento mudanca={m} />
                  <div className="flex flex-wrap gap-2 print:hidden">
                    <Button type="button" onClick={() => window.print()}>
                      Imprimir ou salvar em PDF
                    </Button>
                    <Button type="button" variant="ghost" onClick={() => setFolha(false)}>
                      Fechar folha
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )}

            <Card>
              <CardHeader>
                <CardTitle as="h2">{editavel ? "Editar" : "Plano"}</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-4 pb-5">
                {editavel ? (
                  <MudancaForm
                    inicial={m}
                    rotulo="Salvar mudança"
                    pending={pending}
                    onSalvar={salvar}
                  />
                ) : (
                  <dl className="grid gap-3 sm:grid-cols-2">
                    <Campo rotulo="Janela planejada">
                      {m.inicio_planejado
                        ? `${fmtDateTime(m.inicio_planejado)} a ${fmtDateTime(m.fim_planejado)}`
                        : "—"}
                    </Campo>
                    <Campo rotulo="Execução">
                      {m.executada_inicio
                        ? `${fmtDateTime(m.executada_inicio)}${m.executada_fim ? ` a ${fmtDateTime(m.executada_fim)}` : ""}`
                        : "—"}
                    </Campo>
                    <Campo rotulo="O que muda">{m.descricao || "—"}</Campo>
                    <Campo rotulo="Por que mudar">{m.justificativa || "—"}</Campo>
                    <Campo rotulo="Plano de reversão">{m.plano_reversao || "—"}</Campo>
                    {m.resultado && <Campo rotulo="Resultado">{m.resultado}</Campo>}
                  </dl>
                )}

                {m.proximos.length > 0 && (
                  <div className="flex flex-col gap-3 border-t border-surface-border pt-4">
                    <div
                      className="flex flex-wrap gap-2"
                      role="group"
                      aria-label="Estado da mudança"
                    >
                      {m.proximos.map((para) => (
                        <Button
                          key={para}
                          type="button"
                          variant={
                            para === "rejeitada" || para === "cancelada" || para === "falhou"
                              ? "danger"
                              : "secondary"
                          }
                          disabled={pending}
                          onClick={() =>
                            TEXTO_DA_PASSAGEM[para] ? setPedindo(para) : void passar(para)
                          }
                        >
                          {ESTADO_MUDANCA[para].acao}
                        </Button>
                      ))}
                    </div>
                    {pedindo && textoDe && (
                      <form onSubmit={confirmarComTexto} className="flex flex-col gap-2">
                        <Textarea
                          id="mudanca-texto"
                          name="texto"
                          label={textoDe.obrigatorio ? `${textoDe.rotulo} *` : textoDe.rotulo}
                          required={textoDe.obrigatorio}
                          maxLength={4000}
                          rows={8}
                          className="min-h-48"
                        />
                        {pedindo === "concluida" &&
                          m.vinculos.some(
                            (v) => v.tipo === "incidente" || v.tipo === "problema",
                          ) && (
                            <label className="flex items-center gap-2 rounded-lg border border-accent/40 bg-accent/5 p-3 text-sm">
                              <input
                                id="mudanca-cascata-chk"
                                type="checkbox"
                                name="resolver_em_cascata"
                                className="h-4 w-4 accent-primary"
                              />
                              <span className="text-foreground">
                                <strong>Concluir vínculos junto:</strong> os{" "}
                                {m.vinculos.filter((v) => v.tipo === "incidente").length} incidentes
                                e os {m.vinculos.filter((v) => v.tipo === "problema").length}{" "}
                                problemas entram na mesma gravação. Se algum não puder ser
                                concluído, a mudança permanece como está.
                              </span>
                            </label>
                          )}
                        <div className="flex flex-wrap gap-2">
                          <Button type="submit" loading={pending}>
                            Confirmar: {ESTADO_MUDANCA[pedindo].acao.toLowerCase()}
                          </Button>
                          <Button type="button" variant="ghost" onClick={() => setPedindo(null)}>
                            Voltar
                          </Button>
                        </div>
                      </form>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle as="h2">Decisões</CardTitle>
              </CardHeader>
              <CardContent className="pb-5">
                {m.aprovacoes.length === 0 ? (
                  <p className="text-sm text-muted">Ainda sem decisão.</p>
                ) : (
                  <ol className="flex flex-col gap-2 text-sm">
                    {m.aprovacoes.map((a) => (
                      <li key={a.id}>
                        <span className="font-medium">{DECISAO[a.decisao]}</span>
                        <span className="text-muted">
                          {" "}
                          · {fmtDateTime(a.em)}
                          {a.aprovador ? ` · ${a.aprovador}` : ""}
                        </span>
                        {a.parecer && <p className="whitespace-pre-wrap">{a.parecer}</p>}
                      </li>
                    ))}
                  </ol>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle as="h2">Ligada a</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-3 pb-5">
                {m.vinculos.length === 0 ? (
                  <p className="text-sm text-muted">Nada ligado ainda.</p>
                ) : (
                  <ul className="flex flex-col gap-1 text-sm">
                    {m.vinculos.map((v) => (
                      <li
                        key={v.tipo + v.alvo_id}
                        className="flex flex-wrap items-center justify-between gap-2"
                      >
                        <Link
                          href={
                            v.tipo === "problema"
                              ? `/problemas/${v.alvo_id}`
                              : v.tipo === "incidente"
                                ? `/atendimento/solicitacoes/${v.alvo_id}`
                                : "/ativos"
                          }
                          className="text-primary hover:underline"
                        >
                          {v.rotulo}
                        </Link>
                        {vincula && (
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            disabled={pending}
                            onClick={() =>
                              void agir(
                                () =>
                                  apiClient.delete(
                                    `v1/mudancas/${id}/vinculos/${v.tipo}/${v.alvo_id}`,
                                  ),
                                "Vínculo desfeito",
                              )
                            }
                          >
                            Desligar {v.rotulo}
                          </Button>
                        )}
                      </li>
                    ))}
                  </ul>
                )}
                {vincula && (
                  <div className="flex flex-wrap items-end gap-2">
                    <Select
                      id="mudanca-vinculo-tipo"
                      label="Tipo"
                      value={tipo}
                      onChange={(e) => {
                        setTipo(e.target.value as TipoVinculo);
                        setAlvo("");
                      }}
                      options={[
                        { value: "problema", label: "Problema" },
                        { value: "incidente", label: "Incidente" },
                        { value: "ativo", label: "Ativo" },
                      ]}
                    />
                    <div className="min-w-64">
                      <Select
                        id="mudanca-vinculo-alvo"
                        label="Item"
                        value={alvo}
                        onChange={(e) => setAlvo(e.target.value)}
                        placeholder="Escolha"
                        options={opcoes.filter((o) => !ligados.has(o.value))}
                      />
                    </div>
                    <Button
                      type="button"
                      variant="secondary"
                      disabled={!alvo || pending}
                      onClick={() =>
                        void agir(
                          () => apiClient.put(`v1/mudancas/${id}/vinculos/${tipo}/${alvo}`),
                          "Vínculo registrado",
                        ).then(() => setAlvo(""))
                      }
                    >
                      Ligar
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>
          </>
        )}
      </DataState>
    </div>
  );
}

export default function MudancaPage() {
  return <MudancaDetalhe />;
}
