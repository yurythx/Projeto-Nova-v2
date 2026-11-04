import "server-only";

import { DEFAULT_BRANDING } from "@/components/branding/brandingConfig";
import { publicGetOr } from "@/lib/api/publicServer";
import { BRANDING_REVALIDATE_SECONDS, BRANDING_TAG } from "@/lib/branding/tag";
import type { Branding } from "@/lib/plataforma/types";

export interface ServerBranding {
  appName: string;
  appDescription: string;
  orgName: string;
  supportEmail: string;
  supportPhone: string;
}

/** Identidade white-label (GET /branding) para Server Components — nenhum
 * texto institucional fica fixo no código. O cache dura
 * BRANDING_REVALIDATE_SECONDS; salvar na tela expira na hora. */
export async function getServerBranding(): Promise<ServerBranding> {
  const b = await publicGetOr<Branding | null>("branding", null, BRANDING_REVALIDATE_SECONDS, [BRANDING_TAG]);
  return {
    appName: b?.app_name || DEFAULT_BRANDING.appName,
    appDescription: b?.app_description || "",
    orgName: b?.org_name || DEFAULT_BRANDING.orgName,
    supportEmail: b?.support_email || "",
    supportPhone: b?.support_phone || "",
  };
}
