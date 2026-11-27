"use client";

import Link from "next/link";
import { useMemo, useState, type FormEvent } from "react";

import { ArrowLeft, Edit, List, MoreVertical, Plus, Power, Trash2 } from "lucide-react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { useAction } from "@/components/plataforma/useAction";
import { rotuloDoStatus } from "@/components/protocolos/estilo";
import { STATUS } from "@/components/protocolos/painel";
import {
  Abas,
  BOTAO_PRIMARIO,
  BOTAO_SECUNDARIO,
  Busca,
  CabecalhoDaTela,
  CAMPO,
  Detalhe,
  LinhaExpansivel,
  Marcar,
  Pilulas,
  Rotulado,
  ROTULO,
  SeloAtivo,
  TEXTO_FRACO,
  Vazio,
} from "@/components/protocolos/ui";
import { Dialog } from "@/components/ui/Dialog";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import { reais } from "@/lib/moeda";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { OrgTree, UserAdmin } from "@/lib/plataforma/types";

interface Formulario {
  id: string;
  unidade_id?: string;
  titulo: string;
}

interface Orientacao {
  titulo: string;
  texto: string;
  obrigatorio: boolean;
}

interface Etapa {
  id: string;
  ordem: number;
  status: string;
  prazo_dias: number;
  responsavel_tipo?: string;
  responsavel_id?: string;
}

interface Agenda {
  dias: number[];
  inicio: string;
  fim: string;
  duracao_min: number;
  vagas_dia: number;
}

interface Servico {
  id: string;
  unidade_id: string;
  unidade_nome?: string;
  formulario_id: string;
  nome: string;
  descricao: string;
  categoria: string;
  prazo_dias: number;
  taxa_centavos: number;
  destaque: boolean;
  ativo: boolean;
  aprova_sozinho: boolean;
  exige_aceite?: boolean;
  exige_aprovacao?: boolean;
  aprovador_id?: string;
  detalhes?: Detalhes;
  exige_agendamento: boolean;
  so_responsavel_tecnico: boolean;
  impedir_cpf_aberto: boolean;
  campo_anonimo: string;
  workflow?: { ligado: boolean; etapas: Etapa[] | null };
  agenda?: Agenda;
  requisitos?: Orientacao[] | null;
  documentos?: Orientacao[] | null;
}

type Filtro = "todos" | "ativo" | "inativo";
interface Detalhes {
  quem_pode: string;
  passo_a_passo: string;
  contato: string;
  tempo_estimado: string;
  descricao_taxa: string;
}

interface PessoaDaUnidade {
  id: string;
  nome: string;
}

type Aba = "geral" | "carta" | "prazos" | "acesso" | "workflow" | "orientacoes";

const ABAS: { value: Aba; label: string }[] = [
  { value: "geral", label: "Informações Gerais" },
  { value: "prazos", label: "Taxas e Prazos" },
  { value: "acesso", label: "Acesso e Horário" },
  { value: "workflow", label: "Workflow" },
  { value: "orientacoes", label: "Requisitos e Documentos" },
  { value: "carta", label: "Carta ao Cidadão" },
];

const RESPONSAVEL = [
  { value: "", label: "Sem responsável fixo" },
  { value: "unidade", label: "Unidade" },
  { value: "departamento", label: "Departamento" },
  { value: "usuario", label: "Usuário" },
];

const DIAS = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];

const ETAPA_INICIAL: Etapa = { id: "analise", ordem: 1, status: "em_analise", prazo_dias: 5 };

