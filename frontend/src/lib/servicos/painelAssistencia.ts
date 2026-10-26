/** Ordem e cartões ocultos do dashboard da assistência, neste navegador. */

export interface Disposicao {
  ordem: string[];
  ocultos: string[];
}

export type EstiloGrafico = "barras" | "horizontal" | "rosca" | "linha" | "area";

export const ESTILOS_GRAFICO: { id: EstiloGrafico; label: string }[] = [
  { id: "barras", label: "Barras" },
  { id: "horizontal", label: "Barras deitadas" },
  { id: "rosca", label: "Rosca" },
  { id: "linha", label: "Linha" },
  { id: "area", label: "Área" },
];

const ESTILOS = new Set<string>(ESTILOS_GRAFICO.map((item) => item.id));

export function dispor(ids: string[], salvo: Disposicao | null): Disposicao {
  const conhecidos = new Set(ids);
  const ordem = (salvo?.ordem ?? []).filter((id) => conhecidos.has(id));
  for (const id of ids) {
    if (!ordem.includes(id)) ordem.push(id);
  }
  const ocultos = (salvo?.ocultos ?? []).filter((id) => conhecidos.has(id));
  return { ordem, ocultos };
}

export function mover(disposicao: Disposicao, id: string, direcao: -1 | 1): Disposicao {
  const visiveis = disposicao.ordem.filter((item) => !disposicao.ocultos.includes(item));
  const indice = visiveis.indexOf(id);
  const destino = indice + direcao;
  if (indice < 0 || destino < 0 || destino >= visiveis.length) return disposicao;
  const ordem = [...disposicao.ordem];
  const de = ordem.indexOf(visiveis[indice]!);
  const para = ordem.indexOf(visiveis[destino]!);
  const atual = ordem[de]!;
  ordem[de] = ordem[para]!;
  ordem[para] = atual;
  return { ...disposicao, ordem };
}

export function ocultar(disposicao: Disposicao, id: string): Disposicao {
  if (!disposicao.ordem.includes(id) || disposicao.ocultos.includes(id)) return disposicao;
  return { ...disposicao, ocultos: [...disposicao.ocultos, id] };
}

export function mostrar(disposicao: Disposicao, id: string): Disposicao {
  return { ...disposicao, ocultos: disposicao.ocultos.filter((item) => item !== id) };
}

export function estiloGraficoDe(valor: string | null): EstiloGrafico {
  if (valor && ESTILOS.has(valor)) return valor as EstiloGrafico;
  return "barras";
}
