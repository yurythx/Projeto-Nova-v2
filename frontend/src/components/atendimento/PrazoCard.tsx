"use client";

import { type FormEvent } from "react";

import { fmtDateTime } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Select } from "@/components/ui/Select";
import {
  NIVEIS,
  PRIORIDADE,
  SITUACAO,
  duracao,
  type Relogio,
  type Solicitacao,
} from "@/lib/atendimento/atendimento";

const ESTADO_RELOGIO: Record<Relogio["estado"], string> = {
  correndo: "correndo",
  pausado: "pausado enquanto aguarda",
  parado: "parado",
};

function Linha({ nome, relogio }: { nome: string; relogio: Relogio }) {
  const situacao = SITUACAO[relogio.situacao];
  const cor =
    situacao.tone === "success"
      ? "bg-success"
      : situacao.tone === "warning"
        ? "bg-warning"
        : "bg-danger";
  return (
    <div className="flex flex-col gap-1.5">
      <div className="flex flex-wrap items-center justify-between gap-2 text-sm">
        <span className="font-medium">{nome}</span>
        <Badge tone={situacao.tone}>{situacao.label}</Badge>
      </div>
      <div
        role="meter"
        aria-label={`Prazo de ${nome.toLowerCase()}`}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.min(relogio.percentual, 100)}
        aria-valuetext={`${relogio.percentual}% da meta`}
        className="h-2 overflow-hidden rounded-full bg-black/10 dark:bg-white/10"
      >
        <div
          className={`h-full ${cor}`}
          style={{ width: `${Math.min(relogio.percentual, 100)}%` }}
        />
      </div>
      <p className="text-xs text-muted">
        {duracao(relogio.usado_min)} de {duracao(relogio.meta_min)} úteis ·{" "}
        {ESTADO_RELOGIO[relogio.estado]}
        {relogio.vence_em ? ` · vence ${fmtDateTime(relogio.vence_em)}` : ""}
      </p>
    </div>
  );
}

/** Prazo e classificação da solicitação, para a equipe. Reclassificar
 * troca a prioridade e as metas pela política atual da unidade. */
export function PrazoCard({
  solicitacao: s,
  editavel,
  pending,
  onClassificar,
}: {
  solicitacao: Solicitacao;
  editavel: boolean;
  pending: boolean;
  onClassificar: (impacto: number, urgencia: number) => void;
}) {
  function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    onClassificar(Number(fd.get("impacto")), Number(fd.get("urgencia")));
  }

  const impactos = NIVEIS.map((n) => ({ value: String(n.value), label: n.label }));
  const urgencias = NIVEIS.map((n) => ({ value: String(n.value), label: n.feminino }));
  return (
    <Card>
      <CardHeader className="flex flex-row flex-wrap items-center justify-between gap-2">
        <CardTitle as="h2">Prazo</CardTitle>
        {s.prioridade ? (
          <Badge tone={PRIORIDADE[s.prioridade].tone}>
            Prioridade {PRIORIDADE[s.prioridade].label.toLowerCase()}
          </Badge>
        ) : (
          <Badge>Sem prioridade</Badge>
        )}
      </CardHeader>
      <CardContent className="flex flex-col gap-4 pb-5">
        {s.prazo ? (
          <>
            <Linha nome="Resposta" relogio={s.prazo.resposta} />
            <Linha nome="Solução" relogio={s.prazo.solucao} />
          </>
        ) : (
          <p className="text-sm text-muted">
            A unidade ainda não tem política de prazo. Cadastre em Atendimento → Prazos.
          </p>
        )}
        {editavel && (
          <form
            key={`${s.impacto}-${s.urgencia}`}
            onSubmit={enviar}
            className="flex flex-wrap items-end gap-2 border-t border-surface-border pt-4"
          >
            <div className="min-w-32">
              <Select
                id="prazo-impacto"
                name="impacto"
                label="Impacto"
                defaultValue={String(s.impacto)}
                options={impactos}
              />
            </div>
            <div className="min-w-32">
              <Select
                id="prazo-urgencia"
                name="urgencia"
                label="Urgência"
                defaultValue={String(s.urgencia)}
                options={urgencias}
              />
            </div>
            <Button type="submit" variant="secondary" loading={pending}>
              Reclassificar
            </Button>
          </form>
        )}
      </CardContent>
    </Card>
  );
}
