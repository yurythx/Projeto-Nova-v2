#!/usr/bin/env bash
# Roda o importarglpi (ADR 072) de uma máquina que alcança o GLPI, gravando
# no banco da VM por um túnel SSH. Serve quando a VM não enxerga a rede do
# GLPI (o caso da produção: o GLPI só existe no DNS interno da prefeitura).
#
#   bash scripts/importar-glpi.sh                        # simulação
#   bash scripts/importar-glpi.sh --unidade <uuid>       # ofertas e chamados numa unidade
#   bash scripts/importar-glpi.sh --gravar               # grava
#
# Variáveis:
#   GLPI_ENV      arquivo com GLPI_URL, GLPI_APP_TOKEN e GLPI_USER_TOKEN
#                 (padrão ~/.config/projeto-nova/glpi.env, fora do repositório)
#   NOVA_SSH      root@<vm> (padrão root@192.168.1.70)
#   NOVA_SSH_KEY  chave que entra na VM (padrão ~/.ssh/sismarq_deploy)
#
# Nada de segredo passa pela linha de comando nem pela tela: os tokens vêm
# do arquivo e a senha do banco é lida no .env da VM, dentro do container.
# O banco continua sem porta aberta; o túnel vive só durante o comando.
set -euo pipefail

raiz="$(cd "$(dirname "$0")/.." && pwd)"
glpi_env="${GLPI_ENV:-$HOME/.config/projeto-nova/glpi.env}"
vm="${NOVA_SSH:-root@192.168.1.70}"
chave="${NOVA_SSH_KEY:-$HOME/.ssh/sismarq_deploy}"

[ -f "$glpi_env" ] || { echo "Falta $glpi_env (GLPI_URL, GLPI_APP_TOKEN, GLPI_USER_TOKEN)" >&2; exit 1; }
grep -q '^GLPI_USER_TOKEN=.' "$glpi_env" || { echo "GLPI_USER_TOKEN vazio em $glpi_env" >&2; exit 1; }
[ -f "$chave" ] || { echo "Chave SSH não encontrada: $chave (defina NOVA_SSH_KEY)" >&2; exit 1; }

caminho() { if command -v cygpath >/dev/null; then cygpath -m "$1"; else printf '%s' "$1"; fi; }
export MSYS_NO_PATHCONV=1
tty=(-i)
[ -t 0 ] && tty=(-it)

docker run --rm "${tty[@]}" \
  -v "$(caminho "$raiz"):/src" -v "$(caminho "$glpi_env"):/glpi.env:ro" -v "$(caminho "$chave"):/chave:ro" \
  -v nova-gomod:/go/pkg/mod -v nova-gobuild:/root/.cache/go-build \
  -e VM="$vm" -w /src/backend golang:1.26 bash -ec '
    mkdir -p /root/.ssh && cp /chave /root/.ssh/id && chmod 600 /root/.ssh/id
    ssh_vm() { ssh -i /root/.ssh/id -o BatchMode=yes -o StrictHostKeyChecking=accept-new "$VM" "$@"; }
    pg=$(ssh_vm "cd /opt/projeto-nova && docker inspect -f \"{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}\" \$(docker compose ps -q postgres)" | cut -d" " -f1)
    ssh_vm -f -N -o ExitOnForwardFailure=yes -L "15432:$pg:5432"
    set -a
    . /glpi.env
    eval "$(ssh_vm "grep -E \"^DB_(NAME|USER|PASSWORD)=\" /opt/projeto-nova/.env")"
    set +a
    export DB_HOST=127.0.0.1 DB_PORT=15432 DB_SSLMODE=disable
    go run ./cmd/importarglpi "$@"
  ' importar-glpi "$@"
