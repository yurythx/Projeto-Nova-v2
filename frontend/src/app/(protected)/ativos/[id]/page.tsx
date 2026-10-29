"use client";

import { ArrowLeft, Trash2 } from "lucide-react";
import Link from "next/link";
import { type FormEvent, useState } from "react";

import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useIdDaRota, usePainelEmbutido } from "@/components/plataforma/PainelDoQuadro";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import {
  rotuloEstado,
  rotuloTipo,
  tomEstado,
  type Solicitacao,
} from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import { ESTADO_PROJETO, type Projeto } from "@/lib/projetos/projetos";
import {
  ESTADO_ATIVO,
  ESTADO_MUDANCA,
  ESTADO_PROBLEMA,
  TIPO_ATIVO,
  type Ativo,
  type InventarioMaquina,
  type Mudanca,
  type Problema,
} from "@/lib/servicos/servicos";

interface RelacaoAtivo {
  id: string;
  origem_id: string;
  destino_id: string;
  destino_nome: string;
  tipo: string;
  criada_em: string;
}

const ROTULO_RELACAO: Record<string, string> = {
  depende: "Depende de",
  composto: "Compõe",
  conectado: "Conectado a",
};

function textoBytes(n: number) {
  const unidades = ["B", "KB", "MB", "GB", "TB"];
  let valor = n;
  let i = 0;
  while (valor >= 1024 && i < unidades.length - 1) {
    valor /= 1024;
    i += 1;
  }
  const texto = valor.toLocaleString("pt-BR", { maximumFractionDigits: 1 });
  return `${texto} ${unidades[i]}`;
}

export function InventarioDaMaquina({ inventario }: { inventario: InventarioMaquina }) {
  return (
    <dl className="mt-2 grid gap-1 text-sm sm:grid-cols-2">
      <div>
        <dt className="text-muted">Sistema</dt>
        <dd>{inventario.sistema}</dd>
      </div>
      <div>
        <dt className="text-muted">Processadores</dt>
        <dd>{inventario.processadores}</dd>
      </div>
      {inventario.memoria_bytes ? (
        <div>
          <dt className="text-muted">Memória</dt>
          <dd>{textoBytes(inventario.memoria_bytes)}</dd>
        </div>
      ) : null}
      {(inventario.discos ?? []).length > 0 ? (
        <div>
          <dt className="text-muted">Discos locais</dt>
          <dd>{inventario.discos!.map((d) => `${d.nome} ${textoBytes(d.bytes)}`).join(" · ")}</dd>
        </div>
      ) : null}
      {(inventario.programas ?? []).length > 0 ? (
        <div className="sm:col-span-2">
          <dt className="text-muted">
            Programas{inventario.programas_truncado ? " (lista parcial)" : ""}
          </dt>
          <dd>
            {inventario
              .programas!.map((p) => (p.versao ? `${p.nome} ${p.versao}` : p.nome))
              .join(" · ")}
          </dd>
        </div>
      ) : null}
    </dl>
  );
}

/** Detalhe do Item de Configuração (CMDB) e rastreabilidade reversa ITIL v5
 * nas 4 práticas: Chamados, Problemas, Mudanças e Projetos. */