export default function ServicosPage() {
  const { can } = usePlataforma();
  const gere = can("protocolos:gerir");
  const servicos = useApiQuery<Servico[]>("v1/protocolos/servicos");
  const forms = useApiQuery<Formulario[]>("v1/protocolos/formularios");
  const { run, pending } = useAction();
  const [filtro, setFiltro] = useState<Filtro>("todos");
  const [busca, setBusca] = useState("");
  const [aberto, setAberto] = useState<string | null>(null);
  const [menu, setMenu] = useState<string | null>(null);
  const [edicao, setEdicao] = useState<Servico | "novo" | null>(null);

  const todos = useMemo(() => servicos.data ?? [], [servicos.data]);
  const visiveis = useMemo(() => {
    const termo = busca.trim().toLowerCase();
    return todos.filter((s) => {
      if (filtro === "ativo" && !s.ativo) return false;
      if (filtro === "inativo" && s.ativo) return false;
      if (!termo) return true;
      return `${s.nome} ${s.categoria} ${s.unidade_nome ?? ""}`.toLowerCase().includes(termo);
    });
  }, [todos, filtro, busca]);

  async function alternarAtivo(servico: Servico) {
    setMenu(null);
    const ok = await run(
      () => apiClient.post("v1/protocolos/servicos", { ...servico, ativo: !servico.ativo }),
      servico.ativo ? "Serviço desativado" : "Serviço ativado",
    );
    if (ok) void servicos.mutate();
  }

  return (
    <div className="mx-auto flex w-full max-w-[1400px] flex-col">
      <CabecalhoDaTela
        icone={List}
        titulo="Gerenciamento de Serviços"
        descricao="O serviço usa um formulário da mesma unidade. Requisitos, documentos, taxa, horário e etapas acompanham o pedido."
      >
        <Link href="/protocolos" className={BOTAO_SECUNDARIO}>
          <ArrowLeft className="h-4 w-4" aria-hidden />
          Painel
        </Link>
        {gere && (
          <button type="button" onClick={() => setEdicao("novo")} className={BOTAO_PRIMARIO}>
            <Plus className="h-5 w-5" aria-hidden />
            Novo Serviço
          </button>
        )}
      </CabecalhoDaTela>

      <div className="mb-6 flex flex-wrap items-center gap-4">
        <Pilulas
          rotulo="Situação do serviço"
          valor={filtro}
          onChange={setFiltro}
          opcoes={[
            { value: "todos", label: "Todos", total: todos.length },
            { value: "ativo", label: "Ativos", total: todos.filter((s) => s.ativo).length },
            { value: "inativo", label: "Inativos", total: todos.filter((s) => !s.ativo).length },
          ]}
        />
        <Busca valor={busca} onChange={setBusca} placeholder="Pesquisar serviços..." />
      </div>

      <DataState
        loading={servicos.isLoading}
        error={servicos.error}
        onRetry={() => void servicos.mutate()}
        empty={false}
      >
        {visiveis.length === 0 ? (
          <Vazio
            titulo="Nenhum serviço encontrado"
            texto="Não há serviços cadastrados para exibir."
          />
        ) : (
          <ul className="flex flex-col gap-3">
            {visiveis.map((servico) => {
              const etapas = servico.workflow?.ligado ? (servico.workflow.etapas ?? []) : [];
              return (
                <LinhaExpansivel
                  key={servico.id}
                  titulo={servico.nome}
                  selo={<SeloAtivo ativo={servico.ativo} />}
                  subtitulo={`${servico.categoria || "Sem Categoria"} • ${servico.unidade_nome || "Geral"}`}
                  aberta={aberto === servico.id}
                  onAlternar={() => setAberto(aberto === servico.id ? null : servico.id)}
                  acoes={
                    gere && (
                      <MenuDoServico
                        aberto={menu === servico.id}
                        nome={servico.nome}
                        ativo={servico.ativo}
                        desabilitado={pending}
                        onAlternarMenu={() => setMenu(menu === servico.id ? null : servico.id)}
                        onFechar={() => setMenu(null)}
                        onEditar={() => {
                          setMenu(null);
                          setEdicao(servico);
                        }}
                        onAtivar={() => void alternarAtivo(servico)}
                      />
                    )
                  }
                >
                  <Detalhe rotulo="Descrição Curta">{servico.descricao || "N/A"}</Detalhe>
                  <Detalhe rotulo="Unidade">{servico.unidade_nome || "N/A"}</Detalhe>
                  <Detalhe rotulo="Prazos">
                    {servico.prazo_dias > 0 ? `${servico.prazo_dias} dias úteis` : "Sem prazo"}
                  </Detalhe>
                  <Detalhe rotulo="Custos">
                    {servico.taxa_centavos > 0 ? reais(servico.taxa_centavos) : "Gratuito"}
                  </Detalhe>
                  <Detalhe rotulo="Workflow">
                    {etapas.length > 0
                      ? `Ativo e Configurado (${etapas.length} etapa${etapas.length > 1 ? "s" : ""})`
                      : "Padrão"}
                  </Detalhe>
                  <Detalhe rotulo="Atendimento">
                    {servico.exige_agendamento ? "Com horário marcado" : "Sem agendamento"}
                  </Detalhe>
                  <Detalhe rotulo="Regras">
                    {[
                      servico.destaque && "Destaque na carta",
                      servico.exige_aceite && "Exige aceite",
                      servico.exige_aprovacao && "Conclusão com aprovação",
                      servico.aprova_sozinho && "Aprova ao abrir",
                      servico.so_responsavel_tecnico && "Só responsável técnico",
                      servico.impedir_cpf_aberto && "Um pedido aberto por CPF",
                    ]
                      .filter(Boolean)
                      .join(" · ") || "Nenhuma"}
                  </Detalhe>
                  <Detalhe rotulo="Requisitos e Documentos">
                    {(servico.requisitos ?? []).length} requisito(s) ·{" "}
                    {(servico.documentos ?? []).length} documento(s)
                  </Detalhe>
                  {servico.detalhes?.quem_pode && (
                    <Detalhe rotulo="Quem pode solicitar">{servico.detalhes.quem_pode}</Detalhe>
                  )}
                  {servico.detalhes?.contato && (
                    <Detalhe rotulo="Contato">{servico.detalhes.contato}</Detalhe>
                  )}
                </LinhaExpansivel>
              );
            })}
          </ul>
        )}
      </DataState>

      {edicao !== null && (
        <ModalDoServico
          servico={edicao === "novo" ? null : edicao}
          formularios={forms.data ?? []}
          onFechar={() => setEdicao(null)}
          onSalvo={() => {
            setEdicao(null);
            void servicos.mutate();
          }}
        />
      )}
    </div>
  );
}

