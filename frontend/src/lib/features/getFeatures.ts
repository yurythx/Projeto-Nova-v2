import "server-only";

import { publicGetOr } from "@/lib/api/publicServer";
import type { PublicModule } from "@/lib/plataforma/types";

/** Estado dos módulos com superfície pública (Kernel), para o site
 * institucional esconder seções de plugins desativados. O painel
 * autenticado vê a troca na hora (poll de /system/modules). O site
 * público espera até 5 minutos: 15s fazia esta rota dominar o log da API. */
export async function getFeatures(): Promise<{ features: Record<string, boolean> }> {
  const modules = await publicGetOr<PublicModule[]>("system/public-modules", [], 300);
  const features: Record<string, boolean> = {};
  for (const m of modules) features[m.key] = m.enabled;
  return { features };
}
