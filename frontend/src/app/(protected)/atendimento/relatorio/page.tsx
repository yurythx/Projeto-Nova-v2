"use client";

import { Download, Printer } from "lucide-react";
import { useState } from "react";

import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDate } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Select } from "@/components/ui/Select";
import { useApiQuery, withQuery } from "@/lib/api/swr";
import type {
  LinhaContrato,
  LinhaEquipe,
  LinhaSecretaria,
  MedidaXLA,
  RelatorioSecretarias,
} from "@/lib/atendimento/atendimento";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

const PERIODOS = [
  { value: "30", label: "Últimos 30 dias" },
  { value: "90", label: "Últimos 90 dias" },
  { value: "180", label: "Últimos 180 dias" },
  { value: "366", label: "Último ano" },
];

function v(n: number | null | undefined, sufixo = "") {
  return n == null ? "—" : `${n.toLocaleString("pt-BR")}${sufixo}`;
}

/** O valor com a marca da meta: o leitor do relatório impresso também
 * precisa saber o que ficou abaixo. */
function comMeta(m: MedidaXLA, sufixo = "") {
  if (m.valor == null) return "—";
  const marca = m.atingida == null ? "" : m.atingida ? " ✓" : " ✗";
  return `${v(m.valor, sufixo)}${marca}`;
}

const COLUNAS: { rotulo: string; valor: (l: LinhaSecretaria) => string }[] = [
  { rotulo: "Pedidos", valor: (l) => v(l.total) },
  { rotulo: "Abertos", valor: (l) => v(l.abertas) },
  { rotulo: "Concluídos", valor: (l) => v(l.concluidas) },
  { rotulo: "Cancelados", valor: (l) => v(l.canceladas) },
  { rotulo: "Resolução", valor: (l) => v(l.mttr_horas, " h") },
  { rotulo: "No prazo", valor: (l) => v(l.sla_percentual, "%") },
  { rotulo: "1º contato", valor: (l) => v(l.fcr_percentual, "%") },
  { rotulo: "Nota", valor: (l) => comMeta(l.experiencia.csat) },
  { rotulo: "Facilidade", valor: (l) => comMeta(l.experiencia.facilidade) },
  { rotulo: "Reabertura", valor: (l) => comMeta(l.experiencia.reabertura, "%") },
  { rotulo: "Idas e voltas", valor: (l) => comMeta(l.experiencia.idas_e_voltas) },
  { rotulo: "Espera", valor: (l) => comMeta(l.experiencia.espera_horas, " h") },
];

const COLUNAS_EQUIPE: { rotulo: string; valor: (e: LinhaEquipe) => string }[] = [
  {
    rotulo: "Prazo da equipe",
    valor: (e) => (e.ola_horas == null ? "sem OLA" : `${e.ola_horas} h`),
  },
  { rotulo: "Chamados", valor: (e) => v(e.chamados) },
  { rotulo: "Chamados no prazo", valor: (e) => v(e.chamados_no_prazo_percentual, "%") },
  { rotulo: "Chamados: mediana", valor: (e) => v(e.chamados_mediana_horas, " h") },
  { rotulo: "Com a equipe", valor: (e) => v(e.com_a_equipe) },
  { rotulo: "Chamados atrasados", valor: (e) => v(e.chamados_atrasados) },
  { rotulo: "Tarefas feitas", valor: (e) => v(e.concluidas) },
  { rotulo: "Tarefas no prazo", valor: (e) => v(e.no_prazo_percentual, "%") },
  { rotulo: "Tarefas: mediana", valor: (e) => v(e.mediana_horas, " h") },
  { rotulo: "Tarefas pendentes", valor: (e) => v(e.pendentes) },
  { rotulo: "Tarefas atrasadas", valor: (e) => v(e.atrasadas) },
];

