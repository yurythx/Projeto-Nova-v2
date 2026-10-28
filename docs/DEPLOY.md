# Deploy em servidor (Docker Compose)

Sobe a stack completa em modo produção (`APP_ENV=production`): PostgreSQL,
RabbitMQ, Redis, MinIO, migrations, API, worker e frontend. Camadas
opcionais, ligadas pelo `COMPOSE_FILE` do `.env`:

| Arquivo | O que acrescenta | Ligado por |
|---|---|---|
| `docker-compose.yml` | a stack | sempre |
| `docker-compose.keycloak.yml` | Keycloak **de teste** com usuários fictícios | `make demo-keycloak` |
| `docker-compose.https.yml` | Caddy com HTTPS (CA interna) como única entrada | `scripts/enable-https.sh` |
| `docker-compose.cloudflare.yml` | Cloudflare Tunnel de projetonova.cloud (sem portas expostas no firewall) | `make infra-deploy` (Ansible) |
| `docker-compose.ia.yml` | (opcional) IA local (Ollama, serviço `ia-local`) | acrescentar ao `COMPOSE_FILE` (ver "Inteligência artificial" abaixo) |

## Pré-requisitos

- Docker Engine + Docker Compose v2, `git`, `openssl`, `make`, `python3`
  (cenários de teste) e `jq`.
- ~4 GB de RAM (o build do frontend é a etapa mais pesada; o Keycloak de
  teste usa até 768 MB).
- Portas livres no host:
  - sem HTTPS: **3010** frontend, **8010** API/WebSocket, **9010** MinIO,
    **9011** console MinIO (e **8180** com o Keycloak de teste) — diferentes
    das de dev (3002/8002/9002/9003) para não colidir com outras stacks;
  - com HTTPS: **80**, **443**, **8443**, **9443** e **8543** (Caddy); as
    portas acima passam a escutar só no próprio servidor.

## Ordem recomendada (servidor novo)

```bash
git clone https://github.com/yurythx/Projeto-Nova-v2.git && cd Projeto-nova-v2
./scripts/deploy.sh 192.168.1.42        # 1. stack em produção (gera .env e chave)
make prod-seed-admin                     # 2. admin local (senha impressa UMA vez)
make demo-keycloak                       # 3. (teste) Keycloak + estrutura e usuários fictícios
./scripts/enable-https.sh 192.168.1.42   # 4. HTTPS com o Caddy como única entrada
make demo-popular                        # 5. (teste) dados realistas em todos os apps
make demo-test                           # 6. (teste) cenários ponta a ponta
./scripts/backup.sh --install-cron       # 7. backup diário
```

`deploy.sh` gera o `.env` a partir do `.env.example` com segredos aleatórios
(`openssl rand`), `APP_ENV=production` e as URLs públicas do host; gera
`secrets/local_auth_private_key.pem` (login local RS256); faz
`docker compose up -d --build` e espera todos os serviços ficarem `healthy`.

## Atualizar

Envie as mudanças por git e, no servidor:

```bash
make deploy        # git pull --ff-only + rebuild + restart
```

`.env`, chaves e CA são reaproveitados; migrations novas rodam sozinhas
(serviço `migrate`) antes da API e do worker.

Segredos que entraram depois são gerados na primeira atualização: hoje, a
`TYPESENSE_API_KEY` do índice da Busca Global (ADR 031). O `make deploy` e o
deploy contínuo do CI a acrescentam ao `.env` quando falta. Na primeira
subida, o worker faz a carga completa do índice; até ela terminar, a busca
usa o Postgres.

**Antes de atualizar uma implantação que já tem lotações com escopo**,
rode `make iam-scope-report` e confira o resultado: ele lista cada lotação
e mapeamento de grupo com escopo e, para cada permissão do perfil, se ela
passa a **valer só no escopo** ou **deixa de valer** (permissões de
plataforma só valem com concessão global — ADR 013). Ajuste as lotações de
quem precisa continuar com acesso amplo (concessão global) antes do deploy.

## Cuidados

- **Não apague nem regenere o `.env`** depois que houver dados: a senha do
  banco fica gravada no volume e `CONFIG_ENCRYPTION_KEY` cifra segredos
  salvos pela tela de Configurações.
- Mudou host ou porta pública? Ajuste `FRONTEND_URL`, `API_PUBLIC_URL`,
  `WEBSOCKET_PUBLIC_URL` e `MINIO_PUBLIC_URL` no `.env` e rode `make deploy`
  — as `NEXT_PUBLIC_*` são embutidas no build do frontend.
- Sem Keycloak (`KEYCLOAK_ISSUER_URL` vazio) só o login local funciona.
- **Disco:** cada `make deploy` deixa cache de build (chegou a 29 GB e
  98% do disco no servidor de teste). Rode `make prod-limpeza` de tempos em
  tempos — remove só cache sem uso há 3 dias e imagens órfãs, nunca
  volumes. Com o disco cheio, o Postgres para de gravar.
