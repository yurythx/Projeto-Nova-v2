"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import type { FormEvent } from "react";

import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

interface Campo {
  id: string;
  rotulo: string;
  tipo: string;
}

interface Pedido {
  id: string;
  numero: string;
  nome: string;
  cpf?: string;
  email?: string;
  telefone?: string;
  servico_nome: string;
  unidade_id: string;
  unidade_nome?: string;
  status: string;
  prioridade: string;
  complemento: string;
  decisao: string;
  justificativa: string;
  created_at: string;
  campos?: Campo[];
  respostas?: Record<string, unknown>;
}

interface Historico {
  id: string;
  acao: string;
  descricao: string;
  status: string;
  anexos?: string[];
  created_at: string;
}

interface Ficha {
  pedido: Pedido;
  historico: Historico[];
  agendamento?: { data: string; hora: string; status: string };
}

const FINAIS = new Set(["concluido", "aprovado", "rejeitado", "cancelado"]);

export default function FichaPage() {
  const { id } = useParams<{ id: string }>();
  const { can } = usePlataforma();
  const ficha = useApiQuery<Ficha>(id ? `v1/protocolos/pedidos/${id}` : null);
  const { run, pending } = useAction();
  const pedido = ficha.data?.pedido;
  const trata = can("protocolos:tratar") && pedido && !FINAIS.has(pedido.status);

  async function baixar(campo: string) {
    const link = await run(() =>
      apiClient.get<{ url: string }>(`v1/protocolos/pedidos/${id}/arquivo?campo=${encodeURIComponent(campo)}`),
    );
    if (link?.data.url) window.open(link.data.url, "_blank", "noopener,noreferrer");
  }

  async function acao(path: string, body: unknown, aviso: string, e?: FormEvent<HTMLFormElement>) {
    e?.preventDefault();
    const ok = await run(() => apiClient.post(path, body), aviso);
    if (ok) void ficha.mutate();
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Protocolos"
        title={pedido?.numero ?? "Pedido"}
        description={pedido ? `${pedido.servico_nome} · ${pedido.unidade_nome ?? ""}` : "Ficha do pedido."}
        actions={
          <Link href="/protocolos" className={buttonClass("secondary", "sm")}>
            Mesa
          </Link>
        }
      />
      <DataState
        loading={ficha.isLoading}
        error={ficha.error}
        onRetry={() => void ficha.mutate()}
        empty={!pedido}
        emptyTitle="Pedido fora da sua mesa"
      >
        {pedido && (
          <div className="grid gap-6 lg:grid-cols-[2fr_1fr]">
            <Card>
              <CardHeader>
                <CardTitle as="h2">
                  {pedido.nome} <Badge>{pedido.status}</Badge>
                </CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-3 pb-5 text-sm">
                <p>
                  CPF {pedido.cpf || "—"} · {pedido.email || "sem e-mail"} · {pedido.telefone || "sem telefone"}
                </p>
                <p className="text-muted">
                  Prioridade {pedido.prioridade} · aberto em {fmtDateTime(pedido.created_at)}
                </p>
                {ficha.data?.agendamento && (
                  <div className="flex flex-wrap items-center gap-3">
                    <span>
                      Horário {ficha.data.agendamento.data} às {ficha.data.agendamento.hora} (
                      {ficha.data.agendamento.status})
                    </span>
                    {trata && pedido.cpf && (
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        loading={pending}
                        onClick={() =>
                          void acao(
                            "v1/protocolos/faltas",
                            { cpf: pedido.cpf, unidade_id: pedido.unidade_id },
                            "Falta registrada",
                          )
                        }
                      >
                        Registrar falta
                      </Button>
                    )}
                  </div>
                )}
                {ficha.data?.agendamento && (
                  <p className="text-xs text-muted">Duas faltas bloqueiam novo horário por 30 dias a partir da última.</p>
                )}
                {pedido.complemento && <p>Complementação: {pedido.complemento}</p>}
                <ul className="flex flex-col gap-2">
                  {(pedido.campos ?? [])
                    .filter((campo) => campo.tipo !== "page")
                    .map((campo) => (
                      <li key={campo.id}>
                        <span className="text-muted">{campo.rotulo}: </span>
                        {campo.tipo === "file" && chaveDeArquivo(pedido.respostas?.[campo.id]) ? (
                          <button
                            type="button"
                            className="text-primary underline"
                            onClick={() => void baixar(campo.id)}
                          >
                            Baixar
                          </button>
                        ) : (
                          mostrar(pedido.respostas?.[campo.id])
                        )}
                      </li>
                    ))}
                </ul>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle as="h2">Histórico</CardTitle>
              </CardHeader>
              <CardContent className="pb-5">
                <ul className="flex flex-col gap-3 text-sm">
                  {(ficha.data?.historico ?? []).map((item) => (
                    <li key={item.id}>
                      <p className="font-medium">{item.descricao}</p>
                      <p className="text-xs text-muted">
                        {item.status} · {fmtDateTime(item.created_at)}
                      </p>
                      {(item.anexos ?? []).map((anexo) => (
                        <button key={anexo} type="button" className="text-xs text-primary underline" onClick={() => void baixar(anexo)}>
                          Baixar anexo
                        </button>
                      ))}
                    </li>
                  ))}
                </ul>
              </CardContent>
            </Card>
            {trata && (
              <Card className="lg:col-span-2">
                <CardHeader>
                  <CardTitle as="h2">Tratamento</CardTitle>
                </CardHeader>
                <CardContent className="flex flex-col gap-4 pb-5">
                  <form
                    className="flex flex-col gap-3 sm:flex-row sm:items-end"
                    onSubmit={(e) => {
                      const mensagem = String(new FormData(e.currentTarget).get("mensagem") ?? "");
                      void acao(`v1/protocolos/pedidos/${pedido.id}/complemento`, { mensagem }, "Complementação pedida", e);
                    }}
                  >
                    <Input id="mensagem" name="mensagem" label="Pedir complementação" required maxLength={2000} />
                    <Button type="submit" variant="secondary" loading={pending}>
                      Pedir
                    </Button>
                  </form>
                  <div className="flex flex-wrap gap-2">
                    <Button
                      type="button"
                      variant="secondary"
                      loading={pending}
                      onClick={() => void acao(`v1/protocolos/pedidos/${pedido.id}/resposta`, {}, "Complementação respondida")}
                    >
                      Registrar resposta
                    </Button>
                    <Button
                      type="button"
                      variant="secondary"
                      loading={pending}
                      onClick={() => void acao(`v1/protocolos/pedidos/${pedido.id}/avancar`, {}, "Pedido avançou")}
                    >
                      Avançar etapa
                    </Button>
                    <Button
                      type="button"
                      variant="danger"
                      loading={pending}
                      onClick={() => void acao(`v1/protocolos/pedidos/${pedido.id}/cancelar`, {}, "Pedido cancelado")}
                    >
                      Cancelar
                    </Button>
                  </div>
                  <form
                    className="grid gap-3 sm:grid-cols-[12rem_1fr_auto] sm:items-end"
                    onSubmit={(e) => {
                      const dados = new FormData(e.currentTarget);
                      void acao(
                        `v1/protocolos/pedidos/${pedido.id}/concluir`,
                        {
                          decisao: String(dados.get("decisao") ?? ""),
                          justificativa: String(dados.get("justificativa") ?? ""),
                        },
                        "Pedido concluído",
                        e,
                      );
                    }}
                  >
                    <Select
                      id="decisao"
                      name="decisao"
                      label="Decisão *"
                      required
                      placeholder="Selecione…"
                      options={[
                        { value: "favoravel", label: "Favorável" },
                        { value: "desfavoravel", label: "Desfavorável" },
                      ]}
                    />
                    <Textarea id="justificativa" name="justificativa" label="Justificativa *" required maxLength={4000} />
                    <Button type="submit" loading={pending}>
                      Concluir
                    </Button>
                  </form>
                </CardContent>
              </Card>
            )}
          </div>
        )}
      </DataState>
    </div>
  );
}

function chaveDeArquivo(valor: unknown): boolean {
  return typeof valor === "string" && valor.startsWith("protocolos/");
}

function mostrar(valor: unknown): string {
  if (valor == null || valor === "") return "—";
  if (typeof valor === "boolean") return valor ? "sim" : "não";
  if (Array.isArray(valor)) return valor.map(String).join(", ");
  return String(valor);
}