const COLUNAS_CONTRATO: { rotulo: string; valor: (c: LinhaContrato) => string }[] = [
  {
    rotulo: "Prazo do fornecedor",
    valor: (c) => (c.prazo_horas == null ? "sem prazo" : `${c.prazo_horas} h`),
  },
  { rotulo: "Chamados", valor: (c) => v(c.pedidos) },
  { rotulo: "Voltaram", valor: (c) => v(c.medidos) },
  { rotulo: "No prazo", valor: (c) => v(c.no_prazo_percentual, "%") },
  { rotulo: "Mediana", valor: (c) => v(c.mediana_horas, " h") },
  { rotulo: "Com o fornecedor", valor: (c) => v(c.com_fornecedor) },
  { rotulo: "Atrasados", valor: (c) => v(c.atrasados) },
];

function nomeDe(l: LinhaSecretaria) {
  return l.sigla ? `${l.nome} (${l.sigla})` : l.nome;
}

function baixar(rel: RelatorioSecretarias) {
  const celula = (t: string) => `"${t.replaceAll('"', '""')}"`;
  const linhas = [
    ["Secretaria", ...COLUNAS.map((c) => c.rotulo)],
    ...[...rel.secretarias, rel.total].map((l) => [nomeDe(l), ...COLUNAS.map((c) => c.valor(l))]),
    [],
    ["Equipe", ...COLUNAS_EQUIPE.map((c) => c.rotulo)],
    ...rel.equipes.map((e) => [e.nome, ...COLUNAS_EQUIPE.map((c) => c.valor(e))]),
    [],
    ["Contrato de apoio", ...COLUNAS_CONTRATO.map((c) => c.rotulo)],
    ...rel.contratos.map((k) => [k.rotulo, ...COLUNAS_CONTRATO.map((c) => c.valor(k))]),
  ];
  const conteudo = "﻿" + linhas.map((l) => l.map(celula).join(";")).join("\n");
  const url = URL.createObjectURL(new Blob([conteudo], { type: "text/csv;charset=utf-8;" }));
  const link = document.createElement("a");
  link.href = url;
  link.download = `relatorio-por-secretaria-${rel.dias}-dias.csv`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/** Relatório por secretaria (ADR 070): o que cada secretaria pediu, como a
 * central respondeu e como foi para quem pediu; e o OLA de cada equipe.
 * Serve para a reunião de resultados e para prestar contas. */
export default function AtendimentoRelatorioPage() {
  const { can } = usePlataforma();
  const [dias, setDias] = useState("90");
  const rel = useApiQuery<RelatorioSecretarias>(
    can("atendimento:atender") ? withQuery("v1/atendimento/relatorio/secretarias", { dias }) : null,
  );

  if (!can("atendimento:atender")) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Atendimento" title="Relatório por secretaria" />
        <EmptyState
          title="Seu perfil não atende"
          description="O relatório mostra o escopo de quem atende."
        />
      </div>
    );
  }

  const r = rel.data;
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Atendimento"
        title="Relatório por secretaria"
        description="Pedidos abertos no período, no seu escopo. O pedido de um departamento conta para a sua secretaria. ✓ e ✗ marcam as metas de experiência."
        actions={
          <div className="flex flex-wrap items-end gap-2 print:hidden">
            <Select
              id="relatorio-dias"
              label="Período"
              options={PERIODOS}
              value={dias}
              onChange={(e) => setDias(e.target.value)}
            />
            <Button type="button" variant="secondary" disabled={!r} onClick={() => r && baixar(r)}>
              <Download size={14} aria-hidden="true" className="mr-1" /> Baixar CSV
            </Button>
            <Button type="button" variant="ghost" onClick={() => window.print()}>
              <Printer size={14} aria-hidden="true" className="mr-1" /> Imprimir
            </Button>
          </div>
        }
      />
      <DataState
        loading={rel.isLoading}
        error={rel.error}
        onRetry={() => void rel.mutate()}
        empty={!r || r.total.total === 0}
        emptyTitle="Nenhum pedido no período"
        emptyDescription="Escolha um período maior."
      >
        {r && (
          <>
            <p className="text-sm text-muted">
              Desde {fmtDate(r.desde)}.
              {r.truncado ? " A amostra passou do teto: entram os pedidos mais recentes." : ""}
            </p>
            <Card>
              <CardHeader>
                <CardTitle as="h2">Por secretaria</CardTitle>
              </CardHeader>
              <CardContent className="overflow-x-auto pb-5">
                <table className="w-full text-left text-sm">
                  <thead>
                    <tr className="text-muted">
                      <th scope="col" className="py-1 pr-3 font-medium">
                        Secretaria
                      </th>
                      {COLUNAS.map((c) => (
                        <th key={c.rotulo} scope="col" className="py-1 pr-3 font-medium">
                          {c.rotulo}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {r.secretarias.map((l) => (
                      <tr
                        key={l.unidade_id ?? "institucional"}
                        className="border-t border-surface-border"
                      >
                        <th scope="row" className="py-1 pr-3 font-medium">
                          {nomeDe(l)}
                        </th>
                        {COLUNAS.map((c) => (
                          <td key={c.rotulo} className="py-1 pr-3">
                            {c.valor(l)}
                          </td>
                        ))}
                      </tr>
                    ))}
                    <tr className="border-t-2 border-surface-border font-semibold">
                      <th scope="row" className="py-1 pr-3">
                        Total
                      </th>
                      {COLUNAS.map((c) => (
                        <td key={c.rotulo} className="py-1 pr-3">
                          {c.valor(r.total)}
                        </td>
                      ))}
                    </tr>
                  </tbody>
                </table>
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle as="h2">Equipes (OLA)</CardTitle>
              </CardHeader>
              <CardContent className="overflow-x-auto pb-5">
                {r.equipes.length === 0 ? (
                  <p className="text-sm">Nenhuma equipe com chamado ou tarefa no período.</p>
                ) : (
                  <table className="w-full text-left text-sm">
                    <thead>
                      <tr className="text-muted">
                        <th scope="col" className="py-1 pr-3 font-medium">
                          Equipe
                        </th>
                        {COLUNAS_EQUIPE.map((c) => (
                          <th key={c.rotulo} scope="col" className="py-1 pr-3 font-medium">
                            {c.rotulo}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {r.equipes.map((e) => (
                        <tr key={e.equipe_id} className="border-t border-surface-border">
                          <th scope="row" className="py-1 pr-3 font-medium">
                            {e.nome}
                          </th>
                          {COLUNAS_EQUIPE.map((c) => (
                            <td key={c.rotulo} className="py-1 pr-3">
                              {c.valor(e)}
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </CardContent>
            </Card>
            {r.contratos.length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle as="h2">Contratos de apoio</CardTitle>
                </CardHeader>
                <CardContent className="overflow-x-auto pb-5">
                  <p className="mb-2 text-sm text-muted">
                    Tempo dos chamados em &quot;aguardando terceiro&quot;, contra o prazo do
                    fornecedor copiado quando o contrato foi apontado.
                  </p>
                  <table className="w-full text-left text-sm">
                    <thead>
                      <tr className="text-muted">
                        <th scope="col" className="py-1 pr-3 font-medium">
                          Contrato
                        </th>
                        {COLUNAS_CONTRATO.map((c) => (
                          <th key={c.rotulo} scope="col" className="py-1 pr-3 font-medium">
                            {c.rotulo}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody>
                      {r.contratos.map((k) => (
                        <tr key={k.contrato_id} className="border-t border-surface-border">
                          <th scope="row" className="py-1 pr-3 font-medium">
                            {k.rotulo}
                          </th>
                          {COLUNAS_CONTRATO.map((c) => (
                            <td key={c.rotulo} className="py-1 pr-3">
                              {c.valor(k)}
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </CardContent>
              </Card>
            )}
          </>
        )}
      </DataState>
    </div>
  );
}
