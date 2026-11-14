"use client";

import { useState } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ApiError } from "@/lib/api/client";
import { publicFetch } from "@/lib/api/publicClient";

import { destinoGovBR, guardarDesafio, irPara } from "./sessao";

interface Desafio {
  state: string;
  verifier: string;
  url: string;
}

/** Começa o Login Único. A credencial ausente aparece na tela e o acesso não começa. */
export function Entrada() {
  const [erro, setErro] = useState("");

  async function entrar() {
    setErro("");
    try {
      const desafio = await publicFetch<Desafio>("v1/fazenda/govbr");
      const destino = destinoGovBR(desafio.url);
      if (!destino || !desafio.state || !desafio.verifier) {
        setErro("O endereço de autorização não é o do Login Único.");
        return;
      }
      guardarDesafio(desafio.state, desafio.verifier);
      irPara(destino);
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : "Não foi possível iniciar o Login Único.");
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6 px-6 py-14">
      <h1 className="text-2xl font-bold tracking-tight">Entrar com gov.br</h1>
      <p className="text-sm text-muted">
        O contribuinte entra pelo Login Único. O retorno volta para /govbr/retorno. O CPF fica no
        gov.br.
      </p>
      <Card>
        <CardHeader>
          <CardTitle as="h2">Login Único</CardTitle>
          <CardDescription>Sem credencial, o acesso não começa.</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <Button type="button" onClick={() => void entrar()}>
            Entrar com gov.br
          </Button>
          {erro && (
            <p role="alert" className="text-sm text-danger">
              {erro}
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
