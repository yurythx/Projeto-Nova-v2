"use client";

import Link from "next/link";
import { useState, type ReactNode } from "react";
import { AlertTriangle, ArrowRight, FolderKanban, GitCommit, Headphones } from "lucide-react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Select } from "@/components/ui/Select";
import type { Governanca, PontoIndicador, Solicitacao } from "@/lib/atendimento/atendimento";
import { gravarArmazenado, useArmazenado } from "@/lib/prefs/armazenado";
import type { Projeto } from "@/lib/projetos/projetos";
import { dispor, mostrar, mover, ocultar, type Disposicao } from "@/lib/servicos/painelAssistencia";
import type { Mudanca, Problema } from "@/lib/servicos/servicos";

const MEMORIA = "nova-assistencia:painel";

export interface FiltrosAssistencia {
  tipo: string;
  situacao: string;
  estadoChamado: string;
  estadoProblema: string;
  estadoMudanca: string;
  estadoProjeto: string;
  dias: string;
}

export const filtrosIniciais: FiltrosAssistencia = {
  tipo: "",
  situacao: "",
  estadoChamado: "",
  estadoProblema: "",
  estadoMudanca: "",
  estadoProjeto: "",
  dias: "90",
};

const opcoes = (itens: [string, string][]) => itens.map(([value, label]) => ({ value, label }));

