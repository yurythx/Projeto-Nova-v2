"use client";

import { ArrowLeft } from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { type FormEvent, type ReactNode, useCallback, useState } from "react";

import { ApontamentosCard } from "@/components/atendimento/ApontamentosCard";
import { AtivoVinculadoCard } from "@/components/atendimento/AtivoVinculadoCard";
import { AvaliacaoAtendimentoCard } from "@/components/atendimento/AvaliacaoAtendimentoCard";
import { PrazoCard } from "@/components/atendimento/PrazoCard";
import { ProblemasDoIncidente } from "@/components/atendimento/ProblemasDoIncidente";
import { VisitasCard } from "@/components/atendimento/VisitasCard";
import { useTopic } from "@/components/realtime/RealtimeProvider";
import { ConfirmButton } from "@/components/plataforma/ConfirmButton";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import {
  rotuloAcao,
  rotuloEstado,
  rotuloTipo,
  tomEstado,
  type Detalhe,
  type Estado,
  type Pessoa,
} from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

function Campo({ rotulo, children }: { rotulo: string; children: ReactNode }) {
  return (
    <div>
      <dt className="text-xs font-medium uppercase tracking-wide text-muted">{rotulo}</dt>
      <dd className="mt-0.5 text-sm">{children}</dd>
    </div>
  );
}

