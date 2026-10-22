output "id" {
  description = "ID do container no Proxmox"
  value       = proxmox_virtual_environment_container.this.vm_id
}

output "hostname" {
  description = "Hostname do container"
  value       = var.hostname
}

output "ipv4_address" {
  description = "Endereço IPv4 configurado"
  value       = var.ipv4_address
}

output "tags" {
  description = "Tags do container"
  value       = proxmox_virtual_environment_container.this.tags
}
