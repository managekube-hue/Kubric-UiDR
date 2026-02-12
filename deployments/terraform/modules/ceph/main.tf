# Ceph storage module
# Configures Ceph distributed storage cluster

variable "environment" {
  type = string
}

variable "proxmox_nodes" {
  type = list(string)
}

variable "osd_count" {
  type = number
}

output "ceph_cluster_status" {
  value = "pending"
  # TODO: Implement Ceph OSDs creation
}

output "osd_ids" {
  value = []
  # TODO: Implement Ceph OSDs creation
}
