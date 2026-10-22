output "sismarq_app_vm_id" {
  description = "VMID da aplicação no Proxmox"
  value       = module.vm_sismarq_app.id
}

output "sismarq_app_vm_ip" {
  description = "IPv4 da VM de aplicação"
  value       = module.vm_sismarq_app.ipv4_address
}

output "proxmox_api" {
  description = "API do hypervisor (não é a URL do SISMARQ)"
  value       = var.proxmox_endpoint
}

output "ansible_inventory_path" {
  description = "Inventário gerado para o Ansible"
  value       = local_file.ansible_inventory.filename
}
