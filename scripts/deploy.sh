#!/usr/bin/env bash
# Deploy do Projeto-nova-v2 num servidor com Docker (modo produção, só o
# docker-compose.yml). Idempotente: pode rodar a cada atualização.
#
#   scripts/deploy.sh                 # usa o 1º IP da máquina como host público
#   scripts/deploy.sh 192.168.1.42    # host/IP/DNS que o navegador vai usar
#   scripts/deploy.sh --no-pull ...   # não faz git pull antes do build
#   scripts/deploy.sh --env-only ...  # só gera .env e chaves, sem subir nada
#                                     # (o Ansible grava o domínio antes do build)
#
# Com NOVA_REGISTRY e NOVA_IMAGE_TAG (e NOVA_FRONTEND_TAG) no .env, baixa as
# imagens do CI em vez de compilar. Rollback: voltar a tag e rodar de novo.
#
# Na primeira execução gera .env (segredos aleatórios fortes) e a chave RSA
# do login local em secrets/. Execuções seguintes reaproveitam os dois —
# NUNCA regenere o .env com dados no banco: DB_PASSWORD e
# CONFIG_ENCRYPTION_KEY já estão gravados nos volumes.
#
# Portas (sobrescreva por variável de ambiente na 1ª execução):
#   FRONTEND_PORT=3010 HTTP_PORT=8010 HOST_MINIO_PORT=9010 HOST_MINIO_CONSOLE_PORT=9011
set -euo pipefail

cd "$(dirname "$0")/.."

PULL=1
ENV_ONLY=0
while [ $# -gt 0 ]; do
  case "$1" in
    --no-pull) PULL=0 ;;
    --env-only) ENV_ONLY=1; PULL=0 ;;
    *) break ;;
  esac
  shift
done

PUBLIC_HOST="${1:-${PUBLIC_HOST:-$(hostname -I | awk '{print $1}')}}"
# Sem -f: o docker compose lê COMPOSE_FILE do .env (ex.: Keycloak de
# teste, scripts/demo-keycloak.sh); sem ele, usa só o docker-compose.yml.
COMPOSE=(docker compose)

log() { printf '\n==> %s\n' "$*"; }

# set_kv CHAVE VALOR: troca a linha CHAVE=... do .env ou acrescenta no fim.
set_kv() {
  if grep -q "^$1=" .env; then
    sed -i "s|^$1=.*|$1=$2|" .env
  else
    printf '%s=%s\n' "$1" "$2" >>.env
  fi
}

if [ "$PULL" = 1 ] && [ -d .git ]; then
  log "Atualizando o código (git pull --ff-only)"
  git pull --ff-only
fi

if [ ! -f .env ]; then
  log "Gerando .env para http://$PUBLIC_HOST (segredos aleatórios)"
  cp .env.example .env
  fp="${FRONTEND_PORT:-3010}"
  ap="${HTTP_PORT:-8010}"
  mp="${HOST_MINIO_PORT:-9010}"
  mcp="${HOST_MINIO_CONSOLE_PORT:-9011}"
  set_kv APP_ENV production
  set_kv FRONTEND_PORT "$fp"
  set_kv HTTP_PORT "$ap"
  set_kv HOST_MINIO_PORT "$mp"
  set_kv HOST_MINIO_CONSOLE_PORT "$mcp"
  set_kv FRONTEND_URL "http://$PUBLIC_HOST:$fp"
  set_kv API_PUBLIC_URL "http://$PUBLIC_HOST:$ap"
  set_kv WEBSOCKET_PUBLIC_URL "ws://$PUBLIC_HOST:$ap/ws"
  set_kv MINIO_PUBLIC_URL "http://$PUBLIC_HOST:$mp"
  # hex: seguro dentro das URLs amqp:// e redis:// montadas no compose.
  set_kv DB_PASSWORD "$(openssl rand -hex 32)"
  set_kv RABBITMQ_DEFAULT_PASS "$(openssl rand -hex 32)"
  set_kv REDIS_PASSWORD "$(openssl rand -hex 32)"
  set_kv MINIO_ROOT_PASSWORD "$(openssl rand -hex 32)"
  set_kv KEYCLOAK_FRONTEND_CLIENT_SECRET "$(openssl rand -hex 32)"
  set_kv NEXTAUTH_SECRET "$(openssl rand -hex 32)"
  set_kv METRICS_SCRAPE_TOKEN "$(openssl rand -hex 32)"
  set_kv CONFIG_ENCRYPTION_KEY "$(openssl rand -base64 32)"
  chmod 600 .env
else
  log "Reaproveitando o .env existente"
fi

# Segredos que entraram depois do .env já existir: gera só o que falta (os
# demais nunca são regenerados — ver o aviso no topo).
if ! grep -q '^TYPESENSE_API_KEY=.' .env || grep -q '^TYPESENSE_API_KEY=dev-change-this' .env; then
  log "Gerando TYPESENSE_API_KEY (índice da Busca Global, ADR 031)"
  set_kv TYPESENSE_API_KEY "$(openssl rand -hex 32)"
