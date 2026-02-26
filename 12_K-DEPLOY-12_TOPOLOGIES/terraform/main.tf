# Kubric Platform — AWS EKS Production Infrastructure
# terraform >= 1.9 | AWS provider ~> 5.0

terraform {
  required_version = ">= 1.9.0"
  required_providers {
    aws        = { source = "hashicorp/aws",        version = "~> 5.0"  }
    kubernetes = { source = "hashicorp/kubernetes",  version = "~> 2.30" }
    helm       = { source = "hashicorp/helm",        version = "~> 2.13" }
    vault      = { source = "hashicorp/vault",       version = "~> 4.3"  }
  }
  backend "s3" {
    bucket         = "kubric-terraform-state"
    key            = "prod/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "kubric-terraform-locks"
  }
}

provider "aws" {
  region = var.region
  default_tags {
    tags = {
      Project     = "kubric"
      Environment = var.environment
      ManagedBy   = "terraform"
    }
  }
}

# --- VPC ---
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.0"

  name = "${var.cluster_name}-vpc"
  cidr = var.vpc_cidr

  azs             = ["${var.region}a", "${var.region}b", "${var.region}c"]
  private_subnets = [cidrsubnet(var.vpc_cidr, 4, 0), cidrsubnet(var.vpc_cidr, 4, 1), cidrsubnet(var.vpc_cidr, 4, 2)]
  public_subnets  = [cidrsubnet(var.vpc_cidr, 4, 8), cidrsubnet(var.vpc_cidr, 4, 9), cidrsubnet(var.vpc_cidr, 4, 10)]

  enable_nat_gateway   = true
  single_nat_gateway   = var.environment != "production"
  enable_dns_hostnames = true
  enable_dns_support   = true

  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
  }
  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
    "karpenter.sh/discovery"          = var.cluster_name
  }
}

# --- EKS ---
module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.0"

  cluster_name    = var.cluster_name
  cluster_version = "1.30"

  vpc_id                         = module.vpc.vpc_id
  subnet_ids                     = module.vpc.private_subnet_ids
  cluster_endpoint_public_access = true

  cluster_addons = {
    coredns                = { most_recent = true }
    kube-proxy             = { most_recent = true }
    vpc-cni                = { most_recent = true }
    aws-ebs-csi-driver     = { most_recent = true }
  }

  eks_managed_node_groups = {
    general = {
      instance_types = [var.node_instance_type]
      min_size       = 2
      max_size       = 10
      desired_size   = var.node_count

      labels = { workload = "general" }
      taints = []

      block_device_mappings = {
        xvda = {
          device_name = "/dev/xvda"
          ebs = {
            volume_size           = 100
            volume_type           = "gp3"
            encrypted             = true
            delete_on_termination = true
          }
        }
      }
    }

    security = {
      instance_types = ["c6i.2xlarge"]
      min_size       = 1
      max_size       = 5
      desired_size   = 2

      labels = { workload = "security-tools" }
      taints = [{ key = "workload", value = "security", effect = "NO_SCHEDULE" }]
    }
  }

  # IRSA for AWS services
  enable_irsa = true
}

# --- RDS PostgreSQL ---
resource "aws_db_subnet_group" "kubric" {
  name       = "${var.cluster_name}-db"
  subnet_ids = module.vpc.private_subnet_ids
}

resource "aws_security_group" "rds" {
  name        = "${var.cluster_name}-rds"
  description = "Kubric PostgreSQL"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = module.vpc.private_subnets_cidr_blocks
  }
}

resource "aws_db_instance" "kubric" {
  identifier             = "${var.cluster_name}-postgres"
  engine                 = "postgres"
  engine_version         = "16.3"
  instance_class         = var.db_instance_class
  allocated_storage      = 100
  max_allocated_storage  = 1000
  storage_type           = "gp3"
  storage_encrypted      = true
  db_name                = "kubric"
  username               = "kubric"
  password               = random_password.db.result
  db_subnet_group_name   = aws_db_subnet_group.kubric.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  multi_az               = var.environment == "production"
  backup_retention_period = 7
  deletion_protection     = var.environment == "production"
  skip_final_snapshot     = var.environment != "production"
  performance_insights_enabled = true
  tags = { Name = "${var.cluster_name}-postgres" }
}

resource "random_password" "db" {
  length  = 32
  special = false
}

# --- ElastiCache Redis ---
resource "aws_elasticache_subnet_group" "kubric" {
  name       = "${var.cluster_name}-redis"
  subnet_ids = module.vpc.private_subnet_ids
}

resource "aws_elasticache_replication_group" "kubric" {
  replication_group_id = "${var.cluster_name}-redis"
  description          = "Kubric Redis cluster"
  node_type            = "cache.t4g.medium"
  num_cache_clusters   = var.environment == "production" ? 3 : 1
  automatic_failover_enabled = var.environment == "production"
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  engine_version             = "7.1"
  subnet_group_name          = aws_elasticache_subnet_group.kubric.name
}

# --- S3 Buckets ---
locals {
  s3_buckets = {
    evidence  = "${var.cluster_name}-evidence"
    backups   = "${var.cluster_name}-backups"
    state     = "kubric-terraform-state"
    artifacts = "${var.cluster_name}-build-artifacts"
  }
}

resource "aws_s3_bucket" "kubric" {
  for_each = local.s3_buckets
  bucket   = each.value
  tags     = { purpose = each.key }
}

resource "aws_s3_bucket_versioning" "kubric" {
  for_each = local.s3_buckets
  bucket   = aws_s3_bucket.kubric[each.key].id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "kubric" {
  for_each = local.s3_buckets
  bucket   = aws_s3_bucket.kubric[each.key].id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "kubric" {
  for_each               = local.s3_buckets
  bucket                 = aws_s3_bucket.kubric[each.key].id
  block_public_acls      = true
  block_public_policy    = true
  ignore_public_acls     = true
  restrict_public_buckets = true
}
