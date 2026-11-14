"use client";

import { useEffect, useState } from "react";

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { ApiError } from "@/lib/api/client";
import { publicPost } from "@/lib/api/publicClient";

import { lerDesafio, limparDesafio } from "./sessao";

interface Sessao {
  nome: string;
  email: string;
}

/** Troca o código do Login Único. Mostra nome e e-mail. O CPF não entra na tela. */
export function Retorno({ code, state }: { code: string; state: string }) {
  const [erro, setErro] = useState("");
  const [sessao, setSessao] = useState<Sessao | null>(null);

  useEffect(() => {
    let vivo = true;
    async function concluir() {
      if (!code || !state) {
        setErro("O Login Único não devolveu o código desta autorização.");
        return;
      }
      const guardado = lerDesafio();
      if (!guardado || guardado.state !== state) {
        limparDesafio();
        setErro("O retorno não confere com o pedido feito neste navegador.");
        return;
      }
      try {
        const data = await publicPost<Sessao>("v1/fazenda/govbr/sessao", {
          code,
          verifier: guardado.verifier,
        });
        limparDesafio();
        if (vivo) setSessao(data);
      } catch (err) {
        if (vivo) {
          setErro(err instanceof ApiError ? err.message : "Não foi possível concluir o Login Único.");
        }
      }
    }
    void concluir();
    return () => {
      vivo = false;
    };
  }, [code, state]);

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6 px-6 py-14">
      <h1 className="text-2xl font-bold tracking-tight">Retorno do gov.br</h1>
      {erro && (
        <p role="alert" className="text-sm text-danger">
          {erro}
        </p>
      )}
      {sessao && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Identificação</CardTitle>
          </CardHeader>
          <CardContent>
            <dl className="grid gap-3 text-sm">
              <div>
                <dt className="text-muted">Nome</dt>
                <dd>{sessao.nome}</dd>
              </div>
              <div>
                <dt className="text-muted">E-mail</dt>
                <dd>{sessao.email}</dd>
              </div>
            </dl>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
