"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { type FormEvent, useMemo, useState } from "react";

import { ResumoExpediente, type Expediente } from "@/components/integracoes/ResumoExpediente";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import {
  ativa,
  type Conexao,
  duracao,
  type Inventario,
  type Linha,
  PRIORIDADES,
  ROTULO_ESTADO,
  ROTULO_TAREFA,
  type Tarefa,
  type TipoTarefa,
  TIPOS,
} from "@/lib/integracoes/types";
import type { OrgTree } from "@/lib/plataforma/types";

function texto(fd: FormData, nome: string) {
  return String(fd.get(nome) ?? "").trim();
}

function data(iso?: string) {
  return iso ? new Date(iso).toLocaleString("pt-BR") : "—";
}

export default function ConexaoPage() {
  const { id } = useParams<{ id: string }>();
  const base = `v1/integracoes/conexoes/${id}`;
  const conexao = useApiQuery<Conexao>(base);
  const tarefas = useApiQuery<Tarefa[]>(`${base}/tarefas`, {
    refreshInterval: (lista) => ((lista ?? []).some(ativa) ? 3000 : 0),
  });
  const inventario = useApiQuery<Inventario>(`${base}/inventario`);
  const tabela = useApiQuery<Linha[]>(`${base}/tabela`);
  const arvore = useApiQuery<OrgTree[]>("v1/iam/org-tree");
  const { run, pending } = useAction();
  const [token, setToken] = useState("");

  const unidades = useMemo(
    () => [
      { value: "", label: "— escolha a unidade —" },
      ...(arvore.data ?? []).flatMap((e) =>
        e.unidades.map((u) => ({ value: u.id, label: `${e.nome} — ${u.nome}` })),
      ),
    ],
    [arvore.data],
  );

  function recarregar() {
    void conexao.mutate();
    void tarefas.mutate();
    void inventario.mutate();
    void tabela.mutate();
  }

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const ok = await run(
      () =>
        apiClient.put(base, {
          nome: texto(fd, "nome"),
          url: texto(fd, "url"),
          perfil: texto(fd, "perfil"),
          fuso: texto(fd, "fuso"),
        }),
      "Conexão salva",
    );
    if (ok) void conexao.mutate();
  }

  async function credenciais(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const ok = await run(
      () =>
        apiClient.put(`${base}/credenciais`, {
          app_token: texto(fd, "app_token"),
          user_token: texto(fd, "user_token"),
          login: texto(fd, "login"),
          senha: String(fd.get("senha") ?? ""),
        }),
      "Credenciais guardadas, cifradas",
    );
    if (ok) {
      form.reset();
      void conexao.mutate();
    }
  }

  async function emitirToken() {
    const out = await run(
      () => apiClient.post<{ token: string }>(`${base}/token`),
      "Token emitido",
    );
    if (out) {
      setToken(out.data.token);
      void conexao.mutate();
    }
  }

  async function revogarToken() {
    if (await run(() => apiClient.delete(`${base}/token`), "Token revogado")) {
      setToken("");
      void conexao.mutate();
    }
  }

  async function pedir(tipo: TipoTarefa) {
    if (
      await run(() => apiClient.post(`${base}/tarefas`, { tipo }), `${ROTULO_TAREFA[tipo]} pedida`)
    ) {
      void tarefas.mutate();
    }
  }

  async function cancelar(t: Tarefa) {
    if (
      await run(() => apiClient.post(`v1/integracoes/tarefas/${t.id}/cancelar`), "Tarefa cancelada")
    ) {
      void tarefas.mutate();
    }
  }

  const c = conexao.data;
  const algumaAtiva = (tarefas.data ?? []).some(ativa);

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Integrações"
        title={c?.nome ?? "Conexão"}
        description={
          c
            ? `${c.tipo.toUpperCase()} ${c.modo === "conector" ? "pelo conector" : "direto"} · extraído em ${data(c.extraido_em)}`
            : undefined
        }
      />
      <Link href="/integracoes" className="text-sm text-primary hover:underline">
        Voltar às conexões
      </Link>

      <DataState
        loading={conexao.isLoading}
        error={conexao.error}
        onRetry={() => void conexao.mutate()}
        empty={!c}
        emptyTitle="Conexão não encontrada"
      >
        {c && (
          <>
            <Card>
              <CardHeader>
                <CardTitle as="h2">Conexão</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-6 pb-5">
                <form key={c.id + c.nome} onSubmit={salvar} className="mt-3 grid gap-3">
                  <Input
                    id="nome"
                    name="nome"
                    label="Nome *"
                    required
                    maxLength={160}
                    defaultValue={c.nome}
                  />
                  <Input
                    id="url"
                    name="url"
                    label={
                      c.modo === "direto" ? "Endereço da API *" : "Endereço da API (informativo)"
                    }
                    required={c.modo === "direto"}
                    maxLength={500}
                    defaultValue={c.url}
                  />
                  <Input
                    id="perfil"
                    name="perfil"
                    label="Perfil no GLPI"
                    maxLength={80}
                    defaultValue={c.perfil}
                  />
                  <Input
                    id="fuso"
                    name="fuso"
                    label="Fuso das datas da origem"
                    maxLength={60}
                    defaultValue={c.fuso}
                  />
                  <div>
                    <Button type="submit" loading={pending}>
                      Salvar conexão
                    </Button>
                  </div>
                </form>

                {c.modo === "direto" ? (
                  <form
                    onSubmit={credenciais}
                    className="grid gap-3"
                    aria-label="Credenciais do GLPI"
                  >
                    <p className="text-sm text-muted">
                      {c.tem_credenciais
                        ? "Há credenciais guardadas (cifradas). Preencha para trocar."
                        : "Sem credenciais. Use o token de usuário, ou o login e a senha de uma conta do GLPI."}
                    </p>
                    <Input id="app_token" name="app_token" label="App-Token" autoComplete="off" />
                    <Input
                      id="user_token"
                      name="user_token"
                      label="Token de usuário"
                      autoComplete="off"
                    />
                    <Input id="login" name="login" label="Login" autoComplete="off" />
                    <Input
                      id="senha"
                      name="senha"
                      type="password"
                      label="Senha"
                      autoComplete="new-password"
                    />
                    <div>
                      <Button type="submit" variant="secondary" loading={pending}>
                        Guardar credenciais
                      </Button>
                    </div>
                  </form>
                ) : (
                  <div className="grid gap-3">
                    <p className="text-sm text-muted">
                      O conector roda numa máquina da rede do GLPI, com as credenciais do GLPI. Aqui
                      fica só o token que ele usa para enviar os dados.{" "}
                      {c.tem_token
                        ? `Token emitido em ${data(c.token_em)}.`
                        : "Nenhum token emitido."}
                    </p>
                    {token && (
                      <div className="rounded-lg border border-surface-border bg-surface-muted p-3 text-sm">
                        <p className="font-medium">Copie agora: o token não aparece de novo.</p>
                        <pre className="mt-2 overflow-x-auto whitespace-pre-wrap break-all">{`NOVA_API=${window.location.origin.replace("//", "//api.")}
NOVA_TOKEN=${token}
GLPI_URL=http://servidor-do-glpi/apirest.php
GLPI_APP_TOKEN=...
GLPI_LOGIN=...
GLPI_PASSWORD=...
conector --vigiar`}</pre>
                      </div>
                    )}
                    <div className="flex flex-wrap gap-2">
                      <Button
                        type="button"
                        variant="secondary"
                        onClick={() => void emitirToken()}
                        loading={pending}
                      >
                        {c.tem_token ? "Emitir novo token" : "Emitir token do conector"}
                      </Button>
                      {c.tem_token && (
                        <Button
                          type="button"
                          variant="secondary"
                          onClick={() => void revogarToken()}
                        >
                          Revogar token
                        </Button>
                      )}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle as="h2">Tarefas</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-4 pb-5">
                <p className="mt-3 text-sm text-muted">
                  Extraia (de novo, só o que mudou), revise a tabela de-para e simule. A carga grava
                  no atendimento e pode ser repetida até a virada: o que já entrou não se repete.
                </p>
                <div className="flex flex-wrap gap-2">
                  <Button
                    type="button"
                    disabled={algumaAtiva}
                    onClick={() => void pedir("extracao")}
                  >
                    Extrair
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={algumaAtiva}
                    onClick={() => void pedir("simulacao")}
                  >
                    Simular carga
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={algumaAtiva}
                    onClick={() => void pedir("carga")}
                  >
                    Carregar no atendimento
                  </Button>
                </div>
                <DataState
                  loading={tarefas.isLoading}
                  error={tarefas.error}
                  onRetry={() => void tarefas.mutate()}
                  empty={(tarefas.data ?? []).length === 0}
                  emptyTitle="Nenhuma tarefa ainda"
                >
                  <ul className="flex flex-col divide-y divide-surface-border">
                    {(tarefas.data ?? []).map((t) => (
                      <li key={t.id} className="flex flex-col gap-1 py-3">
                        <p className="font-medium">
                          {ROTULO_TAREFA[t.tipo]} · {ROTULO_ESTADO[t.estado]}
                        </p>
                        <p className="text-sm text-muted">
                          Pedida por {t.pedida_por} em {data(t.criada_em)}
                          {t.total > 0 ? ` · ${t.feitos} de ${t.total}` : ""}
                        </p>
                        {t.total > 0 && ativa(t) && (
                          <progress
                            className="w-full"
                            value={t.feitos}
                            max={t.total}
                            aria-label={`Andamento: ${t.feitos} de ${t.total}`}
                          />
                        )}
                        {t.erro && <p className="text-sm text-danger">{t.erro}</p>}
                        {t.relatorio?.expediente ? (
                          <ResumoExpediente
                            expediente={t.relatorio.expediente as Expediente}
                            gravou={t.relatorio.gravou === true}
                          />
                        ) : null}
                        {t.relatorio && (
                          <details>
                            <summary className="cursor-pointer text-sm">Relatório</summary>
                            <pre className="mt-1 overflow-x-auto text-xs">
                              {JSON.stringify(t.relatorio, null, 2)}
                            </pre>
                          </details>
                        )}
                        {ativa(t) && (
                          <div>
                            <Button
                              type="button"
                              variant="secondary"
                              onClick={() => void cancelar(t)}
                            >
                              Cancelar {ROTULO_TAREFA[t.tipo].toLowerCase()}
                            </Button>
                          </div>
                        )}
                      </li>
                    ))}
                  </ul>
                </DataState>
                <div>
                  <Button type="button" variant="secondary" onClick={recarregar}>
                    Atualizar
                  </Button>
                </div>
              </CardContent>
            </Card>

            <InventarioCard inventario={inventario.data} />

            <TabelaCard
              key={(tabela.data ?? []).length + (tabela.data?.[0]?.oferta ?? "")}
              base={base}
              linhas={tabela.data ?? []}
              secretarias={Object.keys(inventario.data?.secretarias ?? {}).sort()}
              unidades={unidades}
              onSalva={() => void tabela.mutate()}
            />
          </>
        )}
      </DataState>
    </div>
  );
}