- **Instale a CA nas máquinas de teste** (`http://<host>/nova-ca.crt`):
  aceitar o aviso do navegador vale só para a página. Downloads e envios de
  arquivo (MinIO, `:9443`) são feitos por `fetch` e falham em silêncio sem
  a CA; o WebSocket já usa a mesma origem do site (`/ws`).
- **CA recriada:** se o volume `caddy_data` se perder (ex.: `down -v`), o
  Caddy gera uma CA nova. O `deploy.sh` percebe, atualiza `secrets/ca` e
  reinicia API, worker e frontend — mas cada máquina de teste precisa
  instalar a nova raiz (`http://<host>/nova-ca.crt`).
- **Não exponha a stack sem o proxy HTTPS.** Sem um proxy de borda que
  sobrescreva o `X-Forwarded-For`, o frontend não repassa o IP do visitante
  (seria forjável) e a API enxerga todo mundo com o IP do frontend: os
  limites por IP (login, contato) e a prova de consentimento LGPD perdem o
  sentido. Servido por HTTP num IP da rede, o navegador também desliga
  recursos de "contexto seguro".

## Produção no Proxmox com o domínio projetonova.cloud

O domínio público é **projetonova.cloud**, com DNS na Cloudflare. A VM não
abre porta para a internet: o túnel é o do `cloudflared` que já roda no
**LXC 101** do pve1 (192.168.1.101) e atende outras aplicações. Ele só abre conexões de
saída e chega à VM pela rede local.

| Endereço | Destino |
|---|---|
| `https://projetonova.cloud` | frontend (`/ws` vai para a API) |
| `https://api.projetonova.cloud` | API direta (health, integrações) |
| `https://arquivos.projetonova.cloud` | MinIO (URLs pré-assinadas) |
| `https://www.projetonova.cloud` | 301 para `projetonova.cloud` |

Tudo é declarado em código:

- `deploy/infra/terraform`: VM 240 (`projeto-nova-prod-app`, 192.168.1.70)
  no pve1, clonada do template 9000. **As outras VMs do pve1 (201, 210,
  220, 230) são de outras aplicações**: não reaproveite nem ligue nenhuma
  delas.
- Túnel: o do LXC 101, configurado no painel da Cloudflare (tabela abaixo).
  O LXC 101 não muda: só ganha hostnames no painel. O IP dele vai em
  `cloudflare_conector_ip` (`group_vars/all.yml`), e com ele a VM não roda
  cloudflared nem precisa de token.
- `deploy/infra/ansible`: `app_domain` em `group_vars/producao.yml` é a fonte
  das quatro URLs públicas do `.env`. Com o conector de fora, as portas da
  stack escutam no IP da VM, e a cadeia `DOCKER-USER` (em
  `/etc/ufw/after.rules`) só deixa entrar conexão nova vinda do LXC 101. O
  filtro fica ali porque as portas publicadas pelo Docker não passam pelas
  regras comuns do UFW. Qualquer outra máquina da rede que tentar a porta
  direto é descartada, e por isso ninguém forja o `X-Forwarded-For`. O UFW
  libera só o SSH, e `TRUSTED_PROXIES` recebe a sub-rede interna do compose
  e o IP do LXC 101.

Public Hostnames no túnel do LXC 101, **nesta ordem** (a regra do `/ws`
antes da raiz). As outras rotas do túnel ficam como estão:

| Subdomain | Domain | Path | Type | URL |
|---|---|---|---|---|
| | projetonova.cloud | `^/ws/?$` | HTTP | `192.168.1.70:8010` |
| | projetonova.cloud | | HTTP | `192.168.1.70:3010` |
| `api` | projetonova.cloud | | HTTP | `192.168.1.70:8010` |
| `arquivos` | projetonova.cloud | | HTTP | `192.168.1.70:9010` (sem "HTTP Host Header": é parte da assinatura) |
| `www` | projetonova.cloud | | HTTP | `192.168.1.70:3010` (só cria o DNS; a Redirect Rule manda para a raiz) |

Um túnel só do Projeto Nova continua possível: um token em
`cloudflare_tunnel_token_producao` (`cloudflare.secrets.yml`), com
`cloudflare_conector_ip` vazio, liga o `cloudflared` do compose. Aí as URLs
viram os nomes dos containers (`frontend:3000`, `backend-api:8010`,
`minio:9000`), e as portas voltam para 127.0.0.1. **Não use o token do LXC
101 na VM.** Seriam dois conectores do mesmo túnel, a Cloudflare reparte as
requisições entre eles, e cada lado recebe as rotas do outro.

```bash
make infra-apply      # 1. cria a VM 240 e gera ansible/inventory.ini
                      # 2. painel: hostnames no túnel do LXC 101, redirect do www
make infra-deploy     # 3. Docker + stack com o domínio (Ansible; no Windows, pelo WSL)
```

