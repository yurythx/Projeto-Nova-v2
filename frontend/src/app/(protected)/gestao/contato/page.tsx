"use client";

import { Eye } from "lucide-react";
import { useState, type FormEvent } from "react";

import { UnidadeDonaSelect } from "@/components/iam/UnidadeDonaSelect";
import { DataState } from "@/components/sismarq/DataState";
import { PageHeader } from "@/components/sismarq/PageHeader";
import { Pagination } from "@/components/sismarq/Pagination";
import { fmtDateTime, useAction } from "@/components/sismarq/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Dialog } from "@/components/ui/Dialog";
import { Select } from "@/components/ui/Select";
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from "@/components/ui/Table";
import { Textarea } from "@/components/ui/Textarea";
import { apiClient } from "@/lib/api/client";
import { useApiPage, useApiQuery, withQuery } from "@/lib/api/swr";
import { useSismarq } from "@/lib/sismarq/SismarqProvider";
import type { ContactMessage, ContactSummary, OrgTree } from "@/lib/sismarq/types";

const STATUS: Record<ContactSummary["status"], { label: string; tone: "info" | "warning" | "success" | "neutral" }> = {
  new: { label: "Nova", tone: "info" },
  in_progress: { label: "Em atendimento", tone: "warning" },
  answered: { label: "Respondida", tone: "success" },
  archived: { label: "Arquivada", tone: "neutral" },
};

function MessageDetail({ id, onChanged }: { id: string; onChanged: () => void }) {
  const { can } = useSismarq();
  const msg = useApiQuery<ContactMessage>(`v1/contact/messages/${id}`);
  const { run, pending } = useAction();
  const m = msg.data;
  if (!m) return <DataState loading={msg.isLoading} error={msg.error} empty={false}>{null}</DataState>;

  async function triage(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const ok = await run(
      () => apiClient.patch(`v1/contact/messages/${id}`, {
        status: String(fd.get("status")),
        notes: String(fd.get("notes") ?? "").trim(),
        assigned_to: m!.assigned_to ?? null,
        unidade_id: String(fd.get("unidade_id") ?? "") || null,
      }),
      "Mensagem atualizada",
    );
    if (ok) {
      void msg.mutate();
      onChanged();
    }
  }

  return (
    <div className="flex flex-col gap-4 text-sm">
      <dl className="grid gap-2 sm:grid-cols-2">
        <div><dt className="text-xs text-muted">Protocolo</dt><dd className="font-mono break-all">{m.protocol}</dd></div>
        <div><dt className="text-xs text-muted">Recebida</dt><dd>{fmtDateTime(m.created_at)}</dd></div>
        <div><dt className="text-xs text-muted">Nome</dt><dd className="break-all">{m.name}</dd></div>
        <div><dt className="text-xs text-muted">E-mail</dt><dd className="break-all"><a href={`mailto:${m.email}?subject=${encodeURIComponent(`Re: ${m.subject} [${m.protocol}]`)}`} className="text-primary hover:underline">{m.email}</a></dd></div>
        {m.phone && <div><dt className="text-xs text-muted">Telefone</dt><dd>{m.phone}</dd></div>}
        <div><dt className="text-xs text-muted">Consentimento LGPD</dt><dd>{fmtDateTime(m.consent_at)}</dd></div>
      </dl>
      <div>
        <p className="text-xs text-muted">Assunto</p>
        <p className="font-medium">{m.subject}</p>
      </div>
      <p className="whitespace-pre-line rounded-lg bg-surface-hover p-3">{m.message}</p>
      {can("contact:manage") ? (
        <form onSubmit={triage} className="flex flex-col gap-3 border-t border-surface-border pt-4">
          <Select id="ct-status" name="status" label="Situação" defaultValue={m.status} options={Object.entries(STATUS).map(([value, s]) => ({ value, label: s.label }))} />
          {/* Encaminhar (ADR 013): a gestão com escopo só encaminha dentro da
              área dela e sempre pode devolver à caixa geral. */}
          <UnidadeDonaSelect id="ct-unidade" label="Setor (encaminhar para)" permission="contact:manage" optional defaultValue={m.unidade_id} placeholder="Caixa geral (triagem)" />
          <Textarea id="ct-notes" name="notes" label="Anotações internas" rows={3} maxLength={5000} defaultValue={m.notes} />
          <Button type="submit" className="self-end" loading={pending}>
            Salvar triagem
          </Button>
        </form>
      ) : (
        m.notes && <p className="text-xs text-muted">Anotações: {m.notes}</p>
      )}
    </div>
  );
}

