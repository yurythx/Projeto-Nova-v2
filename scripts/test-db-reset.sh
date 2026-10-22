#!/usr/bin/env bash
# Recria o banco de teste vazio (TEMPLATE template0) e aplica as migrations.
# O CI já sobe um Postgres novo; localmente o mesmo banco acumula unidades
# e listas globais deixam de ser determinísticas.
#
#   make test-db-reset
#
# Docker (padrão desta máquina): container projeto-nova-test-pg na rede projeto-nova-test.
# Sem container: usa psql em DB_HOST (localhost) com as variáveis DB_*.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
NAME="${TEST_PG_CONTAINER:-projeto-nova-test-pg}"
DB="${TEST_DB_NAME:-nexus_test}"
USER="${TEST_DB_USER:-nexus}"
PASS="${TEST_DB_PASSWORD:-nexus_ci_password}"

recreate() {
  local psql_cmd=$1
  $psql_cmd -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '${DB}' AND pid <> pg_backend_pid();" >/dev/null
  $psql_cmd -d postgres -c "DROP DATABASE IF EXISTS ${DB};"
  $psql_cmd -d postgres -c "CREATE DATABASE ${DB} TEMPLATE template0;"
}

if docker inspect "$NAME" >/dev/null 2>&1; then
  recreate "docker exec -e PGPASSWORD=${PASS} ${NAME} psql -U ${USER}"
  NET=container:${NAME}
  HOST=127.0.0.1
  if docker network inspect projeto-nova-test >/dev/null 2>&1; then
    NET=projeto-nova-test
    HOST="${NAME}"
  fi
  docker run --rm --network "${NET}" \
    -v "${ROOT}/backend:/src" -w /src \
    -e DB_HOST="${HOST}" -e DB_PORT=5432 -e DB_NAME="${DB}" \
    -e DB_USER="${USER}" -e DB_PASSWORD="${PASS}" \
    golang:1.26 go run ./cmd/migrate up
else
  export PGPASSWORD="${DB_PASSWORD:-$PASS}"
  HOST="${DB_HOST:-localhost}"
  PORT="${DB_PORT:-5432}"
  recreate "psql -h ${HOST} -p ${PORT} -U ${USER}"
  (cd "${ROOT}/backend" && DB_HOST="$HOST" DB_PORT="$PORT" DB_NAME="$DB" DB_USER="${USER}" DB_PASSWORD="${PGPASSWORD}" go run ./cmd/migrate up)
fi
