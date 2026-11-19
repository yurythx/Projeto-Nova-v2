"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { rotuloStatus } from "@/components/protocolos/painel";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { OrgTree, UserAdmin } from "@/lib/plataforma/types";

interface Formulario {
  id: string;
  titulo: string;
}

interface Servico {
  id: string;
  nome: string;
  categoria: string;
  ativo: boolean;
  prazo_dias: number;
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

const STATUS = [
  "novo",
  "em_andamento",
  "pendente",
  "em_analise",
  "em_complementacao",
  "pending_approval",
  "aprovado",
  "rejeitado",
  "concluido",
  "cancelado",
].map((status) => ({ value: status, label: rotuloStatus(status) }));

const RESPONSAVEL = [
  { value: "", label: "Sem responsável fixo" },
  { value: "unidade", label: "Unidade" },
  { value: "departamento", label: "Departamento" },
  { value: "usuario", label: "Usuário" },
];

const DIAS = ["Dom", "Seg", "Ter", "Qua", "Qui", "Sex", "Sáb"];

export default function ServicosPage() {
  const { can } = usePlataforma();
  const servicos = useApiQuery<Servico[]>("v1/protocolos/servicos");
  const forms = useApiQuery<Formulario[]>("v1/protocolos/formularios");
  const { run, pending } = useAction();
  const [exige, setExige] = useState(false);
  const [fluxo, setFluxo] = useState(false);
  const [dias, setDias] = useState<number[]>([1, 2, 3, 4, 5]);
  const [requisitos, setRequisitos] = useState<Orientacao[]>([]);
  const [documentos, setDocumentos] = useState<Orientacao[]>([]);
  const [etapas, setEtapas] = useState<Etapa[]>([
    { id: "analise", ordem: 1, status: "em_analise", prazo_dias: 5 },
  ]);

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const dados = new FormData(e.currentTarget);
    const taxa = Math.max(0, Math.round(Number(dados.get("taxa") ?? 0) * 100));
    const ok = await run(
      () =>
        apiClient.post("v1/protocolos/servicos", {
          unidade_id: String(dados.get("unidade_id") ?? ""),
          formulario_id: String(dados.get("formulario_id") ?? ""),
          nome: String(dados.get("nome") ?? "").trim(),
          descricao: String(dados.get("descricao") ?? "").trim(),
          categoria: String(dados.get("categoria") ?? "").trim(),
          prazo_dias: Number(dados.get("prazo_dias") ?? 0),
          taxa_centavos: Number.isFinite(taxa) ? taxa : 0,
          destaque: dados.get("destaque") === "on",
          aprova_sozinho: dados.get("aprova_sozinho") === "on",
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
      "Serviço salvo",
    );
    if (ok) {
      e.currentTarget.reset();
      setExige(false);
      setFluxo(false);
      setDias([1, 2, 3, 4, 5]);
      setRequisitos([]);
      setDocumentos([]);
      void servicos.mutate();
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Protocolos"
        title="Serviços"
        description="O serviço usa um formulário da mesma unidade. Requisitos, documentos, taxa, horário e etapas acompanham o pedido."
        actions={
          <Link href="/protocolos" className={buttonClass("secondary", "sm")}>
            Mesa
          </Link>
        }
      />
      {can("protocolos:gerir") && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Novo serviço</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <form onSubmit={salvar} className="grid gap-3 sm:grid-cols-2">
              <UnidadeDonaSelect id="unidade-servico" permission="protocolos:gerir" exigir />
              <Select
                id="formulario_id"
                name="formulario_id"
                label="Formulário *"
                required
                placeholder="Selecione…"
                options={(forms.data ?? []).map((form) => ({ value: form.id, label: form.titulo }))}
              />
              <Input id="nome" name="nome" label="Nome *" required maxLength={200} />
              <Input id="categoria" name="categoria" label="Categoria" maxLength={80} />
              <Input
                id="prazo_dias"
                name="prazo_dias"
                label="Prazo em dias úteis"
                type="number"
                min={0}
                defaultValue={0}
              />
              <Input
                id="taxa"
                name="taxa"
                label="Taxa (R$)"
                type="number"
                min={0}
                step="0.01"
                defaultValue={0}
              />
              <Input
                id="campo_anonimo"
                name="campo_anonimo"
                label="Campo que anonimiza"
                maxLength={40}
              />
              <Textarea id="descricao" name="descricao" label="Descrição" maxLength={4000} />
              <Marcador name="destaque" texto="Destaque na carta" />
              <Marcador name="aprova_sozinho" texto="Aprova ao abrir, quando não há etapa" />
              <Marcador name="so_responsavel_tecnico" texto="Somente responsável técnico ativo" />
              <Marcador name="impedir_cpf_aberto" texto="Impede outro pedido aberto do mesmo CPF" />
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  className="h-4 w-4 accent-primary"
                  checked={exige}
                  onChange={(e) => setExige(e.target.checked)}
                />
                Exige horário
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  className="h-4 w-4 accent-primary"
                  checked={fluxo}
                  onChange={(e) => setFluxo(e.target.checked)}
                />
                Workflow ligado
              </label>
              {exige && (
                <div className="flex flex-col gap-3 sm:col-span-2">
                  <p className="text-sm font-medium">Grade</p>
                  <div className="flex flex-wrap gap-3">
                    {DIAS.map((nome, dia) => (
                      <label key={nome} className="flex items-center gap-2 text-sm">
                        <input
                          type="checkbox"
                          className="h-4 w-4 accent-primary"
                          checked={dias.includes(dia)}
                          onChange={(e) =>
                            setDias((atual) =>
                              e.target.checked
                                ? [...atual, dia].sort()
                                : atual.filter((item) => item !== dia),
                            )
                          }
                        />
                        {nome}
                      </label>
                    ))}
                  </div>
                  <div className="grid gap-3 sm:grid-cols-4">
                    <Input
                      id="inicio"
                      name="inicio"
                      label="Início"
                      type="time"
                      defaultValue="08:00"
                      required
                    />
                    <Input
                      id="fim"
                      name="fim"
                      label="Fim"
                      type="time"
                      defaultValue="17:00"
                      required
                    />
                    <Input
                      id="duracao_min"
                      name="duracao_min"
                      label="Duração (min)"
                      type="number"
                      min={1}
                      defaultValue={30}
                      required
                    />
                    <Input
                      id="vagas_dia"
                      name="vagas_dia"
                      label="Vagas no dia"
                      type="number"
                      min={1}
                      defaultValue={8}
                      required
                    />
                  </div>
                </div>
              )}
              {fluxo && <ListaEtapas etapas={etapas} onChange={setEtapas} />}
              <ListaOrientacao titulo="Requisitos" itens={requisitos} onChange={setRequisitos} />
              <ListaOrientacao
                titulo="Documentos esperados"
                itens={documentos}
                onChange={setDocumentos}
              />
              <div className="sm:col-span-2">
                <Button type="submit" loading={pending}>
                  Salvar
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}
      <DataState
        loading={servicos.isLoading}
        error={servicos.error}
        onRetry={() => void servicos.mutate()}
        empty={(servicos.data ?? []).length === 0}
        emptyTitle="Nenhum serviço"
      >
        <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
          {(servicos.data ?? []).map((servico) => (
            <li key={servico.id} className="flex items-center justify-between gap-3 p-4">
              <div>
                <p className="font-medium">{servico.nome}</p>
                <p className="text-sm text-muted">
                  {servico.categoria || "sem categoria"} · {servico.prazo_dias} dias úteis
                </p>
              </div>
              <Badge>{servico.ativo ? "ativo" : "inativo"}</Badge>
            </li>
          ))}
        </ul>
      </DataState>
    </div>
  );
}

function Marcador({ name, texto }: { name: string; texto: string }) {
  return (
    <label className="flex items-center gap-2 text-sm">
      <input type="checkbox" name={name} className="h-4 w-4 accent-primary" />
      {texto}
    </label>
  );
}

function ListaEtapas({
  etapas,
  onChange,
}: {
  etapas: Etapa[];
  onChange: (etapas: Etapa[]) => void;
}) {
  return (
    <div className="flex flex-col gap-3 sm:col-span-2">
      <p className="text-sm font-medium">Etapas</p>
      {etapas.map((etapa, indice) => (
        <div
          key={indice}
          className="grid gap-3 rounded-lg border border-surface-border p-3 sm:grid-cols-4"
        >
          <Input
            id={`etapa-id-${indice}`}
            label="Identificador *"
            value={etapa.id}
            maxLength={40}
            onChange={(e) =>
              onChange(
                etapas.map((item, i) => (i === indice ? { ...item, id: e.target.value } : item)),
              )
            }
          />
          <Input
            id={`etapa-ordem-${indice}`}
            label="Ordem"
            type="number"
            min={0}
            value={etapa.ordem}
            onChange={(e) =>
              onChange(
                etapas.map((item, i) =>
                  i === indice ? { ...item, ordem: Number(e.target.value) } : item,
                ),
              )
            }
          />
          <Select
            id={`etapa-status-${indice}`}
            label="Status"
            value={etapa.status}
            options={STATUS}
            onChange={(e) =>
              onChange(
                etapas.map((item, i) =>
                  i === indice ? { ...item, status: e.target.value } : item,
                ),
              )
            }
          />
          <Input
            id={`etapa-prazo-${indice}`}
            label="Prazo (dias úteis)"
            type="number"
            min={0}
            value={etapa.prazo_dias}
            onChange={(e) =>
              onChange(
                etapas.map((item, i) =>
                  i === indice ? { ...item, prazo_dias: Number(e.target.value) } : item,
                ),
              )
            }
          />
          <Select
            id={`etapa-tipo-${indice}`}
            label="Responsável"
            value={etapa.responsavel_tipo ?? ""}
            options={RESPONSAVEL}
            onChange={(e) =>
              onChange(
                etapas.map((item, i) =>
                  i === indice
                    ? {
                        ...item,
                        responsavel_tipo: e.target.value,
                        responsavel_id: e.target.value ? item.responsavel_id : "",
                      }
                    : item,
                ),
              )
            }
          />
          <ResponsavelDaEtapa
            etapa={etapa}
            indice={indice}
            onEscolher={(responsavelId) =>
              onChange(
                etapas.map((item, i) =>
                  i === indice ? { ...item, responsavel_id: responsavelId } : item,
                ),
              )
            }
          />
        </div>
      ))}
      <Button
        type="button"
        variant="secondary"
        onClick={() =>
          onChange([
            ...etapas,
            { id: "", ordem: etapas.length + 1, status: "em_andamento", prazo_dias: 0 },
          ])
        }
      >
        Etapa
      </Button>
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
      <p className="text-sm text-muted sm:col-span-2">
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
  return (
    <Select
      id={`etapa-resp-${indice}`}
      label={tipo === "usuario" ? "Usuário" : tipo === "departamento" ? "Departamento" : "Unidade"}
      placeholder="Selecione…"
      value={etapa.responsavel_id ?? ""}
      options={opcoes}
      onChange={(e) => onEscolher(e.target.value)}
    />
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
  return (
    <div className="flex flex-col gap-3 sm:col-span-2">
      <p className="text-sm font-medium">{titulo}</p>
      {itens.map((item, indice) => (
        <div
          key={indice}
          className="grid gap-3 rounded-lg border border-surface-border p-3 sm:grid-cols-[1fr_2fr_auto]"
        >
          <Input
            id={`${titulo}-titulo-${indice}`}
            label="Título *"
            value={item.titulo}
            maxLength={200}
            onChange={(e) =>
              onChange(
                itens.map((atual, i) =>
                  i === indice ? { ...atual, titulo: e.target.value } : atual,
                ),
              )
            }
          />
          <Input
            id={`${titulo}-texto-${indice}`}
            label="Texto"
            value={item.texto}
            maxLength={2000}
            onChange={(e) =>
              onChange(
                itens.map((atual, i) =>
                  i === indice ? { ...atual, texto: e.target.value } : atual,
                ),
              )
            }
          />
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              className="h-4 w-4 accent-primary"
              checked={item.obrigatorio}
              onChange={(e) =>
                onChange(
                  itens.map((atual, i) =>
                    i === indice ? { ...atual, obrigatorio: e.target.checked } : atual,
                  ),
                )
              }
            />
            Obrigatório
          </label>
        </div>
      ))}
      <Button
        type="button"
        variant="secondary"
        onClick={() => onChange([...itens, { titulo: "", texto: "", obrigatorio: false }])}
      >
        Incluir
      </Button>
    </div>
  );
}
