"use client";

import Link from "next/link";
import { useMemo, useState } from "react";

import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDateTime } from "@/components/plataforma/useAction";
import {
  etapaLegivel,
  filtrarPedidos,
  paginar,
  PRAZOS,
  PRIORIDADES,
  rotuloPrazo,
  rotuloPrioridade,
  rotuloStatus,
  STATUS,
  tomPrazo,
} from "@/components/protocolos/painel";
import { Badge } from "@/components/ui/Badge";
import { Button, buttonClass } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { useApiQuery } from "@/lib/api/swr";

interface Pedido {
  id: string;
  numero: string;
  nome: string;
  cpf?: string;
  servico_id: string;
  servico_nome: string;
  unidade_nome?: string;
  status: string;
  situacao_prazo?: string;
  etapa_id: string;
  prioridade: string;
  created_at: string;
}

type Visao = "lista" | "kanban";

export default function ProtocolosPage() {
  const lista = useApiQuery<Pedido[]>("v1/protocolos/pedidos");
  const pedidos = useMemo(() => lista.data ?? [], [lista.data]);
  const [visao, setVisao] = useState<Visao>("lista");
  const [prazo, setPrazo] = useState("todos");
  const [busca, setBusca] = useState("");
  const [servico, setServico] = useState("");
  const [status, setStatus] = useState("");
  const [prioridade, setPrioridade] = useState("");
  const [pagina, setPagina] = useState(1);

  const servicos = useMemo(() => {
    const mapa = new Map<string, string>();
    for (const pedido of pedidos) mapa.set(pedido.servico_id, pedido.servico_nome);
    return [...mapa.entries()]
      .map(([value, label]) => ({ value, label }))
      .sort((a, b) => a.label.localeCompare(b.label, "pt-BR"));
  }, [pedidos]);

  const contagemPrazo = useMemo(() => {
    const conta: Record<string, number> = {
      todos: pedidos.length,
      no_prazo: 0,
      atencao: 0,
      atrasado: 0,
      finalizado: 0,
    };
    for (const pedido of pedidos) {
      const chave = pedido.situacao_prazo || "no_prazo";
      conta[chave] = (conta[chave] ?? 0) + 1;
    }
    return conta;
  }, [pedidos]);

  const porStatus = useMemo(() => contar(pedidos, (pedido) => pedido.status), [pedidos]);
  const porPrioridade = useMemo(() => contar(pedidos, (pedido) => pedido.prioridade), [pedidos]);
  const porMes = useMemo(() => {
    const mapa = new Map<string, number>();
    for (const pedido of pedidos) {
      const data = new Date(pedido.created_at);
      const chave = Number.isNaN(data.getTime())
        ? "sem data"
        : data.toLocaleDateString("pt-BR", { month: "short", year: "numeric" });
      mapa.set(chave, (mapa.get(chave) ?? 0) + 1);
    }
    return [...mapa.entries()];
  }, [pedidos]);

  const visiveis = useMemo(
    () => filtrarPedidos(pedidos, { prazo, busca, servico, status, prioridade }),
    [pedidos, prazo, busca, servico, status, prioridade],
  );
  const listaPaginada = useMemo(() => paginar(visiveis, pagina, 20), [visiveis, pagina]);

  const complementos = porStatus.find((item) => item.chave === "em_complementacao")?.total ?? 0;

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Protocolos"
        title="Painel"
        description="Pedidos da carta que a sua unidade pode tratar, com o prazo e o status."
        actions={
          <div className="flex flex-wrap gap-2">
            <Link href="/protocolos/balcao" className={buttonClass("secondary", "sm")}>
              Balcão
            </Link>
            <Link href="/protocolos/formularios" className={buttonClass("secondary", "sm")}>
              Formulários
            </Link>
            <Link href="/protocolos/servicos" className={buttonClass("secondary", "sm")}>
              Serviços
            </Link>
            <Link href="/protocolos/responsaveis" className={buttonClass("secondary", "sm")}>
              Responsáveis
            </Link>
            <Link href="/carta" className={buttonClass("secondary", "sm")}>
              Carta pública
            </Link>
          </div>
        }
      />
      <DataState
        loading={lista.isLoading}
        error={lista.error}
        onRetry={() => void lista.mutate()}
        empty={false}
      >
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
          <Resumo titulo="Total" valor={pedidos.length} />
          {STATUS.map((item) => (
            <button
              key={item.value}
              type="button"
              className="text-left"
              onClick={() => {
                setStatus(status === item.value ? "" : item.value);
                setPagina(1);
              }}
            >
              <Resumo
                titulo={item.label}
                valor={porStatus.find((linha) => linha.chave === item.value)?.total ?? 0}
                ativo={status === item.value}
              />
            </button>
          ))}
        </div>

        {complementos > 0 && (
          <button
            type="button"
            className="rounded-xl border border-warning/40 bg-warning/10 px-4 py-3 text-left text-sm"
            onClick={() => {
              setStatus("em_complementacao");
              setPagina(1);
            }}
          >
            {complementos} pedido(s) aguardando complementação. Ver esses pedidos.
          </button>
        )}

        <div className="grid gap-4 lg:grid-cols-3">
          <Barras
            titulo="Por status"
            linhas={porStatus.map((item) => ({
              nome: rotuloStatus(item.chave),
              total: item.total,
            }))}
          />
          <Barras
            titulo="Por prioridade"
            linhas={porPrioridade.map((item) => ({
              nome: rotuloPrioridade(item.chave),
              total: item.total,
            }))}
          />
          <Barras titulo="Por mês" linhas={porMes.map(([nome, total]) => ({ nome, total }))} />
        </div>

        <div className="flex flex-wrap gap-2">
          {PRAZOS.map((item) => (
            <button
              key={item.value}
              type="button"
              aria-pressed={prazo === item.value}
              className={buttonClass(prazo === item.value ? "primary" : "secondary", "sm")}
              onClick={() => {
                setPrazo(item.value);
                setPagina(1);
              }}
            >
              {item.label} {contagemPrazo[item.value] ?? 0}
            </button>
          ))}
        </div>

        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <Input
            id="busca-protocolo"
            label="Buscar"
            placeholder="Número, nome ou CPF"
            value={busca}
            onChange={(e) => {
              setBusca(e.target.value);
              setPagina(1);
            }}
          />
          <Select
            id="filtro-servico"
            label="Serviço"
            placeholder="Todos os serviços"
            value={servico}
            onChange={(e) => {
              setServico(e.target.value);
              setPagina(1);
            }}
            options={servicos}
          />
          <Select
            id="filtro-status"
            label="Status"
            placeholder="Todos os status"
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPagina(1);
            }}
            options={STATUS}
          />
          <Select
            id="filtro-prioridade"
            label="Prioridade"
            placeholder="Todas as prioridades"
            value={prioridade}
            onChange={(e) => {
              setPrioridade(e.target.value);
              setPagina(1);
            }}
            options={PRIORIDADES}
          />
        </div>

        <div className="flex gap-2">
          <button
            type="button"
            aria-pressed={visao === "lista"}
            className={buttonClass(visao === "lista" ? "primary" : "secondary", "sm")}
            onClick={() => setVisao("lista")}
          >
            Lista
          </button>
          <button
            type="button"
            aria-pressed={visao === "kanban"}
            className={buttonClass(visao === "kanban" ? "primary" : "secondary", "sm")}
            onClick={() => setVisao("kanban")}
          >
            Kanban
          </button>
        </div>

        {visiveis.length === 0 ? (
          <p className="text-sm text-muted">Nenhum pedido com esses filtros.</p>
        ) : visao === "kanban" ? (
          <div className="flex gap-3 overflow-x-auto pb-2">
            {STATUS.filter((item) => visiveis.some((pedido) => pedido.status === item.value)).map(
              (item) => (
                <section key={item.value} className="flex w-72 shrink-0 flex-col gap-2">
                  <h2 className="text-sm font-semibold">
                    {item.label}{" "}
                    <span className="text-muted">
                      {visiveis.filter((pedido) => pedido.status === item.value).length}
                    </span>
                  </h2>
                  {visiveis
                    .filter((pedido) => pedido.status === item.value)
                    .map((pedido) => (
                      <Cartao key={pedido.id} pedido={pedido} />
                    ))}
                </section>
              ),
            )}
          </div>
        ) : (
          <>
            <ul className="flex flex-col gap-3">
              {listaPaginada.itens.map((pedido) => (
                <li key={pedido.id}>
                  <Cartao pedido={pedido} />
                </li>
              ))}
            </ul>
            {listaPaginada.paginas > 1 && (
              <div className="flex items-center justify-between gap-3">
                <p className="text-sm text-muted">
                  Página {listaPaginada.pagina} de {listaPaginada.paginas} · {listaPaginada.total}{" "}
                  pedidos
                </p>
                <div className="flex gap-2">
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    disabled={listaPaginada.pagina <= 1}
                    onClick={() => setPagina((atual) => atual - 1)}
                  >
                    Anterior
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    disabled={listaPaginada.pagina >= listaPaginada.paginas}
                    onClick={() => setPagina((atual) => atual + 1)}
                  >
                    Próxima
                  </Button>
                </div>
              </div>
            )}
          </>
        )}
      </DataState>
    </div>
  );
}

