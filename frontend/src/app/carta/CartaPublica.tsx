"use client";

import Link from "next/link";
import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";

import {
  AlertTriangle,
  Building2,
  CalendarClock,
  CheckCircle2,
  Clock,
  Coins,
  FilePlus2,
  FileText,
  MessageSquareWarning,
  MessagesSquare,
  Search,
  Star,
  XCircle,
} from "lucide-react";

import { fmtDateTime } from "@/components/plataforma/useAction";
import {
  rotuloDaPrioridade,
  rotuloDoStatus,
  seloDaPrioridade,
  seloDoStatus,
} from "@/components/protocolos/estilo";
import {
  BOTAO_PRIMARIO,
  BOTAO_SECUNDARIO,
  CAMPO,
  CARTAO,
  Rotulado,
  TEXTO_FORTE,
  TEXTO_FRACO,
  Vazio,
} from "@/components/protocolos/ui";
import { ApiError } from "@/lib/api/client";
import { publicFetch, publicPost } from "@/lib/api/publicClient";
import { reais } from "@/lib/moeda";
import type { UploadTicket } from "@/lib/plataforma/types";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";

interface Servico {
  id: string;
  nome: string;
  descricao: string;
  categoria: string;
  prazo_dias: number;
  destaque: boolean;
  taxa_centavos: number;
  unidade_id: string;
  unidade_nome?: string;
}

interface Historico {
  id?: string;
  acao: string;
  descricao: string;
  status: string;
  created_at: string;
}

interface Consulta {
  numero: string;
  status: string;
  status_rotulo: string;
  prioridade: string;
  nome: string;
  servico: string;
  unidade: string;
  criado_em: string;
  prazo_em?: string;
  complemento?: string;
  complemento_prazo?: string;
  processo_sei?: string;
  historico: Historico[];
  agendamento?: { data: string; hora: string; status: string };
}

type Modo = "abrir" | "consultar";

