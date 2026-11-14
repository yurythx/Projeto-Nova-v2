"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState, type FormEvent } from "react";

import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { emCuiaba } from "../horario";
import { andamento, nomePrograma, nomeSituacao, tomSituacao } from "../rotulos";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/Card";
import { ApiError, apiClient } from "@/lib/api/client";
import { centavosDeReais } from "@/lib/moeda";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

interface Processo {
  id: string;
  numero: string;
  programa: string;
  situacao: string;
  pendencia: string;
  recibo_em: string;
  codigo?: string;
  exigencia_aberta?: string;
  exigencia_prazo: string;
  despacho_em?: string;
  ciencia_recibo?: string;
  ciencia_exigencia?: string;
  aliquota?: number;
  execucao?: string;
}

/** Acompanhamento e resposta de exigência. O texto da tela não repete débito. */
export default function ProcessoPage() {
  const { id } = useParams<{ id: string }>();
  const processo = useApiQuery<Processo>(id ? `v1/fazenda/processos/${id}` : null);
  const { can } = usePlataforma();
  const [erro, setErro] = useState("");
  const item = processo.data;
  const exigeAberta = item?.situacao === "em_exigencia";
  const encerrado = item?.situacao === "arquivado" || item?.situacao === "recurso";
  const emCurso = !!item && !exigeAberta && !encerrado && item.situacao !== "decidido";
  const podeExigir = emCurso && can("fazenda:analisar");
  const podeDecidir = emCurso && can("fazenda:decidir");
  const podeRecorrer = item?.situacao === "decidido" && can("fazenda:decidir");
  const analisa = can("fazenda:analisar") && !encerrado && !exigeAberta;
  const decide = can("fazenda:decidir") && !encerrado && !exigeAberta;

  useEffect(() => {
    if (!id) return;
    let vivo = true;
    void apiClient
      .post(`v1/fazenda/processos/${id}/consulta`, {})
      .then(() => {
        if (vivo) void processo.mutate();
      })
      .catch(() => undefined);
    return () => {
      vivo = false;
    };
    // A consulta registra a ciência uma vez por abertura da ficha.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  async function agir(caminho: string, corpo: unknown) {
    setErro("");
    try {
      await apiClient.post(`v1/fazenda/processos/${id}/${caminho}`, corpo);
      await processo.mutate();
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : "Não foi possível registrar o ato.");
    }
  }

  async function responder(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setErro("");
    const respondeu = new FormData(e.currentTarget).get("respondeu") === "sim";
    await agir("resposta", { respondeu });
  }

  async function exigir(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fatoNovo = new FormData(e.currentTarget).get("fato_novo") === "sim";
    await agir("exigencia", { fato_novo: fatoNovo });
  }

  async function recorrer(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const rito = new FormData(e.currentTarget).get("rito") === "sim";
    await agir("recurso", { rito_entregue: rito });
  }

  async function parcelar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = new FormData(e.currentTarget);
    await agir("parcelamento", {
      total: centavosDeReais(String(form.get("total") ?? "")),
      parcelas: Number(form.get("parcelas") ?? 0),
    });
  }

  async function baixar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const primeira = new FormData(e.currentTarget).get("primeira") === "sim";
    await agir("baixa", { primeira });
  }

  async function reduzir(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = new FormData(e.currentTarget);
    await agir("reducao", {
      milessimos: Number(form.get("milessimos") ?? 0),
      condicoes: Number(form.get("condicoes") ?? 0),
    });
  }

  async function execucao(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const emExecucao = new FormData(e.currentTarget).get("em_execucao") === "sim";
    await agir("execucao", { em_execucao: emExecucao });
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Secretaria Municipal de Fazenda"
        title={item?.numero ?? "Processo"}
        description="Situação, recibo e prazo deste protocolo. O aviso externo leva o número e a orientação."
        actions={
          <Link href="/fazenda" className={buttonClass("secondary", "sm")}>
            Voltar
          </Link>
        }
      />
      <DataState
        loading={processo.isLoading}
        error={processo.error}
        empty={!item}
        emptyTitle="Processo não encontrado"
        onRetry={() => void processo.mutate()}
      >
        {item && (
          <Card className="max-w-3xl">
            <CardHeader className="flex items-start justify-between gap-3">
              <div>
                <CardTitle as="h2">{nomePrograma(item.programa)}</CardTitle>
                <CardDescription>Recibo em {emCuiaba(item.recibo_em)}</CardDescription>
              </div>
              <Badge tone={tomSituacao(item.situacao)}>{nomeSituacao(item.situacao)}</Badge>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              {item.pendencia && <p className="text-sm">{item.pendencia}</p>}
              {emCuiaba(item.ciencia_recibo) && (
                <p className="text-sm">Ciência do recibo em {emCuiaba(item.ciencia_recibo)}</p>
              )}
              {emCuiaba(item.ciencia_exigencia) && (
                <p className="text-sm">Ciência da exigência em {emCuiaba(item.ciencia_exigencia)}</p>
              )}
              {!!item.aliquota && (
                <p className="text-sm">
                  Alíquota anotada: {(item.aliquota / 1000).toLocaleString("pt-BR")}%
                </p>
              )}
              {item.execucao === "procuradoria" && (
                <p className="text-sm">Sinalizado para a Procuradoria. A suspensão é ato dela.</p>
              )}
              {item.codigo && (
                <p className="text-sm">
                  Código de autenticidade <span className="font-medium">{item.codigo}</span>{" "}
                  <Link href="/autenticidade" className="text-primary">
                    Confirmar autenticidade
                  </Link>
                </p>
              )}
              <section aria-labelledby="andamento">
                <h3 id="andamento" className="text-sm font-semibold">
                  Andamento
                </h3>
                <ol className="mt-3 flex flex-col gap-3 border-l border-surface-border pl-4">
                  {andamento(item).map((marco) => (
                    <li key={marco.rotulo}>
                      <p className="text-sm font-medium">{marco.rotulo}</p>
                      <p className="text-xs text-muted">{marco.quando}</p>
                    </li>
                  ))}
                </ol>
              </section>
              {exigeAberta && (
                <div
                  role="status"
                  className="flex flex-col gap-3 rounded-lg border border-warning/40 bg-warning/10 p-4"
                >
                  <h3 className="text-sm font-semibold">Exigência em aberto</h3>
                  {emCuiaba(item.exigencia_prazo) && (
                    <p className="text-sm">Prazo {emCuiaba(item.exigencia_prazo)}</p>
                  )}
                  <form onSubmit={responder} className="flex flex-col gap-3">
                    <label className="flex items-center gap-2 text-sm">
                      <input type="checkbox" name="respondeu" value="sim" /> A exigência foi
                      respondida
                    </label>
                    <Button type="submit">Registrar resposta</Button>
                  </form>
                </div>
              )}
              {(podeExigir || podeDecidir || podeRecorrer || analisa || decide) && (
                <section aria-labelledby="atos" className="flex flex-col gap-3">
                  <h3 id="atos" className="text-sm font-semibold">
                    Atos da equipe
                  </h3>
                  {podeExigir && (
                    <form onSubmit={exigir} className="flex flex-col gap-3">
                      <label className="flex items-center gap-2 text-sm">
                        <input type="checkbox" name="fato_novo" value="sim" /> Há fato ou documento
                        novo
                      </label>
                      <Button type="submit">Abrir exigência</Button>
                    </form>
                  )}
                  {podeDecidir && (
                    <Button type="button" onClick={() => void agir("decisao", {})}>
                      Registrar decisão
                    </Button>
                  )}
                  {podeRecorrer && (
                    <form onSubmit={recorrer} className="flex flex-col gap-3">
                      <label className="flex items-center gap-2 text-sm">
                        <input type="checkbox" name="rito" value="sim" /> A SEFAZ já descreveu o
                        rito da Lei 1.800/1990
                      </label>
                      <Button type="submit">Registrar recurso</Button>
                    </form>
                  )}
                  {analisa && item.programa === "regulariza" && item.situacao !== "decidido" && (
                    <form onSubmit={parcelar} className="flex flex-col gap-3">
                      <Input id="total" name="total" label="Valor do débito" placeholder="1.200,00" />
                      <Input id="parcelas" name="parcelas" type="number" min={1} max={12} label="Parcelas" />
                      <Button type="submit">Calcular parcelamento</Button>
                    </form>
                  )}
                  {analisa && item.situacao === "aguardando_pagamento" && (
                    <form onSubmit={baixar} className="flex flex-col gap-3">
                      <label className="flex items-center gap-2 text-sm">
                        <input type="checkbox" name="primeira" value="sim" /> A primeira parcela foi
                        paga
                      </label>
                      <Button type="submit">Registrar retorno</Button>
                    </form>
                  )}
                  {decide && item.programa === "regulariza" && (
                    <form onSubmit={reduzir} className="flex flex-col gap-3">
                      <Input
                        id="milessimos"
                        name="milessimos"
                        type="number"
                        min={1}
                        label="Alíquota ordinária em milésimos"
                      />
                      <Input
                        id="condicoes"
                        name="condicoes"
                        type="number"
                        min={0}
                        max={5}
                        label="Condições do módulo C"
                      />
                      <Button type="submit">Anotar redução</Button>
                    </form>
                  )}
                  {analisa && (
                    <form onSubmit={execucao} className="flex flex-col gap-3">
                      <label className="flex items-center gap-2 text-sm">
                        <input type="checkbox" name="em_execucao" value="sim" /> Há débito em
                        execução fiscal
                      </label>
                      <Button type="submit">Sinalizar execução fiscal</Button>
                    </form>
                  )}
                </section>
              )}
              {erro && (
                <p role="alert" className="text-sm text-danger">
                  {erro}
                </p>
              )}
            </CardContent>
          </Card>
        )}
      </DataState>
    </div>
  );
}
