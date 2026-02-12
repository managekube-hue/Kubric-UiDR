# Proxmox module for infrastructure provisioning
# This module creates VMs in Proxmox for Kubernetes nodes

variable "environment" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "vm_count" {
  type = number
}

variable "cpu_cores" {
  type = number
}

variable "memory_mb" {
  type = number
}

output "node_ids" {
  value = []
  # TODO: Implement Proxmox VM creation
}

output "node_ips" {
  value = []
  # TODO: Implement Proxmox VM creation
}
