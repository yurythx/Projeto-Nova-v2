"use client";

import Link from "next/link";
import { useState } from "react";
import { FileText, Clock, Shield } from "lucide-react";

import { DataState } from "@/components/plataforma/DataState";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { ApiError, apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

import { emCuiaba } from "./horario";
import { nomePrograma, nomeSituacao, OPCOES_SITUACAO, tomSituacao } from "./rotulos";
import { GradeServicos, SERVICOS } from "./Servicos";

interface Processo {
  id: string;
  numero: string;
  programa: string;
  situacao: string;
  titular: string;
  pendencia: string;
  recibo_em: string;
  exigencia_prazo: string;
}

interface Prazo {
  nome: string;
  limite: string;
  situacao: string;
}

const DESTAQUES = [
  {
    titulo: "Protocolo eletrônico",
    texto: "O recibo grava o número e a data legal no horário de Cuiabá.",
    icone: FileText,
  },
  {
    titulo: "Acompanhamento",
    texto: "Situação, prazo e pendência ficam neste protocolo.",
    icone: Clock,
  },
  {
    titulo: "Sigilo",
    texto: "O aviso leva o número do processo e a orientação.",
    icone: Shield,
  },
];

/** Entrada no desenho do protocolo de referência, com os tokens deste sistema. */
export default function FazendaPage() {
  const { can } = usePlataforma();
  const equipe = can("fazenda:analisar") || can("fazenda:decidir");
  const list = useApiQuery<Processo[]>("v1/fazenda/processos");
  const prazos = useApiQuery<Prazo[]>(equipe ? "v1/fazenda/processos/painel" : null);
  const [aviso, setAviso] = useState("");
  const [erro, setErro] = useState("");
  const [busca, setBusca] = useState("");
  const [situacao, setSituacao] = useState("");
  const termo = busca.trim().toLowerCase();
  const processos = (list.data ?? []).filter((item) => {
    const numero = !termo || item.numero.toLowerCase().includes(termo);
    const estado = !situacao || item.situacao === situacao;
    return numero && estado;
  });

  async function publicar() {
    setErro("");
    try {
      const { data } = await apiClient.post<{ publicadas: number }>(
        "v1/fazenda/processos/alertas",
        {},
      );
      setAviso(`${data.publicadas} alerta(s) publicado(s) para o canal.`);
    } catch (err) {
      setAviso("");
      setErro(err instanceof ApiError ? err.message : "Não foi possível publicar o alerta.");
    }
  }

  return (
    <div className="flex flex-col gap-8">
      <section className="rounded-2xl bg-primary px-6 py-10 text-center text-primary-foreground">
        <h1 className="text-3xl font-bold tracking-tight">Protocolo da Fazenda</h1>
        <p className="mt-1 text-lg">Secretaria Municipal de Fazenda</p>
        <p className="mx-auto mt-3 max-w-2xl text-sm">
          Abra o Regulariza ou o Avança e acompanhe o processo pelo número. O aviso externo leva o
          número e a orientação.
        </p>
        <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
          <Link
            href="/fazenda/novo"
            className="inline-flex items-center justify-center rounded-lg bg-surface px-4 py-2 text-sm font-medium text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-surface"
          >
            Abrir protocolo
          </Link>
          <a
            href="#acompanhar"
            className="inline-flex items-center justify-center rounded-lg border border-primary-foreground px-4 py-2 text-sm font-medium text-primary-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-surface"
          >
            Consultar protocolo
          </a>
        </div>
        <ul className="mt-8 grid gap-3 text-left md:grid-cols-3">
          {DESTAQUES.map((item) => (
            <li
              key={item.titulo}
              className="rounded-xl border border-primary-foreground/20 bg-primary-foreground/10 p-4"
            >
              <item.icone aria-hidden="true" className="mb-2" size={20} />
              <h2 className="text-sm font-semibold">{item.titulo}</h2>
              <p className="mt-1 text-xs">{item.texto}</p>
            </li>
          ))}
        </ul>
      </section>

      <section aria-labelledby="servicos-fazenda" className="flex flex-col gap-3">
        <h2 id="servicos-fazenda" className="text-center text-2xl font-semibold text-primary">
          Serviços
        </h2>
        <p className="text-center text-sm text-muted">
          Regulariza e Avança Rondonópolis. Cada um abre o próprio formulário.
        </p>
        <GradeServicos itens={SERVICOS} />
      </section>

      <section id="acompanhar" aria-labelledby="acompanhar-titulo" className="flex flex-col gap-4">
        <div className="text-center">
          <h2 id="acompanhar-titulo" className="text-2xl font-semibold text-primary">
            Acompanhar protocolo
          </h2>
          <p className="mt-1 text-sm text-muted">
            Digite o número do protocolo para ver o andamento.
          </p>
        </div>
        <form
          className="mx-auto flex w-full max-w-2xl flex-col gap-3 sm:flex-row sm:items-end"
          onSubmit={(e) => {
            e.preventDefault();
            document.getElementById("resultado-protocolo")?.focus();
          }}
        >
          <div className="flex-1">
            <Input
              id="busca-protocolo"
              label="Número do protocolo"
              placeholder="000001/2026"
              value={busca}
              onChange={(e) => setBusca(e.target.value)}
              autoComplete="off"
            />
          </div>
          <div className="sm:w-56">
            <Select
              id="situacao-protocolo"
              label="Situação"
              placeholder="Todas"
              options={OPCOES_SITUACAO}
              value={situacao}
              onChange={(e) => setSituacao(e.target.value)}
            />
          </div>
          <Button type="submit">Consultar protocolo</Button>
        </form>
        <div className="flex flex-wrap items-center justify-center gap-3">
          <Link href="/fazenda/dua" className={buttonClass("secondary", "sm")}>
            Entregar DUA
          </Link>
          {equipe && (
            <Button type="button" variant="secondary" size="sm" onClick={() => void publicar()}>
              Publicar alertas
            </Button>
          )}
        </div>
        {aviso && <p className="text-center text-sm">{aviso}</p>}
        {erro && (
          <p role="alert" className="text-center text-sm text-danger">
            {erro}
          </p>
        )}
        {equipe && (list.data ?? []).length > 0 && (
          <div className="flex flex-wrap justify-center gap-2">
            {OPCOES_SITUACAO.map((opcao) => {
              const quantidade = (list.data ?? []).filter((item) => item.situacao === opcao.value).length;
              if (quantidade === 0) return null;
              return (
                <Button
                  key={opcao.value}
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => setSituacao(opcao.value)}
                >
                  {opcao.label}: {quantidade}
                </Button>
              );
            })}
          </div>
        )}
        {equipe && (prazos.data ?? []).length > 0 && (
          <ul className="flex flex-col gap-2 rounded-xl border border-surface-border bg-surface p-4 text-sm">
            {(prazos.data ?? []).map((prazo) => (
              <li key={`${prazo.nome}-${prazo.limite}`}>
                {prazo.nome} até {emCuiaba(prazo.limite)}: {prazo.situacao}
                {prazo.situacao === "alerta" && (
                  <span className="text-muted"> — há prazo a vencer, acompanhe na área restrita</span>
                )}
              </li>
            ))}
          </ul>
        )}
        <DataState
          loading={list.isLoading}
          error={list.error}
          onRetry={() => void list.mutate()}
          empty={(list.data ?? []).length === 0}
          emptyTitle="Nenhum processo"
        >
          <div id="resultado-protocolo" tabIndex={-1}>
            {processos.length === 0 ? (
              <p className="text-sm text-muted">
                {termo ? "Nenhum protocolo com esse número." : "Nenhum protocolo nessa situação."}
              </p>
            ) : (
              <ul className="flex flex-col divide-y divide-surface-border rounded-xl border border-surface-border bg-surface">
                {processos.map((item) => (
                  <li key={item.id} className="flex items-center justify-between gap-3 p-4">
                    <div>
                      <Link href={`/fazenda/${item.id}`} className="font-medium">
                        {item.numero}
                      </Link>
                      <p className="text-sm text-muted">{nomePrograma(item.programa)}</p>
                      {item.pendencia && <p className="text-sm">{item.pendencia}</p>}
                      {emCuiaba(item.exigencia_prazo) && (
                        <p className="text-xs text-muted">Prazo {emCuiaba(item.exigencia_prazo)}</p>
                      )}
                    </div>
                    <span className="flex items-center gap-2 text-xs text-muted">
                      <Badge tone={tomSituacao(item.situacao)}>{nomeSituacao(item.situacao)}</Badge>
                      {emCuiaba(item.recibo_em)}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </DataState>
      </section>
    </div>
  );
}
