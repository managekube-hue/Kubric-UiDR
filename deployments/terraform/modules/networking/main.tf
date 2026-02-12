# Networking module
# Configures VPC, subnets, and security groups

variable "environment" {
  type = string
}

variable "cluster_name" {
  type = string
}

variable "vpc_cidr" {
  type = string
}

output "vpc_id" {
  value = ""
  # TODO: Implement VPC creation
}

output "subnet_ids" {
  value = []
  # TODO: Implement subnet creation
}

output "security_group_ids" {
  value = []
  # TODO: Implement security group creation
}
