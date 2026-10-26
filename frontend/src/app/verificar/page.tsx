import type { Metadata } from "next";
import { ShieldCheck } from "lucide-react";

import { PublicShell } from "@/components/layout/PublicShell";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";

import { verificar } from "./actions";

import { VerificarClientTabs } from "@/components/public/VerificarClientTabs";

export const metadata: Metadata = {
  title: "Verificar assinatura",
  description: "Confira a autenticidade de um documento assinado eletronicamente.",
};

export default async function VerificarIndex({
  searchParams,
}: {
  searchParams: Promise<{ erro?: string }>;
}) {
  const { erro } = await searchParams;
  return (
    <PublicShell>
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-6 px-6 py-14">
        <div className="flex items-center gap-3">
          <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <ShieldCheck size={28} aria-hidden="true" />
          </div>
          <div>
            <span className="text-xs font-semibold uppercase tracking-wider text-primary">
              Autenticidade de Documentos Oficiais
            </span>
            <h1 className="text-2xl font-bold tracking-tight">Verificar assinatura eletrônica</h1>
          </div>
        </div>
        <p className="text-sm text-muted">
          Informe o código de verificação impresso no documento assinado, aponte a câmera para o QR
          Code ou selecione o arquivo PDF para validar a integridade.
        </p>

        <VerificarClientTabs>
          <form
            action={verificar}
            className="flex flex-col gap-4 rounded-xl border border-surface-border bg-surface-raised p-5"
          >
            <Input
              id="codigo"
              name="codigo"
              label="Código de verificação"
              required
              placeholder="00000000-0000-0000-0000-000000000000"
              error={erro ? "Código inválido — confira os 36 caracteres." : undefined}
              autoComplete="off"
            />
            <div className="flex items-center justify-between pt-1">
              <span className="text-[11px] text-muted">
                Exemplo: 550e8400-e29b-41d4-a716-446655440000
              </span>
              <Button type="submit">Verificar</Button>
            </div>
          </form>
        </VerificarClientTabs>
      </div>
    </PublicShell>
  );
}
