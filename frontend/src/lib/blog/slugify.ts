/**
 * Gera um identificador único e consistente para âncoras de cabeçalho
 * no Markdown e no Sumário (Table of Contents).
 */
export function slugifyHeading(text: string): string {
  return text
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "") // remove acentos
    .replace(/[^\w\s-]/g, "") // remove caracteres especiais
    .trim()
    .replace(/\s+/g, "-");
}
