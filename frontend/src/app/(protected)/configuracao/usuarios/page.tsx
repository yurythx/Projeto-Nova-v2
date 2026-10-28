"use client";

import { KeyRound, Lock, Plus, RefreshCw, Search, Trash2, Unlock, UserCog } from "lucide-react";
import { useRef, useState, type FormEvent } from "react";

import { ScopePicker } from "@/components/iam/ScopePicker";
import { ConfirmButton } from "@/components/plataforma/ConfirmButton";
import { DataState } from "@/components/plataforma/DataState";
import { Pagination } from "@/components/plataforma/Pagination";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Dialog } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeaderCell,
  TableRow,
} from "@/components/ui/Table";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { Perfil, ScopeRef, UserAdmin } from "@/lib/plataforma/types";

const PASSWORD_HINT =
  "Mínimo de 12 caracteres, com letras maiúsculas, minúsculas, dígitos e símbolos.";

function CreateUserForm({ onDone }: { onDone: () => void }) {
  const { run, pending } = useAction();
  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const ok = await run(
      () =>
        apiClient.post<UserAdmin>("v1/users", {
          username: String(fd.get("username") ?? "").trim(),
          email: String(fd.get("email") ?? "").trim(),
          display_name: String(fd.get("display_name") ?? "").trim(),
          password: String(fd.get("password") ?? ""),
          roles: [],
        }),
      "Conta local criada",
    );
    if (ok) onDone();
  }
  return (
    <form onSubmit={submit} className="flex flex-col gap-3">
      <p className="text-xs text-muted">
        Contas locais são a contingência do Keycloak (RS256 local). Usuários do AD são provisionados
        no primeiro login federado.
      </p>
      <div className="grid gap-3 sm:grid-cols-2">
        <Input
          id="nu-username"
          name="username"
          label="Usuário *"
          required
          minLength={3}
          maxLength={80}
          autoComplete="off"
        />
        <Input
          id="nu-email"
          name="email"
          type="email"
          label="E-mail *"
          required
          maxLength={200}
          autoComplete="off"
        />
      </div>
      <Input id="nu-display" name="display_name" label="Nome de exibição" maxLength={200} />
      <div>
        <Input
          id="nu-password"
          name="password"
          type="password"
          label="Senha inicial *"
          required
          minLength={12}
          maxLength={256}
          autoComplete="new-password"
        />
        <p className="mt-1 text-xs text-muted">{PASSWORD_HINT}</p>
      </div>
      <p className="text-xs text-muted">
        Depois de criar, atribua perfis na aba de lotações do usuário.
      </p>
      <div className="flex justify-end">
        <Button type="submit" loading={pending}>
          Criar conta
        </Button>
      </div>
    </form>
  );
}