function MenuDoServico({
  aberto,
  nome,
  ativo,
  desabilitado,
  onAlternarMenu,
  onFechar,
  onEditar,
  onAtivar,
}: {
  aberto: boolean;
  nome: string;
  ativo: boolean;
  desabilitado: boolean;
  onAlternarMenu: () => void;
  onFechar: () => void;
  onEditar: () => void;
  onAtivar: () => void;
}) {
  return (
    <div className="relative">
      <button
        type="button"
        aria-label={`Ações do serviço ${nome}`}
        aria-expanded={aberto}
        onClick={onAlternarMenu}
        className={`flex h-9 w-9 items-center justify-center rounded-lg transition-colors ${
          aberto
            ? "bg-blue-50 text-blue-600 dark:bg-sky-400/15 dark:text-sky-300"
            : "text-gray-500 hover:bg-gray-100 dark:text-slate-400 dark:hover:bg-slate-700"
        }`}
      >
        <MoreVertical className="h-5 w-5" aria-hidden />
      </button>
      {aberto && (
        <>
          <button
            type="button"
            aria-label="Fechar menu"
            className="fixed inset-0 z-40 cursor-default"
            onClick={onFechar}
          />
          <div
            role="menu"
            className="absolute right-0 top-full z-50 mt-2 min-w-[200px] overflow-hidden rounded-lg border border-slate-200 bg-white shadow-[0_10px_15px_-3px_rgba(0,0,0,0.1)] dark:border-slate-600 dark:bg-slate-800 dark:shadow-[0_10px_25px_rgba(0,0,0,0.5)]"
          >
            <ItemDoMenu onClick={onEditar}>
              <Edit className="h-4 w-4" aria-hidden /> Editar Info
            </ItemDoMenu>
            <ItemDoMenu onClick={onAtivar} disabled={desabilitado} perigo={ativo}>
              {ativo ? (
                <Trash2 className="h-4 w-4" aria-hidden />
              ) : (
                <Power className="h-4 w-4" aria-hidden />
              )}
              {ativo ? "Desativar" : "Ativar"}
            </ItemDoMenu>
          </div>
        </>
      )}
    </div>
  );
}

function ItemDoMenu({
  onClick,
  disabled,
  perigo,
  children,
}: {
  onClick: () => void;
  disabled?: boolean;
  perigo?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      role="menuitem"
      disabled={disabled}
      onClick={onClick}
      className={`flex w-full items-center gap-3 px-4 py-3 text-left text-sm font-medium transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-slate-700/60 ${
        perigo ? "text-red-600 dark:text-red-400" : "text-gray-700 dark:text-slate-200"
      }`}
    >
      {children}
    </button>
  );
}

