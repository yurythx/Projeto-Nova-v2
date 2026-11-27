"use client";

import Link from "next/link";
import { useMemo, useState, type FormEvent } from "react";

import { ArrowLeft, Plus, UserCheck } from "lucide-react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { useAction } from "@/components/plataforma/useAction";
import {
  BOTAO_PRIMARIO,
  BOTAO_SECUNDARIO,
  Busca,
  CabecalhoDaTela,
  CAMPO,
  CARTAO,
  Pilulas,
  Rotulado,
  SeloAtivo,
  TEXTO_FORTE,
  TEXTO_FRACO,
  Vazio,
} from "@/components/protocolos/ui";
import { Dialog } from "@/components/ui/Dialog";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

interface Responsavel {
  id: string;
  nome: string;
  cpf: string;
  ativo?: boolean;
}

type Filtro = "todos" | "ativo" | "inativo";

export default function ResponsaveisPage() {
  const { can } = usePlataforma();
  const gere = can("protocolos:gerir");
  const { run, pending } = useAction();
  const [novo, setNovo] = useState(false);
  const [ultimo, setUltimo] = useState<Responsavel | null>(null);
  const [filtro, setFiltro] = useState<Filtro>("todos");
  const [busca, setBusca] = useState("");
  const lista = useApiQuery<Responsavel[]>(gere ? "v1/protocolos/responsaveis" : null);

  const todos = useMemo(() => lista.data ?? [], [lista.data]);
  const visiveis = useMemo(() => {
    const termo = busca.trim().toLowerCase();
    const digitos = termo.replace(/\D/g, "");
    return todos.filter((p) => {
      const ativo = p.ativo !== false;
      if (filtro === "ativo" && !ativo) return false;
      if (filtro === "inativo" && ativo) return false;
      if (!termo) return true;
      return (
        p.nome.toLowerCase().includes(termo) ||
        (digitos !== "" && p.cpf.replace(/\D/g, "").includes(digitos))
      );
    });
  }, [todos, filtro, busca]);

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const dados = new FormData(e.currentTarget);
    const criado = await run(
      () =>
        apiClient.post<Responsavel>("v1/protocolos/responsaveis", {
          unidade_id: String(dados.get("unidade_id") ?? ""),
          nome: String(dados.get("nome") ?? "").trim(),
          cpf: String(dados.get("cpf") ?? "").trim(),
        }),
      "Responsável técnico cadastrado",
    );
    if (criado) {
      setUltimo(criado.data);
      setNovo(false);
      void lista.mutate();
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-[1400px] flex-col">
      <CabecalhoDaTela
        icone={UserCheck}
        titulo="Responsáveis Técnicos"
        descricao="Serviço marcado como exclusivo só abre para um CPF ativo neste cadastro. O mesmo CPF não entra duas vezes."
      >
        <Link href="/protocolos" className={BOTAO_SECUNDARIO}>
          <ArrowLeft className="h-4 w-4" aria-hidden />
          Painel
        </Link>
        {gere && (
          <button type="button" onClick={() => setNovo(true)} className={BOTAO_PRIMARIO}>
            <Plus className="h-5 w-5" aria-hidden />
            Novo Responsável
          </button>
        )}
      </CabecalhoDaTela>

      {ultimo && (
        <div
          role="status"
          className="mb-6 rounded-xl border border-l-4 border-emerald-200 border-l-emerald-500 bg-emerald-50 px-5 py-4 text-sm text-emerald-800 dark:border-emerald-500/30 dark:border-l-emerald-400 dark:bg-emerald-500/10 dark:text-emerald-200"
        >
          <strong>{ultimo.nome}</strong> ficou ativo ({ultimo.cpf}). O serviço exclusivo confere o
          CPF na abertura do pedido.
        </div>
      )}

      {gere ? (
        <>
          <div className="mb-6 flex flex-wrap items-center gap-4">
            <Pilulas
              rotulo="Situação do responsável"
              valor={filtro}
              onChange={setFiltro}
              opcoes={[
                { value: "todos", label: "Todos", total: todos.length },
                {
                  value: "ativo",
                  label: "Ativos",
                  total: todos.filter((p) => p.ativo !== false).length,
                },
                {
                  value: "inativo",
                  label: "Inativos",
                  total: todos.filter((p) => p.ativo === false).length,
                },
              ]}
            />
            <Busca valor={busca} onChange={setBusca} placeholder="Pesquisar por nome ou CPF..." />
          </div>

          <DataState
            loading={lista.isLoading}
            error={lista.error}
            onRetry={() => void lista.mutate()}
            empty={false}
          >
            {visiveis.length === 0 ? (
              <Vazio
                titulo="Nenhum responsável técnico"
                texto="Cadastre o profissional que pode abrir os serviços exclusivos."
              />
            ) : (
              <div className={`${CARTAO} overflow-x-auto`}>
                <table className="w-full min-w-[520px] text-left text-sm">
                  <thead className="border-b border-gray-200 bg-slate-50 text-xs uppercase tracking-wide text-gray-500 dark:border-slate-700 dark:bg-slate-900/50 dark:text-slate-400">
                    <tr>
                      <th className="px-6 py-3 font-semibold">Responsável</th>
                      <th className="px-6 py-3 font-semibold">CPF</th>
                      <th className="px-6 py-3 text-right font-semibold">Status</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100 dark:divide-slate-700">
                    {visiveis.map((pessoa) => (
                      <tr
                        key={pessoa.id}
                        className="transition-colors hover:bg-slate-50 dark:hover:bg-slate-700/30"
                      >
                        <td className="px-6 py-4">
                          <span className="flex items-center gap-3">
                            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-blue-100 text-sm font-semibold text-blue-700 dark:bg-sky-400/15 dark:text-sky-300">
                              {pessoa.nome.trim().charAt(0).toUpperCase()}
                            </span>
                            <span className={`font-semibold ${TEXTO_FORTE}`}>{pessoa.nome}</span>
                          </span>
                        </td>
                        <td className={`px-6 py-4 font-mono ${TEXTO_FRACO}`}>{pessoa.cpf}</td>
                        <td className="px-6 py-4 text-right">
                          <SeloAtivo ativo={pessoa.ativo !== false} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </DataState>
        </>
      ) : (
        <Vazio
          titulo="Sem acesso ao cadastro"
          texto="A lista de responsáveis técnicos pede a permissão de gerir o protocolo."
        />
      )}

      <Dialog
        open={novo}
        onClose={() => setNovo(false)}
        title="Novo Responsável Técnico"
        description="O CPF passa a abrir os serviços marcados como exclusivos."
      >
        <form onSubmit={(e) => void salvar(e)} className="flex flex-col gap-4">
          <UnidadeDonaSelect id="unidade-responsavel" permission="protocolos:gerir" exigir />
          <Rotulado id="nome" rotulo="Nome *">
            <input id="nome" name="nome" required minLength={2} maxLength={200} className={CAMPO} />
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
          <div className="flex justify-end gap-3 border-t border-gray-100 pt-4 dark:border-slate-700">
            <button type="button" onClick={() => setNovo(false)} className={BOTAO_SECUNDARIO}>
              Cancelar
            </button>
            <button type="submit" disabled={pending} className={BOTAO_PRIMARIO}>
              Cadastrar
            </button>
          </div>
        </form>
      </Dialog>
    </div>
  );
}