export function CartaPublica() {
  const [modo, setModo] = useState<Modo>("abrir");
  const [servicos, setServicos] = useState<Servico[] | null>(null);
  const [filtro, setFiltro] = useState("");
  const [unidade, setUnidade] = useState("");
  const [consulta, setConsulta] = useState<Consulta | null>(null);
  const [erro, setErro] = useState("");
  const [consultando, setConsultando] = useState(false);

  useEffect(() => {
    let vivo = true;
    publicFetch<Servico[]>("v1/protocolos/carta")
      .then((lista) => {
        if (vivo) setServicos(lista);
      })
      .catch((e: unknown) => {
        if (vivo) setErro(e instanceof ApiError ? e.message : "A carta não pôde ser carregada.");
      });
    return () => {
      vivo = false;
    };
  }, []);

  const secretarias = useMemo(() => {
    const mapa = new Map<string, string>();
    for (const servico of servicos ?? [])
      mapa.set(servico.unidade_id, servico.unidade_nome || "Secretaria");
    return [...mapa.entries()]
      .map(([value, label]) => ({ value, label }))
      .sort((a, b) => a.label.localeCompare(b.label, "pt-BR"));
  }, [servicos]);

  const visiveis = useMemo(() => {
    const termo = filtro.trim().toLowerCase();
    return (servicos ?? [])
      .filter((servico) => {
        if (unidade && servico.unidade_id !== unidade) return false;
        if (!termo) return true;
        return `${servico.nome} ${servico.categoria} ${servico.descricao} ${servico.unidade_nome ?? ""}`
          .toLowerCase()
          .includes(termo);
      })
      .sort((a, b) => Number(b.destaque) - Number(a.destaque));
  }, [servicos, filtro, unidade]);

  async function consultar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setErro("");
    setConsulta(null);
    setConsultando(true);
    const numero = String(new FormData(e.currentTarget).get("numero") ?? "").trim();
    try {
      setConsulta(
        await publicFetch<Consulta>(`v1/protocolos/consulta/${encodeURIComponent(numero)}`),
      );
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : "Número não encontrado.");
    } finally {
      setConsultando(false);
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-8 px-4 py-10 sm:px-6">
      {/* Banner */}
      <section className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-[#1e3a8a] via-[#2563eb] to-[#3b82f6] px-6 py-10 text-white shadow-[0_8px_32px_rgba(37,99,235,0.25)] sm:px-10">
        <span
          className="pointer-events-none absolute -right-16 -top-16 h-64 w-64 rounded-full bg-white/10"
          aria-hidden
        />
        <span
          className="pointer-events-none absolute -bottom-20 right-24 h-48 w-48 rounded-full bg-white/5"
          aria-hidden
        />
        <div className="relative flex flex-col gap-4">
          <p className="text-sm font-semibold uppercase tracking-widest text-white/80">
            Carta de serviços
          </p>
          <h1 className="text-3xl font-bold sm:text-4xl">Protocolo Digital</h1>
          <p className="max-w-2xl text-base text-white/90">
            Abra um pedido da carta de serviços ou consulte o andamento pelo número. A consulta não
            mostra CPF, telefone nem as respostas do formulário.
          </p>
          <div
            className="mt-2 flex w-fit gap-1 rounded-xl bg-white/15 p-1 backdrop-blur"
            role="tablist"
            aria-label="O que você quer fazer"
          >
            {(
              [
                { value: "abrir", label: "Abrir protocolo", icone: FilePlus2 },
                { value: "consultar", label: "Consultar protocolo", icone: Search },
              ] as const
            ).map(({ value, label, icone: Icone }) => (
              <button
                key={value}
                type="button"
                role="tab"
                aria-selected={modo === value}
                onClick={() => setModo(value)}
                className={`flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-semibold transition-all ${
                  modo === value
                    ? "bg-white text-blue-700 shadow-md"
                    : "text-white hover:bg-white/10"
                }`}
              >
                <Icone className="h-4 w-4" aria-hidden />
                {label}
              </button>
            ))}
          </div>
        </div>
      </section>

      {erro && (
        <p
          role="alert"
          className="rounded-xl border border-l-4 border-red-200 border-l-red-500 bg-red-50 px-5 py-4 text-sm text-red-700 dark:border-red-500/30 dark:border-l-red-400 dark:bg-red-500/10 dark:text-red-300"
        >
          {erro}
        </p>
      )}

      {modo === "consultar" ? (
        <section className={`${CARTAO} flex flex-col gap-6 p-6 sm:p-8`} aria-label="Consultar">
          <div>
            <h2 className={`text-2xl font-bold ${TEXTO_FORTE}`}>Acompanhe seu protocolo</h2>
            <p className={`mt-1 text-sm ${TEXTO_FRACO}`}>
              Digite o número que você recebeu ao abrir o pedido.
            </p>
          </div>
          <form onSubmit={(e) => void consultar(e)} className="flex flex-col gap-3 sm:flex-row">
            <div className="relative flex-1">
              <Search
                className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-500 dark:text-slate-400"
                aria-hidden
              />
              <input
                id="numero"
                name="numero"
                aria-label="Número do protocolo"
                required
                maxLength={40}
                autoComplete="off"
                placeholder="Ex.: PREF202690001"
                className={`${CAMPO} py-3.5 pl-11 text-base uppercase`}
              />
            </div>
            <button type="submit" disabled={consultando} className={`${BOTAO_PRIMARIO} px-6`}>
              <Search className="h-4 w-4" aria-hidden />
              Consultar
            </button>
          </form>
          {consulta && (
            <ResultadoConsulta consulta={consulta} onAtualizar={setConsulta} onErro={setErro} />
          )}
        </section>
      ) : (
        <section className="flex flex-col gap-5" aria-label="Abrir">
          <div className={`${CARTAO} grid gap-4 p-5 md:grid-cols-[2fr_1fr]`}>
            <div className="relative">
              <Search
                className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-gray-500 dark:text-slate-400"
                aria-hidden
              />
              <input
                type="search"
                aria-label="Buscar serviço"
                placeholder="Buscar por nome, categoria ou secretaria..."
                value={filtro}
                onChange={(e) => setFiltro(e.target.value)}
                className={`${CAMPO} pl-11`}
              />
            </div>
            <select
              aria-label="Secretaria"
              value={unidade}
              onChange={(e) => setUnidade(e.target.value)}
              className={CAMPO}
            >
              <option value="">Todas as secretarias</option>
              {secretarias.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.label}
                </option>
              ))}
            </select>
          </div>

          {servicos && (
            <p className={`text-sm ${TEXTO_FRACO}`}>
              {visiveis.length === 1
                ? "1 serviço disponível"
                : `${visiveis.length} serviços disponíveis`}
            </p>
          )}

          {servicos && visiveis.length === 0 ? (
            <Vazio
              titulo="Nenhum serviço encontrado"
              texto="Tente outra palavra ou escolha todas as secretarias."
            />
          ) : (
            <ul className="grid gap-5 md:grid-cols-2 xl:grid-cols-3">
              {visiveis.map((servico) => (
                <li key={servico.id}>
                  <CartaoDoServico servico={servico} />
                </li>
              ))}
            </ul>
          )}
        </section>
      )}
    </div>
  );
}

