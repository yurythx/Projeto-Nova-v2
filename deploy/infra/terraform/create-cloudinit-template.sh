#!/usr/bin/env bash
# ==============================================================================
# Script Auxiliar para Proxmox VE: Criar Template Cloud-Init (VM 9000)
# Execute este script diretamente no shell do Proxmox Host (pve1).
# ==============================================================================
set -euo pipefail

TEMPLATE_ID="${1:-9000}"
TEMPLATE_NAME="${2:-debian-12-cloudinit}"
STORAGE_POOL="${3:-zfs}"
MEMORY="${4:-2048}"
CORES="${5:-2}"
BRIDGE="${6:-vmbr0}"

echo "==> Verificando se o template VM $TEMPLATE_ID já existe..."
if qm status "$TEMPLATE_ID" &>/dev/null; then
  echo "Template VM $TEMPLATE_ID já existe no Proxmox. Nenhuma ação necessária."
  exit 0
fi

WORKDIR=$(mktemp -d)
trap 'rm -rf "$WORKDIR"' EXIT
cd "$WORKDIR"

echo "==> Baixando imagem oficial Debian 12 Generic Cloud (QCOW2)..."
IMAGE_URL="https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-genericcloud-amd64.qcow2"
wget -q --show-progress "$IMAGE_URL" -O debian-cloud.qcow2

echo "==> Criando Máquina Virtual base (VMID: $TEMPLATE_ID)..."
qm create "$TEMPLATE_ID" \
  --name "$TEMPLATE_NAME" \
  --memory "$MEMORY" \
  --cores "$CORES" \
  --cpu host \
  --net0 "virtio,bridge=$BRIDGE"

echo "==> Importando disco para o storage pool '$STORAGE_POOL'..."
qm importdisk "$TEMPLATE_ID" debian-cloud.qcow2 "$STORAGE_POOL"

echo "==> Configurando controladora SCSI VirtIO e Cloud-Init drive..."
# Pega o nome do volume importado
DISK_NAME=$(pvesm list "$STORAGE_POOL" | grep "vm-$TEMPLATE_ID-disk" | awk '{print $1}' | tail -n 1)

qm set "$TEMPLATE_ID" \
  --scsihw virtio-scsi-pci \
  --scsi0 "$DISK_NAME,discard=on,ssd=1" \
  --ide2 "$STORAGE_POOL:cloudinit" \
  --boot c \
  --bootdisk scsi0 \
  --serial0 socket \
  --vga serial0 \
  --agent enabled=1

echo "==> Redimensionando disco base para 20G..."
qm disk resize "$TEMPLATE_ID" scsi0 20G

echo "==> Convertendo VM $TEMPLATE_ID em TEMPLATE..."
qm template "$TEMPLATE_ID"

echo "=================================================================="
echo "Sucesso! Template Cloud-Init VM $TEMPLATE_ID ($TEMPLATE_NAME) criado!"
echo "Agora o Terraform pode clonar esta VM automaticamente via Proxmox API."
echo "=================================================================="
