"use client";

import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { type FormEvent, Suspense, useState } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/plataforma/DataState";
import { ListaFaixas, NomeClicavel, type ColunaFaixa } from "@/components/plataforma/Faixas";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiQuery, withQuery } from "@/lib/api/swr";
import {
  ESTADO_ATIVO,
  TIPO_ATIVO,
  type Ativo,
  type CredencialAgente,
  type EstadoAtivo,
  type InventarioMaquina,
  type TipoAtivo,
} from "@/lib/servicos/servicos";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";

interface Produto {
  id: string;
  nome: string;
}

function textoBytes(n: number) {
  const unidades = ["B", "KB", "MB", "GB", "TB"];
  let valor = n;
  let i = 0;
  while (valor >= 1024 && i < unidades.length - 1) {
    valor /= 1024;
    i += 1;
  }
  const texto = valor.toLocaleString("pt-BR", { maximumFractionDigits: 1 });
  return `${texto} ${unidades[i]}`;
}

function InventarioDaMaquina({ inventario }: { inventario: InventarioMaquina }) {
  return (
    <dl className="mt-2 grid gap-1 text-sm sm:grid-cols-2">
      <div>
        <dt className="text-muted">Sistema</dt>
        <dd>{inventario.sistema}</dd>
      </div>
      <div>
        <dt className="text-muted">Processadores</dt>
        <dd>{inventario.processadores}</dd>
      </div>
      {inventario.memoria_bytes ? (
        <div>
          <dt className="text-muted">Memória</dt>
          <dd>{textoBytes(inventario.memoria_bytes)}</dd>
        </div>
      ) : null}
      {(inventario.discos ?? []).length > 0 ? (
        <div>
          <dt className="text-muted">Discos locais</dt>
          <dd>
            {inventario.discos!.map((d) => `${d.nome} ${textoBytes(d.bytes)}`).join(" · ")}
          </dd>
        </div>
      ) : null}
      {(inventario.programas ?? []).length > 0 ? (
        <div className="sm:col-span-2">
          <dt className="text-muted">
            Programas{inventario.programas_truncado ? " (lista parcial)" : ""}
          </dt>
          <dd>{inventario.programas!.map((p) => (p.versao ? `${p.nome} ${p.versao}` : p.nome)).join(" · ")}</dd>
        </div>
      ) : null}
    </dl>
  );
}

export default function AtivosPage() {
  return (
    <Suspense fallback={null}>
      <Ativos />
    </Suspense>
  );
}

/** Ativos (itens de configuração): o que sustenta cada produto digital.
 * Com ?produto=, mostra só os ativos daquele produto. */