function CartaoDoServico({ servico }: { servico: Servico }) {
  return (
    <article
      className={`flex h-full flex-col gap-4 rounded-xl border p-6 shadow-[0_1px_3px_rgba(0,0,0,0.1)] transition-all hover:-translate-y-0.5 hover:shadow-[0_4px_12px_rgba(0,0,0,0.15)] ${
        servico.destaque
          ? "border-amber-400 bg-gradient-to-b from-amber-50 to-white hover:border-amber-500 dark:border-amber-400/60 dark:from-amber-500/10 dark:to-slate-800"
          : "border-gray-200 bg-surface hover:border-blue-500 dark:border-slate-700 dark:hover:border-sky-400"
      }`}
    >
      <header className="flex items-start gap-3">
        <span className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg border border-gray-200 bg-white text-blue-500 dark:border-slate-600 dark:bg-slate-900/60 dark:text-sky-400">
          <FileText className="h-6 w-6" aria-hidden />
        </span>
        <div className="min-w-0 flex-1">
          <h3 className={`text-lg font-bold leading-snug ${TEXTO_FORTE}`}>{servico.nome}</h3>
          <div className="mt-1.5 flex flex-wrap gap-1.5">
            <span className="rounded-full bg-blue-100 px-3 py-0.5 text-xs font-medium text-blue-800 dark:bg-sky-400/15 dark:text-sky-200">
              {servico.categoria || "Serviço da carta"}
            </span>
            {servico.destaque && (
              <span className="inline-flex items-center gap-1 rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-semibold text-amber-800 dark:bg-amber-400/20 dark:text-amber-200">
                <Star className="h-3 w-3" aria-hidden />
                Destaque
              </span>
            )}
          </div>
        </div>
      </header>
      {servico.descricao && (
        <p className={`line-clamp-3 text-sm ${TEXTO_FRACO}`}>{servico.descricao}</p>
      )}
      <div className={`mt-auto flex flex-col gap-1.5 text-xs ${TEXTO_FRACO}`}>
        <span className="flex items-center gap-1.5">
          <Building2 className="h-3.5 w-3.5" aria-hidden />
          {servico.unidade_nome || "Secretaria"}
        </span>
        <span className="flex items-center gap-1.5">
          <Clock className="h-3.5 w-3.5" aria-hidden />
          {servico.prazo_dias > 0 ? `Prazo de ${servico.prazo_dias} dias úteis` : "Sem prazo fixo"}
        </span>
        <span className="flex items-center gap-1.5">
          <Coins className="h-3.5 w-3.5" aria-hidden />
          {servico.taxa_centavos > 0 ? reais(servico.taxa_centavos) : "Gratuito"}
        </span>
      </div>
      <div className="flex gap-2 border-t border-gray-100 pt-4 dark:border-slate-700">
        <Link href={`/carta/${servico.id}`} className={`${BOTAO_SECUNDARIO} flex-1`}>
          Ver detalhes
        </Link>
        <Link href={`/carta/${servico.id}?pedir=1`} className={`${BOTAO_PRIMARIO} flex-1`}>
          Solicitar
        </Link>
      </div>
    </article>
  );
}

