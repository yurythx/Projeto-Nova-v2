"use client";

import { CheckCircle2, Copy, MessageCircle, Send, ShieldAlert, XCircle } from "lucide-react";
import { type FormEvent, useState } from "react";

import { fmtDateTime, useAction } from "@/components/plataforma/useAction";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Toggle } from "@/components/ui/Toggle";
import { apiClient } from "@/lib/api/client";
import type { WhatsAppCanal } from "@/lib/plataforma/canais";

const PROVEDORES = [
  { value: "meta", label: "API oficial da Meta (recomendada)" },
  { value: "evolution", label: "Evolution API (não oficial)" },
];

/** Credencial gravada: em branco mantém; a caixa apaga. */
function Segredo({
  id,
  nome,
  rotulo,
  gravado,
  remover,
  onRemover,
}: {
  id: string;
  nome: string;
  rotulo: string;
  gravado: boolean;
  remover: boolean;
  onRemover: (v: boolean) => void;
}) {
  return (
    <div className="flex flex-col gap-1">
      <Input
        id={id}
        name={nome}
        type="password"
        label={rotulo}
        maxLength={1000}
        autoComplete="new-password"
        placeholder={gravado ? "Gravado — deixe em branco para manter" : ""}
        disabled={remover}
      />
      {gravado && (
        <label className="flex items-center gap-2 text-xs">
          <input
            type="checkbox"
            checked={remover}
            onChange={(e) => onRemover(e.target.checked)}
            className="h-4 w-4 accent-primary"
          />
          Apagar {rotulo.toLowerCase()} gravado
        </label>
      )}
    </div>
  );
}

