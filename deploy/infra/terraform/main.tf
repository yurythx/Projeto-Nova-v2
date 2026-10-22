# ==============================================================================
# Projeto-nova-v2 no Proxmox VE (192.168.1.13) — VM KVM + inventário Ansible
# O host 192.168.1.13 é a API do Proxmox, não a VM da aplicação.
# VMID 220 / projeto-nova-prod-app: VM nova. No pve1 a 201 e a 210 já estão
# ocupadas; clona o template 9000 (ubuntu-2404-cloud).
# SSO oficial: sso.rondonopolis.mt.gov.br (não sobe o Keycloak de teste).
# ==============================================================================

# O módulo se chamava vm_sismarq_app: o state troca de endereço sem
# recriar a VM.
moved {
  from = module.vm_sismarq_app
  to   = module.vm_app
}

module "vm_app" {
  source = "./modules/proxmox_vm"

  node_name         = var.proxmox_node_name
  vm_id             = var.vm_id
  name              = var.vm_name
  description       = "VM KVM Docker — Projeto-nova-v2 (SSO sso.rondonopolis.mt.gov.br)"
  tags              = ["prod", "docker", "projeto-nova", "kvm"]
  cores             = var.vm_cores
  memory            = var.vm_memory
  disk_storage      = var.storage_disk_pool
  disk_size         = var.vm_disk_size
  network_bridge    = var.network_bridge
  ipv4_address      = var.app_ipv4
  ipv4_gateway      = var.default_gateway
  admin_user        = "root"
  admin_password    = var.vm_admin_password
  ssh_public_keys   = compact([var.admin_ssh_public_key])
  clone_template_id = var.vm_template_id
  start_on_boot     = true
}

resource "local_file" "ansible_inventory" {
  filename        = "${path.module}/../ansible/inventory.ini"
  file_permission = "0644"
  content         = <<-EOT
    # Gerado pelo Terraform — não edite à mão.
    [all:vars]
    ansible_user=root
    ansible_python_interpreter=/usr/bin/python3
    ansible_ssh_common_args='-o StrictHostKeyChecking=no'

    [docker_hosts]
    ${var.vm_name} ansible_host=${split("/", var.app_ipv4)[0]} proxmox_vmid=${module.vm_app.id}

    [app_servers]
    ${var.vm_name}
  EOT
}
