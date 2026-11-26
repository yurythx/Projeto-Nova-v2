"use client";

import Link from "next/link";
import { Suspense, type FormEvent, type ReactNode } from "react";

import {
  ArrowLeft,
  Building2,
  CalendarClock,
  CalendarDays,
  Download,
  GitBranch,
  Mail,
  MessageSquareWarning,
  Printer,
  UserRound,
} from "lucide-react";

import {
  etapaLegivel,
  rotuloPrazo,
  rotuloPrioridade,
  rotuloStatus,
  tomPrazo,
} from "@/components/protocolos/painel";
import { tomPrioridade, tomStatus } from "@/components/protocolos/quadro";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useIdDaRota, usePainelEmbutido } from "@/components/plataforma/PainelDoQuadro";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { OrgTree, UserAdmin } from "@/lib/plataforma/types";

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
  departamento_id?: string;
  status: string;
  situacao_prazo?: string;
  etapa_id?: string;
  prioridade: string;
  prazo_em?: string;
  responsavel_id?: string;
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

const ROTULO_ACAO: Record<string, string> = {
  criado: "Pedido aberto",
  avanco: "Avançou de etapa",
  complemento: "Complementação pedida",
  resposta: "Complementação respondida",
  nota: "Nota interna",
  edicao: "Pedido editado",
  conclusao: "Pedido concluído",
  cancelado: "Pedido cancelado",
};

/** Descrições que o servidor grava quando o evento não tem texto próprio: o título já diz. */
const FATO_PADRAO = new Set([
  "pedido aberto",
  "pedido avançou de etapa",
  "pedido cancelado",
  "pedido concluído",
  "há complementação",
  "complementação respondida",
  "pedido atualizado",
]);

const PONTO: Record<string, string> = {
  neutral: "bg-foreground/40",
  info: "bg-accent",
  warning: "bg-warning",
  success: "bg-success",
  danger: "bg-danger",
};

export default function FichaPage() {
  return (
    <Suspense fallback={null}>
      <FichaConteudo />
    </Suspense>
  );
}

