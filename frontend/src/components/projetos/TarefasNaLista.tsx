"use client";

import { type FormEvent, Fragment, useState } from "react";

import { DataState } from "@/components/plataforma/DataState";
import { PessoaComAvatar } from "@/components/plataforma/AvatarPessoa";
import { useAction } from "@/components/plataforma/useAction";
import { ProgressoProjeto } from "@/components/projetos/ProgressoProjeto";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from "@/components/ui/Table";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { ESTADO_TAREFA, type Tarefa } from "@/lib/projetos/projetos";

function estadoDa(tarefa: Tarefa) {
  const estado = ESTADO_TAREFA[tarefa.estado];
  if (estado) return <Badge tone={estado.tone}>{estado.label}</Badge>;
  return <span className="text-sm text-muted">{tarefa.estado}</span>;
}

/** Tarefas do projeto abertas na linha da lista, com subtarefas, estado e responsável. */
export function TarefasNaLista({
  projetoId,
  titulo,
  podeCriar = false,
}: {
  projetoId: string;
  titulo: string;
  podeCriar?: boolean;
}) {
  const tarefas = useApiQuery<Tarefa[]>(`v1/projetos/${projetoId}/tarefas`);
  const { run, pending } = useAction();
  const [titulos, setTitulos] = useState<Record<string, string>>({});
  const lista = tarefas.data ?? [];
  const raizes = lista.filter((tarefa) => !tarefa.pai_id);

  async function criar(e: FormEvent, pai: string, ordem: number) {
    e.preventDefault();
    const nome = (titulos[pai] ?? "").trim();
    if (!nome) return;
    const ok = await run(
      () =>
        apiClient.post(`v1/projetos/${projetoId}/tarefas`, {
          titulo: nome,
          pai_id: pai,
          estado: "a_fazer",
          ordem,
        }),
      "Subtarefa adicionada",
    );
    if (ok) {
      setTitulos((atual) => ({ ...atual, [pai]: "" }));
      void tarefas.mutate();
    }
  }

  return (
    <div className="px-4 py-3 pl-12">
      <DataState
        loading={tarefas.isLoading}
        error={tarefas.error}
        empty={false}
        onRetry={() => void tarefas.mutate()}
        rows={2}
      >
        {raizes.length === 0 ? (
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
              {raizes.map((tarefa) => {
                const filhas = lista.filter((item) => item.pai_id === tarefa.id);
                const feitas = filhas.filter((item) => item.estado === "concluida").length;
                return (
                  <Fragment key={tarefa.id}>
                    <TableRow>
                      <TableCell>
                        <div className="flex flex-col gap-1">
                          <span className="text-sm text-foreground">{tarefa.titulo}</span>
                          {filhas.length > 0 && (
                            <ProgressoProjeto
                              titulo={tarefa.titulo}
                              total={filhas.length}
                              concluidas={feitas}
                              unidade="subtarefas"
                            />
                          )}
                        </div>
                      </TableCell>
                      <TableCell>{estadoDa(tarefa)}</TableCell>
                      <TableCell>
                        <PessoaComAvatar
                          id={tarefa.responsavel_id}
                          nome={tarefa.responsavel_nome}
                          className="text-sm"
                        />
                      </TableCell>
                    </TableRow>
                    {filhas.map((filha) => (
                      <TableRow key={filha.id}>
                        <TableCell>
                          <span className="pl-6 text-sm text-foreground">{filha.titulo}</span>
                        </TableCell>
                        <TableCell>{estadoDa(filha)}</TableCell>
                        <TableCell>
                          <PessoaComAvatar
                            id={filha.responsavel_id}
                            nome={filha.responsavel_nome}
                            className="text-sm"
                          />
                        </TableCell>
                      </TableRow>
                    ))}
                    {podeCriar && (
                      <TableRow>
                        <TableCell colSpan={3}>
                          <form
                            onSubmit={(e) => void criar(e, tarefa.id, filhas.length + 1)}
                            className="flex items-center gap-2 pl-6"
                          >
                            <Input
                              id={`subtarefa-${tarefa.id}`}
                              value={titulos[tarefa.id] ?? ""}
                              onChange={(e) =>
                                setTitulos((atual) => ({ ...atual, [tarefa.id]: e.target.value }))
                              }
                              placeholder="Nova subtarefa"
                              aria-label={`Subtarefa de ${tarefa.titulo}`}
                              maxLength={160}
                              className="h-8"
                            />
                            <Button type="submit" size="sm" variant="secondary" disabled={pending}>
                              Adicionar
                            </Button>
                          </form>
                        </TableCell>
                      </TableRow>
                    )}
                  </Fragment>
                );
              })}
            </TableBody>
          </Table>
        )}
      </DataState>
    </div>
  );
}
