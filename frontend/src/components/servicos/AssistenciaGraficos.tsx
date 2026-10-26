"use client";

import { Badge } from "@/components/ui/Badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import type { PontoIndicador } from "@/lib/atendimento/atendimento";
import { ESTILOS_GRAFICO, type EstiloGrafico } from "@/lib/servicos/painelAssistencia";

export interface PontoGrafico {
  rotulo: string;
  valor: number;
}

const CORES = ["text-primary", "text-warning", "text-success", "text-info", "text-danger"];

function contar(itens: { estado: string }[], rotulos: Record<string, string>): PontoGrafico[] {
  const mapa = new Map<string, number>();
  for (const item of itens) mapa.set(item.estado, (mapa.get(item.estado) ?? 0) + 1);
  return [...mapa.entries()].map(([estado, valor]) => ({
    rotulo: rotulos[estado] ?? estado,
    valor,
  }));
}

function diasComValor(
  serie: PontoIndicador[] | undefined,
  campo: keyof PontoIndicador,
): PontoGrafico[] {
  return (serie ?? [])
    .filter((ponto) => typeof ponto[campo] === "number")
    .map((ponto) => ({ rotulo: ponto.dia.slice(0, 10), valor: ponto[campo] as number }));
}

function Figura({
  titulo,
  descricao,
  pontos,
  estilo,
  nota,
}: {
  titulo: string;
  descricao: string;
  pontos: PontoGrafico[];
  estilo: EstiloGrafico;
  nota?: string;
}) {
  const total = pontos.reduce((soma, ponto) => soma + ponto.valor, 0);
  const maximo = Math.max(1, ...pontos.map((ponto) => ponto.valor));
  return (
    <Card data-estilo={estilo}>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between gap-2">
          <CardTitle as="h3" className="text-sm font-semibold">
            {titulo}
          </CardTitle>
          <Badge tone="neutral">{pontos.length === 0 ? "sem amostra" : String(total)}</Badge>
        </div>
        <p className="text-xs text-muted">{descricao}</p>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 pt-2">
        {pontos.length === 0 ? (
          <p className="text-sm">Sem amostra neste recorte.</p>
        ) : (
          <Desenho pontos={pontos} estilo={estilo} total={total} maximo={maximo} />
        )}
        {nota ? <p className="text-xs text-muted">{nota}</p> : null}
        {pontos.length > 0 ? (
          <ul className="flex flex-col gap-1 text-xs">
            {pontos.map((ponto, indice) => (
              <li key={ponto.rotulo} className="flex items-center justify-between gap-2">
                <span className={`flex items-center gap-1.5 ${CORES[indice % CORES.length]}`}>
                  <span className="h-2.5 w-2.5 rounded-full bg-current" />
                  <span className="text-muted">{ponto.rotulo}</span>
                </span>
                <span className="font-semibold text-foreground">
                  {ponto.valor.toLocaleString("pt-BR")}
                </span>
              </li>
            ))}
          </ul>
        ) : null}
      </CardContent>
    </Card>
  );
}