/** WhatsApp dos avisos (ADR 078): Meta ou Evolution, um ativo de cada vez. */
export function WhatsAppCard({
  whatsapp,
  onSalvo,
}: {
  whatsapp: WhatsAppCanal;
  onSalvo: () => void;
}) {
  const { run, pending } = useAction();
  const testando = useAction();
  const [ativo, setAtivo] = useState(whatsapp.ativo);
  const [receber, setReceber] = useState(whatsapp.receber);
  const [novoWebhook, setNovoWebhook] = useState(false);
  const [copiado, setCopiado] = useState(false);
  const [provedor, setProvedor] = useState(whatsapp.provedor);
  const [removerToken, setRemoverToken] = useState(false);
  const [removerChave, setRemoverChave] = useState(false);
  const [para, setPara] = useState("");
  const [teste, setTeste] = useState<{ ok: boolean; erro?: string } | null>(null);

  async function salvar(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    const texto = (n: string) => String(fd.get(n) ?? "").trim();
    // Os campos do provedor escondido vêm do que está gravado.
    const ok = await run(
      () =>
        apiClient.put<WhatsAppCanal>("v1/canais/whatsapp", {
          ativo,
          provedor,
          meta_numero_id: provedor === "meta" ? texto("meta_numero_id") : whatsapp.meta_numero_id,
          meta_token: provedor === "meta" ? String(fd.get("meta_token") ?? "") : "",
          remover_token: removerToken,
          meta_modelo: provedor === "meta" ? texto("meta_modelo") : whatsapp.meta_modelo,
          meta_idioma: provedor === "meta" ? texto("meta_idioma") : whatsapp.meta_idioma,
          evolution_url: provedor === "evolution" ? texto("evolution_url") : whatsapp.evolution_url,
          evolution_instancia:
            provedor === "evolution" ? texto("evolution_instancia") : whatsapp.evolution_instancia,
          evolution_chave: provedor === "evolution" ? String(fd.get("evolution_chave") ?? "") : "",
          remover_chave: removerChave,
          receber,
          meta_app_secret: provedor === "meta" ? String(fd.get("meta_app_secret") ?? "") : "",
          novo_webhook: novoWebhook,
        }),
      "Configuração do WhatsApp salva",
    );
    if (ok) {
      setRemoverToken(false);
      setRemoverChave(false);
      setNovoWebhook(false);
      onSalvo();
    }
  }

  async function copiar(endereco: string) {
    try {
      await navigator.clipboard.writeText(endereco);
      setCopiado(true);
    } catch {
      setCopiado(false);
    }
  }

  async function testar() {
    setTeste(null);
    const r = await testando.run(() =>
      apiClient.post<{ ok: boolean; erro?: string }>("v1/canais/whatsapp/testar", {
        para: para.trim(),
      }),
    );
    if (r) setTeste(r.data);
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle as="h2" className="flex flex-wrap items-center gap-2">
          <MessageCircle size={18} aria-hidden="true" /> WhatsApp
          <Badge tone={whatsapp.ativo ? "success" : "neutral"}>
            {whatsapp.ativo ? "Ligado" : "Desligado"}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="flex flex-col gap-5 pb-5">
        <p className="text-sm text-muted">
          Os mesmos avisos do e-mail, para quem deixou o número na central ou no perfil.
          {whatsapp.updated_at &&
            ` Alterado em ${fmtDateTime(whatsapp.updated_at)}${whatsapp.updated_by ? ` por ${whatsapp.updated_by}` : ""}.`}
        </p>
        <form
          onSubmit={salvar}
          aria-label="Provedor do WhatsApp"
          className="grid gap-3 sm:grid-cols-2"
        >
          <div className="flex items-center gap-3 sm:col-span-2">
            <Toggle checked={ativo} onChange={setAtivo} label="Enviar avisos pelo WhatsApp" />
            <span className="text-sm">Enviar avisos pelo WhatsApp</span>
          </div>
          <div className="sm:col-span-2">
            <Select
              id="whatsapp-provedor"
              label="Provedor"
              value={provedor}
              onChange={(e) => setProvedor(e.target.value as WhatsAppCanal["provedor"])}
              options={PROVEDORES}
            />
          </div>
          {provedor === "meta" ? (
            <>
              <Input
                id="whatsapp-numero"
                name="meta_numero_id"
                label="Identificador do número (Phone number ID)"
                inputMode="numeric"
                maxLength={32}
                defaultValue={whatsapp.meta_numero_id}
                autoComplete="off"
              />
              <Segredo
                id="whatsapp-token"
                nome="meta_token"
                rotulo="Token"
                gravado={whatsapp.tem_token}
                remover={removerToken}
                onRemover={setRemoverToken}
              />
              <Input
                id="whatsapp-modelo"
                name="meta_modelo"
                label="Modelo aprovado"
                placeholder="aviso_chamado"
                maxLength={512}
                defaultValue={whatsapp.meta_modelo}
                autoComplete="off"
              />
              <Input
                id="whatsapp-idioma"
                name="meta_idioma"
                label="Idioma do modelo"
                maxLength={10}
                defaultValue={whatsapp.meta_idioma || "pt_BR"}
                autoComplete="off"
              />
              <p className="text-xs text-muted sm:col-span-2">
                Fora da janela de 24 horas, a Meta só entrega mensagens de modelo aprovado. Crie no
                WhatsApp Manager um modelo de utilidade com três variáveis no corpo, nesta ordem:
                título, mensagem e link. Exemplo: “{"{{1}}"} — {"{{2}}"}. Acompanhe: {"{{3}}"}”.
              </p>
            </>
          ) : (
            <>
              <p
                role="note"
                className="flex items-start gap-2 rounded-lg border border-warning/40 bg-warning/10 p-3 text-sm sm:col-span-2"
              >
                <ShieldAlert size={16} aria-hidden="true" className="mt-0.5 shrink-0" />A Evolution
                usa um WhatsApp comum conectado pelo QR code. Isso fere os termos da Meta, e o
                número pode ser bloqueado. Use um número só para os avisos.
              </p>
              <Input
                id="whatsapp-evolution-url"
                name="evolution_url"
                label="Endereço da Evolution API"
                placeholder="http://evolution:8080"
                maxLength={500}
                defaultValue={whatsapp.evolution_url}
                autoComplete="off"
              />
              <Input
                id="whatsapp-instancia"
                name="evolution_instancia"
                label="Instância"
                maxLength={100}
                defaultValue={whatsapp.evolution_instancia}
                autoComplete="off"
              />
              <Segredo
                id="whatsapp-chave"
                nome="evolution_chave"
                rotulo="Chave"
                gravado={whatsapp.tem_chave}
                remover={removerChave}
                onRemover={setRemoverChave}
              />
            </>
          )}
          <fieldset className="grid gap-3 rounded-lg border border-surface-border p-3 sm:col-span-2">
            <legend className="px-1 text-sm font-medium">Respostas pelo WhatsApp</legend>
            <div className="flex items-center gap-3">
              <Toggle
                checked={receber}
                onChange={setReceber}
                label="Receber respostas pelo WhatsApp"
              />
              <span className="text-sm">
                Receber respostas: a mensagem de quem pediu vira comentário no chamado
              </span>
            </div>
            {provedor === "meta" && (
              <Input
                id="whatsapp-app-secret"
                name="meta_app_secret"
                type="password"
                label="App Secret (confere a assinatura das mensagens)"
                maxLength={200}
                autoComplete="new-password"
                placeholder={whatsapp.tem_app_secret ? "Gravado — deixe em branco para manter" : ""}
              />
            )}
            {whatsapp.webhook_url ? (
              <div className="flex flex-col gap-2 text-sm">
                <span className="font-medium">Endereço do webhook</span>
                <div className="flex flex-wrap items-center gap-2">
                  <code className="break-all rounded bg-black/5 px-2 py-1 text-xs dark:bg-white/10">
                    {whatsapp.webhook_url}
                  </code>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={() => void copiar(whatsapp.webhook_url!)}
                  >
                    <Copy size={14} aria-hidden="true" className="mr-1" />
                    {copiado ? "Copiado" : "Copiar"}
                  </Button>
                </div>
                <p className="text-xs text-muted">
                  {provedor === "meta"
                    ? "No app da Meta, em WhatsApp > Configuração, cadastre este endereço como URL de callback, use o token do fim do endereço como token de verificação e assine o campo messages."
                    : "Na instância da Evolution, cadastre este endereço como webhook com o evento MESSAGES_UPSERT."}{" "}
                  O endereço é secreto: quem o tiver pode mandar mensagens em nome do provedor.
                </p>
                <label className="flex items-center gap-2 text-xs">
                  <input
                    type="checkbox"
                    checked={novoWebhook}
                    onChange={(e) => setNovoWebhook(e.target.checked)}
                    className="h-4 w-4 accent-primary"
                  />
                  Gerar outro endereço ao salvar (o atual deixa de valer)
                </label>
              </div>
            ) : (
              receber && (
                <p className="text-xs text-muted">
                  O endereço do webhook aparece aqui depois de salvar.
                </p>
              )
            )}
          </fieldset>
          <div className="sm:col-span-2">
            <Button type="submit" loading={pending}>
              Salvar
            </Button>
          </div>
        </form>

        <section
          aria-labelledby="teste-whatsapp-titulo"
          className="flex flex-col gap-3 border-t border-surface-border pt-4"
        >
          <h3 id="teste-whatsapp-titulo" className="text-sm font-semibold">
            Testar o envio
          </h3>
          <div className="flex flex-wrap items-end gap-2">
            <div className="min-w-64 flex-1">
              <Input
                id="whatsapp-teste"
                type="tel"
                label="Mandar o teste para"
                value={para}
                onChange={(e) => setPara(e.target.value)}
                placeholder="(66) 99999-9999"
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
                ? "Mensagem enviada. Confira o WhatsApp."
                : `O provedor recusou: ${teste.erro}`}
            </p>
          )}
        </section>
      </CardContent>
    </Card>
  );
}
