variable "proxmox_endpoint" {
  description = "URL da API do Proxmox VE (ex: https://192.168.1.13:8006/)"
  type        = string
  default     = "https://192.168.1.13:8006/"
}

variable "proxmox_insecure" {
  description = "Permitir certificados TLS autoassinados no Proxmox"
  type        = bool
  default     = true
}

variable "proxmox_api_token_id" {
  description = "ID completo do Token de API (ex: terraform-prov@pve!tf-token)"
  type        = string
  default     = "terraform-prov@pve!tf-token"
}

variable "proxmox_api_token_secret" {
  description = "Segredo gerado do Token de API"
  type        = string
  sensitive   = true
}

variable "proxmox_node_name" {
  description = "Nome do nó do cluster Proxmox"
  type        = string
  default     = "pve1"
}

variable "storage_disk_pool" {
  description = "Storage para discos das VMs e rootfs dos containers"
  type        = string
  default     = "zfs"
}

variable "storage_template_pool" {
  description = "Storage onde ficam os templates LXC e imagens ISO"
  type        = string
  default     = "local"
}

variable "network_bridge" {
  description = "Interface de bridge de rede principal"
  type        = string
  default     = "vmbr0"
}

variable "admin_ssh_public_key" {
  description = "Chave SSH pública injetada nas VMs e containers para acesso do Ansible"
  type        = string
  default     = ""
}

variable "default_gateway" {
  description = "Gateway padrão da rede local"
  type        = string
  default     = "192.168.1.1"
}

variable "vm_template_id" {
  description = "ID do template Cloud-Init da VM no Proxmox (ex: 9000)"
  type        = number
  default     = 9000
}

variable "vm_id" {
  description = "VMID exclusivo desta instância (201, 210, 220 e 230 são de outras aplicações)"
  type        = number
  default     = 240
}

variable "vm_name" {
  description = "Nome da VM de aplicação"
  type        = string
  default     = "projeto-nova-prod-app"
}

variable "app_ipv4" {
  description = "IPv4 da VM de aplicação (CIDR). Não use o IP do Proxmox (192.168.1.13)."
  type        = string
  default     = "192.168.1.70/24"
}

variable "vm_cores" {
  type    = number
  default = 4
}

variable "vm_memory" {
  description = "RAM em MB da VM de aplicação"
  type        = number
  default     = 4096
}

variable "vm_disk_size" {
  type    = number
  default = 60
}

variable "vm_admin_password" {
  description = "Senha administrativa da VM para acesso via console NoVNC"
  type        = string
  sensitive   = true
}

variable "homologacao" {
  description = "VM de homologação. Desligada por padrão: o pve1 tem 11 GB de RAM e cada VM de outra aplicação reserva 4 GB."
  type = object({
    criar   = bool
    vm_id   = optional(number, 241)
    nome    = optional(string, "projeto-nova-homolog-app")
    ipv4    = optional(string, "192.168.1.71/24")
    cores   = optional(number, 2)
    memoria = optional(number, 3072)
    disco   = optional(number, 40)
  })
  default = { criar = false }
}
