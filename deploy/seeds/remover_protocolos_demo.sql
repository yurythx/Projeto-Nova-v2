-- ==============================================================================
-- Remove os dados fictícios do Protocolo Digital (seed_protocolos_demo.sql):
-- os seis serviços de teste com todos os pedidos abertos neles (inclusive os
-- de teste feitos à mão), histórico, agendamentos, os formulários, os
-- responsáveis técnicos e as faltas dos CPFs fictícios.
--
-- Não toca em serviço, formulário ou pedido criado de verdade.
-- ==============================================================================

BEGIN;

CREATE TEMP TABLE demo_servicos ON COMMIT DROP AS
SELECT unnest(ARRAY[
    'd0c0de00-0000-4000-8000-000000005e01', 'd0c0de00-0000-4000-8000-000000005e02',
    'd0c0de00-0000-4000-8000-000000005e03', 'd0c0de00-0000-4000-8000-000000005e04',
    'd0c0de00-0000-4000-8000-000000005e05', 'd0c0de00-0000-4000-8000-000000005e06']::uuid[]) AS id;

-- Faltas: só CPFs que aparecem exclusivamente em pedidos dos serviços de teste.
DELETE FROM protocolos_faltas f
 WHERE f.cpf IN (SELECT cpf FROM protocolos_pedidos WHERE servico_id IN (SELECT id FROM demo_servicos))
   AND NOT EXISTS (SELECT 1 FROM protocolos_pedidos p
                    WHERE p.cpf = f.cpf AND p.servico_id NOT IN (SELECT id FROM demo_servicos));

-- Histórico e agendamentos caem em cascata.
DELETE FROM protocolos_pedidos WHERE servico_id IN (SELECT id FROM demo_servicos);
DELETE FROM protocolos_servicos WHERE id IN (SELECT id FROM demo_servicos);
DELETE FROM protocolos_formularios WHERE id IN (
    'd0c0de00-0000-4000-8000-00000000f001', 'd0c0de00-0000-4000-8000-00000000f002',
    'd0c0de00-0000-4000-8000-00000000f003', 'd0c0de00-0000-4000-8000-00000000f004',
    'd0c0de00-0000-4000-8000-00000000f005', 'd0c0de00-0000-4000-8000-00000000f006');
DELETE FROM protocolos_responsaveis WHERE id IN (
    md5('protocolo-demo-rt-1')::uuid, md5('protocolo-demo-rt-2')::uuid,
    md5('protocolo-demo-rt-3')::uuid, md5('protocolo-demo-rt-4')::uuid);

SELECT (SELECT count(*) FROM protocolos_servicos) AS servicos_restantes,
       (SELECT count(*) FROM protocolos_pedidos) AS pedidos_restantes;

COMMIT;