export function BarraDeFiltros({
  filtros,
  onChange,
}: {
  filtros: FiltrosAssistencia;
  onChange: (filtros: FiltrosAssistencia) => void;
}) {
  function campo(nome: keyof FiltrosAssistencia, valor: string) {
    onChange({ ...filtros, [nome]: valor });
  }
  const [aberto, setAberto] = useState(false);
  const sujo = Object.entries(filtros).some(([nome, valor]) =>
    nome === "dias" ? valor !== "90" : valor !== "",
  );
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          variant="secondary"
          aria-expanded={aberto}
          aria-controls="filtros-assistencia"
          onClick={() => setAberto((atual) => !atual)}
        >
          {aberto ? "Recolher filtros" : sujo ? "Filtros ativos" : "Filtros"}
        </Button>
        {sujo && !aberto ? (
          <Button type="button" variant="secondary" onClick={() => onChange(filtrosIniciais)}>
            Limpar filtros
          </Button>
        ) : null}
      </div>
      {aberto ? (
        <div id="filtros-assistencia" className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <Select
            id="filtro-tipo"
            label="Tipo do chamado"
            placeholder="Todos"
            value={filtros.tipo}
            onChange={(ev) => campo("tipo", ev.target.value)}
            options={opcoes([
              ["incidente", "Incidente"],
              ["requisicao", "Requisição"],
            ])}
          />
          <Select
            id="filtro-situacao"
            label="Situação da fila"
            placeholder="Em curso"
            value={filtros.situacao}
            onChange={(ev) => campo("situacao", ev.target.value)}
            options={opcoes([
              ["abertas", "Só abertas"],
              ["todas", "Incluir encerradas"],
            ])}
          />
          <Select
            id="filtro-estado-chamado"
            label="Estado do chamado"
            placeholder="Qualquer"
            value={filtros.estadoChamado}
            onChange={(ev) => campo("estadoChamado", ev.target.value)}
            options={opcoes([
              ["aberto", "Aberto"],
              ["em_atendimento", "Em atendimento"],
              ["aguardando_servidor", "Aguardando quem pediu"],
              ["aguardando_terceiro", "Aguardando terceiro"],
              ["resolvido", "Resolvido"],
              ["atendido", "Atendido"],
              ["encerrado", "Encerrado"],
              ["cancelado", "Cancelado"],
            ])}
          />
          <Select
            id="filtro-dias"
            label="Histórico"
            value={filtros.dias}
            onChange={(ev) => campo("dias", ev.target.value)}
            options={opcoes([
              ["30", "30 dias"],
              ["90", "90 dias"],
              ["180", "180 dias"],
              ["366", "366 dias"],
            ])}
          />
          <Select
            id="filtro-problema"
            label="Estado do problema"
            placeholder="Qualquer"
            value={filtros.estadoProblema}
            onChange={(ev) => campo("estadoProblema", ev.target.value)}
            options={opcoes([
              ["registrado", "Registrado"],
              ["em_analise", "Em análise"],
              ["erro_conhecido", "Erro conhecido"],
              ["resolvido", "Resolvido"],
              ["encerrado", "Encerrado"],
            ])}
          />
          <Select
            id="filtro-mudanca"
            label="Estado da mudança"
            placeholder="Qualquer"
            value={filtros.estadoMudanca}
            onChange={(ev) => campo("estadoMudanca", ev.target.value)}
            options={opcoes([
              ["rascunho", "Rascunho"],
              ["em_avaliacao", "Em avaliação"],
              ["aprovada", "Aprovada"],
              ["rejeitada", "Rejeitada"],
              ["em_execucao", "Em execução"],
              ["concluida", "Concluída"],
              ["falhou", "Falhou"],
              ["cancelada", "Cancelada"],
            ])}
          />
          <Select
            id="filtro-projeto"
            label="Estado do projeto"
            placeholder="Qualquer"
            value={filtros.estadoProjeto}
            onChange={(ev) => campo("estadoProjeto", ev.target.value)}
            options={opcoes([
              ["planejado", "Planejado"],
              ["em_andamento", "Em andamento"],
              ["suspenso", "Suspenso"],
              ["concluido", "Concluído"],
              ["cancelado", "Cancelado"],
            ])}
          />
          {sujo ? (
            <div className="flex items-end">
              <Button type="button" variant="secondary" onClick={() => onChange(filtrosIniciais)}>
                Limpar filtros
              </Button>
            </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function lerDisposicao(cru: string | null): Disposicao | null {
  if (!cru) return null;
  try {
    const dado = JSON.parse(cru) as Partial<Disposicao>;
    if (!Array.isArray(dado.ordem) || !Array.isArray(dado.ocultos)) return null;
    return {
      ordem: dado.ordem.filter((id) => typeof id === "string"),
      ocultos: dado.ocultos.filter((id) => typeof id === "string"),
    };
  } catch {
    return null;
  }
}

function textoMedida(valor: number | null | undefined, sufixo: string) {
  if (valor == null) return "sem amostra";
  return `${valor.toLocaleString("pt-BR")}${sufixo}`;
}

function Kpi({
  rotulo,
  valor,
  detalhe,
  icone,
}: {
  rotulo: string;
  valor: string;
  detalhe: string;
  icone: ReactNode;
}) {
  return (
    <div className="flex items-center justify-between rounded-xl border border-surface-border bg-surface p-4 shadow-sm">
      <div>
        <p className="text-xs text-muted">{rotulo}</p>
        <p className="mt-0.5 text-2xl font-bold text-foreground">{valor}</p>
        <span className="text-[11px] text-muted">{detalhe}</span>
      </div>
      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
        {icone}
      </div>
    </div>
  );
}

export function PainelDaAssistencia({
  editar,
  solicitacoes,
  totalFila,
  veFila,
  governanca,
  serie,
  serieErro,
  problemas,
  mudancas,
  projetos,
  modulos,
}: {
  editar: boolean;
  solicitacoes: Solicitacao[];
  totalFila: number;
  veFila: boolean;
  governanca?: Governanca;
  serie?: PontoIndicador[];
  serieErro?: unknown;
  problemas: Problema[];
  mudancas: Mudanca[];
  projetos: Projeto[];
  modulos: { atendimento: boolean; problemas: boolean; mudancas: boolean; projetos: boolean };
}) {
  const ids = [
    "fila",
    "erros",
    "janelas",
    "projetos-ativos",
    "indicadores",
    "satisfacao",
    "deflexao",
    "historico",
    "atendimento",
    "problemas",
    "mudancas",
    "projetos",
  ].filter((id) => {
    if (["fila", "indicadores", "satisfacao", "deflexao", "historico", "atendimento"].includes(id))
      return modulos.atendimento;
    if (id === "erros" || id === "problemas") return modulos.problemas;
    if (id === "janelas" || id === "mudancas") return modulos.mudancas;
    return modulos.projetos;
  });
  // Na hidratação vem null e os cartões saem na ordem padrão; o gravado
  // entra logo depois.
  const salva = lerDisposicao(useArmazenado(MEMORIA));
  const disposicao = dispor(ids, salva);

  function aplicar(proxima: Disposicao) {
    gravarArmazenado(MEMORIA, JSON.stringify(dispor(ids, proxima)));
  }

  const abertas = solicitacoes.filter((s) => s.estado === "aberto").length;
  const emAtendimento = solicitacoes.filter((s) => s.estado === "em_atendimento").length;
  const aguardando = solicitacoes.filter(
    (s) => s.estado === "aguardando_servidor" || s.estado === "aguardando_terceiro",
  ).length;
  const concluidas = solicitacoes.filter(
    (s) => s.estado === "resolvido" || s.estado === "atendido",
  ).length;
  const incidentes = solicitacoes.filter((s) => s.tipo === "incidente").length;
  const requisicoes = solicitacoes.filter((s) => s.tipo === "requisicao").length;
  const errosConhecidos = problemas.filter((p) => p.estado === "erro_conhecido").length;
  const probEmAnalise = problemas.filter((p) => p.estado === "em_analise").length;
  const probResolvidos = problemas.filter(
    (p) => p.estado === "resolvido" || p.estado === "encerrado",
  ).length;
  const mudAvaliacao = mudancas.filter((m) => m.estado === "em_avaliacao").length;
  const mudAprovadas = mudancas.filter((m) => m.estado === "aprovada").length;
  const mudExecucao = mudancas.filter((m) => m.estado === "em_execucao").length;
  const mudConcluidas = mudancas.filter((m) => m.estado === "concluida").length;
  const mudFalhas = mudancas.filter((m) => m.estado === "falhou").length;
  const projPlanejados = projetos.filter((p) => p.estado === "planejado").length;
  const projEmAndamento = projetos.filter((p) => p.estado === "em_andamento").length;
  const projSuspensos = projetos.filter((p) => p.estado === "suspenso").length;
  const projConcluidos = projetos.filter((p) => p.estado === "concluido").length;
  const totalOrcamento = projetos.reduce((acc, p) => acc + (Number(p.orcamento) || 0), 0);
  const notaFila = !veFila
    ? "Fora do seu alcance"
    : totalFila > solicitacoes.length
      ? `${abertas} abertas • ${emAtendimento} em curso · ${solicitacoes.length} de ${totalFila} nesta página`
      : `${abertas} abertas • ${emAtendimento} em curso`;

  const titulos: Record<string, string> = {
    fila: "Fila de atendimento",
    erros: "Erros conhecidos",
    janelas: "Janelas de mudança",
    "projetos-ativos": "Projetos ativos",
    indicadores: "Indicadores",
    satisfacao: "Satisfação do atendimento",
    deflexao: "Artigos antes do chamado",
    historico: "Histórico dos indicadores",
    atendimento: "Atendimento",
    problemas: "Problemas",
    mudancas: "Mudanças",
    projetos: "Projetos",
  };
  const largos = new Set([
    "indicadores",
    "satisfacao",
    "deflexao",
    "historico",
    "atendimento",
    "problemas",
    "mudancas",
    "projetos",
  ]);
  const atual = dispor(ids, disposicao);
  const visiveis = atual.ordem.filter((id) => !atual.ocultos.includes(id));

  function corpo(id: string): ReactNode | null {
    if (id === "fila") {
      return (
        <Kpi
          rotulo="Fila de Atendimento"
          valor={veFila ? String(solicitacoes.length) : "—"}
          detalhe={notaFila}
          icone={<Headphones className="h-5 w-5" />}
        />
      );
    }
    if (id === "erros") {
      return (
        <Kpi
          rotulo="Erros Conhecidos"
          valor={String(errosConhecidos)}
          detalhe={`${probEmAnalise} problemas em análise`}
          icone={<AlertTriangle className="h-5 w-5" />}
        />
      );
    }
    if (id === "janelas") {
      return (
        <Kpi
          rotulo="Janelas de Mudança"
          valor={String(mudAprovadas + mudExecucao)}
          detalhe={`${mudAvaliacao} sob aprovação do CAB`}
          icone={<GitCommit className="h-5 w-5" />}
        />
      );
    }
    if (id === "projetos-ativos") {
      return (
        <Kpi
          rotulo="Projetos Ativos"
          valor={String(projEmAndamento + projPlanejados)}
          detalhe={`${projConcluidos} projetos concluídos`}
          icone={<FolderKanban className="h-5 w-5" />}
        />
      );
    }
    if (id === "indicadores") {
      if (!veFila && !editar) return null;
      return (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Indicadores</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-3 pb-5 sm:grid-cols-2 lg:grid-cols-4">
            {!veFila ? (
              <p className="text-sm">Fora do seu alcance.</p>
            ) : (
              [
                ["Resolução", textoMedida(governanca?.mttr_horas, " h")],
                ["Entre falhas", textoMedida(governanca?.mtbf_horas, " h")],
                ["Prazo", textoMedida(governanca?.sla_percentual, "%")],
                ["Primeiro contato", textoMedida(governanca?.fcr_percentual, "%")],
              ].map(([rotulo, valor]) => (
                <p key={rotulo} className="text-sm">
                  <span className="block text-xs text-muted">{rotulo}</span>
                  <span className="text-lg font-semibold">{valor}</span>
                </p>
              ))
            )}
          </CardContent>
        </Card>
      );
    }
    if (id === "satisfacao") {
      if (!governanca?.csat)
        return editar ? <AmostraVazia titulo="Satisfação do atendimento" /> : null;
      return (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Satisfação do atendimento</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 pb-5">
            <p className="text-sm">
              Média <strong>{governanca.csat.media.toLocaleString("pt-BR")}</strong> de 5, em{" "}
              {governanca.csat.total} {governanca.csat.total === 1 ? "avaliação" : "avaliações"}.
            </p>
            <ul className="flex flex-col gap-2">
              {governanca.csat.recentes.map((a) => (
                <li key={a.id} className="text-sm">
                  <span className="font-medium">{a.nota}/5</span>
                  <span className="text-muted"> · solicitação #{a.numero}</span>
                  {a.comentario ? <span className="text-muted"> — {a.comentario}</span> : null}
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      );
    }
    if (id === "deflexao") {
      if (!governanca?.deflexao)
        return editar ? <AmostraVazia titulo="Artigos antes do chamado" /> : null;
      const deflexao = governanca.deflexao;
      return (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Artigos antes do chamado</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <p className="text-sm">
              {deflexao.sem_pedido} de {deflexao.viram}{" "}
              {deflexao.viram === 1 ? "pessoa viu" : "pessoas viram"} um artigo e não abriram pedido
              ({deflexao.taxa.toLocaleString("pt-BR")}%).
            </p>
          </CardContent>
        </Card>
      );
    }
    if (id === "historico") {
      if (!veFila && !editar) return null;
      return <Historico serie={serie} erro={serieErro} />;
    }
    if (id === "atendimento")
      return cartaoFila({
        abertas,
        emAtendimento,
        aguardando,
        incidentes,
        requisicoes,
        concluidas,
        total: veFila ? totalFila : 0,
        veFila,
      });
    if (id === "problemas")
      return cartaoProblemas(problemas.length, probEmAnalise, errosConhecidos, probResolvidos);
    if (id === "mudancas")
      return cartaoMudancas(
        mudancas.length,
        mudAvaliacao,
        mudExecucao,
        mudAprovadas,
        mudConcluidas,
        mudFalhas,
      );
    if (id === "projetos")
      return cartaoProjetos(
        projetos.length,
        projEmAndamento,
        projPlanejados,
        projConcluidos,
        projSuspensos,
        totalOrcamento,
      );
    return null;
  }

  return (
    <div className="flex flex-col gap-4">
      {atual.ocultos.length > 0 ? (
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm text-muted">Cartões ocultos</span>
          {atual.ocultos.map((id) => (
            <Button
              key={id}
              type="button"
              variant="secondary"
              onClick={() => aplicar(mostrar(atual, id))}
            >
              Mostrar {titulos[id]}
            </Button>
          ))}
        </div>
      ) : null}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {visiveis.map((id) => {
          const miolo = corpo(id);
          if (!miolo && !editar) return null;
          const indice = visiveis.indexOf(id);
          return (
            <div key={id} className={largos.has(id) ? "sm:col-span-2 lg:col-span-4" : ""}>
              {editar ? (
                <div className="mb-2 flex flex-wrap gap-2">
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={indice === 0}
                    onClick={() => aplicar(mover(atual, id, -1))}
                  >
                    Subir {titulos[id]}
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={indice === visiveis.length - 1}
                    onClick={() => aplicar(mover(atual, id, 1))}
                  >
                    Descer {titulos[id]}
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() => aplicar(ocultar(atual, id))}
                  >
                    Ocultar {titulos[id]}
                  </Button>
                </div>
              ) : null}
              {miolo ?? <AmostraVazia titulo={titulos[id] ?? id} />}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function AmostraVazia({ titulo }: { titulo: string }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">{titulo}</CardTitle>
      </CardHeader>
      <CardContent className="pb-5">
        <p className="text-sm">Aparece quando houver amostra.</p>
      </CardContent>
    </Card>
  );
}

function Historico({ serie, erro }: { serie?: PontoIndicador[]; erro?: unknown }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Histórico dos indicadores</CardTitle>
      </CardHeader>
      <CardContent className="pb-5">
        <p className="mb-3 text-sm text-muted">
          O dia segue o fuso da unidade. O dia de hoje é atualizado. Os anteriores ficam como foram
          medidos.
        </p>
        {erro ? (
          <p className="text-sm text-danger" role="alert">
            Não foi possível carregar o histórico.
          </p>
        ) : null}
        {(serie ?? []).length === 0 && !erro ? (
          <p className="text-sm">Ainda não há dia medido neste escopo.</p>
        ) : null}
        {(serie ?? []).length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="text-muted">
                  <th className="py-1 pr-3 font-medium">Dia</th>
                  <th className="py-1 pr-3 font-medium">Resolução</th>
                  <th className="py-1 pr-3 font-medium">Entre falhas</th>
                  <th className="py-1 pr-3 font-medium">Prazo</th>
                  <th className="py-1 pr-3 font-medium">Primeiro contato</th>
                  <th className="py-1 font-medium">Satisfação</th>
                </tr>
              </thead>
              <tbody>
                {serie!.map((p) => (
                  <tr key={p.dia} className="border-t border-surface-border">
                    <td className="py-1 pr-3">
                      {p.dia.slice(0, 10)}
                      {p.truncado ? " · amostra parcial" : ""}
                    </td>
                    <td className="py-1 pr-3">{textoMedida(p.mttr_horas, " h")}</td>
                    <td className="py-1 pr-3">{textoMedida(p.mtbf_horas, " h")}</td>
                    <td className="py-1 pr-3">{textoMedida(p.sla_percentual, "%")}</td>
                    <td className="py-1 pr-3">{textoMedida(p.fcr_percentual, "%")}</td>
                    <td className="py-1">{textoMedida(p.csat_media, "")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : null}
      </CardContent>
    </Card>
  );
}

function cartaoFila(n: {
  abertas: number;
  emAtendimento: number;
  aguardando: number;
  incidentes: number;
  requisicoes: number;
  concluidas: number;
  total: number;
  veFila: boolean;
}) {
  return (
    <Card className="flex h-full flex-col justify-between">
      <div>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <div className="flex items-center gap-2">
            <Headphones className="h-5 w-5 text-primary" />
            <CardTitle as="h2">Atendimento</CardTitle>
          </div>
          <Badge tone={n.abertas > 0 ? "warning" : "success"}>
            {n.veFila ? `${n.total} na fila` : "fora do alcance"}
          </Badge>
        </CardHeader>
        <CardContent className="flex flex-col gap-4 pt-2">
          {n.veFila ? (
            <>
              <div className="grid grid-cols-2 gap-2 text-center">
                <div className="rounded-lg border border-surface-border p-2">
                  <span className="block text-xl font-bold">{n.abertas}</span>
                  <span className="text-xs text-muted">Abertas</span>
                </div>
                <div className="rounded-lg border border-surface-border p-2">
                  <span className="block text-xl font-bold text-primary">{n.emAtendimento}</span>
                  <span className="text-xs text-muted">Em atendimento</span>
                </div>
              </div>
              <div className="flex flex-col gap-1.5 text-xs text-muted">
                <div className="flex justify-between">
                  <span>Aguardando resposta</span>
                  <span className="font-semibold text-foreground">{n.aguardando}</span>
                </div>
                <div className="flex justify-between">
                  <span>Incidentes vs Requisições</span>
                  <span className="font-semibold text-foreground">
                    {n.incidentes} / {n.requisicoes}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span>Resolvidas ou atendidas neste recorte</span>
                  <span className="font-semibold text-success">{n.concluidas}</span>
                </div>
              </div>
            </>
          ) : (
            <p className="text-sm">Fora do seu alcance.</p>
          )}
        </CardContent>
      </div>
      <div className="border-t border-surface-border px-5 py-3">
        <Link
          href="/atendimento/fila"
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
        >
          Abrir fila e quadro de atendimento
          <ArrowRight className="h-3.5 w-3.5" />
        </Link>
      </div>
    </Card>
  );
}

function cartaoProblemas(total: number, analise: number, erros: number, resolvidos: number) {
  return (
    <Card className="flex h-full flex-col justify-between">
      <div>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle as="h2">Problemas</CardTitle>
          <Badge tone="neutral">{total} itens</Badge>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 pt-2 text-sm">
          <p>{analise} em investigação</p>
          <p>{erros} erros conhecidos</p>
          <p>{resolvidos} resolvidos ou encerrados</p>
        </CardContent>
      </div>
      <div className="border-t border-surface-border px-5 py-3">
        <Link
          href="/problemas"
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
        >
          Gerenciar problemas e contornos
          <ArrowRight className="h-3.5 w-3.5" />
        </Link>
      </div>
    </Card>
  );
}

function cartaoMudancas(
  total: number,
  avaliacao: number,
  execucao: number,
  aprovadas: number,
  concluidas: number,
  falhas: number,
) {
  return (
    <Card className="flex h-full flex-col justify-between">
      <div>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle as="h2">Mudanças</CardTitle>
          <Badge tone="neutral">{total} registradas</Badge>
        </CardHeader>
        <CardContent className="flex flex-col gap-2 pt-2 text-sm">
          <p>{avaliacao} em avaliação</p>
          <p>{execucao} em execução</p>
          <p>{aprovadas} aprovadas para janela</p>
          <p>{concluidas} concluídas</p>
          {falhas > 0 ? <p className="text-danger">{falhas} com falha</p> : null}
        </CardContent>
      </div>
      <div className="border-t border-surface-border px-5 py-3">
        <Link
          href="/mudancas"
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
        >
          Planejar e acompanhar mudanças
          <ArrowRight className="h-3.5 w-3.5" />
        </Link>
      </div>
    </Card>
  );
}

function cartaoProjetos(
  total: number,
  andamento: number,
  planejados: number,
  concluidos: number,
  suspensos: number,
  orcamento: number,
) {
  return (
    <Card className="flex h-full flex-col justify-between">
      <div>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle as="h2">Projetos</CardTitle>
          <Badge tone={andamento > 0 ? "warning" : "neutral"}>{total} cadastrados</Badge>
        </CardHeader>
        <CardContent className="flex flex-col gap-2 pt-2 text-sm">
          <p>{andamento} em andamento</p>
          <p>{planejados} planejados</p>
          <p>{concluidos} concluídos</p>
          {suspensos > 0 ? <p>{suspensos} suspensos</p> : null}
          {orcamento > 0 ? (
            <p>
              Orçamento alocado R$ {orcamento.toLocaleString("pt-BR", { minimumFractionDigits: 2 })}
            </p>
          ) : null}
        </CardContent>
      </div>
      <div className="border-t border-surface-border px-5 py-3">
        <Link
          href="/projetos"
          className="inline-flex items-center gap-1.5 text-xs font-semibold text-primary hover:underline"
        >
          Gerenciar projetos e tarefas
          <ArrowRight className="h-3.5 w-3.5" />
        </Link>
      </div>
    </Card>
  );
}
