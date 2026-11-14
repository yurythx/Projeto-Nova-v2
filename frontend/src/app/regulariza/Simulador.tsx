"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";

import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { publicPost } from "@/lib/api/publicClient";
import { reais } from "@/lib/moeda";

interface Simulado {
  devido: number;
  remitido: number;
  itbi: number;
  plano?: { parcelas: number; valor: number };
}

/** Formulário do simulador. Não abre processo e não mostra CPF. */
export function Simulador() {
  const [resultado, setResultado] = useState<Simulado | null>(null);
  const [erro, setErro] = useState("");

  async function simular(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setErro("");
    const inscricao = String(new FormData(e.currentTarget).get("inscricao") ?? "").trim();
    try {
      setResultado(await publicPost<Simulado>("v1/fazenda/simulador", { inscricao }));
    } catch {
      setResultado(null);
      setErro("Não foi possível simular esta inscrição.");
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6 px-6 py-14">
      <h1 className="text-2xl font-bold tracking-tight">Simulador do Regulariza</h1>
      <p className="text-sm text-muted">
        A consulta mostra débitos com remissão, parcelas e ITBI estimado. Ela não abre processo. Para
        protocolar, entre com o gov.br.
      </p>
      <Link href="/govbr" className={buttonClass("secondary", "sm")}>
        Entrar com gov.br
      </Link>
      <form onSubmit={simular} className="flex flex-col gap-3">
        <Input id="inscricao" name="inscricao" label="Inscrição" required autoComplete="off" />
        <Button type="submit">Simular</Button>
      </form>
      {erro && (
        <p role="alert" className="text-sm text-danger">
          {erro}
        </p>
      )}
      {resultado && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Estimativa</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid gap-3 text-sm sm:grid-cols-3">
              <div>
                <dt className="text-muted">Devido</dt>
                <dd>{reais(resultado.devido)}</dd>
              </div>
              <div>
                <dt className="text-muted">Remitido</dt>
                <dd>{reais(resultado.remitido)}</dd>
              </div>
              <div>
                <dt className="text-muted">ITBI estimado</dt>
                <dd>{reais(resultado.itbi)}</dd>
              </div>
            </dl>
            {resultado.plano && resultado.plano.parcelas > 0 && (
              <p className="mt-3 text-sm text-muted">
                {resultado.plano.parcelas} parcelas de {reais(resultado.plano.valor)}. O estado
                permanece aguardando pagamento até a primeira baixa.
              </p>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
