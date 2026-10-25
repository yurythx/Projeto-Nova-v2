import { ESTADO_MUDANCA, RISCO, TIPO_MUDANCA, type Mudanca } from "@/lib/servicos/servicos";

function quando(valor?: string | null) {
  if (!valor) return "não informado";
  return new Date(valor).toLocaleString("pt-BR");
}

const DECISAO: Record<string, string> = {
  aprovada: "Aprovada",
  rejeitada: "Rejeitada",
  automatica: "Pré-aprovada (padrão)",
};

/** Folha da RFC para leitura e impressão. O navegador gera o PDF. */
export function RfcDocumento({ mudanca }: { mudanca: Mudanca }) {
  return (
    <article id="rfc-documento" className="flex flex-col gap-4 text-sm text-foreground" aria-labelledby="rfc-titulo">
      <header className="border-b border-surface-border pb-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-muted">Requisição de mudança</p>
        <h2 id="rfc-titulo" className="text-lg font-semibold">
          RFC #{mudanca.numero} — {mudanca.titulo}
        </h2>
        <p className="text-muted">
          {TIPO_MUDANCA[mudanca.tipo]} · {RISCO[mudanca.risco].label}
          {mudanca.risco_score != null ? ` (${mudanca.risco_score}/15)` : ""} · {ESTADO_MUDANCA[mudanca.estado].label}
        </p>
      </header>
      <dl className="grid gap-3 sm:grid-cols-2">
        <div>
          <dt className="text-xs text-muted">Solicitante</dt>
          <dd>{mudanca.solicitante_nome}</dd>
        </div>
        <div>
          <dt className="text-xs text-muted">Janela planejada</dt>
          <dd>
            {quando(mudanca.inicio_planejado)} até {quando(mudanca.fim_planejado)}
          </dd>
        </div>
        <div className="sm:col-span-2">
          <dt className="text-xs text-muted">O que muda</dt>
          <dd className="whitespace-pre-wrap">{mudanca.descricao || "não informado"}</dd>
        </div>
        <div className="sm:col-span-2">
          <dt className="text-xs text-muted">Justificativa</dt>
          <dd className="whitespace-pre-wrap">{mudanca.justificativa || "não informada"}</dd>
        </div>
        <div className="sm:col-span-2">
          <dt className="text-xs text-muted">Plano de reversão</dt>
          <dd className="whitespace-pre-wrap">{mudanca.plano_reversao || "não informado"}</dd>
        </div>
      </dl>
      <section>
        <h3 className="text-sm font-semibold">Decisões do comitê</h3>
        {mudanca.aprovacoes.length === 0 ? (
          <p className="text-muted">Nenhuma decisão registrada.</p>
        ) : (
          <ol className="mt-2 flex flex-col gap-2">
            {mudanca.aprovacoes.map((a) => (
              <li key={a.id}>
                <p className="font-medium">
                  {DECISAO[a.decisao] ?? a.decisao} · {a.aprovador} · {quando(a.em)}
                </p>
                {a.parecer ? <p className="whitespace-pre-wrap text-muted">{a.parecer}</p> : null}
              </li>
            ))}
          </ol>
        )}
      </section>
      <section>
        <h3 className="text-sm font-semibold">Vínculos</h3>
        {mudanca.vinculos.length === 0 ? (
          <p className="text-muted">Nenhum vínculo.</p>
        ) : (
          <ul className="mt-2 list-disc pl-5">
            {mudanca.vinculos.map((v) => (
              <li key={`${v.tipo}-${v.alvo_id}`}>{v.rotulo}</li>
            ))}
          </ul>
        )}
      </section>
      <p className="text-xs text-muted">
        A decisão registrada acima é o parecer do comitê. Esta folha não substitui o registro da mudança no sistema.
      </p>
    </article>
  );
}
