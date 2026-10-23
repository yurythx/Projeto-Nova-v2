"use client";

import { CheckCircle2, ChevronRight, HelpCircle, Lightbulb, Sparkles, X } from "lucide-react";
import { useMemo, useState } from "react";

import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Card, CardContent } from "@/components/ui/Card";

export interface GuiaAutoatendimento {
  id: string;
  palavrasChave: string[];
  titulo: string;
  resumo: string;
  passoAPasso: string[];
  categoria: string;
}

const BASE_CONHECIMENTO_SUGESTOES: GuiaAutoatendimento[] = [
  {
    id: "kb-senha",
    palavrasChave: ["senha", "expirou", "esqueci", "redefinir", "bloqueada", "login", "acesso", "bloqueio"],
    titulo: "Como redefinir sua senha de rede ou desbloquear sua conta",
    categoria: "Acesso & Contas",
    resumo: "Instruções para redefinir sua credencial institucional sem precisar aguardar atendimento.",
    passoAPasso: [
      "Acesse o portal de autoatendimento de identidade pelo link 'Esqueci minha senha' na tela de login.",
      "Informe seu CPF ou usuário de rede e responda às perguntas de segurança cadastradas.",
      "Você receberá um código temporário via SMS ou e-mail secundário cadastrado.",
      "Crie uma nova senha com no mínimo 8 caracteres (contendo maiúsculas, números e símbolos).",
      "Aguarde cerca de 2 minutos para que a replicação nos controladores de domínio seja concluída.",
    ],
  },
  {
    id: "kb-email",
    palavrasChave: ["email", "e-mail", "outlook", "correio", "mensagem", "anexo", "caixa", "postal"],
    titulo: "Problemas comuns no e-mail institucional e Webmail",
    categoria: "Comunicação",
    resumo: "Soluções rápidas para caixa cheia, sincronização do Outlook e envio de anexos.",
    passoAPasso: [
      "Verifique se sua cota de armazenamento não atingiu 100% esvaziando a pasta 'Itens Excluídos' e 'Lixo Eletrônico'.",
      "Caso o Outlook desktop apresente falha de conexão, acesse temporariamente pelo navegador via Webmail.",
      "Para mensagens travadas na Caixa de Saída, confirme se o anexo não ultrapassa o limite de 25 MB.",
      "Reinicie o cliente de e-mail segurando a tecla CTRL para iniciar em modo de segurança se houver travamento.",
    ],
  },
  {
    id: "kb-vpn",
    palavrasChave: ["vpn", "remoto", "casa", "home office", "conectar", "tunel", "teletrabalho"],
    titulo: "Guia de conexão e solução de falhas na VPN corporativa",
    categoria: "Redes & Infraestrutura",
    resumo: "Passo a passo para conectar à rede interna a partir do trabalho remoto.",
    passoAPasso: [
      "Certifique-se de que sua conexão local com a internet está ativa e estável.",
      "Abra o cliente VPN oficial homologado pelo órgão e confirme o endereço do gateway institucional.",
      "Digite suas credenciais de rede habituais e autorize o duplo fator de autenticação (MFA).",
      "Se ocorrer erro de certificado ou tempo limite, desconecte, aguarde 30 segundos e tente novamente.",
    ],
  },
  {
    id: "kb-impressora",
    palavrasChave: ["impressora", "imprimir", "toner", "papel", "fila", "spooler", "scanner", "digitalizar"],
    titulo: "Impressora não responde ou fila de impressão travada",
    categoria: "Hardware & Periféricos",
    resumo: "Como destravar o serviço de impressão e reconectar a impressora de rede do setor.",
    passoAPasso: [
      "Verifique se o cabo de rede está conectado e a impressora exibe 'Pronta' no visor LCD.",
      "No Windows, abra 'Configurações > Dispositivos > Impressoras' e clique em 'Abrir Fila de Impressão'.",
      "Cancele todos os documentos pendentes com status de 'Erro'.",
      "Abra o menu Iniciar, digite 'services.msc', localize 'Spooler de Impressão' e clique em 'Reiniciar serviço'.",
    ],
  },
  {
    id: "kb-wifi",
    palavrasChave: ["wifi", "wi-fi", "wireless", "sem fio", "rede", "conexao", "internet", "cabo"],
    titulo: "Conexão com a rede Wi-Fi corporativa e cabeada",
    categoria: "Redes & Infraestrutura",
    resumo: "Configuração do protocolo WPA2-Enterprise para servidores públicos.",
    passoAPasso: [
      "Selecione a rede Wi-Fi oficial 'GOV-CORPORATIVO' na lista de redes disponíveis.",
      "Selecione o método EAP 'PEAP' e autenticação da fase 2 'MSCHAPv2'.",
      "Insira suas credenciais institucionais e marque a opção de confiar no certificado do servidor.",
      "Para redes cabeadas, verifique se a porta física da tomada de rede está com a luz verde piscando.",
    ],
  },
  {
    id: "kb-certificado",
    palavrasChave: ["certificado", "token", "a3", "leitora", "assinador", "smartcard", "pki", "signum"],
    titulo: "Token criptográfico A3 não reconhecido pelo assinador",
    categoria: "Certificação Digital",
    resumo: "Diagnóstico de drivers e inicialização do token de assinatura digital.",
    passoAPasso: [
      "Desconecte o token USB e plugue-o em outra porta USB traseira do computador.",
      "Abra o software 'SafeNet' ou 'Administration Tool' correspondente à sua leitora para testar o PIN.",
      "Verifique se a cadeia de certificados da ICP-Brasil está instalada no repositório do Windows.",
      "Certifique-se de que o aplicativo Signum Desktop ou Assinador SERPRO está em execução na barra de tarefas.",
    ],
  },
];