Sem Ansible na máquina, `bash scripts/infra-deploy.sh` (ou
`make infra-deploy-docker`) roda o mesmo playbook num container e só precisa
do Docker. Funciona no Git Bash do Windows. A chave das VMs vem de
`NOVA_SSH_KEY` (padrão `~/.ssh/sismarq_deploy`), e os argumentos vão para o
`ansible-playbook`: `bash scripts/infra-deploy.sh --limit producao`.

Na zona, deixe Rocket Loader e Email Address Obfuscation desligados: os dois
injetam scripts que a CSP do frontend bloqueia.

### Versão instalada e rollback

Cada push na `main` com CI verde gera as imagens daquele commit no GHCR
(workflow `Release`, [ADR 063](adr/063-imagens-do-ci-no-registro.md)). Para
instalar, use a tag do commit em `group_vars/producao.yml`:

```yaml
app_image_tag: "sha-4121339"   # Actions > Release mostra a tag de cada commit
```

e rode `make infra-deploy`. O Ansible alinha o checkout da VM ao mesmo
commit (compose e scripts batem com as imagens), baixa as imagens e sobe
sem compilar. **Rollback** é a mesma coisa com a tag anterior. A exceção é
um commit que migrou o banco: as migrations só vão para a frente, então
voltar pede uma migration de correção.

Com `app_image_tag: ""`, a VM compila tudo, como antes. Isso vale para o
primeiro deploy, antes de existir release.

### Segredos do Ansible

`secrets.yml` e `cloudflare.secrets.yml` ficam em texto puro na máquina de
quem opera. Prefira o cofre cifrado, que também fica fora do git (o
repositório é público e um cofre versionado ficaria exposto para sempre a
força bruta):

```bash
mkdir -p ~/.config/projeto-nova && openssl rand -base64 32 > ~/.config/projeto-nova/vault-pass
chmod 600 ~/.config/projeto-nova/vault-pass   # guarde também num cofre de senhas
make infra-segredos-migrar   # junta os dois arquivos no cofre e apaga os abertos
make infra-segredos          # para editar depois
```

### Homologação

A VM 241 (`projeto-nova-homolog-app`, 192.168.1.71, 3 GB) existe no código,
mas está desligada: o pve1 tem 11 GB de RAM, e cada VM de outra aplicação
reserva 4 GB. Para ligar, quando houver memória:

1. `homologacao = { criar = true }` em `terraform.tfvars` e `make infra-apply`.
   O inventário ganha o grupo `homologacao`.
2. No túnel do LXC 101, os mesmos hostnames da tabela acima, com o IP
   192.168.1.71, trocando `projetonova.cloud` por `homolog.projetonova.cloud`
   e `api`/`arquivos` por `api-homolog`/`arquivos-homolog`. Os nomes ficam um
   nível abaixo do domínio porque o certificado universal da Cloudflare
   cobre `*.projetonova.cloud`, não `*.homolog.projetonova.cloud`.
3. `make infra-deploy ANSIBLE_ARGS="--limit homologacao"` (sem ele, os dois).

O workflow `Release` já gera a imagem do frontend de homologação
(`frontend:homologacao-sha-<commit>`). A homologação não manda backup para
fora e não sobe o monitoramento (`group_vars/homologacao.yml`).

### State do Terraform

O state (`deploy/infra/terraform/terraform.tfstate`) fica no notebook de
quem opera, com a senha da VM dentro. Perder esse disco tira do Terraform o
controle das VMs. Para levar o state a um bucket privado (R2 da Cloudflare,
com trava contra dois `apply` ao mesmo tempo):

1. Crie o bucket `projeto-nova-tfstate` e um token R2 só para ele.
2. Copie `backend.tf.example` para `backend.tf` e troque `<conta>`.
3. `AWS_ACCESS_KEY_ID=... AWS_SECRET_ACCESS_KEY=... terraform init -migrate-state`.

Depois disso, apague o `terraform.tfstate` local e os `*.backup`.

O SSO oficial (`sso.rondonopolis.mt.gov.br`) precisa aceitar
`https://projetonova.cloud/*` nas URLs de redirecionamento e de pós-logout
do client `projeto-nova`, e `https://projetonova.cloud` em Web Origins.

Quem entra pelo SSO ganha o perfil pelos grupos dele (`iam_mapeamentos` em
`group_vars/all.yml`, [ADR 074](adr/074-perfis-pelos-grupos-do-sso.md)): os
grupos de administração e TI do AD (`Administrators`, `Domain Admins`,
`Departamento de Tecnologia da Informação`, `Grupo Seciti Tecnologia`) são
administradores do sistema, e todo o resto
recebe o perfil básico `servidor` pelo grupo `*`. O deploy grava esses
mapeamentos, e os demais se cadastram na tela de IAM.