fi
# O valor do .env.example é público e a API recusa subir com ele. Só troca
# o que nunca serviu em produção: uma chave real já embrulha as senhas do
# cofre e não pode mudar.
if ! grep -q '^COFRE_MASTER_KEY=.' .env || grep -q '^COFRE_MASTER_KEY=MTExMTExMTExMTExMTExMTExMTExMTExMTExMTExMTE=$' .env; then
  log "Gerando COFRE_MASTER_KEY (cofre de senhas, ADR 053)"
  set_kv COFRE_MASTER_KEY "$(openssl rand -base64 32)"
fi

if [ ! -f secrets/local_auth_private_key.pem ]; then
  log "Gerando a chave RSA do login local (secrets/local_auth_private_key.pem)"
  mkdir -p secrets
  openssl genrsa -out secrets/local_auth_private_key.pem 2048
  # O container roda como usuário não-root (nix) e lê a chave pelo bind
  # mount /run/secrets — precisa ser legível por ele.
  chmod 644 secrets/local_auth_private_key.pem
fi

if [ "$ENV_ONLY" = 1 ]; then
  log "Pronto (--env-only): .env e chaves gerados, nada subiu"
  exit 0
fi

# Borda HTTPS ativa? O Caddy sobe primeiro: a CA interna dele é recriada
# quando o volume caddy_data se perde (ex.: `down -v`), e API, worker e
# frontend confiam em secrets/ca — com a cópia desatualizada a API não
# valida o Keycloak e não sobe. Sincroniza antes de subir o resto.
ca_changed=0
if "${COMPOSE[@]}" config --services 2>/dev/null | grep -qx caddy; then
  log "Subindo o Caddy e conferindo a CA interna"
  "${COMPOSE[@]}" up -d caddy
  tmp=$(mktemp)
  for _ in $(seq 1 30); do
    "${COMPOSE[@]}" cp caddy:/data/caddy/pki/authorities/local/root.crt "$tmp" >/dev/null 2>&1 && [ -s "$tmp" ] && break
    sleep 2
  done
  if [ -s "$tmp" ] && ! cmp -s "$tmp" secrets/ca/nova-ca.crt; then
    mkdir -p secrets/ca
    cp "$tmp" secrets/ca/nova-ca.crt
    chmod 644 secrets/ca/nova-ca.crt
    ca_changed=1
    echo "  CA nova em secrets/ca — reinstale nas máquinas de teste (http://<host>/nova-ca.crt)"
  fi
  rm -f "$tmp"
fi

# Com NOVA_IMAGE_TAG no .env, as imagens vêm prontas do registro (as que o
# CI testou, ADR 063): nada compila aqui. Sem ela, compila no servidor.
if grep -q '^NOVA_IMAGE_TAG=.' .env; then
  log "Baixando as imagens $(grep '^NOVA_IMAGE_TAG=' .env | cut -d= -f2-) do registro"
  "${COMPOSE[@]}" pull --quiet migrate backend-api backend-worker frontend
  log "Subida dos serviços"
  "${COMPOSE[@]}" up -d --no-build --remove-orphans
else
  log "Build e subida dos serviços"
  "${COMPOSE[@]}" up -d --build --remove-orphans
fi

if [ "$ca_changed" = 1 ]; then
  log "Reiniciando API, worker e frontend com a CA nova"
  "${COMPOSE[@]}" restart backend-api backend-worker frontend
fi

# O Caddy relê o Caddyfile sem derrubar conexões: mudanças de TLS/rotas
# valem sem recriar o container.
if "${COMPOSE[@]}" ps --services 2>/dev/null | grep -qx caddy; then
  log "Recarregando a configuração do Caddy"
  "${COMPOSE[@]}" exec -T caddy caddy reload --config /etc/caddy/Caddyfile --adapter caddyfile
fi

log "Aguardando os serviços ficarem saudáveis"
for _ in $(seq 1 60); do
  unhealthy=$("${COMPOSE[@]}" ps --format '{{.Service}} {{.Health}}' | awk '$2 != "healthy" && $2 != "" {print $1}')
  [ -z "$unhealthy" ] && break
  sleep 5
done
"${COMPOSE[@]}" ps

if [ -n "${unhealthy:-}" ]; then
  echo "Serviços ainda não saudáveis: $unhealthy — veja: docker compose logs <serviço>" >&2
  exit 1
fi

env_val() { grep "^$1=" .env | cut -d= -f2- || true; }
log "Pronto"
echo "  Frontend: $(env_val FRONTEND_URL)"
echo "  API:      $(env_val API_PUBLIC_URL)/health"
echo "  Admin local (1ª vez, ou para resetar a senha): make prod-seed-admin"
