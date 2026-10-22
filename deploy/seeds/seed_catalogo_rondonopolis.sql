-- ==============================================================================
-- PROJETO NOVA — CATÁLOGO DE SERVIÇOS CORPORATIVOS E TI (RONDONÓPOLIS - MT)
-- ==============================================================================
-- Garante o alinhamento estrito com o escopo planejado (26 soluções de TI / Infraestrutura)
-- e remove serviços departamentais adicionados fora do escopo original.
-- Associa os serviços de tecnologia ao Hub Aurora (Unidade de Inovação & P&D).
-- Idempotente: pode rodar a qualquer momento.
-- ==============================================================================

BEGIN;

DO $$
DECLARE
    v_u_aurora UUID := '20000000-0000-0000-0000-000000000002'; -- Hub Aurora (P&D / TI)
BEGIN
    -- 1. Remoção de serviços fora do escopo inicial de tecnologia e infraestrutura
    DELETE FROM catalog_services
    WHERE slug IN (
        'reserva-salas-auditorio',
        'parecer-juridico-contratos',
        'acesso-sso-vpn',
        'suporte-apis-agrotech',
        'vistoria-frotas-bitrem',
        'requisicao-almoxarifado-central',
        'cadastro-cooperativas-agro',
        'emissao-certidoes-institucionais',
        'pesagem-balanca-ferroviaria',
        'agendamento-descarga-graos'
    );

    -- 2. Garantir que as soluções de tecnologia estejam associadas ao Hub Aurora
    UPDATE catalog_services
    SET responsible_unidade_id = v_u_aurora
    WHERE responsible_unidade_id IS NULL;

END $$;

COMMIT;
