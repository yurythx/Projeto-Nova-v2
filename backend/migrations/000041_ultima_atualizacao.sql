-- +goose Up
-- Última atualização do chamado (ADR 080): a lista ordena por ela, como o
-- GLPI. O gatilho passa a respeitar o updated_at dado de propósito (a
-- carga grava a data do GLPI); sem ele, continua sendo agora.
-- +goose StatementBegin
CREATE FUNCTION public.set_updated_at_se_omitido() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF NEW.updated_at IS NOT DISTINCT FROM OLD.updated_at THEN
        NEW.updated_at = now();
    END IF;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

DROP TRIGGER trg_atendimento_solicitacoes_updated ON public.atendimento_solicitacoes;
CREATE TRIGGER trg_atendimento_solicitacoes_updated BEFORE UPDATE ON public.atendimento_solicitacoes
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at_se_omitido();

-- Os chamados já importados ficaram com a hora da carga. Os que ninguém
-- mexeu desde então voltam para a data da última mudança no GLPI, que
-- está na área de preparo (a versão do item).
UPDATE atendimento_solicitacoes s SET updated_at = x.em
  FROM (SELECT i.destino, i.created_at AS carga,
               max(it.versao::timestamp AT TIME ZONE c.fuso) AS em
          FROM atendimento_importacao i
          JOIN integracoes_itens it ON it.tipo = 'chamado' AND it.id_origem::text = i.id_origem
          JOIN integracoes_conexoes c ON c.id = it.conexao_id
         WHERE i.origem = 'glpi' AND i.tipo = 'solicitacao'
           AND it.versao ~ '^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$'
         GROUP BY i.destino, i.created_at) x
 WHERE s.id = x.destino AND s.updated_at = x.carga;

-- +goose Down
DROP TRIGGER trg_atendimento_solicitacoes_updated ON public.atendimento_solicitacoes;
CREATE TRIGGER trg_atendimento_solicitacoes_updated BEFORE UPDATE ON public.atendimento_solicitacoes
    FOR EACH ROW EXECUTE FUNCTION public.set_updated_at();
DROP FUNCTION public.set_updated_at_se_omitido();
