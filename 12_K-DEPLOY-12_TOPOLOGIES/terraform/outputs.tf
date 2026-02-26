output "cluster_endpoint" {
  description = "EKS cluster API server endpoint"
  value       = module.eks.cluster_endpoint
  sensitive   = true
}

output "cluster_certificate_authority_data" {
  description = "Base64-encoded certificate authority data for EKS"
  value       = module.eks.cluster_certificate_authority_data
  sensitive   = true
}

output "cluster_name" {
  value = module.eks.cluster_name
}

output "kubeconfig_command" {
  description = "AWS CLI command to update kubeconfig"
  value       = "aws eks update-kubeconfig --region ${var.region} --name ${module.eks.cluster_name}"
}

output "rds_endpoint" {
  description = "RDS PostgreSQL endpoint"
  value       = aws_db_instance.kubric.endpoint
  sensitive   = true
}

output "redis_endpoint" {
  description = "ElastiCache Redis primary endpoint"
  value       = aws_elasticache_replication_group.kubric.primary_endpoint_address
  sensitive   = true
}

output "s3_evidence_bucket" {
  description = "S3 bucket name for forensic evidence storage"
  value       = aws_s3_bucket.kubric["evidence"].id
}

output "s3_backups_bucket" {
  value = aws_s3_bucket.kubric["backups"].id
}

output "vpc_id" {
  value = module.vpc.vpc_id
}
