import { Badge } from "@/components/ui/Badge";
import {
  ESTADO,
  IMPACTO,
  SITUACAO,
  fmtHorario,
  type OcorrenciaPublica,
  type Situacao,
} from "@/lib/ocorrencias/ocorrencias";

function Ocorrencia({ o }: { o: OcorrenciaPublica }) {
  const impacto = IMPACTO[o.impacto];
  return (
    <article
      className="flex flex-col gap-3 rounded-lg border border-surface-border p-4"
      aria-labelledby={`oc-${o.id}`}
    >
      <div className="flex flex-wrap items-center gap-2">
        <h3 id={`oc-${o.id}`} className="text-base font-semibold text-foreground">
          {o.titulo}
        </h3>
        <Badge tone={o.estado === "resolvido" ? "success" : impacto.tone}>
          {o.estado === "resolvido" ? "Resolvido" : impacto.label}
        </Badge>
      </div>
      <p className="text-sm text-muted">
        Afeta: {o.servicos.join(", ")} · desde {fmtHorario(o.iniciada_em)}
        {o.resolvida_em && ` · resolvido em ${fmtHorario(o.resolvida_em)}`}
      </p>
      <ol className="flex flex-col gap-3 border-l border-surface-border pl-4">
        {o.atualizacoes.map((a) => (
          <li key={`${a.em}-${a.estado}`} className="flex flex-col gap-0.5 text-sm">
            <span className="font-medium text-foreground">
              {ESTADO[a.estado]}{" "}
              <span className="font-normal text-muted">· {fmtHorario(a.em)}</span>
            </span>
            <span className="whitespace-pre-line text-foreground">{a.texto}</span>
          </li>
        ))}
      </ol>
    </article>
  );
}

/** A página de status pública: a situação geral, o que está aberto e o
 * que foi resolvido na última semana (ADR 066). */
export function PaginaDeStatus({ situacao }: { situacao: Situacao }) {
  const geral = SITUACAO[situacao.geral];
  return (
    <div className="flex flex-col gap-8">
      <section
        role="status"
        aria-live="polite"
        className={`rounded-lg border p-5 text-lg font-semibold ${
          geral.tone === "success"
            ? "border-success/40 bg-success/10 text-success"
            : geral.tone === "warning"
              ? "border-warning/40 bg-warning/10 text-warning"
              : "border-danger/40 bg-danger/10 text-danger"
        }`}
      >
        {geral.titulo}
      </section>

      {situacao.abertas.length > 0 && (
        <section className="flex flex-col gap-4" aria-labelledby="abertas">
          <h2 id="abertas" className="text-xl font-semibold">
            Em andamento
          </h2>
          {situacao.abertas.map((o) => (
            <Ocorrencia key={o.id} o={o} />
          ))}
        </section>
      )}

      <section className="flex flex-col gap-4" aria-labelledby="recentes">
        <h2 id="recentes" className="text-xl font-semibold">
          Últimos 7 dias
        </h2>
        {situacao.recentes.length === 0 ? (
          <p className="text-sm text-muted">Nenhuma ocorrência resolvida nos últimos 7 dias.</p>
        ) : (
          situacao.recentes.map((o) => <Ocorrencia key={o.id} o={o} />)
        )}
      </section>
    </div>
  );
}
