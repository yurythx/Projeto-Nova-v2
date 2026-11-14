export const CHAVE_GOVBR = "fazenda.govbr";

const HOSTS = new Set(["sso.acesso.gov.br", "sso.staging.acesso.gov.br"]);

/** Só o Login Único oficial pode receber o navegador. */
export function destinoGovBR(url: string): string | null {
  try {
    const alvo = new URL(url);
    if (alvo.protocol !== "https:" || !HOSTS.has(alvo.hostname)) return null;
    return alvo.toString();
  } catch {
    return null;
  }
}

export function guardarDesafio(state: string, verifier: string) {
  sessionStorage.setItem(CHAVE_GOVBR, JSON.stringify({ state, verifier }));
}

export function lerDesafio(): { state: string; verifier: string } | null {
  const bruto = sessionStorage.getItem(CHAVE_GOVBR);
  if (!bruto) return null;
  try {
    const dado = JSON.parse(bruto) as { state?: string; verifier?: string };
    if (!dado.state || !dado.verifier) return null;
    return { state: dado.state, verifier: dado.verifier };
  } catch {
    return null;
  }
}

export function limparDesafio() {
  sessionStorage.removeItem(CHAVE_GOVBR);
}

export function irPara(url: string) {
  window.location.assign(url);
}
