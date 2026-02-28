# ─────────────────────────────────────────────────────────────────────
# Terraform – AWS EKS Cluster for Kubric UiDR Large Deployment
# ─────────────────────────────────────────────────────────────────────
# Target: > 1 000 monitored endpoints
# Provisions:
#   • EKS 1.29 control plane with OIDC for IRSA
#   • Managed node groups: general, inference (GPU), storage
#   • EBS CSI, CoreDNS, kube-proxy, VPC-CNI managed add-ons
# ─────────────────────────────────────────────────────────────────────

terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.40"
    }
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.27"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
  }

  backend "s3" {
    bucket         = "kubric-terraform-state"
    key            = "eks/large/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "kubric-terraform-locks"
    encrypt        = true
  }
}

# ── Variables ────────────────────────────────────────────────────────

variable "aws_region" {
  description = "AWS region for the EKS cluster"
  type        = string
  default     = "us-east-1"
}

variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  default     = "kubric-prod"
}

variable "cluster_version" {
  description = "Kubernetes version for EKS"
  type        = string
  default     = "1.29"
}

variable "vpc_id" {
  description = "VPC ID where EKS will be deployed (from K-DEPLOY-LG-002)"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for EKS worker nodes"
  type        = list(string)
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"
}

variable "tags" {
  description = "Additional tags for all resources"
  type        = map(string)
  default     = {}
}

# ── Provider ─────────────────────────────────────────────────────────

provider "aws" {
  region = var.aws_region

  default_tags {
    tags = merge({
      Project     = "kubric-uidr"
      Environment = var.environment
      ManagedBy   = "terraform"
      Component   = "eks"
    }, var.tags)
  }
}

# ── Data Sources ─────────────────────────────────────────────────────

data "aws_caller_identity" "current" {}
data "aws_partition" "current" {}

data "aws_ami" "eks_optimized" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["amazon-eks-node-${var.cluster_version}-v*"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }
}

data "aws_ami" "eks_gpu_optimized" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["amazon-eks-gpu-node-${var.cluster_version}-v*"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }
}

# ── KMS Key for Envelope Encryption ─────────────────────────────────

resource "aws_kms_key" "eks_secrets" {
  description             = "KMS key for EKS secrets envelope encryption – ${var.cluster_name}"
  deletion_window_in_days = 14
  enable_key_rotation     = true

  tags = {
    Name = "${var.cluster_name}-eks-secrets"
  }
}

resource "aws_kms_alias" "eks_secrets" {
  name          = "alias/${var.cluster_name}-eks-secrets"
  target_key_id = aws_kms_key.eks_secrets.key_id
}

# ── EKS Cluster (terraform-aws-modules) ─────────────────────────────

module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.8"

  cluster_name    = var.cluster_name
  cluster_version = var.cluster_version

  vpc_id     = var.vpc_id
  subnet_ids = var.private_subnet_ids

  # Control-plane logging
  cluster_enabled_log_types = [
    "api",
    "audit",
    "authenticator",
    "controllerManager",
    "scheduler",
  ]

  # Envelope encryption of Kubernetes secrets with KMS
  cluster_encryption_config = {
    provider_key_arn = aws_kms_key.eks_secrets.arn
    resources        = ["secrets"]
  }

  # OIDC provider for IAM Roles for Service Accounts (IRSA)
  enable_irsa = true

  # Public + private endpoint access (restrict in production via CIDR)
  cluster_endpoint_public_access  = true
  cluster_endpoint_private_access = true
  cluster_endpoint_public_access_cidrs = ["0.0.0.0/0"] # Tighten in prod

  # ── Managed add-ons ──────────────────────────────────────────────
  cluster_addons = {
    coredns = {
      most_recent = true
      configuration_values = jsonencode({
        computeType = "Fargate"
        replicaCount = 3
      })
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent              = true
      service_account_role_arn = module.vpc_cni_irsa.iam_role_arn
      configuration_values = jsonencode({
        env = {
          ENABLE_PREFIX_DELEGATION = "true"
          WARM_PREFIX_TARGET       = "1"
        }
      })
    }
    aws-ebs-csi-driver = {
      most_recent              = true
      service_account_role_arn = module.ebs_csi_irsa.iam_role_arn
    }
  }

  # ── Managed Node Groups ─────────────────────────────────────────
  eks_managed_node_groups = {

    # General-purpose – Kubric API, NATS, control-plane services
    general = {
      name            = "${var.cluster_name}-general"
      instance_types  = ["m5.2xlarge"]
      ami_id          = data.aws_ami.eks_optimized.id
      capacity_type   = "ON_DEMAND"

      min_size     = 3
      max_size     = 15
      desired_size = 10

      disk_size = 100 # GB

      labels = {
        "kubric.io/node-role" = "general"
        "kubric.io/workload"  = "services"
      }

      tags = {
        Name       = "${var.cluster_name}-general"
        NodeGroup  = "general"
      }
    }

    # GPU inference – vLLM, model serving
    inference = {
      name            = "${var.cluster_name}-inference"
      instance_types  = ["g5.2xlarge"]
      ami_id          = data.aws_ami.eks_gpu_optimized.id
      capacity_type   = "ON_DEMAND"

      min_size     = 2
      max_size     = 10
      desired_size = 5

      disk_size = 200 # GB – model weights

      labels = {
        "kubric.io/node-role" = "inference"
        "kubric.io/workload"  = "gpu"
        "nvidia.com/gpu"      = "true"
      }

      taints = [
        {
          key    = "nvidia.com/gpu"
          value  = "true"
          effect = "NO_SCHEDULE"
        }
      ]

      tags = {
        Name       = "${var.cluster_name}-inference"
        NodeGroup  = "inference"
      }
    }

    # Storage-optimised – ClickHouse, MinIO, PostgreSQL
    storage = {
      name            = "${var.cluster_name}-storage"
      instance_types  = ["r5.2xlarge"]
      ami_id          = data.aws_ami.eks_optimized.id
      capacity_type   = "ON_DEMAND"

      min_size     = 3
      max_size     = 6
      desired_size = 3

      disk_size = 500 # GB

      labels = {
        "kubric.io/node-role" = "storage"
        "kubric.io/workload"  = "data"
      }

      taints = [
        {
          key    = "kubric.io/storage"
          value  = "true"
          effect = "NO_SCHEDULE"
        }
      ]

      tags = {
        Name       = "${var.cluster_name}-storage"
        NodeGroup  = "storage"
      }
    }
  }

  tags = {
    Cluster = var.cluster_name
  }
}

