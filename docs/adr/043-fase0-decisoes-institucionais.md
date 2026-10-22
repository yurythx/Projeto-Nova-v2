# 043 — Fase 0: decisões institucionais (sem inventar ato)

- **Status:** **Parcialmente superado** pelo [ADR 046](046-saida-do-atlas-e-do-acervo.md) (a parte do Atlas/Acervo saiu do código). Antes: Aceito (registro do que a Prefeitura respondeu; o que não existe continua ausente)
- **Data:** 2026-10-16
- **Relacionado:** [FASE0.md](../acervo/FASE0.md), [HOMOLOGACAO.md](../acervo/HOMOLOGACAO.md), ADRs [040](040-pades-icp-brasil-adiado.md), [041](041-acervo-ritos-35-36-signum-e-copia.md), [042](042-ingresso-sei-adiado.md)

## Contexto

A Fase 0 pedia decreto, portarias, certificado ICP-Brasil, URL do SEI e
RIPD. Inventar qualquer um desses no git fingiria um ato que o município
não tem. As respostas humanas fecharam o que dá para fechar; o restante
segue a lei municipal e a federal, com fallback já previsto no roadmap.

## Decisão

### IAM (Keycloak / LDAP)

1. **Piloto:** SEMAD (Secretaria Municipal de Administração, Gestão de
   Pessoas e Inovação). A primeira Relação de Eliminação sai de uma
   unidade produtora desse órgão.
2. **`acervo:central` (global):** Superintendência de Arquivo Público —
   quem recebe o acervo recolhido, autoriza eliminação ordinária, cadastro
   de locais, comissões e recolhimento. Logins e grupos vêm do Keycloak
   (federação LDAP). Não há lista nominal no git.
3. **`acervo:semad` (global):** perfil da SEMAD (não do Arquivo) que
   aprova os arts. 35 e 36 e recebe a cópia do Termo (art. 38). Sem
   destinatário, `termo_copia_semad` fica 0 — o Termo não é bloqueado.
4. **`acervo:avaliacao`:** permissão no **perfil**, escopo no **órgão**.
   Os **membros** da CSPAD/CCPAD são **usuários** nomeados só quando
   existir portaria (art. 31). Sem portaria não se inventa composição.

### Direito aplicável

5. **Não há decreto regulamentador municipal** além da Lei 10.301/2019.
   O sistema opera a lei municipal e respeita a legislação federal
   (Lei 8.159/1991; Decreto 10.278/2020; Código e TTDD de atividades-meio
   do CONARQ). Os Anexos I a IV são os da lei. O art. 33 fala em “deste
   Decreto” porque o modelo foi redigido como se um decreto republicasse
   os anexos: esse decreto **não existe**; não alteramos o Anexo IV.
6. **Regimento Interno (art. 47)** ainda não existe. A tela de normas
   (ADR 036) já recebe a IN/regimento quando for publicado.
7. **Comissões (art. 31):** cadastro em `/acervo/comissoes` depois das
   portarias. Até lá a Relação piloto só anda se a Superintendência
   cadastrar a CSPAD da SEMAD com a portaria real.
8. **Atividades-meio (art. 26):** o município **adota** as classes 01 a
   07 do CONARQ. O Atlas explica o porquê (o art. 26 manda usar o código
   de atividades-meio da Administração Pública). A carga dos códigos
   entra pela tela “Atualizar TTDD”, sem inventar tabela municipal.
9. **Art. 36:** rito `ART_36` só para órgão **sem** TTDD de atividades-fim
   no Atlas. Os órgãos já publicados em `deploy/ttdd/ttdd.json` usam
   `ORDINARIO` ou `ART_35`. Não há lista inventada de secretarias.
10. **Nome nos documentos gerados:** Superintendência de Arquivo Público
    (nome atual da TTDD 2025). A lei continua falando em Divisão; a FAQ
    registra os dois.
11. **Edital:** Diário Oficial do Município, veículo **DIORONDON-e**
    (art. 37 prevalece sobre o “Diário do Estado” do Anexo III).
12. **“Enquanto vigorar”:** quem cadastra informa a **data do evento**
    comprovada (fim de contrato, de gestão, de vínculo etc., conforme a
    série). A TTDD pode ser republicada: série revogada não apaga a
    unidade (ADR 019).
13. **Digitalização:** o Arquivo Municipal digitaliza. Pasta SMB/FTP
    ainda será definida. **Não há** certificado A1/A3/HSM: `pades` e
    `ata_signum`/`termo_signum` continuam `AUSENTE`. A porta `AssinadorICP`
    existe no domínio e **não é composta** até o certificado (ADR 040).
    O padrão a seguir, quando o certificado existir, é o Decreto 10.278.
14. **SEI:** envelope pronto (`ProcessoSEI` / `UnidadeDoSEI`); **sem**
    cliente HTTP, URL nem mapeamento de órgãos (ADR 042). O Trâmite já
    ingressa (ADR 039).
15. **RIPD:** o DPO ainda não começou. Vale o modelo
    [`RIPD_MODELO.md`](../RIPD_MODELO.md).
16. **Volume:** passivo grande (cerca de 70 anos de cidade, ~300 mil
    habitantes); sem metros lineares nem política de backup medida.

## Consequências

- Homologação C1–C5 continua reunião com a Superintendência
  ([HOMOLOGACAO.md](../acervo/HOMOLOGACAO.md)); o git não marca “bateu”.
- IAM: mapear grupos LDAP em Configurações → Mapeamento AD.
- Aprovação da TTDD de atividades-fim (art. 32, §§ 4º a 8º), capacitação
  e ato do art. 64 ficam fora desta decisão.
