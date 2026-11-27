"use client";

import Link from "next/link";
import { useEffect, useState, type FormEvent, type ReactNode } from "react";

import {
  ArrowLeft,
  Building2,
  CalendarClock,
  CheckCircle2,
  Circle,
  Clock,
  Coins,
  FileText,
  Phone,
} from "lucide-react";

import {
  CampoInput,
  oculto,
  respostasVisiveis,
  type Campo,
} from "@/components/protocolos/CamposDoPedido";
import { HorarioLivre } from "@/components/protocolos/HorarioLivre";
import {
  BOTAO_PRIMARIO,
  BOTAO_SECUNDARIO,
  CAMPO,
  CARTAO,
  Rotulado,
  TEXTO_FORTE,
  TEXTO_FRACO,
} from "@/components/protocolos/ui";
import { ApiError } from "@/lib/api/client";
import { publicFetch, publicPost } from "@/lib/api/publicClient";
import { reais } from "@/lib/moeda";
import type { UploadTicket } from "@/lib/plataforma/types";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";

interface Orientacao {
  titulo: string;
  texto: string;
  obrigatorio: boolean;
}

interface CartaItem {
  servico: {
    id: string;
    nome: string;
    descricao: string;
    categoria?: string;
    unidade_nome?: string;
    exige_agendamento: boolean;
    prazo_dias: number;
    taxa_centavos: number;
    requisitos?: Orientacao[];
    documentos?: Orientacao[];
    detalhes?: {
      quem_pode?: string;
      passo_a_passo?: string;
      contato?: string;
      tempo_estimado?: string;
      descricao_taxa?: string;
    };
  };
  formulario: { titulo: string; instrucao: string; campos: Campo[] };
}

