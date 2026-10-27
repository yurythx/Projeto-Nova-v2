#!/usr/bin/env bash
# Backup do Projeto-nova-v2 (stack docker compose deste diretório):
#   db.dump        pg_dump -Fc do banco do Projeto-nova-v2 (restaurável com restore.sh)
#   minio/         espelho de todos os buckets (arquivos, anexos, WORM)
#   caddy.tgz      CA interna do HTTPS, se existir (chave privada inclusive)
#   keycloak.tgz   volume do Keycloak de teste, se existir (melhor esforço:
#                  o realm também é reproduzível por deploy/keycloak)
#   config/        .env e secrets/ — SEM eles o backup não restaura: a senha
#                  do banco e a CONFIG_ENCRYPTION_KEY estão aqui
#   SHA256SUMS     hashes de tudo acima
#
#   scripts/backup.sh                 # grava em $BACKUP_DIR/<data-hora>
#   scripts/backup.sh --install-cron  # agenda diariamente às 02:30
#
# BACKUP_DIR (padrão /root/backups/nova) e BACKUP_KEEP_DAYS (padrão 7).
# A cópia local fica no MESMO disco: protege contra erro humano e dado
# corrompido, não contra perda da máquina. Para isso, a cópia de fora:
#
#   BACKUP_REMOTE         destino rclone (ex.: pve:/backups/projeto-nova)
#   BACKUP_AGE_RECIPIENT  chave pública age; sem ela nada sai daqui, porque
#                         o backup leva o .env com todos os segredos
#   BACKUP_REMOTE_KEEP_DAYS  retenção lá fora (padrão 30)
#
# A chave privada age fica FORA desta máquina: quem invadir o servidor não
# lê os backups antigos. Restaurar de lá: docs/DEPLOY.md, "Backup".
#
# Cada backup que termina grava a hora em $BACKUP_DIR/.ultimo-ok, e cada
# verificação semanal em $BACKUP_DIR/.ultima-verificacao-ok (o
# monitoramento alerta quando ficam velhos).
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT="$(pwd)"
BACKUP_DIR="${BACKUP_DIR:-/root/backups/nova}"
KEEP_DAYS="${BACKUP_KEEP_DAYS:-7}"
REMOTE="${BACKUP_REMOTE:-}"
RECIPIENT="${BACKUP_AGE_RECIPIENT:-}"
REMOTE_KEEP_DAYS="${BACKUP_REMOTE_KEEP_DAYS:-30}"

# Métrica para o node-exporter (alertas NovaBackup*). Escreve num
# temporário e renomeia: o coletor nunca lê um arquivo pela metade.
TEXTFILE_DIR="${BACKUP_TEXTFILE_DIR:-/var/lib/node_exporter/textfile}"
publicar_metricas() {
  [ -d "$TEXTFILE_DIR" ] || return 0
  local tmp="$TEXTFILE_DIR/nova_backup.prom.$$"
  {
    echo "# HELP nova_backup_ultimo_ok_timestamp_seconds Fim do último backup bem-sucedido."
    echo "# TYPE nova_backup_ultimo_ok_timestamp_seconds gauge"
    [ -f "$BACKUP_DIR/.ultimo-ok" ] && echo "nova_backup_ultimo_ok_timestamp_seconds $(cat "$BACKUP_DIR/.ultimo-ok")"
    echo "# HELP nova_backup_verificacao_ok_timestamp_seconds Fim da última restauração de teste bem-sucedida."
    echo "# TYPE nova_backup_verificacao_ok_timestamp_seconds gauge"
    [ -f "$BACKUP_DIR/.ultima-verificacao-ok" ] && echo "nova_backup_verificacao_ok_timestamp_seconds $(cat "$BACKUP_DIR/.ultima-verificacao-ok")"
  } >"$tmp"
  chmod 644 "$tmp" && mv "$tmp" "$TEXTFILE_DIR/nova_backup.prom"
}

if [ "${1:-}" = "--install-cron" ]; then
  vars="BACKUP_DIR=$BACKUP_DIR BACKUP_KEEP_DAYS=$KEEP_DAYS"
  [ -n "$REMOTE" ] && vars="$vars BACKUP_REMOTE='$REMOTE' BACKUP_AGE_RECIPIENT='$RECIPIENT' BACKUP_REMOTE_KEEP_DAYS=$REMOTE_KEEP_DAYS"
  cat >/etc/cron.d/nova-backup <<EOF
