# ==============================================================================
# Projeto-nova-v2 no Proxmox VE (192.168.1.13) — VMs KVM + inventário Ansible
# O host 192.168.1.13 é a API do Proxmox, não a VM da aplicação.
# Produção: VMID 240 / projeto-nova-prod-app, clonada do template 9000
# (ubuntu-2404-cloud). Homologação: var.homologacao, desligada por padrão.
# As outras VMs do pve1 (201, 210, 220, 230) têm cada uma a sua aplicação:
# nunca reaproveite nem ligue uma delas daqui.
# Domínio: projetonova.cloud, pelo Cloudflare Tunnel configurado no painel.
# SSO oficial: sso.rondonopolis.mt.gov.br (não sobe o Keycloak de teste).
# ==============================================================================

locals {
  ambientes = merge(
    {
      producao = {
        vm_id   = var.vm_id
        nome    = var.vm_name
        ipv4    = var.app_ipv4
        cores   = var.vm_cores
        memoria = var.vm_memory
        disco   = var.vm_disk_size
        tag     = "prod"
      }
    },
    var.homologacao.criar ? {
      homologacao = {
        vm_id   = var.homologacao.vm_id
        nome    = var.homologacao.nome
        ipv4    = var.homologacao.ipv4
        cores   = var.homologacao.cores
        memoria = var.homologacao.memoria
        disco   = var.homologacao.disco
        tag     = "homolog"
      }
    } : {}
  )
}

# A VM de produção era um módulo só: muda de endereço no state, sem recriar.
moved {
  from = module.vm_app
  to   = module.vm["producao"]
}

module "vm" {
  source   = "./modules/proxmox_vm"
  for_each = local.ambientes

  node_name         = var.proxmox_node_name
  vm_id             = each.value.vm_id
  name              = each.value.nome
  description       = "VM KVM Docker — Projeto-nova-v2 (projetonova.cloud)"
  tags              = [each.value.tag, "docker", "projeto-nova", "kvm"]
  cores             = each.value.cores
  memory            = each.value.memoria
  disk_storage      = var.storage_disk_pool
  disk_size         = each.value.disco
  network_bridge    = var.network_bridge
  ipv4_address      = each.value.ipv4
  ipv4_gateway      = var.default_gateway
  admin_user        = "root"
  admin_password    = var.vm_admin_password
  ssh_public_keys   = compact([var.admin_ssh_public_key])
  clone_template_id = var.vm_template_id
  start_on_boot     = true
}

# Um grupo por ambiente (o nome casa com ansible/group_vars/<ambiente>.yml).
resource "local_file" "ansible_inventory" {
  filename        = "${path.module}/../ansible/inventory.ini"
  file_permission = "0644"
  content = join("\n", concat(
    [
      "# Gerado pelo Terraform — não edite à mão.",
      "[all:vars]",
      "ansible_user=root",
      "ansible_python_interpreter=/usr/bin/python3",
      "ansible_ssh_common_args='-o StrictHostKeyChecking=no'",
      "",
    ],
    flatten([for amb, vm in local.ambientes : [
      "[${amb}]",
      "${vm.nome} ansible_host=${split("/", vm.ipv4)[0]} proxmox_vmid=${module.vm[amb].id}",
      "",
    ]]),
    [
      "[docker_hosts:children]",
      join("\n", keys(local.ambientes)),
      "",
      "[app_servers:children]",
      join("\n", keys(local.ambientes)),
      "",
    ],
  ))
}