export function AbrirPedido({ id, pedirInicial = false }: { id: string; pedirInicial?: boolean }) {
  const [item, setItem] = useState<CartaItem | null>(null);
  const [erro, setErro] = useState("");
  const [numero, setNumero] = useState("");
  const [respostas, setRespostas] = useState<Record<string, unknown>>({});
  const [pedir, setPedir] = useState(pedirInicial);
  const [enviando, setEnviando] = useState(false);
  const [subindo, setSubindo] = useState(false);

  useEffect(() => {
    let vivo = true;
    publicFetch<CartaItem>(`v1/protocolos/carta/${id}`)
      .then((carta) => {
        if (vivo) setItem(carta);
      })
      .catch((e: unknown) => {
        if (vivo) setErro(e instanceof ApiError ? e.message : "Serviço indisponível.");
      });
    return () => {
      vivo = false;
    };
  }, [id]);

  async function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!item) return;
    setEnviando(true);
    setErro("");
    const dados = new FormData(e.currentTarget);
    const preenchido = respostasVisiveis(item.formulario.campos ?? [], respostas);
    try {
      const pedido = await publicPost<{ numero: string }>(`v1/protocolos/carta/${id}/pedido`, {
        servico_id: item.servico.id,
        nome: String(dados.get("nome") ?? "").trim(),
        cpf: String(dados.get("cpf") ?? "").trim(),
        email: String(dados.get("email") ?? "").trim(),
        telefone: String(dados.get("telefone") ?? "").trim(),
        data: String(dados.get("data") ?? ""),
        hora: String(dados.get("hora") ?? ""),
        respostas: preenchido,
      });
      setNumero(pedido.numero);
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : "Não foi possível abrir o pedido.");
    } finally {
      setEnviando(false);
    }
  }

  async function enviarArquivo(campo: string, arquivo: File) {
    setSubindo(true);
    setErro("");
    try {
      const ticket = await publicPost<UploadTicket>(`v1/protocolos/carta/${id}/arquivo`, {
        nome: arquivo.name,
        content_type: mimeOf(arquivo),
      });
      await putToTicket(ticket, arquivo);
      setRespostas((atual) => ({ ...atual, [campo]: ticket.object_key }));
    } catch (err) {
      setErro(
        err instanceof ApiError || err instanceof Error
          ? err.message
          : "Não foi possível enviar o arquivo.",
      );
    } finally {
      setSubindo(false);
    }
  }

  if (numero) {
    return (
      <div className="mx-auto flex w-full max-w-xl flex-col px-4 py-14 sm:px-6">
        <div className={`${CARTAO} flex flex-col items-center gap-4 p-10 text-center`}>
          <span className="flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-[#43e97b] to-[#38f9d7] text-white shadow-[0_4px_15px_rgba(67,233,123,0.35)]">
            <CheckCircle2 className="h-9 w-9" aria-hidden />
          </span>
          <h1 className={`text-2xl font-bold ${TEXTO_FORTE}`}>Pedido registrado</h1>
          <p className="rounded-lg bg-indigo-50 px-5 py-2 font-mono text-2xl font-bold tracking-wide text-indigo-600 dark:bg-indigo-500/15 dark:text-indigo-300">
            {numero}
          </p>
          <p className={`text-sm ${TEXTO_FRACO}`}>
            Guarde este número. A consulta pública não mostra CPF nem as respostas.
          </p>
          <Link href="/carta" className={BOTAO_SECUNDARIO}>
            <ArrowLeft className="h-4 w-4" aria-hidden />
            Voltar à carta
          </Link>
        </div>
      </div>
    );
  }

  const requisitos = item?.servico.requisitos ?? [];
  const documentos = item?.servico.documentos ?? [];

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 px-4 py-10 sm:px-6">
      <div className="flex flex-wrap items-center gap-4">
        <Link href="/carta" className={BOTAO_SECUNDARIO}>
          <ArrowLeft className="h-4 w-4" aria-hidden />
          Voltar
        </Link>
        <div className="min-w-0">
          <h1 className={`text-2xl font-bold sm:text-3xl ${TEXTO_FORTE}`}>
            {item?.servico.nome ?? "Abrir pedido"}
          </h1>
          {item?.servico.categoria && (
            <span className="mt-1 inline-block rounded-full bg-blue-100 px-3 py-0.5 text-xs font-medium text-blue-800 dark:bg-sky-400/15 dark:text-sky-200">
              {item.servico.categoria}
            </span>
          )}
        </div>
      </div>

      {erro && (
        <p
          role="alert"
          className="rounded-xl border border-l-4 border-red-200 border-l-red-500 bg-red-50 px-5 py-4 text-sm text-red-700 dark:border-red-500/30 dark:border-l-red-400 dark:bg-red-500/10 dark:text-red-300"
        >
          {erro}
        </p>
      )}

      {item && (
        <div className="grid items-start gap-6 lg:grid-cols-[1fr_360px]">
          <div className="flex min-w-0 flex-col gap-6">
            {!pedir ? (
              <>
                <section className={`${CARTAO} flex flex-col gap-4 p-6`}>
                  <h2 className={`flex items-center gap-2 text-lg font-semibold ${TEXTO_FORTE}`}>
                    <FileText className="h-5 w-5 text-blue-500 dark:text-sky-400" aria-hidden />
                    Sobre o serviço
                  </h2>
                  {item.servico.descricao ? (
                    <p className="whitespace-pre-wrap text-sm leading-relaxed text-gray-700 dark:text-slate-200">
                      {item.servico.descricao}
                    </p>
                  ) : (
                    <p className={`text-sm ${TEXTO_FRACO}`}>Sem descrição.</p>
                  )}
                  {item.formulario.instrucao && (
                    <p className="rounded-lg border border-l-4 border-blue-200 border-l-blue-500 bg-blue-50 p-4 text-sm text-blue-900 dark:border-sky-400/30 dark:border-l-sky-400 dark:bg-sky-400/10 dark:text-sky-100">
                      {item.formulario.instrucao}
                    </p>
                  )}
                </section>
                <ListaDeOrientacao titulo="Requisitos" itens={requisitos} />
                <ListaDeOrientacao titulo="Documentos necessários" itens={documentos} />
                <TextoDoServico
                  titulo="Quem pode solicitar"
                  texto={item.servico.detalhes?.quem_pode}
                />
                <TextoDoServico
                  titulo="Passo a passo"
                  texto={item.servico.detalhes?.passo_a_passo}
                />
                <button
                  type="button"
                  onClick={() => setPedir(true)}
                  className={`${BOTAO_PRIMARIO} self-start px-8 py-3 text-base`}
                >
                  Solicitar
                </button>
              </>
            ) : (
              <form
                onSubmit={(e) => void enviar(e)}
                className={`${CARTAO} flex flex-col gap-8 p-6`}
              >
                <div>
                  <h2 className={`text-xl font-bold ${TEXTO_FORTE}`}>{item.formulario.titulo}</h2>
                  {item.formulario.instrucao && (
                    <p className={`mt-1 text-sm ${TEXTO_FRACO}`}>{item.formulario.instrucao}</p>
                  )}
                </div>

                <section className="flex flex-col gap-4">
                  <Secao numero={1}>Seus dados</Secao>
                  <div className="grid gap-4 sm:grid-cols-2">
                    <Rotulado id="nome" rotulo="Nome completo *" className="sm:col-span-2">
                      <input
                        id="nome"
                        name="nome"
                        required
                        minLength={2}
                        maxLength={200}
                        autoComplete="name"
                        className={CAMPO}
                      />
                    </Rotulado>
                    <Rotulado id="cpf" rotulo="CPF *">
                      <input
                        id="cpf"
                        name="cpf"
                        required
                        inputMode="numeric"
                        maxLength={14}
                        placeholder="000.000.000-00"
                        className={CAMPO}
                      />
                    </Rotulado>
                    <Rotulado id="telefone" rotulo="Telefone">
                      <input
                        id="telefone"
                        name="telefone"
                        type="tel"
                        maxLength={20}
                        autoComplete="tel"
                        className={CAMPO}
                      />
                    </Rotulado>
                    <Rotulado id="email" rotulo="E-mail" className="sm:col-span-2">
                      <input
                        id="email"
                        name="email"
                        type="email"
                        maxLength={200}
                        autoComplete="email"
                        className={CAMPO}
                      />
                    </Rotulado>
                  </div>
                </section>

                {item.servico.exige_agendamento && (
                  <section className="flex flex-col gap-4">
                    <Secao numero={2}>Horário do atendimento</Secao>
                    <HorarioLivre servicoId={item.servico.id} obrigatorio />
                  </section>
                )}

                {(item.formulario.campos ?? []).length > 0 && (
                  <section className="flex flex-col gap-4">
                    <Secao numero={item.servico.exige_agendamento ? 3 : 2}>
                      Informações do pedido
                    </Secao>
                    <div className="flex flex-col gap-4">
                      {(item.formulario.campos ?? []).map((campo) =>
                        oculto(item.formulario.campos, respostas, campo.id) ? null : (
                          <CampoInput
                            key={campo.id}
                            campo={campo}
                            valor={respostas[campo.id]}
                            onChange={(valor) =>
                              setRespostas((atual) => ({ ...atual, [campo.id]: valor }))
                            }
                            onArquivo={(arquivo) => void enviarArquivo(campo.id, arquivo)}
                          />
                        ),
                      )}
                    </div>
                  </section>
                )}

                <div className="flex flex-wrap justify-end gap-3 border-t border-gray-100 pt-5 dark:border-slate-700">
                  <button
                    type="button"
                    onClick={() => setPedir(false)}
                    className={BOTAO_SECUNDARIO}
                  >
                    Voltar aos detalhes
                  </button>
                  <button type="submit" disabled={enviando || subindo} className={BOTAO_PRIMARIO}>
                    <CheckCircle2 className="h-4 w-4" aria-hidden />
                    Protocolar
                  </button>
                </div>
              </form>
            )}
          </div>

          <aside className={`${CARTAO} flex flex-col gap-4 p-6 lg:sticky lg:top-4`}>
            <h2 className={`text-base font-semibold ${TEXTO_FORTE}`}>Resumo</h2>
            <Resumo icone={<Building2 className="h-4 w-4" aria-hidden />} rotulo="Secretaria">
              {item.servico.unidade_nome || "Secretaria"}
            </Resumo>
            <Resumo icone={<Clock className="h-4 w-4" aria-hidden />} rotulo="Prazo">
              {item.servico.prazo_dias > 0
                ? `${item.servico.prazo_dias} dias úteis`
                : "Sem prazo fixo"}
            </Resumo>
            {item.servico.detalhes?.tempo_estimado && (
              <Resumo icone={<Clock className="h-4 w-4" aria-hidden />} rotulo="Tempo estimado">
                {item.servico.detalhes.tempo_estimado}
              </Resumo>
            )}
            <Resumo icone={<Coins className="h-4 w-4" aria-hidden />} rotulo="Taxa">
              {item.servico.taxa_centavos > 0 ? reais(item.servico.taxa_centavos) : "Gratuito"}
              {item.servico.detalhes?.descricao_taxa && (
                <span className={`block text-xs font-normal ${TEXTO_FRACO}`}>
                  {item.servico.detalhes.descricao_taxa}
                </span>
              )}
            </Resumo>
            {item.servico.detalhes?.contato && (
              <Resumo icone={<Phone className="h-4 w-4" aria-hidden />} rotulo="Contato">
                {item.servico.detalhes.contato}
              </Resumo>
            )}
            <Resumo icone={<CalendarClock className="h-4 w-4" aria-hidden />} rotulo="Atendimento">
              {item.servico.exige_agendamento ? "Com horário marcado" : "Sem agendamento"}
            </Resumo>
            {pedir && (requisitos.length > 0 || documentos.length > 0) && (
              <p
                className={`border-t border-gray-100 pt-4 text-xs dark:border-slate-700 ${TEXTO_FRACO}`}
              >
                {requisitos.length} requisito(s) e {documentos.length} documento(s) — veja em
                &quot;Voltar aos detalhes&quot;.
              </p>
            )}
          </aside>
        </div>
      )}
    </div>
  );
}

