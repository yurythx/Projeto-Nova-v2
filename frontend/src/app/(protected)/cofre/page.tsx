"use client";

import { type FormEvent, useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

interface Segredo {
  id: string;
  nome: string;
  usuario: string;
  unidade_id?: string;
  versao: number;
}

interface Revelado extends Segredo {
  senha: string;
  notas: string;
}

/** A lista nunca pede a senha. Revelar é um pedido à parte e fica auditado. */
export default function CofrePage() {
  const { can } = usePlataforma();
  const ve = can("cofre:read") || can("cofre:reveal") || can("cofre:manage") || can("cofre:grant");
  const gere = can("cofre:manage");
  const revela = can("cofre:reveal") || can("cofre:manage");
  const concede = can("cofre:grant");
  const lista = useApiQuery<Segredo[]>(ve ? "v1/cofre/segredos" : null);
  const { run, pending } = useAction();
  const [abertos, setAbertos] = useState<Record<string, { senha: string; notas: string }>>({});
  const [pedindo, setPedindo] = useState<string | null>(null);

  if (!ve) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Cofre" title="Cofre de senhas" />
        <EmptyState title="Seu perfil não abre o cofre" description="Peça o acesso à unidade dona do segredo." />
      </div>
    );
  }

  async function guardar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const unidade = String(fd.get("unidade_id") ?? "");
    const ok = await run(async () => {
      await apiClient.post("v1/cofre/segredos", {
        nome: String(fd.get("nome") ?? ""),
        usuario: String(fd.get("usuario") ?? ""),
        senha: String(fd.get("senha") ?? ""),
        notas: String(fd.get("notas") ?? ""),
        unidade_id: unidade || null,
      });
      form.reset();
      await lista.mutate();
    }, "Segredo guardado");
    if (ok === undefined) return;
  }

  async function revelar(e: FormEvent<HTMLFormElement>, segredo: Segredo) {
    e.preventDefault();
    const form = e.currentTarget;
    const senhaDeAcesso = String(new FormData(form).get("senha_de_acesso") ?? "");
    const item = await run(
      () =>
        apiClient
          .post<Revelado>(`v1/cofre/segredos/${segredo.id}/revelar`, { senha_de_acesso: senhaDeAcesso })
          .then((res) => res.data),
      "Senha revelada. A consulta ficou na auditoria.",
    );
    form.reset();
    if (!item) return;
    setPedindo(null);
    setAbertos((atual) => ({ ...atual, [segredo.id]: { senha: item.senha, notas: item.notas } }));
  }

  async function trocar(e: FormEvent<HTMLFormElement>, segredo: Segredo) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(async () => {
      await apiClient.put(`v1/cofre/segredos/${segredo.id}`, {
        nome: segredo.nome,
        usuario: segredo.usuario,
        senha: String(fd.get("senha") ?? ""),
      });
      form.reset();
      setAbertos((atual) => {
        const copia = { ...atual };
        delete copia[segredo.id];
        return copia;
      });
      await lista.mutate();
    }, "Senha trocada");
    if (ok === undefined) return;
  }

  async function mover(e: FormEvent<HTMLFormElement>, segredo: Segredo) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const unidade = String(fd.get("unidade_id") ?? "");
    await run(async () => {
      await apiClient.put(`v1/cofre/segredos/${segredo.id}`, {
        nome: segredo.nome,
        usuario: segredo.usuario,
        mover_unidade: true,
        unidade_id: unidade || null,
      });
      await lista.mutate();
    }, "Segredo movido");
  }

  async function excluir(segredo: Segredo) {
    await run(async () => {
      await apiClient.delete(`v1/cofre/segredos/${segredo.id}`);
      await lista.mutate();
    }, "Segredo excluído");
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Cofre"
        title="Cofre de senhas"
        description="A lista mostra o nome e o usuário. Revelar pede a sua senha de acesso de novo, e essa consulta fica na auditoria."
      />

      {gere && (
        <Card>
          <CardHeader>
            <CardTitle>Novo segredo</CardTitle>
          </CardHeader>
          <CardContent>
            <form className="grid gap-3 sm:grid-cols-2" onSubmit={(e) => void guardar(e)}>
              <Input id="cofre-nome" name="nome" label="Nome *" required maxLength={160} />
              <Input id="cofre-usuario" name="usuario" label="Usuário" maxLength={160} autoComplete="off" />
              <Input id="cofre-senha" name="senha" type="password" label="Senha *" required maxLength={2000} autoComplete="new-password" />
              <Textarea id="cofre-notas" name="notas" label="Notas" maxLength={4000} rows={3} />
              <UnidadeDonaSelect id="cofre-unidade" permission="cofre:manage" label="Unidade" />
              <div className="sm:col-span-2">
                <Button type="submit" loading={pending}>
                  Guardar no cofre
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      )}

      <DataState
        loading={lista.isLoading}
        error={lista.error}
        onRetry={() => void lista.mutate()}
        empty={(lista.data ?? []).length === 0}
        emptyTitle="Nenhum segredo neste recorte"
        emptyDescription="Os segredos da sua unidade aparecem aqui, sem a senha."
      >
        <ul className="flex flex-col gap-3">
          {(lista.data ?? []).map((segredo) => {
            const aberto = abertos[segredo.id];
            return (
              <li key={segredo.id}>
                <Card>
                  <CardHeader>
                    <CardTitle>{segredo.nome}</CardTitle>
                  </CardHeader>
                  <CardContent className="flex flex-col gap-3">
                    <p className="text-sm text-muted">{segredo.usuario || "Sem usuário"}</p>
                    {aberto && (
                      <div className="flex flex-col gap-2">
                        <Input
                          id={`senha-revelada-${segredo.id}`}
                          label={`Senha de ${segredo.nome}`}
                          readOnly
                          value={aberto.senha}
                          autoComplete="off"
                        />
                        {aberto.notas && <p className="text-sm text-muted">{aberto.notas}</p>}
                        <div>
                          <Button type="button" variant="secondary" onClick={() => setAbertos((atual) => {
                            const copia = { ...atual };
                            delete copia[segredo.id];
                            return copia;
                          })}>
                            Ocultar {segredo.nome}
                          </Button>
                        </div>
                      </div>
                    )}
                    <div className="flex flex-wrap gap-2">
                      {revela && !aberto && pedindo !== segredo.id && (
                        <Button type="button" variant="secondary" onClick={() => setPedindo(segredo.id)}>
                          Revelar {segredo.nome}
                        </Button>
                      )}
                      {gere && (
                        <Button type="button" variant="danger" onClick={() => void excluir(segredo)}>
                          Excluir {segredo.nome}
                        </Button>
                      )}
                    </div>
                    {revela && !aberto && pedindo === segredo.id && (
                      <form className="flex flex-col gap-2 sm:flex-row sm:items-end" onSubmit={(e) => void revelar(e, segredo)}>
                        <Input
                          id={`acesso-${segredo.id}`}
                          name="senha_de_acesso"
                          type="password"
                          label="Sua senha de acesso"
                          required
                          autoComplete="current-password"
                        />
                        <Button type="submit" variant="secondary">
                          Confirmar revelação de {segredo.nome}
                        </Button>
                      </form>
                    )}
                    {gere && (
                      <form className="flex flex-col gap-2 sm:flex-row sm:items-end" onSubmit={(e) => void trocar(e, segredo)}>
                        <Input
                          id={`nova-senha-${segredo.id}`}
                          name="senha"
                          type="password"
                          label={`Nova senha de ${segredo.nome}`}
                          required
                          autoComplete="new-password"
                        />
                        <Button type="submit" variant="secondary">
                          Trocar senha de {segredo.nome}
                        </Button>
                      </form>
                    )}
                    {concede && (
                      <form className="flex flex-col gap-2 sm:flex-row sm:items-end" onSubmit={(e) => void mover(e, segredo)}>
                        <UnidadeDonaSelect
                          id={`mover-${segredo.id}`}
                          permission="cofre:grant"
                          label={`Unidade de ${segredo.nome}`}
                          defaultValue={segredo.unidade_id}
                        />
                        <Button type="submit" variant="secondary">
                          Mover {segredo.nome}
                        </Button>
                      </form>
                    )}
                  </CardContent>
                </Card>
              </li>
            );
          })}
        </ul>
      </DataState>
    </div>
  );
}
