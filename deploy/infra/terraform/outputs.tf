output "app_vm_id" {
  description = "VMID da produção no Proxmox"
  value       = module.vm["producao"].id
}

output "app_vm_ip" {
  description = "IPv4 da VM de produção"
  value       = module.vm["producao"].ipv4_address
}

output "vms" {
  description = "VMID e IPv4 de cada ambiente"
  value       = { for amb, vm in module.vm : amb => { id = vm.id, ipv4 = vm.ipv4_address } }
}

output "proxmox_api" {
  description = "API do hypervisor (não é a URL do Projeto-nova-v2)"
  value       = var.proxmox_endpoint
}

output "ansible_inventory_path" {
  description = "Inventário gerado para o Ansible"
  value       = local_file.ansible_inventory.filename
}
