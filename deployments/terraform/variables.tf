variable "environment" {
  description = "Environment name (prod, staging)"
  type        = string
  validation {
    condition     = contains(["prod", "staging"], var.environment)
    error_message = "Environment must be either 'prod' or 'staging'."
  }
}

variable "cluster_name" {
  description = "Kubernetes cluster name"
  type        = string
}

variable "proxmox_endpoint" {
  description = "Proxmox API endpoint"
  type        = string
  sensitive   = true
}

variable "proxmox_username" {
  description = "Proxmox username"
  type        = string
  sensitive   = true
}

variable "proxmox_password" {
  description = "Proxmox password"
  type        = string
  sensitive   = true
}

variable "proxmox_insecure" {
  description = "Proxmox insecure TLS"
  type        = bool
  default     = false
}

variable "proxmox_vm_count" {
  description = "Number of VMs to create"
  type        = number
  default     = 3
}

variable "proxmox_cpu_cores" {
  description = "CPU cores per VM"
  type        = number
  default     = 8
}

variable "proxmox_memory_mb" {
  description = "Memory in MB per VM"
  type        = number
  default     = 16384
}

variable "ceph_osd_count" {
  description = "Number of Ceph OSDs"
  type        = number
  default     = 3
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "kubeconfig_path" {
  description = "Path to kubeconfig file"
  type        = string
  default     = "~/.kube/config"
}
