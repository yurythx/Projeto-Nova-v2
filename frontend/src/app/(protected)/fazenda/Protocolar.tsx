"use client";

import { useRouter } from "next/navigation";
import { useState, type FormEvent } from "react";

import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { ApiError, apiClient } from "@/lib/api/client";

import { NOMES, pecasLegais } from "./pecas";
import { SERVICOS, type Programa } from "./Servicos";

interface Aberto {
  id: string;
  numero: string;
}

/** Formulário do serviço já escolhido. As peças seguem o Anexo II desse programa. */
export function Protocolar({ programa }: { programa: Programa }) {
  const router = useRouter();
  const [erro, setErro] = useState("");
  const [entrada, setEntrada] = useState("govbr");
  const servico = SERVICOS.find((item) => item.id === programa);
  const [moduloC, setModuloC] = useState(false);
  const [temContrato, setTemContrato] = useState(false);
  const [temCessao, setTemCessao] = useState(false);
  const [itbi, setItbi] = useState(false);
  const pecas = pecasLegais({ programa, entrada, itbi, moduloC, temCessao, temContrato });

  async function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setErro("");
    const form = new FormData(e.currentTarget);
    const emitida = String(form.get("emitida_em") ?? "");
    const titular = String(form.get("titular") ?? "").trim();
    const documentos = pecas
      .filter((tipo) => form.get(tipo) === "on")
      .map((tipo) => ({
        tipo,
        presente: true,
        emitida_em:
          tipo === "matricula" && emitida
            ? new Date(`${emitida}T12:00:00-04:00`).toISOString()
            : undefined,
      }));
    try {
      const { data } = await apiClient.post<Aberto>("v1/fazenda/processos", {
        programa,
        entrada,
        titular: titular || undefined,
        modulo_a: programa === "regulariza",
        modulo_c: moduloC,
        tem_contrato: temContrato,
        tem_cessao: temCessao,
        itbi,
        data_contrato: form.get("data_contrato")
          ? new Date(`${form.get("data_contrato")}T12:00:00-04:00`).toISOString()
          : undefined,
        documentos,
      });
      router.push(`/fazenda/${data.id}`);
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : "Não foi possível protocolar.");
    }
  }

  return (
    <Card className="max-w-3xl">
      <CardHeader>
        <CardTitle as="h2">Solicitação</CardTitle>
        <CardDescription>
          {servico?.titulo}. O envio segue com as peças obrigatórias marcadas.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={enviar} className="flex flex-col gap-4">
          <Select
            id="entrada"
            name="entrada"
            label="Entrada"
            options={[
              { value: "govbr", label: "gov.br" },
              { value: "balcao", label: "Balcão" },
              { value: "procurador", label: "Procurador" },
            ]}
            value={entrada}
            onChange={(e) => setEntrada(e.target.value)}
          />
          <Input
            id="titular"
            name="titular"
            label="Titular, quando o protocolo é de outra pessoa"
            autoComplete="off"
          />
          {programa === "regulariza" && (
            <fieldset className="flex flex-col gap-2">
              <legend className="text-sm font-medium">Condições do requerimento</legend>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  name="modulo_c"
                  checked={moduloC}
                  onChange={(e) => setModuloC(e.target.checked)}
                />{" "}
                Módulo C
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  name="tem_contrato"
                  checked={temContrato}
                  onChange={(e) => setTemContrato(e.target.checked)}
                />{" "}
                Há contrato
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  name="tem_cessao"
                  checked={temCessao}
                  onChange={(e) => setTemCessao(e.target.checked)}
                />{" "}
                Há cessão
              </label>
              <Input id="data_contrato" name="data_contrato" type="date" label="Data do contrato" />
            </fieldset>
          )}
          {programa === "avanca" && (
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                name="itbi"
                checked={itbi}
                onChange={(e) => setItbi(e.target.checked)}
              />{" "}
              ITBI no imóvel matriz
            </label>
          )}
          <fieldset className="flex flex-col gap-2">
            <legend className="text-sm font-medium">Peças do Anexo II</legend>
            {pecas.map((tipo) => (
              <label key={tipo} className="flex items-center gap-2 text-sm">
                <input type="checkbox" name={tipo} /> {NOMES[tipo]}
              </label>
            ))}
          </fieldset>
          {pecas.includes("matricula") && (
            <Input
              id="emitida_em"
              name="emitida_em"
              type="date"
              label="Data de emissão da matrícula"
            />
          )}
          {erro && (
            <p role="alert" className="text-sm text-danger">
              {erro}
            </p>
          )}
          <Button type="submit">Protocolar</Button>
        </form>
      </CardContent>
    </Card>
  );
}
