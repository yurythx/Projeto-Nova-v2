"use client";

import { type FormEvent, useState } from "react";
import { Plus, Trash2 } from "lucide-react";

import { PoliticaForm, sugestao, type PoliticaCorpo } from "@/components/atendimento/PoliticaForm";
import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { ConfirmButton } from "@/components/plataforma/ConfirmButton";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDate, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Dialog } from "@/components/ui/Dialog";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery, withQuery } from "@/lib/api/swr";
import {
  TIPO_FERIADO,
  type Categoria,
  type Feriado,
  type Politica,
  type SLA,
  type TipoFeriado,
} from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

function formatarMinutos(minutos: number): string {
  if (minutos < 60) return `${minutos} min`;
  const horas = minutos / 60;
  if (horas < 24) return `${horas}h`;
  const dias = horas / 24;
  return `${dias}d (${horas}h)`;
}

/** Prazos da unidade: quando se trabalha, SLAs independentes amarradas
 * por categoria (estilo GLPI), como impacto e urgência viram prioridade
 * e os feriados. A subunidade sem política própria usa a institucional. */
export default function AtendimentoPrazosPage() {
  const { can, canGlobal } = usePlataforma();
  const gere = can("atendimento:manage");
  const global = canGlobal("atendimento:manage");
  // Gestão global começa pela institucional; a de unidade escolhe a sua.
  const [escolha, setEscolha] = useState<string | null>(null);
  const unidade = escolha ?? (global ? "" : null);
  const filtro = unidade === null ? null : { unidade_id: unidade || undefined };

  const politica = useApiQuery<Politica | null>(
    filtro ? withQuery("v1/atendimento/politica", filtro) : null,
  );
  const feriados = useApiQuery<Feriado[]>(
    filtro ? withQuery("v1/atendimento/feriados", filtro) : null,
  );
  const slasQuery = useApiQuery<SLA[]>(
    filtro ? withQuery("v1/atendimento/slas", filtro) : null,
  );
  const categoriasQuery = useApiQuery<Categoria[]>(
    gere ? "v1/atendimento/categorias" : null,
  );

  const { run, pending } = useAction();

  // Estados para nova SLA independente (estilo GLPI)
  const [modalSLA, setModalSLA] = useState(false);
  const [slaNome, setSlaNome] = useState("");
  const [slaDescricao, setSlaDescricao] = useState("");
  const [slaRespostaMin, setSlaRespostaMin] = useState(30);
  const [slaSolucaoMin, setSlaSolucaoMin] = useState(240);
  const [slaCalendario, setSlaCalendario] = useState<"expediente" | "corrido_24x7">("expediente");
  const [slaPausar, setSlaPausar] = useState(true);

  if (!gere) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Atendimento" title="Prazos" />
        <EmptyState
          title="Seu perfil não gere o atendimento"
          description="A política de prazo é cadastrada pela gestão do atendimento da unidade."
        />
      </div>
    );
  }

  async function salvar(corpo: PoliticaCorpo) {
    const ok = await run(
      () => apiClient.put("v1/atendimento/politica", { ...corpo, unidade_id: unidade || null }),
      "Política salva",
    );
    if (ok) void politica.mutate();
  }

  async function criarFeriado(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(
      () =>
        apiClient.post("v1/atendimento/feriados", {
          unidade_id: unidade || null,
          data: String(fd.get("data")),
          nome: String(fd.get("nome")).trim(),
          tipo: String(fd.get("tipo")),
        }),
      "Feriado cadastrado",
    );
    if (ok) {
      form.reset();
      void feriados.mutate();
    }
  }

  async function excluirFeriado(f: Feriado) {
    const ok = await run(
      () => apiClient.delete(`v1/atendimento/feriados/${f.id}`),
      "Feriado excluído",
    );
    if (ok) void feriados.mutate();
  }

  async function salvarSLA(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const ok = await run(
      () =>
        apiClient.post("v1/atendimento/slas", {
          nome: slaNome.trim(),
          descricao: slaDescricao.trim(),
          tempo_resposta_minutos: Number(slaRespostaMin),
          tempo_solucao_minutos: Number(slaSolucaoMin),
          calendario_tipo: slaCalendario,
          pausar_em_espera: slaPausar,
          unidade_id: unidade || null,
        }),
      "SLA cadastrada com sucesso",
    );
    if (ok) {
      setModalSLA(false);
      setSlaNome("");
      setSlaDescricao("");
      setSlaRespostaMin(30);
      setSlaSolucaoMin(240);
      void slasQuery.mutate();
    }
  }

  async function excluirSLA(id: string, nome: string) {
    const ok = await run(
      () => apiClient.delete(`v1/atendimento/slas/${id}`),
      `SLA ${nome} excluída`,
    );
    if (ok) {
      void slasQuery.mutate();
      void categoriasQuery.mutate();
    }
  }

  async function vincularSlaCategoria(categoriaId: string, slaId: string) {
    const ok = await run(
      () =>
        apiClient.put(`v1/atendimento/categorias/${categoriaId}/sla`, {
          sla_id: slaId || null,
        }),
      "Amarração de SLA atualizada",
    );
    if (ok) void categoriasQuery.mutate();
  }

  const p = politica.data;
  const propria = p && !p.herdada ? p : null;
  const listaSLAs = slasQuery.data ?? [];
  const listaCategorias = categoriasQuery.data ?? [];
  const slaErro =
    slasQuery.error &&
    (slasQuery.error as { code?: string; status?: number }).code !== "NOT_FOUND" &&
    (slasQuery.error as { code?: string; status?: number }).status !== 404
      ? slasQuery.error
      : null;

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Atendimento"
        title="Prazos e Acordos de Nível de Serviço (SLA)"
        description="Gestão de prazos e amarrações com as categorias do GLPI. O tempo útil é contabilizado conforme expediente e feriados cadastrados."
      />

      <div className="max-w-md">
        <UnidadeDonaSelect
          id="prazos-unidade"
          permission="atendimento:manage"
          label="Unidade"
          defaultValue={escolha}
          onChange={setEscolha}
        />
      </div>

      {filtro === null ? (
        <EmptyState
          title="Escolha a unidade"
          description="A política, feriados e SLAs são definidos por unidade ou institucionalmente."
        />
      ) : (
        <>
          {/* 1. SEÇÃO DE SLAS E AMARRAÇÕES POR CATEGORIA (ESTILO GLPI) */}
          <Card>
            <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
              <div>
                <CardTitle as="h2">SLAs e Amarrações de Categorias (Estilo GLPI)</CardTitle>
                <p className="mt-1 text-sm text-muted">
                  Defina prazos de Primeira Resposta e Solução independentes e amarre-os diretamente
                  às categorias do catálogo.
                </p>
              </div>
              <Button onClick={() => setModalSLA(true)} icon={<Plus size={16} />}>
                Nova SLA
              </Button>
            </CardHeader>
            <CardContent className="flex flex-col gap-6 pb-5">
              {/* Grid de SLAs cadastradas */}
              <div className="flex flex-col gap-3">
                <h3 className="text-sm font-semibold text-foreground">Catálogo de SLAs Oficiais</h3>
                <DataState
                  loading={slasQuery.isLoading}
                  error={slaErro}
                  onRetry={() => void slasQuery.mutate()}
                  empty={listaSLAs.length === 0}
                  emptyTitle="Nenhuma SLA cadastrada"
                >
                  <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                    {listaSLAs.map((sla) => (
                      <div
                        key={sla.id}
                        className="flex flex-col justify-between rounded-xl border border-surface-border bg-surface-card p-4 transition-all hover:border-primary/40"
                      >
                        <div>
                          <div className="flex items-center justify-between gap-2">
                            <span className="font-semibold text-foreground">{sla.nome}</span>
                            <Badge tone={sla.calendario_tipo === "corrido_24x7" ? "warning" : "info"}>
                              {sla.calendario_tipo === "corrido_24x7" ? "24x7" : "Expediente"}
                            </Badge>
                          </div>
                          {sla.descricao && (
                            <p className="mt-1 line-clamp-2 text-xs text-muted">{sla.descricao}</p>
                          )}
                          <div className="mt-3 flex flex-wrap gap-2 text-xs">
                            <span className="inline-flex items-center gap-1 rounded bg-surface px-2 py-1 font-mono text-foreground">
                              <strong>1ª Resp:</strong> {formatarMinutos(sla.tempo_resposta_minutos)}
                            </span>
                            <span className="inline-flex items-center gap-1 rounded bg-surface px-2 py-1 font-mono text-foreground">
                              <strong>Solução:</strong> {formatarMinutos(sla.tempo_solucao_minutos)}
                            </span>
                          </div>
                        </div>
                        <div className="mt-4 flex items-center justify-between border-t border-surface-border pt-3 text-xs text-muted">
                          <span>{sla.pausar_em_espera ? "Pausa em espera" : "Tempo corrido"}</span>
                          <ConfirmButton
                            title={`Excluir SLA ${sla.nome}?`}
                            description="Categorias que usam esta SLA passarão a usar a matriz geral de prazos."
                            confirmLabel="Excluir"
                            onConfirm={() => excluirSLA(sla.id, sla.nome)}
                          >
                            <Trash2 size={14} className="text-danger" />
                          </ConfirmButton>
                        </div>
                      </div>
                    ))}
                  </div>
                </DataState>
              </div>

              {/* Tabela de Amarração com Categorias */}
              {listaCategorias.length > 0 && (
                <div className="flex flex-col gap-3 border-t border-surface-border pt-5">
                  <h3 className="text-sm font-semibold text-foreground">
                    Amarração de SLAs nas Categorias de Chamado
                  </h3>
                  <p className="text-xs text-muted">
                    Quando um chamado é aberto nesta categoria, o prazo de atendimento e solução
                    será atribuído automaticamente pela SLA vinculada.
                  </p>
                  <div className="overflow-x-auto rounded-xl border border-surface-border">
                    <table className="w-full text-left text-sm">
                      <thead className="border-b border-surface-border bg-surface-hover/50 text-xs font-semibold text-muted">
                        <tr>
                          <th className="p-3">Categoria do GLPI / Catálogo</th>
                          <th className="p-3">SLA Vinculada</th>
                          <th className="p-3">Prazos Calculados</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-surface-border">
                        {listaCategorias.map((cat) => {
                          const slaVinculada = listaSLAs.find((s) => s.id === cat.sla_id);
                          return (
                            <tr key={cat.id} className="hover:bg-surface-hover/30">
                              <td className="p-3">
                                <span className="font-medium text-foreground">{cat.nome}</span>
                                {cat.descricao && (
                                  <p className="text-xs text-muted">{cat.descricao}</p>
                                )}
                              </td>
                              <td className="p-3">
                                <div className="max-w-xs">
                                  <Select
                                    id={`categoria-sla-${cat.id}`}
                                    aria-label={`SLA da categoria ${cat.nome}`}
                                    value={cat.sla_id ?? ""}
                                    onChange={(e) => void vincularSlaCategoria(cat.id, e.target.value)}
                                    options={[
                                      { value: "", label: "Nenhuma (Usar Matriz de Prioridade)" },
                                      ...listaSLAs.map((s) => ({
                                        value: s.id,
                                        label: `${s.nome} (${formatarMinutos(s.tempo_solucao_minutos)})`,
                                      })),
                                    ]}
                                  />
                                </div>
                              </td>
                              <td className="p-3">
                                {slaVinculada ? (
                                  <div className="flex items-center gap-2">
                                    <Badge tone="success">
                                      Resp: {formatarMinutos(slaVinculada.tempo_resposta_minutos)}
                                    </Badge>
                                    <Badge tone="info">
                                      Sol: {formatarMinutos(slaVinculada.tempo_solucao_minutos)}
                                    </Badge>
                                  </div>
                                ) : (
                                  <span className="text-xs text-muted">Matriz de Prioridade</span>
                                )}
                              </td>
                            </tr>
                          );
                        })}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Modal Nova SLA */}
          {modalSLA && (
            <Dialog open={modalSLA} onClose={() => setModalSLA(false)} title="Cadastrar Nova SLA">
              <form onSubmit={salvarSLA} className="flex flex-col gap-4">
                <Input
                  id="sla-nome"
                  label="Nome da SLA *"
                  value={slaNome}
                  onChange={(e) => setSlaNome(e.target.value)}
                  placeholder="Ex.: SLA Crítica N1 - Saúde (2h)"
                  required
                  maxLength={120}
                />
                <Textarea
                  id="sla-desc"
                  label="Descrição e Escopo"
                  value={slaDescricao}
                  onChange={(e) => setSlaDescricao(e.target.value)}
                  placeholder="Detalhamento sobre quando esta SLA deve ser aplicada..."
                  rows={2}
                />
                <div className="grid grid-cols-2 gap-3">
                  <Input
                    id="sla-resp"
                    type="number"
                    label="Primeira Resposta (minutos) *"
                    min={1}
                    value={slaRespostaMin}
                    onChange={(e) => setSlaRespostaMin(Number(e.target.value))}
                    required
                  />
                  <Input
                    id="sla-sol"
                    type="number"
                    label="Tempo de Solução (minutos) *"
                    min={1}
                    value={slaSolucaoMin}
                    onChange={(e) => setSlaSolucaoMin(Number(e.target.value))}
                    required
                  />
                </div>
                <div className="grid grid-cols-2 gap-3">
                  <Select
                    id="sla-cal"
                    label="Calendário"
                    value={slaCalendario}
                    onChange={(e) => setSlaCalendario(e.target.value as "expediente" | "corrido_24x7")}
                    options={[
                      { value: "expediente", label: "Horário de Expediente" },
                      { value: "corrido_24x7", label: "Contínuo 24x7" },
                    ]}
                  />
                  <div className="flex flex-col justify-center pt-5">
                    <label className="flex cursor-pointer items-center gap-2 text-sm text-foreground">
                      <input
                        type="checkbox"
                        checked={slaPausar}
                        onChange={(e) => setSlaPausar(e.target.checked)}
                        className="rounded border-surface-border text-primary focus:ring-primary"
                      />
                      Pausar quando aguarda servidor/terceiro
                    </label>
                  </div>
                </div>
                <div className="mt-4 flex justify-end gap-2">
                  <Button variant="ghost" onClick={() => setModalSLA(false)}>
                    Cancelar
                  </Button>
                  <Button type="submit" loading={pending}>
                    Salvar SLA
                  </Button>
                </div>
              </form>
            </Dialog>
          )}

          {/* 2. POLÍTICA DE EXPEDIENTE E MATRIZ */}
          <Card>
            <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
              <CardTitle as="h2">Política de prazo</CardTitle>
              {p?.herdada && <Badge tone="info">Herdada</Badge>}
              {!p && !politica.isLoading && !politica.error && (
                <Badge tone="warning">Sem política</Badge>
              )}
            </CardHeader>
            <CardContent className="flex flex-col gap-4 pb-5">
              <DataState
                loading={politica.isLoading}
                error={politica.error}
                onRetry={() => void politica.mutate()}
                empty={false}
              >
                {p?.herdada && (
                  <p role="status" className="text-sm text-muted">
                    Esta unidade usa a política de uma unidade acima dela. Salvar cria a política
                    própria, a partir desta.
                  </p>
                )}
                {!p && (
                  <p role="status" className="text-sm text-muted">
                    Sem política, as solicitações desta unidade ficam sem prazo. Os valores abaixo
                    são uma sugestão e só valem depois de salvos.
                  </p>
                )}
                <PoliticaForm
                  key={`${unidade}-${p?.id ?? "nova"}-${p?.herdada ?? false}`}
                  inicial={p ?? sugestao()}
                  pending={pending}
                  onSalvar={(corpo) => void salvar(corpo)}
                />
              </DataState>
            </CardContent>
          </Card>

          {/* 3. FERIADOS E PONTOS FACULTATIVOS */}
          <Card>
            <CardHeader>
              <CardTitle as="h2">Feriados e pontos facultativos</CardTitle>
            </CardHeader>
            <CardContent className="flex flex-col gap-4 pb-5">
              <p className="text-sm text-muted">
                Valem para esta unidade e as abaixo dela.
                {propria ? "" : " Os da unidade acima também valem aqui."}
              </p>
              <form
                onSubmit={criarFeriado}
                className="grid gap-3 sm:grid-cols-[10rem_1fr_12rem_auto] sm:items-end"
              >
                <Input id="feriado-data" name="data" type="date" label="Data *" required />
                <Input
                  id="feriado-nome"
                  name="nome"
                  label="Nome *"
                  required
                  maxLength={160}
                  autoComplete="off"
                />
                <Select
                  id="feriado-tipo"
                  name="tipo"
                  label="Tipo"
                  defaultValue="municipal"
                  options={(Object.keys(TIPO_FERIADO) as TipoFeriado[]).map((t) => ({
                    value: t,
                    label: TIPO_FERIADO[t],
                  }))}
                />
                <Button type="submit" loading={pending}>
                  Adicionar
                </Button>
              </form>
              <DataState
                loading={feriados.isLoading}
                error={feriados.error}
                onRetry={() => void feriados.mutate()}
                empty={(feriados.data ?? []).length === 0}
                emptyTitle="Nenhum feriado cadastrado nesta unidade"
              >
                <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border">
                  {(feriados.data ?? []).map((f) => (
                    <li
                      key={f.id}
                      className="flex flex-wrap items-center justify-between gap-2 p-3 text-sm"
                    >
                      <span>
                        <span className="font-medium tabular-nums">
                          {fmtDate(`${f.data}T12:00:00`)}
                        </span>{" "}
                        · {f.nome} <Badge>{TIPO_FERIADO[f.tipo]}</Badge>
                      </span>
                      <ConfirmButton
                        title={`Excluir ${f.nome}?`}
                        description="O dia volta a contar como tempo útil nas próximas medições."
                        confirmLabel="Excluir"
                        onConfirm={() => excluirFeriado(f)}
                      >
                        Excluir {f.nome}
                      </ConfirmButton>
                    </li>
                  ))}
                </ul>
              </DataState>
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}
