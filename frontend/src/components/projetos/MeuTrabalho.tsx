"use client";

import Link from "next/link";

import { PessoaComAvatar } from "@/components/plataforma/AvatarPessoa";
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from "@/components/ui/Table";
import type { CargaPessoa, TarefaMinha } from "@/lib/projetos/projetos";

import { diaBrasil, faixaDoPrazo, type FaixaPrazo } from "./datas";

const GRUPOS: { id: FaixaPrazo; titulo: string }[] = [
  { id: "atrasada", titulo: "Atrasadas" },
  { id: "semana", titulo: "Desta semana" },
  { id: "sem", titulo: "Sem prazo" },
  { id: "depois", titulo: "Depois desta semana" },
];

export function MeuTrabalho({ tarefas }: { tarefas: TarefaMinha[] }) {
  if (tarefas.length === 0) {
    return <p className="text-sm text-muted">Nenhuma tarefa aberta com você como responsável.</p>;
  }
  return (
    <div className="flex flex-col gap-4">
      {GRUPOS.map((grupo) => {
        const itens = tarefas.filter((t) => faixaDoPrazo(t.data_limite) === grupo.id);
        if (itens.length === 0) return null;
        return (
          <section key={grupo.id} className="flex flex-col gap-2">
            <h2 className="text-sm font-medium text-foreground">
              {grupo.titulo} <span className="text-muted">({itens.length})</span>
            </h2>
            <Table caption={grupo.titulo}>
              <TableHead>
                <TableRow>
                  <TableHeaderCell>Tarefa</TableHeaderCell>
                  <TableHeaderCell>Projeto</TableHeaderCell>
                  <TableHeaderCell>Prazo</TableHeaderCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {itens.map((t) => (
                  <TableRow key={t.id}>
                    <TableCell>
                      <Link href={`/projetos/${t.projeto_id}`} className="text-sm font-medium hover:underline">
                        {t.pai_id ? `Subtarefa · ${t.titulo}` : t.titulo}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm text-muted">
                        #{t.projeto_numero} {t.projeto_titulo}
                      </span>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm text-muted">{diaBrasil(t.data_limite) || "Sem prazo"}</span>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </section>
        );
      })}
    </div>
  );
}

export function CargaEquipe({ pessoas }: { pessoas: CargaPessoa[] }) {
  if (pessoas.length === 0) {
    return <p className="text-sm text-muted">Nenhuma tarefa aberta com responsável.</p>;
  }
  return (
    <Table caption="Carga de cada responsável">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Pessoa</TableHeaderCell>
          <TableHeaderCell>Abertas</TableHeaderCell>
          <TableHeaderCell>Atrasadas</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {pessoas.map((pessoa) => (
          <TableRow key={pessoa.id}>
            <TableCell>
              <PessoaComAvatar id={pessoa.id} nome={pessoa.nome} className="text-sm" />
            </TableCell>
            <TableCell>
              <span className="text-sm">{pessoa.abertas}</span>
            </TableCell>
            <TableCell>
              <span className="text-sm">{pessoa.atrasadas}</span>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}