/** Ícone e cor da linha do tempo, como o ProtocolTracker original. */
function marcaDoHistorico(acao: string): { cor: string; icone: ReactNode } {
  const classe = "h-4 w-4";
  switch (acao) {
    case "mensagem_cidadao":
    case "mensagem_mesa":
      return {
        cor: "bg-sky-100 text-sky-600 dark:bg-sky-500/20 dark:text-sky-300",
        icone: <MessagesSquare className={classe} aria-hidden />,
      };
    case "criado":
      return {
        cor: "bg-blue-100 text-blue-600 dark:bg-blue-500/20 dark:text-blue-300",
        icone: <FilePlus2 className={classe} aria-hidden />,
      };
    case "conclusao":
      return {
        cor: "bg-emerald-100 text-emerald-600 dark:bg-emerald-500/20 dark:text-emerald-300",
        icone: <CheckCircle2 className={classe} aria-hidden />,
      };
    case "complemento":
      return {
        cor: "bg-orange-100 text-orange-600 dark:bg-orange-500/20 dark:text-orange-300",
        icone: <AlertTriangle className={classe} aria-hidden />,
      };
    case "cancelado":
      return {
        cor: "bg-red-100 text-red-600 dark:bg-red-500/20 dark:text-red-300",
        icone: <XCircle className={classe} aria-hidden />,
      };
    default:
      return {
        cor: "bg-amber-100 text-amber-600 dark:bg-amber-500/20 dark:text-amber-300",
        icone: <Clock className={classe} aria-hidden />,
      };
  }
}

