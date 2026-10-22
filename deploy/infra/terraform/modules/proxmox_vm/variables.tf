variable "node_name" {
  description = "Nó do Proxmox onde a VM será criada"
  type        = string
}

variable "vm_id" {
  description = "ID único da VM no Proxmox (ex: 210)"
  type        = number
}

variable "name" {
  description = "Nome da máquina virtual"
  type        = string
}

variable "description" {
  description = "Anotação explicativa da VM"
  type        = string
  default     = "VM gerenciada via Terraform — SISMARQ"
}

variable "tags" {
  description = "Tags de classificação no Proxmox"
  type        = list(string)
  default     = ["terraform", "sismarq", "vm"]
}

variable "cores" {
  description = "Número de vCPUs"
  type        = number
  default     = 4
}

variable "memory" {
  description = "Memória RAM em MB"
  type        = number
  default     = 8192
}

variable "disk_storage" {
  description = "Storage onde ficará o disco da VM (ex: zfs ou local-lvm)"
  type        = string
  default     = "zfs"
}

variable "disk_size" {
  description = "Tamanho do disco em GB"
  type        = number
  default     = 60
}

variable "network_bridge" {
  description = "Bridge de rede (ex: vmbr0)"
  type        = string
  default     = "vmbr0"
}

variable "ipv4_address" {
  description = "Endereço IPv4 com CIDR (ex: 192.168.1.42/24) ou 'dhcp'"
  type        = string
  default     = "dhcp"
}

variable "ipv4_gateway" {
  description = "Gateway da rede IPv4 (ex: 192.168.1.1)"
  type        = string
  default     = ""
}

variable "admin_user" {
  description = "Nome do usuário administrador configurado via Cloud-Init"
  type        = string
  default     = "root"
}

variable "ssh_public_keys" {
  description = "Lista de chaves SSH públicas autorizadas"
  type        = list(string)
  default     = []
}

variable "admin_password" {
  description = "Senha do usuário administrador para acesso ao console NoVNC"
  type        = string
  default     = null
  sensitive   = true
}

variable "clone_template_id" {
  description = "ID da VM de template para clonar (opcional)"
  type        = number
  default     = null
}

variable "start_on_boot" {
  description = "Iniciar a VM automaticamente"
  type        = bool
  default     = true
}

variable "user_data_file_id" {
  description = "ID do arquivo/snippet de user-data do Cloud-Init no Proxmox"
  type        = string
  default     = null
}
