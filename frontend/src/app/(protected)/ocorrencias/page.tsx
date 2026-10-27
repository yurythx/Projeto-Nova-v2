"use client";

import { ExternalLink, Siren } from "lucide-react";
import Link from "next/link";
import { useState, type FormEvent } from "react";

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
import { useApiQuery } from "@/lib/api/swr";
import {
  ESTADO,
  ESTADOS,
  IMPACTO,
  IMPACTOS,
  minutosSemNoticia,
  type Ocorrencia,
} from "@/lib/ocorrencias/ocorrencias";

const opcoesImpacto = IMPACTOS.map((i) => ({ value: i, label: IMPACTO[i].label }));
const opcoesEstado = ESTADOS.map((e) => ({ value: e, label: ESTADO[e] }));

/** Sem notícia há mais disto, o painel avisa: o público está esperando. */
const SEM_NOTICIA_MIN = 60;

function Declarar({ onDeclarada }: { onDeclarada: (id: string) => void }) {
  const { run, pending } = useAction();
  async function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const res = await run(
      () =>
        apiClient.post<Ocorrencia>("v1/ocorrencias", {
          titulo: String(fd.get("titulo") ?? "").trim(),
          servicos: String(fd.get("servicos") ?? "")
            .split(",")
            .map((s) => s.trim())
            .filter(Boolean),
          impacto: String(fd.get("impacto")),
          texto: String(fd.get("texto") ?? "").trim(),
        }),
      "Ocorrência declarada e publicada em /status",
    );
    if (res) {
      form.reset();
      onDeclarada(res.data.id);
    }
  }
  return (
    <form onSubmit={enviar} className="grid gap-3">
      <p className="text-sm text-muted">
        O que você escrever aqui vai para a página pública. Não copie o chamado: nada de nomes,
        documentos ou detalhes internos.
      </p>
      <Input
        id="oc-titulo"
        name="titulo"
        label="Título público *"
        required
        maxLength={160}
        autoComplete="off"
      />
      <Input
        id="oc-servicos"
        name="servicos"
        label="Serviços afetados (separe por vírgula) *"
        required
        autoComplete="off"
      />
      <Select
        id="oc-impacto"
        name="impacto"
        label="Impacto *"
        options={opcoesImpacto}
        defaultValue="fora"
      />
      <Textarea
        id="oc-texto"
        name="texto"
        label="Primeira mensagem ao público *"
        required
        maxLength={4000}
        rows={3}
      />
      <div>
        <Button type="submit" loading={pending}>
          Declarar ocorrência
        </Button>
      </div>
    </form>
  );
}

function Detalhe({ id, onMudou }: { id: string; onMudou: () => void }) {
  const detalhe = useApiQuery<Ocorrencia>(`v1/ocorrencias/${id}`);
  const { run, pending } = useAction();
  const o = detalhe.data;

  async function atualizar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(
      () =>
        apiClient.post(`v1/ocorrencias/${id}/atualizacoes`, {
          estado: String(fd.get("estado")),
          impacto: String(fd.get("impacto")),
          texto: String(fd.get("texto") ?? "").trim(),
        }),
      "Atualização publicada",
    );
    if (ok) {
      form.reset();
      void detalhe.mutate();
      onMudou();
    }
  }

  async function revisar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const ok = await run(
      () =>
        apiClient.put(`v1/ocorrencias/${id}/revisao`, {
          texto: String(fd.get("revisao") ?? "").trim(),
        }),
      "Revisão gravada",
    );
    if (ok) void detalhe.mutate();
  }

  return (
    <DataState loading={detalhe.isLoading} error={detalhe.error} empty={!o}>
      {o && (
        <div className="flex flex-col gap-5">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-lg font-semibold">{o.titulo}</h3>
            <Badge tone={o.estado === "resolvido" ? "success" : IMPACTO[o.impacto].tone}>
              {o.estado === "resolvido" ? "Resolvida" : IMPACTO[o.impacto].label}
            </Badge>
          </div>
          <p className="text-sm text-muted">
            {o.servicos.join(", ")} · coordenação: {o.coordenador_nome || "—"}
            {o.chamado_id && (
              <>
                {" · "}
                <Link href={`/atendimento/solicitacoes/${o.chamado_id}`} className="underline">
                  chamado principal
                </Link>
              </>
            )}
          </p>

          <ol
            aria-label="Linha do tempo"
            className="flex flex-col gap-3 border-l border-surface-border pl-4"
          >
            {[...o.atualizacoes].reverse().map((a) => (
              <li key={a.id} className="flex flex-col gap-0.5 text-sm">
                <span className="font-medium">
                  {ESTADO[a.estado]}{" "}
                  <span className="font-normal text-muted">
                    · {IMPACTO[a.impacto].label} · {fmtDateTime(a.created_at)}
                    {a.autor_nome && ` · ${a.autor_nome}`}
                  </span>
                </span>
                <span className="whitespace-pre-line">{a.texto}</span>
              </li>
            ))}
          </ol>

          {o.estado !== "resolvido" ? (
            <form onSubmit={atualizar} className="grid gap-3 border-t border-surface-border pt-4">
              <h4 className="text-sm font-semibold">Publicar atualização</h4>
              <div className="grid gap-3 sm:grid-cols-2">
                <Select
                  id="up-estado"
                  name="estado"
                  label="Estado"
                  options={opcoesEstado}
                  defaultValue={o.estado}
                />
                <Select
                  id="up-impacto"
                  name="impacto"
                  label="Impacto"
                  options={opcoesImpacto}
                  defaultValue={o.impacto}
                />
              </div>
              <Textarea
                id="up-texto"
                name="texto"
                label="Mensagem ao público *"
                required
                maxLength={4000}
                rows={3}
              />
              <div>
                <Button type="submit" loading={pending}>
                  Publicar
                </Button>
              </div>
            </form>
          ) : (
            <form onSubmit={revisar} className="grid gap-3 border-t border-surface-border pt-4">
              <h4 className="text-sm font-semibold">Revisão pós-incidente (interna)</h4>
              <Textarea
                id="up-revisao"
                name="revisao"
                label="Causa, o que funcionou e o que fazer para não repetir"
                required
                maxLength={4000}
                rows={4}
                defaultValue={o.revisao ?? ""}
              />
              <div>
                <Button type="submit" loading={pending}>
                  Gravar revisão
                </Button>
              </div>
            </form>
          )}
        </div>
      )}
    </DataState>
  );
}