function Ativos() {
  const { can } = usePlataforma();
  const gere = can("ativos:manage");
  const produto = useSearchParams().get("produto") ?? undefined;
  const ativos = useApiQuery<Ativo[]>(
    gere ? withQuery("v1/ativos", { produto_id: produto }) : null,
  );
  const produtos = useApiQuery<Produto[]>(gere && can("produtos:manage") ? "v1/produtos" : null);
  const [editando, setEditando] = useState<Ativo | null>(null);
  const { run, pending } = useAction();

  if (!gere) {
    return (
      <div className="flex flex-col gap-6">
        <PageHeader eyebrow="Gestão de serviços" title="Ativos" />
        <EmptyState
          title="Seu perfil não gere ativos"
          description="Peça o acesso à gestão da sua unidade."
        />
      </div>
    );
  }

  const nomeDoProduto = new Map((produtos.data ?? []).map((p) => [p.id, p.nome]));

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    const alvo = editando;
    const body = {
      nome: String(fd.get("nome")).trim(),
      tipo: String(fd.get("tipo")),
      estado: String(fd.get("estado")),
      identificador: String(fd.get("identificador")).trim(),
      descricao: String(fd.get("descricao")).trim(),
      produto_id: String(fd.get("produto_id")) || null,
      unidade_id: String(fd.get("unidade_id")) || null,
    };
    const ok = await run(
      () =>
        alvo ? apiClient.put(`v1/ativos/${alvo.id}`, body) : apiClient.post("v1/ativos", body),
      alvo ? "Ativo atualizado" : "Ativo cadastrado",
    );
    if (ok) {
      setEditando(null);
      if (!alvo) form.reset();
      void ativos.mutate();
    }
  }

  async function criarCard(colunaId: string, nome: string) {
    const ok = await run(
      () =>
        apiClient.post("v1/ativos", {
          nome: nome.slice(0, 160),
          tipo: "outro",
          estado: colunaId,
          identificador: "",
          descricao: "",
        }),
      "Ativo cadastrado",
    );
    if (ok) void ativos.mutate();
  }

  async function renomearAtivo(a: Ativo, nome: string) {
    const ok = await run(
      () =>
        apiClient.put(`v1/ativos/${a.id}`, {
          nome: nome.slice(0, 160),
          tipo: a.tipo,
          estado: a.estado,
          identificador: a.identificador,
          descricao: a.descricao,
          produto_id: a.produto_id ?? null,
          unidade_id: a.unidade_id ?? null,
        }),
      "Nome atualizado",
    );
    if (ok) void ativos.mutate();
  }

  const lista = ativos.data ?? [];
  const colunasAtivo: ColunaFaixa[] = (Object.keys(ESTADO_ATIVO) as EstadoAtivo[]).map((id) => ({
    id,
    titulo: ESTADO_ATIVO[id].label,
    tom: ESTADO_ATIVO[id].tone,
  }));
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Gestão de serviços"
        title="Ativos"
        description="Equipamentos, sistemas, licenças, links e serviços contratados que sustentam os produtos digitais."
      />
      {produto && (
        <p role="status" className="text-sm">
          Mostrando os ativos de <strong>{nomeDoProduto.get(produto) ?? "um produto"}</strong>.{" "}
          <Link href="/ativos" className="text-primary hover:underline">
            Ver todos
          </Link>
        </p>
      )}

      <Card>
        <CardHeader>
          <CardTitle as="h2">{editando ? "Editar ativo" : "Novo ativo"}</CardTitle>
        </CardHeader>
        <CardContent className="pb-5">
          <form
            key={editando?.id ?? "novo"}
            onSubmit={salvar}
            className="mt-3 grid gap-3 sm:grid-cols-2"
          >
            <Input
              id="ativo-nome"
              name="nome"
              label="Nome *"
              required
              maxLength={160}
              defaultValue={editando?.nome}
              autoComplete="off"
            />
            <Input
              id="ativo-identificador"
              name="identificador"
              label="Patrimônio ou número de série"
              maxLength={80}
              defaultValue={editando?.identificador}
              autoComplete="off"
            />
            <Select
              id="ativo-tipo"
              name="tipo"
              label="Tipo"
              defaultValue={editando?.tipo ?? "hardware"}
              options={(Object.keys(TIPO_ATIVO) as TipoAtivo[]).map((t) => ({
                value: t,
                label: TIPO_ATIVO[t],
              }))}
            />
            <Select
              id="ativo-estado"
              name="estado"
              label="Estado"
              defaultValue={editando?.estado ?? "em_uso"}
              options={(Object.keys(ESTADO_ATIVO) as EstadoAtivo[]).map((e) => ({
                value: e,
                label: ESTADO_ATIVO[e].label,
              }))}
            />
            <Select
              id="ativo-produto"
              name="produto_id"
              label="Produto que sustenta"
              defaultValue={editando?.produto_id ?? produto ?? ""}
              options={[
                { value: "", label: "Nenhum" },
                ...(produtos.data ?? []).map((p) => ({ value: p.id, label: p.nome })),
              ]}
            />
            <UnidadeDonaSelect
              id="ativo-unidade"
              permission="ativos:manage"
              label="Unidade"
              defaultValue={editando?.unidade_id}
            />
            <div className="sm:col-span-2">
              <Textarea
                id="ativo-descricao"
                name="descricao"
                label="Descrição"
                maxLength={4000}
                rows={2}
                defaultValue={editando?.descricao}
              />
            </div>
            <div className="flex flex-wrap gap-2 sm:col-span-2">
              <Button type="submit" loading={pending}>
                {editando ? "Salvar ativo" : "Cadastrar ativo"}
              </Button>
              {editando && (
                <Button type="button" variant="secondary" onClick={() => setEditando(null)}>
                  Cancelar edição
                </Button>
              )}
            </div>
          </form>
        </CardContent>
      </Card>

      <DataState
        loading={ativos.isLoading}
        error={ativos.error}
        onRetry={() => void ativos.mutate()}
        empty={false}
      >
        {lista.length === 0 && <p className="text-sm text-muted">Nenhum ativo cadastrado</p>}
        <ListaFaixas
          rotulo="Ativos cadastrados"
          memoria="nova-faixas:ativos"
          colunas={colunasAtivo}
          itens={lista}
          colunaDe={(a) => a.estado}
          idDe={(a) => a.id}
          nomeDe={(a) => a.nome}
          extra={(colunaId) =>
            colunaId === "em_uso" ? (
              <Button
                type="button"
                size="sm"
                variant="secondary"
                onClick={() => document.getElementById("ativo-nome")?.focus()}
              >
                Novo ativo
              </Button>
            ) : null
          }
          aoCriar={(coluna, nome) => void criarCard(coluna, nome)}
          aoRenomear={(a, nome) => void renomearAtivo(a, nome)}
          aoAbrir={(a) => {
            setEditando(a);
            document.getElementById("ativo-nome")?.focus();
          }}
          render={(a, vista) => (
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <NomeClicavel
                  nome={vista.titulo || a.nome}
                  onSalvar={vista.salvarNome}
                  className="text-left font-medium text-foreground hover:underline"
                />
                <p className="text-sm text-muted">
                  {TIPO_ATIVO[a.tipo]}
                  {a.identificador ? ` · ${a.identificador}` : ""}
                  {a.produto_id ? ` · ${nomeDoProduto.get(a.produto_id) ?? "produto vinculado"}` : ""}
                  {a.origem === "agente" ? " · informado pela máquina" : ""}
                </p>
                {a.inventario ? <InventarioDaMaquina inventario={a.inventario} /> : null}
              </div>
              <div className="flex items-center gap-2">
                <Badge tone={ESTADO_ATIVO[a.estado].tone}>{ESTADO_ATIVO[a.estado].label}</Badge>
                <Button type="button" variant="secondary" onClick={() => setEditando(a)}>
                  Editar {a.nome}
                </Button>
              </div>
            </div>
          )}
        />
      </DataState>

      <CredenciaisDaMaquina />

      {lista.length >= 2 ? <Relacoes origemOpcoes={lista} onMudou={() => void ativos.mutate()} /> : null}
    </div>
  );
}

