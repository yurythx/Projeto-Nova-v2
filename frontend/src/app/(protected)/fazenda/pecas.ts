/** Peças do Anexo II. A lista muda com o programa e a entrada; tipo fora dela não entra. */
export const NOMES: Record<string, string> = {
  identificacao: "Identificação",
  instrumento: "Instrumento",
  comprovante_endereco: "Comprovante de endereço",
  prova_posse: "Prova de posse",
  prova_data: "Prova da data",
  matricula: "Matrícula",
  cessao: "Cessão",
  procuracao: "Procuração",
  documento_unico_regularidade: "Documento único de regularidade",
  orcamento_cronograma: "Orçamento e cronograma",
  quadro_metas: "Quadro de metas",
  documento_pessoal: "Documento pessoal",
  declaracao_nao_inicio: "Declaração de não início",
};

export interface OpcoesPecas {
  programa: string;
  entrada: string;
  itbi: boolean;
  moduloC: boolean;
  temCessao: boolean;
  temContrato: boolean;
}

/** Espelha a lista legal do domínio. Portaria não acrescenta tipo. */
export function pecasLegais(op: OpcoesPecas): string[] {
  const itens: string[] = [];
  if (op.programa === "avanca") {
    itens.push(
      "documento_unico_regularidade",
      "matricula",
      "orcamento_cronograma",
      "quadro_metas",
      "documento_pessoal",
      "declaracao_nao_inicio",
    );
    if (op.itbi) itens.push("instrumento");
  } else {
    itens.push("identificacao", "instrumento", "comprovante_endereco");
    if (op.temCessao) itens.push("cessao");
    if (op.moduloC) itens.push("prova_data", "matricula");
    if (!op.temContrato) itens.push("prova_posse");
  }
  if (op.entrada === "procurador") itens.push("procuracao");
  return itens;
}