export function DeflexaoConhecimento({
  termo,
  onResolvido,
}: {
  termo: string;
  onResolvido?: () => void;
}) {
  const [guiaSelecionado, setGuiaSelecionado] = useState<GuiaAutoatendimento | null>(null);
  const [defletido, setDefletido] = useState(false);

  const sugestoes = useMemo(() => {
    const limpo = termo.trim().toLowerCase();
    if (limpo.length < 3) return [];

    const palavras = limpo.split(/\s+/).filter((w) => w.length >= 3);
    if (palavras.length === 0) return [];

    return BASE_CONHECIMENTO_SUGESTOES.filter((g) => {
      return palavras.some((p) =>
        g.palavrasChave.some((k) => k.includes(p) || p.includes(k)) ||
        g.titulo.toLowerCase().includes(p)
      );
    });
  }, [termo]);

  if (defletido) {
    return (
      <div
        role="status"
        className="flex items-center gap-3 rounded-xl border border-success/40 bg-success/10 p-4 text-sm animate-in fade-in duration-300"
      >
        <CheckCircle2 className="h-6 w-6 text-success shrink-0" />
        <div className="flex-1">
          <p className="font-semibold text-foreground">Excelente! Problema resolvido com sucesso.</p>
          <p className="text-muted text-xs mt-0.5">
            Ao utilizar o autoatendimento, você poupou seu tempo e ajudou a manter a fila de suporte focada em demandas críticas.
          </p>
        </div>
      </div>
    );
  }

  if (sugestoes.length === 0) return null;

  return (
    <div
      data-testid="deflexao-conhecimento"
      className="flex flex-col gap-3 rounded-xl border border-accent/40 bg-accent/[0.04] p-4 shadow-xs"
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <Lightbulb className="h-5 w-5 text-accent animate-pulse" />
          <h3 className="text-sm font-semibold text-foreground">
            Encontramos soluções rápidas para o seu caso
          </h3>
          <Badge tone="info" className="ml-1 text-[10px]">
            Autoatendimento
          </Badge>
        </div>
        <span className="text-xs text-muted">Base de Conhecimento</span>
      </div>

      <p className="text-xs text-muted">
        Antes de aguardar na fila de atendimento, veja se um destes tutoriais resolve seu problema imediatamente:
      </p>

      <div className="grid gap-2 sm:grid-cols-2">
        {sugestoes.map((g) => (
          <div
            key={g.id}
            onClick={() => setGuiaSelecionado(g)}
            className="group flex flex-col justify-between gap-1.5 rounded-lg border border-surface-border bg-surface p-3 text-left transition-all hover:border-accent hover:shadow-xs cursor-pointer"
          >
            <div className="flex flex-col gap-1">
              <span className="text-[10px] font-medium uppercase tracking-wider text-muted">
                {g.categoria}
              </span>
              <p className="text-sm font-semibold text-foreground group-hover:text-primary transition-colors line-clamp-1">
                {g.titulo}
              </p>
              <p className="text-xs text-muted line-clamp-2">{g.resumo}</p>
            </div>
            <div className="flex items-center justify-between pt-1 border-t border-surface-border/60 text-xs font-medium text-accent">
              <span>Ver passo a passo</span>
              <ChevronRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </div>
          </div>
        ))}
      </div>

      {guiaSelecionado && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4 backdrop-blur-xs animate-in fade-in">
          <div className="flex flex-col max-h-[90vh] w-full max-w-lg rounded-xl border border-surface-border bg-surface shadow-2xl overflow-hidden">
            <div className="flex items-center justify-between border-b border-surface-border p-4">
              <div className="flex items-center gap-2">
                <Sparkles className="h-5 w-5 text-accent" />
                <div>
                  <span className="text-[10px] uppercase font-bold text-muted">
                    {guiaSelecionado.categoria}
                  </span>
                  <h4 className="text-base font-bold text-foreground line-clamp-1">
                    {guiaSelecionado.titulo}
                  </h4>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setGuiaSelecionado(null)}
                className="text-muted hover:text-foreground p-1 rounded-md"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto p-5 flex flex-col gap-4">
              <p className="text-sm text-foreground bg-accent/[0.06] border border-accent/20 rounded-lg p-3">
                {guiaSelecionado.resumo}
              </p>

              <div>
                <h5 className="text-xs font-bold uppercase tracking-wide text-muted mb-2">
                  Procedimento Recomendado:
                </h5>
                <ol className="flex flex-col gap-2.5">
                  {guiaSelecionado.passoAPasso.map((passo, idx) => (
                    <li key={idx} className="flex items-start gap-2.5 text-sm">
                      <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-accent/20 text-xs font-bold text-accent">
                        {idx + 1}
                      </span>
                      <span className="text-foreground leading-relaxed">{passo}</span>
                    </li>
                  ))}
                </ol>
              </div>
            </div>

            <div className="border-t border-surface-border bg-black/[0.02] dark:bg-white/[0.02] p-4 flex flex-wrap items-center justify-between gap-3">
              <span className="text-xs text-muted">Isso resolveu sua solicitação?</span>
              <div className="flex items-center gap-2">
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => setGuiaSelecionado(null)}
                >
                  Não resolveu, continuar chamado
                </Button>
                <Button
                  type="button"
                  size="sm"
                  onClick={() => {
                    setGuiaSelecionado(null);
                    setDefletido(true);
                    onResolvido?.();
                  }}
                  className="bg-success text-success-foreground hover:bg-success/90"
                >
                  <CheckCircle2 className="h-4 w-4 mr-1" />
                  Sim, resolveu!
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