function ModalDoServico({
  servico,
  formularios,
  onFechar,
  onSalvo,
}: {
  servico: Servico | null;
  formularios: Formulario[];
  onFechar: () => void;
  onSalvo: () => void;
}) {
  const { run, pending } = useAction();
  const [aba, setAba] = useState<Aba>("geral");
  const [unidade, setUnidade] = useState(servico?.unidade_id ?? "");
  const [aprovacao, setAprovacao] = useState(servico?.exige_aprovacao ?? false);
  const pessoas = useApiQuery<PessoaDaUnidade[]>(
    aprovacao && unidade
      ? withQuery("v1/protocolos/servicos/pessoas", { unidade_id: unidade })
      : null,
  );
  const detalhes = servico?.detalhes;
  const [exige, setExige] = useState(servico?.exige_agendamento ?? false);
  const [fluxo, setFluxo] = useState(servico?.workflow?.ligado ?? false);
  const [dias, setDias] = useState<number[]>(
    servico?.agenda?.dias?.length ? servico.agenda.dias : [1, 2, 3, 4, 5],
  );
  const [requisitos, setRequisitos] = useState<Orientacao[]>(servico?.requisitos ?? []);
  const [documentos, setDocumentos] = useState<Orientacao[]>(servico?.documentos ?? []);
  const [etapas, setEtapas] = useState<Etapa[]>(
    servico?.workflow?.etapas?.length ? servico.workflow.etapas : [ETAPA_INICIAL],
  );
  const agenda = servico?.agenda;
  const opcoesForm = formularios.filter(
    (f) => !unidade || !f.unidade_id || f.unidade_id === unidade,
  );

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const dados = new FormData(e.currentTarget);
    const taxa = Math.max(0, Math.round(Number(dados.get("taxa") ?? 0) * 100));
    const ok = await run(
      () =>
        apiClient.post("v1/protocolos/servicos", {
          ...(servico ? { id: servico.id, ativo: servico.ativo } : {}),
          unidade_id: String(dados.get("unidade_id") ?? ""),
          formulario_id: String(dados.get("formulario_id") ?? ""),
          nome: String(dados.get("nome") ?? "").trim(),
          descricao: String(dados.get("descricao") ?? "").trim(),
          categoria: String(dados.get("categoria") ?? "").trim(),
          prazo_dias: Number(dados.get("prazo_dias") ?? 0),
          taxa_centavos: Number.isFinite(taxa) ? taxa : 0,
          destaque: dados.get("destaque") === "on",
          aprova_sozinho: dados.get("aprova_sozinho") === "on",
          exige_aceite: dados.get("exige_aceite") === "on",
          exige_aprovacao: aprovacao,
          aprovador_id:
            aprovacao && dados.get("aprovador_id") ? String(dados.get("aprovador_id")) : null,
          detalhes: {
            quem_pode: String(dados.get("quem_pode") ?? "").trim(),
            passo_a_passo: String(dados.get("passo_a_passo") ?? "").trim(),
            contato: String(dados.get("contato") ?? "").trim(),
            tempo_estimado: String(dados.get("tempo_estimado") ?? "").trim(),
            descricao_taxa: String(dados.get("descricao_taxa") ?? "").trim(),
          },
          exige_agendamento: exige,
          so_responsavel_tecnico: dados.get("so_responsavel_tecnico") === "on",
          impedir_cpf_aberto: dados.get("impedir_cpf_aberto") === "on",
          campo_anonimo: String(dados.get("campo_anonimo") ?? "").trim(),
          workflow: { ligado: fluxo, etapas: fluxo ? etapas : [] },
          agenda: exige
            ? {
                dias,
                inicio: String(dados.get("inicio") ?? "08:00"),
                fim: String(dados.get("fim") ?? "17:00"),
                duracao_min: Number(dados.get("duracao_min") ?? 30),
                vagas_dia: Number(dados.get("vagas_dia") ?? 8),
              }
            : { dias: [], inicio: "", fim: "", duracao_min: 0, vagas_dia: 0 },
          requisitos,
          documentos,
        }),
      servico ? "Serviço atualizado" : "Serviço salvo",
    );
    if (ok) onSalvo();
  }

  return (
    <Dialog
      open
      onClose={onFechar}
      size="xl"
      title={servico ? "Editar Serviço" : "Novo Serviço"}
      description={servico?.nome}
    >
      <form
        onSubmit={(e) => void salvar(e)}
        onInvalidCapture={(e) => {
          // O campo obrigatório pode estar numa aba escondida: mostra a aba dele.
          const painel = (e.target as HTMLElement).closest<HTMLElement>("[data-aba]");
          if (painel?.dataset.aba) setAba(painel.dataset.aba as Aba);
        }}
      >
        <Abas rotulo="Partes do serviço" valor={aba} opcoes={ABAS} onChange={setAba} />

        <div data-aba="geral" hidden={aba !== "geral"} className="grid gap-4 sm:grid-cols-2">
          <UnidadeDonaSelect
            id="unidade-servico"
            permission="protocolos:gerir"
            defaultValue={servico?.unidade_id}
            onChange={setUnidade}
            exigir
          />
          <Rotulado id="formulario_id" rotulo="Formulário *">
            <select
              id="formulario_id"
              name="formulario_id"
              required
              defaultValue={servico?.formulario_id ?? ""}
              className={CAMPO}
            >
              <option value="">Selecione…</option>
              {opcoesForm.map((form) => (
                <option key={form.id} value={form.id}>
                  {form.titulo}
                </option>
              ))}
            </select>
          </Rotulado>
          <Rotulado id="nome" rotulo="Nome do Serviço *">
            <input
              id="nome"
              name="nome"
              required
              maxLength={200}
              defaultValue={servico?.nome}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id="categoria" rotulo="Categoria">
            <input
              id="categoria"
              name="categoria"
              maxLength={80}
              defaultValue={servico?.categoria}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id="descricao" rotulo="Descrição" className="sm:col-span-2">
            <textarea
              id="descricao"
              name="descricao"
              rows={4}
              maxLength={4000}
              defaultValue={servico?.descricao}
              className={`${CAMPO} min-h-[100px] resize-y`}
            />
          </Rotulado>
        </div>

        <div data-aba="prazos" hidden={aba !== "prazos"} className="grid gap-4 sm:grid-cols-2">
          <Rotulado id="prazo_dias" rotulo="Prazo em dias úteis">
            <input
              id="prazo_dias"
              name="prazo_dias"
              type="number"
              min={0}
              defaultValue={servico?.prazo_dias ?? 0}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id="taxa" rotulo="Taxa (R$)">
            <input
              id="taxa"
              name="taxa"
              type="number"
              min={0}
              step="0.01"
              defaultValue={(servico?.taxa_centavos ?? 0) / 100}
              className={CAMPO}
            />
          </Rotulado>
          <Marcar name="destaque" texto="Destaque na carta" defaultChecked={servico?.destaque} />
          <Marcar
            name="aprova_sozinho"
            texto="Aprova ao abrir, quando não há etapa"
            defaultChecked={servico?.aprova_sozinho}
          />
        </div>

        <div data-aba="acesso" hidden={aba !== "acesso"} className="grid gap-4 sm:grid-cols-2">
          <Marcar
            name="so_responsavel_tecnico"
            texto="Somente responsável técnico ativo"
            defaultChecked={servico?.so_responsavel_tecnico}
          />
          <Marcar
            name="impedir_cpf_aberto"
            texto="Impede outro pedido aberto do mesmo CPF"
            defaultChecked={servico?.impedir_cpf_aberto}
          />
          <Rotulado id="campo_anonimo" rotulo="Campo que anonimiza">
            <input
              id="campo_anonimo"
              name="campo_anonimo"
              maxLength={40}
              defaultValue={servico?.campo_anonimo}
              className={CAMPO}
            />
          </Rotulado>
          <div className="flex items-end">
            <Marcar texto="Exige horário" checked={exige} onChange={setExige} />
          </div>
          {exige && (
            <div className="flex flex-col gap-3 sm:col-span-2">
              <p className={ROTULO}>Dias de atendimento</p>
              <div className="flex flex-wrap gap-2">
                {DIAS.map((nome, dia) => (
                  <Marcar
                    key={nome}
                    texto={nome}
                    checked={dias.includes(dia)}
                    onChange={(marcado) =>
                      setDias((atual) =>
                        marcado
                          ? [...atual, dia].sort((a, b) => a - b)
                          : atual.filter((item) => item !== dia),
                      )
                    }
                  />
                ))}
              </div>
              <div className="grid gap-4 sm:grid-cols-4">
                <Rotulado id="inicio" rotulo="Início">
                  <input
                    id="inicio"
                    name="inicio"
                    type="time"
                    required
                    defaultValue={agenda?.inicio || "08:00"}
                    className={CAMPO}
                  />
                </Rotulado>
                <Rotulado id="fim" rotulo="Fim">
                  <input
                    id="fim"
                    name="fim"
                    type="time"
                    required
                    defaultValue={agenda?.fim || "17:00"}
                    className={CAMPO}
                  />
                </Rotulado>
                <Rotulado id="duracao_min" rotulo="Duração (min)">
                  <input
                    id="duracao_min"
                    name="duracao_min"
                    type="number"
                    min={1}
                    required
                    defaultValue={agenda?.duracao_min || 30}
                    className={CAMPO}
                  />
                </Rotulado>
                <Rotulado id="vagas_dia" rotulo="Vagas no dia">
                  <input
                    id="vagas_dia"
                    name="vagas_dia"
                    type="number"
                    min={1}
                    required
                    defaultValue={agenda?.vagas_dia || 8}
                    className={CAMPO}
                  />
                </Rotulado>
              </div>
            </div>
          )}
        </div>

        <div data-aba="workflow" hidden={aba !== "workflow"} className="flex flex-col gap-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <Marcar
              name="exige_aceite"
              texto="Exige aceite antes de tratar"
              defaultChecked={servico?.exige_aceite}
            />
            <Marcar
              texto="Conclusão passa por aprovação"
              checked={aprovacao}
              onChange={setAprovacao}
            />
          </div>
          {aprovacao && (
            <Rotulado id="aprovador_id" rotulo="Quem aprova">
              <select
                id="aprovador_id"
                name="aprovador_id"
                key={`${unidade}-${(pessoas.data ?? []).length}`}
                defaultValue={servico?.aprovador_id ?? ""}
                className={CAMPO}
              >
                <option value="">Quem gere a unidade</option>
                {(pessoas.data ?? []).map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.nome}
                  </option>
                ))}
              </select>
            </Rotulado>
          )}
          <p className={`text-sm ${TEXTO_FRACO}`}>
            Com aceite, ninguém trata o protocolo até alguém assumir. Com aprovação, a conclusão
            espera quem gere a unidade aprovar ou devolver.
          </p>
          <Marcar texto="Workflow ligado" checked={fluxo} onChange={setFluxo} />
          {fluxo ? (
            <ListaEtapas etapas={etapas} onChange={setEtapas} />
          ) : (
            <p className={`text-sm ${TEXTO_FRACO}`}>
              Sem workflow, o pedido não passa por etapas: a mesa trata e conclui direto.
            </p>
          )}
        </div>

        <div data-aba="carta" hidden={aba !== "carta"} className="grid gap-4 sm:grid-cols-2">
          <Rotulado id="quem_pode" rotulo="Quem pode solicitar" className="sm:col-span-2">
            <textarea
              id="quem_pode"
              name="quem_pode"
              rows={3}
              maxLength={4000}
              defaultValue={detalhes?.quem_pode}
              className={`${CAMPO} resize-y`}
            />
          </Rotulado>
          <Rotulado id="passo_a_passo" rotulo="Passo a passo" className="sm:col-span-2">
            <textarea
              id="passo_a_passo"
              name="passo_a_passo"
              rows={4}
              maxLength={4000}
              defaultValue={detalhes?.passo_a_passo}
              className={`${CAMPO} resize-y`}
            />
          </Rotulado>
          <Rotulado id="tempo_estimado" rotulo="Tempo estimado de atendimento">
            <input
              id="tempo_estimado"
              name="tempo_estimado"
              maxLength={120}
              placeholder="Ex.: 5 a 10 dias úteis"
              defaultValue={detalhes?.tempo_estimado}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id="contato" rotulo="Contato">
            <input
              id="contato"
              name="contato"
              maxLength={500}
              placeholder="Telefone, e-mail ou endereço"
              defaultValue={detalhes?.contato}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id="descricao_taxa" rotulo="Sobre a taxa" className="sm:col-span-2">
            <input
              id="descricao_taxa"
              name="descricao_taxa"
              maxLength={500}
              placeholder="Como e quando a taxa é paga"
              defaultValue={detalhes?.descricao_taxa}
              className={CAMPO}
            />
          </Rotulado>
          <p className={`text-sm sm:col-span-2 ${TEXTO_FRACO}`}>
            O prazo máximo é o da aba Taxas e Prazos. Estes textos aparecem na página do serviço na
            carta pública.
          </p>
        </div>

        <div data-aba="orientacoes" hidden={aba !== "orientacoes"} className="flex flex-col gap-6">
          <ListaOrientacao titulo="Requisitos" itens={requisitos} onChange={setRequisitos} />
          <ListaOrientacao
            titulo="Documentos esperados"
            itens={documentos}
            onChange={setDocumentos}
          />
        </div>

        <div className="mt-6 flex justify-end gap-3 border-t border-gray-100 pt-4 dark:border-slate-700">
          <button type="button" onClick={onFechar} className={BOTAO_SECUNDARIO}>
            Cancelar
          </button>
          <button type="submit" disabled={pending} className={BOTAO_PRIMARIO}>
            {servico ? "Salvar alterações" : "Salvar"}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

