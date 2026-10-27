#!/usr/bin/env bash
# make infra-deploy sem Ansible instalado: o playbook roda num container,
# e só o Docker é preciso (Windows com Git Bash, WSL ou Linux).
#
#   bash scripts/infra-deploy.sh                       # todos os ambientes
#   bash scripts/infra-deploy.sh --limit producao      # argumentos do ansible-playbook
#
# Variáveis:
#   NOVA_SSH_KEY   chave que entra nas VMs (padrão ~/.ssh/sismarq_deploy)
#   VAULT_PASS     arquivo com a senha do cofre (padrão
#                  ~/.config/projeto-nova/vault-pass); sem ele, o Ansible pergunta
set -euo pipefail

raiz="$(cd "$(dirname "$0")/.." && pwd)"
ansible_dir="$raiz/deploy/infra/ansible"
chave="${NOVA_SSH_KEY:-$HOME/.ssh/sismarq_deploy}"
vault_pass="${VAULT_PASS:-$HOME/.config/projeto-nova/vault-pass}"

[ -f "$chave" ] || { echo "Chave SSH não encontrada: $chave (defina NOVA_SSH_KEY)" >&2; exit 1; }
[ -f "$ansible_dir/inventory.ini" ] || { echo "Falta ansible/inventory.ini: rode make infra-apply" >&2; exit 1; }

# No Git Bash, o Docker quer caminhos do Windows e não pode reescrever os
# do container.
caminho() { if command -v cygpath >/dev/null; then cygpath -m "$1"; else printf '%s' "$1"; fi; }
export MSYS_NO_PATHCONV=1

segredos=()
for f in secrets.yml cloudflare.secrets.yml; do
  [ -f "$ansible_dir/$f" ] && segredos+=(-e "@$f")
done
montagens=(-v "$(caminho "$raiz"):/work" -v "$(caminho "$chave"):/chave:ro")
if [ -f "$ansible_dir/secrets.vault.yml" ]; then
  segredos+=(-e @secrets.vault.yml)
  if [ -f "$vault_pass" ]; then
    montagens+=(-v "$(caminho "$vault_pass"):/vault-pass:ro")
    segredos+=(--vault-password-file /vault-pass)
  else
    segredos+=(--ask-vault-pass)
  fi
fi

tty=(-i)
[ -t 0 ] && tty=(-it)

# ansible.cfg numa pasta montada do Windows fica "gravável por todos", e o
# Ansible o ignora: ANSIBLE_CONFIG o carrega assim mesmo.
docker run --rm "${tty[@]}" "${montagens[@]}" \
  -e ANSIBLE_CONFIG=/work/deploy/infra/ansible/ansible.cfg \
  -w /work/deploy/infra/ansible \
  python:3.12-slim sh -ec '
    apt-get update -qq >/dev/null
    apt-get install -y -qq --no-install-recommends openssh-client git >/dev/null
    pip install -q --root-user-action=ignore ansible-core==2.18.6
    mkdir -p /root/.ssh && cp /chave /root/.ssh/id && chmod 600 /root/.ssh/id
    ansible-galaxy collection install -r requirements.yml >/dev/null
    exec ansible-playbook site.yml --private-key /root/.ssh/id "$@"
  ' infra-deploy "${segredos[@]}" "$@"
