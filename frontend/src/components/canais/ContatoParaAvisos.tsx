"use client";

import { type FormEvent } from "react";

import { useAction } from "@/components/plataforma/useAction";
import { Button } from "@/components/ui/Button";
import { Dialog } from "@/components/ui/Dialog";
import { Input } from "@/components/ui/Input";
import { apiClient } from "@/lib/api/client";
import { useApiQuery } from "@/lib/api/swr";
import { formatarWhatsApp, type Contato } from "@/lib/plataforma/canais";

/** Formulário do contato para avisos: e-mail e WhatsApp (ADR 078). */
export function FormularioDeContato({
  contato,
  onSalvo,
  recusar,
}: {
  contato?: Contato;
  onSalvo: (c: Contato) => void;
  /** Mostra "Agora não", que grava a resposta vazia. */
  recusar?: boolean;
}) {
  const { run, pending } = useAction();

  async function salvar(corpo: { email: string; whatsapp: string }) {
    const salvo = await run(() => apiClient.put<Contato>("v1/me/contato", corpo), "Contato salvo");
    if (salvo) onSalvo(salvo.data);
  }

  function enviar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    void salvar({
      email: String(fd.get("email") ?? "").trim(),
      whatsapp: String(fd.get("whatsapp") ?? "").trim(),
    });
  }

  return (
    <form onSubmit={enviar} aria-label="Contato para avisos" className="grid gap-3">
      <Input
        id="contato-email"
        name="email"
        type="email"
        label="E-mail"
        maxLength={254}
        autoComplete="email"
        defaultValue={contato?.email || contato?.email_do_sso || ""}
        placeholder="nome@exemplo.gov.br"
      />
      <Input
        id="contato-whatsapp"
        name="whatsapp"
        type="tel"
        label="WhatsApp"
        maxLength={32}
        autoComplete="tel"
        inputMode="tel"
        defaultValue={contato?.whatsapp ? formatarWhatsApp(contato.whatsapp) : ""}
        placeholder="(66) 99999-9999"
      />
      <p className="text-xs text-muted">
        Usamos só para avisar sobre os seus chamados: confirmação, respostas da equipe, conclusão e
        a pesquisa de satisfação. Você pode mudar ou apagar quando quiser, no seu perfil.
      </p>
      <div className="flex flex-wrap gap-2">
        <Button type="submit" loading={pending}>
          Salvar contato
        </Button>
        {recusar && (
          <Button
            type="button"
            variant="secondary"
            disabled={pending}
            onClick={() => void salvar({ email: "", whatsapp: "" })}
          >
            Agora não
          </Button>
        )}
      </div>
    </form>
  );
}

/** Na primeira visita à central, pede o contato a quem ainda não respondeu. */
export function PedidoDeContato() {
  const contato = useApiQuery<Contato>("v1/me/contato");
  const aberto = !!contato.data && !contato.data.respondido;
  return (
    <Dialog
      open={aberto}
      onClose={() => void contato.mutate({ ...contato.data!, respondido: true }, false)}
      title="Como quer receber os avisos?"
      description="Deixe um e-mail ou WhatsApp para saber quando o seu chamado andar, sem precisar entrar no sistema."
    >
      <FormularioDeContato
        contato={contato.data}
        recusar
        onSalvo={(c) => void contato.mutate(c, false)}
      />
    </Dialog>
  );
}