As contas do SSO também são trazidas de uma vez, sem esperar o primeiro login
([ADR 077](adr/077-usuarios-do-sso.md)): botão **Sincronizar com o SSO** em
Configurações > Usuários, e o worker repete todo dia. A sincronização roda em
segundo plano (com 7.700 pessoas, alguns minutos), e a tela mostra o
andamento e o resumo da última. No Keycloak, o client
`projeto-nova` precisa de **Service accounts** habilitado e, na aba *Service
account roles*, a função `view-users` do client `realm-management`. Quem é
desativado no SSO é desativado aqui; quem tem o usuário ou o e-mail de uma
conta local aparece em conflitos, para vincular à mão.

Os avisos por e-mail ([ADR 078](adr/078-central-do-solicitante-e-canais.md)) se configuram em
Configurações > Canais de aviso (permissão `canais:manage`): servidor SMTP,
porta, usuário, senha (fica cifrada) e remetente. Teste o envio antes de ligar.
As variáveis `SMTP_*` valem só enquanto ninguém configurou pela tela. Cada
pessoa deixa o e-mail e o WhatsApp de contato na central (no primeiro acesso)
ou no perfil.

No plano gratuito, a Cloudflare limita o corpo de cada requisição a 100 MB.
Um envio de arquivo maior que isso falha em `arquivos.projetonova.cloud`.

## HTTPS (Caddy com CA interna)

```bash
scripts/enable-https.sh 192.168.1.42
```

Idempotente. Coloca o Caddy (`docker-compose.https.yml`,
`deploy/caddy/Caddyfile`) na frente e troca as URLs do `.env` (cópia em
`.env.bak-*`):

| Serviço | Endereço |
|---|---|
| Projeto-nova-v2 e WebSocket de notificações | `https://<host>` (`wss://<host>/ws`) |
| API direta (health, integrações) | `https://<host>:8443` |
| MinIO (URLs pré-assinadas) | `https://<host>:9443` |
| Keycloak de teste | `https://<host>:8543` |
| Raiz da CA (para instalar) | `http://<host>/nova-ca.crt` |

A porta 80 só serve a CA e redireciona o resto para https.

**TLS 1.2 e 1.3.** O Caddy recusa TLS 1.1 ou anterior e, no 1.2, só oferece cifras modernas. O 1.2 continua aceito porque o Windows 10 (TLS do sistema) não tem 1.3; para exigir só 1.3, troque `protocols tls1.2 tls1.3` por `protocols tls1.3` em `deploy/caddy/Caddyfile` e rode `make deploy`.

**O Caddy é a única entrada.** O script define `HOST_BIND=127.0.0.1`: as
portas diretas (3010/8010/9010/9011/8180) só escutam no próprio servidor —
abertas na rede, deixavam o cliente forjar o `X-Forwarded-For` e escapar dos
limites por IP. O Caddy sobrescreve o cabeçalho com o IP real, o frontend o
repassa à API (`TRUST_PROXY_HEADERS=true`, ligado pelo overlay) e a API
confia nele vindo da rede docker (`TRUSTED_PROXIES`, definido pelo script).

**CA interna.** Sem domínio público, os certificados vêm de uma CA do Caddy
(volume `caddy_data`, incluído no backup). A API e o frontend confiam nela
sozinhos (`secrets/ca/`). **Cada máquina de teste instala a raiz uma vez**:

- Windows: baixe `http://<host>/nova-ca.crt` → duplo clique → Instalar
  certificado → Máquina local → "Autoridades de Certificação Raiz
  Confiáveis" (ou `certutil -addstore -f Root nova-ca.crt` como
  administrador); reabra o navegador.
- Firefox usa repositório próprio: Configurações → Certificados → Importar.
- Linux: copie para `/usr/local/share/ca-certificates/` e rode
  `update-ca-certificates`.

Sem a CA o navegador bloqueia (e "continuar" numa porta não libera as
outras). **Confira o relógio da máquina**: os certificados valem a partir da
emissão e um relógio atrasado os vê como "ainda não válidos".

Com domínio e certificado oficiais, troque `tls internal` no Caddyfile.

## Backup e restauração

```bash
scripts/backup.sh                    # agora, em /root/backups/nova/<data-hora>
scripts/backup.sh --install-cron     # todo dia às 02:30 (log: /var/log/nova-backup.log)
scripts/restore.sh --verify <dir>    # restaura num banco temporário e compara — não altera nada
scripts/restore.sh --yes <dir> --minio   # SUBSTITUI banco (e objetos) pelo backup
```

Cada backup tem o dump do Postgres, o espelho dos buckets do MinIO, os
volumes do Keycloak de teste e da CA do Caddy (se existirem), `.env` +
`secrets/` e `SHA256SUMS`. Retenção de `BACKUP_KEEP_DAYS` (7) dias.
O volume do Typesense (`typesense_data`) não entra no backup: o índice é
derivado do banco, e o worker o reconstrói sozinho depois da restauração.
**O `.env` do backup é indispensável** (senha do banco e
`CONFIG_ENCRYPTION_KEY`): numa máquina nova, copie `<dir>/config/.env` e
`<dir>/config/secrets` antes de restaurar.

