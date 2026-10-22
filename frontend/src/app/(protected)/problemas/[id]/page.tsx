"use client";

import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { type FormEvent, useState } from "react";

import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import type { Solicitacao } from "@/lib/atendimento/atendimento";
import {
  ESTADO_MUDANCA,
  ESTADO_PROBLEMA,
  type Ativo,
  type Mudanca,
  type Problema,
} from "@/lib/servicos/servicos";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

export default function ProblemaPage() {
  const { id } = useParams<{ id: string }>();
  const { can } = usePlataforma();
  const problema = useApiQuery<Problema>(id ? `v1/problemas/${id}` : null);
  const p = problema.data;
  const editavel = can("problemas:manage") && p?.estado !== "encerrado";
  const incidentes = useApiPage<Solicitacao>(
    editavel && can("atendimento:atender")
      ? withQuery("v1/atendimento/fila", { tipo: "incidente", situacao: "todas", page_size: 100 })
      : null,
  );
  const ativos = useApiQuery<Ativo[]>(editavel && can("ativos:manage") ? "v1/ativos" : null);
  const mudancas = useApiQuery<Mudanca[]>(
    id && (can("mudancas:manage") || can("mudancas:aprovar"))
      ? withQuery("v1/mudancas", { alvo_tipo: "problema", alvo_id: id })
      : null,
  );
  const [incidente, setIncidente] = useState("");
  const [ativo, setAtivo] = useState("");
  const { run, pending } = useAction();

  async function agir(fn: () => Promise<unknown>, ok: string) {
    if (await run(fn, ok)) void problema.mutate();
  }

  function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    void agir(
      () =>
        apiClient.put(`v1/problemas/${id}`, {
          titulo: String(fd.get("titulo")).trim(),
          descricao: String(fd.get("descricao")).trim(),
          causa_raiz: String(fd.get("causa_raiz")).trim(),
          contorno: String(fd.get("contorno")).trim(),
          produto_id: p?.produto_id ?? null,
          unidade_id: p?.unidade_id ?? null,
        }),
      "Problema atualizado",
    );
  }

  const ligados = new Set(p?.incidentes.map((i) => i.solicitacao_id));
  const ativosLigados = new Set(p?.ativos.map((a) => a.ativo_id));

  return (
    <div className="flex flex-col gap-6">
      <Link
        href="/problemas"
        className="inline-flex w-fit items-center gap-1 text-sm text-primary hover:underline"
      >
        <ArrowLeft aria-hidden className="h-4 w-4" />
        Voltar para os problemas
      </Link>
      <DataState
        loading={problema.isLoading}
        error={problema.error}
        onRetry={() => void problema.mutate()}
        empty={!p}
      >
        {p && (
          <>
            <PageHeader
              eyebrow={`Problema #${p.numero}`}
              title={p.titulo}
              actions={
                <Badge tone={ESTADO_PROBLEMA[p.estado].tone}>
                  {ESTADO_PROBLEMA[p.estado].label}
                </Badge>
              }
            />

            {p.contorno && (
              <p
                role="status"
                className="rounded-lg border border-warning/40 bg-warning/10 p-3 text-sm"
              >
                <strong>Solução de contorno:</strong> {p.contorno}
              </p>
            )}

            <Card>
              <CardHeader>
                <CardTitle as="h2">Investigação</CardTitle>
              </CardHeader>
              <CardContent className="pb-5">
                {editavel ? (
                  <form key={p.id + p.estado} onSubmit={salvar} className="grid gap-3">
                    <Input
                      id="problema-titulo"
                      name="titulo"
                      label="Título *"
                      required
                      maxLength={160}
                      defaultValue={p.titulo}
                    />
                    <Textarea
                      id="problema-descricao"
                      name="descricao"
                      label="O que se observa"
                      maxLength={4000}
                      rows={2}
                      defaultValue={p.descricao}
                    />
                    <Textarea
                      id="problema-contorno"
                      name="contorno"
                      label="Solução de contorno (exigida no erro conhecido)"
                      maxLength={4000}
                      rows={2}
                      defaultValue={p.contorno}
                    />
                    <Textarea
                      id="problema-causa"
                      name="causa_raiz"
                      label="Causa raiz (exigida para resolver)"
                      maxLength={4000}
                      rows={2}
                      defaultValue={p.causa_raiz}
                    />
                    <div>
                      <Button type="submit" variant="secondary" loading={pending}>
                        Salvar investigação
                      </Button>
                    </div>
                  </form>
                ) : (
                  <dl className="grid gap-3 text-sm">
                    <div>
                      <dt className="text-xs text-muted">O que se observa</dt>
                      <dd className="whitespace-pre-wrap">{p.descricao || "—"}</dd>
                    </div>
                    <div>
                      <dt className="text-xs text-muted">Causa raiz</dt>
                      <dd className="whitespace-pre-wrap">{p.causa_raiz || "—"}</dd>
                    </div>
                  </dl>
                )}
                {/* O servidor só devolve passagens para quem gere. */}
                {p.proximos.length > 0 && (
                  <div
                    className="mt-4 flex flex-wrap gap-2 border-t border-surface-border pt-4"
                    role="group"
                    aria-label="Estado do problema"
                  >
                    {p.proximos.map((para) => (
                      <Button
                        key={para}
                        type="button"
                        variant="secondary"
                        disabled={pending}
                        onClick={() =>
                          void agir(
                            () => apiClient.post(`v1/problemas/${id}/estado`, { estado: para }),
                            "Estado atualizado",
                          )
                        }
                      >
                        {ESTADO_PROBLEMA[para].acao}
                      </Button>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle as="h2">Incidentes</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-3 pb-5">
                {p.incidentes.length === 0 ? (
                  <p className="text-sm text-muted">Nenhum incidente ligado.</p>
                ) : (
                  <ul className="flex flex-col gap-1 text-sm">
                    {p.incidentes.map((i) => (
                      <li
                        key={i.solicitacao_id}
                        className="flex flex-wrap items-center justify-between gap-2"
                      >
                        <Link
                          href={`/atendimento/solicitacoes/${i.solicitacao_id}`}
                          className="text-primary hover:underline"
                        >
                          #{i.numero} · {i.titulo}
                        </Link>
                        {editavel && (
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            disabled={pending}
                            onClick={() =>
                              void agir(
                                () =>
                                  apiClient.delete(
                                    `v1/problemas/${id}/incidentes/${i.solicitacao_id}`,
                                  ),
                                "Incidente desligado",
                              )
                            }
                          >
                            Desligar #{i.numero}
                          </Button>
                        )}
                      </li>
                    ))}
                  </ul>
                )}
                {editavel && incidentes.data && (
                  <div className="flex flex-wrap items-end gap-2">
                    <div className="min-w-64">
                      <Select
                        id="problema-incidente"
                        label="Ligar incidente"
                        value={incidente}
                        onChange={(e) => setIncidente(e.target.value)}
                        placeholder="Escolha um incidente"
                        options={incidentes.data.items
                          .filter((s) => !ligados.has(s.id))
                          .map((s) => ({ value: s.id, label: `#${s.numero} · ${s.titulo}` }))}
                      />
                    </div>
                    <Button
                      type="button"
                      variant="secondary"
                      disabled={!incidente || pending}
                      onClick={() =>
                        void agir(
                          () => apiClient.put(`v1/problemas/${id}/incidentes/${incidente}`),
                          "Incidente ligado",
                        ).then(() => setIncidente(""))
                      }
                    >
                      Ligar
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle as="h2">Ativos afetados</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-3 pb-5">
                {p.ativos.length === 0 ? (
                  <p className="text-sm text-muted">Nenhum ativo ligado.</p>
                ) : (
                  <ul className="flex flex-col gap-1 text-sm">
                    {p.ativos.map((a) => (
                      <li
                        key={a.ativo_id}
                        className="flex flex-wrap items-center justify-between gap-2"
                      >
                        <span>{a.nome || "Ativo indisponível"}</span>
                        {editavel && (
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            disabled={pending}
                            onClick={() =>
                              void agir(
                                () => apiClient.delete(`v1/problemas/${id}/ativos/${a.ativo_id}`),
                                "Ativo desligado",
                              )
                            }
                          >
                            Desligar {a.nome || "ativo"}
                          </Button>
                        )}
                      </li>
                    ))}
                  </ul>
                )}
                {editavel && ativos.data && (
                  <div className="flex flex-wrap items-end gap-2">
                    <div className="min-w-64">
                      <Select
                        id="problema-ativo"
                        label="Ligar ativo"
                        value={ativo}
                        onChange={(e) => setAtivo(e.target.value)}
                        placeholder="Escolha um ativo"
                        options={ativos.data
                          .filter((a) => !ativosLigados.has(a.id))
                          .map((a) => ({ value: a.id, label: a.nome }))}
                      />
                    </div>
                    <Button
                      type="button"
                      variant="secondary"
                      disabled={!ativo || pending}
                      onClick={() =>
                        void agir(
                          () => apiClient.put(`v1/problemas/${id}/ativos/${ativo}`),
                          "Ativo ligado",
                        ).then(() => setAtivo(""))
                      }
                    >
                      Ligar ativo
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>

            {(mudancas.data ?? []).length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle as="h2">Mudanças</CardTitle>
                </CardHeader>
                <CardContent className="pb-5">
                  <ul className="flex flex-col gap-1 text-sm">
                    {(mudancas.data ?? []).map((m) => (
                      <li key={m.id} className="flex flex-wrap items-center justify-between gap-2">
                        <Link href={`/mudancas/${m.id}`} className="text-primary hover:underline">
                          #{m.numero} · {m.titulo}
                        </Link>
                        <Badge tone={ESTADO_MUDANCA[m.estado].tone}>
                          {ESTADO_MUDANCA[m.estado].label}
                        </Badge>
                      </li>
                    ))}
                  </ul>
                </CardContent>
              </Card>
            )}

            <p className="text-xs text-muted">
              Registrado em {fmtDateTime(p.created_at)}
              {p.resolvido_em ? ` · resolvido em ${fmtDateTime(p.resolvido_em)}` : ""}
            </p>
          </>
        )}
      </DataState>
    </div>
  );
}
