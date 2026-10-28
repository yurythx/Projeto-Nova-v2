"use client";

import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";

/** O relatório do desfazer da carga (ADR 079). */
export interface Desfeito {
  gravou: boolean;
  chamados: number;
  ofertas: number;
  categorias: number;
  equipes: number;
  politicas: number;
  feriados: number;
  chamados_mantidos: number;
  exemplos_mantidos: number[];
  ofertas_mantidas: number;
  cadastros_mantidos: number;
}

/** Resumo legível do desfazer, simulado ou feito. */
export function ResumoDesfazer({ relatorio: r }: { relatorio: Desfeito }) {
  const verbo = r.gravou ? "removido(s)" : "seriam removido(s)";
  return (
    <section
      aria-label="Desfazer a carga"
      className="rounded-lg border border-surface-border p-3 text-sm"
    >
      <h4 className="font-semibold">
        Desfazer a carga{r.gravou ? "" : " (simulação: nada foi removido)"}
      </h4>
      <ul className="mt-2 flex list-disc flex-col gap-1 pl-5">
        <li>
          {r.chamados} chamado(s) {verbo}, com o histórico, os comentários e as avaliações.
        </li>
        <li>
          {r.ofertas} oferta(s), {r.categorias} categoria(s), {r.equipes} equipe(s), {r.politicas}{" "}
          política(s) de prazos e {r.feriados} feriado(s) criados pela carga {verbo}.
        </li>
        {r.chamados_mantidos > 0 && (
          <li>
            {r.chamados_mantidos} chamado(s) ficam porque alguém mexeu neles aqui depois da carga
            {r.exemplos_mantidos.length > 0 ? ` (nº ${r.exemplos_mantidos.join(", ")}…)` : ""}.
          </li>
        )}
        {r.ofertas_mantidas + r.cadastros_mantidos > 0 && (
          <li>
            {r.ofertas_mantidas} oferta(s) e {r.cadastros_mantidos} outro(s) cadastro(s) ficam
            porque ainda estão em uso ou foram alterados aqui.
          </li>
        )}
      </ul>
    </section>
  );
}

/**
 * Desfaz a carga: simular primeiro e, para remover de fato, digitar o
 * nome da conexão. O que alguém mexeu aqui depois da carga fica.
 */
export function DesfazerCarga({
  nome,
  ocupado,
  onPedir,
}: {
  nome: string;
  ocupado: boolean;
  onPedir: (tipo: "desfazer_simulacao" | "desfazer", confirmacao?: string) => Promise<boolean>;
}) {
  const [confirmacao, setConfirmacao] = useState("");
  const confere = confirmacao.trim() === nome;

  async function desfazer() {
    if (await onPedir("desfazer", confirmacao.trim())) setConfirmacao("");
  }

  return (
    <section aria-labelledby="desfazer-titulo" className="grid gap-3">
      <h3 id="desfazer-titulo" className="font-semibold">
        Desfazer a carga
      </h3>
      <p className="text-sm text-muted">
        Remove os chamados que vieram do GLPI e os cadastros que a carga criou (ofertas, categorias,
        equipes, política de prazos e feriados). Fica o que alguém mexeu aqui depois da carga e o
        que ainda está em uso. O GLPI não é alterado. Simule antes e faça um backup do banco.
      </p>
      <div>
        <Button
          type="button"
          variant="secondary"
          disabled={ocupado}
          onClick={() => void onPedir("desfazer_simulacao")}
        >
          Simular desfazer
        </Button>
      </div>
      <div className="flex flex-wrap items-end gap-2">
        <label className="grid gap-1 text-sm">
          <span>
            Para remover, digite o nome da conexão: <strong>{nome}</strong>
          </span>
          <Input
            value={confirmacao}
            onChange={(e) => setConfirmacao(e.target.value)}
            autoComplete="off"
          />
        </label>
        <Button
          type="button"
          variant="danger"
          disabled={ocupado || !confere}
          onClick={() => void desfazer()}
        >
          Desfazer carga
        </Button>
      </div>
    </section>
  );
}