### Cópia fora da VM e verificação

A cópia local protege contra erro humano, não contra perder a máquina. Com
`BACKUP_REMOTE`, cada backup também sai, **cifrado**, para um destino do
`rclone`: SFTP num storage do Proxmox ou num NAS, ou S3. O backup leva o
`.env` com todos os segredos, então sem `BACKUP_AGE_RECIPIENT` nada sai.

1. Na máquina de quem opera (não na VM): `age-keygen -o projeto-nova-backup.key`.
   Guarde o arquivo num cofre de senhas: sem ele, a cópia de fora não abre.
2. Em `group_vars/all.yml`: `backup_remote` (ex.: `fora:/backups/projeto-nova`)
   e `backup_age_recipient` (a linha `age1...` da chave pública).
3. Em `secrets.yml`: `backup_rclone_conf`, o destino com credencial (modelo
   em `secrets.yml.example`).
4. `make infra-deploy`. A role `nova_backup` instala `age` e `rclone` e agenda
   o backup (02:30, retenção de 30 dias lá fora) e a verificação (domingo,
   04:30).

A verificação restaura o último backup num banco temporário e compara as
contagens (`restore.sh --verify`). Um backup que não restaura não é backup.
Cada sucesso grava a hora em `.ultimo-ok` e `.ultima-verificacao-ok`, dentro
de `BACKUP_DIR`.

Restaurar da cópia de fora, numa máquina nova:

```bash
rclone copy fora:/backups/projeto-nova/projeto-nova-<data-hora>.tar.age .
age -d -i projeto-nova-backup.key projeto-nova-<data-hora>.tar.age | tar -xf -
cp <data-hora>/config/.env .env && cp -r <data-hora>/config/secrets .
scripts/restore.sh --yes <data-hora> --minio
```

## Monitoramento

A role `nova_monitoring` (`monitoring_enabled: true`) sobe, na mesma VM,
Prometheus, Alertmanager, Grafana e node-exporter (`deploy/observability`).
Nada disso passa pelo túnel: as portas escutam só em 127.0.0.1.

```bash
ssh -L 3005:127.0.0.1:3005 -L 9090:127.0.0.1:9090 -L 9093:127.0.0.1:9093 root@192.168.1.70
# Grafana http://localhost:3005 (admin; senha em deploy/observability/.env na VM)
```

O token de `/metrics`, o token dos alertas e a senha do Grafana são gerados
na VM e não entram no repositório.

**Os alertas viram chamado** ([ADR 067](adr/067-eventos-do-monitoramento.md)).
O Alertmanager manda cada alerta para a entrada de eventos do atendimento,
autenticado pelo `ATENDIMENTO_EVENTOS_TOKEN` que o `deploy.sh` gera. Em
Atendimento > Cadastro, quem gere o atendimento da instituição escolhe a
oferta de incidente em que eles abrem. Sem oferta escolhida, a entrada
responde 409 e o Alertmanager tenta de novo; nada se perde em silêncio. O
mesmo alerta de novo vira nota no mesmo chamado, e a volta ao normal também,
sem concluir o chamado.

| Alerta | Quando |
|---|---|
| `NovaServiceDown` | API ou worker sem responder por 1 min |
| `NovaHttp5xxErrorRateHigh` | mais de 1% de respostas 5xx em 5 min |
| `NovaLatencyP95High` | p95 acima de 300 ms por 3 min |
| `NovaOutboxLagHigh` | mais de 50 eventos parados no outbox |
| `NovaDiscoQuaseCheio` / `NovaDiscoCritico` | disco raiz acima de 85% / 95% |
| `NovaBackupAtrasado` | nenhum backup bem-sucedido em 26 h |
| `NovaBackupSemVerificacao` | nenhuma restauração de teste em 8 dias |

**De fora**, o workflow `Disponibilidade` do GitHub confere o site e a API
pelo domínio a cada 5 minutos e abre a issue "Fora do ar" quando falham. Ele
pega o que o Prometheus não vê: a VM inteira fora, o túnel ou o DNS. Ligue
com a variável de repositório `MONITORAR_DISPONIBILIDADE=true` depois do
primeiro deploy.

### Fora do ar

1. **Só o site ou também a API?** A issue diz qual endereço falhou.
2. **O túnel está de pé?** Painel da Cloudflare > Zero Trust > Networks >
   Tunnels: `projeto-nova-prod` precisa estar *Healthy*. Na VM:
   `docker compose logs --tail 50 cloudflared`.
3. **A stack está saudável?** Na VM: `make prod-ps`; um serviço
   `unhealthy` aponta o culpado (`docker compose logs <serviço>`).
4. **A VM responde?** Sem SSH em 192.168.1.70, abra o console da VM 240 no
   Proxmox. Disco cheio é a causa mais comum: `df -h /` e `make prod-limpeza`.
5. **Voltou?** A issue fecha sozinha na próxima conferência. Registre a
   causa nela antes, para a revisão do incidente.

