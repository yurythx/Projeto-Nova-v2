"use client";

import { CalendarClock } from "lucide-react";

import { PessoaComAvatar } from "@/components/plataforma/AvatarPessoa";
import { Quadro } from "@/components/plataforma/Faixas";
import { colunasVisiveis } from "@/components/plataforma/useQuadro";
import { ListaTabela } from "@/components/projetos/ListaTabela";
import { Badge } from "@/components/ui/Badge";
import { TableCell } from "@/components/ui/Table";

import { etapaLegivel, rotuloPrazo, rotuloPrioridade, tomPrazo } from "./painel";
import {
  COLUNAS_PROTOCOLO,
  diaDoPrazo,
  molduraPrazo,
  tomPrioridade,
  type PedidoResumo,
} from "./quadro";

const colunaDe = (p: PedidoResumo) => p.status;
const idDe = (p: PedidoResumo) => p.id;
const nomeDe = (p: PedidoResumo) => p.servico_nome;

function Prazo({ pedido }: { pedido: PedidoResumo }) {
  const dia = diaDoPrazo(pedido.prazo_em);
  const situacao = pedido.situacao_prazo;
  const cor =
    situacao === "atrasado"
      ? "text-danger"
      : situacao === "atencao"
        ? "text-warning"
        : "text-muted";
  return (
    <span className={`inline-flex shrink-0 items-center gap-1 ${cor}`}>
      <CalendarClock className="h-3 w-3" aria-hidden />
      {dia || "Sem prazo"}
      {situacao === "atrasado" || situacao === "atencao" ? (
        <span className="sr-only"> ({rotuloPrazo(situacao)})</span>
      ) : null}
    </span>
  );
}

/** Quadro da mesa: uma coluna por status. O pedido só muda pela ficha, com motivo. */
export function ProtocolosQuadro({
  itens,
  aoAbrir,
}: {
  itens: PedidoResumo[];
  aoAbrir: (p: PedidoResumo) => void;
}) {
  return (
    <Quadro
      rotulo="Quadro de pedidos do protocolo"
      memoria="nova-faixas:protocolos-quadro"
      aparencia="nova-faixas:protocolos"
      colunas={colunasVisiveis(COLUNAS_PROTOCOLO, itens, colunaDe)}
      itens={itens}
      colunaDe={colunaDe}
      idDe={idDe}
      nomeDe={nomeDe}
      aoAbrir={aoAbrir}
      moldura={molduraPrazo}
      testeId="protocolos-kanban"
      render={(p) => (
        <>
          <div className="flex items-center justify-between gap-2 text-xs">
            <span className="font-mono font-semibold tabular-nums text-foreground">{p.numero}</span>
            <Badge tone={tomPrioridade(p.prioridade)}>{rotuloPrioridade(p.prioridade)}</Badge>
          </div>
          <button
            type="button"
            onClick={() => aoAbrir(p)}
            className="line-clamp-2 text-left text-sm font-semibold text-foreground hover:underline"
          >
            {p.servico_nome}
          </button>
          <div className="flex items-center justify-between gap-2 text-[11px] text-muted">
            <PessoaComAvatar nome={p.nome} vazio="Sem nome" className="min-w-0" />
            <Prazo pedido={p} />
          </div>
          {(p.etapa_id || p.unidade_nome) && (
            <div className="flex flex-wrap items-center gap-1.5 text-[11px] text-muted">
              {p.etapa_id && (
                <span className="rounded-md bg-black/5 px-1.5 py-0.5 dark:bg-white/10">
                  {etapaLegivel(p.etapa_id)}
                </span>
              )}
              {p.unidade_nome && <span className="truncate">{p.unidade_nome}</span>}
            </div>
          )}
        </>
      )}
    />
  );
}

/** Lista da mesa agrupada por status, com cabeçalho em cada grupo, como a de projetos. */
export function ProtocolosLista({
  itens,
  aoAbrir,
}: {
  itens: PedidoResumo[];
  aoAbrir: (p: PedidoResumo) => void;
}) {
  return (
    <ListaTabela
      rotulo="Lista de pedidos do protocolo"
      memoria="nova-faixas:protocolos-lista"
      aparencia="nova-faixas:protocolos"
      colunas={colunasVisiveis(COLUNAS_PROTOCOLO, itens, colunaDe)}
      itens={itens}
      colunaDe={colunaDe}
      idDe={idDe}
      nomeDe={nomeDe}
      aoAbrir={aoAbrir}
      cabecalhos={[
        { id: "numero", rotulo: "Número" },
        { id: "servico", rotulo: "Serviço" },
        { id: "requerente", rotulo: "Requerente" },
        { id: "etapa", rotulo: "Etapa" },
        { id: "prioridade", rotulo: "Prioridade" },
        { id: "prazo", rotulo: "Prazo" },
      ]}
      render={(p) => (
        <>
          <TableCell>
            <span className="font-mono text-sm font-semibold text-primary">{p.numero}</span>
          </TableCell>
          <TableCell>
            <button
              type="button"
              onClick={() => aoAbrir(p)}
              className="truncate text-left text-sm font-semibold text-foreground hover:underline"
            >
              {p.servico_nome}
            </button>
            {p.unidade_nome && <p className="truncate text-xs text-muted">{p.unidade_nome}</p>}
          </TableCell>
          <TableCell>
            <PessoaComAvatar nome={p.nome} vazio="Sem nome" className="text-sm text-foreground" />
          </TableCell>
          <TableCell>
            <span className="text-sm text-muted">
              {p.etapa_id ? etapaLegivel(p.etapa_id) : "—"}
            </span>
          </TableCell>
          <TableCell>
            <Badge tone={tomPrioridade(p.prioridade)}>{rotuloPrioridade(p.prioridade)}</Badge>
          </TableCell>
          <TableCell>
            <div className="flex flex-col items-start gap-1 text-sm">
              <span className="text-muted">{diaDoPrazo(p.prazo_em) || "Sem prazo"}</span>
              {p.situacao_prazo && p.situacao_prazo !== "finalizado" && (
                <Badge tone={tomPrazo(p.situacao_prazo)}>{rotuloPrazo(p.situacao_prazo)}</Badge>
              )}
            </div>
          </TableCell>
        </>
      )}
    />
  );
}
