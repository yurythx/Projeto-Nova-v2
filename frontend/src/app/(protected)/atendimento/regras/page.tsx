"use client";

import { type FormEvent, useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { Oferta } from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

interface Equipe {
  id: string;
  nome: string;
}

interface Regra {
  id: string;
  nome: string;
  ordem: number;
  palavra?: string;
  prioridade?: string;
  oferta_id?: string;
  equipe_id?: string;
}

interface Recorrencia {
  id: string;
  titulo: string;
  dia: number;
  oferta_id: string;
}

interface Resposta {
  publico?: string;
  interna?: string;
}

function texto(fd: FormData, nome: string) {
  return String(fd.get(nome) ?? "").trim();
}

function idOuNulo(fd: FormData, nome: string) {
  return texto(fd, nome) || null;
}

const prioridades = [
  { value: "", label: "Sem mudança de prioridade" },
  { value: "critica", label: "Crítica" },
  { value: "alta", label: "Alta" },
  { value: "media", label: "Média" },
  { value: "baixa", label: "Baixa" },
];

export default function RegrasPage() {
  const { can } = usePlataforma();
  const gerencia = can("atendimento:manage");
  const regras = useApiQuery<Regra[]>(gerencia ? "v1/atendimento/regras" : null);
  const recorrencias = useApiQuery<Recorrencia[]>(gerencia ? "v1/atendimento/recorrencias" : null);
  const ofertas = useApiQuery<Oferta[]>(gerencia ? "v1/atendimento/ofertas" : null);
  const equipes = useApiQuery<Equipe[]>(gerencia ? "v1/atendimento/equipes" : null);
  const { run, pending } = useAction();
  const [ofertaResposta, setOfertaResposta] = useState("");
  // O que se digita fica num rascunho da oferta escolhida; sem rascunho,
  // os campos mostram o que está gravado.
  const [rascunho, setRascunho] = useState<{ oferta: string; publico?: string; interna?: string }>({ oferta: "" });
  const resposta = useApiQuery<Resposta>(
    ofertaResposta ? `v1/atendimento/ofertas/${ofertaResposta}/respostas` : null,
  );
  const opcoesOferta = [
    { value: "", label: "Qualquer oferta" },
    ...(ofertas.data ?? []).map((o) => ({ value: o.id, label: o.nome })),
  ];
  const opcoesEquipe = [
    { value: "", label: "Não escolhe equipe" },
    ...(equipes.data ?? []).map((e) => ({ value: e.id, label: e.nome })),
  ];

  const doRascunho = rascunho.oferta === ofertaResposta ? rascunho : { oferta: ofertaResposta };
  const publico = doRascunho.publico ?? resposta.data?.publico ?? "";
  const interna = doRascunho.interna ?? resposta.data?.interna ?? "";

  async function criarRegra(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(
      () =>
        apiClient.post("v1/atendimento/regras", {
          nome: texto(fd, "nome"),
          ordem: Number(fd.get("ordem") || 0),
          palavra: texto(fd, "palavra"),
          prioridade: texto(fd, "prioridade"),
          oferta_id: idOuNulo(fd, "oferta_id"),
          equipe_id: idOuNulo(fd, "equipe_id"),
          unidade_id: idOuNulo(fd, "unidade_id"),
          unidade_solicitante_id: idOuNulo(fd, "unidade_solicitante_id"),
        }),
      "Regra gravada",
    );
    if (ok) {
      form.reset();
      void regras.mutate();
    }
  }

  async function criarRecorrencia(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(
      () =>
        apiClient.post("v1/atendimento/recorrencias", {
          oferta_id: texto(fd, "oferta_id"),
          titulo: texto(fd, "titulo"),
          descricao: texto(fd, "descricao"),
          dia: Number(fd.get("dia")),
        }),
      "Chamado mensal gravado",
    );
    if (ok) {
      form.reset();
      void recorrencias.mutate();
    }
  }

  async function salvarResposta(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!ofertaResposta) return;
    const ok = await run(
      () =>
        apiClient.put(`v1/atendimento/ofertas/${ofertaResposta}/respostas`, {
          publico: publico.trim(),
          interna: interna.trim(),
        }),
      publico.trim() || interna.trim() ? "Resposta pronta gravada" : "Resposta pronta removida",
    );
    if (ok) void resposta.mutate();
  }

  if (!gerencia) {
    return <EmptyState title="Seu perfil não cadastra a triagem" description="Peça atendimento:manage na unidade." />;
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Atendimento"
        title="Triagem"
        description="A primeira regra que casa define a prioridade e avisa a equipe. A resposta pronta só entra na caixa quando alguém cola. O chamado mensal abre uma vez por mês, no dia de 1 a 28."
      />

      <Card>
        <CardHeader>
          <CardTitle as="h2">Regra</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 pb-5">
          <form onSubmit={(ev) => void criarRegra(ev)} className="grid gap-3 sm:grid-cols-2">
            <Input id="regra-nome" name="nome" label="Nome *" required maxLength={160} />
            <Input id="regra-ordem" name="ordem" type="number" min={0} max={999} label="Ordem" defaultValue={10} />
            <Input id="regra-palavra" name="palavra" label="Palavra no título" maxLength={80} />
            <Select id="regra-prioridade" name="prioridade" label="Prioridade" options={prioridades} />
            <Select id="regra-oferta" name="oferta_id" label="Oferta" options={opcoesOferta} />
            <Select id="regra-equipe" name="equipe_id" label="Equipe avisada" options={opcoesEquipe} />
            <UnidadeDonaSelect id="regra-unidade" name="unidade_id" permission="atendimento:manage" label="Unidade da regra" optional />
            <UnidadeDonaSelect
              id="regra-solicitante"
              name="unidade_solicitante_id"
              permission="atendimento:manage"
              label="Unidade de quem pede"
              optional
            />
            <div className="sm:col-span-2">
              <Button type="submit" loading={pending}>
                Gravar regra
              </Button>
            </div>
          </form>
          <DataState
            loading={regras.isLoading}
            error={regras.error}
            onRetry={() => void regras.mutate()}
            empty={(regras.data ?? []).length === 0}
            emptyTitle="Nenhuma regra"
          >
            <ul className="divide-y divide-surface-border text-sm">
              {(regras.data ?? []).map((regra) => (
                <li key={regra.id} className="flex items-center justify-between gap-2 py-2">
                  <span>
                    {regra.ordem}. {regra.nome}
                    {regra.palavra ? ` · “${regra.palavra}”` : ""}
                    {regra.prioridade ? ` · ${regra.prioridade}` : ""}
                  </span>
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() =>
                      void run(
                        () => apiClient.delete(`v1/atendimento/regras/${regra.id}`),
                        "Regra removida",
                      ).then(() => regras.mutate())
                    }
                  >
                    Remover {regra.nome}
                  </Button>
                </li>
              ))}
            </ul>
          </DataState>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle as="h2">Resposta pronta</CardTitle>
        </CardHeader>
        <CardContent className="pb-5">
          <form onSubmit={(ev) => void salvarResposta(ev)} className="grid gap-3">
            <Select
              id="resposta-oferta"
              label="Oferta da resposta *"
              placeholder="Escolha a oferta"
              value={ofertaResposta}
              onChange={(ev) => {
                setOfertaResposta(ev.target.value);
                setRascunho({ oferta: ev.target.value });
              }}
              options={(ofertas.data ?? []).map((o) => ({ value: o.id, label: o.nome }))}
            />
            <Textarea
              id="resposta-publica"
              label="Resposta pública"
              maxLength={4000}
              rows={3}
              value={publico}
              onChange={(ev) => setRascunho({ ...doRascunho, publico: ev.target.value })}
            />
            <Textarea
              id="resposta-interna"
              label="Nota interna"
              maxLength={4000}
              rows={3}
              value={interna}
              onChange={(ev) => setRascunho({ ...doRascunho, interna: ev.target.value })}
            />
            <div>
              <Button type="submit" loading={pending} disabled={!ofertaResposta}>
                Gravar resposta
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle as="h2">Chamado mensal</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 pb-5">
          <form onSubmit={(ev) => void criarRecorrencia(ev)} className="grid gap-3 sm:grid-cols-2">
            <Select
              id="recorrencia-oferta"
              name="oferta_id"
              label="Oferta *"
              required
              options={(ofertas.data ?? []).map((o) => ({ value: o.id, label: o.nome }))}
            />
            <Input id="recorrencia-titulo" name="titulo" label="Título *" required maxLength={160} />
            <Input id="recorrencia-dia" name="dia" type="number" min={1} max={28} label="Dia do mês *" required defaultValue={1} />
            <Textarea id="recorrencia-descricao" name="descricao" label="Descrição" maxLength={4000} rows={2} />
            <div className="sm:col-span-2">
              <Button type="submit" loading={pending}>
                Gravar chamado mensal
              </Button>
            </div>
          </form>
          <ul className="divide-y divide-surface-border text-sm">
            {(recorrencias.data ?? []).map((item) => (
              <li key={item.id} className="flex items-center justify-between gap-2 py-2">
                <span>
                  Dia {item.dia} · {item.titulo}
                </span>
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() =>
                    void run(
                      () => apiClient.delete(`v1/atendimento/recorrencias/${item.id}`),
                      "Chamado mensal removido",
                    ).then(() => recorrencias.mutate())
                  }
                >
                  Remover {item.titulo}
                </Button>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>
    </div>
  );
}
