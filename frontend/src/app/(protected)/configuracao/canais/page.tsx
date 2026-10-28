"use client";

import { CheckCircle2, Mail, Send, XCircle } from "lucide-react";
import { type FormEvent, useState } from "react";

import { WhatsAppCard } from "@/components/canais/WhatsAppCard";
import { DataState } from "@/components/plataforma/DataState";
import { PageHeader } from "@/components/plataforma/PageHeader";
import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Toggle } from "@/components/ui/Toggle";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import type { EmailCanal, WhatsAppCanal } from "@/lib/plataforma/canais";

const ORIGEM: Record<EmailCanal["origem"], string> = {
  tela: "Configurado aqui",
  ambiente: "Vindo das variáveis de ambiente",
  nenhuma: "Ainda não configurado",
};

function EmailCard({ email, onSalvo }: { email: EmailCanal; onSalvo: () => void }) {
  const { run, pending } = useAction();
  const [ativo, setAtivo] = useState(email.ativo);
  const [removerSenha, setRemoverSenha] = useState(false);
  const [para, setPara] = useState("");
  const [teste, setTeste] = useState<{ ok: boolean; erro?: string } | null>(null);
  const testando = useAction();

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const ok = await run(
      () =>
        apiClient.put<EmailCanal>("v1/canais/email", {
          ativo,
          host: String(fd.get("host") ?? "").trim(),
          porta: Number(fd.get("porta") || 587),
          usuario: String(fd.get("usuario") ?? "").trim(),
          senha: String(fd.get("senha") ?? ""),
          remover_senha: removerSenha,
          remetente: String(fd.get("remetente") ?? "").trim(),
        }),
      "Configuração do e-mail salva",
    );
    if (ok) {
      setRemoverSenha(false);
      onSalvo();
    }
  }

  async function testar() {
    setTeste(null);
    const r = await testando.run(() =>
      apiClient.post<{ ok: boolean; erro?: string }>("v1/canais/email/testar", {
        para: para.trim(),
      }),
    );
    if (r) setTeste(r.data);
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2" className="flex flex-wrap items-center gap-2">
          <Mail size={18} aria-hidden="true" /> E-mail
          <Badge tone={email.ativo ? "success" : "neutral"}>
            {email.ativo ? "Ligado" : "Desligado"}
          </Badge>
          <Badge tone="info">{ORIGEM[email.origem]}</Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-5 pb-5">
        <p className="text-sm text-muted">
          Quem pediu recebe por e-mail a confirmação do chamado com o link de acompanhamento, as
          respostas da equipe, as mudanças de estado, a conclusão e o pedido de avaliação.
          Desligado, os avisos continuam no sino.
          {email.updated_at &&
            ` Alterado em ${fmtDateTime(email.updated_at)}${email.updated_by ? ` por ${email.updated_by}` : ""}.`}
        </p>
        <form
          onSubmit={salvar}
          aria-label="Servidor de e-mail"
          className="grid gap-3 sm:grid-cols-2"
        >
          <div className="flex items-center gap-3 sm:col-span-2">
            <Toggle checked={ativo} onChange={setAtivo} label="Enviar avisos por e-mail" />
            <span className="text-sm">Enviar avisos por e-mail</span>
          </div>
          <Input
            id="email-host"
            name="host"
            label="Servidor SMTP"
            placeholder="smtp.prefeitura.gov.br"
            maxLength={255}
            defaultValue={email.host}
            autoComplete="off"
          />
          <Input
            id="email-porta"
            name="porta"
            type="number"
            label="Porta (587 com STARTTLS, 465 com TLS direto)"
            min={1}
            max={65535}
            defaultValue={email.porta || 587}
          />
          <Input
            id="email-usuario"
            name="usuario"
            label="Usuário"
            maxLength={255}
            defaultValue={email.usuario}
            autoComplete="off"
          />
          <div className="flex flex-col gap-1">
            <Input
              id="email-senha"
              name="senha"
              type="password"
              label="Senha"
              maxLength={500}
              autoComplete="new-password"
              placeholder={email.tem_senha ? "Gravada — deixe em branco para manter" : ""}
              disabled={removerSenha}
            />
            {email.tem_senha && (
              <label className="flex items-center gap-2 text-xs">
                <input
                  type="checkbox"
                  checked={removerSenha}
                  onChange={(e) => setRemoverSenha(e.target.checked)}
                  className="h-4 w-4 accent-primary"
                />
                Apagar a senha gravada
              </label>
            )}
          </div>
          <div className="sm:col-span-2">
            <Input
              id="email-remetente"
              name="remetente"
              label="Remetente"
              maxLength={255}
              placeholder="Central de Atendimento <atendimento@prefeitura.gov.br>"
              defaultValue={email.remetente}
              autoComplete="off"
            />
          </div>
          <div className="sm:col-span-2">
            <Button type="submit" loading={pending}>
              Salvar
            </Button>
          </div>
        </form>

        <section
          aria-labelledby="teste-titulo"
          className="flex flex-col gap-3 border-t border-surface-border pt-4"
        >
          <h3 id="teste-titulo" className="text-sm font-semibold">
            Testar o envio
          </h3>
          <p className="text-xs text-muted">
            Usa a configuração salva, ligada ou não: confira antes de ligar.
          </p>
          <div className="flex flex-wrap items-end gap-2">
            <div className="min-w-64 flex-1">
              <Input
                id="email-teste"
                type="email"
                label="Mandar o teste para"
                value={para}
                onChange={(e) => setPara(e.target.value)}
                placeholder="voce@prefeitura.gov.br"
              />
            </div>
            <Button
              type="button"
              variant="secondary"
              loading={testando.pending}
              disabled={!para.trim()}
              onClick={() => void testar()}
            >
              <Send size={16} aria-hidden="true" className="mr-1" /> Enviar teste
            </Button>
          </div>
          {teste && (
            <p
              role="status"
              className={`flex items-start gap-2 text-sm ${teste.ok ? "text-success" : "text-danger"}`}
            >
              {teste.ok ? (
                <CheckCircle2 size={16} aria-hidden="true" className="mt-0.5" />
              ) : (
                <XCircle size={16} aria-hidden="true" className="mt-0.5" />
              )}
              {teste.ok
                ? "Mensagem enviada. Confira a caixa de entrada (e o lixo eletrônico)."
                : `O servidor recusou: ${teste.erro}`}
            </p>
          )}
        </section>
      </CardContent>
    </Card>
  );
}

