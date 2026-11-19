"use client";

import Link from "next/link";
import { useEffect, useState, type FormEvent } from "react";

import {
  CampoInput,
  oculto,
  respostasVisiveis,
  type Campo,
} from "@/components/protocolos/CamposDoPedido";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Button, buttonClass } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { ApiError, apiClient } from "@/lib/api/client";
import { publicFetch, publicPost } from "@/lib/api/publicClient";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { UploadTicket } from "@/lib/plataforma/types";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";

interface Servico {
  id: string;
  nome: string;
  categoria: string;
}

interface CartaItem {
  servico: {
    id: string;
    nome: string;
    exige_agendamento: boolean;
  };
  formulario: { titulo: string; campos: Campo[] };
}

interface Pedido {
  id: string;
  numero: string;
}

export default function BalcaoPage() {
  const { can } = usePlataforma();
  const { run, pending } = useAction();
  const [servicos, setServicos] = useState<Servico[]>([]);
  const [item, setItem] = useState<CartaItem | null>(null);
  const [servicoId, setServicoId] = useState("");
  const [servicoCarregado, setServicoCarregado] = useState("");
  const [respostas, setRespostas] = useState<Record<string, unknown>>({});
  const [pedido, setPedido] = useState<Pedido | null>(null);
  const [erro, setErro] = useState("");
  const [subindo, setSubindo] = useState(false);

  useEffect(() => {
    let vivo = true;
    publicFetch<Servico[]>("v1/protocolos/carta")
      .then((lista) => {
        if (vivo) setServicos(lista ?? []);
      })
      .catch((e: unknown) => {
        if (vivo) setErro(e instanceof ApiError ? e.message : "A carta não pôde ser carregada.");
      });
    return () => {
      vivo = false;
    };
  }, []);

  if (servicoCarregado !== servicoId) {
    setServicoCarregado(servicoId);
    setItem(null);
    setRespostas({});
  }

  useEffect(() => {
    if (!servicoId) return;
    let vivo = true;
    publicFetch<CartaItem>(`v1/protocolos/carta/${servicoId}`)
      .then((carta) => {
        if (vivo) setItem(carta);
      })
      .catch((e: unknown) => {
        if (vivo) setErro(e instanceof ApiError ? e.message : "Serviço indisponível.");
      });
    return () => {
      vivo = false;
    };
  }, [servicoId]);

  async function enviarArquivo(campo: string, arquivo: File) {
    setSubindo(true);
    setErro("");
    try {
      const ticket = await publicPost<UploadTicket>(`v1/protocolos/carta/${servicoId}/arquivo`, {
        nome: arquivo.name,
        content_type: mimeOf(arquivo),
      });
      await putToTicket(ticket, arquivo);
      setRespostas((atual) => ({ ...atual, [campo]: ticket.object_key }));
    } catch (err) {
      setErro(
        err instanceof ApiError || err instanceof Error
          ? err.message
          : "Não foi possível enviar o arquivo.",
      );
    } finally {
      setSubindo(false);
    }
  }

  async function abrir(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!item) return;
    setErro("");
    const dados = new FormData(e.currentTarget);
    const criado = await run(
      () =>
        apiClient.post<Pedido>("v1/protocolos/balcao", {
          servico_id: item.servico.id,
          nome: String(dados.get("nome") ?? "").trim(),
          cpf: String(dados.get("cpf") ?? "").trim(),
          email: String(dados.get("email") ?? "").trim(),
          telefone: String(dados.get("telefone") ?? "").trim(),
          prioridade: String(dados.get("prioridade") ?? "normal"),
          data: String(dados.get("data") ?? ""),
          hora: String(dados.get("hora") ?? ""),
          respostas: respostasVisiveis(item.formulario.campos ?? [], respostas),
        }),
      "Pedido aberto no balcão",
    );
    if (criado) setPedido(criado.data);
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Protocolos"
        title="Balcão"
        description="Abre o pedido em nome do cidadão. A sessão precisa de protocolos:abrir na unidade do serviço."
        actions={
          <Link href="/protocolos" className={buttonClass("secondary", "sm")}>
            Mesa
          </Link>
        }
      />
      {erro && <p className="text-sm text-danger">{erro}</p>}
      {pedido && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Pedido {pedido.numero}</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col gap-3 pb-5 text-sm">
            <p>O número fica com o cidadão. A ficha da mesa mostra CPF, respostas e o arquivo.</p>
            <Link href={`/protocolos/${pedido.id}`} className={buttonClass("secondary", "sm")}>
              Abrir a ficha
            </Link>
          </CardContent>
        </Card>
      )}
      {can("protocolos:abrir") && !pedido && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Novo pedido</CardTitle>
          </CardHeader>
          <CardContent className="pb-5">
            <form onSubmit={(e) => void abrir(e)} className="flex flex-col gap-3">
              <Select
                id="servico"
                label="Serviço *"
                required
                placeholder="Selecione…"
                value={servicoId}
                options={servicos.map((servico) => ({
                  value: servico.id,
                  label: servico.categoria
                    ? `${servico.nome} · ${servico.categoria}`
                    : servico.nome,
                }))}
                onChange={(e) => setServicoId(e.target.value)}
              />
              {item && (
                <>
                  <Input
                    id="nome"
                    name="nome"
                    label="Nome completo *"
                    required
                    minLength={2}
                    maxLength={200}
                  />
                  <Input
                    id="cpf"
                    name="cpf"
                    label="CPF *"
                    required
                    inputMode="numeric"
                    maxLength={14}
                  />
                  <Input id="email" name="email" label="E-mail" type="email" maxLength={200} />
                  <Input id="telefone" name="telefone" label="Telefone" maxLength={20} />
                  <Select
                    id="prioridade"
                    name="prioridade"
                    label="Prioridade"
                    defaultValue="normal"
                    options={[
                      { value: "baixa", label: "Baixa" },
                      { value: "normal", label: "Normal" },
                      { value: "alta", label: "Alta" },
                      { value: "urgente", label: "Urgente" },
                    ]}
                  />
                  {item.servico.exige_agendamento && (
                    <>
                      <Input id="data" name="data" label="Data do atendimento" type="date" />
                      <Input id="hora" name="hora" label="Hora" type="time" />
                    </>
                  )}
                  {(item.formulario.campos ?? []).map((campo) =>
                    oculto(item.formulario.campos, respostas, campo.id) ? null : (
                      <CampoInput
                        key={campo.id}
                        campo={campo}
                        valor={respostas[campo.id]}
                        onChange={(valor) =>
                          setRespostas((atual) => ({ ...atual, [campo.id]: valor }))
                        }
                        onArquivo={(arquivo) => void enviarArquivo(campo.id, arquivo)}
                      />
                    ),
                  )}
                  <Button type="submit" loading={pending || subindo}>
                    Protocolar
                  </Button>
                </>
              )}
            </form>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