function UserDetail({ userId, onChanged }: { userId: string; onChanged: () => void }) {
  const { can } = usePlataforma();
  const user = useApiQuery<UserAdmin>(`v1/users/${userId}`);
  const perfis = useApiQuery<Perfil[]>(can("iam:manage") ? "v1/iam/perfis" : null);
  const { run, pending } = useAction();
  const [scope, setScope] = useState<ScopeRef>({});
  const [perfilId, setPerfilId] = useState("");
  const [password, setPassword] = useState("");

  const refresh = () => {
    void user.mutate();
    onChanged();
  };
  const u = user.data;
  if (!u)
    return (
      <DataState loading={user.isLoading} error={user.error} empty={false}>
        {null}
      </DataState>
    );

  const locked = u.locked_until && new Date(u.locked_until) > new Date();

  return (
    <div className="flex flex-col gap-5">
      <dl className="grid gap-2 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-xs text-muted">Usuário</dt>
          <dd className="font-mono">{u.username}</dd>
        </div>
        <div>
          <dt className="text-xs text-muted">E-mail</dt>
          <dd>{u.email}</dd>
        </div>
        <div>
          <dt className="text-xs text-muted">Origem</dt>
          <dd>
            {u.federated ? "Federado (Keycloak/AD)" : "Local"}
            {u.local_login && u.federated ? " + login local" : ""}
          </dd>
        </div>
        <div>
          <dt className="text-xs text-muted">Último acesso</dt>
          <dd>{fmtDateTime(u.last_seen_at)}</dd>
        </div>
        {u.ad_synced_at && (
          <div>
            <dt className="text-xs text-muted">Sincronizado com o AD</dt>
            <dd>{fmtDateTime(u.ad_synced_at)}</dd>
          </div>
        )}
        {u.groups.length > 0 && (
          <div className="sm:col-span-2">
            <dt className="text-xs text-muted">Grupos do AD</dt>
            <dd className="mt-1 flex flex-wrap gap-1">
              {u.groups.map((g) => (
                <Badge key={g}>{g}</Badge>
              ))}
            </dd>
          </div>
        )}
      </dl>

      {can("users:manage") && (
        <section
          aria-labelledby="ud-actions"
          className="flex flex-col gap-3 rounded-lg border border-surface-border p-4"
        >
          <h3 id="ud-actions" className="text-sm font-semibold">
            Conta
          </h3>
          <div className="flex flex-wrap gap-2">
            <Button
              variant={u.active ? "secondary" : "primary"}
              size="sm"
              loading={pending}
              onClick={() =>
                void run(
                  () =>
                    apiClient.patch(`v1/users/${u.id}`, {
                      display_name: u.display_name,
                      active: !u.active,
                      roles: u.roles,
                    }),
                  u.active ? "Conta desativada" : "Conta reativada",
                ).then(refresh)
              }
            >
              {u.active ? (
                <Lock size={14} aria-hidden="true" className="mr-1" />
              ) : (
                <Unlock size={14} aria-hidden="true" className="mr-1" />
              )}
              {u.active ? "Desativar" : "Reativar"}
            </Button>
            {locked && (
              <Button
                variant="secondary"
                size="sm"
                onClick={() =>
                  void run(
                    () => apiClient.post(`v1/users/${u.id}/unlock`),
                    "Bloqueio removido",
                  ).then(refresh)
                }
              >
                <Unlock size={14} aria-hidden="true" className="mr-1" /> Remover bloqueio (até{" "}
                {fmtDateTime(u.locked_until)})
              </Button>
            )}
          </div>
          {!u.federated ? (
            <form
              className="flex flex-col gap-2 sm:flex-row sm:items-end"
              onSubmit={(e) => {
                e.preventDefault();
                void run(
                  () => apiClient.post(`v1/users/${u.id}/password`, { password }),
                  "Senha redefinida",
                ).then((ok) => ok !== undefined && setPassword(""));
              }}
            >
              <div className="flex-1">
                <Input
                  id="ud-password"
                  type="password"
                  label="Nova senha local"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  minLength={12}
                  required
                  autoComplete="new-password"
                />
              </div>
              <Button type="submit" variant="secondary" loading={pending}>
                <KeyRound size={14} aria-hidden="true" className="mr-1" /> Redefinir
              </Button>
            </form>
          ) : (
            <p className="text-xs text-muted">
              Senha gerida pelo Active Directory: contas federadas não recebem senha local (seria
              uma entrada que ignora a política e a desativação do diretório).
            </p>
          )}
        </section>
      )}

      <section aria-labelledby="ud-lotacoes" className="flex flex-col gap-3">
        <h3 id="ud-lotacoes" className="text-sm font-semibold">
          Lotações (perfil × escopo)
        </h3>
        {(u.lotacoes ?? []).length === 0 ? (
          <p className="text-sm text-muted">
            Sem lotações manuais. Perfis vindos do AD são resolvidos no login.
          </p>
        ) : (
          <ul className="flex flex-col divide-y divide-surface-border rounded-lg border border-surface-border">
            {(u.lotacoes ?? []).map((l) => (
              <li key={l.id} className="flex items-center justify-between gap-2 px-3 py-2 text-sm">
                <span>
                  <strong>{l.perfil_nome}</strong> · {l.scope_label || "Global"}{" "}
                  {l.principal && (
                    <Badge tone="info" className="ml-1">
                      Principal
                    </Badge>
                  )}
                </span>
                {can("iam:manage") && (
                  <ConfirmButton
                    aria-label={`Remover lotação ${l.perfil_nome}`}
                    title="Remover lotação?"
                    confirmLabel="Remover"
                    onConfirm={() =>
                      run(
                        () => apiClient.delete(`v1/users/${u.id}/lotacoes/${l.id}`),
                        "Lotação removida",
                      ).then(refresh)
                    }
                  >
                    <Trash2 size={14} aria-hidden="true" />
                  </ConfirmButton>
                )}
              </li>
            ))}
          </ul>
        )}
        {can("iam:manage") && (
          <form
            className="flex flex-col gap-3 rounded-lg border border-dashed border-surface-border p-3"
            onSubmit={(e) => {
              e.preventDefault();
              void run(
                () =>
                  apiClient.post(`v1/users/${u.id}/lotacoes`, {
                    perfil_id: perfilId,
                    principal: (u.lotacoes ?? []).length === 0,
                    ...scope,
                  }),
                "Lotação adicionada",
              ).then((ok) => {
                if (ok !== undefined) {
                  setPerfilId("");
                  setScope({});
                  refresh();
                }
              });
            }}
          >
            <Select
              id="ud-perfil"
              label="Perfil *"
              required
              placeholder="Selecione…"
              value={perfilId}
              onChange={(e) => setPerfilId(e.target.value)}
              options={(perfis.data ?? [])
                .filter((p) => p.ativo)
                .map((p) => ({ value: p.id, label: p.nome }))}
            />
            <ScopePicker
              idPrefix="ud-scope"
              value={scope}
              onChange={setScope}
              permission="iam:manage"
            />
            <div className="flex justify-end">
              <Button type="submit" size="sm" loading={pending}>
                <Plus size={14} aria-hidden="true" className="mr-1" /> Adicionar lotação
              </Button>
            </div>
          </form>
        )}
      </section>
    </div>
  );
}

