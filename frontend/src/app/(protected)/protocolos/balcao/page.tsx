"use client";

import Link from "next/link";
import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";

import { ArrowLeft, CalendarClock, CheckCircle2, Clock, Coins, Store } from "lucide-react";

import {
  CampoInput,
  oculto,
  respostasVisiveis,
  type Campo,
} from "@/components/protocolos/CamposDoPedido";
import { HorarioLivre } from "@/components/protocolos/HorarioLivre";
import { useAction } from "@/components/plataforma/useAction";
import {
  BOTAO_PRIMARIO,
  BOTAO_SECUNDARIO,
  Busca,
  CabecalhoDaTela,
  CAMPO,
  CARTAO,
  Rotulado,
  TEXTO_FORTE,
  TEXTO_FRACO,
  Vazio,
} from "@/components/protocolos/ui";
import { ApiError, apiClient } from "@/lib/api/client";
import { publicFetch, publicPost } from "@/lib/api/publicClient";
import { reais } from "@/lib/moeda";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { UploadTicket } from "@/lib/plataforma/types";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";

interface Servico {
  id: string;
  nome: string;
  categoria: string;
  unidade_nome?: string;
}

interface CartaItem {
  servico: {
    id: string;
    nome: string;
    descricao?: string;
    exige_agendamento: boolean;
    prazo_dias?: number;
    taxa_centavos?: number;
  };
  formulario: { titulo: string; instrucao?: string; campos: Campo[] };
}

interface Pedido {
  id: string;
  numero: string;
}