function ListaEtapas({
  etapas,
  onChange,
}: {
  etapas: Etapa[];
  onChange: (etapas: Etapa[]) => void;
}) {
  function mudar(indice: number, patch: Partial<Etapa>) {
    onChange(etapas.map((item, i) => (i === indice ? { ...item, ...patch } : item)));
  }
  return (
    <div className="flex flex-col gap-3">
      {etapas.map((etapa, indice) => (
        <div
          key={indice}
          className="grid gap-3 rounded-lg border border-gray-200 bg-slate-50/60 p-4 sm:grid-cols-4 dark:border-slate-700 dark:bg-slate-900/40"
        >
          <p className="flex items-center gap-2 text-sm font-semibold text-gray-700 sm:col-span-4 dark:text-slate-200">
            <span className="flex h-6 w-6 items-center justify-center rounded-full bg-blue-500 text-xs text-white">
              {indice + 1}
            </span>
            Etapa {indice + 1}
          </p>
          <Rotulado id={`etapa-id-${indice}`} rotulo="Identificador *">
            <input
              id={`etapa-id-${indice}`}
              required
              maxLength={40}
              value={etapa.id}
              onChange={(e) => mudar(indice, { id: e.target.value })}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id={`etapa-ordem-${indice}`} rotulo="Ordem">
            <input
              id={`etapa-ordem-${indice}`}
              type="number"
              min={0}
              value={etapa.ordem}
              onChange={(e) => mudar(indice, { ordem: Number(e.target.value) })}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id={`etapa-status-${indice}`} rotulo="Status">
            <select
              id={`etapa-status-${indice}`}
              value={etapa.status}
              onChange={(e) => mudar(indice, { status: e.target.value })}
              className={CAMPO}
            >
              {STATUS.map((s) => (
                <option key={s.value} value={s.value}>
                  {rotuloDoStatus(s.value)}
                </option>
              ))}
            </select>
          </Rotulado>
          <Rotulado id={`etapa-prazo-${indice}`} rotulo="Prazo (dias úteis)">
            <input
              id={`etapa-prazo-${indice}`}
              type="number"
              min={0}
              value={etapa.prazo_dias}
              onChange={(e) => mudar(indice, { prazo_dias: Number(e.target.value) })}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id={`etapa-tipo-${indice}`} rotulo="Responsável" className="sm:col-span-2">
            <select
              id={`etapa-tipo-${indice}`}
              value={etapa.responsavel_tipo ?? ""}
              onChange={(e) =>
                mudar(indice, {
                  responsavel_tipo: e.target.value,
                  responsavel_id: e.target.value ? etapa.responsavel_id : "",
                })
              }
              className={CAMPO}
            >
              {RESPONSAVEL.map((r) => (
                <option key={r.value} value={r.value}>
                  {r.label}
                </option>
              ))}
            </select>
          </Rotulado>
          <ResponsavelDaEtapa
            etapa={etapa}
            indice={indice}
            onEscolher={(responsavelId) => mudar(indice, { responsavel_id: responsavelId })}
          />
          {etapas.length > 1 && (
            <div className="flex items-end justify-end sm:col-span-4">
              <button
                type="button"
                onClick={() => onChange(etapas.filter((_, i) => i !== indice))}
                className="inline-flex items-center gap-1.5 text-sm font-medium text-red-600 hover:underline dark:text-red-400"
              >
                <Trash2 className="h-4 w-4" aria-hidden />
                Remover etapa
              </button>
            </div>
          )}
        </div>
      ))}
      <button
        type="button"
        className={`${BOTAO_SECUNDARIO} self-start`}
        onClick={() =>
          onChange([
            ...etapas,
            { id: "", ordem: etapas.length + 1, status: "em_andamento", prazo_dias: 0 },
          ])
        }
      >
        <Plus className="h-4 w-4" aria-hidden />
        Etapa
      </button>
    </div>
  );
}