## Inteligência artificial

A plataforma guarda as conexões de IA e a escolha de qual usar, mas
**nenhum módulo atual consulta a IA**. A configuração fica pronta para o
primeiro módulo que precisar. É
feita **pela tela**, em **Configurações → Inteligência artificial**
(permissão `ia:manage`, ADR 020):

- **Conexões:** IA local (Ollama), OpenAI, Azure OpenAI, Google Gemini,
  Anthropic (Claude), Groq, OpenRouter, Mistral ou qualquer serviço
  compatível com a API de chat da OpenAI. Endereço, modelo, tempo limite e
  **chave de API** — cifrada no banco (`CONFIG_ENCRYPTION_KEY`), nunca
  devolvida pela API nem gravada na auditoria. Ao salvar, a conexão é
  **testada** com uma conversa real; se falhar, nada é gravado. Trocar o
  endereço exige informar a chave de novo.
- **Uso:** conexão **principal** e **reserva** da plataforma. Vale na
  próxima chamada, sem reiniciar.
- **Fornecedor externo:** exige autorização explícita (registrada com quem
  e quando — LGPD art. 33) e, por padrão, CPF, CNPJ, e-mail e telefone são
  mascarados na pergunta antes de sair da rede.

Enquanto nada for salvo na tela, valem `IA_ENDPOINT`,
`IA_API_KEY`, `IA_MODELO` e `IA_TIMEOUT` do `.env`.

**IA local (serviço `ia-local`, Ollama).** Overlay opcional:

```bash
# .env
COMPOSE_FILE=docker-compose.yml:...:docker-compose.ia.yml   # (+ os overlays que já usa)
IA_LOCAL_MODELO=qwen2.5:1.5b    # opcional; baixado na 1ª subida (serviço ia-local-modelo)
```
```bash
make deploy          # sobe ia-local e baixa o modelo; make ia-modelo baixa de novo
```

Sem porta publicada (só a API o alcança), com limite de memória e CPU
(`IA_LOCAL_MEMORIA`, padrão 2560m; `IA_LOCAL_CPUS`, padrão 3). **Precisa de
RAM sobrando:** sem GPU, um modelo de 1,5B usa ~1,2 GB e um de 3B ~2,5 GB,
além da stack (~2 GB). No servidor de teste (4 GB) o modelo esgotou a
memória e a geração caiu a 0,3 palavra/s — use ≥ 8 GB, uma máquina
dedicada (a conexão "IA local" aceita qualquer endereço da rede, ex.
`http://<servidor-ia>:11434`) ou um fornecedor externo.

## Modelar a organização

**Estrutura real a partir do AD.** `deploy/estrutura/` traz a estrutura da
prefeitura extraída do export do Active Directory (25 entidades, 255
unidades, 55 departamentos — sem dados pessoais):

```bash
make estrutura-aplicar                 # idempotente: pode rodar de novo
make estrutura-gerar AD=<pasta-export> # regera a partir de um export novo
```

O export bruto (`docs/AD/`, ignorado pelo git) tem nomes de servidores e não
deve ser versionado. O gerador (`scripts/estrutura/ad-para-estrutura.mjs`)
ignora contas de usuário, só lista os grupos de segurança (viram
mapeamentos na tela de IAM) e documenta no cabeçalho as decisões de
modelagem. **Não misture com os dados fictícios** (`make demo-seed`,
`demo-popular`, `demo-test`): os cenários de teste dependem da prefeitura
fictícia.

