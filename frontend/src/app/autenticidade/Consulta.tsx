"use client";

import { useState, type FormEvent } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { ApiError } from "@/lib/api/client";
import { publicFetch } from "@/lib/api/publicClient";

interface Resultado {
  codigo: string;
  numero: string;
  valido: boolean;
  mensagem: string;
}

/** Confere o código no protocolo e desenha o QR neste navegador. */
export function Consulta() {
  const [resultado, setResultado] = useState<Resultado | null>(null);
  const [qr, setQr] = useState("");
  const [erro, setErro] = useState("");

  async function consultar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setErro("");
    const codigo = String(new FormData(e.currentTarget).get("codigo") ?? "").trim();
    try {
      const data = await publicFetch<Resultado>(
        `v1/fazenda/autenticidade/${encodeURIComponent(codigo)}`,
      );
      setResultado(data);
      if (!data.valido) {
        setQr("");
        return;
      }
      const { default: QRCode } = await import("qrcode");
      const svg = await QRCode.toString(codigo, { type: "svg", margin: 1 });
      setQr(`data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`);
    } catch (err) {
      setResultado(null);
      setQr("");
      setErro(err instanceof ApiError ? err.message : "Não foi possível confirmar o código.");
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6 px-6 py-14">
      <h1 className="text-2xl font-bold tracking-tight">Autenticidade</h1>
      <p className="text-sm text-muted">
        O código confirma o documento. O QR é gerado aqui e não sai para outro serviço.
      </p>
      <form onSubmit={consultar} className="flex flex-col gap-3">
        <Input id="codigo" name="codigo" label="Código" required autoComplete="off" />
        <Button type="submit">Confirmar</Button>
      </form>
      {erro && (
        <p role="alert" className="text-sm text-danger">
          {erro}
        </p>
      )}
      {resultado && (
        <Card>
          <CardHeader className="flex items-start justify-between gap-3">
            <div>
              <CardTitle as="h2">
                {resultado.valido ? "Documento autêntico" : "Código não confirmado"}
              </CardTitle>
              <CardDescription>{resultado.mensagem}</CardDescription>
            </div>
            <Badge tone={resultado.valido ? "success" : "danger"}>
              {resultado.valido ? "Confirmado" : "Não confirmado"}
            </Badge>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 text-sm">
            {resultado.numero && <p>Processo {resultado.numero}</p>}
            {qr && (
              <img
                src={qr}
                alt={`QR do código ${resultado.codigo || "consultado"}`}
                className="h-40 w-40"
              />
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
