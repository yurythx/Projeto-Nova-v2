"use client";

import { AlertTriangle } from "lucide-react";
import { useRouter } from "next/navigation";
import type { ReactNode } from "react";

import { PrazoBadge } from "@/components/atendimento/PrazoBadge";
import {
  NomeClicavel,
  Quadro,
  colunasEmOrdem,
  type ColunaFaixa,
  type ControlesColuna,
} from "@/components/plataforma/Faixas";
import { Badge } from "@/components/ui/Badge";
import {
  relogioEmFoco,
  rotuloEstado,
  rotuloTipo,
  type Estado,
  type Solicitacao,
} from "@/lib/atendimento/atendimento";

type Coluna =
  | "aguardando_aprovacao"
  | "aberto"
  | "em_atendimento"
  | "aguardando_servidor"
  | "aguardando_terceiro"
  | "concluida";

/** A mesma numeração no quadro e na lista, a partir de zero. O quadro fica
 * no fluxo em andamento; a lista segue com encerradas (6) e canceladas (7). */
export const FAIXAS_FILA: ColunaFaixa[] = colunasEmOrdem([
  { id: "aguardando_aprovacao", titulo: "Aguardando aprovação", tom: "warning" },
  { id: "aberto", titulo: "Abertas", tom: "info" },
  { id: "em_atendimento", titulo: "Em atendimento", tom: "info" },
  { id: "aguardando_servidor", titulo: "Aguardando o servidor", tom: "warning" },
  { id: "aguardando_terceiro", titulo: "Aguardando terceiro", tom: "warning" },
  { id: "concluida", titulo: "Concluídas", tom: "success" },
  { id: "encerrado", titulo: "Encerradas", tom: "neutral" },
  { id: "cancelado", titulo: "Canceladas", tom: "danger" },
]);

const COLUNAS_QUADRO = FAIXAS_FILA.filter(
  (coluna) => coluna.id !== "encerrado" && coluna.id !== "cancelado",
);

/** A coluna de conclusão junta os dois fins: resolvido (incidente) e
 * atendido (requisição). */
export function faixaDaFila(estado: Estado): string {
  if (estado === "resolvido" || estado === "atendido") return "concluida";
  return estado;
}

function destino(s: Solicitacao, coluna: string): Estado {
  if (coluna !== "concluida") return coluna as Estado;
  return s.tipo === "incidente" ? "resolvido" : "atendido";
}

/** A coluna aceita o cartão só se a máquina deixa (as passagens vêm da API). */
function aceita(colunas: ColunaFaixa[], s: Solicitacao, coluna: string): boolean {
  const alvo = colunas.find((c) => c.id === coluna);
  if (!alvo || alvo.ativa === false) return false;
  return faixaDaFila(s.estado) !== coluna && (s.proximos ?? []).includes(destino(s, coluna));
}

function moldura(s: Solicitacao): string | undefined {
  const foco = s.prazo ? relogioEmFoco(s.prazo) : null;
  const situacao = foco?.relogio.situacao;
  if (situacao === "vencido" || situacao === "descumprido") {
    return "border-danger/60 border-l-danger bg-danger/[0.02]";
  }
  if (situacao === "critico" || situacao === "atencao") {
    return "border-warning/60 border-l-warning bg-warning/[0.02]";
  }
  return undefined;
}

/** A mesma fila da lista, por estado. Mover o cartão chama a mesma
 * passagem da tela de detalhe; se o domínio recusar, o cartão fica onde
 * estava. Arrastar é atalho de mouse: cada cartão tem o seletor "Mover
 * para", que serve ao teclado, ao toque e ao leitor de tela. */