Como desenhar entidades, unidades, departamentos, grupos e mapeamentos (e
o que o escopo restringe ou não): [REGRAS_DE_NEGOCIO.md](REGRAS_DE_NEGOCIO.md#6-como-modelar-uma-implantação).

## Migrar do GLPI

O caminho recomendado é a tela **Integrações** ([ADR 076](adr/076-modulo-de-integracoes.md)), com o módulo ligado e a permissão `integracoes:manage` global:

1. Crie a conexão. Se o servidor não alcança o GLPI (o caso comum), escolha **conector** e emita o token.
2. Numa máquina da rede do GLPI, rode o conector com o token e as credenciais do GLPI. Elas ficam nessa máquina:

   ```bash
   NOVA_API=https://api.projetonova.cloud NOVA_TOKEN=nova_con_... \
   GLPI_URL=http://helpdesk/apirest.php GLPI_APP_TOKEN=... GLPI_LOGIN=... GLPI_PASSWORD=... \
     go run ./cmd/conector --vigiar
   ```

   Com `--vigiar`, ele atende cada extração pedida na tela. Sem a opção, atende a extração pendente e sai.
3. Na tela: **Extrair**, conferir o **inventário**, **sugerir a tabela de-para** com a unidade de cada secretaria, revisar, **simular** e **carregar**. Repita a extração e a carga até a virada: só entra o que mudou.

O comando `importarglpi`, abaixo, continua para quem implanta sem a tela.

O comando `importarglpi` traz o histórico do GLPI para o atendimento ([ADR 072](adr/072-importador-do-glpi.md)): categorias viram ofertas **inativas** (revise o catálogo e ative as que valem), chamados viram solicitações com a conversa e a avaliação, e as pessoas são casadas pelo e-mail e depois pelo login com quem já existe aqui. Quem não casa fica na conta de sistema "Importação", e o nome original vai no texto.

1. No GLPI, ative a API REST (Configurar > Geral > API), crie um cliente de API (App-Token) e gere o token de API de um usuário que veja todos os chamados.
2. Rode a **simulação** (não grava nada) e leia o relatório: quantos chamados, quantas ofertas novas e a lista `sem_par`, das pessoas sem par.

   ```bash
   GLPI_URL=https://glpi.prefeitura/apirest.php GLPI_APP_TOKEN=... GLPI_USER_TOKEN=...    DB_HOST=... DB_PORT=5432 DB_NAME=nova DB_USER=nova DB_PASSWORD=...      go run ./cmd/importarglpi --unidade <uuid da unidade dona>
   ```

   Se a VM não enxerga o GLPI (na produção, o GLPI só existe no DNS interno
   da prefeitura e a VM fica na rede do Proxmox), rode de uma máquina da
   rede da prefeitura: `bash scripts/importar-glpi.sh [--unidade <uuid>] [--gravar]`.
   Ele lê os tokens de `~/.config/projeto-nova/glpi.env`, a senha do banco
   no `.env` da VM, e grava por um túnel SSH que só vive durante o comando.

   Quando as categorias do GLPI definem o SLA ou se repetem por entidade,
   use a **tabela de-para** ([ADR 075](adr/075-de-para-do-glpi.md)):
   `--mapa tabela.csv` (no script, `GLPI_MAPA=tabela.csv`). Cada linha leva
   uma secretaria e uma categoria do GLPI (ou `*`) a uma unidade, oferta,
   tipo e prioridade daqui. A simulação mostra as unidades que faltam criar
   e o que a tabela não cobre. A tabela tem dados da organização: guarde-a
   fora do repositório.

3. Corrija no AD/SSO o e-mail de quem ficou sem par, se for o caso, e rode de novo com `--gravar`.
4. Rodar de novo só traz o que é novo. Faça uma carga grande antes da virada e uma final no dia em que o GLPI deixar de receber chamados.

Os tokens ficam no ambiente, não na linha de comando. As datas do GLPI vêm sem fuso; o padrão é `--fuso America/Sao_Paulo`. A importação não avisa ninguém nem publica eventos, e registra uma linha na auditoria com os totais. Anexos e ativos do GLPI não entram.

## Ambiente de teste: Keycloak + dados fictícios

Enquanto não há um Keycloak oficial, `make demo-keycloak` sobe um
**Keycloak de teste** (`docker-compose.keycloak.yml`, Keycloak 26 em
`start-dev`) com o realm `nova` e carrega no Postgres uma estrutura
organizacional fictícia:

| Entidade | Unidades | Departamentos |
|---|---|---|
| Prefeitura Municipal (PREF) | Sede Administrativa | RH, TI, Contabilidade, Compras, Administração, Governo |
| Secretaria de Saúde (SEMSA) | PSF Sagrada Família, PSF Conjunto, PSF Marechal Rondon, UPA 24h | padrão* |
| Secretaria de Educação (SEMED) | Escolas Maria Elza, Marechal Dutra, Silvestre, Elizabete | padrão* |
| SEMPRAS | Sede, CRAS Conjunto, CRAS Ana Carla, CRAS Alfredo, CREAS, Centro POP | padrão* |

\* Direção, Administração, Atendimento, Recursos Humanos, Almoxarifado.

- **50 usuários por unidade**, divididos igualmente entre os departamentos
  (754 no total, com as contas `teste.admin`, `teste.auditor`, `teste.iam`
  e `teste.conteudo`). Lista em `deploy/demo/usuarios.csv`; senha comum em
  `DEMO_USER_PASSWORD` no `.env` do servidor.
- Cada departamento é um grupo no Keycloak (ex.: `SEMSA-UPA-ADM`) mapeado
  para o perfil **Servidor** com lotação no departamento; a Administração
  de cada unidade também recebe **Protocolo** na unidade.
- Os usuários já existem no Projeto-nova-v2 antes do 1º login (Diretório com cargo e
  ramal); o login pelo Keycloak só os atualiza.
- Console: `https://<host>:8543/admin` (sem HTTPS: `http://<host>:8180/admin`),
  usuário `admin` e `KEYCLOAK_ADMIN_PASSWORD` do `.env`. Os usuários do
  Projeto-nova-v2 ficam no realm **nova**.
- `make demo-popular` (uma vez; marca em `.demo-popular`) povoa todos os
  apps com dados criados pelos próprios usuários fictícios: serviços do
  catálogo, notícias, manuais na Wiki, salas e eventos, salas de equipe e
  conversas, processos entre unidades em vários estados, documentos
  assinados, pasta por unidade e mensagens de cidadãos.

Mudar a estrutura: edite `scripts/demo-data/generate.mjs`, rode
`make demo-generate` e faça commit dos arquivos gerados. O SQL é reaplicável
(`make demo-seed`), mas o realm só é importado quando ainda não existe —
para reimportar, `docker compose rm -sf keycloak && docker volume rm
projeto-nova_keycloak_data` e `make deploy` (os usuários voltam com os
mesmos ids; quem estava logado entra de novo).

### Cenários ponta a ponta (`make demo-test`)

Com o Keycloak de teste no ar, as suítes de `scripts/demo-data/cenarios/`
fazem login **de verdade** pelo Keycloak (mesmo fluxo do navegador) com
usuários de lotações diferentes e exercitam casos permitidos e negados:

| Suíte | Cobre |
|---|---|
| `tramite.py` | abertura por perfil/unidade, sigilo, tramitação de ida e volta, conclusão |
| `tramite_documentos.py` | anexo no MinIO, edição, assinatura via Signum, acesso a restrito, caixa da unidade, concluir → arquivar → reabrir |
| `modulos.py` | Signum, Mercúrio, Agenda, Arquivos, Diretório, IAM, Busca, Auditoria, LGPD |
| `conteudo.py` | Blog, Wiki (revisões e conflito de edição), Catálogo (Lei 13.460), Contato anônimo, Egress (anti-SSRF) |
| `colaboracao.py` | tempo real por WebSocket (mensagem, edição, exclusão, difusão via outbox → RabbitMQ), salas de departamento, reserva de sala com conflito, ACL de pastas, Signum sequencial/recusa/cancelamento |
| `integracoes.py` | webhooks entregues e com retentativa, busca global (inclusive por prefixo), auditoria por tipo de ação |
| `plataforma.py` | CRUD do IAM, anti-escalada de privilégio, lotação com efeito imediato, usuários locais e bloqueio, módulos e dependências, branding, transparência, monitoramento, LGPD |

Chamadas avulsas como qualquer usuário:
`scripts/demo-data/cenarios/nx <usuario> GET me`.

### Trocar pelo Keycloak oficial

No realm oficial, dois clients (o realm de teste em
`deploy/keycloak/realm-nova.json` serve de modelo):

- **`nova-frontend`** — confidencial, Standard Flow com PKCE S256,
  redirect `https://<host>/*`, *audience mapper* incluindo `nova-backend`
  no access token e *group membership mapper* `groups` com
  `full.path=false` (é o nome do grupo que o IAM cruza com os mapeamentos).
- **`nova-backend`** — confidencial, **só Direct Access Grants**: é a
  audience dos tokens e o client da reautenticação do Signum (senha na hora
  de assinar). Sem ele, ninguém assina com conta do Keycloak.

Depois, no `.env`: `KEYCLOAK_ISSUER_URL`, `KEYCLOAK_CLIENT_SECRET`,
`KEYCLOAK_FRONTEND_CLIENT_SECRET`; tire `docker-compose.keycloak.yml` do
`COMPOSE_FILE` e rode `make deploy`. Se o Keycloak usar certificado da CA
interna, ele já é confiável; se usar outra CA privada, coloque a raiz em
`secrets/ca/`.

## Segurança do login local

O login local (contingência) tem dois bloqueios progressivos (Redis):

| Sujeito | Libera | Depois | Contador |
|---|---|---|---|
| Conta | 5 falhas | 1 min, dobrando até 24 h | 24 h (zera no login certo ou no desbloqueio pelo gestor) |
| IP | `LOGIN_LOCKOUT_IP_THRESHOLD` (30) falhas | 1 min, dobrando até 1 h | 1 h |

O limite do IP é alto de propósito: atrás de NAT, um IP é um prédio
inteiro. O contador do IP não zera com login bem-sucedido (senão uma conta
válida "limparia" o IP durante password spraying). Há ainda o limitador de
login por IP (10/min), que cai à metade a cada 3 falhas e volta ao normal no
primeiro login certo.

## Comandos úteis

| Comando | O que faz |
|---|---|
| `make deploy` | git pull + rebuild + restart, esperando `healthy` |
| `make prod-ps` / `make prod-logs` | estado e logs dos serviços |
| `make prod-seed-admin` | cria/reseta o admin local |
| `make prod-down` | para a stack (mantém os volumes) |
| `make demo-keycloak` | liga o Keycloak de teste e carrega a estrutura fictícia |
| `make demo-generate` / `make demo-seed` | regera / reaplica os dados fictícios |
| `make demo-popular` | povoa todos os apps (uma vez) |
| `make demo-test` | roda os cenários ponta a ponta |
| `make iam-scope-report` | impacto da permissão com escopo nas lotações atuais (ADR 013) |
| `scripts/backup.sh` / `scripts/restore.sh` | backup e restauração |