function Desenho({
  pontos,
  estilo,
  total,
  maximo,
}: {
  pontos: PontoGrafico[];
  estilo: EstiloGrafico;
  total: number;
  maximo: number;
}) {
  if (estilo === "barras" || estilo === "horizontal") {
    const deitado = estilo === "horizontal";
    return (
      <div
        className={`flex gap-2 overflow-x-auto ${deitado ? "flex-col" : "h-36 items-end"}`}
        role="img"
        aria-label="Gráfico"
      >
        {pontos.map((ponto, indice) => {
          const fracao = `${Math.round((ponto.valor / maximo) * 100)}%`;
          return deitado ? (
            <div key={ponto.rotulo} className="grid grid-cols-[7rem_1fr] items-center gap-2">
              <span className="truncate text-xs text-muted">{ponto.rotulo}</span>
              <div className="h-3 rounded-full bg-surface-border">
                <div
                  className={`h-3 rounded-full bg-current ${CORES[indice % CORES.length]}`}
                  style={{ width: fracao }}
                />
              </div>
            </div>
          ) : (
            <div
              key={ponto.rotulo}
              className="flex h-full min-w-8 flex-1 flex-col justify-end gap-1"
            >
              <div
                className={`w-full rounded-t bg-current ${CORES[indice % CORES.length]}`}
                style={{ height: fracao }}
              />
            </div>
          );
        })}
      </div>
    );
  }

  if (estilo === "rosca") {
    const raio = 40;
    const perimetro = 2 * Math.PI * raio;
    let cursou = 0;
    return (
      <div className="flex items-center justify-center" role="img" aria-label="Gráfico">
        <svg width="120" height="120" viewBox="0 0 100 100" className="-rotate-90">
          <circle
            cx="50"
            cy="50"
            r={raio}
            fill="transparent"
            stroke="currentColor"
            strokeWidth="14"
            className="text-surface-border"
          />
          {total > 0
            ? pontos.map((ponto, indice) => {
                const trecho = (ponto.valor / total) * perimetro;
                const inicio = cursou;
                cursou += trecho;
                return (
                  <circle
                    key={ponto.rotulo}
                    cx="50"
                    cy="50"
                    r={raio}
                    fill="transparent"
                    stroke="currentColor"
                    strokeWidth="14"
                    strokeDasharray={`${trecho} ${perimetro - trecho}`}
                    strokeDashoffset={-inicio}
                    className={CORES[indice % CORES.length]}
                  />
                );
              })
            : null}
        </svg>
      </div>
    );
  }

  const coordenadas = pontos.map((ponto, indice) => {
    const x = pontos.length === 1 ? 50 : (indice / (pontos.length - 1)) * 100;
    const y = 38 - (ponto.valor / maximo) * 34;
    return `${x},${y}`;
  });
  const linha = coordenadas.join(" ");
  return (
    <svg viewBox="0 0 100 40" className="h-36 w-full text-primary" role="img" aria-label="Gráfico">
      {estilo === "area" ? (
        <polygon points={`0,40 ${linha} 100,40`} className="fill-current opacity-20" />
      ) : null}
      <polyline
        points={linha}
        fill="none"
        stroke="currentColor"
        strokeWidth="1.5"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}

export function AssistenciaGraficos({
  estilo,
  onEstilo,
  incidentes,
  requisicoes,
  solicitacoes,
  problemas,
  mudancas,
  projetos,
  serie,
  notaFila,
}: {
  estilo: EstiloGrafico;
  onEstilo: (estilo: EstiloGrafico) => void;
  incidentes: number;
  requisicoes: number;
  solicitacoes: { estado: string }[];
  problemas: { estado: string }[];
  mudancas: { estado: string; risco?: string }[];
  projetos: { estado: string }[];
  serie?: PontoIndicador[];
  notaFila?: string;
}) {
  const riscos = ["baixo", "medio", "alto"].map((risco) => ({
    rotulo: risco === "baixo" ? "Baixo" : risco === "medio" ? "Médio" : "Alto",
    valor: mudancas.filter((item) => item.risco === risco).length,
  }));
  const semRisco = mudancas.filter(
    (item) => !["baixo", "medio", "alto"].includes(item.risco ?? ""),
  ).length;
  if (semRisco > 0) riscos.push({ rotulo: "Sem faixa", valor: semRisco });

  const figuras: { titulo: string; descricao: string; pontos: PontoGrafico[]; nota?: string }[] = [
    {
      titulo: "Demandas de Atendimento",
      descricao: "Incidentes e requisições do recorte da fila.",
      pontos: [
        { rotulo: "Incidentes", valor: incidentes },
        { rotulo: "Requisições", valor: requisicoes },
      ],
      nota: notaFila,
    },
    {
      titulo: "Estados da fila",
      descricao: "Como os chamados deste recorte estão agora.",
      pontos: contar(solicitacoes, {
        aberto: "Aberto",
        em_atendimento: "Em atendimento",
        aguardando_servidor: "Aguardando quem pediu",
        aguardando_terceiro: "Aguardando terceiro",
        resolvido: "Resolvido",
        atendido: "Atendido",
        encerrado: "Encerrado",
        cancelado: "Cancelado",
      }),
      nota: notaFila,
    },
    {
      titulo: "Problemas",
      descricao: "Estados dos problemas do recorte.",
      pontos: contar(problemas, {
        registrado: "Registrado",
        em_analise: "Em análise",
        erro_conhecido: "Erro conhecido",
        resolvido: "Resolvido",
        encerrado: "Encerrado",
      }),
    },
    {
      titulo: "Matriz de Risco das Mudanças",
      descricao: "Faixa de risco das mudanças do recorte.",
      pontos: riscos,
    },
    {
      titulo: "Progresso Estratégico",
      descricao: "Estados dos projetos do recorte.",
      pontos: contar(projetos, {
        planejado: "Planejado",
        em_andamento: "Em andamento",
        suspenso: "Suspenso",
        concluido: "Concluído",
        cancelado: "Cancelado",
      }),
    },
    {
      titulo: "Resolução no período",
      descricao: "Horas úteis médias de resolução, nos dias em que houve amostra.",
      pontos: diasComValor(serie, "mttr_horas"),
      nota: "Dias sem amostra ficam de fora.",
    },
    {
      titulo: "Prazo no período",
      descricao: "Parcela do prazo cumprido, nos dias em que houve amostra.",
      pontos: diasComValor(serie, "sla_percentual"),
      nota: "Dias sem amostra ficam de fora.",
    },
    {
      titulo: "Satisfação no período",
      descricao: "Média das notas, nos dias em que houve avaliação.",
      pontos: diasComValor(serie, "csat_media"),
      nota: "Dias sem amostra ficam de fora.",
    },
  ];

  return (
    <div className="flex flex-col gap-4" data-testid="assistencia-graficos">
      <div role="radiogroup" aria-label="Estilo dos gráficos" className="flex flex-wrap gap-2">
        {ESTILOS_GRAFICO.map((item) => (
          <button
            key={item.id}
            type="button"
            role="radio"
            aria-checked={estilo === item.id}
            onClick={() => onEstilo(item.id)}
            className={`rounded-md border px-3 py-1.5 text-sm font-medium ${
              estilo === item.id
                ? "border-primary bg-primary/10 text-primary"
                : "border-surface-border text-muted hover:text-foreground"
            }`}
          >
            {item.label}
          </button>
        ))}
      </div>
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {figuras.map((figura) => (
          <Figura key={figura.titulo} {...figura} estilo={estilo} />
        ))}
      </div>
    </div>
  );
}
