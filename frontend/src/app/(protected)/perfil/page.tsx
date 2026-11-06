"use client";

import { Download, ShieldCheck, UserX } from "lucide-react";
import { useRef, useState, type ChangeEvent, type FormEvent } from "react";

import { FormularioDeContato } from "@/components/canais/ContatoParaAvisos";
import { AvatarPessoa, avisarAvatar } from "@/components/plataforma/AvatarPessoa";
import { ScopePicker } from "@/components/iam/ScopePicker";
import { DataState } from "@/components/plataforma/DataState";
import { ConfirmButton } from "@/components/plataforma/ConfirmButton";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { mimeOf, putToTicket } from "@/lib/plataforma/upload";
import type { UploadTicket } from "@/lib/plataforma/types";
import { useApiQuery } from "@/lib/api/swr";
import type { Contato } from "@/lib/plataforma/canais";
import { usePlataforma } from "@/lib/plataforma/PlataformaProvider";
import type { Person, ScopeRef } from "@/lib/plataforma/types";

interface DSR {
  id: string;
  kind: string;
  status: string;
  detail: string;
  created_at: string;
  completed_at?: string | null;
}

function FotoDoPerfil() {
  const { me } = usePlataforma();
  const arquivo = useRef<HTMLInputElement>(null);
  const { run, pending } = useAction();
  const [erro, setErro] = useState("");

  async function escolher(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file || !me?.id) return;
    if (!file.type.startsWith("image/") || file.size > 2 * 1024 * 1024) {
      setErro("Use uma imagem de até 2 MB.");
      return;
    }
    setErro("");
    await run(async () => {
      const ticket = await apiClient.post<UploadTicket>("v1/directory/me/avatar/uploads", {
        filename: file.name || "foto.png",
        content_type: mimeOf(file),
      });
      await putToTicket(ticket.data, file);
      await apiClient.put("v1/directory/me/avatar", { object_key: ticket.data.object_key });
      avisarAvatar(me.id);
    }, "Foto do perfil atualizada");
  }

  async function remover() {
    if (!me?.id) return;
    setErro("");
    await run(async () => {
      await apiClient.delete("v1/directory/me/avatar");
      avisarAvatar(me.id);
    }, "Foto do perfil removida");
  }

  return (
    <div className="flex flex-wrap items-center gap-4">
      <AvatarPessoa id={me?.id} nome={me?.name || me?.username} tamanho="md" />
      <div className="flex flex-col gap-2">
        <p className="text-sm text-muted">
          Esta foto aparece nas listas e nos quadros. Sem foto, o círculo mostra as iniciais.
        </p>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="secondary" loading={pending} onClick={() => arquivo.current?.click()}>
            Alterar foto
          </Button>
          <Button type="button" variant="ghost" disabled={pending} onClick={() => void remover()}>
            Remover
          </Button>
        </div>
        {erro && <p className="text-xs text-danger">{erro}</p>}
        <input
          ref={arquivo}
          type="file"
          accept="image/png,image/jpeg,image/webp,image/gif"
          className="sr-only"
          aria-label="Escolher foto do perfil"
          onChange={(e) => void escolher(e)}
        />
      </div>
    </div>
  );
}

function DirectoryProfile() {
  const person = useApiQuery<Person>("v1/directory/me");
  const { run, pending } = useAction();
  const [scope, setScope] = useState<ScopeRef | null>(null);
  const p = person.data;
  if (!p)
    return (
      <p className="text-sm text-muted">
        {person.error ? "Perfil do Diretório indisponível." : "Carregando…"}
      </p>
    );

  const current: ScopeRef = scope ?? {
    unidade_id: p.unidade_id,
    departamento_id: p.departamento_id,
  };

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    await run(
      () =>
        apiClient.put("v1/directory/me", {
          job_title: String(fd.get("job_title") ?? "").trim(),
          phone: String(fd.get("phone") ?? "").trim(),
          extension: String(fd.get("extension") ?? "").trim(),
          bio: String(fd.get("bio") ?? "").trim(),
          visible: fd.get("visible") === "on",
          unidade_id: current.unidade_id ?? null,
          departamento_id: current.departamento_id ?? null,
        }),
      "Perfil do Diretório atualizado",
    );
    void person.mutate();
  }

  return (
    <form onSubmit={submit} className="flex flex-col gap-3">
      {p.from_ad && (
        <p className="text-xs text-muted">
          Lotação sincronizada do Active Directory — alterações manuais podem ser sobrescritas na
          próxima sincronização.
        </p>
      )}
      <div className="grid gap-3 sm:grid-cols-3">
        <Input
          id="dir-job"
          name="job_title"
          label="Cargo / função"
          maxLength={120}
          defaultValue={p.job_title}
        />
        <Input id="dir-phone" name="phone" label="Telefone" maxLength={30} defaultValue={p.phone} />
        <Input
          id="dir-ext"
          name="extension"
          label="Ramal"
          maxLength={15}
          defaultValue={p.extension}
        />
      </div>
      <ScopePicker idPrefix="dir-scope" value={current} onChange={setScope} />
      <Textarea
        id="dir-bio"
        name="bio"
        label="Sobre você"
        rows={3}
        maxLength={1000}
        defaultValue={p.bio}
      />
      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          name="visible"
          defaultChecked={p.visible}
          className="h-4 w-4 accent-primary"
        />{" "}
        Aparecer no Diretório interno
      </label>
      <div className="flex justify-end">
        <Button type="submit" loading={pending}>
          Salvar
        </Button>
      </div>
    </form>
  );
}