/** Ficha do pedido. Na página inteira e no painel por cima da mesa. */
export function FichaConteudo() {
  const id = useIdDaRota();
  const embutido = usePainelEmbutido();
  const { can } = usePlataforma();
  const ficha = useApiQuery<Ficha>(id ? `v1/protocolos/pedidos/${id}` : null);
  const { run, pending } = useAction();
  const pedido = ficha.data?.pedido;
  const arvore = useApiQuery<OrgTree[]>(id ? "v1/iam/org-tree" : null);
  const pessoas = useApiPage<UserAdmin>(
    can("users:read") ? withQuery("v1/users", { active: true, page_size: 50 }) : null,
  );
  const trata = can("protocolos:tratar") && pedido && !FINAIS.has(pedido.status);
  const complementando = pedido?.status === "em_complementacao";
  const agendamento = ficha.data?.agendamento;

  async function baixar(campo: string) {
    const link = await run(() =>
      apiClient.get<{ url: string }>(
        `v1/protocolos/pedidos/${id}/arquivo?campo=${encodeURIComponent(campo)}`,
      ),
    );
    if (link?.data.url) window.open(link.data.url, "_blank", "noopener,noreferrer");
  }

  async function acao(path: string, body: unknown, aviso: string, e?: FormEvent<HTMLFormElement>) {
    e?.preventDefault();
    const formulario = e?.currentTarget;
    const ok = await run(() => apiClient.post(path, body), aviso);
    if (ok) {
      formulario?.reset();
      void ficha.mutate();
    }
  }

  return (
    <div className="flex flex-col gap-6">
      {!embutido && (
        <div>
          <Link
            href="/protocolos"
            className="inline-flex items-center gap-1.5 text-xs text-muted hover:text-foreground"
          >
            <ArrowLeft className="h-3.5 w-3.5" aria-hidden />
            Voltar para a mesa
          </Link>
        </div>
      )}
      <DataState
        loading={ficha.isLoading}
        error={ficha.error}
        onRetry={() => void ficha.mutate()}
        empty={!pedido}
        emptyTitle="Pedido fora da sua mesa"
      >
        {pedido && (
          <>
            <PageHeader
              eyebrow={`Protocolo · ${pedido.servico_nome}`}
              title={`Pedido ${pedido.numero}`}
              description={`Aberto por ${pedido.nome} em ${fmtDateTime(pedido.created_at)}.`}
              actions={
                <div className="flex flex-wrap items-center gap-2">
                  <Badge tone={tomStatus(pedido.status)}>{rotuloStatus(pedido.status)}</Badge>
                  {pedido.situacao_prazo && pedido.situacao_prazo !== "finalizado" && (
                    <Badge tone={tomPrazo(pedido.situacao_prazo)}>
                      {rotuloPrazo(pedido.situacao_prazo)}
                    </Badge>
                  )}
                  <Badge tone={tomPrioridade(pedido.prioridade)}>
                    Prioridade {rotuloPrioridade(pedido.prioridade).toLowerCase()}
                  </Badge>
                  <Button
                    size="sm"
                    variant="secondary"
                    type="button"
                    onClick={() => window.print()}
                    className="inline-flex items-center gap-1.5"
                  >
                    <Printer className="h-4 w-4" aria-hidden />
                    Imprimir
                  </Button>
                </div>
              }
            />

            {complementando && pedido.complemento && (
              <div className="flex items-start gap-3 rounded-xl border border-warning/40 bg-warning/10 p-4 text-sm">
                <MessageSquareWarning
                  className="mt-0.5 h-4 w-4 shrink-0 text-warning"
                  aria-hidden
                />
                <div>
                  <p className="font-semibold text-foreground">Aguardando o cidadão</p>
                  <p className="text-foreground">{pedido.complemento}</p>
                </div>
              </div>
            )}
            {FINAIS.has(pedido.status) && pedido.justificativa && (
              <div
                className={`rounded-xl border p-4 text-sm ${
                  pedido.decisao === "desfavoravel"
                    ? "border-danger/40 bg-danger/5"
                    : "border-success/40 bg-success/5"
                }`}
              >
                <p className="font-semibold text-foreground">
                  Decisão {pedido.decisao === "desfavoravel" ? "desfavorável" : "favorável"}
                </p>
                <p className="text-foreground">{pedido.justificativa}</p>
              </div>
            )}

            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
              <Info icone={<UserRound className="h-4 w-4" />} titulo="Requerente">
                <p className="font-medium text-foreground">{pedido.nome}</p>
                <p className="text-xs text-muted">CPF {pedido.cpf || "—"}</p>
              </Info>
              <Info icone={<Mail className="h-4 w-4" />} titulo="Contato">
                <p className="truncate text-foreground">{pedido.email || "Sem e-mail"}</p>
                <p className="text-xs text-muted">{pedido.telefone || "Sem telefone"}</p>
              </Info>
              <Info icone={<GitBranch className="h-4 w-4" />} titulo="Etapa e responsável">
                <p className="font-medium text-foreground">{etapaLegivel(pedido.etapa_id ?? "")}</p>
                <p className="truncate text-xs text-muted">
                  {responsavelVisivel(pedido, arvore.data, pessoas.data?.items)}
                </p>
              </Info>
              <Info icone={<CalendarClock className="h-4 w-4" />} titulo="Prazo">
                <p className="font-medium text-foreground">
                  {pedido.prazo_em ? fmtDia(pedido.prazo_em) : "Sem prazo"}
                </p>
                <p className="text-xs text-muted">
                  {pedido.unidade_nome ? (
                    <span className="inline-flex items-center gap-1">
                      <Building2 className="h-3 w-3" aria-hidden />
                      {pedido.unidade_nome}
                    </span>
                  ) : (
                    "Dias úteis em Cuiabá"
                  )}
                </p>
              </Info>
            </div>

            {agendamento && (
              <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-surface-border bg-surface p-4 shadow-sm">
                <div className="flex items-center gap-3">
                  <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-accent/10 text-accent">
                    <CalendarDays className="h-4 w-4" aria-hidden />
                  </span>
                  <div className="text-sm">
                    <p className="font-semibold text-foreground">
                      {fmtDataIso(agendamento.data)} às {agendamento.hora}
                    </p>
                    <p className="text-xs text-muted">
                      Horário {rotuloHorario(agendamento.status)} · duas faltas bloqueiam novo
                      horário por 30 dias
                    </p>
                  </div>
                </div>
                {trata &&
                  (agendamento.status === "agendado" || agendamento.status === "confirmado") && (
                    <Button
                      type="button"
                      variant="secondary"
                      size="sm"
                      loading={pending}
                      onClick={() =>
                        void acao(
                          "v1/protocolos/faltas",
                          { pedido_id: pedido.id },
                          "Falta registrada",
                        )
                      }
                    >
                      Registrar falta
                    </Button>
                  )}
              </div>
            )}

            <div className={`grid gap-6 ${embutido ? "" : "lg:grid-cols-[3fr_2fr]"}`}>
              <Card>
                <CardHeader>
                  <CardTitle as="h2">Dados do pedido</CardTitle>
                </CardHeader>
                <CardContent className="pb-5">
                  <dl className="grid gap-x-6 gap-y-4 sm:grid-cols-2">
                    {(pedido.campos ?? [])
                      .filter((campo) => campo.tipo !== "page")
                      .map((campo) => {
                        const valor = pedido.respostas?.[campo.id];
                        const longo = campo.tipo === "textarea" || mostrar(valor).length > 60;
                        return (
                          <div key={campo.id} className={longo ? "sm:col-span-2" : ""}>
                            <dt className="text-xs font-medium text-muted">{campo.rotulo}</dt>
                            <dd className="mt-0.5 text-sm text-foreground">
                              {campo.tipo === "file" && chaveDeArquivo(valor) ? (
                                <button
                                  type="button"
                                  className="inline-flex items-center gap-1 text-primary hover:underline"
                                  onClick={() => void baixar(campo.id)}
                                >
                                  <Download className="h-3.5 w-3.5" aria-hidden />
                                  Baixar arquivo
                                </button>
                              ) : (
                                <span className="whitespace-pre-wrap break-words">
                                  {mostrar(valor)}
                                </span>
                              )}
                            </dd>
                          </div>
                        );
                      })}
                  </dl>
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle as="h2">Histórico</CardTitle>
                </CardHeader>
                <CardContent className="pb-5">
                  <ol className="relative flex flex-col gap-5 border-l border-surface-border pl-5">
                    {(ficha.data?.historico ?? []).map((item) => (
                      <li key={item.id} className="relative">
                        <span
                          className={`absolute -left-[26px] top-1 h-3 w-3 rounded-full ring-4 ring-surface ${
                            item.acao === "nota"
                              ? "bg-foreground/30"
                              : PONTO[tomStatus(item.status)]
                          }`}
                          aria-hidden
                        />
                        <p className="text-xs font-semibold uppercase tracking-wide text-muted">
                          {ROTULO_ACAO[item.acao] ?? item.acao}
                        </p>
                        {item.descricao &&
                          !FATO_PADRAO.has(item.descricao.trim().toLowerCase()) && (
                            <p className="mt-0.5 text-sm text-foreground">{item.descricao}</p>
                          )}
                        <p className="mt-0.5 text-xs text-muted">
                          {rotuloStatus(item.status)} · {fmtDateTime(item.created_at)}
                        </p>
                        {(item.anexos ?? []).map((anexo) => (
                          <button
                            key={anexo}
                            type="button"
                            className="mt-1 inline-flex items-center gap-1 text-xs text-primary hover:underline"
                            onClick={() => void baixar(anexo)}
                          >
                            <Download className="h-3 w-3" aria-hidden />
                            Baixar anexo
                          </button>
                        ))}
                      </li>
                    ))}
                  </ol>
                </CardContent>
              </Card>
            </div>

            {trata && (
              <Card>
                <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-3">
                  <CardTitle as="h2">Tratamento</CardTitle>
                  <div className="flex flex-wrap gap-2">
                    {!complementando && (
                      <Button
                        type="button"
                        variant="secondary"
                        size="sm"
                        loading={pending}
                        onClick={() =>
                          void acao(
                            `v1/protocolos/pedidos/${pedido.id}/avancar`,
                            {},
                            "Pedido avançou",
                          )
                        }
                      >
                        Avançar etapa
                      </Button>
                    )}
                    <Button
                      type="button"
                      variant="danger"
                      size="sm"
                      loading={pending}
                      onClick={() =>
                        void acao(
                          `v1/protocolos/pedidos/${pedido.id}/cancelar`,
                          {},
                          "Pedido cancelado",
                        )
                      }
                    >
                      Cancelar pedido
                    </Button>
                  </div>
                </CardHeader>
                <CardContent className="grid gap-6 pb-5 lg:grid-cols-2">
                  <div className="flex flex-col gap-4">
                    {!complementando ? (
                      <form
                        className="flex flex-col gap-2"
                        onSubmit={(e) => {
                          const mensagem = String(
                            new FormData(e.currentTarget).get("mensagem") ?? "",
                          );
                          void acao(
                            `v1/protocolos/pedidos/${pedido.id}/complemento`,
                            { mensagem },
                            "Complementação pedida",
                            e,
                          );
                        }}
                      >
                        <Textarea
                          id="mensagem"
                          name="mensagem"
                          label="Pedir complementação ao cidadão"
                          required
                          maxLength={2000}
                          rows={3}
                        />
                        <div>
                          <Button type="submit" variant="secondary" size="sm" loading={pending}>
                            Pedir complementação
                          </Button>
                        </div>
                      </form>
                    ) : (
                      <form
                        className="flex flex-col gap-2"
                        onSubmit={(e) => {
                          const mensagem = String(
                            new FormData(e.currentTarget).get("resposta") ?? "",
                          );
                          void acao(
                            `v1/protocolos/pedidos/${pedido.id}/resposta`,
                            { mensagem },
                            "Complementação respondida",
                            e,
                          );
                        }}
                      >
                        <Textarea
                          id="resposta"
                          name="resposta"
                          label="Resposta recebida na mesa"
                          required
                          maxLength={2000}
                          rows={3}
                        />
                        <div>
                          <Button type="submit" variant="secondary" size="sm" loading={pending}>
                            Registrar resposta
                          </Button>
                        </div>
                      </form>
                    )}
                    <form
                      className="flex flex-col gap-2"
                      onSubmit={(e) => {
                        const mensagem = String(new FormData(e.currentTarget).get("nota") ?? "");
                        void acao(
                          `v1/protocolos/pedidos/${pedido.id}/nota`,
                          { mensagem },
                          "Nota registrada",
                          e,
                        );
                      }}
                    >
                      <Textarea
                        id="nota"
                        name="nota"
                        label="Nota interna (não aparece para o cidadão)"
                        required
                        maxLength={2000}
                        rows={2}
                      />
                      <div>
                        <Button type="submit" variant="secondary" size="sm" loading={pending}>
                          Anotar
                        </Button>
                      </div>
                    </form>
                  </div>

                  <form
                    className="flex flex-col gap-3 rounded-xl border border-surface-border p-4"
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
                    <p className="text-sm font-semibold text-foreground">Concluir o pedido</p>
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
                    <Textarea
                      id="justificativa"
                      name="justificativa"
                      label="Justificativa *"
                      required
                      maxLength={4000}
                      rows={3}
                    />
                    <div>
                      <Button type="submit" size="sm" loading={pending}>
                        Concluir
                      </Button>
                    </div>
                  </form>

                  {(pedido.campos ?? []).some((campo) => CAMPO_EDITAVEL.has(campo.tipo)) && (
                    <details className="rounded-xl border border-surface-border p-4 lg:col-span-2">
                      <summary className="cursor-pointer text-sm font-semibold text-foreground">
                        Corrigir dados do pedido
                      </summary>
                      <form
                        className="mt-3 grid gap-3 sm:grid-cols-2"
                        onSubmit={(e) => {
                          const dados = new FormData(e.currentTarget);
                          const respostas: Record<string, unknown> = {
                            ...(pedido.respostas ?? {}),
                          };
                          for (const campo of pedido.campos ?? []) {
                            if (!CAMPO_EDITAVEL.has(campo.tipo)) continue;
                            respostas[campo.id] = String(dados.get(`campo-${campo.id}`) ?? "");
                          }
                          void acao(
                            `v1/protocolos/pedidos/${pedido.id}/editar`,
                            { justificativa: String(dados.get("justificativa") ?? ""), respostas },
                            "Pedido atualizado",
                          );
                          e.preventDefault();
                        }}
                      >
                        {(pedido.campos ?? [])
                          .filter((campo) => CAMPO_EDITAVEL.has(campo.tipo))
                          .map((campo) => (
                            <Input
                              key={campo.id}
                              id={`campo-${campo.id}`}
                              name={`campo-${campo.id}`}
                              label={campo.rotulo}
                              defaultValue={
                                mostrar(pedido.respostas?.[campo.id]) === "—"
                                  ? ""
                                  : mostrar(pedido.respostas?.[campo.id])
                              }
                            />
                          ))}
                        <div className="flex flex-col gap-3 sm:col-span-2 sm:flex-row sm:items-end">
                          <Input
                            id="justificativa-edicao"
                            name="justificativa"
                            label="Justificativa da edição"
                            required
                            maxLength={4000}
                          />
                          <Button type="submit" variant="secondary" loading={pending}>
                            Salvar edição
                          </Button>
                        </div>
                      </form>
                    </details>
                  )}
                </CardContent>
              </Card>
            )}
          </>
        )}
      </DataState>
    </div>
  );
}

