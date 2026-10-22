variable "node_name" {
  description = "Nó do Proxmox onde o container será criado"
  type        = string
}

variable "vm_id" {
  description = "ID único do container no Proxmox (ex: 200)"
  type        = number
}

variable "hostname" {
  description = "Nome do host / container"
  type        = string
}

variable "description" {
  description = "Descrição ou anotação do container"
  type        = string
  default     = "Gerenciado via Terraform - Projeto-nova-v2"
}

variable "tags" {
  description = "Tags de classificação no Proxmox"
  type        = list(string)
  default     = ["terraform", "nova", "lxc"]
}

variable "unprivileged" {
  description = "Container não-privilegiado (melhor prática de segurança)"
  type        = bool
  default     = true
}

variable "cores" {
  description = "Número de vCPUs alocadas"
  type        = number
  default     = 2
}

variable "memory" {
  description = "Memória RAM em MB"
  type        = number
  default     = 1024
}

variable "swap" {
  description = "Memória SWAP em MB"
  type        = number
  default     = 512
}

variable "disk_storage" {
  description = "Pool de armazenamento do disco rootfs"
  type        = string
  default     = "zfs"
}

variable "disk_size" {
  description = "Tamanho do disco em GB (inteiro)"
  type        = number
  default     = 16
}

variable "template_file_id" {
  description = "ID do template de container baixado no storage (ex: local:vztmpl/debian-12-standard_12.12-1_amd64.tar.zst)"
  type        = string
}

variable "os_type" {
  description = "Tipo do sistema operacional (debian, ubuntu, etc)"
  type        = string
  default     = "debian"
}

variable "network_bridge" {
  description = "Bridge de rede (ex: vmbr0)"
  type        = string
  default     = "vmbr0"
}

variable "ipv4_address" {
  description = "Endereço IPv4 com máscara CIDR (ex: 192.168.1.50/24) ou 'dhcp'"
  type        = string
  default     = "dhcp"
}

variable "ipv4_gateway" {
  description = "Gateway IPv4 padrão (ex: 192.168.1.1)"
  type        = string
  default     = ""
}

variable "ssh_public_keys" {
  description = "Lista de chaves SSH públicas autorizadas para o root"
  type        = list(string)
  default     = []
}

variable "root_password" {
  description = "Senha do root (opcional se usar SSH)"
  type        = string
  default     = null
  sensitive   = true
}

variable "start_on_boot" {
  description = "Iniciar o container automaticamente após criação"
  type        = bool
  default     = true
}

variable "nesting" {
  description = "Habilitar nesting para suporte a containers Docker/LXC internos"
  type        = bool
  default     = false
}

variable "keyctl" {
  description = "Habilitar keyctl para suporte ao gerenciamento de chaves do kernel em containers unprivileged"
  type        = bool
  default     = false
}