/** Ocorrências (ADR 066): incidente grave, comunicação ao público e
 * revisão. A permissão ocorrencias:manage só vale com concessão global. */
export default function OcorrenciasPage() {
  const lista = useApiQuery<Ocorrencia[]>("v1/ocorrencias");
  const [selecionada, setSelecionada] = useState<string | null>(null);
  // Um só "agora" por montagem: o aviso de "sem notícia" não muda entre
  // renderizações.
  const [agora] = useState(() => Date.now());

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Gestão de serviços"
        title="Ocorrências"
        description="Incidente grave: quem coordena, o que o público lê em /status e a revisão depois."
        actions={
          <Link
            href="/status"
            target="_blank"
            className="inline-flex items-center gap-1 text-sm underline"
          >
            Ver a página pública <ExternalLink size={14} aria-hidden="true" />
          </Link>
        }
      />

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)]">
        <div className="flex flex-col gap-6">
          <Card>
            <CardHeader>
              <CardTitle as="h2" className="flex items-center gap-2">
                <Siren size={18} aria-hidden="true" className="text-danger" /> Declarar ocorrência
              </CardTitle>
            </CardHeader>
            <CardContent className="pb-5">
              <Declarar
                onDeclarada={(id) => {
                  void lista.mutate();
                  setSelecionada(id);
                }}
              />
            </CardContent>
          </Card>

          <DataState
            loading={lista.isLoading}
            error={lista.error}
            onRetry={() => void lista.mutate()}
            empty={(lista.data ?? []).length === 0}
            emptyTitle="Nenhuma ocorrência registrada"
          >
            <ul
              aria-label="Ocorrências"
              className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface"
            >
              {(lista.data ?? []).map((o) => {
                const aberta = o.estado !== "resolvido";
                const sem = minutosSemNoticia(o, agora);
                return (
                  <li key={o.id}>
                    <button
                      type="button"
                      aria-current={selecionada === o.id ? "true" : undefined}
                      onClick={() => setSelecionada(o.id)}
                      className="flex w-full flex-col gap-1 p-4 text-left hover:bg-black/[0.03] dark:hover:bg-white/[0.04]"
                    >
                      <span className="flex flex-wrap items-center gap-2">
                        <span className="font-medium">{o.titulo}</span>
                        <Badge tone={aberta ? IMPACTO[o.impacto].tone : "success"}>
                          {aberta ? ESTADO[o.estado] : "Resolvida"}
                        </Badge>
                      </span>
                      {aberta && sem >= SEM_NOTICIA_MIN && (
                        <span className="text-xs font-medium text-danger">
                          Sem notícia ao público há {Math.floor(sem / 60)} h {sem % 60} min
                        </span>
                      )}
                    </button>
                  </li>
                );
              })}
            </ul>
          </DataState>
        </div>

        <Card>
          <CardContent className="py-5">
            {selecionada ? (
              <Detalhe key={selecionada} id={selecionada} onMudou={() => void lista.mutate()} />
            ) : (
              <p className="text-sm text-muted">
                Escolha uma ocorrência para ver a linha do tempo.
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