function CredenciaisDaMaquina() {
  const lista = useApiQuery<CredencialAgente[]>("v1/ativos/credenciais");
  const [segredo, setSegredo] = useState("");
  const [erro, setErro] = useState("");

  async function gerar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    setErro("");
    try {
      const res = await apiClient.post<CredencialAgente>("v1/ativos/credenciais", {
        nome: String(fd.get("nome")).trim(),
        unidade_id: String(fd.get("unidade_id")),
      });
      setSegredo(res.data.token ?? "");
      e.currentTarget.reset();
      await lista.mutate();
    } catch {
      setErro("Não foi possível gerar a credencial.");
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Credencial da máquina</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 pb-5">
        <p className="text-sm text-muted">
          O segredo aparece uma vez. A máquina usa esse token no lugar da senha de uma pessoa e só pode informar o próprio patrimônio.
        </p>
        {lista.error ? (
          <p className="text-sm text-danger" role="alert">
            Não foi possível carregar as credenciais.
          </p>
        ) : null}
        <ul className="text-sm">
          {(lista.data ?? []).map((c) => (
            <li key={c.id}>{c.nome}</li>
          ))}
        </ul>
        <form className="flex flex-col gap-3" onSubmit={(e) => void gerar(e)}>
          <Input id="credencial-nome" name="nome" label="Nome da máquina" required maxLength={160} />
          <UnidadeDonaSelect id="credencial-unidade" permission="ativos:manage" label="Unidade" />
          {erro ? (
            <p className="text-sm text-danger" role="alert">
              {erro}
            </p>
          ) : null}
          {segredo ? (
            <p className="text-sm">
              Copie agora. Este segredo não aparece de novo: <code>{segredo}</code>
            </p>
          ) : null}
          <Button type="submit">Gerar credencial</Button>
        </form>
      </CardContent>
    </Card>
  );
}

function Relacoes({
  origemOpcoes,
  onMudou,
}: {
  origemOpcoes: Ativo[];
  onMudou: () => void;
}) {
  const [origem, setOrigem] = useState(origemOpcoes[0]?.id ?? "");
  const relacoes = useApiQuery<{ id: string; destino_nome: string; tipo: string }[]>(
    origem ? `v1/ativos/${origem}/relacoes` : null,
  );
  const { run, pending } = useAction();

  async function ligar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const ok = await run(
      () =>
        apiClient.post(`v1/ativos/${origem}/relacoes`, {
          destino_id: String(fd.get("destino_id")),
          tipo: String(fd.get("tipo")),
        }),
      "Relação registrada",
    );
    if (ok) {
      e.currentTarget.reset();
      void relacoes.mutate();
      onMudou();
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2">Relações</CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-3 pb-5">
        <p className="text-sm text-muted">
          O que depende do quê, o que compõe um item e o que está conectado. Dependência em ciclo não entra.
        </p>
        <Select
          id="relacao-origem"
          label="A partir de"
          value={origem}
          onChange={(e) => setOrigem(e.target.value)}
          options={origemOpcoes.map((a) => ({ value: a.id, label: a.nome }))}
        />
        <form onSubmit={ligar} className="grid gap-3 sm:grid-cols-2">
          <Select
            id="relacao-destino"
            name="destino_id"
            label="Aponta para *"
            required
            placeholder="Escolha o ativo"
            options={origemOpcoes.filter((a) => a.id !== origem).map((a) => ({ value: a.id, label: a.nome }))}
          />
          <Select
            id="relacao-tipo"
            name="tipo"
            label="Tipo da relação"
            defaultValue="depende"
            options={[
              { value: "depende", label: "Depende" },
              { value: "composto", label: "Compõe" },
              { value: "conectado", label: "Conectado" },
            ]}
          />
          <div>
            <Button type="submit" loading={pending}>
              Registrar relação
            </Button>
          </div>
        </form>
        <ul aria-label="Relações do ativo" className="text-sm">
          {(relacoes.data ?? []).map((r) => (
            <li key={r.id}>
              {r.tipo} · {r.destino_nome}
            </li>
          ))}
        </ul>
      </CardContent>
    </Card>
  );
}
