-- +goose Up
-- Nome público: Projeto Nova. Só troca o padrão de fábrica ainda gravado.
-- Uma instalação que já personalizou o nome permanece como está.

UPDATE branding_settings
   SET app_name = 'Projeto Nova', updated_at = now()
 WHERE app_name = 'Projeto-nova-v2';

UPDATE branding_settings
   SET org_name = 'Projeto Nova', updated_at = now()
 WHERE org_name = 'Projeto-nova-v2';

UPDATE branding_settings
   SET app_description = 'Gestão de produtos e serviços no ITIL 5: incidente, requisição, problema, mudança e projeto.',
       updated_at = now()
 WHERE app_description = 'Microkernel de plug-ins: identidade, auditoria e módulos que nascem desligados.';

-- +goose Down
UPDATE branding_settings
   SET app_name = 'Projeto-nova-v2', updated_at = now()
 WHERE app_name = 'Projeto Nova';

UPDATE branding_settings
   SET org_name = 'Projeto-nova-v2', updated_at = now()
 WHERE org_name = 'Projeto Nova';

UPDATE branding_settings
   SET app_description = 'Microkernel de plug-ins: identidade, auditoria e módulos que nascem desligados.',
       updated_at = now()
 WHERE app_description = 'Gestão de produtos e serviços no ITIL 5: incidente, requisição, problema, mudança e projeto.';
