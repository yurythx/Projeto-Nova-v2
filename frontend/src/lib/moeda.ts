/** O domínio grava centavos. A tela mostra reais. */
export function reais(centavos: number): string {
  return (centavos / 100).toLocaleString("pt-BR", { style: "currency", currency: "BRL" });
}

/** Converte um valor digitado em reais para centavos. Texto inválido vale zero. */
export function centavosDeReais(texto: string): number {
  const limpo = texto.trim().replace(/\s/g, "").replace(/\./g, "").replace(",", ".");
  if (!/^\d+(\.\d{1,2})?$/.test(limpo)) return 0;
  const [inteiro, frac = ""] = limpo.split(".");
  return Number(inteiro) * 100 + Number((frac + "00").slice(0, 2));
}
