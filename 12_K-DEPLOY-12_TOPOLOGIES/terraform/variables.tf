variable "region" {
  description = "AWS region for all resources"
  type        = string
  default     = "us-east-1"
}

variable "cluster_name" {
  description = "EKS cluster name (also used as resource prefix)"
  type        = string
  default     = "kubric-prod"
}

variable "environment" {
  description = "Deployment environment: development | staging | production"
  type        = string
  default     = "production"
  validation {
    condition     = contains(["development", "staging", "production"], var.environment)
    error_message = "environment must be development, staging, or production"
  }
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "node_count" {
  description = "Desired number of EKS worker nodes (general node group)"
  type        = number
  default     = 3
}

variable "node_instance_type" {
  description = "EC2 instance type for general EKS worker nodes"
  type        = string
  default     = "m6i.2xlarge"
}

variable "db_instance_class" {
  description = "RDS PostgreSQL instance class"
  type        = string
  default     = "db.t4g.medium"
}

variable "enable_velociraptor" {
  description = "Deploy Velociraptor DFIR sidecar"
  type        = bool
  default     = true
}

variable "enable_wazuh" {
  description = "Deploy Wazuh SIEM"
  type        = bool
  default     = true
}
