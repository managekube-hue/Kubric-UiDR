output "proxmox_infrastructure" {
  description = "Proxmox infrastructure details"
  value       = module.proxmox_infrastructure
}

output "ceph_cluster_status" {
  description = "Ceph cluster status"
  value       = module.ceph_storage
}

output "networking_details" {
  description = "Networking configuration details"
  value       = module.networking
}

output "nats_release" {
  description = "NATS Helm release info"
  value       = helm_release.nats
}

output "temporal_release" {
  description = "Temporal Helm release info"
  value       = helm_release.temporal
}
