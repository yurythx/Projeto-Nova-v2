import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { CheckCircle2, ShieldCheck, XCircle } from "lucide-react";

import { PublicShell } from "@/components/layout/PublicShell";
import { VerifyDocument } from "@/components/public/VerifyDocument";
import { PublicApiError, publicGet } from "@/lib/api/publicServer";
import type { Verification } from "@/lib/plataforma/types";

import { CertificateActions } from "@/components/public/CertificateActions";

export const metadata: Metadata = { title: "Verificação de assinatura", robots: { index: false } };

const STATUS: Record<Verification["status"], string> = {
  pending: "Aguardando assinaturas",
  completed: "Todas as assinaturas concluídas",
  refused: "Recusado por um signatário",
  cancelled: "Cancelado",
};

export default async function VerificarPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  if (!/^[0-9a-f-]{36}$/i.test(id)) notFound();
  let v: Verification;
  try {
    v = (await publicGet<Verification>(`signum/verify/${id}`, 0)).data;
  } catch (err) {
    if (err instanceof PublicApiError && err.status === 404) notFound();
    throw err;
  }

  const isCompleted = v.status === "completed";

  return (
    <PublicShell>
      <div className="mx-auto flex w-full max-w-3xl flex-col gap-6 px-6 py-12">
        {/* Banner de Autenticidade Institucional */}
        <div
          className={`flex items-start gap-4 rounded-2xl border p-5 transition-all ${
            isCompleted
              ? "border-success/30 bg-success/10 text-success"
              : v.status === "pending"
                ? "border-amber-500/30 bg-amber-500/10 text-amber-600 dark:text-amber-400"
                : "border-danger/30 bg-danger/10 text-danger"
          }`}
        >
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-surface/80 shadow-xs">
            <ShieldCheck size={28} aria-hidden="true" className="text-current" />
          </div>
          <div className="flex-1">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-xs font-bold uppercase tracking-wider">
                Certidão de Autenticidade Digital
              </span>
              <span className="rounded-full bg-surface/90 px-2.5 py-0.5 text-xs font-semibold shadow-2xs">
                Lei Federal 14.063/2020
              </span>
            </div>
            <p className="dateline mt-1 text-xs opacity-90">Verificação de autenticidade</p>
            <h1 className="text-xl font-bold tracking-tight text-foreground sm:text-2xl">{v.title}</h1>
          </div>
        </div>

        {/* Barra de Ações (Imprimir, Copiar, QR Code) */}
        <CertificateActions
          envelopeId={v.envelope_id}
          documentSha256={v.document_sha256}
          title={v.title}
        />

        {/* Metadados Criptográficos */}
        <dl className="grid gap-3 rounded-xl border border-surface-border bg-surface-raised p-5 text-sm">
          <div>
            <dt className="text-xs text-muted">Situação</dt>
            <dd className="font-medium text-foreground">{STATUS[v.status]}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted">Código Verificador (UUID)</dt>
            <dd className="break-all font-mono text-xs select-all text-primary font-semibold">{v.envelope_id}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted">SHA-256 do documento</dt>
            <dd className="break-all font-mono text-xs select-all">{v.document_sha256}</dd>
          </div>
          <div>
            <dt className="text-xs text-muted">Consultado em</dt>
            <dd className="font-medium">{new Date(v.checked_at).toLocaleString("pt-BR")}</dd>
          </div>
        </dl>

        {/* Quadro de Signatários */}
        <section aria-labelledby="assinaturas" className="flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <h2 id="assinaturas" className="text-lg font-semibold">
              Assinaturas
            </h2>
            <span className="text-xs text-muted">
              {v.signatures.length} {v.signatures.length === 1 ? "signatário" : "signatários"}
            </span>
          </div>

          <ul className="flex flex-col gap-2.5">
            {v.signatures.map((s, i) => (
              <li
                key={i}
                className="flex items-center justify-between gap-3 rounded-xl border border-surface-border bg-surface p-4 text-sm shadow-2xs"
              >
                <div className="flex flex-col gap-0.5">
                  <span className="font-medium text-foreground">{s.name}</span>
                  {s.signed_at && (
                    <span className="block text-xs text-muted">
                      {new Date(s.signed_at).toLocaleString("pt-BR")} · {s.method}
                    </span>
                  )}
                </div>

                <div className="shrink-0">
                  {s.status === "signed" ? (
                    s.valid ? (
                      <span className="flex items-center gap-1.5 rounded-full bg-success/15 px-2.5 py-1 text-xs font-semibold text-success">
                        <CheckCircle2 size={16} aria-hidden="true" /> Válida
                      </span>
                    ) : (
                      <span className="flex items-center gap-1.5 rounded-full bg-danger/15 px-2.5 py-1 text-xs font-semibold text-danger">
                        <XCircle size={16} aria-hidden="true" /> Inválida
                      </span>
                    )
                  ) : (
                    <span className="rounded-full bg-surface-subtle px-2.5 py-1 text-xs font-medium text-muted">
                      {s.status === "refused" ? "Recusou" : "Pendente"}
                    </span>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </section>

        {/* Verificador de Arquivo Local */}
        <div className="rounded-xl border border-surface-border bg-surface-raised p-5 print:hidden">
          <div className="mb-3 flex flex-col gap-1">
            <h3 className="font-semibold text-foreground">Conferência de Integridade do Arquivo</h3>
            <p className="text-xs text-muted">
              Envie o arquivo PDF original em sua posse para confirmar que ele não sofreu nenhuma alteração desde que foi assinado.
            </p>
          </div>
          <VerifyDocument envelopeId={v.envelope_id} />
        </div>

        {/* Respaldo Jurídico para Auditoria */}
        <div className="rounded-xl border border-surface-border/60 bg-surface-subtle/50 p-4 text-xs text-muted leading-relaxed">
          <p>
            <strong>Nota de fé pública:</strong> Este registro atesta que o documento com o hash criptográfico acima foi submetido à cerimônia de assinatura eletrônica do Projeto Nova, com carimbo de tempo e custódia na trilha de auditoria.
          </p>
        </div>
      </div>
    </PublicShell>
  );
}
