provider "proxmox" {
  endpoint = var.proxmox_endpoint
  insecure = var.proxmox_insecure

  api_token = "${var.proxmox_api_token_id}=${var.proxmox_api_token_secret}"

  ssh {
    agent    = true
    username = "root"
  }
}
