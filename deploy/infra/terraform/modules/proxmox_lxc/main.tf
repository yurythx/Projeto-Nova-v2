terraform {
  required_providers {
    proxmox = {
      source  = "bpg/proxmox"
      version = ">= 0.68.0"
    }
  }
}

resource "proxmox_virtual_environment_container" "this" {
  node_name    = var.node_name
  vm_id        = var.vm_id
  description  = var.description
  tags         = var.tags
  unprivileged = var.unprivileged

  cpu {
    cores = var.cores
  }

  memory {
    dedicated = var.memory
    swap      = var.swap
  }

  disk {
    datastore_id = var.disk_storage
    size         = var.disk_size
  }

  dynamic "features" {
    for_each = var.nesting ? [1] : []
    content {
      nesting = var.nesting
    }
  }

  initialization {
    hostname = var.hostname

    ip_config {
      ipv4 {
        address = var.ipv4_address
        gateway = var.ipv4_gateway != "" ? var.ipv4_gateway : null
      }
    }

    user_account {
      keys     = length(var.ssh_public_keys) > 0 ? var.ssh_public_keys : null
      password = var.root_password
    }
  }

  operating_system {
    template_file_id = var.template_file_id
    type             = var.os_type
  }

  network_interface {
    name   = "eth0"
    bridge = var.network_bridge
  }

  started = var.start_on_boot
}
