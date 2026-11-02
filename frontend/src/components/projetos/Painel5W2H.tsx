"use client";

import { type FormEvent, useState } from "react";

import { DataState } from "@/components/plataforma/DataState";
import { PessoaDaUnidade } from "@/components/projetos/PessoaDaUnidade";
import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Dialog } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";

export interface VisaoCarta {
  projeto_id: string;
  titulo: string;
  descricao: string;
  objetivo: string;
  onde: string;
  patrocinador: string;
  metodologia: string;
  centro_custo: string;
  data_inicio_planejada?: string;
  data_fim_planejada?: string;
  unidade_id?: string;
  gerente_id?: string;
  gerente_nome: string;
  orcamento: number;
  existe: boolean;
  etapas_how: number;
  etapas_how_concluidas: number;
  custo_realizado: number;
  updated_at?: string;
}

function moeda(valor: number) {
  return valor.toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
}

function dia(iso?: string) {
  if (!iso) return "";
  const data = new Date(iso);
  if (Number.isNaN(data.getTime())) return "";
  return data.toLocaleDateString("pt-BR", { timeZone: "UTC" });
}

function diaInput(iso?: string) {
  return iso ? iso.slice(0, 10) : "";
}

function diaIso(diaValor: string) {
  return diaValor ? `${diaValor}T12:00:00Z` : null;
}

function texto(valor: string) {
  return valor.trim() ? valor : "Não informado.";
}

function Barra({
  rotulo,
  percentual,
  tom,
}: {
  rotulo: string;
  percentual: number | null;
  tom: "primary" | "danger";
}) {
  const largura = percentual == null ? 0 : Math.max(0, Math.min(100, percentual));
  return (
    <div
      role="meter"
      aria-label={rotulo}
      aria-valuemin={0}
      aria-valuemax={100}
      aria-valuenow={percentual == null ? undefined : Math.round(percentual)}
      className="mt-3 h-2 overflow-hidden rounded-full bg-black/10 dark:bg-white/10"
    >
      <div
        className={`h-full rounded-full ${tom === "danger" ? "bg-danger" : "bg-primary"}`}
        style={{ width: `${largura}%` }}
      />
    </div>
  );
}

export function Painel5W2H({
  projetoId,
  podeEditar,
  onSalvo,
}: {
  projetoId: string;
  podeEditar: boolean;
  onSalvo: () => void;
}) {
  const carta = useApiQuery<VisaoCarta>(`v1/projetos/${projetoId}/carta`);
  const v = carta.data;
  const { run, pending } = useAction();
  const [editando, setEditando] = useState(false);

  const fisico =
    v && v.etapas_how > 0 ? Math.round((v.etapas_how_concluidas / v.etapas_how) * 100) : null;
  const financeiro =
    v && Number(v.orcamento) > 0
      ? Math.round((Number(v.custo_realizado) / Number(v.orcamento)) * 100)
      : null;

  return (
    <DataState
      loading={carta.isLoading}
      error={carta.error}
      empty={!v}
      emptyTitle="Carta indisponível"
      onRetry={() => void carta.mutate()}
    >
      {v && (
        <div className="grid gap-4" data-testid="painel-5w2h">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p className="text-sm text-muted">
              Plano diretor do projeto. Salvar de novo atualiza estes campos e não gera tarefa.
            </p>
            {podeEditar && (
              <Button type="button" size="sm" onClick={() => setEditando(true)}>
                Editar escopo 5W2H
              </Button>
            )}
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <Card data-testid="progresso-fisico">
              <CardHeader>
                <CardTitle as="h2">Progresso físico</CardTitle>
              </CardHeader>
              <CardContent className="pb-5">
                {v.etapas_how === 0 ? (
                  <p className="text-sm text-muted">Nenhuma etapa veio do passo Como.</p>
                ) : (
                  <>
                    <p className="text-2xl font-semibold text-foreground">{fisico}%</p>
                    <p className="mt-1 text-sm text-muted">
                      {v.etapas_how_concluidas} de {v.etapas_how} etapas concluídas
                    </p>
                    <Barra rotulo="Progresso físico" percentual={fisico} tom="primary" />
                  </>
                )}
              </CardContent>
            </Card>

            <Card data-testid="progresso-financeiro">
              <CardHeader>
                <CardTitle as="h2">Progresso financeiro</CardTitle>
              </CardHeader>
              <CardContent className="pb-5">
                <p className="text-sm text-foreground">
                  Realizado {moeda(Number(v.custo_realizado) || 0)}
                </p>
                <p className="mt-1 text-sm text-muted">
                  Orçamento {moeda(Number(v.orcamento) || 0)}
                </p>
                {financeiro == null ? (
                  <p className="mt-3 text-sm text-muted">Sem orçamento previsto.</p>
                ) : (
                  <>
                    <p
                      className={`mt-3 text-2xl font-semibold ${financeiro > 100 ? "text-danger" : "text-foreground"}`}
                    >
                      {financeiro}%
                    </p>
                    <Barra
                      rotulo="Progresso financeiro"
                      percentual={financeiro}
                      tom={financeiro > 100 ? "danger" : "primary"}
                    />
                  </>
                )}
              </CardContent>
            </Card>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <Bloco titulo="O que" texto={texto(v.descricao ? `${v.titulo}\n\n${v.descricao}` : v.titulo)} />
            <Bloco titulo="Por que" texto={texto(v.objetivo)} />
            <Bloco titulo="Onde" texto={texto(v.onde)} />
            <Bloco
              titulo="Quando"
              texto={
                v.data_inicio_planejada || v.data_fim_planejada
                  ? `${dia(v.data_inicio_planejada) || "sem início"} até ${dia(v.data_fim_planejada) || "sem término"}`
                  : "Não informado."
              }
            />
            <Bloco
              titulo="Quem"
              texto={
                [
                  v.gerente_nome ? `Responsável: ${v.gerente_nome}` : "",
                  v.patrocinador ? `Patrocinador: ${v.patrocinador}` : "",
                ]
                  .filter(Boolean)
                  .join("\n") || "Não informado."
              }
            />
            <Bloco
              titulo="Como"
              texto={
                [
                  v.metodologia,
                  v.etapas_how > 0 ? `${v.etapas_how} etapas no quadro de tarefas.` : "",
                ]
                  .filter(Boolean)
                  .join("\n\n") || "Não informado."
              }
            />
            <Bloco
              titulo="Quanto"
              texto={
                [
                  `Orçamento: ${moeda(Number(v.orcamento) || 0)}`,
                  v.centro_custo ? `Centro de custo: ${v.centro_custo}` : "",
                ]
                  .filter(Boolean)
                  .join("\n")
              }
            />
          </div>

          {editando && (
            <EditorCarta
              visao={v}
              pending={pending}
              onClose={() => setEditando(false)}
              onSalvar={async (corpo) => {
                const ok = await run(
                  () => apiClient.put(`v1/projetos/${projetoId}/carta`, corpo),
                  "Escopo 5W2H atualizado",
                );
                if (ok) {
                  setEditando(false);
                  void carta.mutate();
                  onSalvo();
                }
              }}
            />
          )}
        </div>
      )}
    </DataState>
  );
}

