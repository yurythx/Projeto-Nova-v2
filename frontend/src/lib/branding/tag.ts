/** Tag do cache de dados do Next para GET /branding (layout, metadados e
 * páginas públicas). Salvar o branding a expira na hora — ver
 * components/settings/brandingActions.ts. */
export const BRANDING_TAG = "branding";

/** Quanto o Next guarda GET /branding entre gravações. Salvar na tela
 * expira na hora pela tag; este prazo só cobre o intervalo. */
export const BRANDING_REVALIDATE_SECONDS = 300;