# ── IRSA: VPC CNI ───────────────────────────────────────────────────

module "vpc_cni_irsa" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.37"

  role_name             = "${var.cluster_name}-vpc-cni"
  attach_vpc_cni_policy = true
  vpc_cni_enable_ipv4   = true

  oidc_providers = {
    main = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["kube-system:aws-node"]
    }
  }
}

# ── IRSA: EBS CSI Driver ────────────────────────────────────────────

module "ebs_csi_irsa" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.37"

  role_name             = "${var.cluster_name}-ebs-csi"
  attach_ebs_csi_policy = true

  oidc_providers = {
    main = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["kube-system:ebs-csi-controller-sa"]
    }
  }
}

# ── IRSA: Kubric API (S3, SES, SQS access) ──────────────────────────

module "kubric_api_irsa" {
  source  = "terraform-aws-modules/iam/aws//modules/iam-role-for-service-accounts-eks"
  version = "~> 5.37"

  role_name = "${var.cluster_name}-kubric-api"

  role_policy_arns = {
    policy = aws_iam_policy.kubric_api.arn
  }

  oidc_providers = {
    main = {
      provider_arn               = module.eks.oidc_provider_arn
      namespace_service_accounts = ["kubric:kubric-api"]
    }
  }
}

resource "aws_iam_policy" "kubric_api" {
  name        = "${var.cluster_name}-kubric-api"
  description = "Kubric API service permissions"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "S3Artifacts"
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject",
          "s3:ListBucket",
          "s3:DeleteObject",
        ]
        Resource = [
          "arn:${data.aws_partition.current.partition}:s3:::kubric-artifacts-${data.aws_caller_identity.current.account_id}",
          "arn:${data.aws_partition.current.partition}:s3:::kubric-artifacts-${data.aws_caller_identity.current.account_id}/*",
        ]
      },
      {
        Sid    = "SQSQueues"
        Effect = "Allow"
        Action = [
          "sqs:SendMessage",
          "sqs:ReceiveMessage",
          "sqs:DeleteMessage",
          "sqs:GetQueueAttributes",
        ]
        Resource = "arn:${data.aws_partition.current.partition}:sqs:${var.aws_region}:${data.aws_caller_identity.current.account_id}:kubric-*"
      },
      {
        Sid    = "KMS"
        Effect = "Allow"
        Action = [
          "kms:Decrypt",
          "kms:GenerateDataKey",
        ]
        Resource = aws_kms_key.eks_secrets.arn
      },
    ]
  })
}

# ── Outputs ──────────────────────────────────────────────────────────

output "cluster_name" {
  description = "EKS cluster name"
  value       = module.eks.cluster_name
}

output "cluster_endpoint" {
  description = "EKS cluster API endpoint"
  value       = module.eks.cluster_endpoint
}

output "cluster_certificate_authority_data" {
  description = "Base64-encoded CA cert for the cluster"
  value       = module.eks.cluster_certificate_authority_data
  sensitive   = true
}

output "oidc_provider_arn" {
  description = "OIDC provider ARN for IRSA"
  value       = module.eks.oidc_provider_arn
}

output "cluster_security_group_id" {
  description = "Security group ID attached to the EKS cluster"
  value       = module.eks.cluster_security_group_id
}

output "node_security_group_id" {
  description = "Security group ID attached to EKS managed node groups"
  value       = module.eks.node_security_group_id
}