function InventarioCard({ inventario }: { inventario?: Inventario }) {
  const itens = inventario?.itens ?? {};
  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Inventário da origem</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4 pb-5">
        {(itens.chamado ?? 0) === 0 ? (
          <p className="mt-3 text-sm text-muted">Nada extraído ainda.</p>
        ) : (
          <>
            <p className="mt-3 text-sm">
              {itens.chamado ?? 0} chamados, {itens.pessoa ?? 0} pessoas, {itens.categoria ?? 0}{" "}
              categorias, {itens.entidade ?? 0} entidades e {itens.sla ?? 0} prazos (SLA).
            </p>
            <ul className="text-sm">
              {Object.entries(inventario?.secretarias ?? {}).map(([s, n]) => (
                <li key={s}>
                  {s}: {n} chamados
                </li>
              ))}
            </ul>
            {(inventario?.niveis ?? []).length > 0 && (
              <div>
                <p className="text-sm font-medium">Prazos da origem</p>
                <ul className="text-sm">
                  {(inventario?.niveis ?? []).map((n) => (
                    <li key={n.id}>
                      {n.nome}: {duracao(n.minutos)} · {n.chamados} chamados
                      {n.prioridade ? ` · prioridade sugerida ${n.prioridade}` : ""}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}

function TabelaCard({
  base,
  linhas,
  secretarias,
  unidades,
  onSalva,
}: {
  base: string;
  linhas: Linha[];
  secretarias: string[];
  unidades: { value: string; label: string }[];
  onSalva: () => void;
}) {
  const { run, pending } = useAction();
  const [edicao, setEdicao] = useState<Linha[]>(linhas);

  function mudar(i: number, campo: keyof Linha, valor: string) {
    setEdicao((atual) =>
      atual.map((l, j) =>
        j === i ? { ...l, [campo]: campo === "unidade_id" ? valor || null : valor } : l,
      ),
    );
  }

  async function sugerir(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const mapa: Record<string, string> = {};
    for (const s of secretarias) {
      const u = String(fd.get(`sec-${s}`) ?? "");
      if (u) mapa[s] = u;
    }
    const out = await run(
      () => apiClient.post<Linha[]>(`${base}/tabela/sugestao`, { unidades: mapa }),
      "Tabela sugerida a partir do inventário",
    );
    if (out) {
      setEdicao(out.data);
      onSalva();
    }
  }

  async function salvar() {
    if (await run(() => apiClient.put(`${base}/tabela`, { linhas: edicao }), "Tabela salva"))
      onSalva();
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Tabela de-para</CardTitle>
      </CardHeader>
      <CardContent className="grid gap-4 pb-5">
        <p className="mt-3 text-sm text-muted">
          Cada categoria de cada secretaria da origem vai a uma oferta e a uma unidade daqui, com a
          prioridade que substitui o SLA da categoria. Ofertas com o mesmo nome se juntam. A linha
          &ldquo;(qualquer outra)&rdquo; recebe o que não estiver na tabela. A categoria no catálogo
          é o bloco em que a oferta aparece na central: a sugestão usa a raiz da árvore do GLPI
          (&ldquo;Hardware &gt; Impressora&rdquo; fica em &ldquo;Hardware&rdquo;); em branco, a
          oferta fica em &ldquo;Outros&rdquo;.
        </p>
        {secretarias.length > 0 && (
          <form onSubmit={sugerir} className="grid gap-3" aria-label="Sugerir a tabela">
            {secretarias.map((s) => (
              <Select
                key={s}
                id={`sec-${s}`}
                name={`sec-${s}`}
                label={`Unidade de ${s}`}
                options={unidades}
              />
            ))}
            <div>
              <Button type="submit" variant="secondary" loading={pending}>
                Sugerir a tabela a partir do inventário
              </Button>
            </div>
          </form>
        )}
        {edicao.length > 0 && (
          <>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left">
                    <th className="p-2">Secretaria</th>
                    <th className="p-2">Categoria</th>
                    <th className="p-2">Chamados</th>
                    <th className="p-2">Unidade</th>
                    <th className="p-2">Oferta</th>
                    <th className="p-2">Categoria no catálogo</th>
                    <th className="p-2">Tipo</th>
                    <th className="p-2">Prioridade</th>
                  </tr>
                </thead>
                <tbody>
                  {edicao.map((l, i) => {
                    const nome = `${l.secretaria} ${l.categoria_nome}`;
                    return (
                      <tr
                        key={`${l.secretaria}-${l.categoria}`}
                        className="border-t border-surface-border align-top"
                      >
                        <td className="p-2">{l.secretaria}</td>
                        <td className="p-2">
                          {l.categoria_nome}
                          {l.nivel ? (
                            <span className="block text-xs text-muted">{l.nivel}</span>
                          ) : null}
                        </td>
                        <td className="p-2">{l.chamados}</td>
                        <td className="p-2">
                          <Select
                            aria-label={`Unidade de ${nome}`}
                            value={l.unidade_id ?? ""}
                            onChange={(e) => mudar(i, "unidade_id", e.target.value)}
                            options={unidades}
                          />
                        </td>
                        <td className="p-2">
                          <Input
                            aria-label={`Oferta de ${nome}`}
                            value={l.oferta}
                            maxLength={160}
                            onChange={(e) => mudar(i, "oferta", e.target.value)}
                          />
                        </td>
                        <td className="p-2">
                          <Input
                            aria-label={`Categoria no catálogo de ${nome}`}
                            value={l.catalogo ?? ""}
                            maxLength={80}
                            placeholder="Outros"
                            onChange={(e) => mudar(i, "catalogo", e.target.value)}
                          />
                        </td>
                        <td className="p-2">
                          <Select
                            aria-label={`Tipo de ${nome}`}
                            value={l.tipo}
                            onChange={(e) => mudar(i, "tipo", e.target.value)}
                            options={TIPOS}
                          />
                        </td>
                        <td className="p-2">
                          <Select
                            aria-label={`Prioridade de ${nome}`}
                            value={l.prioridade}
                            onChange={(e) => mudar(i, "prioridade", e.target.value)}
                            options={PRIORIDADES}
                          />
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
            <div>
              <Button type="button" onClick={() => void salvar()} loading={pending}>
                Salvar tabela
              </Button>
            </div>
          </>
        )}
      </CardContent>
    </Card>
  );
}