function Secao({ numero, children }: { numero: number; children: ReactNode }) {
  return (
    <h3 className={`flex items-center gap-3 text-base font-semibold ${TEXTO_FORTE}`}>
      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-blue-500 text-sm font-bold text-white dark:bg-blue-600">
        {numero}
      </span>
      {children}
    </h3>
  );
}

function Resumo({
  icone,
  rotulo,
  children,
}: {
  icone: ReactNode;
  rotulo: string;
  children: ReactNode;
}) {
  return (
    <div className="flex items-start gap-3">
      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-blue-50 text-blue-600 dark:bg-sky-400/15 dark:text-sky-300">
        {icone}
      </span>
      <div>
        <p className={`text-xs font-medium uppercase ${TEXTO_FRACO}`}>{rotulo}</p>
        <p className="text-sm font-medium text-gray-800 dark:text-slate-100">{children}</p>
      </div>
    </div>
  );
}

function ListaDeOrientacao({ titulo, itens }: { titulo: string; itens: Orientacao[] }) {
  if (itens.length === 0) return null;
  return (
    <section className={`${CARTAO} flex flex-col gap-3 p-6`}>
      <h2 className={`text-lg font-semibold ${TEXTO_FORTE}`}>{titulo}</h2>
      <ul className="flex flex-col gap-3">
        {itens.map((item) => (
          <li key={item.titulo} className="flex items-start gap-3 text-sm">
            {item.obrigatorio ? (
              <CheckCircle2
                className="mt-0.5 h-4 w-4 shrink-0 text-emerald-600 dark:text-emerald-400"
                aria-hidden
              />
            ) : (
              <Circle
                className="mt-0.5 h-4 w-4 shrink-0 text-gray-400 dark:text-slate-500"
                aria-hidden
              />
            )}
            <div>
              <p className="font-medium text-gray-800 dark:text-slate-100">
                {item.titulo}
                {item.obrigatorio && (
                  <span className="ml-2 rounded bg-red-50 px-1.5 py-0.5 text-[0.7rem] font-semibold text-red-700 dark:bg-red-500/15 dark:text-red-300">
                    obrigatório
                  </span>
                )}
              </p>
              {item.texto && <p className={TEXTO_FRACO}>{item.texto}</p>}
            </div>
          </li>
        ))}
      </ul>
    </section>
  );
}

function TextoDoServico({ titulo, texto }: { titulo: string; texto?: string }) {
  if (!texto) return null;
  return (
    <section className={`${CARTAO} flex flex-col gap-3 p-6`}>
      <h2 className={`text-lg font-semibold ${TEXTO_FORTE}`}>{titulo}</h2>
      <p className="whitespace-pre-wrap text-sm leading-relaxed text-gray-700 dark:text-slate-200">
        {texto}
      </p>
    </section>
  );
}