# Backup diário e verificação semanal do Projeto-nova-v2
# (scripts/backup.sh --install-cron)
SHELL=/bin/bash
30 2 * * * root cd $ROOT && $vars ./scripts/backup.sh >>/var/log/nova-backup.log 2>&1
30 4 * * 0 root cd $ROOT && BACKUP_DIR=$BACKUP_DIR ./scripts/backup.sh --verificar >>/var/log/nova-backup.log 2>&1
EOF
  chmod 644 /etc/cron.d/nova-backup
  echo "Agendado: /etc/cron.d/nova-backup (backup 02:30, verificação domingo 04:30, log em /var/log/nova-backup.log)"
  exit 0
fi

# Restaura o backup mais recente num banco temporário e compara as contagens
# (scripts/restore.sh --verify): um backup que não restaura não é backup.
if [ "${1:-}" = "--verificar" ]; then
  ultimo="$(find "$BACKUP_DIR" -mindepth 1 -maxdepth 1 -type d -name '2*' | sort | tail -1)"
  [ -n "$ultimo" ] || { echo "$(date '+%F %T') verificação: nenhum backup em $BACKUP_DIR" >&2; exit 1; }
  echo "$(date '+%F %T') verificação de $ultimo"
  ./scripts/restore.sh --verify "$ultimo"
  date +%s >"$BACKUP_DIR/.ultima-verificacao-ok"
  publicar_metricas
  exit 0
fi

if [ -n "$REMOTE" ] && [ -z "$RECIPIENT" ]; then
  echo "BACKUP_REMOTE sem BACKUP_AGE_RECIPIENT: o backup leva os segredos do .env e não sai daqui sem cifra." >&2
  exit 1
fi

env_val() { grep "^$1=" .env | tail -1 | cut -d= -f2- || true; }
DB_USER="$(env_val DB_USER)"; DB_USER="${DB_USER:-nova}"
DB_NAME="$(env_val DB_NAME)"; DB_NAME="${DB_NAME:-nova}"
PROJECT="$(docker compose config 2>/dev/null | sed -n 's/^name: //p' | head -1)"

dest="$BACKUP_DIR/$(date +%Y%m%d-%H%M%S)"
umask 077
mkdir -p "$dest/config" "$dest/minio"
log() { printf '%s %s\n' "$(date '+%F %T')" "$*"; }
log "backup -> $dest"

log "postgres: pg_dump $DB_NAME"
docker compose exec -T postgres pg_dump -U "$DB_USER" -d "$DB_NAME" -Fc --no-owner >"$dest/db.dump"

log "minio: mc mirror de todos os buckets"
docker compose run --rm --no-deps -T -v "$dest/minio:/out" --entrypoint sh minio -c \
  'export MC_HOST_l="http://$MINIO_ROOT_USER:$MINIO_ROOT_PASSWORD@minio:9000"; mc mirror --quiet --preserve l /out' >/dev/null

# Volumes opcionais: Keycloak de teste e a CA interna do Caddy (HTTPS) —
# perder a CA obriga toda máquina de teste a instalar uma raiz nova.
for vol in keycloak caddy; do
  if docker volume inspect "${PROJECT}_${vol}_data" >/dev/null 2>&1; then
    log "$vol: volume ${PROJECT}_${vol}_data"
    docker run --rm -v "${PROJECT}_${vol}_data:/data:ro" -v "$dest:/out" postgres:16-bookworm \
      tar czf "/out/$vol.tgz" -C /data .
    chmod 600 "$dest/$vol.tgz"
  fi
done

log "config: .env e secrets/"
cp .env "$dest/config/.env"
[ -d secrets ] && cp -r secrets "$dest/config/"

(cd "$dest" && find . -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 sha256sum >SHA256SUMS)
log "ok: $(du -sh "$dest" | cut -f1), $(wc -l <"$dest/SHA256SUMS") arquivos"

if [ -n "$REMOTE" ]; then
  nome="projeto-nova-$(basename "$dest").tar.age"
  log "fora: cifrando (age) e enviando para $REMOTE/$nome"
  # Cifra no fluxo: o arquivo aberto nunca passa por outro disco.
  tar -C "$BACKUP_DIR" -cf - "$(basename "$dest")" | age -r "$RECIPIENT" | rclone rcat "$REMOTE/$nome"
  rclone delete --min-age "${REMOTE_KEEP_DAYS}d" --include 'projeto-nova-*.tar.age' "$REMOTE"
  log "fora: ok"
fi

date +%s >"$BACKUP_DIR/.ultimo-ok"
publicar_metricas

find "$BACKUP_DIR" -mindepth 1 -maxdepth 1 -type d -name '2*' -mtime +"$KEEP_DAYS" -print -exec rm -rf {} + |
  sed 's/^/removido (retenção): /'
