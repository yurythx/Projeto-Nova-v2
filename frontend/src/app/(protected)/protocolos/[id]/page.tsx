"use client";

import { useRouter } from "next/navigation";
import { Suspense, useState, type FormEvent, type ReactNode } from "react";

import {
  ArrowLeft,
  CalendarDays,
  CheckCircle2,
  Download,
  FileText,
  History,
  MessageSquareWarning,
  Pencil,
  Printer,
  User,
  Zap,
} from "lucide-react";

import { DataState } from "@/components/plataforma/DataState";
import { useIdDaRota } from "@/components/plataforma/PainelDoQuadro";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import {
  BOTAO_PRIMARIO,
  BOTAO_SECUNDARIO,
  ModalDeAcao,
  type AcaoAberta,
} from "@/components/protocolos/AcoesDoPedido";
import {
  dataCurta,
  FINAIS,
  gradienteDaPrioridade,
  gradienteDoStatus,
  nomeDaEtapa,
  rotuloDaPrioridade,
  rotuloDoStatus,
  seloDoPrazo,
  textoDoPrazo,
} from "@/components/protocolos/estilo";
import { Dialog } from "@/components/ui/Dialog";
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

interface Etapa {
  id: string;
  ordem: number;
  status: string;
  prazo_dias: number;
}

interface Ficha {
  pedido: Pedido;
  historico: Historico[];
  agendamento?: { data: string; hora: string; status: string };
  etapas?: Etapa[];
}