function ResultadoConsulta({
  consulta,
  onAtualizar,
  onErro,
}: {
  consulta: Consulta;
  onAtualizar: (consulta: Consulta) => void;
  onErro: (erro: string) => void;
}) {
  const [enviando, setEnviando] = useState(false);

  async function responder(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setEnviando(true);
    onErro("");
    const dados = new FormData(e.currentTarget);
    const cpf = String(dados.get("cpf") ?? "").trim();
    const mensagem = String(dados.get("mensagem") ?? "").trim();
    const arquivos = dados
      .getAll("arquivos")
      .filter((item): item is File => item instanceof File && item.size > 0);
    try {
      const chaves: string[] = [];
      for (const arquivo of arquivos) {
        const ticket = await publicPost<UploadTicket>(
          `v1/protocolos/consulta/${encodeURIComponent(consulta.numero)}/arquivo`,
          { cpf, nome: arquivo.name, content_type: mimeOf(arquivo) },
        );
        await putToTicket(ticket, arquivo);
        chaves.push(ticket.object_key);
      }
      onAtualizar(
        await publicPost<Consulta>(
          `v1/protocolos/consulta/${encodeURIComponent(consulta.numero)}/resposta`,
          {
            cpf,
            mensagem,
            arquivos: chaves,
          },
        ),
      );
    } catch (err) {
      onErro(
        err instanceof ApiError || err instanceof Error
          ? err.message
          : "Não foi possível enviar a resposta.",
      );
    } finally {
      setEnviando(false);
    }
  }

  async function escrever(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    setEnviando(true);
    onErro("");
    const dados = new FormData(form);
    try {
      onAtualizar(
        await publicPost<Consulta>(
          `v1/protocolos/consulta/${encodeURIComponent(consulta.numero)}/mensagem`,
          {
            cpf: String(dados.get("cpf") ?? "").trim(),
            mensagem: String(dados.get("mensagem") ?? "").trim(),
          },
        ),
      );
      form.reset();
    } catch (err) {
      onErro(
        err instanceof ApiError || err instanceof Error
          ? err.message
          : "Não foi possível enviar a mensagem.",
      );
    } finally {
      setEnviando(false);
    }
  }

  const historico = [...(consulta.historico ?? [])].reverse();
  const aberto = !["concluido", "aprovado", "rejeitado", "cancelado", "em_complementacao"].includes(
    consulta.status,
  );

  return (
    <div className="flex flex-col gap-6 rounded-xl border border-gray-200 p-5 sm:p-6 dark:border-slate-700">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className={`text-xs font-semibold uppercase tracking-wide ${TEXTO_FRACO}`}>
            Protocolo
          </p>
          <p className="font-mono text-2xl font-bold text-indigo-600 dark:text-indigo-300">
            {consulta.numero}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <span
            className={`rounded-full px-4 py-1.5 text-xs font-semibold uppercase text-white ${seloDoStatus(consulta.status)}`}
          >
            {consulta.status_rotulo || rotuloDoStatus(consulta.status)}
          </span>
          <span
            className={`rounded-full px-4 py-1.5 text-xs font-semibold uppercase text-white ${seloDaPrioridade(consulta.prioridade)}`}
          >
            {rotuloDaPrioridade(consulta.prioridade)}
          </span>
        </div>
      </div>

      <dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <Info rotulo="Requerente">{consulta.nome}</Info>
        <Info rotulo="Serviço">{consulta.servico}</Info>
        <Info rotulo="Secretaria">{consulta.unidade}</Info>
        <Info rotulo="Aberto em">{fmtDateTime(consulta.criado_em)}</Info>
        {consulta.prazo_em && <Info rotulo="Prazo">{fmtDateTime(consulta.prazo_em)}</Info>}
        {consulta.processo_sei && (
          <Info rotulo="Processo SEI">
            <span className="font-mono">{consulta.processo_sei}</span>
          </Info>
        )}
        {consulta.agendamento && (
          <Info rotulo="Atendimento">
            <span className="inline-flex items-center gap-1.5">
              <CalendarClock className="h-4 w-4 text-blue-500 dark:text-sky-400" aria-hidden />
              {consulta.agendamento.data} às {consulta.agendamento.hora} ·{" "}
              {consulta.agendamento.status}
            </span>
          </Info>
        )}
      </dl>

      {consulta.complemento && (
        <div className="flex items-start gap-3 rounded-xl border border-l-4 border-amber-500 bg-amber-50 p-5 text-sm dark:bg-amber-500/10">
          <MessageSquareWarning
            className="h-5 w-5 shrink-0 text-amber-600 dark:text-amber-400"
            aria-hidden
          />
          <div>
            <p className="font-semibold text-amber-700 dark:text-amber-300">
              Complementação solicitada
            </p>
            <p className="text-gray-700 dark:text-slate-200">{consulta.complemento}</p>
            {consulta.complemento_prazo && (
              <p className="mt-1 text-xs font-semibold text-amber-700 dark:text-amber-300">
                Responda até {fmtDateTime(consulta.complemento_prazo)}. Sem resposta no prazo, o
                protocolo é cancelado.
              </p>
            )}
          </div>
        </div>
      )}

      {historico.length > 0 && (
        <div className="border-t border-gray-200 pt-6 dark:border-slate-700">
          <h3 className={`mb-4 text-lg font-semibold ${TEXTO_FORTE}`}>Histórico</h3>
          <ol className="flex flex-col">
            {historico.map((item, indice) => {
              const marca = marcaDoHistorico(item.acao);
              return (
                <li key={item.id ?? indice} className="relative flex gap-4 pb-6 last:pb-0">
                  {indice < historico.length - 1 && (
                    <span
                      className="absolute bottom-0 left-4 top-8 w-0.5 bg-gray-200 dark:bg-slate-700"
                      aria-hidden
                    />
                  )}
                  <span
                    className={`relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${marca.cor}`}
                  >
                    {marca.icone}
                  </span>
                  <div className="min-w-0 pt-1">
                    {(item.acao === "mensagem_cidadao" || item.acao === "mensagem_mesa") && (
                      <p className="text-xs font-semibold uppercase text-sky-700 dark:text-sky-300">
                        {item.acao === "mensagem_cidadao"
                          ? "Sua mensagem"
                          : "Mensagem da prefeitura"}
                      </p>
                    )}
                    <p className="whitespace-pre-wrap text-sm font-medium text-gray-800 dark:text-slate-100">
                      {item.descricao}
                    </p>
                    <p className={`text-xs ${TEXTO_FRACO}`}>
                      {rotuloDoStatus(item.status)} · {fmtDateTime(item.created_at)}
                    </p>
                  </div>
                </li>
              );
            })}
          </ol>
        </div>
      )}

      {consulta.status === "em_complementacao" && (
        <form
          onSubmit={(e) => void responder(e)}
          className="flex flex-col gap-4 rounded-xl border border-violet-200 bg-violet-50/50 p-5 dark:border-violet-500/30 dark:bg-violet-500/10"
        >
          <h3 className={`text-base font-semibold ${TEXTO_FORTE}`}>Responder à complementação</h3>
          <Rotulado id="cpf" rotulo="CPF do pedido *">
            <input
              id="cpf"
              name="cpf"
              required
              inputMode="numeric"
              maxLength={14}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id="mensagem" rotulo="Resposta *">
            <textarea
              id="mensagem"
              name="mensagem"
              required
              rows={4}
              maxLength={1000}
              className={`${CAMPO} min-h-[100px] resize-y`}
            />
          </Rotulado>
          <Rotulado id="arquivos" rotulo="Arquivos">
            <input
              id="arquivos"
              name="arquivos"
              type="file"
              multiple
              className={`${CAMPO} file:mr-3 file:rounded-md file:border-0 file:bg-blue-50 file:px-3 file:py-1.5 file:text-sm file:font-medium file:text-blue-700 dark:file:bg-sky-400/15 dark:file:text-sky-300`}
            />
          </Rotulado>
          <button type="submit" disabled={enviando} className={`${BOTAO_PRIMARIO} self-end`}>
            Enviar resposta
          </button>
        </form>
      )}

      {aberto && (
        <form
          onSubmit={(e) => void escrever(e)}
          aria-label="Mensagem à prefeitura"
          className="flex flex-col gap-4 rounded-xl border border-sky-200 bg-sky-50/50 p-5 dark:border-sky-500/30 dark:bg-sky-500/10"
        >
          <div>
            <h3 className={`text-base font-semibold ${TEXTO_FORTE}`}>Mensagem à prefeitura</h3>
            <p className={`text-xs ${TEXTO_FRACO}`}>
              Tire uma dúvida ou acrescente uma informação. A resposta aparece no histórico, e você
              recebe um aviso no contato informado no pedido.
            </p>
          </div>
          <Rotulado id="cpf-mensagem" rotulo="CPF do pedido *">
            <input
              id="cpf-mensagem"
              name="cpf"
              required
              inputMode="numeric"
              maxLength={14}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id="texto-mensagem" rotulo="Mensagem *">
            <textarea
              id="texto-mensagem"
              name="mensagem"
              required
              rows={3}
              maxLength={1000}
              className={`${CAMPO} min-h-[80px] resize-y`}
            />
          </Rotulado>
          <button type="submit" disabled={enviando} className={`${BOTAO_PRIMARIO} self-end`}>
            Enviar mensagem
          </button>
        </form>
      )}
    </div>
  );
}

function Info({ rotulo, children }: { rotulo: string; children: ReactNode }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-slate-50 p-4 dark:border-slate-700 dark:bg-slate-900/40">
      <dt className={`text-xs font-medium uppercase ${TEXTO_FRACO}`}>{rotulo}</dt>
      <dd className="mt-1 break-words text-sm font-medium text-gray-800 dark:text-slate-100">
        {children}
      </dd>
    </div>
  );
}