function Bloco({ titulo, texto: corpo }: { titulo: string; texto: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">{titulo}</CardTitle>
      </CardHeader>
      <CardContent className="pb-5">
        <p className="whitespace-pre-wrap text-sm text-foreground">{corpo}</p>
      </CardContent>
    </Card>
  );
}

function EditorCarta({
  visao,
  pending,
  onClose,
  onSalvar,
}: {
  visao: VisaoCarta;
  pending: boolean;
  onClose: () => void;
  onSalvar: (corpo: Record<string, unknown>) => Promise<void>;
}) {
  const [titulo, setTitulo] = useState(visao.titulo);
  const [descricao, setDescricao] = useState(visao.descricao);
  const [objetivo, setObjetivo] = useState(visao.objetivo);
  const [onde, setOnde] = useState(visao.onde);
  const [inicio, setInicio] = useState(diaInput(visao.data_inicio_planejada));
  const [fim, setFim] = useState(diaInput(visao.data_fim_planejada));
  const [gerente, setGerente] = useState({ id: visao.gerente_id ?? "", nome: visao.gerente_nome });
  const [patrocinador, setPatrocinador] = useState(visao.patrocinador);
  const [metodologia, setMetodologia] = useState(visao.metodologia);
  const [orcamento, setOrcamento] = useState(String(visao.orcamento ?? 0));
  const [centro, setCentro] = useState(visao.centro_custo);

  function enviar(e: FormEvent) {
    e.preventDefault();
    void onSalvar({
      titulo: titulo.trim(),
      descricao: descricao.trim(),
      objetivo: objetivo.trim(),
      onde: onde.trim(),
      patrocinador: patrocinador.trim(),
      metodologia: metodologia.trim(),
      centro_custo: centro.trim(),
      data_inicio_planejada: diaIso(inicio),
      data_fim_planejada: diaIso(fim),
      gerente_id: gerente.id || null,
      gerente_nome: gerente.nome.trim(),
      orcamento: Number(orcamento) || 0,
    });
  }

  return (
    <Dialog open title="Editar escopo 5W2H" onClose={onClose} size="lg">
      <form onSubmit={enviar} className="mt-3 grid gap-3">
        <Input id="carta-titulo" label="Nome do projeto *" value={titulo} onChange={(e) => setTitulo(e.target.value)} maxLength={160} required />
        <Textarea id="carta-descricao" label="Escopo e entregáveis" value={descricao} onChange={(e) => setDescricao(e.target.value)} rows={3} maxLength={4000} />
        <Textarea id="carta-objetivo" label="Por que" value={objetivo} onChange={(e) => setObjetivo(e.target.value)} rows={3} maxLength={4000} />
        <Textarea id="carta-onde" label="Onde" value={onde} onChange={(e) => setOnde(e.target.value)} rows={2} maxLength={2000} />
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Input id="carta-inicio" label="Início" type="date" value={inicio} onChange={(e) => setInicio(e.target.value)} />
          <Input id="carta-fim" label="Término" type="date" value={fim} onChange={(e) => setFim(e.target.value)} />
        </div>
        <PessoaDaUnidade
          id="carta-gerente"
          label="Responsável"
          unidadeId={visao.unidade_id}
          value={gerente}
          onChange={setGerente}
        />
        <Input id="carta-patrocinador" label="Patrocinador" value={patrocinador} onChange={(e) => setPatrocinador(e.target.value)} maxLength={400} />
        <Textarea id="carta-metodologia" label="Como" value={metodologia} onChange={(e) => setMetodologia(e.target.value)} rows={3} maxLength={4000} />
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <Input id="carta-orcamento" label="Orçamento (R$)" type="number" min="0" step="0.01" value={orcamento} onChange={(e) => setOrcamento(e.target.value)} />
          <Input id="carta-centro" label="Centro de custo" value={centro} onChange={(e) => setCentro(e.target.value)} maxLength={120} />
        </div>
        <div className="flex justify-end gap-2 pt-1">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancelar
          </Button>
          <Button type="submit" loading={pending}>
            Salvar escopo
          </Button>
        </div>
      </form>
    </Dialog>
  );
}
