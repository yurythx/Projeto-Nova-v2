import { fmtDateTime } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import type { Aprovacao } from "@/lib/atendimento/atendimento";

/** Aprovação em etapas do pedido (ADR 068): a etapa que falta e as
 * decisões já tomadas, com o motivo da recusa. */
export function AprovacoesCard({
  etapaAtual,
  aprovacoes,
}: {
  etapaAtual?: string;
  aprovacoes: Aprovacao[];
}) {
  if (!etapaAtual && aprovacoes.length === 0) return null;
  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Aprovação</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 pb-5">
        {etapaAtual && (
          <p
            role="status"
            className="rounded-lg border border-warning/40 bg-warning/10 p-3 text-sm"
          >
            Aguardando a etapa <strong>{etapaAtual}</strong>. A equipe só atende depois da
            aprovação.
          </p>
        )}
        {aprovacoes.length > 0 && (
          <ol className="flex flex-col gap-2">
            {aprovacoes.map((a) => (
              <li key={a.id} className="flex flex-col gap-1 text-sm">
                <span className="flex flex-wrap items-center gap-2">
                  <Badge tone={a.decisao === "aprovada" ? "success" : "danger"}>
                    {a.decisao === "aprovada" ? "Aprovada" : "Recusada"}
                  </Badge>
                  <span>
                    {a.etapa_nome}
                    {a.aprovador_nome ? ` · ${a.aprovador_nome}` : ""} · {fmtDateTime(a.created_at)}
                  </span>
                </span>
                {a.motivo && <span className="text-muted">{a.motivo}</span>}
              </li>
            ))}
          </ol>
        )}
      </CardContent>
    </Card>
  );
}