function Resumo({ titulo, valor, ativo }: { titulo: string; valor: number; ativo?: boolean }) {
  return (
    <div
      className={`rounded-xl border px-4 py-3 ${ativo ? "border-primary bg-primary/5" : "border-surface-border bg-surface"}`}
    >
      <p className="text-xs text-muted">{titulo}</p>
      <p className="text-2xl font-semibold">{valor}</p>
    </div>
  );
}

function Barras({ titulo, linhas }: { titulo: string; linhas: { nome: string; total: number }[] }) {
  const maior = Math.max(1, ...linhas.map((linha) => linha.total));
  return (
    <section className="rounded-xl border border-surface-border bg-surface p-4">
      <h2 className="mb-3 text-sm font-semibold">{titulo}</h2>
      {linhas.length === 0 ? (
        <p className="text-sm text-muted">Sem pedidos.</p>
      ) : (
        <ul className="flex flex-col gap-2">
          {linhas.map((linha) => (
            <li key={linha.nome}>
              <div className="mb-1 flex justify-between text-xs">
                <span>{linha.nome}</span>
                <span>{linha.total}</span>
              </div>
              <div className="h-2 rounded-full bg-black/5 dark:bg-white/10">
                <div
                  className="h-2 rounded-full bg-primary"
                  style={{ width: `${Math.round((linha.total / maior) * 100)}%` }}
                />
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function Cartao({ pedido }: { pedido: Pedido }) {
  return (
    <Link
      href={`/protocolos/${pedido.id}`}
      className="block rounded-xl border border-surface-border bg-surface p-4"
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="font-medium">{pedido.numero}</p>
          <p className="text-sm">{pedido.servico_nome}</p>
          <p className="text-sm text-muted">
            Etapa: {etapaLegivel(pedido.etapa_id)} · {pedido.nome} ·{" "}
            {pedido.unidade_nome || "Unidade"} · {fmtDateTime(pedido.created_at)}
          </p>
        </div>
        <div className="flex flex-col items-end gap-1">
          <Badge>{rotuloStatus(pedido.status)}</Badge>
          {pedido.situacao_prazo && (
            <Badge tone={tomPrazo(pedido.situacao_prazo)}>
              {rotuloPrazo(pedido.situacao_prazo)}
            </Badge>
          )}
          <Badge>{rotuloPrioridade(pedido.prioridade)}</Badge>
        </div>
      </div>
    </Link>
  );
}

function contar(pedidos: Pedido[], chave: (pedido: Pedido) => string) {
  const mapa = new Map<string, number>();
  for (const pedido of pedidos) {
    const valor = chave(pedido) || "—";
    mapa.set(valor, (mapa.get(valor) ?? 0) + 1);
  }
  return [...mapa.entries()].map(([chaveItem, total]) => ({ chave: chaveItem, total }));
}
