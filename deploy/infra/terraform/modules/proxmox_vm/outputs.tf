output "id" {
  description = "ID da VM no Proxmox"
  value       = proxmox_virtual_environment_vm.this.vm_id
}

output "name" {
  description = "Nome da VM"
  value       = var.name
}

output "ipv4_address" {
  description = "Endereço IPv4 configurado"
  value       = var.ipv4_address
}

output "tags" {
  description = "Tags atribuídas à VM"
  value       = proxmox_virtual_environment_vm.this.tags
}
