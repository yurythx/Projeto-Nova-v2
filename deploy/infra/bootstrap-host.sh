#!/usr/bin/env bash
# Bootstrap da VM SISMARQ quando o Ansible não roda no Windows.
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
PUBLIC_HOST="${1:-192.168.1.52}"
APP_DIR=/opt/sismarq

apt-get update
apt-get install -y curl wget git ca-certificates gnupg ufw fail2ban

timedatectl set-timezone America/Cuiaba || true

if ! command -v docker >/dev/null 2>&1; then
  curl -fsSL https://get.docker.com | sh
  systemctl enable --now docker
fi

ufw allow 22/tcp || true
ufw allow 80/tcp || true
ufw allow 443/tcp || true
ufw allow 3010/tcp || true
ufw allow 8010/tcp || true
ufw --force enable || true

mkdir -p "$(dirname "$APP_DIR")"
if [ ! -d "$APP_DIR/.git" ]; then
  git clone https://github.com/yurythx/SISMARQ.git "$APP_DIR"
else
  git -C "$APP_DIR" fetch --all
  git -C "$APP_DIR" checkout main
  git -C "$APP_DIR" pull --ff-only
fi

chmod +x "$APP_DIR/scripts/deploy.sh"
cd "$APP_DIR"
bash scripts/deploy.sh --no-pull "$PUBLIC_HOST"

echo "BOOTSTRAP_OK"