export default function SolicitacaoPage() {
  const { id } = useParams<{ id: string }>();
  const { me } = usePlataforma();
  const detalhe = useApiQuery<Detalhe>(id ? `v1/atendimento/solicitacoes/${id}` : null);
  const d = detalhe.data;
  const tecnico = d?.papeis.includes("tecnico") ?? false;
  const solicitante = d?.papeis.includes("solicitante") ?? false;
  const s = d?.solicitacao;
  const finalizada = s?.estado === "encerrado" || s?.estado === "cancelado";
  const responsaveis = useApiQuery<Pessoa[]>(
    tecnico && !finalizada ? `v1/atendimento/solicitacoes/${id}/responsaveis` : null,
  );
  const [escolhido, setEscolhido] = useState("");
  const { mutate } = detalhe;
  const recarregar = useCallback(() => void mutate(), [mutate]);
  // A equipe ouve também as notas internas; quem abriu, só o que pode ver.
  useTopic(
    d ? `atendimento:${tecnico ? "equipe" : "solicitacao"}:${s!.id}` : null,
    () => void detalhe.mutate(),
  );
  const { run, pending } = useAction();

  async function passar(para: Estado) {
    const ok = await run(
      () => apiClient.post(`v1/atendimento/solicitacoes/${id}/estado`, { estado: para }),
      "Estado atualizado",
    );
    if (ok) void detalhe.mutate();
  }

  async function classificar(impacto: number, urgencia: number) {
    const ok = await run(
      () => apiClient.put(`v1/atendimento/solicitacoes/${id}/classificacao`, { impacto, urgencia }),
      "Classificação atualizada",
    );
    if (ok) void detalhe.mutate();
  }

  async function atribuir(responsavel: string) {
    const ok = await run(
      () =>
        apiClient.put(`v1/atendimento/solicitacoes/${id}/responsavel`, {
          responsavel_id: responsavel,
        }),
      "Responsável definido",
    );
    if (ok) {
      setEscolhido("");
      void detalhe.mutate();
    }
  }

  async function comentar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const interno = fd.get("interno") === "on";
    const ok = await run(
      () =>
        apiClient.post(`v1/atendimento/solicitacoes/${id}/comentarios`, {
          texto: String(fd.get("texto") ?? "").trim(),
          interno,
        }),
      interno ? "Nota registrada" : "Resposta enviada",
    );
    if (ok) {
      form.reset();
      void detalhe.mutate();
    }
  }

  const voltar = tecnico ? "/atendimento/fila" : "/atendimento";

  return (
    <div className="flex flex-col gap-6">
      <Link
        href={voltar}
        className="inline-flex w-fit items-center gap-1 text-sm text-primary hover:underline"
      >
        <ArrowLeft aria-hidden className="h-4 w-4" />
        {tecnico ? "Voltar para a fila" : "Voltar para minhas solicitações"}
      </Link>

      <DataState
        loading={detalhe.isLoading}
        error={detalhe.error}
        onRetry={() => void detalhe.mutate()}
        empty={!d}
      >
        {d && s && (
          <>
            <PageHeader
              eyebrow={`Solicitação #${s.numero}`}
              title={s.titulo}
              description={s.oferta_nome}
              actions={
                <>
                  {tecnico && <Badge>{rotuloTipo(s.tipo)}</Badge>}
                  <Badge tone={tomEstado(s.estado)}>{rotuloEstado(s.estado)}</Badge>
                </>
              }
            />

            {solicitante && s.estado === "aguardando_servidor" && (
              <p
                role="status"
                className="rounded-lg border border-warning/40 bg-warning/10 p-3 text-sm"
              >
                A equipe precisa de uma informação sua. Responda na conversa abaixo e a solicitação
                volta para o atendimento.
              </p>
            )}
            {solicitante && (s.estado === "resolvido" || s.estado === "atendido") && (
              <p
                role="status"
                className="rounded-lg border border-success/40 bg-success/10 p-3 text-sm"
              >
                A equipe concluiu o atendimento. Confirme se ficou tudo certo ou reabra a
                solicitação.
              </p>
            )}

            <Card>
              <CardHeader>
                <CardTitle as="h2">Detalhes</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-4 pb-5">
                <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                  <Campo rotulo="Aberta por">{s.solicitante_nome}</Campo>
                  <Campo rotulo="Aberta em">{fmtDateTime(s.created_at)}</Campo>
                  <Campo rotulo="Responsável">
                    {s.responsavel_nome || "Ainda sem responsável"}
                  </Campo>
                  {s.resolvida_em && (
                    <Campo rotulo="Concluída em">{fmtDateTime(s.resolvida_em)}</Campo>
                  )}
                  {s.encerrada_em && (
                    <Campo rotulo="Encerrada em">{fmtDateTime(s.encerrada_em)}</Campo>
                  )}
                </dl>
                {s.descricao ? <p className="whitespace-pre-wrap text-sm">{s.descricao}</p> : null}

                {(d.proximos.length > 0 || (tecnico && !finalizada)) && (
                  <div className="flex flex-col gap-3 border-t border-surface-border pt-4">
                    {tecnico && !finalizada && (
                      <div className="flex flex-wrap items-end gap-2">
                        {me && s.responsavel_id !== me.id && (
                          <Button
                            type="button"
                            disabled={pending}
                            onClick={() => void atribuir(me.id)}
                          >
                            Assumir
                          </Button>
                        )}
                        <div className="min-w-56">
                          <Select
                            id="solicitacao-responsavel"
                            label="Passar para"
                            value={escolhido}
                            onChange={(e) => setEscolhido(e.target.value)}
                            placeholder="Escolha alguém da equipe"
                            options={(responsaveis.data ?? [])
                              .filter((p) => p.id !== s.responsavel_id)
                              .map((p) => ({ value: p.id, label: p.nome }))}
                          />
                        </div>
                        <Button
                          type="button"
                          variant="secondary"
                          disabled={!escolhido || pending}
                          onClick={() => void atribuir(escolhido)}
                        >
                          Definir responsável
                        </Button>
                      </div>
                    )}
                    {d.proximos.length > 0 && (
                      <div
                        className="flex flex-wrap gap-2"
                        aria-label="Ações da solicitação"
                        role="group"
                      >
                        {d.proximos.map((para) =>
                          para === "cancelado" ? (
                            <ConfirmButton
                              key={para}
                              variant="danger"
                              size="md"
                              title="Cancelar a solicitação?"
                              description="A solicitação sai da fila e não recebe mais respostas."
                              confirmLabel="Cancelar solicitação"
                              onConfirm={() => passar(para)}
                            >
                              {rotuloAcao(para, s.estado, solicitante && !tecnico)}
                            </ConfirmButton>
                          ) : (
                            <Button
                              key={para}
                              type="button"
                              variant={para === "encerrado" ? "primary" : "secondary"}
                              disabled={pending}
                              onClick={() => void passar(para)}
                            >
                              {rotuloAcao(para, s.estado, solicitante && !tecnico)}
                            </Button>
                          ),
                        )}
                      </div>
                    )}
                  </div>
                )}
              </CardContent>
            </Card>

            {tecnico && (
              <PrazoCard
                solicitacao={s}
                editavel={!finalizada}
                pending={pending}
                onClassificar={(impacto, urgencia) => void classificar(impacto, urgencia)}
              />
            )}

            {tecnico && s.tipo === "incidente" && <ProblemasDoIncidente solicitacaoId={s.id} />}

            <AtivoVinculadoCard
              solicitacaoId={s.id}
              tecnico={tecnico}
              finalizada={finalizada}
            />

            <VisitasCard
              solicitacaoId={s.id}
              visitas={d.visitas ?? []}
              tecnico={tecnico}
              meuId={me?.id}
              finalizada={finalizada}
              onMudou={recarregar}
            />

            <ApontamentosCard
              solicitacaoId={s.id}
              tecnico={tecnico}
              finalizada={finalizada}
            />

            <AvaliacaoAtendimentoCard
              solicitacao={s}
              solicitante={solicitante}
              tecnico={tecnico}
              onAvaliado={recarregar}
            />

            <Card>
              <CardHeader>
                <CardTitle as="h2">Conversa</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-4 pb-5">
                {d.comentarios.length === 0 ? (
                  <p className="text-sm text-muted">Nenhuma mensagem ainda.</p>
                ) : (
                  <ol className="flex flex-col gap-3">
                    {d.comentarios.map((c) => (
                      <li
                        key={c.id}
                        className={`rounded-lg border p-3 ${
                          c.interno ? "border-warning/40 bg-warning/5" : "border-surface-border"
                        }`}
                      >
                        <p className="flex flex-wrap items-center gap-2 text-xs text-muted">
                          <span className="font-medium text-foreground">
                            {c.autor_nome || "Usuário removido"}
                          </span>
                          <span>{fmtDateTime(c.created_at)}</span>
                          {c.interno && <Badge tone="warning">Nota interna</Badge>}
                        </p>
                        <p className="mt-1 whitespace-pre-wrap text-sm">{c.texto}</p>
                      </li>
                    ))}
                  </ol>
                )}
                {!finalizada && (
                  <form
                    onSubmit={comentar}
                    className="grid gap-3 border-t border-surface-border pt-4"
                  >
                    <Textarea
                      id="comentario-texto"
                      name="texto"
                      label={tecnico ? "Mensagem" : "Sua resposta"}
                      required
                      maxLength={4000}
                      rows={3}
                    />
                    {tecnico && (
                      <label className="flex items-center gap-2 text-sm">
                        <input
                          id="comentario-interno"
                          name="interno"
                          type="checkbox"
                          className="h-4 w-4 accent-primary"
                        />
                        Nota interna (quem abriu não vê)
                      </label>
                    )}
                    <div>
                      <Button type="submit" loading={pending}>
                        Enviar
                      </Button>
                    </div>
                  </form>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle as="h2">Histórico</CardTitle>
              </CardHeader>
              <CardContent className="pb-5">
                <ol className="flex flex-col gap-2 text-sm">
                  {d.passagens.map((p) => (
                    <li key={p.id} className="flex flex-wrap gap-x-2">
                      <span className="font-medium">{rotuloEstado(p.para)}</span>
                      <span className="text-muted">
                        {fmtDateTime(p.em)}
                        {p.autor_nome ? ` · ${p.autor_nome}` : ""}
                      </span>
                    </li>
                  ))}
                </ol>
              </CardContent>
            </Card>
          </>
        )}
      </DataState>
    </div>
  );
}
