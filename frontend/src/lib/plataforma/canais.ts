/** Configuração do e-mail dos avisos (ADR 078). A senha nunca vem. */
export interface EmailCanal {
  ativo: boolean;
  host: string;
  porta: number;
  usuario: string;
  tem_senha: boolean;
  remetente: string;
  origem: "tela" | "ambiente" | "nenhuma";
  updated_at?: string;
  updated_by?: string;
}

/** Onde a pessoa recebe os avisos. */
export interface Contato {
  email: string;
  whatsapp: string;
  respondido: boolean;
  respondido_em?: string;
  email_do_sso?: string;
}

/** +5566999991234 → +55 (66) 99999-1234, para mostrar. */
export function formatarWhatsApp(numero: string): string {
  const m = /^\+55(\d{2})(\d{4,5})(\d{4})$/.exec(numero);
  return m ? `+55 (${m[1]}) ${m[2]}-${m[3]}` : numero;
}

/** Configuração do WhatsApp dos avisos (ADR 078). As credenciais nunca vêm. */
export interface WhatsAppCanal {
  ativo: boolean;
  provedor: "meta" | "evolution";
  meta_numero_id: string;
  tem_token: boolean;
  meta_modelo: string;
  meta_idioma: string;
  evolution_url: string;
  evolution_instancia: string;
  tem_chave: boolean;
  /** Entrada (fase 4): a resposta pelo WhatsApp vira comentário. */
  receber: boolean;
  webhook_url?: string;
  tem_app_secret: boolean;
  updated_at?: string;
  updated_by?: string;
}
