"use client";

import { useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { PessoaDaUnidade } from "@/components/projetos/PessoaDaUnidade";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import {
  PRIORIDADE_PROJETO,
  type EstadoProjeto,
  type PrioridadeProjeto,
} from "@/lib/projetos/projetos";

const PASSOS = [
  {
    chave: "what",
    titulo: "O que",
    pergunta: "Qual é a entrega principal deste projeto?",
  },
  {
    chave: "why",
    titulo: "Por que",
    pergunta: "Qual problema ou benefício justifica este esforço?",
  },
  {
    chave: "where",
    titulo: "Onde",
    pergunta: "Onde esta entrega será aplicada?",
  },
  {
    chave: "when",
    titulo: "Quando",
    pergunta: "Qual o horizonte planejado?",
  },
  {
    chave: "who",
    titulo: "Quem",
    pergunta: "Quem responde pelo projeto e quem apoia?",
  },
  {
    chave: "how",
    titulo: "Como",
    pergunta: "Como o projeto será executado?",
  },
  {
    chave: "how-much",
    titulo: "Quanto",
    pergunta: "Qual a previsão de custo?",
  },
] as const;

export interface Corpo5W2H {
  titulo: string;
  descricao: string;
  objetivo: string;
  prioridade: PrioridadeProjeto;
  gerente_id?: string;
  gerente_nome: string;
  orcamento: number;
  unidade_id?: string;
  produto_id?: string;
  data_inicio_planejada?: string;
  data_fim_planejada?: string;
  carta: {
    onde: string;
    patrocinador: string;
    metodologia: string;
    centro_custo: string;
  };
  etapas: string[];
}

function diaIso(dia: string) {
  if (!dia) return undefined;
  return `${dia}T12:00:00Z`;
}

export function Wizard5W2H({
  produtos,
  colunas,
  pending,
  onCancel,
  onCriar,
}: {
  produtos: { id: string; nome: string; unidade_id?: string }[];
  colunas: { value: string; label: string }[];
  pending: boolean;
  onCancel: () => void;
  onCriar: (corpo: Corpo5W2H, quadro: EstadoProjeto) => void;
}) {
  const [passo, setPasso] = useState(0);
  const [titulo, setTitulo] = useState("");
  const [descricao, setDescricao] = useState("");
  const [objetivo, setObjetivo] = useState("");
  const [onde, setOnde] = useState("");
  const [inicio, setInicio] = useState("");
  const [fim, setFim] = useState("");
  const [gerente, setGerente] = useState({ id: "", nome: "" });
  const [patrocinador, setPatrocinador] = useState("");
  const [metodologia, setMetodologia] = useState("");
  const [rascunho, setRascunho] = useState("");
  const [etapas, setEtapas] = useState<string[]>([]);
  const [orcamento, setOrcamento] = useState("");
  const [centro, setCentro] = useState("");
  const [prioridade, setPrioridade] = useState<PrioridadeProjeto>("media");
  const [unidadeId, setUnidadeId] = useState("");
  const [produtoId, setProdutoId] = useState("");
  const [quadro, setQuadro] = useState<EstadoProjeto>(
    (colunas[0]?.value as EstadoProjeto) || "planejado",
  );
  const [erro, setErro] = useState("");

  const atual = PASSOS[passo]!;
  const ultimo = passo === PASSOS.length - 1;
  const tituloOk = titulo.trim().length > 0 && titulo.trim().length <= 160;

  function adicionarEtapa() {
    const nome = rascunho.trim();
    if (!nome || etapas.length >= 30) return;
    setEtapas((lista) => [...lista, nome.slice(0, 200)]);
    setRascunho("");
  }

  function criar() {
    if (!tituloOk) {
      setErro("Informe o nome do projeto.");
      setPasso(0);
      return;
    }
    if (inicio && fim && fim < inicio) {
      setErro("A data de término precisa ser igual ou posterior à de início.");
      setPasso(3);
      return;
    }
    setErro("");
    onCriar(
      {
        titulo: titulo.trim(),
        descricao: descricao.trim(),
        objetivo: objetivo.trim(),
        prioridade,
        gerente_id: gerente.id || undefined,
        gerente_nome: gerente.id ? gerente.nome : "",
        orcamento: Number(orcamento) || 0,
        unidade_id: unidadeId || undefined,
        produto_id: produtoId || undefined,
        data_inicio_planejada: diaIso(inicio),
        data_fim_planejada: diaIso(fim),
        carta: {
          onde: onde.trim(),
          patrocinador: patrocinador.trim(),
          metodologia: metodologia.trim(),
          centro_custo: centro.trim(),
        },
        etapas,
      },
      quadro,
    );
  }

  return (
    <div className="mt-3 grid gap-4" data-testid="wizard-5w2h">
      <div>
        <p className="text-xs font-medium text-muted">
          Passo {passo + 1} de {PASSOS.length} · {atual.titulo}
        </p>
        <div className="mt-2 flex gap-1" aria-hidden="true">
          {PASSOS.map((item, i) => (
            <span
              key={item.chave}
              className={`h-1 flex-1 rounded-full ${i <= passo ? "bg-primary" : "bg-black/10 dark:bg-white/10"}`}
            />
          ))}
        </div>
        <h3 className="mt-3 text-base font-semibold text-foreground">{atual.pergunta}</h3>
      </div>

      {erro && (
        <p role="alert" className="text-sm text-danger">
          {erro}
        </p>
      )}

      {atual.chave === "what" && (
        <div className="grid gap-3">
          <Input
            id="w5-titulo"
            label="Nome do projeto *"
            value={titulo}
            onChange={(e) => setTitulo(e.target.value)}
            maxLength={160}
            placeholder="Ex.: Modernização do Data Center"
          />
          <Textarea
            id="w5-escopo"
            label="Escopo e entregáveis"
            value={descricao}
            onChange={(e) => setDescricao(e.target.value)}
            rows={4}
            maxLength={4000}
          />
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Select
              id="w5-prio"
              label="Prioridade"
              value={prioridade}
              onChange={(e) => setPrioridade(e.target.value as PrioridadeProjeto)}
              options={Object.entries(PRIORIDADE_PROJETO).map(([k, v]) => ({
                value: k,
                label: v.label,
              }))}
            />
            <Select
              id="w5-quadro"
              label="Quadro"
              value={quadro}
              onChange={(e) => setQuadro(e.target.value as EstadoProjeto)}
              options={
                colunas.length > 0 ? colunas : [{ value: "planejado", label: "Planejado" }]
              }
            />
          </div>
          <UnidadeDonaSelect
            id="w5-unidade"
            label="Unidade"
            permission="projetos:manage"
            defaultValue={unidadeId}
            onChange={setUnidadeId}
          />
          {produtos.length > 0 && (
            <Select
              id="w5-produto"
              label="Produto digital (opcional)"
              value={produtoId}
              onChange={(e) => setProdutoId(e.target.value)}
              options={[
                { value: "", label: "Nenhum produto associado" },
                ...produtos
                  .filter((prod) => !unidadeId || !prod.unidade_id || prod.unidade_id === unidadeId)
                  .map((prod) => ({ value: prod.id, label: prod.nome })),
              ]}
            />
          )}
        </div>
      )}

      {atual.chave === "why" && (
        <Textarea
          id="w5-objetivo"
          label="Justificativa e impacto esperado"
          value={objetivo}
          onChange={(e) => setObjetivo(e.target.value)}
          rows={5}
          maxLength={4000}
        />
      )}

      {atual.chave === "where" && (
        <Textarea
          id="w5-onde"
          label="Departamento, unidade, ambiente ou sistema"
          value={onde}
          onChange={(e) => setOnde(e.target.value)}
          rows={3}
          maxLength={2000}
        />
      )}

      {atual.chave === "when" && (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Input
            id="w5-inicio"
            label="Data de início"
            type="date"
            value={inicio}
            onChange={(e) => setInicio(e.target.value)}
          />
          <Input
            id="w5-fim"
            label="Data de término"
            type="date"
            value={fim}
            onChange={(e) => setFim(e.target.value)}
          />
        </div>
      )}

      {atual.chave === "who" && (
        <div className="grid gap-3">
          <PessoaDaUnidade
            id="w5-gerente"
            label="Responsável direto"
            unidadeId={unidadeId}
            value={gerente}
            onChange={setGerente}
          />
          <Input
            id="w5-patrocinador"
            label="Patrocinador ou solicitante"
            value={patrocinador}
            onChange={(e) => setPatrocinador(e.target.value)}
            maxLength={400}
          />
        </div>
      )}

      {atual.chave === "how" && (
        <div className="grid gap-3">
          <Textarea
            id="w5-metodologia"
            label="Resumo da metodologia"
            value={metodologia}
            onChange={(e) => setMetodologia(e.target.value)}
            rows={3}
            maxLength={4000}
          />
          <div className="flex items-end gap-2">
            <div className="flex-1">
              <Input
                id="w5-etapa"
                label="Etapa macro"
                value={rascunho}
                onChange={(e) => setRascunho(e.target.value)}
                maxLength={200}
                placeholder="Ex.: Planejamento"
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    adicionarEtapa();
                  }
                }}
              />
            </div>
            <Button type="button" variant="secondary" onClick={adicionarEtapa}>
              Adicionar etapa
            </Button>
          </div>
          <p className="text-xs text-muted">
            Cada etapa vira uma tarefa em &quot;A fazer&quot;. Editar a carta depois não cria tarefa de novo.
          </p>
          {etapas.length > 0 && (
            <ol className="grid gap-1 text-sm">
              {etapas.map((etapa, i) => (
                <li key={`${etapa}-${i}`} className="flex items-center justify-between gap-2">
                  <span>
                    {i + 1}. {etapa}
                  </span>
                  <Button
                    type="button"
                    size="sm"
                    variant="ghost"
                    onClick={() => setEtapas((lista) => lista.filter((_, idx) => idx !== i))}
                  >
                    Remover
                  </Button>
                </li>
              ))}
            </ol>
          )}
        </div>
      )}

      {atual.chave === "how-much" && (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Input
            id="w5-orcamento"
            label="Orçamento estimado (R$)"
            type="number"
            min="0"
            step="0.01"
            value={orcamento}
            onChange={(e) => setOrcamento(e.target.value)}
          />
          <Input
            id="w5-centro"
            label="Centro de custo"
            value={centro}
            onChange={(e) => setCentro(e.target.value)}
            maxLength={120}
          />
        </div>
      )}

      <div className="flex flex-wrap justify-end gap-2 pt-1">
        <Button type="button" variant="secondary" onClick={onCancel}>
          Cancelar
        </Button>
        {passo > 0 && (
          <Button type="button" variant="secondary" onClick={() => setPasso((n) => n - 1)}>
            Voltar
          </Button>
        )}
        {passo > 0 && !ultimo && (
          <Button type="button" variant="ghost" onClick={() => setPasso((n) => n + 1)}>
            Pular
          </Button>
        )}
        {ultimo ? (
          <Button type="button" onClick={criar} loading={pending} data-testid="wizard-criar">
            Criar projeto
          </Button>
        ) : (
          <Button
            type="button"
            onClick={() => setPasso((n) => n + 1)}
            disabled={passo === 0 && !tituloOk}
            data-testid="wizard-proximo"
          >
            Próximo
          </Button>
        )}
      </div>
    </div>
  );
}