interface RelatorioSSO {
  lidos: number;
  criados: number;
  atualizados: number;
  desativados: number;
  /** Os que o próprio SSO tinha desativado e voltaram. */
  reativados?: number;
  conflitos: string[];
}

interface EstadoSSO {
  andamento: boolean;
  ultima?: {
    em: string;
    batida_em: string;
    terminada_em?: string;
    relatorio: Partial<RelatorioSSO>;
    erro?: string;
  };
}

function resumoSSO(r: Partial<RelatorioSSO>): string {
  const conflitos = r.conflitos ?? [];
  return (
    `${r.lidos ?? 0} pessoas lidas, ${r.criados ?? 0} novas, ${r.atualizados ?? 0} atualizadas, ` +
    `${r.desativados ?? 0} desativadas` +
    (r.reativados ? `, ${r.reativados} reativadas.` : ".") +
    (conflitos.length > 0
      ? ` Com conta local de mesmo usuário ou e-mail (vincule à mão): ${conflitos.join(", ")}.`
      : "")
  );
}

/**
 * Traz as pessoas do SSO (ADR 077); o worker repete todo dia. Num realm
 * federado ao AD leva de minutos a horas: o pedido só começa, cada página
 * lida já fica gravada, e a tela acompanha a sincronização mais recente,
 * seja ela da tela ou do worker.
 */
function SincronizarSSO({ onFeito }: { onFeito: () => void }) {
  const { run, pending } = useAction();
  const [pedida, setPedida] = useState(false);
  const rodando = useRef(false);
  const estado = useApiQuery<EstadoSSO>("v1/iam/sso/sincronizacao", {
    refreshInterval: (e) => (e?.andamento || pedida ? 3000 : 0),
    onSuccess: (e) => {
      if (e.andamento) {
        rodando.current = true;
      } else if (rodando.current) {
        rodando.current = false;
        setPedida(false);
        onFeito();
      }
    },
  });
  const andamento = pedida || !!estado.data?.andamento;
  const ultima = estado.data?.ultima;

  async function sincronizar() {
    if (await run(() => apiClient.post("v1/iam/sso/sincronizar"))) {
      setPedida(true);
      rodando.current = true;
      void estado.mutate();
    }
  }

  return (
    <div className="flex flex-col gap-1">
      <Button variant="secondary" onClick={() => void sincronizar()} loading={pending || andamento}>
        <RefreshCw size={16} aria-hidden="true" className="mr-1" /> Sincronizar com o SSO
      </Button>
      {andamento ? (
        <p role="status" className="text-xs text-muted">
          Sincronizando com o SSO
          {ultima && !ultima.terminada_em
            ? ` desde ${fmtDateTime(ultima.em)}: até agora ${resumoSSO(ultima.relatorio)}`
            : "."}{" "}
          Pode levar algumas horas; quem já foi lido já está gravado.
        </p>
      ) : (
        ultima && (
          <p role="status" className={`text-xs ${ultima.erro ? "text-danger" : "text-muted"}`}>
            {ultima.erro
              ? `A última sincronização (${fmtDateTime(ultima.em)}) parou com erro: ${ultima.erro}. Antes disso: `
              : `Última sincronização em ${fmtDateTime(ultima.em)}: `}
            {resumoSSO(ultima.relatorio)}
          </p>
        )
      )}
    </div>
  );
}

