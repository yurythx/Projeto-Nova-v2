"use client";

import { type FormEvent, useState } from "react";

import { AlertasDoMonitoramentoCard } from "@/components/atendimento/AlertasDoMonitoramentoCard";
import { EtapasDeAprovacao } from "@/components/atendimento/EtapasDeAprovacao";
import { TarefasDaOferta } from "@/components/atendimento/TarefasDaOferta";
import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import {
  rotuloTipo,
  type CampoOferta,
  type Oferta,
  type Pessoa,
} from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

interface Produto {
  id: string;
  nome: string;
}

interface Equipe {
  id: string;
  nome: string;
  unidade_id?: string;
  membros: string[];
  ola_horas?: number;
}

function texto(fd: FormData, nome: string) {
  return String(fd.get(nome) ?? "").trim();
}

function idOuNulo(fd: FormData, nome: string) {
  return texto(fd, nome) || null;
}

const opcoesTipo = [
  {
    value: "requisicao",
    label: "Requisição — o servidor pede algo (acesso, equipamento, instalação)",
  },
  { value: "incidente", label: "Incidente — algo parou de funcionar" },
];

export default function AtendimentoCadastroPage() {
  const { can } = usePlataforma();
  const gerencia = can("atendimento:manage");
  const ofertas = useApiQuery<Oferta[]>("v1/atendimento/ofertas");
  const equipes = useApiQuery<Equipe[]>("v1/atendimento/equipes");
  const produtos = useApiQuery<Produto[]>(can("produtos:manage") ? "v1/produtos" : null);
  const { run, pending } = useAction();
  const [editOferta, setEditOferta] = useState<Oferta | null>(null);
  const [etapasDe, setEtapasDe] = useState<Oferta | null>(null);
  const [tarefasDe, setTarefasDe] = useState<Oferta | null>(null);
  const [campos, setCampos] = useState<CampoOferta[]>([]);
  const [editEquipe, setEditEquipe] = useState<Equipe | null>(null);
  const [unidadeEquipe, setUnidadeEquipe] = useState("");
  const candidatos = useApiQuery<Pessoa[]>(
    unidadeEquipe ? `v1/atendimento/candidatos?unidade_id=${unidadeEquipe}` : null,
  );

  async function salvarOferta(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const editando = editOferta;
    const perguntas = campos
      .map((c) => ({
        ...c,
        rotulo: c.rotulo.trim(),
        opcoes: c.tipo === "lista" ? c.opcoes?.map((o) => o.trim()).filter(Boolean) : undefined,
      }))
      .filter((c) => c.rotulo !== "");
    const body: Record<string, unknown> = {
      nome: texto(fd, "nome"),
      descricao: texto(fd, "descricao"),
      tipo: texto(fd, "tipo"),
      produto_id: idOuNulo(fd, "produto_id"),
      unidade_id: idOuNulo(fd, "unidade_id"),
      ativa: fd.get("ativa") === "on",
    };
    if (editando || perguntas.length > 0) body.campos = perguntas;
    const ok = await run(
      () =>
        editando
          ? apiClient.put(`v1/atendimento/ofertas/${editando.id}`, body)
          : apiClient.post("v1/atendimento/ofertas", body),
      editando ? "Oferta atualizada" : "Oferta cadastrada",
    );
    if (ok) {
      setEditOferta(null);
      setCampos([]);
      if (!editando) form.reset();
      void ofertas.mutate();
    }
  }

  async function salvarEquipe(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const editando = editEquipe;
    const body = {
      nome: texto(fd, "nome"),
      unidade_id: idOuNulo(fd, "unidade_id"),
      membros: fd.getAll("membros").map(String),
      ola_horas: texto(fd, "ola_horas") ? Number(texto(fd, "ola_horas")) : null,
    };
    const ok = await run(
      () =>
        editando
          ? apiClient.put(`v1/atendimento/equipes/${editando.id}`, body)
          : apiClient.post("v1/atendimento/equipes", body),
      editando ? "Equipe atualizada" : "Equipe cadastrada",
    );
    if (ok) {
      setEditEquipe(null);
      setUnidadeEquipe("");
      if (!editando) form.reset();
      void equipes.mutate();
    }
  }

  const opcoesProduto = [
    { value: "", label: "Sem produto vinculado" },
    ...(produtos.data ?? []).map((produto) => ({ value: produto.id, label: produto.nome })),
  ];
  const listaCandidatos = candidatos.data ?? [];

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Atendimento"
        title="Ofertas e equipes"
        description="O que o servidor pode pedir, em linguagem simples, e as equipes que atendem cada unidade. Só membros de uma equipe da unidade podem assumir as solicitações dela."
      />

      {gerencia && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">{editOferta ? "Editar oferta" : "Nova oferta"}</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <form
              key={editOferta?.id ?? "nova-oferta"}
              onSubmit={salvarOferta}
              className="mt-3 grid gap-3"
            >
              <Input
                id="oferta-nome"
                name="nome"
                label="Nome *"
                required
                maxLength={160}
                defaultValue={editOferta?.nome}
                autoComplete="off"
              />
              <Textarea
                id="oferta-descricao"
                name="descricao"
                label="Descrição"
                maxLength={4000}
                defaultValue={editOferta?.descricao}
                rows={3}
              />
              <Select
                id="oferta-tipo"
                name="tipo"
                label="Tipo"
                defaultValue={editOferta?.tipo ?? "requisicao"}
                options={opcoesTipo}
              />
              <Select
                id="oferta-produto"
                name="produto_id"
                label="Produto"
                defaultValue={editOferta?.produto_id ?? ""}
                options={opcoesProduto}
              />
              <UnidadeDonaSelect
                id="oferta-unidade"
                permission="atendimento:manage"
                label="Unidade da oferta"
                defaultValue={editOferta?.unidade_id}
              />
              <label className="flex items-center gap-2 text-sm">
                <input
                  id="oferta-ativa"
                  name="ativa"
                  type="checkbox"
                  defaultChecked={editOferta ? editOferta.ativa : true}
                  className="h-4 w-4 accent-primary"
                />
                Oferta ativa
              </label>
              <fieldset className="grid gap-2 rounded-lg border border-surface-border p-3">
                <legend className="px-1 text-sm font-medium">Perguntas na abertura</legend>
                {campos.map((campo, i) => (
                  <div key={i} className="grid gap-2 sm:grid-cols-[1fr_8rem_auto]">
                    <Input
                      id={`campo-rotulo-${i}`}
                      label={i === 0 ? "Rótulo" : undefined}
                      aria-label={`Rótulo da pergunta ${i + 1}`}
                      value={campo.rotulo}
                      maxLength={80}
                      onChange={(e) =>
                        setCampos(
                          campos.map((c, j) => (j === i ? { ...c, rotulo: e.target.value } : c)),
                        )
                      }
                    />
                    <Select
                      id={`campo-tipo-${i}`}
                      label={i === 0 ? "Tipo" : undefined}
                      aria-label={`Tipo da pergunta ${i + 1}`}
                      value={campo.tipo}
                      onChange={(e) =>
                        setCampos(
                          campos.map((c, j) =>
                            j === i ? { ...c, tipo: e.target.value as CampoOferta["tipo"] } : c,
                          ),
                        )
                      }
                      options={[
                        { value: "texto", label: "Texto" },
                        { value: "numero", label: "Número" },
                        { value: "lista", label: "Lista" },
                      ]}
                    />
                    <label className="flex items-end gap-2 pb-2 text-sm">
                      <input
                        type="checkbox"
                        checked={campo.obrigatorio}
                        aria-label={`Pergunta ${i + 1} obrigatória`}
                        onChange={(e) =>
                          setCampos(
                            campos.map((c, j) =>
                              j === i ? { ...c, obrigatorio: e.target.checked } : c,
                            ),
                          )
                        }
                      />
                      Obrigatória
                    </label>
                    {campo.tipo === "lista" ? (
                      <div className="sm:col-span-3">
                        <Input
                          id={`campo-opcoes-${i}`}
                          label="Opções, separadas por vírgula"
                          value={(campo.opcoes ?? []).join(", ")}
                          onChange={(e) =>
                            setCampos(
                              campos.map((c, j) =>
                                j === i ? { ...c, opcoes: e.target.value.split(",") } : c,
                              ),
                            )
                          }
                        />
                      </div>
                    ) : null}
                  </div>
                ))}
                <Button
                  type="button"
                  variant="secondary"
                  disabled={campos.length >= 20}
                  onClick={() =>
                    setCampos([
                      ...campos,
                      { chave: "", rotulo: "", tipo: "texto", obrigatorio: false },
                    ])
                  }
                >
                  Adicionar pergunta
                </Button>
              </fieldset>
              <div className="flex flex-wrap gap-2">
                <Button type="submit" loading={pending}>
                  {editOferta ? "Salvar oferta" : "Cadastrar oferta"}
                </Button>
                {editOferta && (
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() => {
                      setEditOferta(null);
                      setCampos([]);
                    }}
                  >
                    Cancelar edição da oferta
                  </Button>
                )}
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <DataState
        loading={ofertas.isLoading}
        error={ofertas.error}
        onRetry={() => void ofertas.mutate()}
        empty={(ofertas.data ?? []).length === 0}
        emptyTitle="Nenhuma oferta cadastrada"
      >
        <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
          {(ofertas.data ?? []).map((oferta) => (
            <li
              key={oferta.id}
              className="flex flex-col gap-2 p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div>
                <p className="font-medium">{oferta.nome}</p>
                {oferta.descricao ? <p className="text-sm text-muted">{oferta.descricao}</p> : null}
                <p className="text-xs text-muted">
                  {rotuloTipo(oferta.tipo)} ·{" "}
                  {oferta.produto_id ? "Vinculada a um produto" : "Sem produto"}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Badge>{oferta.ativa ? "Ativa" : "Inativa"}</Badge>
                {gerencia && (
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() => {
                      setEditOferta(oferta);
                      setCampos(oferta.campos ?? []);
                    }}
                  >
                    Editar {oferta.nome}
                  </Button>
                )}
                {gerencia && oferta.tipo === "requisicao" && (
                  <Button type="button" variant="secondary" onClick={() => setEtapasDe(oferta)}>
                    Aprovação de {oferta.nome}
                  </Button>
                )}
                {gerencia && oferta.tipo === "requisicao" && (
                  <Button type="button" variant="secondary" onClick={() => setTarefasDe(oferta)}>
                    Tarefas de {oferta.nome}
                  </Button>
                )}
              </div>
            </li>
          ))}
        </ul>
      </DataState>

      {etapasDe && (
        <EtapasDeAprovacao key={etapasDe.id} oferta={etapasDe} onFechar={() => setEtapasDe(null)} />
      )}
      {tarefasDe && (
        <TarefasDaOferta
          key={tarefasDe.id}
          oferta={tarefasDe}
          equipes={equipes.data ?? []}
          onFechar={() => setTarefasDe(null)}
        />
      )}

      {gerencia && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">{editEquipe ? "Editar equipe" : "Nova equipe"}</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <form
              key={editEquipe?.id ?? "nova-equipe"}
              onSubmit={salvarEquipe}
              className="mt-3 grid gap-3"
            >
              <Input
                id="equipe-nome"
                name="nome"
                label="Nome *"
                required
                maxLength={160}
                defaultValue={editEquipe?.nome}
                autoComplete="off"
              />
              <Input
                id="equipe-ola"
                name="ola_horas"
                type="number"
                min={1}
                max={8760}
                label="Prazo da equipe para as tarefas (OLA, em horas corridas)"
                defaultValue={editEquipe?.ola_horas}
              />
              <UnidadeDonaSelect
                id="equipe-unidade"
                permission="atendimento:manage"
                label="Unidade da equipe"
                defaultValue={editEquipe?.unidade_id}
                onChange={setUnidadeEquipe}
              />
              {!unidadeEquipe && (
                <p className="text-sm text-muted">
                  Escolha a unidade para ver quem pode entrar na equipe.
                </p>
              )}
              {unidadeEquipe && !candidatos.isLoading && listaCandidatos.length === 0 && (
                <p className="text-sm text-muted">Nenhum servidor lotado nesta unidade.</p>
              )}
              {listaCandidatos.length > 0 && (
                <fieldset className="flex flex-col gap-2">
                  <legend className="text-sm font-medium">Membros</legend>
                  {listaCandidatos.map((candidato) => (
                    <label key={candidato.id} className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        name="membros"
                        value={candidato.id}
                        defaultChecked={editEquipe?.membros.includes(candidato.id)}
                        className="h-4 w-4 accent-primary"
                      />
                      {candidato.nome}
                    </label>
                  ))}
                </fieldset>
              )}
              <div className="flex flex-wrap gap-2">
                <Button type="submit" loading={pending}>
                  {editEquipe ? "Salvar equipe" : "Cadastrar equipe"}
                </Button>
                {editEquipe && (
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() => {
                      setEditEquipe(null);
                      setUnidadeEquipe("");
                    }}
                  >
                    Cancelar edição da equipe
                  </Button>
                )}
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <DataState
        loading={equipes.isLoading}
        error={equipes.error}
        onRetry={() => void equipes.mutate()}
        empty={(equipes.data ?? []).length === 0}
        emptyTitle="Nenhuma equipe cadastrada"
      >
        <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
          {(equipes.data ?? []).map((equipe) => (
            <li key={equipe.id} className="flex items-center justify-between gap-3 p-4">
              <div>
                <p className="font-medium">{equipe.nome}</p>
                <p className="text-sm text-muted">
                  {equipe.membros.length === 0
                    ? "Sem membros"
                    : `${equipe.membros.length} membro(s)`}
                </p>
              </div>
              {gerencia && (
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => {
                    setEditEquipe(equipe);
                    setUnidadeEquipe(equipe.unidade_id ?? "");
                  }}
                >
                  Editar {equipe.nome}
                </Button>
              )}
            </li>
          ))}
        </ul>
      </DataState>

      {gerencia && <AlertasDoMonitoramentoCard ofertas={ofertas.data ?? []} />}
    </div>
  );
}
