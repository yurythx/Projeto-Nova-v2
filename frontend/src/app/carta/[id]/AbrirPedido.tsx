"use client";

import Link from "next/link";
import { useEffect, useState, type FormEvent } from "react";

import { CampoInput, oculto, respostasVisiveis, type Campo } from "@/components/protocolos/CamposDoPedido";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { ApiError } from "@/lib/api/client";
import { publicFetch, publicPost } from "@/lib/api/publicClient";
import { reais } from "@/lib/moeda";
import type { UploadTicket } from "@/lib/plataforma/types";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";

interface Orientacao {
  titulo: string;
  texto: string;
  obrigatorio: boolean;
}

interface CartaItem {
  servico: {
    id: string;
    nome: string;
    descricao: string;
    exige_agendamento: boolean;
    prazo_dias: number;
    taxa_centavos: number;
    requisitos?: Orientacao[];
    documentos?: Orientacao[];
  };
  formulario: { titulo: string; instrucao: string; campos: Campo[] };
}

export function AbrirPedido({ id }: { id: string }) {
  const [item, setItem] = useState<CartaItem | null>(null);
  const [erro, setErro] = useState("");
  const [numero, setNumero] = useState("");
  const [respostas, setRespostas] = useState<Record<string, unknown>>({});
  const [pedir, setPedir] = useState(false);
  const [enviando, setEnviando] = useState(false);
  const [subindo, setSubindo] = useState(false);

  useEffect(() => {
    let vivo = true;
    publicFetch<CartaItem>(`v1/protocolos/carta/${id}`)
      .then((carta) => {
        if (vivo) setItem(carta);
      })
      .catch((e: unknown) => {
        if (vivo) setErro(e instanceof ApiError ? e.message : "Serviço indisponível.");
      });
    return () => {
      vivo = false;
    };
  }, [id]);

  async function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!item) return;
    setEnviando(true);
    setErro("");
    const dados = new FormData(e.currentTarget);
    const preenchido = respostasVisiveis(item.formulario.campos ?? [], respostas);
    try {
      const pedido = await publicPost<{ numero: string }>(`v1/protocolos/carta/${id}/pedido`, {
        servico_id: item.servico.id,
        nome: String(dados.get("nome") ?? "").trim(),
        cpf: String(dados.get("cpf") ?? "").trim(),
        email: String(dados.get("email") ?? "").trim(),
        telefone: String(dados.get("telefone") ?? "").trim(),
        data: String(dados.get("data") ?? ""),
        hora: String(dados.get("hora") ?? ""),
        respostas: preenchido,
      });
      setNumero(pedido.numero);
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : "Não foi possível abrir o pedido.");
    } finally {
      setEnviando(false);
    }
  }

  async function enviarArquivo(campo: string, arquivo: File) {
    setSubindo(true);
    setErro("");
    try {
      const ticket = await publicPost<UploadTicket>(`v1/protocolos/carta/${id}/arquivo`, {
        nome: arquivo.name,
        content_type: mimeOf(arquivo),
      });
      await putToTicket(ticket, arquivo);
      setRespostas((atual) => ({ ...atual, [campo]: ticket.object_key }));
    } catch (err) {
      setErro(err instanceof ApiError || err instanceof Error ? err.message : "Não foi possível enviar o arquivo.");
    } finally {
      setSubindo(false);
    }
  }

  if (numero) {
    return (
      <div className="mx-auto flex w-full max-w-xl flex-col gap-4 px-6 py-14">
        <h1 className="text-2xl font-bold">Pedido {numero}</h1>
        <p className="text-sm text-muted">Guarde este número. A consulta pública não mostra CPF nem as respostas.</p>
        <Link href="/carta" className={buttonClass("secondary", "sm")}>
          Voltar à carta
        </Link>
      </div>
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-xl flex-col gap-6 px-6 py-14">
      <h1 className="text-2xl font-bold">{item?.servico.nome ?? "Abrir pedido"}</h1>
      {item?.servico.descricao && <p className="text-sm text-muted">{item.servico.descricao}</p>}
      {item?.formulario.instrucao && <p className="text-sm text-muted">{item.formulario.instrucao}</p>}
      {item && <p className="text-sm">Prazo de {item.servico.prazo_dias} dias úteis.</p>}
      {item && item.servico.taxa_centavos > 0 && <p className="text-sm">Taxa {reais(item.servico.taxa_centavos)}</p>}
      {item && <Lista titulo="Requisitos" itens={item.servico.requisitos} />}
      {item && <Lista titulo="Documentos" itens={item.servico.documentos} />}
      {erro && <p className="text-sm text-danger">{erro}</p>}
      {item && !pedir && (
        <Button type="button" onClick={() => setPedir(true)}>
          Solicitar
        </Button>
      )}
      {item && pedir && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">{item.formulario.titulo}</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <form onSubmit={enviar} className="flex flex-col gap-3">
              <Input id="nome" name="nome" label="Nome completo *" required minLength={2} maxLength={200} />
              <Input id="cpf" name="cpf" label="CPF *" required inputMode="numeric" maxLength={14} />
              <Input id="email" name="email" label="E-mail" type="email" maxLength={200} />
              <Input id="telefone" name="telefone" label="Telefone" maxLength={20} />
              {item.servico.exige_agendamento && (
                <>
                  <Input id="data" name="data" label="Data do atendimento *" type="date" required />
                  <Input id="hora" name="hora" label="Hora *" type="time" required />
                </>
              )}
              {(item.formulario.campos ?? []).map((campo) =>
                oculto(item.formulario.campos, respostas, campo.id) ? null : (
                  <CampoInput
                    key={campo.id}
                    campo={campo}
                    valor={respostas[campo.id]}
                    onChange={(valor) => setRespostas((atual) => ({ ...atual, [campo.id]: valor }))}
                    onArquivo={(arquivo) => void enviarArquivo(campo.id, arquivo)}
                  />
                ),
              )}
              <div className="flex flex-wrap gap-3">
                <Button type="button" variant="secondary" onClick={() => setPedir(false)}>
                  Voltar aos detalhes
                </Button>
                <Button type="submit" loading={enviando || subindo}>
                  Protocolar
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}
    </div>
  );
}

function Lista({ titulo, itens }: { titulo: string; itens?: Orientacao[] }) {
  if (!itens || itens.length === 0) return null;
  return (
    <div className="flex flex-col gap-2">
      <h2 className="text-sm font-semibold">{titulo}</h2>
      <ul className="flex flex-col gap-2 text-sm">
        {itens.map((item) => (
          <li key={item.titulo}>
            <span className="font-medium">{item.titulo}</span>
            {item.obrigatorio ? " (obrigatório)" : ""}
            {item.texto && <span className="text-muted"> — {item.texto}</span>}
          </li>
        ))}
      </ul>
    </div>
  );
}