export default function UsuariosPage() {
  const { can } = usePlataforma();
  const [q, setQ] = useState("");
  const [query, setQuery] = useState("");
  const [active, setActive] = useState("");
  const [page, setPage] = useState(1);
  const [creating, setCreating] = useState(false);
  const [selected, setSelected] = useState<UserAdmin | null>(null);
  const list = useApiPage<UserAdmin>(
    withQuery("v1/users", { q: query, active, page, page_size: 25 }),
  );

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <form
          role="search"
          className="flex flex-wrap items-end gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            setPage(1);
            setQuery(q.trim());
          }}
        >
          <Input
            id="users-q"
            label="Buscar"
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="nome, usuário ou e-mail"
          />
          <Select
            id="users-active"
            label="Situação"
            value={active}
            onChange={(e) => {
              setPage(1);
              setActive(e.target.value);
            }}
            options={[
              { value: "", label: "Todas" },
              { value: "true", label: "Ativas" },
              { value: "false", label: "Inativas" },
            ]}
          />
          <Button type="submit" variant="secondary">
            <Search size={14} aria-hidden="true" className="mr-1" /> Buscar
          </Button>
        </form>
        {can("users:manage") && (
          <div className="flex flex-wrap items-start gap-2">
            <SincronizarSSO onFeito={() => void list.mutate()} />
            <Button onClick={() => setCreating(true)}>
              <Plus size={16} aria-hidden="true" className="mr-1" /> Conta local
            </Button>
          </div>
        )}
      </div>

      <DataState
        loading={list.isLoading}
        error={list.error}
        onRetry={() => void list.mutate()}
        empty={(list.data?.items ?? []).length === 0}
        emptyTitle="Nenhum usuário encontrado"
      >
        <Table caption="Usuários da plataforma">
          <TableHead>
            <TableRow>
              <TableHeaderCell>Nome</TableHeaderCell>
              <TableHeaderCell>Origem</TableHeaderCell>
              <TableHeaderCell>Situação</TableHeaderCell>
              <TableHeaderCell>Último acesso</TableHeaderCell>
              <TableHeaderCell>
                <span className="sr-only">Ações</span>
              </TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {(list.data?.items ?? []).map((u) => (
              <TableRow key={u.id}>
                <TableCell>
                  <span className="font-medium">{u.display_name || u.username}</span>
                  <span className="block text-xs text-muted">
                    {u.username} · {u.email}
                  </span>
                </TableCell>
                <TableCell>
                  {u.federated ? <Badge tone="info">AD / Keycloak</Badge> : <Badge>Local</Badge>}
                </TableCell>
                <TableCell>
                  {u.active ? (
                    <Badge tone="success">Ativa</Badge>
                  ) : (
                    <Badge tone="danger">Inativa</Badge>
                  )}
                  {u.locked_until && new Date(u.locked_until) > new Date() && (
                    <Badge tone="warning" className="ml-1">
                      Bloqueada
                    </Badge>
                  )}
                </TableCell>
                <TableCell className="text-xs text-muted">{fmtDateTime(u.last_seen_at)}</TableCell>
                <TableCell className="text-right">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setSelected(u)}
                    aria-label={`Gerenciar ${u.username}`}
                  >
                    <UserCog size={16} aria-hidden="true" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <Pagination meta={list.data?.meta} onPage={setPage} />
      </DataState>

      <Dialog open={creating} onClose={() => setCreating(false)} title="Nova conta local" size="lg">
        {creating && (
          <CreateUserForm
            onDone={() => {
              setCreating(false);
              void list.mutate();
            }}
          />
        )}
      </Dialog>
      <Dialog
        open={selected !== null}
        onClose={() => setSelected(null)}
        title={selected ? selected.display_name || selected.username : ""}
        size="xl"
      >
        {selected && (
          <UserDetail key={selected.id} userId={selected.id} onChanged={() => void list.mutate()} />
        )}
      </Dialog>
    </div>
  );
}