export function Kanban({
  itens,
  onMover,
  pending,
  extra,
  aoAbrir,
  colunas = COLUNAS_QUADRO,
  controles,
}: {
  itens: Solicitacao[];
  onMover: (s: Solicitacao, para: Estado) => void;
  pending: boolean;
  extra?: (colunaId: string) => ReactNode;
  aoAbrir?: (s: Solicitacao) => void;
  colunas?: ColunaFaixa[];
  controles?: ControlesColuna;
}) {
  const router = useRouter();
  function abrir(s: Solicitacao) {
    if (aoAbrir) aoAbrir(s);
    else router.push(`/atendimento/solicitacoes/${s.id}`);
  }
  return (
    <Quadro
      rotulo="Quadro da fila"
      memoria="nova-faixas:fila-quadro"
      aparencia="nova-faixas:fila"
      colunas={colunas}
      itens={itens}
      colunaDe={(s) => {
        const faixa = faixaDaFila(s.estado);
        return colunas.some((c) => c.id === faixa) ? faixa : null;
      }}
      idDe={(s) => s.id}
      nomeDe={(s) => s.titulo}
      pending={pending}
      extra={extra}
      arrastavel={(s) => (s.proximos ?? []).some((p) => p !== "cancelado")}
      podeSoltar={(s, coluna) => aceita(colunas, s, coluna)}
      {...controles}
      aoSoltar={(s, coluna) => onMover(s, destino(s, coluna as Coluna))}
      moldura={moldura}
      aoAbrir={abrir}
      render={(s, vista) => {
        const moves = (s.proximos ?? []).filter((p) => p !== "cancelado");
        const foco = s.prazo ? relogioEmFoco(s.prazo) : null;
        const isVencido =
          foco?.relogio.situacao === "vencido" || foco?.relogio.situacao === "descumprido";
        const isCritico =
          foco?.relogio.situacao === "critico" || foco?.relogio.situacao === "atencao";
        const percentualSLA = Math.min(100, Math.round(foco?.relogio.percentual ?? 0));
        return (
          <>
            <div className="flex items-center justify-between gap-2 text-xs text-muted">
              <span className="tabular-nums">#{s.numero}</span>
              <Badge>{rotuloTipo(s.tipo)}</Badge>
            </div>
            <NomeClicavel
              nome={vista.titulo || s.titulo}
              href={`/atendimento/solicitacoes/${s.id}`}
              onSalvar={vista.salvarNome}
            />
            <p className="text-xs text-muted">
              {s.oferta_nome} · {s.responsavel_nome || "sem responsável"}
            </p>
            <p>
              <PrazoBadge prazo={s.prazo} />
            </p>
            {foco && foco.relogio.estado === "correndo" && (
              <div
                className="mt-0.5 flex flex-col gap-1"
                title={`Consumo de SLA: ${foco.relogio.percentual}%`}
              >
                <div className="flex items-center justify-between text-[11px] text-muted">
                  <span className="flex items-center gap-1 font-medium">
                    {isVencido && <AlertTriangle className="h-3 w-3 text-danger animate-pulse" />}
                    {isCritico && !isVencido && <AlertTriangle className="h-3 w-3 text-warning" />}
                    {foco.nome}
                  </span>
                  <span
                    className={`font-mono text-xs ${
                      isVencido
                        ? "font-bold text-danger"
                        : isCritico
                          ? "font-semibold text-warning"
                          : "text-muted"
                    }`}
                  >
                    {foco.relogio.percentual}%
                  </span>
                </div>
                <div className="h-1.5 w-full overflow-hidden rounded-full bg-surface-border">
                  <div
                    className={`h-full transition-all ${
                      isVencido ? "bg-danger" : isCritico ? "bg-warning" : "bg-success"
                    }`}
                    style={{ width: `${percentualSLA}%` }}
                  />
                </div>
              </div>
            )}
            <button
              type="button"
              className="text-left text-xs font-medium text-primary hover:underline"
              onClick={() => abrir(s)}
            >
              Editar
            </button>
            {moves.length > 0 && (
              <label className="flex flex-col gap-1 text-xs">
                <span className="sr-only">Mover a solicitação #{s.numero} para</span>
                <select
                  value=""
                  disabled={pending}
                  onChange={(e) => {
                    if (e.target.value) onMover(s, e.target.value as Estado);
                  }}
                  className="h-8 rounded-md border border-surface-border bg-surface px-2 text-xs focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
                >
                  <option value="">Mover para…</option>
                  {moves.map((p) => (
                    <option key={p} value={p}>
                      {colunas.find((coluna) => coluna.id === p)?.titulo ?? rotuloEstado(p)}
                    </option>
                  ))}
                </select>
              </label>
            )}
          </>
        );
      }}
    />
  );
}
