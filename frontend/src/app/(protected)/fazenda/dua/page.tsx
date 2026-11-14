"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";

import { PageHeader } from "@/components/plataforma/PageHeader";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { ApiError, apiClient } from "@/lib/api/client";
import { reais } from "@/lib/moeda";

interface Linha {
  inscricao: string;
  matricula: string;
  adquirente: string;
  documento: string;
  endereco: string;
  data: string;
  situacao: string;
  quitacao: string;
}

interface Recibo {
  recibo: string;
  multa: number;
  linhas: number;
  falhas: { linha: number; motivo: string }[];
}

function linhasDaPlanilha(texto: string): Linha[] {
  return texto
    .split(/\r?\n/)
    .map((linha) => linha.trim())
    .filter(Boolean)
    .map((linha) => {
      const [inscricao, matricula, adquirente, documento, endereco, data, situacao, quitacao] =
        linha.split(";").map((parte) => parte.trim());
      return {
        inscricao: inscricao ?? "",
        matricula: matricula ?? "",
        adquirente: adquirente ?? "",
        documento: documento ?? "",
        endereco: endereco ?? "",
        data: data ?? "",
        situacao: situacao ?? "",
        quitacao: quitacao ?? "",
      };
    });
}

/** Entrega da DUA na área restrita. O recibo mostrado não repete o documento da linha. */
export default function DuaPage() {
  const [recibo, setRecibo] = useState<Recibo | null>(null);
  const [erro, setErro] = useState("");

  async function entregar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setErro("");
    const form = new FormData(e.currentTarget);
    const omitidas = Number(form.get("omitidas") ?? 0);
    const linha = {
      inscricao: String(form.get("inscricao") ?? "").trim(),
      matricula: String(form.get("matricula") ?? "").trim(),
      adquirente: String(form.get("adquirente") ?? "").trim(),
      documento: String(form.get("documento") ?? "").trim(),
      endereco: String(form.get("endereco") ?? "").trim(),
      data: String(form.get("data") ?? "").trim(),
      situacao: String(form.get("situacao") ?? ""),
      quitacao: String(form.get("quitacao") ?? "").trim(),
    };
    const linhas = [
      ...linhasDaPlanilha(String(form.get("planilha") ?? "")),
      ...(linha.inscricao ? [linha] : []),
    ];
    try {
      const { data } = await apiClient.post<Recibo>("v1/fazenda/processos/dua", {
        linhas,
        omitidas,
      });
      setRecibo(data);
    } catch (err) {
      setRecibo(null);
      setErro(err instanceof ApiError ? err.message : "Não foi possível receber a DUA.");
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Secretaria Municipal de Fazenda"
        title="Entrega da DUA"
        description="Cada linha da planilha usa ponto e vírgula: inscrição, matrícula, adquirente, documento, endereço, data, situação e quitação. O recibo externo não repete esses dados."
        actions={
          <Link href="/fazenda" className={buttonClass("secondary", "sm")}>
            Voltar
          </Link>
        }
      />
      <Card className="max-w-3xl">
        <CardHeader>
          <CardTitle as="h2">Planilha</CardTitle>
          <CardDescription>
            Uma unidade por linha. A multa aparece só nesta área restrita.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={entregar} className="grid gap-3">
            <Textarea
              id="planilha"
              name="planilha"
              label="Planilha"
              rows={4}
              placeholder="Uma unidade por linha, separada por ponto e vírgula"
            />
            <Input id="inscricao" name="inscricao" label="Inscrição" autoComplete="off" />
            <Input id="matricula" name="matricula" label="Matrícula" autoComplete="off" />
            <Input id="adquirente" name="adquirente" label="Adquirente" autoComplete="off" />
            <Input id="documento" name="documento" label="Documento" autoComplete="off" />
            <Input id="endereco" name="endereco" label="Endereço" autoComplete="off" />
            <Input id="data" name="data" label="Data" placeholder="DD/MM/AAAA" autoComplete="off" />
            <Select
              id="situacao"
              name="situacao"
              label="Situação"
              defaultValue="Em curso"
              options={[
                { value: "Em curso", label: "Em curso" },
                { value: "Quitado", label: "Quitado" },
              ]}
            />
            <Input id="quitacao" name="quitacao" label="Quitação" autoComplete="off" />
            <Input
              id="omitidas"
              name="omitidas"
              type="number"
              min={0}
              label="Unidades omitidas"
              defaultValue={0}
            />
            {erro && (
              <p role="alert" className="text-sm text-danger">
                {erro}
              </p>
            )}
            <Button type="submit">Entregar</Button>
          </form>
        </CardContent>
      </Card>
      {recibo && (
        <Card className="max-w-3xl">
          <CardHeader>
            <CardTitle as="h2">Recibo</CardTitle>
            <CardDescription>
              {recibo.linhas} linha(s). Multa na área restrita: {reais(recibo.multa)}.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-2 text-sm">
            <p>{recibo.recibo}</p>
            {recibo.falhas.map((falha) => (
              <p key={falha.linha} role="alert">
                Linha {falha.linha}: {falha.motivo}
              </p>
            ))}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