/** O contato em que a pessoa recebe os avisos dos chamados (ADR 078). */
function ContatoDoPerfil() {
  const contato = useApiQuery<Contato>("v1/me/contato");
  return (
    <DataState
      loading={contato.isLoading}
      error={contato.error}
      onRetry={() => void contato.mutate()}
      empty={false}
      rows={2}
    >
      {contato.data && (
        <FormularioDeContato
          key={contato.data.respondido_em ?? "sem"}
          contato={contato.data}
          onSalvo={(c) => void contato.mutate(c, false)}
        />
      )}
    </DataState>
  );
}

export default function PerfilPage() {
  const { me, enabled } = usePlataforma();
  const requests = useApiQuery<DSR[]>("v1/lgpd/minhas-solicitacoes");
  const { run } = useAction();
  const pendingErasure = (requests.data ?? []).some(
    (r) => r.kind === "erasure" && (r.status === "pending" || r.status === "processing"),
  );

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Minha conta"
        title={me?.name || me?.username || "Meu perfil"}
        description={me?.email}
      />

      <Card>
        <CardHeader>
          <CardTitle as="h2">Foto</CardTitle>
        </CardHeader>
        <CardContent className="pb-5 pt-3">
          <FotoDoPerfil />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle as="h2" className="flex items-center gap-2">
            <ShieldCheck size={18} aria-hidden="true" /> Acesso efetivo
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-3 pb-5 pt-3 text-sm">
          <p>
            Autenticado via{" "}
            <strong>
              {me?.source === "keycloak" ? "Keycloak (SSO corporativo)" : "credencial local"}
            </strong>
            .
          </p>
          <div>
            <p className="text-xs font-semibold text-muted">Lotações</p>
            <ul className="mt-1 flex flex-wrap gap-1">
              {(me?.scopes ?? []).length === 0 && <li className="text-muted">Nenhuma</li>}
              {(me?.scopes ?? []).map((s, i) => (
                <li key={i}>
                  <Badge tone={s.origem === "ad" ? "info" : "neutral"}>
                    {s.perfil} {s.origem === "ad" && "· AD"}
                  </Badge>
                </li>
              ))}
            </ul>
          </div>
          <details>
            <summary className="cursor-pointer text-xs text-muted hover:text-foreground">
              {(me?.permissions ?? []).length} permissões efetivas
            </summary>
            <p className="mt-2 font-mono text-xs leading-relaxed">
              {(me?.permissions ?? []).join(" · ")}
            </p>
          </details>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle as="h2">Contato para avisos</CardTitle>
        </CardHeader>
        <CardContent className="pb-5 pt-3">
          <ContatoDoPerfil />
        </CardContent>
      </Card>

      {enabled("directory") && (
        <Card>
          <CardHeader>
            <CardTitle as="h2">Perfil no Diretório</CardTitle>
          </CardHeader>
          <CardContent className="pb-5 pt-3">
            <DirectoryProfile />
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle as="h2">Meus dados pessoais (LGPD, art. 18)</CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col gap-4 pb-5 pt-3 text-sm">
          <div className="flex flex-wrap gap-2">
            <a
              href="/api/backend/v1/lgpd/meus-dados"
              download="meus-dados.json"
              className="inline-flex items-center gap-1 rounded-lg border border-surface-border px-4 py-2 font-medium hover:bg-surface-hover"
            >
              <Download size={16} aria-hidden="true" /> Baixar meus dados (JSON)
            </a>
            <ConfirmButton
              variant="danger"
              size="md"
              disabled={pendingErasure}
              title="Solicitar anonimização da conta?"
              description="Seus dados pessoais serão anonimizados. Registros exigidos por lei (trilha de auditoria) são preservados de forma pseudonimizada. A ação não pode ser desfeita."
              confirmLabel="Solicitar"
              onConfirm={() =>
                run(
                  () => apiClient.post("v1/lgpd/solicitar-exclusao"),
                  "Solicitação registrada",
                ).then(() => requests.mutate())
              }
            >
              <UserX size={16} aria-hidden="true" className="mr-1" /> Solicitar exclusão
            </ConfirmButton>
          </div>
          {(requests.data ?? []).length > 0 && (
            <ul className="flex flex-col divide-y divide-surface-border rounded-lg border border-surface-border">
              {(requests.data ?? []).map((r) => (
                <li
                  key={r.id}
                  className="flex flex-wrap items-center justify-between gap-2 px-3 py-2"
                >
                  <span>
                    {r.kind === "erasure" ? "Exclusão / anonimização" : r.kind} ·{" "}
                    {fmtDateTime(r.created_at)}
                  </span>
                  <Badge
                    tone={
                      r.status === "completed"
                        ? "success"
                        : r.status === "rejected"
                          ? "danger"
                          : "warning"
                    }
                  >
                    {r.status}
                  </Badge>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