function Info({
  icone,
  titulo,
  children,
}: {
  icone: ReactNode;
  titulo: string;
  children: ReactNode;
}) {
  return (
    <div className="flex items-start gap-3 rounded-xl border border-surface-border bg-surface p-4 text-sm shadow-sm">
      <span
        className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-primary"
        aria-hidden
      >
        {icone}
      </span>
      <div className="min-w-0">
        <p className="text-xs font-medium text-muted">{titulo}</p>
        {children}
      </div>
    </div>
  );
}

const CAMPO_EDITAVEL = new Set([
  "text",
  "textarea",
  "number",
  "email",
  "phone",
  "date",
  "cpf",
  "cnpj",
  "cep",
  "url",
]);

function fmtDia(iso: string) {
  const data = new Date(iso);
  if (Number.isNaN(data.getTime())) return iso;
  return data.toLocaleDateString("pt-BR", { timeZone: "America/Cuiaba" });
}

function fmtDataIso(dia: string) {
  const [ano, mes, d] = dia.split("-");
  return ano && mes && d ? `${d}/${mes}/${ano}` : dia;
}

function rotuloHorario(status: string) {
  if (status === "confirmado") return "confirmado";
  if (status === "agendado") return "agendado";
  if (status === "falta") return "com falta";
  return status;
}

function responsavelVisivel(
  pedido: Pedido,
  arvore: OrgTree[] | undefined,
  pessoas: UserAdmin[] | undefined,
) {
  if (pedido.responsavel_id) return nomeResponsavel(pedido.responsavel_id, arvore, pessoas);
  if (pedido.departamento_id) return nomeResponsavel(pedido.departamento_id, arvore, pessoas);
  if (pedido.unidade_nome) return pedido.unidade_nome;
  return nomeResponsavel(pedido.unidade_id, arvore, pessoas);
}

function nomeResponsavel(
  id: string | undefined,
  arvore: OrgTree[] | undefined,
  pessoas: UserAdmin[] | undefined,
) {
  if (!id) return "Sem responsável";
  for (const entidade of arvore ?? []) {
    for (const unidade of entidade.unidades) {
      if (unidade.id === id)
        return unidade.sigla ? `${unidade.sigla} · ${unidade.nome}` : unidade.nome;
      for (const departamento of unidade.departamentos) {
        if (departamento.id === id) return `${unidade.nome} · ${departamento.nome}`;
      }
    }
  }
  const pessoa = pessoas?.find((item) => item.id === id);
  if (pessoa) return pessoa.display_name || pessoa.username;
  return "Responsável definido";
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
