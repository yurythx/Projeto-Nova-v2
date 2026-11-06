"use client";

import { useMemo, useState } from "react";

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
import { Input } from "@/components/ui/Input";
import { TableCell } from "@/components/ui/Table";
import { ESTADO_TAREFA, type EstadoTarefa, type Tarefa } from "@/lib/projetos/projetos";

import { diaBrasil, diaISO, faixaDoPrazo } from "./datas";
import { ListaTabela } from "./ListaTabela";
import { ResponsavelDaTarefa, type ResponsavelOpcao } from "./ResponsavelDaTarefa";

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
  onAtribuir,
  onPrazo,
  usuarioId,
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
  onAtribuir?: (tarefa: Tarefa, pessoa: ResponsavelOpcao) => void;
  onPrazo?: (tarefa: Tarefa, dia: string) => void;
  usuarioId?: string;
}) {
  const look = useAparencia(`nova-faixas:tarefas:${projetoId}`);
  const [busca, setBusca] = useState("");
  const [minhas, setMinhas] = useState(false);
  const [atrasadas, setAtrasadas] = useState(false);
  const [semana, setSemana] = useState(false);
  const [pessoaAberta, setPessoaAberta] = useState<string | null>(null);
  const visiveis = useMemo(() => {
    const termo = busca.trim().toLocaleLowerCase("pt-BR");
    return tarefas.filter((t) => {
      if (termo && !t.titulo.toLocaleLowerCase("pt-BR").includes(termo)) return false;
      if (minhas && t.responsavel_id !== usuarioId) return false;
      const faixa = faixaDoPrazo(t.data_limite);
      if (atrasadas && faixa !== "atrasada") return false;
      if (semana && faixa !== "semana") return false;
      return true;
    });
  }, [atrasadas, busca, minhas, semana, tarefas, usuarioId]);

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
      <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-end">
        <div className="min-w-48 flex-1">
          <Input
            id={`busca-tarefa-${projetoId}`}
            label="Buscar tarefa"
            value={busca}
            onChange={(e) => setBusca(e.target.value)}
            placeholder="Nome da tarefa"
          />
        </div>
        <div className="flex flex-wrap gap-2 pb-1">
          <Filtro ligado={minhas} onClick={() => setMinhas((v) => !v)} nome="Só as minhas" />
          <Filtro ligado={atrasadas} onClick={() => setAtrasadas((v) => !v)} nome="Atrasadas" />
          <Filtro ligado={semana} onClick={() => setSemana((v) => !v)} nome="Desta semana" />
        </div>
      </div>
      <SectionTabsInline
        label="Forma de ver as tarefas"
        value={vista}
        onChange={onVista}
        tabs={[
          { value: "quadro", label: "Cartões" },
          { value: "lista", label: "Lista" },
        ]}
      />
      {tarefas.length > 0 && visiveis.length === 0 && (
        <p className="text-sm text-muted">Nenhuma tarefa com esse filtro.</p>
      )}
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
          itens={visiveis}
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
                  <CelulaPessoa
                    tarefa={t}
                    projetoId={projetoId}
                    mutavel={mutavel}
                    aberta={pessoaAberta === t.id}
                    onAbrir={() => setPessoaAberta(t.id)}
                    onAtribuir={(pessoa) => {
                      setPessoaAberta(null);
                      onAtribuir?.(t, pessoa);
                    }}
                  />
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
                  <CelulaPrazo tarefa={t} mutavel={mutavel} onPrazo={(dia) => onPrazo?.(t, dia)} />
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
          itens={visiveis}
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

function Filtro({ ligado, onClick, nome }: { ligado: boolean; onClick: () => void; nome: string }) {
  return (
    <button
      type="button"
      aria-pressed={ligado}
      onClick={onClick}
      className={`rounded-full border px-3 py-1 text-xs ${
        ligado
          ? "border-primary bg-primary/10 text-primary"
          : "border-surface-border text-muted hover:bg-surface-hover"
      }`}
    >
      {nome}
    </button>
  );
}

function CelulaPessoa({
  tarefa,
  projetoId,
  mutavel,
  aberta,
  onAbrir,
  onAtribuir,
}: {
  tarefa: Tarefa;
  projetoId: string;
  mutavel: boolean;
  aberta: boolean;
  onAbrir: () => void;
  onAtribuir: (pessoa: ResponsavelOpcao) => void;
}) {
  if (!mutavel || !aberta) {
    const nome = tarefa.responsavel_nome || "Sem responsável";
    if (!mutavel) return <span className="text-sm">{nome}</span>;
    return (
      <button type="button" className="text-left text-sm hover:underline" onClick={onAbrir}>
        {tarefa.responsavel_nome || "Atribuir"}
      </button>
    );
  }
  return (
    <ResponsavelDaTarefa
      id={`resp-${tarefa.id}`}
      projetoId={projetoId}
      value={{ id: tarefa.responsavel_id ?? "", nome: tarefa.responsavel_nome }}
      onChange={onAtribuir}
    />
  );
}

function CelulaPrazo({
  tarefa,
  mutavel,
  onPrazo,
}: {
  tarefa: Tarefa;
  mutavel: boolean;
  onPrazo: (dia: string) => void;
}) {
  if (!mutavel) {
    return <span className="text-sm text-muted">{diaBrasil(tarefa.data_limite) || "Sem prazo"}</span>;
  }
  return (
    <input
      type="date"
      aria-label={`Prazo da tarefa ${tarefa.titulo}`}
      value={diaISO(tarefa.data_limite)}
      onChange={(e) => onPrazo(e.target.value)}
      className="h-8 rounded-md border border-surface-border bg-surface px-2 text-xs text-foreground"
    />
  );
}