export function AtivoDetalhe() {
  const id = useIdDaRota();
  const embutido = usePainelEmbutido();
  const { can } = usePlataforma();
  const gere = can("ativos:manage");
  const le = gere || can("ativos:read");

  const ativo = useApiQuery<Ativo>(id && le ? `v1/ativos/${id}` : null);
  const relacoes = useApiQuery<RelacaoAtivo[]>(id && le ? `v1/ativos/${id}/relacoes` : null);
  const todosAtivos = useApiQuery<Ativo[]>(id && gere ? "v1/ativos" : null);

  const chamados = useApiPage<Solicitacao>(
    id && le && can("atendimento:atender")
      ? withQuery("v1/atendimento/fila", { situacao: "todas", ativo_id: id, page_size: 50 })
      : null,
  );
  const problemas = useApiQuery<Problema[]>(
    id && le && (can("problemas:read") || can("problemas:manage"))
      ? withQuery("v1/problemas", { ativo_id: id })
      : null,
  );
  const mudancas = useApiQuery<Mudanca[]>(
    id && le && (can("mudancas:manage") || can("mudancas:aprovar"))
      ? withQuery("v1/mudancas", { alvo_tipo: "ativo", alvo_id: id })
      : null,
  );
  const projetos = useApiQuery<Projeto[]>(
    id && le && (can("projetos:read") || can("projetos:manage"))
      ? withQuery("v1/projetos", { alvo_tipo: "ativo", alvo_id: id })
      : null,
  );

  const [destinoId, setDestinoId] = useState("");
  const [tipoRelacao, setTipoRelacao] = useState("depende");
  const { run, pending } = useAction();

  if (!le) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Gestão de serviços" title="Ativo" />
        <EmptyState
          title="Seu perfil não consulta ativos"
          description="Peça o acesso à gestão da sua unidade."
        />
      </div>
    );
  }

  async function adicionarRelacao(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!destinoId) return;
    const ok = await run(
      () =>
        apiClient.post(`v1/ativos/${id}/relacoes`, {
          destino_id: destinoId,
          tipo: tipoRelacao,
        }),
      "Relação registrada",
    );
    if (ok) {
      setDestinoId("");
      void relacoes.mutate();
    }
  }

  async function removerRelacao(relId: string) {
    const ok = await run(
      () => apiClient.delete(`v1/ativos/${id}/relacoes/${relId}`),
      "Relação desfeita",
    );
    if (ok) void relacoes.mutate();
  }

  const a = ativo.data;
  const listaRelacoes = relacoes.data ?? [];
  const listaChamados = chamados.data?.items ?? [];
  const listaProblemas = problemas.data ?? [];
  const listaMudancas = mudancas.data ?? [];
  const listaProjetos = projetos.data ?? [];
  const opcoesDestino = (todosAtivos.data ?? []).filter((item) => item.id !== id);

  return (
    <div className="flex flex-col gap-6">
      {!embutido && (
        <Link
          href="/ativos"
          className="inline-flex w-fit items-center gap-1 text-sm text-primary hover:underline"
        >
          <ArrowLeft aria-hidden className="h-4 w-4" />
          Voltar para os ativos
        </Link>
      )}

      <DataState
        loading={ativo.isLoading}
        error={ativo.error}
        onRetry={() => void ativo.mutate()}
        empty={!a}
      >
        {a && (
          <>
            <PageHeader
              eyebrow={TIPO_ATIVO[a.tipo] ?? a.tipo}
              title={a.nome}
              description={
                a.identificador ? `Patrimônio / série: ${a.identificador}` : "Sem identificador"
              }
              actions={
                <Badge tone={ESTADO_ATIVO[a.estado]?.tone ?? "neutral"}>
                  {ESTADO_ATIVO[a.estado]?.label ?? a.estado}
                </Badge>
              }
            />

            <Card>
              <CardHeader>
                <CardTitle as="h2">Ficha do item de configuração</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-3 pb-5 text-sm">
                <dl className="grid gap-3 sm:grid-cols-2">
                  <div>
                    <dt className="text-xs text-muted">Tipo</dt>
                    <dd>{TIPO_ATIVO[a.tipo] ?? a.tipo}</dd>
                  </div>
                  <div>
                    <dt className="text-xs text-muted">Origem</dt>
                    <dd>
                      {a.origem === "agente" ? "Informado pelo agente da máquina" : "Cadastro manual"}
                    </dd>
                  </div>
                  <div className="sm:col-span-2">
                    <dt className="text-xs text-muted">Descrição</dt>
                    <dd className="whitespace-pre-wrap">{a.descricao || "—"}</dd>
                  </div>
                </dl>
                {a.inventario ? (
                  <div className="border-t border-surface-border pt-3">
                    <h3 className="text-xs font-semibold uppercase text-muted">
                      Inventário coletado
                    </h3>
                    <InventarioDaMaquina inventario={a.inventario} />
                  </div>
                ) : null}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle as="h2">Relações na CMDB</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-3 pb-5">
                {listaRelacoes.length === 0 ? (
                  <p className="text-sm text-muted">Nenhuma relação cadastrada para este ativo.</p>
                ) : (
                  <ul aria-label="Relações do ativo" className="flex flex-col gap-1.5 text-sm">
                    {listaRelacoes.map((r) => (
                      <li
                        key={r.id}
                        className="flex flex-wrap items-center justify-between gap-2 border-b border-surface-border/60 pb-1.5 last:border-b-0"
                      >
                        <span>
                          <span className="text-muted">{ROTULO_RELACAO[r.tipo] ?? r.tipo}:</span>{" "}
                          <Link
                            href={`/ativos/${r.destino_id}`}
                            className="font-medium text-primary hover:underline"
                          >
                            {r.destino_nome}
                          </Link>
                        </span>
                        {gere && (
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            disabled={pending}
                            aria-label={`Desfazer relação com ${r.destino_nome}`}
                            onClick={() => void removerRelacao(r.id)}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        )}
                      </li>
                    ))}
                  </ul>
                )}

                {gere && opcoesDestino.length > 0 && (
                  <form
                    onSubmit={(e) => void adicionarRelacao(e)}
                    className="mt-2 grid gap-3 border-t border-surface-border pt-3 sm:grid-cols-3"
                  >
                    <Select
                      id="relacao-tipo-detalhe"
                      label="Tipo da relação"
                      value={tipoRelacao}
                      onChange={(e) => setTipoRelacao(e.target.value)}
                      options={[
                        { value: "depende", label: "Depende" },
                        { value: "composto", label: "Compõe" },
                        { value: "conectado", label: "Conectado" },
                      ]}
                    />
                    <Select
                      id="relacao-destino-detalhe"
                      label="Aponta para *"
                      value={destinoId}
                      onChange={(e) => setDestinoId(e.target.value)}
                      placeholder="Escolha o ativo"
                      options={opcoesDestino.map((op) => ({ value: op.id, label: op.nome }))}
                    />
                    <div className="flex items-end">
                      <Button type="submit" variant="secondary" disabled={!destinoId || pending}>
                        Registrar relação
                      </Button>
                    </div>
                  </form>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle as="h2">Chamados vinculados</CardTitle>
              </CardHeader>
              <CardContent className="pb-5">
                {listaChamados.length === 0 ? (
                  <p className="text-sm text-muted">Nenhum chamado vinculado a este ativo.</p>
                ) : (
                  <ul className="flex flex-col gap-1.5 text-sm">
                    {listaChamados.map((c) => (
                      <li key={c.id} className="flex flex-wrap items-center justify-between gap-2">
                        <Link
                          href={`/atendimento/solicitacoes/${c.id}`}
                          className="text-primary hover:underline"
                        >
                          #{c.numero} · {c.titulo}
                        </Link>
                        <div className="flex items-center gap-1.5">
                          <Badge tone={c.tipo === "incidente" ? "warning" : "info"}>
                            {rotuloTipo(c.tipo)}
                          </Badge>
                          <Badge tone={tomEstado(c.estado)}>{rotuloEstado(c.estado)}</Badge>
                        </div>
                      </li>
                    ))}
                  </ul>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle as="h2">Problemas vinculados</CardTitle>
              </CardHeader>
              <CardContent className="pb-5">
                {listaProblemas.length === 0 ? (
                  <p className="text-sm text-muted">Nenhum problema vinculado a este ativo.</p>
                ) : (
                  <ul className="flex flex-col gap-1.5 text-sm">
                    {listaProblemas.map((p) => (
                      <li key={p.id} className="flex flex-wrap items-center justify-between gap-2">
                        <Link href={`/problemas/${p.id}`} className="text-primary hover:underline">
                          #{p.numero} · {p.titulo}
                        </Link>
                        <Badge tone={ESTADO_PROBLEMA[p.estado].tone}>
                          {ESTADO_PROBLEMA[p.estado].label}
                        </Badge>
                      </li>
                    ))}
                  </ul>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle as="h2">Mudanças vinculadas</CardTitle>
              </CardHeader>
              <CardContent className="pb-5">
                {listaMudancas.length === 0 ? (
                  <p className="text-sm text-muted">Nenhuma mudança vinculada a este ativo.</p>
                ) : (
                  <ul className="flex flex-col gap-1.5 text-sm">
                    {listaMudancas.map((m) => (
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
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle as="h2">Projetos vinculados</CardTitle>
              </CardHeader>
              <CardContent className="pb-5">
                {listaProjetos.length === 0 ? (
                  <p className="text-sm text-muted">Nenhum projeto vinculado a este ativo.</p>
                ) : (
                  <ul className="flex flex-col gap-1.5 text-sm">
                    {listaProjetos.map((pr) => (
                      <li key={pr.id} className="flex flex-wrap items-center justify-between gap-2">
                        <Link href={`/projetos/${pr.id}`} className="text-primary hover:underline">
                          #{pr.numero} · {pr.titulo}
                        </Link>
                        <Badge tone={ESTADO_PROJETO[pr.estado].tone}>
                          {ESTADO_PROJETO[pr.estado].label}
                        </Badge>
                      </li>
                    ))}
                  </ul>
                )}
              </CardContent>
            </Card>

            {a.created_at ? (
              <p className="text-xs text-muted">Cadastrado em {fmtDateTime(a.created_at)}</p>
            ) : null}
          </>
        )}
      </DataState>
    </div>
  );
}

export default function AtivoPage() {
  return <AtivoDetalhe />;
}