export default function BalcaoPage() {
  const { can } = usePlataforma();
  const { run, pending } = useAction();
  const [servicos, setServicos] = useState<Servico[]>([]);
  const [busca, setBusca] = useState("");
  const [item, setItem] = useState<CartaItem | null>(null);
  const [servicoId, setServicoId] = useState("");
  const [servicoCarregado, setServicoCarregado] = useState("");
  const [respostas, setRespostas] = useState<Record<string, unknown>>({});
  const [pedido, setPedido] = useState<Pedido | null>(null);
  const [erro, setErro] = useState("");
  const [subindo, setSubindo] = useState(false);

  useEffect(() => {
    let vivo = true;
    publicFetch<Servico[]>("v1/protocolos/carta")
      .then((lista) => {
        if (vivo) setServicos(lista ?? []);
      })
      .catch((e: unknown) => {
        if (vivo) setErro(e instanceof ApiError ? e.message : "A carta não pôde ser carregada.");
      });
    return () => {
      vivo = false;
    };
  }, []);

  if (servicoCarregado !== servicoId) {
    setServicoCarregado(servicoId);
    setItem(null);
    setRespostas({});
  }

  useEffect(() => {
    if (!servicoId) return;
    let vivo = true;
    publicFetch<CartaItem>(`v1/protocolos/carta/${servicoId}`)
      .then((carta) => {
        if (vivo) setItem(carta);
      })
      .catch((e: unknown) => {
        if (vivo) setErro(e instanceof ApiError ? e.message : "Serviço indisponível.");
      });
    return () => {
      vivo = false;
    };
  }, [servicoId]);

  const visiveis = useMemo(() => {
    const termo = busca.trim().toLowerCase();
    return servicos.filter(
      (s) =>
        !termo || `${s.nome} ${s.categoria} ${s.unidade_nome ?? ""}`.toLowerCase().includes(termo),
    );
  }, [servicos, busca]);

  async function enviarArquivo(campo: string, arquivo: File) {
    setSubindo(true);
    setErro("");
    try {
      const ticket = await publicPost<UploadTicket>(`v1/protocolos/carta/${servicoId}/arquivo`, {
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

  async function abrir(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!item) return;
    setErro("");
    const dados = new FormData(e.currentTarget);
    const criado = await run(
      () =>
        apiClient.post<Pedido>("v1/protocolos/balcao", {
          servico_id: item.servico.id,
          nome: String(dados.get("nome") ?? "").trim(),
          cpf: String(dados.get("cpf") ?? "").trim(),
          email: String(dados.get("email") ?? "").trim(),
          telefone: String(dados.get("telefone") ?? "").trim(),
          prioridade: String(dados.get("prioridade") ?? "normal"),
          data: String(dados.get("data") ?? ""),
          hora: String(dados.get("hora") ?? ""),
          respostas: respostasVisiveis(item.formulario.campos ?? [], respostas),
        }),
      "Pedido aberto no balcão",
    );
    if (criado) setPedido(criado.data);
  }

  function novoAtendimento() {
    setPedido(null);
    setServicoId("");
    setErro("");
  }

  return (
    <div className="mx-auto flex w-full max-w-[1400px] flex-col">
      <CabecalhoDaTela
        icone={Store}
        titulo="Balcão de Atendimento"
        descricao="Abre o pedido em nome do cidadão. A sessão precisa de protocolos:abrir na unidade do serviço."
      >
        <Link href="/protocolos" className={BOTAO_SECUNDARIO}>
          <ArrowLeft className="h-4 w-4" aria-hidden />
          Painel
        </Link>
      </CabecalhoDaTela>

      {erro && (
        <p
          role="alert"
          className="mb-6 rounded-xl border border-l-4 border-red-200 border-l-red-500 bg-red-50 px-5 py-4 text-sm text-red-700 dark:border-red-500/30 dark:border-l-red-400 dark:bg-red-500/10 dark:text-red-300"
        >
          {erro}
        </p>
      )}

      {pedido ? (
        <div
          className={`${CARTAO} mx-auto flex w-full max-w-xl flex-col items-center gap-4 p-10 text-center`}
        >
          <span className="flex h-16 w-16 items-center justify-center rounded-full bg-gradient-to-br from-[#43e97b] to-[#38f9d7] text-white shadow-[0_4px_15px_rgba(67,233,123,0.35)]">
            <CheckCircle2 className="h-9 w-9" aria-hidden />
          </span>
          <h2 className={`text-xl font-bold ${TEXTO_FORTE}`}>Protocolo registrado</h2>
          <p className="rounded-lg bg-indigo-50 px-5 py-2 font-mono text-2xl font-bold tracking-wide text-indigo-600 dark:bg-indigo-500/15 dark:text-indigo-300">
            {pedido.numero}
          </p>
          <p className={`text-sm ${TEXTO_FRACO}`}>
            O número fica com o cidadão. A ficha da mesa mostra CPF, respostas e o arquivo.
          </p>
          <div className="flex flex-wrap justify-center gap-3">
            <Link href={`/protocolos/${pedido.id}`} className={BOTAO_PRIMARIO}>
              Abrir a ficha
            </Link>
            <button type="button" onClick={novoAtendimento} className={BOTAO_SECUNDARIO}>
              Novo atendimento
            </button>
          </div>
        </div>
      ) : !can("protocolos:abrir") ? (
        <Vazio
          titulo="Sem acesso ao balcão"
          texto="Abrir pedido em nome do cidadão pede a permissão protocolos:abrir."
        />
      ) : (
        <div className="grid items-start gap-6 lg:grid-cols-[380px_1fr]">
          {/* Serviços */}
          <section className={`${CARTAO} flex flex-col gap-4 p-5 lg:sticky lg:top-4`}>
            <TituloDaEtapa numero={1}>Escolha o serviço</TituloDaEtapa>
            <Busca valor={busca} onChange={setBusca} placeholder="Pesquisar serviço..." />
            <ul
              className="flex max-h-[60vh] flex-col gap-2 overflow-y-auto pr-1"
              aria-label="Serviços"
            >
              {visiveis.map((servico) => {
                const ativo = servico.id === servicoId;
                return (
                  <li key={servico.id}>
                    <button
                      type="button"
                      aria-pressed={ativo}
                      onClick={() => setServicoId(servico.id)}
                      className={`w-full rounded-lg border border-l-4 px-4 py-3 text-left transition-all ${
                        ativo
                          ? "border-blue-200 border-l-blue-500 bg-blue-50 dark:border-sky-400/30 dark:border-l-sky-400 dark:bg-sky-400/10"
                          : "border-gray-200 border-l-transparent hover:border-gray-300 hover:bg-gray-50 dark:border-slate-700 dark:hover:border-slate-500 dark:hover:bg-slate-700/40"
                      }`}
                    >
                      <span
                        className={`block text-sm font-semibold ${
                          ativo ? "text-blue-700 dark:text-sky-200" : TEXTO_FORTE
                        }`}
                      >
                        {servico.nome}
                      </span>
                      <span className={`mt-0.5 block text-xs ${TEXTO_FRACO}`}>
                        {[servico.categoria, servico.unidade_nome].filter(Boolean).join(" • ") ||
                          "Serviço da carta"}
                      </span>
                    </button>
                  </li>
                );
              })}
              {visiveis.length === 0 && (
                <li className={`px-2 py-6 text-center text-sm ${TEXTO_FRACO}`}>
                  Nenhum serviço encontrado.
                </li>
              )}
            </ul>
          </section>

          {/* Pedido */}
          {!servicoId ? (
            <Vazio
              titulo="Selecione um serviço"
              texto="O formulário do serviço aparece aqui para preencher com o cidadão."
            />
          ) : !item ? (
            <div className={`${CARTAO} p-10 text-center text-sm ${TEXTO_FRACO}`}>
              Carregando o formulário…
            </div>
          ) : (
            <form onSubmit={(e) => void abrir(e)} className={`${CARTAO} flex flex-col gap-8 p-6`}>
              <div className="flex flex-col gap-2 rounded-lg bg-slate-50 p-4 dark:bg-slate-900/50">
                <h2 className={`text-lg font-bold ${TEXTO_FORTE}`}>{item.servico.nome}</h2>
                {item.servico.descricao && (
                  <p className={`text-sm ${TEXTO_FRACO}`}>{item.servico.descricao}</p>
                )}
                <div className="flex flex-wrap gap-2 pt-1">
                  {(item.servico.prazo_dias ?? 0) > 0 && (
                    <Chip icone={<Clock className="h-3.5 w-3.5" aria-hidden />}>
                      Prazo de {item.servico.prazo_dias} dias úteis
                    </Chip>
                  )}
                  <Chip icone={<Coins className="h-3.5 w-3.5" aria-hidden />}>
                    {(item.servico.taxa_centavos ?? 0) > 0
                      ? `Taxa ${reais(item.servico.taxa_centavos ?? 0)}`
                      : "Gratuito"}
                  </Chip>
                  {item.servico.exige_agendamento && (
                    <Chip icone={<CalendarClock className="h-3.5 w-3.5" aria-hidden />}>
                      Com horário marcado
                    </Chip>
                  )}
                </div>
              </div>

              <section className="flex flex-col gap-4">
                <TituloDaEtapa numero={2}>Dados do Cidadão</TituloDaEtapa>
                <div className="grid gap-4 sm:grid-cols-2">
                  <Rotulado id="nome" rotulo="Nome completo *" className="sm:col-span-2">
                    <input
                      id="nome"
                      name="nome"
                      required
                      minLength={2}
                      maxLength={200}
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
                      placeholder="(66) 90000-0000"
                      className={CAMPO}
                    />
                  </Rotulado>
                  <Rotulado id="email" rotulo="E-mail" className="sm:col-span-2">
                    <input id="email" name="email" type="email" maxLength={200} className={CAMPO} />
                  </Rotulado>
                </div>
              </section>

              <section className="flex flex-col gap-4">
                <TituloDaEtapa numero={3}>Atendimento</TituloDaEtapa>
                <div className="grid gap-4 sm:grid-cols-3">
                  <Rotulado id="prioridade" rotulo="Prioridade">
                    <select
                      id="prioridade"
                      name="prioridade"
                      defaultValue="normal"
                      className={CAMPO}
                    >
                      <option value="baixa">Baixa</option>
                      <option value="normal">Normal</option>
                      <option value="alta">Alta</option>
                      <option value="urgente">Urgente</option>
                    </select>
                  </Rotulado>
                  {item.servico.exige_agendamento && (
                    <div className="sm:col-span-2">
                      <HorarioLivre servicoId={item.servico.id} obrigatorio={false} />
                    </div>
                  )}
                </div>
              </section>

              {(item.formulario.campos ?? []).length > 0 && (
                <section className="flex flex-col gap-4">
                  <TituloDaEtapa numero={4}>{item.formulario.titulo || "Formulário"}</TituloDaEtapa>
                  {item.formulario.instrucao && (
                    <p className={`text-sm ${TEXTO_FRACO}`}>{item.formulario.instrucao}</p>
                  )}
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

              <div className="flex justify-end gap-3 border-t border-gray-100 pt-5 dark:border-slate-700">
                <button type="button" onClick={() => setServicoId("")} className={BOTAO_SECUNDARIO}>
                  Trocar serviço
                </button>
                <button type="submit" disabled={pending || subindo} className={BOTAO_PRIMARIO}>
                  <CheckCircle2 className="h-4 w-4" aria-hidden />
                  Protocolar
                </button>
              </div>
            </form>
          )}
        </div>
      )}
    </div>
  );
}

function TituloDaEtapa({ numero, children }: { numero: number; children: ReactNode }) {
  return (
    <h2 className={`flex items-center gap-3 text-base font-semibold ${TEXTO_FORTE}`}>
      <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-blue-500 text-sm font-bold text-white dark:bg-blue-600">
        {numero}
      </span>
      {children}
    </h2>
  );
}

function Chip({ icone, children }: { icone: ReactNode; children: ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5 rounded-full border border-gray-200 bg-surface px-3 py-1 text-xs font-medium text-gray-600 dark:border-slate-600 dark:text-slate-300">
      {icone}
      {children}
    </span>
  );
}
