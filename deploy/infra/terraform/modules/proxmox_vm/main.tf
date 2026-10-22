terraform {
  required_providers {
    proxmox = {
      source  = "bpg/proxmox"
      version = ">= 0.68.0"
    }
  }
}

resource "proxmox_virtual_environment_vm" "this" {
  node_name   = var.node_name
  vm_id       = var.vm_id
  name        = var.name
  description = var.description
  tags        = var.tags

  cpu {
    cores = var.cores
    type  = "host"
  }

  memory {
    dedicated = var.memory
  }

  agent {
    enabled = false
  }

  disk {
    datastore_id = var.disk_storage
    size         = var.disk_size
    interface    = "scsi0"
    file_format  = "raw"
    discard      = "on"
    ssd          = true
  }

  network_device {
    bridge = var.network_bridge
    model  = "virtio"
  }

  dynamic "clone" {
    for_each = var.clone_template_id != null ? [1] : []
    content {
      vm_id = var.clone_template_id
      full  = true
    }
  }

  initialization {
    ip_config {
      ipv4 {
        address = var.ipv4_address
        gateway = var.ipv4_gateway != "" ? var.ipv4_gateway : null
      }
    }

    user_account {
      username = var.admin_user
      password = var.admin_password
      keys     = length(var.ssh_public_keys) > 0 ? var.ssh_public_keys : null
    }
  }

  operating_system {
    type = "l26"
  }

  started = var.start_on_boot
}