export default function GestaoContatoPage() {
  const [status, setStatus] = useState("new");
  const [page, setPage] = useState(1);
  const [open, setOpen] = useState<ContactSummary | null>(null);
  const list = useApiPage<ContactSummary>(withQuery("v1/contact/messages", { status, page, page_size: 25 }));
  const tree = useApiQuery<OrgTree[]>("v1/iam/org-tree");
  const setor = (id?: string) => (id && (tree.data ?? []).flatMap((e) => e.unidades).find((u) => u.id === id)?.nome) || (id ? "—" : "Caixa geral");

  return (
    <div className="flex flex-col gap-6">
      <PageHeader eyebrow="Contato" title="Mensagens recebidas" description="Mensagens do formulário público do Arquivo (SIC/contato), com protocolo e consentimento LGPD." />
      <Select
        id="ct-filter"
        label="Situação"
        value={status}
        onChange={(e) => {
          setPage(1);
          setStatus(e.target.value);
        }}
        options={[{ value: "", label: "Todas" }, ...Object.entries(STATUS).map(([value, s]) => ({ value, label: s.label }))]}
        className="w-full sm:max-w-xs"
      />
      <DataState loading={list.isLoading} error={list.error} onRetry={() => void list.mutate()} empty={(list.data?.items ?? []).length === 0} emptyTitle="Nenhuma mensagem">
        <div className="flex items-center justify-between text-[11px] text-muted sm:hidden px-1 mb-1">
          <span>↔ Deslize a tabela para ver todas as colunas</span>
        </div>
        <Table caption="Mensagens de contato" className="min-w-[720px]">
          <TableHead>
            <TableRow>
              <TableHeaderCell className="w-28">Protocolo</TableHeaderCell>
              <TableHeaderCell className="min-w-[160px]">Assunto</TableHeaderCell>
              <TableHeaderCell className="min-w-[140px]">Remetente</TableHeaderCell>
              <TableHeaderCell className="min-w-[130px]">Setor</TableHeaderCell>
              <TableHeaderCell className="w-28">Situação</TableHeaderCell>
              <TableHeaderCell className="w-36">Recebida</TableHeaderCell>
              <TableHeaderCell className="w-12 text-right">
                <span className="sr-only">Abrir</span>
              </TableHeaderCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {(list.data?.items ?? []).map((m) => (
              <TableRow key={m.id}>
                <TableCell className="font-mono text-xs whitespace-nowrap">{m.protocol}</TableCell>
                <TableCell className="min-w-[160px]">
                  <span className="font-medium text-foreground">{m.subject}</span>
                  <span className="block text-xs text-muted">{m.category}</span>
                </TableCell>
                <TableCell className="text-sm whitespace-nowrap">{m.name}</TableCell>
                <TableCell className="text-sm whitespace-nowrap">{setor(m.unidade_id)}</TableCell>
                <TableCell className="whitespace-nowrap">
                  <Badge tone={STATUS[m.status].tone}>{STATUS[m.status].label}</Badge>
                </TableCell>
                <TableCell className="text-xs text-muted whitespace-nowrap">{fmtDateTime(m.created_at)}</TableCell>
                <TableCell className="text-right w-12">
                  <Button variant="ghost" size="sm" aria-label={`Abrir ${m.protocol}`} onClick={() => setOpen(m)}>
                    <Eye size={16} aria-hidden="true" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
        <Pagination meta={list.data?.meta} onPage={setPage} />
      </DataState>
      <Dialog open={open !== null} onClose={() => setOpen(null)} title={open ? `Mensagem ${open.protocol}` : ""} size="lg">
        {open && <MessageDetail key={open.id} id={open.id} onChanged={() => void list.mutate()} />}
      </Dialog>
    </div>
  );
}
