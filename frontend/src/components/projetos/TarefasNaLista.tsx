"use client";

import { DataState } from "@/components/plataforma/DataState";
import { PessoaComAvatar } from "@/components/plataforma/AvatarPessoa";
import { Badge } from "@/components/ui/Badge";
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from "@/components/ui/Table";
import { useApiQuery } from "@/lib/api/swr";
import { ESTADO_TAREFA, type Tarefa } from "@/lib/projetos/projetos";

/** Tarefas do projeto abertas na linha da lista, com estado e responsável. */
export function TarefasNaLista({ projetoId, titulo }: { projetoId: string; titulo: string }) {
  const tarefas = useApiQuery<Tarefa[]>(`v1/projetos/${projetoId}/tarefas`);
  const lista = tarefas.data ?? [];

  return (
    <div className="px-4 py-3 pl-12">
      <DataState
        loading={tarefas.isLoading}
        error={tarefas.error}
        empty={false}
        onRetry={() => void tarefas.mutate()}
        rows={2}
      >
        {lista.length === 0 ? (
          <p className="text-sm text-muted">Nenhuma tarefa neste projeto.</p>
        ) : (
          <Table caption={`Tarefas de ${titulo}`} className="min-w-0">
            <TableHead>
              <TableRow>
                <TableHeaderCell>Tarefa</TableHeaderCell>
                <TableHeaderCell>Estado</TableHeaderCell>
                <TableHeaderCell>Responsável</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {lista.map((tarefa) => {
                const estado = ESTADO_TAREFA[tarefa.estado];
                return (
                  <TableRow key={tarefa.id}>
                    <TableCell>
                      <span className="text-sm text-foreground">{tarefa.titulo}</span>
                    </TableCell>
                    <TableCell>
                      {estado ? (
                        <Badge tone={estado.tone}>{estado.label}</Badge>
                      ) : (
                        <span className="text-sm text-muted">{tarefa.estado}</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <PessoaComAvatar
                        id={tarefa.responsavel_id}
                        nome={tarefa.responsavel_nome}
                        className="text-sm"
                      />
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </DataState>
    </div>
  );
}