function ResponsavelDaEtapa({
  etapa,
  indice,
  onEscolher,
}: {
  etapa: Etapa;
  indice: number;
  onEscolher: (id: string) => void;
}) {
  const { can } = usePlataforma();
  const tipo = etapa.responsavel_tipo ?? "";
  const arvore = useApiQuery<OrgTree[]>(
    tipo === "unidade" || tipo === "departamento" ? "v1/iam/org-tree" : null,
  );
  const pessoas = useApiPage<UserAdmin>(
    tipo === "usuario" && can("users:read")
      ? withQuery("v1/users", { active: true, page_size: 50 })
      : null,
  );
  if (!tipo) return null;
  if (tipo === "usuario" && !can("users:read")) {
    return (
      <p className={`text-sm sm:col-span-2 ${TEXTO_FRACO}`}>
        A lista de usuários pede a permissão de leitura do diretório.
      </p>
    );
  }
  const opcoes =
    tipo === "unidade"
      ? (arvore.data ?? []).flatMap((entidade) =>
          entidade.unidades
            .filter((unidade) => unidade.ativo)
            .map((unidade) => ({
              value: unidade.id,
              label: unidade.sigla ? `${unidade.sigla} · ${unidade.nome}` : unidade.nome,
            })),
        )
      : tipo === "departamento"
        ? (arvore.data ?? []).flatMap((entidade) =>
            entidade.unidades.flatMap((unidade) =>
              unidade.departamentos
                .filter((departamento) => departamento.ativo)
                .map((departamento) => ({
                  value: departamento.id,
                  label: `${unidade.nome} · ${departamento.nome}`,
                })),
            ),
          )
        : (pessoas.data?.items ?? [])
            .filter((pessoa) => pessoa.active)
            .map((pessoa) => ({ value: pessoa.id, label: pessoa.display_name || pessoa.username }));
  const id = `etapa-resp-${indice}`;
  return (
    <Rotulado
      id={id}
      rotulo={tipo === "usuario" ? "Usuário" : tipo === "departamento" ? "Departamento" : "Unidade"}
      className="sm:col-span-2"
    >
      <select
        id={id}
        value={etapa.responsavel_id ?? ""}
        onChange={(e) => onEscolher(e.target.value)}
        className={CAMPO}
      >
        <option value="">Selecione…</option>
        {opcoes.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </Rotulado>
  );
}

function ListaOrientacao({
  titulo,
  itens,
  onChange,
}: {
  titulo: string;
  itens: Orientacao[];
  onChange: (itens: Orientacao[]) => void;
}) {
  function mudar(indice: number, patch: Partial<Orientacao>) {
    onChange(itens.map((atual, i) => (i === indice ? { ...atual, ...patch } : atual)));
  }
  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm font-semibold text-gray-800 dark:text-slate-100">{titulo}</p>
      {itens.length === 0 && <p className={`text-sm ${TEXTO_FRACO}`}>Nenhum item.</p>}
      {itens.map((item, indice) => (
        <div
          key={indice}
          className="grid gap-3 rounded-lg border border-gray-200 bg-slate-50/60 p-4 sm:grid-cols-[1fr_2fr_auto] dark:border-slate-700 dark:bg-slate-900/40"
        >
          <Rotulado id={`${titulo}-titulo-${indice}`} rotulo="Título *">
            <input
              id={`${titulo}-titulo-${indice}`}
              required
              maxLength={200}
              value={item.titulo}
              onChange={(e) => mudar(indice, { titulo: e.target.value })}
              className={CAMPO}
            />
          </Rotulado>
          <Rotulado id={`${titulo}-texto-${indice}`} rotulo="Texto">
            <input
              id={`${titulo}-texto-${indice}`}
              maxLength={2000}
              value={item.texto}
              onChange={(e) => mudar(indice, { texto: e.target.value })}
              className={CAMPO}
            />
          </Rotulado>
          <div className="flex items-end gap-2">
            <Marcar
              texto="Obrigatório"
              checked={item.obrigatorio}
              onChange={(marcado) => mudar(indice, { obrigatorio: marcado })}
            />
            <button
              type="button"
              aria-label={`Remover ${item.titulo || "item"}`}
              onClick={() => onChange(itens.filter((_, i) => i !== indice))}
              className="flex h-[42px] w-[42px] items-center justify-center rounded-lg text-red-600 transition-colors hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-500/10"
            >
              <Trash2 className="h-4 w-4" aria-hidden />
            </button>
          </div>
        </div>
      ))}
      <button
        type="button"
        className={`${BOTAO_SECUNDARIO} self-start`}
        onClick={() => onChange([...itens, { titulo: "", texto: "", obrigatorio: false }])}
      >
        <Plus className="h-4 w-4" aria-hidden />
        Incluir
      </button>
    </div>
  );
}
