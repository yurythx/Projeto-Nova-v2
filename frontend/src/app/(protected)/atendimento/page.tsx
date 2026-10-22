"use client";

import Link from "next/link";
import { type FormEvent, useState } from "react";

import { useFrames } from "@/components/realtime/RealtimeProvider";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { Pagination } from "@/components/plataforma/Pagination";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import {
  FRAME_SOLICITACAO,
  rotuloEstado,
  tomEstado,
  type Oferta,
  type Solicitacao,
} from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

type Situacao = "abertas" | "todas";

/** Portal do servidor: pede ajuda pela oferta, em linguagem simples, e
 * acompanha o que abriu. A palavra "incidente" não aparece aqui. */
export default function AtendimentoPortalPage() {
  const { can } = usePlataforma();
  const abre = can("atendimento:abrir");
  const [ofertaId, setOfertaId] = useState("");
  const [situacao, setSituacao] = useState<Situacao>("abertas");
  const [pagina, setPagina] = useState(1);
  const catalogo = useApiQuery<Oferta[]>(abre ? "v1/atendimento/catalogo" : null);
  const minhas = useApiPage<Solicitacao>(
    abre ? withQuery("v1/atendimento/minhas", { situacao, page: pagina, page_size: 20 }) : null,
  );
  const { run, pending } = useAction();
  // A mudança feita pela equipe chega no tópico pessoal de quem abriu.
  useFrames((f) => {
    if (f.type === FRAME_SOLICITACAO && f.topic?.startsWith("user:")) void minhas.mutate();
  });

  if (!abre) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Atendimento" title="Minhas solicitações" />
        <EmptyState
          title="Seu perfil não abre solicitações"
          description="Peça à gestão da sua unidade o acesso para abrir e acompanhar pedidos de atendimento."
          action={
            can("atendimento:atender") ? (
              <Link href="/atendimento/fila" className={buttonClass("secondary")}>
                Ir para a fila da equipe
              </Link>
            ) : undefined
          }
        />
      </div>
    );
  }

  const ofertas = catalogo.data ?? [];
  const escolhida = ofertas.find((o) => o.id === ofertaId);

  async function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    // Os três campos estão sempre no formulário.
    const body = {
      oferta_id: String(fd.get("oferta_id")),
      titulo: String(fd.get("titulo")).trim(),
      descricao: String(fd.get("descricao")).trim(),
    };
    const criada = await run(
      () => apiClient.post<Solicitacao>("v1/atendimento/solicitacoes", body),
      "Solicitação enviada",
    );
    if (criada) {
      form.reset();
      setOfertaId("");
      setSituacao("abertas");
      setPagina(1);
      void minhas.mutate();
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Atendimento"
        title="Minhas solicitações"
        description="Escolha com o que precisa de ajuda e conte o que está acontecendo. A equipe responsável responde por aqui e você recebe o aviso no sino."
      />

      <Card>
        <CardHeader>
          <CardTitle as="h2">Nova solicitação</CardTitle>
        </CardHeader>
        <CardContent className="pb-5">
          <DataState
            loading={catalogo.isLoading}
            error={catalogo.error}
            onRetry={() => void catalogo.mutate()}
            empty={ofertas.length === 0}
            emptyTitle="Nenhum atendimento disponível"
            emptyDescription="A gestão ainda não cadastrou o que pode ser pedido por aqui."
            rows={2}
          >
            <form onSubmit={enviar} className="mt-3 grid gap-3">
              <Select
                id="solicitacao-oferta"
                name="oferta_id"
                label="Preciso de ajuda com *"
                required
                placeholder="Escolha uma opção"
                value={ofertaId}
                onChange={(e) => setOfertaId(e.target.value)}
                options={ofertas.map((o) => ({ value: o.id, label: o.nome }))}
              />
              {escolhida?.descricao ? (
                <p className="text-sm text-muted">{escolhida.descricao}</p>
              ) : null}
              <Input
                id="solicitacao-titulo"
                name="titulo"
                label="Resumo *"
                required
                maxLength={160}
                autoComplete="off"
                placeholder="Ex.: a senha do e-mail expirou"
              />
              <Textarea
                id="solicitacao-descricao"
                name="descricao"
                label="Detalhes"
                maxLength={4000}
                rows={4}
                placeholder="Desde quando, onde aparece o erro, um telefone para contato…"
              />
              <div>
                <Button type="submit" loading={pending}>
                  Enviar solicitação
                </Button>
              </div>
            </form>
          </DataState>
        </CardContent>
      </Card>

      <section aria-labelledby="minhas-titulo" className="flex flex-col gap-3">
        <h2 id="minhas-titulo" className="text-lg font-semibold">
          Acompanhar
        </h2>
        <SectionTabsInline
          label="Situação das solicitações"
          value={situacao}
          onChange={(v) => {
            setSituacao(v);
            setPagina(1);
          }}
          tabs={[
            { value: "abertas", label: "Em andamento" },
            { value: "todas", label: "Todas" },
          ]}
        />
        <DataState
          loading={minhas.isLoading}
          error={minhas.error}
          onRetry={() => void minhas.mutate()}
          empty={(minhas.data?.items ?? []).length === 0}
          emptyTitle={
            situacao === "abertas" ? "Nada em andamento" : "Você ainda não abriu solicitações"
          }
        >
          <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
            {(minhas.data?.items ?? []).map((s) => (
              <li
                key={s.id}
                className="flex flex-col gap-2 p-4 sm:flex-row sm:items-center sm:justify-between"
              >
                <div className="min-w-0">
                  <Link
                    href={`/atendimento/solicitacoes/${s.id}`}
                    className="font-medium text-primary hover:underline"
                  >
                    #{s.numero} · {s.titulo}
                  </Link>
                  <p className="text-sm text-muted">
                    {s.oferta_nome} · atualizada em {fmtDateTime(s.updated_at)}
                  </p>
                </div>
                <Badge tone={tomEstado(s.estado)}>{rotuloEstado(s.estado)}</Badge>
              </li>
            ))}
          </ul>
          <Pagination meta={minhas.data?.meta} onPage={setPagina} />
        </DataState>
      </section>
    </div>
  );
}
