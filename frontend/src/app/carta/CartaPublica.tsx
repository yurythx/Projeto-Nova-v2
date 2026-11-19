"use client";

import Link from "next/link";
import { useEffect, useMemo, useState, type FormEvent } from "react";

import { fmtDateTime } from "@/components/plataforma/useAction";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { ApiError } from "@/lib/api/client";
import { publicFetch, publicPost } from "@/lib/api/publicClient";
import type { UploadTicket } from "@/lib/plataforma/types";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";

interface Servico {
  id: string;
  nome: string;
  descricao: string;
  categoria: string;
  prazo_dias: number;
  destaque: boolean;
  taxa_centavos: number;
}

interface Historico {
  id?: string;
  acao: string;
  descricao: string;
  status: string;
  created_at: string;
}

interface Consulta {
  numero: string;
  status: string;
  status_rotulo: string;
  prioridade: string;
  nome: string;
  servico: string;
  unidade: string;
  criado_em: string;
  prazo_em?: string;
  complemento?: string;
  historico: Historico[];
  agendamento?: { data: string; hora: string; status: string };
}

const PRIORIDADE: Record<string, string> = {
  baixa: "Baixa",
  normal: "Normal",
  alta: "Alta",
  urgente: "Urgente",
};

export function CartaPublica() {
  const [servicos, setServicos] = useState<Servico[] | null>(null);
  const [filtro, setFiltro] = useState("");
  const [consulta, setConsulta] = useState<Consulta | null>(null);
  const [erro, setErro] = useState("");

  useEffect(() => {
    let vivo = true;
    publicFetch<Servico[]>("v1/protocolos/carta")
      .then((lista) => {
        if (vivo) setServicos(lista);
      })
      .catch((e: unknown) => {
        if (vivo) setErro(e instanceof ApiError ? e.message : "A carta não pôde ser carregada.");
      });
    return () => {
      vivo = false;
    };
  }, []);

  const visiveis = useMemo(() => {
    const termo = filtro.trim().toLowerCase();
    return (servicos ?? []).filter((servico) => {
      if (!termo) return true;
      return `${servico.nome} ${servico.categoria} ${servico.descricao}`.toLowerCase().includes(termo);
    });
  }, [servicos, filtro]);

  async function consultar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setErro("");
    setConsulta(null);
    const numero = String(new FormData(e.currentTarget).get("numero") ?? "").trim();
    try {
      setConsulta(await publicFetch<Consulta>(`v1/protocolos/consulta/${encodeURIComponent(numero)}`));
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : "Número não encontrado.");
    }
  }

  return (
    <div className="mx-auto flex w-full max-w-3xl flex-col gap-8 px-6 py-14">
      <section className="flex flex-col gap-4">
        <h1 className="text-3xl font-bold tracking-tight">Protocolos digitais</h1>
        <p className="max-w-xl text-sm text-muted">
          Abra um pedido da carta de serviços ou consulte o andamento pelo número. A consulta não mostra CPF, telefone
          nem as respostas do formulário.
        </p>
        <div className="flex flex-wrap gap-3">
          <a href="#abrir" className={buttonClass("primary")}>
            Abrir protocolo
          </a>
          <a href="#consultar" className={buttonClass("secondary")}>
            Consultar protocolo
          </a>
        </div>
      </section>

      <Card id="consultar">
        <CardHeader>
          <CardTitle as="h2">Consultar protocolo</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4 pb-5">
          <form onSubmit={consultar} className="flex flex-col gap-3 sm:flex-row sm:items-end">
            <Input id="numero" name="numero" label="Número *" required maxLength={40} autoComplete="off" />
            <Button type="submit">Consultar</Button>
          </form>
          {consulta && <ResultadoConsulta consulta={consulta} onAtualizar={setConsulta} onErro={setErro} />}
        </CardContent>
      </Card>

      {erro && <p className="text-sm text-danger">{erro}</p>}

      <section id="abrir" className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">Abrir protocolo</h2>
        <Input
          id="filtro"
          label="Filtrar por nome ou categoria"
          value={filtro}
          onChange={(e) => setFiltro(e.target.value)}
        />
        {servicos && visiveis.length === 0 && <p className="text-sm text-muted">Nenhum serviço encontrado.</p>}
        <ul className="flex flex-col gap-3">
          {visiveis.map((servico) => (
            <li key={servico.id}>
              <Card>
                <CardHeader>
                  <CardTitle as="h3">{servico.nome}</CardTitle>
                </CardHeader>
                <CardContent className="flex items-end justify-between gap-3 pb-5">
                  <p className="text-sm text-muted">
                    {servico.destaque ? "Destaque · " : ""}
                    {servico.categoria || servico.descricao || "Serviço da carta"} · prazo {servico.prazo_dias} dias úteis
                  </p>
                  <Link href={`/carta/${servico.id}`} className={buttonClass("primary", "sm")}>
                    Ver detalhes
                  </Link>
                </CardContent>
              </Card>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}

function ResultadoConsulta({
  consulta,
  onAtualizar,
  onErro,
}: {
  consulta: Consulta;
  onAtualizar: (consulta: Consulta) => void;
  onErro: (erro: string) => void;
}) {
  const [enviando, setEnviando] = useState(false);

  async function responder(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setEnviando(true);
    onErro("");
    const dados = new FormData(e.currentTarget);
    const cpf = String(dados.get("cpf") ?? "").trim();
    const mensagem = String(dados.get("mensagem") ?? "").trim();
    const arquivos = dados.getAll("arquivos").filter((item): item is File => item instanceof File && item.size > 0);
    try {
      const chaves: string[] = [];
      for (const arquivo of arquivos) {
        const ticket = await publicPost<UploadTicket>(
          `v1/protocolos/consulta/${encodeURIComponent(consulta.numero)}/arquivo`,
          { cpf, nome: arquivo.name, content_type: mimeOf(arquivo) },
        );
        await putToTicket(ticket, arquivo);
        chaves.push(ticket.object_key);
      }
      onAtualizar(
        await publicPost<Consulta>(`v1/protocolos/consulta/${encodeURIComponent(consulta.numero)}/resposta`, {
          cpf,
          mensagem,
          arquivos: chaves,
        }),
      );
    } catch (err) {
      onErro(err instanceof ApiError || err instanceof Error ? err.message : "Não foi possível enviar a resposta.");
    } finally {
      setEnviando(false);
    }
  }

  return (
    <div className="flex flex-col gap-3 rounded-lg border border-surface-border p-4 text-sm">
      <p className="font-medium">
        {consulta.numero} · {consulta.status_rotulo || consulta.status}
      </p>
      <p>
        {consulta.nome} · {consulta.servico} · {consulta.unidade} · {PRIORIDADE[consulta.prioridade] ?? consulta.prioridade}
      </p>
      <p className="text-muted">Aberto em {fmtDateTime(consulta.criado_em)}</p>
      {consulta.prazo_em && <p className="text-muted">Prazo {fmtDateTime(consulta.prazo_em)}</p>}
      {consulta.agendamento && (
        <p>
          Atendimento {consulta.agendamento.data} às {consulta.agendamento.hora} · {consulta.agendamento.status}
        </p>
      )}
      {consulta.complemento && <p>Complementação pedida: {consulta.complemento}</p>}
      <ul className="flex flex-col gap-2">
        {(consulta.historico ?? []).map((item, indice) => (
          <li key={item.id ?? indice}>
            <p>{item.descricao}</p>
            <p className="text-xs text-muted">
              {item.status} · {fmtDateTime(item.created_at)}
            </p>
          </li>
        ))}
      </ul>
      {consulta.status === "em_complementacao" && (
        <form onSubmit={responder} className="flex flex-col gap-3 border-t border-surface-border pt-3">
          <Input id="cpf" name="cpf" label="CPF do pedido *" required inputMode="numeric" maxLength={14} />
          <Textarea id="mensagem" name="mensagem" label="Resposta *" required maxLength={1000} />
          <Input id="arquivos" name="arquivos" label="Arquivos" type="file" multiple />
          <Button type="submit" loading={enviando}>
            Enviar resposta
          </Button>
        </form>
      )}
    </div>
  );
}