/** Canais dos avisos (ADR 078): e-mail e WhatsApp, cada um liga e desliga. */
export default function CanaisPage() {
  const email = useApiQuery<EmailCanal>("v1/canais/email");
  const whatsapp = useApiQuery<WhatsAppCanal>("v1/canais/whatsapp");
  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        eyebrow="Configurações"
        title="Canais de aviso"
        description="Por onde os avisos saem do sistema para quem pediu atendimento. Cada canal liga e desliga aqui; desligado, o aviso fica só no sino."
      />
      <DataState
        loading={email.isLoading}
        error={email.error}
        onRetry={() => void email.mutate()}
        empty={false}
        rows={4}
      >
        {email.data && (
          <EmailCard
            key={email.data.updated_at ?? "novo"}
            email={email.data}
            onSalvo={() => void email.mutate()}
          />
        )}
      </DataState>
      <DataState
        loading={whatsapp.isLoading}
        error={whatsapp.error}
        onRetry={() => void whatsapp.mutate()}
        empty={false}
        rows={4}
      >
        {whatsapp.data && (
          <WhatsAppCard
            key={whatsapp.data.updated_at ?? "novo"}
            whatsapp={whatsapp.data}
            onSalvo={() => void whatsapp.mutate()}
          />
        )}
      </DataState>
    </div>
  );
}