const ROTULO_ACAO: Record<string, string> = {
  criado: "Protocolo Criado",
  avanco: "Encaminhado para a Próxima Etapa",
  complemento: "Complementação Solicitada",
  resposta: "Complementação Respondida",
  nota: "Nota Interna",
  edicao: "Dados Editados",
  conclusao: "Protocolo Concluído",
  cancelado: "Protocolo Cancelado",
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

export default function FichaPage() {
  return (
    <Suspense fallback={null}>
      <FichaConteudo />
    </Suspense>
  );
}

/** Ficha do protocolo, como o ProtocoloDetalhes do Protocolo Digital. */
export function FichaConteudo() {
  const id = useIdDaRota();
  const router = useRouter();
  const { can } = usePlataforma();
  const ficha = useApiQuery<Ficha>(id ? `v1/protocolos/pedidos/${id}` : null);
  const arvore = useApiQuery<OrgTree[]>(id ? "v1/iam/org-tree" : null);
  const pessoas = useApiPage<UserAdmin>(
    can("users:read") ? withQuery("v1/users", { active: true, page_size: 50 }) : null,
  );
  const { run, pending } = useAction();
  const [acao, setAcao] = useState<AcaoAberta | null>(null);
  const [editando, setEditando] = useState(false);

  const pedido = ficha.data?.pedido;
  const etapas = ficha.data?.etapas ?? [];
  const agendamento = ficha.data?.agendamento;
  const trata = Boolean(can("protocolos:tratar") && pedido && !FINAIS.has(pedido.status));
  const complementando = pedido?.status === "em_complementacao";

  async function baixar(campo: string) {
    const link = await run(() =>
      apiClient.get<{ url: string }>(
        `v1/protocolos/pedidos/${id}/arquivo?campo=${encodeURIComponent(campo)}`,
      ),
    );
    if (link?.data.url) window.open(link.data.url, "_blank", "noopener,noreferrer");
  }

  async function postar(caminho: string, corpo: unknown, aviso: string) {
    const ok = await run(() => apiClient.post(caminho, corpo), aviso);
    if (ok) void ficha.mutate();
    return ok;
  }

  async function salvarEdicao(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!pedido) return;
    const dados = new FormData(e.currentTarget);
    const respostas: Record<string, unknown> = { ...(pedido.respostas ?? {}) };
    for (const campo of pedido.campos ?? []) {
      if (!CAMPO_EDITAVEL.has(campo.tipo)) continue;
      respostas[campo.id] = String(dados.get(`campo-${campo.id}`) ?? "");
    }
    const ok = await postar(
      `v1/protocolos/pedidos/${pedido.id}/editar`,
      { justificativa: String(dados.get("justificativa") ?? ""), respostas },
      "Dados do protocolo atualizados",
    );
    if (ok) setEditando(false);
  }

  return (
    <div className="mx-auto w-full max-w-[1400px] [&_svg]:shrink-0">
      <DataState
        loading={ficha.isLoading}
        error={ficha.error}
        onRetry={() => void ficha.mutate()}
        empty={!pedido}
        emptyTitle="Protocolo fora da sua mesa"
      >
        {pedido && (
          <>
            {/* Cabeçalho */}
            <div className="sticky top-0 z-30 mb-8 flex flex-wrap items-center gap-4 border-b border-gray-200/50 bg-gray-50/90 py-4 backdrop-blur dark:border-surface-border dark:bg-background/90">
              <button
                type="button"
                onClick={() => router.push("/protocolos")}
                className="flex items-center gap-2 rounded-lg border border-gray-300 bg-gray-100 px-4 py-3 text-sm font-medium text-gray-700 transition-colors hover:border-gray-400 hover:bg-gray-200 dark:bg-white/10 dark:text-foreground"
              >
                <ArrowLeft className="h-5 w-5" aria-hidden />
                Voltar
              </button>
              <h1 className="text-3xl font-bold text-gray-800 dark:text-foreground">
                Protocolo #{pedido.numero}
              </h1>
            </div>

            <div className="grid gap-8 xl:grid-cols-[1fr_400px]">
              <div className="flex min-w-0 flex-col gap-8">
                {/* Fluxo do processo */}
                {etapas.length > 0 && (
                  <Cartao>
                    <TituloCartao>Fluxo do Processo</TituloCartao>
                    <div className="relative mt-4 flex min-h-[80px] justify-between overflow-x-auto px-4 before:absolute before:left-8 before:right-8 before:top-4 before:z-0 before:h-0.5 before:bg-gray-200">
                      {etapas.map((etapa, indice) => {
                        const atual = etapas.findIndex((e) => e.id === pedido.etapa_id);
                        const ativa = etapa.id === pedido.etapa_id;
                        const feita = atual >= 0 && indice < atual;
                        return (
                          <div
                            key={etapa.id}
                            className="relative z-10 flex min-w-[100px] flex-1 flex-col items-center"
                          >
                            <span
                              className={`mb-2 flex h-8 w-8 items-center justify-center rounded-full border-2 text-sm font-semibold shadow-[0_2px_4px_rgba(0,0,0,0.05)] transition-all ${
                                ativa
                                  ? "border-blue-500 bg-blue-500 text-white"
                                  : feita
                                    ? "border-emerald-500 bg-emerald-500 text-white"
                                    : "border-gray-200 bg-white text-gray-500"
                              }`}
                            >
                              {feita ? (
                                <CheckCircle2 className="h-5 w-5" aria-hidden />
                              ) : (
                                indice + 1
                              )}
                            </span>
                            <span
                              className={`max-w-[100px] text-center text-xs leading-tight ${
                                ativa
                                  ? "font-semibold text-gray-800 dark:text-foreground"
                                  : "text-gray-500 dark:text-muted"
                              }`}
                            >
                              {nomeDaEtapa(etapa.id)}
                            </span>
                          </div>
                        );
                      })}
                    </div>
                    <div className="mt-4 rounded bg-gray-50 p-4 text-sm text-gray-700 dark:bg-white/5 dark:text-foreground">
                      <strong>Etapa Atual: </strong>
                      {nomeDaEtapa(pedido.etapa_id)}
                      {pedido.prazo_em ? ` · prazo até ${dataCurta(pedido.prazo_em)}` : ""}
                    </div>
                  </Cartao>
                )}

                {/* Aviso de complementação */}
                {complementando && pedido.complemento && (
                  <div className="flex items-start gap-3 rounded-xl border border-l-4 border-amber-500 bg-amber-50 p-5 text-sm dark:bg-amber-500/10">
                    <MessageSquareWarning className="h-5 w-5 text-amber-600" aria-hidden />
                    <div>
                      <p className="font-semibold text-amber-700">Complementação Solicitada</p>
                      <p className="text-gray-700 dark:text-foreground">{pedido.complemento}</p>
                    </div>
                  </div>
                )}

                {/* Decisão */}
                {FINAIS.has(pedido.status) && pedido.justificativa && (
                  <div
                    className={`rounded-xl border border-l-4 p-5 text-sm ${
                      pedido.decisao === "desfavoravel"
                        ? "border-red-500 bg-red-50 dark:bg-red-500/10"
                        : "border-emerald-500 bg-emerald-50 dark:bg-emerald-500/10"
                    }`}
                  >
                    <p
                      className={`font-semibold ${
                        pedido.decisao === "desfavoravel" ? "text-red-700" : "text-emerald-700"
                      }`}
                    >
                      {pedido.decisao === "desfavoravel"
                        ? "Decisão Desfavorável (indeferido)"
                        : "Decisão Favorável (deferido)"}
                    </p>
                    <p className="text-gray-700 dark:text-foreground">{pedido.justificativa}</p>
                  </div>
                )}

                {/* Informações do protocolo */}
                <Cartao>
                  <TituloCartao>
                    <FileText className="h-5 w-5" aria-hidden />
                    Informações do Protocolo
                  </TituloCartao>
                  <GradeInfo>
                    <Info rotulo="Número">{pedido.numero}</Info>
                    <Info rotulo="Status">
                      <span
                        className={`inline-flex w-fit items-center rounded-full px-3 py-1 text-xs font-medium uppercase text-white ${gradienteDoStatus(pedido.status)}`}
                      >
                        {rotuloDoStatus(pedido.status)}
                      </span>
                    </Info>
                    <Info rotulo="Prioridade">
                      <span
                        className={`inline-flex w-fit items-center rounded-full px-3 py-1 text-xs font-medium uppercase text-white ${gradienteDaPrioridade(pedido.prioridade)}`}
                      >
                        {rotuloDaPrioridade(pedido.prioridade)}
                      </span>
                    </Info>
                    <Info rotulo="Serviço">{pedido.servico_nome || "N/A"}</Info>
                    <Info rotulo="Departamento">{pedido.unidade_nome || "N/A"}</Info>
                    <Info rotulo="Setor">
                      {nomeNaArvore(pedido.departamento_id, arvore.data) || "N/A"}
                    </Info>
                    <Info rotulo="Responsável">
                      {nomeDaPessoa(pedido.responsavel_id, pessoas.data?.items) || "N/A"}
                    </Info>
                    <Info rotulo="Data de Criação">{dataCurta(pedido.created_at)}</Info>
                    <Info rotulo="Prazo">
                      <span
                        className={`inline-flex w-fit items-center rounded-full px-3 py-1 text-xs font-medium ${seloDoPrazo(pedido.situacao_prazo)}`}
                      >
                        {pedido.prazo_em ? `${dataCurta(pedido.prazo_em)} · ` : ""}
                        {textoDoPrazo(pedido)}
                      </span>
                    </Info>
                  </GradeInfo>
                </Cartao>

                {/* Dados do cidadão */}
                <Cartao>
                  <TituloCartao>
                    <User className="h-5 w-5" aria-hidden />
                    Dados do Cidadão
                  </TituloCartao>
                  <GradeInfo>
                    <Info rotulo="Nome">{pedido.nome}</Info>
                    <Info rotulo="CPF">{pedido.cpf || "N/A"}</Info>
                    <Info rotulo="Email">{pedido.email || "N/A"}</Info>
                    <Info rotulo="Telefone">{pedido.telefone || "N/A"}</Info>
                  </GradeInfo>
                </Cartao>

                {/* Agendamento */}
                {agendamento && (
                  <Cartao>
                    <TituloCartao>
                      <CalendarDays className="h-5 w-5" aria-hidden />
                      Agendamento
                    </TituloCartao>
                    <GradeInfo>
                      <Info rotulo="Data">{dataIso(agendamento.data)}</Info>
                      <Info rotulo="Horário">{agendamento.hora}</Info>
                      <Info rotulo="Situação">{rotuloHorario(agendamento.status)}</Info>
                    </GradeInfo>
                    <p className="mt-3 text-xs text-gray-500 dark:text-muted">
                      Duas faltas bloqueiam novo horário por 30 dias a partir da última.
                    </p>
                  </Cartao>
                )}

                {/* Dados do formulário */}
                <Cartao>
                  <TituloCartao>
                    <span className="flex items-center gap-2">
                      <FileText className="h-5 w-5" aria-hidden />
                      Dados do Formulário
                    </span>
                    <button
                      type="button"
                      onClick={() => window.print()}
                      className="ml-auto flex items-center gap-1 rounded-md bg-blue-500 px-3 py-2 text-xs font-medium text-white transition-colors hover:bg-blue-600"
                    >
                      <Printer className="h-4 w-4" aria-hidden />
                      Imprimir
                    </button>
                  </TituloCartao>
                  <div className="flex flex-col gap-4">
                    {(pedido.campos ?? [])
                      .filter((campo) => campo.tipo !== "page")
                      .map((campo) => {
                        const valor = pedido.respostas?.[campo.id];
                        return (
                          <div
                            key={campo.id}
                            className="flex flex-col gap-1 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-surface-border dark:bg-white/5"
                          >
                            <span className="text-sm font-semibold text-gray-700 dark:text-foreground">
                              {campo.rotulo}
                            </span>
                            {campo.tipo === "file" && chaveDeArquivo(valor) ? (
                              <button
                                type="button"
                                onClick={() => void baixar(campo.id)}
                                className="flex w-fit items-center gap-1 rounded-md bg-blue-500 px-3 py-2 text-xs font-medium text-white transition-colors hover:bg-blue-600"
                              >
                                <Download className="h-4 w-4" aria-hidden />
                                Baixar
                              </button>
                            ) : (
                              <span className="whitespace-pre-wrap break-words text-sm text-gray-500 dark:text-muted">
                                {mostrar(valor)}
                              </span>
                            )}
                          </div>
                        );
                      })}
                  </div>
                </Cartao>
              </div>

              {/* Barra lateral */}
              <div className="flex flex-col gap-6">
                {trata && (
                  <Cartao className="border-t-4 border-t-blue-500">
                    <TituloCartao>
                      <Zap className="h-5 w-5 text-blue-500" aria-hidden />
                      Ações
                    </TituloCartao>
                    <div className="flex flex-col gap-2">
                      {pedido.etapa_id && !complementando && (
                        <button
                          type="button"
                          disabled={pending}
                          className={BOTAO_PRIMARIO}
                          onClick={() =>
                            void postar(
                              `v1/protocolos/pedidos/${pedido.id}/avancar`,
                              {},
                              "Protocolo avançou de etapa",
                            )
                          }
                        >
                          Encaminhar para a próxima etapa
                        </button>
                      )}
                      {complementando ? (
                        <button
                          type="button"
                          className={BOTAO_SECUNDARIO}
                          onClick={() =>
                            setAcao({
                              tipo: "resposta",
                              pedidoId: pedido.id,
                              numero: pedido.numero,
                            })
                          }
                        >
                          Registrar resposta
                        </button>
                      ) : (
                        <button
                          type="button"
                          className={BOTAO_SECUNDARIO}
                          onClick={() =>
                            setAcao({
                              tipo: "complemento",
                              pedidoId: pedido.id,
                              numero: pedido.numero,
                            })
                          }
                        >
                          Solicitar complementação
                        </button>
                      )}
                      <button
                        type="button"
                        className={BOTAO_SECUNDARIO}
                        onClick={() =>
                          setAcao({ tipo: "nota", pedidoId: pedido.id, numero: pedido.numero })
                        }
                      >
                        Adicionar nota
                      </button>
                      {(pedido.campos ?? []).some((c) => CAMPO_EDITAVEL.has(c.tipo)) && (
                        <button
                          type="button"
                          className={BOTAO_SECUNDARIO}
                          onClick={() => setEditando(true)}
                        >
                          <Pencil className="h-4 w-4" aria-hidden />
                          Editar dados
                        </button>
                      )}
                      {agendamento &&
                        (agendamento.status === "agendado" ||
                          agendamento.status === "confirmado") && (
                          <button
                            type="button"
                            disabled={pending}
                            className={BOTAO_SECUNDARIO}
                            onClick={() =>
                              void postar(
                                "v1/protocolos/faltas",
                                { pedido_id: pedido.id },
                                "Falta registrada",
                              )
                            }
                          >
                            Registrar falta
                          </button>
                        )}
                      {!complementando && (
                        <button
                          type="button"
                          className="inline-flex items-center justify-center gap-2 rounded-lg bg-emerald-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-emerald-700"
                          onClick={() =>
                            setAcao({
                              tipo: "concluir",
                              pedidoId: pedido.id,
                              numero: pedido.numero,
                            })
                          }
                        >
                          Concluir protocolo
                        </button>
                      )}
                      <button
                        type="button"
                        className="inline-flex items-center justify-center gap-2 rounded-lg border border-red-200 bg-surface px-4 py-2.5 text-sm font-medium text-red-600 transition-colors hover:bg-red-50 dark:hover:bg-red-500/10"
                        onClick={() =>
                          setAcao({ tipo: "cancelar", pedidoId: pedido.id, numero: pedido.numero })
                        }
                      >
                        Cancelar protocolo
                      </button>
                    </div>
                  </Cartao>
                )}

                {/* Histórico */}
                <Cartao>
                  <TituloCartao>
                    <History className="h-5 w-5" aria-hidden />
                    Histórico de Movimentações
                  </TituloCartao>
                  <div className="flex flex-col gap-4">
                    {(ficha.data?.historico ?? []).length === 0 && (
                      <span className="text-xs text-gray-500">Nenhuma movimentação registrada</span>
                    )}
                    {[...(ficha.data?.historico ?? [])].reverse().map((item) => (
                      <div
                        key={item.id}
                        className="flex gap-4 rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-surface-border dark:bg-white/5"
                      >
                        <span className="flex h-8 w-8 items-center justify-center rounded-full bg-gray-200 text-gray-500 dark:bg-white/10">
                          <History className="h-4 w-4" aria-hidden />
                        </span>
                        <div className="flex min-w-0 flex-1 flex-col gap-1">
                          <span className="text-sm font-medium text-gray-700 dark:text-foreground">
                            {ROTULO_ACAO[item.acao] ?? item.acao}
                          </span>
                          <span className="text-xs text-gray-500 dark:text-muted">
                            {rotuloDoStatus(item.status)} • {fmtDateTime(item.created_at)}
                          </span>
                          {item.descricao &&
                            !FATO_PADRAO.has(item.descricao.trim().toLowerCase()) && (
                              <span className="whitespace-pre-wrap break-words text-xs text-gray-500 dark:text-muted">
                                {item.descricao}
                              </span>
                            )}
                          {(item.anexos ?? []).map((anexo) => (
                            <button
                              key={anexo}
                              type="button"
                              onClick={() => void baixar(anexo)}
                              className="mt-1 inline-flex w-fit items-center gap-1.5 text-[0.8125rem] text-sky-600 hover:underline"
                            >
                              <FileText className="h-3.5 w-3.5" aria-hidden />
                              Anexo desta etapa
                            </button>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                </Cartao>
              </div>
            </div>

            <ModalDeAcao
              acao={acao}
              onFechar={() => setAcao(null)}
              onFeito={() => void ficha.mutate()}
            />

            <Dialog
              open={editando}
              onClose={() => setEditando(false)}
              title="Editar Dados do Protocolo"
              description={`Protocolo #${pedido.numero}`}
              size="lg"
            >
              <form onSubmit={(e) => void salvarEdicao(e)} className="mt-4 flex flex-col gap-4">
                {(pedido.campos ?? [])
                  .filter((campo) => CAMPO_EDITAVEL.has(campo.tipo))
                  .map((campo) => (
                    <div key={campo.id}>
                      <label
                        htmlFor={`campo-${campo.id}`}
                        className="mb-2 block text-sm font-medium text-gray-700 dark:text-muted"
                      >
                        {campo.rotulo}
                      </label>
                      <input
                        id={`campo-${campo.id}`}
                        name={`campo-${campo.id}`}
                        defaultValue={
                          mostrar(pedido.respostas?.[campo.id]) === "—"
                            ? ""
                            : mostrar(pedido.respostas?.[campo.id])
                        }
                        className="w-full rounded-lg border border-gray-300 bg-surface px-4 py-3 text-sm text-foreground focus:border-blue-500 focus:outline-none focus:ring-[3px] focus:ring-blue-500/10"
                      />
                    </div>
                  ))}
                <div>
                  <label
                    htmlFor="justificativa-edicao"
                    className="mb-2 block text-sm font-medium text-gray-700 dark:text-muted"
                  >
                    Justificativa da edição *
                  </label>
                  <textarea
                    id="justificativa-edicao"
                    name="justificativa"
                    required
                    rows={3}
                    maxLength={4000}
                    className="w-full rounded-lg border border-gray-300 bg-surface px-4 py-3 text-sm text-foreground focus:border-blue-500 focus:outline-none focus:ring-[3px] focus:ring-blue-500/10"
                  />
                </div>
                <div className="flex justify-end gap-3">
                  <button
                    type="button"
                    onClick={() => setEditando(false)}
                    className={BOTAO_SECUNDARIO}
                  >
                    Cancelar
                  </button>
                  <button type="submit" disabled={pending} className={BOTAO_PRIMARIO}>
                    Salvar edição
                  </button>
                </div>
              </form>
            </Dialog>
          </>
        )}
      </DataState>
    </div>
  );
}

function Cartao({ className = "", children }: { className?: string; children: ReactNode }) {
  return (
    <section
      className={`rounded-xl border border-slate-200 bg-surface p-6 shadow-[0_4px_6px_-1px_rgba(0,0,0,0.05),0_2px_4px_-1px_rgba(0,0,0,0.03)] transition-shadow hover:shadow-[0_10px_15px_-3px_rgba(0,0,0,0.05),0_4px_6px_-2px_rgba(0,0,0,0.03)] dark:border-surface-border ${className}`}
    >
      {children}
    </section>
  );
}

function TituloCartao({ children }: { children: ReactNode }) {
  return (
    <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold text-gray-800 dark:text-foreground">
      {children}
    </h2>
  );
}

function GradeInfo({ children }: { children: ReactNode }) {
  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-4">{children}</div>
  );
}

function Info({ rotulo, children }: { rotulo: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-1">
      <span className="text-xs font-medium uppercase text-gray-500 dark:text-muted">{rotulo}</span>
      <span className="break-words text-sm font-medium text-gray-800 dark:text-foreground">
        {children}
      </span>
    </div>
  );
}

function dataIso(dia: string) {
  const [ano, mes, d] = dia.split("-");
  return ano && mes && d ? `${d}/${mes}/${ano}` : dia;
}

function rotuloHorario(status: string) {
  if (status === "confirmado") return "Confirmado";
  if (status === "agendado") return "Agendado";
  if (status === "falta") return "Falta registrada";
  return status;
}

function nomeNaArvore(id: string | undefined, arvore: OrgTree[] | undefined): string {
  if (!id) return "";
  for (const entidade of arvore ?? []) {
    for (const unidade of entidade.unidades) {
      if (unidade.id === id) return unidade.nome;
      for (const departamento of unidade.departamentos) {
        if (departamento.id === id) return departamento.nome;
      }
    }
  }
  return "";
}

function nomeDaPessoa(id: string | undefined, pessoas: UserAdmin[] | undefined): string {
  if (!id) return "";
  const pessoa = pessoas?.find((item) => item.id === id);
  return pessoa ? pessoa.display_name || pessoa.username : "Responsável definido";
}

function chaveDeArquivo(valor: unknown): boolean {
  return typeof valor === "string" && valor.startsWith("protocolos/");
}

function mostrar(valor: unknown): string {
  if (valor == null || valor === "") return "—";
  if (typeof valor === "boolean") return valor ? "Sim" : "Não";
  if (Array.isArray(valor)) return valor.map(String).join(", ");
  return String(valor);
}
