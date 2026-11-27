"use client";

import Link from "next/link";
import { useMemo, useState, type FormEvent } from "react";

import { ArrowDown, ArrowLeft, ArrowUp, Copy, Edit, FileText, Plus, Trash2 } from "lucide-react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { useAction } from "@/components/plataforma/useAction";
import { dataCurta } from "@/components/protocolos/estilo";
import {
  BOTAO_PRIMARIO,
  BOTAO_SECUNDARIO,
  Busca,
  CabecalhoDaTela,
  CAMPO,
  CARTAO,
  Marcar,
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

const TIPOS: { value: string; label: string }[] = [
  { value: "text", label: "Texto" },
  { value: "textarea", label: "Texto longo" },
  { value: "number", label: "Número" },
  { value: "email", label: "E-mail" },
  { value: "phone", label: "Telefone" },
  { value: "date", label: "Data" },
  { value: "time", label: "Hora" },
  { value: "url", label: "Endereço web" },
  { value: "password", label: "Senha" },
  { value: "cpf", label: "CPF" },
  { value: "cnpj", label: "CNPJ" },
  { value: "cep", label: "CEP" },
  { value: "select", label: "Lista de opções" },
  { value: "radio", label: "Escolha única" },
  { value: "checkbox", label: "Caixa de seleção" },
  { value: "checkbox_group", label: "Várias escolhas" },
  { value: "boolean", label: "Sim / Não" },
  { value: "file", label: "Arquivo" },
  { value: "map", label: "Ponto no mapa" },
  { value: "map_area", label: "Área no mapa" },
  { value: "ambulante", label: "Ambulante" },
  { value: "page", label: "Quebra de página" },
];

const COM_OPCOES = new Set(["select", "radio", "checkbox_group"]);

interface Campo {
  id: string;
  tipo: string;
  rotulo: string;
  obrigatorio: boolean;
  opcoes?: string[];
  revela?: string[];
}

interface Formulario {
  id: string;
  unidade_id: string;
  titulo: string;
  descricao: string;
  instrucao?: string;
  ativo: boolean;
  campos: Campo[] | null;
  created_at?: string;
  updated_at?: string;
}

interface ServicoResumo {
  id: string;
  nome: string;
  formulario_id: string;
}

type Edicao = { form: Formulario | null; copia: boolean };

export default function FormulariosPage() {
  const { can } = usePlataforma();
  const gere = can("protocolos:gerir");
  const lista = useApiQuery<Formulario[]>("v1/protocolos/formularios");
  const servicos = useApiQuery<ServicoResumo[]>("v1/protocolos/servicos");
  const { run, pending } = useAction();
  const [busca, setBusca] = useState("");
  const [edicao, setEdicao] = useState<Edicao | null>(null);

  const visiveis = useMemo(() => {
    const termo = busca.trim().toLowerCase();
    return (lista.data ?? []).filter(
      (f) => !termo || `${f.titulo} ${f.descricao}`.toLowerCase().includes(termo),
    );
  }, [lista.data, busca]);

  function vinculados(id: string) {
    return (servicos.data ?? []).filter((s) => s.formulario_id === id);
  }

  async function alternarAtivo(form: Formulario) {
    const ok = await run(
      () =>
        apiClient.post("v1/protocolos/formularios", {
          ...form,
          campos: form.campos ?? [],
          ativo: !form.ativo,
        }),
      form.ativo ? "Formulário desativado" : "Formulário ativado",
    );
    if (ok) void lista.mutate();
  }

  return (
    <div className="mx-auto flex w-full max-w-[1400px] flex-col">
      <CabecalhoDaTela
        icone={FileText}
        titulo="Gerenciar Formulários"
        descricao="O desenho que o cidadão preenche. O sim/não pode revelar outros campos. O arquivo sobe ao armazenamento."
      >
        <Link href="/protocolos" className={BOTAO_SECUNDARIO}>
          <ArrowLeft className="h-4 w-4" aria-hidden />
          Painel
        </Link>
        {gere && (
          <button
            type="button"
            onClick={() => setEdicao({ form: null, copia: false })}
            className={BOTAO_PRIMARIO}
          >
            <Plus className="h-5 w-5" aria-hidden />
            Novo Formulário
          </button>
        )}
      </CabecalhoDaTela>

      <div className="mb-6 flex">
        <Busca valor={busca} onChange={setBusca} placeholder="Pesquisar formulários..." />
      </div>

      <DataState
        loading={lista.isLoading}
        error={lista.error}
        onRetry={() => void lista.mutate()}
        empty={false}
      >
        {visiveis.length === 0 ? (
          <Vazio
            titulo={busca ? "Nenhum formulário encontrado" : "Nenhum formulário cadastrado"}
            texto={
              busca
                ? "Tente buscar com outras palavras."
                : 'Clique em "Novo Formulário" para começar.'
            }
          />
        ) : (
          <div className={`${CARTAO} overflow-x-auto`}>
            <table className="w-full min-w-[760px] text-left text-sm">
              <thead className="border-b border-gray-200 bg-slate-50 text-xs uppercase tracking-wide text-gray-500 dark:border-slate-700 dark:bg-slate-900/50 dark:text-slate-400">
                <tr>
                  <th className="px-6 py-3 font-semibold">Formulário</th>
                  <th className="px-6 py-3 font-semibold">Campos</th>
                  <th className="px-6 py-3 font-semibold">Serviços Vinculados</th>
                  <th className="px-6 py-3 font-semibold">Status</th>
                  <th className="px-6 py-3 font-semibold">Atualizado Em</th>
                  <th className="px-6 py-3 text-right font-semibold">Ações</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-100 dark:divide-slate-700">
                {visiveis.map((form) => {
                  const ligados = vinculados(form.id);
                  return (
                    <tr
                      key={form.id}
                      className="transition-colors hover:bg-slate-50 dark:hover:bg-slate-700/30"
                    >
                      <td className="px-6 py-4">
                        <p className={`font-semibold ${TEXTO_FORTE}`}>{form.titulo}</p>
                        <p className={`mt-0.5 line-clamp-1 text-xs ${TEXTO_FRACO}`}>
                          {form.descricao || "Sem descrição"}
                        </p>
                      </td>
                      <td className="px-6 py-4">
                        <span className="rounded-full bg-blue-100 px-2.5 py-0.5 text-xs font-semibold text-blue-700 dark:bg-sky-400/15 dark:text-sky-300">
                          {(form.campos ?? []).length} campos
                        </span>
                      </td>
                      <td className="px-6 py-4">
                        {ligados.length > 0 ? (
                          <div className="flex flex-col items-start gap-1">
                            {ligados.map((s) => (
                              <span
                                key={s.id}
                                title={s.nome}
                                className="max-w-[220px] truncate rounded-full bg-violet-100 px-2.5 py-0.5 text-xs font-medium text-violet-700 dark:bg-violet-500/15 dark:text-violet-300"
                              >
                                {s.nome}
                              </span>
                            ))}
                          </div>
                        ) : (
                          <span className={`text-xs italic ${TEXTO_FRACO}`}>Nenhum serviço</span>
                        )}
                      </td>
                      <td className="px-6 py-4">
                        {gere ? (
                          <button
                            type="button"
                            disabled={pending}
                            title={form.ativo ? "Desativar" : "Ativar"}
                            aria-label={`${form.ativo ? "Desativar" : "Ativar"} ${form.titulo}`}
                            onClick={() => void alternarAtivo(form)}
                            className="disabled:opacity-50"
                          >
                            <SeloAtivo ativo={form.ativo} />
                          </button>
                        ) : (
                          <SeloAtivo ativo={form.ativo} />
                        )}
                      </td>
                      <td className={`px-6 py-4 ${TEXTO_FRACO}`}>
                        {dataCurta(form.updated_at || form.created_at)}
                      </td>
                      <td className="px-6 py-4">
                        {gere && (
                          <div className="flex justify-end gap-2">
                            <BotaoIcone
                              rotulo={`Editar ${form.titulo}`}
                              onClick={() => setEdicao({ form, copia: false })}
                            >
                              <Edit className="h-4 w-4" aria-hidden />
                            </BotaoIcone>
                            <BotaoIcone
                              rotulo={`Duplicar ${form.titulo}`}
                              cor="text-emerald-600 dark:text-emerald-400"
                              onClick={() => setEdicao({ form, copia: true })}
                            >
                              <Copy className="h-4 w-4" aria-hidden />
                            </BotaoIcone>
                          </div>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </DataState>

      {edicao && (
        <ModalDoFormulario
          form={edicao.form}
          copia={edicao.copia}
          onFechar={() => setEdicao(null)}
          onSalvo={() => {
            setEdicao(null);
            void lista.mutate();
          }}
        />
      )}
    </div>
  );
}

function BotaoIcone({
  rotulo,
  cor = "text-gray-600 dark:text-slate-300",
  onClick,
  children,
}: {
  rotulo: string;
  cor?: string;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      title={rotulo}
      aria-label={rotulo}
      onClick={onClick}
      className={`flex h-9 w-9 items-center justify-center rounded-lg border border-gray-200 transition-colors hover:bg-gray-50 dark:border-slate-600 dark:hover:bg-slate-700 ${cor}`}
    >
      {children}
    </button>
  );
}

function ModalDoFormulario({
  form,
  copia,
  onFechar,
  onSalvo,
}: {
  form: Formulario | null;
  copia: boolean;
  onFechar: () => void;
  onSalvo: () => void;
}) {
  const { run, pending } = useAction();
  const editando = form !== null && !copia;
  // _k: chave estável da linha, para reordenar sem trocar os campos não controlados.
  const [campos, setCampos] = useState<(Campo & { _k: number })[]>(() =>
    (form?.campos?.length
      ? form.campos
      : [{ id: "nome_social", tipo: "text", rotulo: "Nome", obrigatorio: false }]
    ).map((c, i) => ({ ...c, _k: i })),
  );
  const [proxima, setProxima] = useState(campos.length);

  function mudar(indice: number, patch: Partial<Campo>) {
    setCampos((atual) => atual.map((c, i) => (i === indice ? { ...c, ...patch } : c)));
  }

  function mover(indice: number, passo: number) {
    setCampos((atual) => {
      const alvo = indice + passo;
      if (alvo < 0 || alvo >= atual.length) return atual;
      const nova = [...atual];
      [nova[indice], nova[alvo]] = [nova[alvo]!, nova[indice]!];
      return nova;
    });
  }

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const dados = new FormData(e.currentTarget);
    const ok = await run(
      () =>
        apiClient.post("v1/protocolos/formularios", {
          ...(editando ? { id: form.id } : {}),
          unidade_id: String(dados.get("unidade_id") ?? ""),
          titulo: String(dados.get("titulo") ?? "").trim(),
          descricao: String(dados.get("descricao") ?? "").trim(),
          instrucao: String(dados.get("instrucao") ?? "").trim(),
          campos: campos.map((campo) => semChave(campo)),
          ativo: editando ? form.ativo : true,
        }),
      editando ? "Formulário atualizado" : "Formulário salvo",
    );
    if (ok) onSalvo();
  }

  return (
    <Dialog
      open
      onClose={onFechar}
      size="xl"
      title={editando ? "Editar Formulário" : copia ? "Duplicar Formulário" : "Novo Formulário"}
      description={form?.titulo}
    >
      <form onSubmit={(e) => void salvar(e)} className="flex flex-col gap-4">
        <div className="grid gap-4 sm:grid-cols-2">
          <UnidadeDonaSelect
            id="unidade-form"
            permission="protocolos:gerir"
            defaultValue={form?.unidade_id}
            exigir
          />
          <Rotulado id="titulo" rotulo="Título *">
            <input
              id="titulo"
              name="titulo"
              required
              maxLength={200}
              defaultValue={form ? (copia ? `${form.titulo} (cópia)` : form.titulo) : ""}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id="descricao" rotulo="Descrição">
            <textarea
              id="descricao"
              name="descricao"
              rows={3}
              maxLength={4000}
              defaultValue={form?.descricao}
              className={`${CAMPO} resize-y`}
            />
          </Rotulado>
          <Rotulado id="instrucao" rotulo="Instrução ao cidadão">
            <textarea
              id="instrucao"
              name="instrucao"
              rows={3}
              maxLength={4000}
              defaultValue={form?.instrucao}
              className={`${CAMPO} resize-y`}
            />
          </Rotulado>
        </div>

        <div className="flex items-center justify-between gap-3 border-t border-gray-100 pt-4 dark:border-slate-700">
          <p className={`text-sm font-semibold ${TEXTO_FORTE}`}>
            Campos do formulário ({campos.length})
          </p>
          <button
            type="button"
            className={BOTAO_SECUNDARIO}
            onClick={() => {
              setCampos((atual) => [
                ...atual,
                { id: "", tipo: "text", rotulo: "", obrigatorio: false, _k: proxima },
              ]);
              setProxima((n) => n + 1);
            }}
          >
            <Plus className="h-4 w-4" aria-hidden />
            Campo
          </button>
        </div>

        <ul className="flex flex-col gap-3">
          {campos.map((campo, indice) => (
            <li
              key={campo._k}
              className="grid gap-3 rounded-lg border border-l-4 border-gray-200 border-l-blue-500 bg-slate-50/60 p-4 sm:grid-cols-3 dark:border-slate-700 dark:border-l-sky-400 dark:bg-slate-900/40"
            >
              <Rotulado id={`campo-rotulo-${indice}`} rotulo="Rótulo *">
                <input
                  id={`campo-rotulo-${indice}`}
                  required
                  maxLength={200}
                  value={campo.rotulo}
                  onChange={(e) => mudar(indice, { rotulo: e.target.value })}
                  className={CAMPO}
                />
              </Rotulado>
              <Rotulado id={`campo-id-${indice}`} rotulo="Identificador *">
                <input
                  id={`campo-id-${indice}`}
                  required
                  maxLength={40}
                  value={campo.id}
                  onChange={(e) => mudar(indice, { id: e.target.value })}
                  className={CAMPO}
                />
              </Rotulado>
              <Rotulado id={`campo-tipo-${indice}`} rotulo="Tipo">
                <select
                  id={`campo-tipo-${indice}`}
                  value={campo.tipo}
                  onChange={(e) => mudar(indice, { tipo: e.target.value })}
                  className={CAMPO}
                >
                  {TIPOS.map((t) => (
                    <option key={t.value} value={t.value}>
                      {t.label}
                    </option>
                  ))}
                </select>
              </Rotulado>
              {COM_OPCOES.has(campo.tipo) && (
                <Rotulado
                  id={`campo-opcoes-${indice}`}
                  rotulo="Opções (separadas por vírgula)"
                  className="sm:col-span-3"
                >
                  <input
                    id={`campo-opcoes-${indice}`}
                    defaultValue={(campo.opcoes ?? []).join(", ")}
                    onChange={(e) =>
                      mudar(indice, {
                        opcoes: e.target.value
                          .split(",")
                          .map((item) => item.trim())
                          .filter(Boolean),
                      })
                    }
                    className={CAMPO}
                  />
                </Rotulado>
              )}
              {campo.tipo === "boolean" && (
                <Rotulado
                  id={`campo-revela-${indice}`}
                  rotulo="Ao marcar Sim, revela (identificadores)"
                  className="sm:col-span-3"
                >
                  <input
                    id={`campo-revela-${indice}`}
                    defaultValue={(campo.revela ?? []).join(", ")}
                    onChange={(e) =>
                      mudar(indice, {
                        revela: e.target.value
                          .split(",")
                          .map((item) => item.trim())
                          .filter(Boolean),
                      })
                    }
                    className={CAMPO}
                  />
                </Rotulado>
              )}
              <div className="flex flex-wrap items-center justify-between gap-2 sm:col-span-3">
                <Marcar
                  texto="Obrigatório"
                  checked={campo.obrigatorio}
                  onChange={(marcado) => mudar(indice, { obrigatorio: marcado })}
                />
                <div className="flex gap-1">
                  <BotaoIcone
                    rotulo={`Subir ${campo.rotulo || "campo"}`}
                    onClick={() => mover(indice, -1)}
                  >
                    <ArrowUp className="h-4 w-4" aria-hidden />
                  </BotaoIcone>
                  <BotaoIcone
                    rotulo={`Descer ${campo.rotulo || "campo"}`}
                    onClick={() => mover(indice, 1)}
                  >
                    <ArrowDown className="h-4 w-4" aria-hidden />
                  </BotaoIcone>
                  <BotaoIcone
                    rotulo={`Remover ${campo.rotulo || "campo"}`}
                    cor="text-red-600 dark:text-red-400"
                    onClick={() => setCampos((atual) => atual.filter((_, i) => i !== indice))}
                  >
                    <Trash2 className="h-4 w-4" aria-hidden />
                  </BotaoIcone>
                </div>
              </div>
            </li>
          ))}
        </ul>

        <div className="flex justify-end gap-3 border-t border-gray-100 pt-4 dark:border-slate-700">
          <button type="button" onClick={onFechar} className={BOTAO_SECUNDARIO}>
            Cancelar
          </button>
          <button type="submit" disabled={pending} className={BOTAO_PRIMARIO}>
            {editando ? "Salvar alterações" : "Salvar"}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

/** Tira a chave da linha antes de gravar. */
function semChave(campo: Campo & { _k: number }): Campo {
  const copia: Campo & { _k?: number } = { ...campo };
  delete copia._k;
  return copia;
}
