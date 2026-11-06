"use client";

import {
  NomeClicavel,
  Quadro,
  ordenarColunas,
  seloDoEstado,
  useAparencia,
  type ColunaFaixa,
  type VistaCard,
} from "@/components/plataforma/Faixas";
import { SectionTabsInline } from "@/components/plataforma/SectionTabsInline";
import { rotuloQuadro } from "@/components/plataforma/useQuadro";
import { TableCell } from "@/components/ui/Table";
import { ESTADO_TAREFA, type EstadoTarefa, type Tarefa } from "@/lib/projetos/projetos";

import { diaBrasil } from "./datas";
import { ListaTabela } from "./ListaTabela";

const CABECALHOS = [
  { id: "tarefa", rotulo: "Tarefa" },
  { id: "responsavel", rotulo: "Responsável" },
  { id: "estado", rotulo: "Estado" },
  { id: "prazo", rotulo: "Prazo" },
];

function depoisDe(tarefas: Tarefa[], id?: string) {
  if (!id) return "";
  return tarefas.find((item) => item.id === id)?.titulo || "outra tarefa";
}

export function TarefasDoProjeto({
  projetoId,
  tarefas,
  colunas,
  oficial,
  mutavel,
  pending,
  vista,
  onVista,
  onMover,
  onRenomear,
  onAbrir,
}: {
  projetoId: string;
  tarefas: Tarefa[];
  colunas: ColunaFaixa[];
  oficial: boolean;
  mutavel: boolean;
  pending: boolean;
  vista: "quadro" | "lista";
  onVista: (vista: "quadro" | "lista") => void;
  onMover: (tarefa: Tarefa, estado: EstadoTarefa) => void;
  onRenomear?: (tarefa: Tarefa, nome: string) => void;
  onAbrir: (tarefa: Tarefa) => void;
}) {
  const look = useAparencia(`nova-faixas:tarefas:${projetoId}`);

  function estadoDa(t: Tarefa) {
    const ativas = ordenarColunas(
      colunas.filter((coluna) => coluna.ativa !== false || coluna.id === t.estado),
    );
    return (
      <select
        aria-label={`Estado da tarefa ${t.titulo}`}
        value={t.estado}
        disabled={!t.responsavel_id || pending}
        onChange={(e) => onMover(t, e.target.value as EstadoTarefa)}
        className="h-8 w-full min-w-0 rounded-md border border-surface-border bg-surface px-2 text-xs text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-primary"
      >
        {ativas.map((coluna) => (
          <option key={coluna.id} value={coluna.id}>
            {rotuloQuadro(oficial, colunas, look.gravado, coluna.id)}
          </option>
        ))}
      </select>
    );
  }

  function nome(t: Tarefa, vistaCard: VistaCard, classe: string) {
    return (
      <NomeClicavel
        nome={vistaCard.titulo || t.titulo}
        onSalvar={vistaCard.salvarNome}
        className={classe}
      />
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <SectionTabsInline
        label="Forma de ver as tarefas"
        value={vista}
        onChange={onVista}
        tabs={[
          { value: "quadro", label: "Cartões" },
          { value: "lista", label: "Lista" },
        ]}
      />
      {tarefas.length === 0 && (
        <p className="text-sm text-muted">
          Nenhuma tarefa criada. Adicione a primeira tarefa para estruturar as entregas deste
          projeto.
        </p>
      )}
      {vista === "lista" ? (
        <ListaTabela
          rotulo="Lista de tarefas"
          memoria={`nova-faixas:tarefas-lista:${projetoId}`}
          aparencia={`nova-faixas:tarefas:${projetoId}`}
          colunas={colunas}
          oficial={oficial}
          itens={tarefas}
          colunaDe={(t) => t.estado}
          idDe={(t) => t.id}
          nomeDe={(t) => t.titulo}
          aoAbrir={onAbrir}
          aoRenomear={onRenomear}
          cabecalhos={CABECALHOS}
          render={(t, vistaCard) => {
            const anterior = depoisDe(tarefas, t.predecessora_id);
            return (
              <>
                <TableCell>
                  <div data-testid={`tarefa-${t.id}`}>
                    {nome(t, vistaCard, "block truncate text-left text-sm font-medium text-foreground hover:underline")}
                    {anterior && (
                      <p className="truncate text-[11px] text-muted">Depois de {anterior}</p>
                    )}
                  </div>
                </TableCell>
                <TableCell>
                  <span className="text-sm">{t.responsavel_nome || "Sem responsável"}</span>
                </TableCell>
                <TableCell>
                  {mutavel ? (
                    estadoDa(t)
                  ) : (
                    <span className="text-sm text-muted">
                      {seloDoEstado(colunas, t.estado, ESTADO_TAREFA[t.estado]).label}
                    </span>
                  )}
                </TableCell>
                <TableCell>
                  <span className="text-sm text-muted">{diaBrasil(t.data_limite) || "Sem prazo"}</span>
                </TableCell>
              </>
            );
          }}
        />
      ) : (
        <Quadro
          rotulo="Quadro de tarefas"
          memoria={`nova-faixas:tarefas:${projetoId}`}
          aparencia={`nova-faixas:tarefas:${projetoId}`}
          colunas={colunas}
          oficial={oficial}
          itens={tarefas}
          colunaDe={(t) => t.estado}
          idDe={(t) => t.id}
          nomeDe={(t) => t.titulo}
          pending={pending}
          arrastavel={(t) => mutavel && Boolean(t.responsavel_id)}
          podeSoltar={(t, coluna) =>
            Boolean(t.responsavel_id) &&
            t.estado !== coluna &&
            colunas.find((item) => item.id === coluna)?.ativa !== false
          }
          aoSoltar={(t, coluna) => onMover(t, coluna as EstadoTarefa)}
          aoAbrir={onAbrir}
          aoRenomear={onRenomear}
          render={(t, vistaCard) => {
            const anterior = depoisDe(tarefas, t.predecessora_id);
            return (
              <div className="flex flex-col gap-1.5" data-testid={`tarefa-${t.id}`}>
                {nome(
                  t,
                  vistaCard,
                  "line-clamp-2 text-left text-sm font-medium text-foreground hover:underline",
                )}
                <div className="flex items-center justify-between gap-2 text-[11px] text-muted">
                  <span className="truncate">{t.responsavel_nome || "Sem responsável"}</span>
                  <span className="shrink-0">{diaBrasil(t.data_limite) || "Sem prazo"}</span>
                </div>
                {!t.responsavel_id && (
                  <span className="text-[11px] text-muted">Atribua alguém para mover</span>
                )}
                {anterior && <span className="text-[11px] text-muted">Depois de {anterior}</span>}
                {mutavel && estadoDa(t)}
              </div>
            );
          }}
        />
      )}
    </div>
  );
}
