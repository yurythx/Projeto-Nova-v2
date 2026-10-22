import { fmtDateTime } from "@/components/sismarq/useAction";
import { Badge } from "@/components/ui/Badge";
import { SITUACAO, relogioEmFoco, type Prazo } from "@/lib/atendimento/atendimento";

/** Semáforo do prazo na fila e no quadro: o relógio em foco, a situação e,
 * se estiver correndo, quando vence. Sem prazo (unidade sem política),
 * diz isso — não inventa um. */
export function PrazoBadge({ prazo }: { prazo?: Prazo }) {
  if (!prazo) return <span className="text-xs text-muted">Sem prazo</span>;
  const { nome, relogio } = relogioEmFoco(prazo);
  const situacao = SITUACAO[relogio.situacao];
  return (
    <span className="inline-flex flex-col gap-0.5">
      <Badge tone={situacao.tone}>
        {nome}: {situacao.label}
        {relogio.estado === "pausado" ? " (pausado)" : ""}
      </Badge>
      {relogio.vence_em && (
        <span className="text-xs text-muted">vence {fmtDateTime(relogio.vence_em)}</span>
      )}
    </span>
  );
}
