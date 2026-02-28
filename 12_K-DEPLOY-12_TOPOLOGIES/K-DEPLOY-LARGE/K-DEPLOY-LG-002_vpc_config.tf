# ─────────────────────────────────────────────────────────────────────
# Terraform – VPC Configuration for Kubric UiDR AWS Deployment
# ─────────────────────────────────────────────────────────────────────
# Creates a production VPC with:
#   • /16 CIDR split across 3 AZs
#   • Private subnets for EKS workloads
#   • Public subnets for ALB / NAT gateways
#   • Intra subnets for internal-only traffic (databases)
#   • NAT gateway (one per AZ for HA)
#   • VPC Flow Logs to CloudWatch
#   • Tags for EKS auto-discovery
# ─────────────────────────────────────────────────────────────────────

terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.40"
    }
  }
}

# ── Variables ────────────────────────────────────────────────────────

variable "vpc_name" {
  description = "Name for the VPC"
  type        = string
  default     = "kubric-prod-vpc"
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "cluster_name" {
  description = "EKS cluster name – used for subnet tagging"
  type        = string
  default     = "kubric-prod"
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"
}

variable "enable_vpn_gateway" {
  description = "Whether to create a VPN gateway for site-to-site VPN"
  type        = bool
  default     = true
}

variable "flow_log_retention_days" {
  description = "CloudWatch log group retention for VPC flow logs"
  type        = number
  default     = 90
}

# ── Data Sources ─────────────────────────────────────────────────────

data "aws_availability_zones" "available" {
  state = "available"

  filter {
    name   = "opt-in-status"
    values = ["opt-in-not-required"]
  }
}

locals {
  azs = slice(data.aws_availability_zones.available.names, 0, 3)

  # Subnet CIDR allocation:
  # Public:  10.0.0.0/22,  10.0.4.0/22,  10.0.8.0/22    (1022 IPs each)
  # Private: 10.0.16.0/20, 10.0.32.0/20, 10.0.48.0/20   (4094 IPs each)
  # Intra:   10.0.64.0/22, 10.0.68.0/22, 10.0.72.0/22   (1022 IPs each)
  public_subnets  = ["10.0.0.0/22", "10.0.4.0/22", "10.0.8.0/22"]
  private_subnets = ["10.0.16.0/20", "10.0.32.0/20", "10.0.48.0/20"]
  intra_subnets   = ["10.0.64.0/22", "10.0.68.0/22", "10.0.72.0/22"]
}

# ── VPC Module ───────────────────────────────────────────────────────

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.7"

  name = var.vpc_name
  cidr = var.vpc_cidr
  azs  = local.azs

  public_subnets  = local.public_subnets
  private_subnets = local.private_subnets
  intra_subnets   = local.intra_subnets

  # ── NAT Gateway (one per AZ for high availability) ───────────────
  enable_nat_gateway     = true
  single_nat_gateway     = false
  one_nat_gateway_per_az = true

  # ── VPN Gateway ──────────────────────────────────────────────────
  enable_vpn_gateway = var.enable_vpn_gateway

  # ── DNS ──────────────────────────────────────────────────────────
  enable_dns_hostnames = true
  enable_dns_support   = true

  # ── Public subnet tags (ALB controller auto-discovery) ──────────
  public_subnet_tags = {
    "kubernetes.io/role/elb"                      = "1"
    "kubernetes.io/cluster/${var.cluster_name}"    = "shared"
    "kubric.io/subnet-type"                       = "public"
  }

  # ── Private subnet tags (internal LB + EKS node placement) ──────
  private_subnet_tags = {
    "kubernetes.io/role/internal-elb"              = "1"
    "kubernetes.io/cluster/${var.cluster_name}"    = "shared"
    "kubric.io/subnet-type"                       = "private"
    "karpenter.sh/discovery"                       = var.cluster_name
  }

  # ── Intra subnet tags (no internet, databases only) ─────────────
  intra_subnet_tags = {
    "kubric.io/subnet-type" = "intra"
    "kubric.io/use"         = "databases"
  }

  tags = {
    Project     = "kubric-uidr"
    Environment = var.environment
    ManagedBy   = "terraform"
    Component   = "vpc"
  }
}

# ── VPC Flow Logs ────────────────────────────────────────────────────

resource "aws_cloudwatch_log_group" "vpc_flow_logs" {
  name              = "/aws/vpc/${var.vpc_name}/flow-logs"
  retention_in_days = var.flow_log_retention_days

  tags = {
    Name = "${var.vpc_name}-flow-logs"
  }
}

resource "aws_iam_role" "vpc_flow_logs" {
  name = "${var.vpc_name}-flow-logs-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "vpc-flow-logs.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy" "vpc_flow_logs" {
  name = "${var.vpc_name}-flow-logs-policy"
  role = aws_iam_role.vpc_flow_logs.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogGroups",
          "logs:DescribeLogStreams",
        ]
        Resource = "${aws_cloudwatch_log_group.vpc_flow_logs.arn}:*"
      }
    ]
  })
}

resource "aws_flow_log" "vpc" {
  vpc_id                   = module.vpc.vpc_id
  traffic_type             = "ALL"
  log_destination_type     = "cloud-watch-logs"
  log_destination          = aws_cloudwatch_log_group.vpc_flow_logs.arn
  iam_role_arn             = aws_iam_role.vpc_flow_logs.arn
  max_aggregation_interval = 60 # seconds

  tags = {
    Name = "${var.vpc_name}-flow-log"
  }
}

# ── VPC Endpoints (reduce NAT costs & improve latency) ───────────────

resource "aws_security_group" "vpc_endpoints" {
  name_prefix = "${var.vpc_name}-vpce-"
  description = "Security group for VPC endpoints"
  vpc_id      = module.vpc.vpc_id

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = [var.vpc_cidr]
    description = "HTTPS from VPC"
  }

  tags = {
    Name = "${var.vpc_name}-vpce-sg"
  }
}

# S3 Gateway Endpoint (free, no NAT needed for S3)
resource "aws_vpc_endpoint" "s3" {
  vpc_id            = module.vpc.vpc_id
  service_name      = "com.amazonaws.${var.aws_region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = module.vpc.private_route_table_ids

  tags = {
    Name = "${var.vpc_name}-s3-endpoint"
  }
}

# ECR API Interface Endpoint
resource "aws_vpc_endpoint" "ecr_api" {
  vpc_id              = module.vpc.vpc_id
  service_name        = "com.amazonaws.${var.aws_region}.ecr.api"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.vpc_endpoints.id]
  private_dns_enabled = true

  tags = {
    Name = "${var.vpc_name}-ecr-api-endpoint"
  }
}

# ECR Docker Interface Endpoint
resource "aws_vpc_endpoint" "ecr_dkr" {
  vpc_id              = module.vpc.vpc_id
  service_name        = "com.amazonaws.${var.aws_region}.ecr.dkr"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.vpc_endpoints.id]
  private_dns_enabled = true

  tags = {
    Name = "${var.vpc_name}-ecr-dkr-endpoint"
  }
}

# STS Interface Endpoint (for IRSA)
resource "aws_vpc_endpoint" "sts" {
  vpc_id              = module.vpc.vpc_id
  service_name        = "com.amazonaws.${var.aws_region}.sts"
  vpc_endpoint_type   = "Interface"
  subnet_ids          = module.vpc.private_subnets
  security_group_ids  = [aws_security_group.vpc_endpoints.id]
  private_dns_enabled = true

  tags = {
    Name = "${var.vpc_name}-sts-endpoint"
  }
}

# ── Outputs ──────────────────────────────────────────────────────────

output "vpc_id" {
  description = "VPC ID"
  value       = module.vpc.vpc_id
}

output "vpc_cidr_block" {
  description = "VPC CIDR block"
  value       = module.vpc.vpc_cidr_block
}

output "private_subnet_ids" {
  description = "Private subnet IDs for EKS worker nodes"
  value       = module.vpc.private_subnets
}

output "public_subnet_ids" {
  description = "Public subnet IDs for ALB / NAT"
  value       = module.vpc.public_subnets
}

output "intra_subnet_ids" {
  description = "Intra subnet IDs for databases"
  value       = module.vpc.intra_subnets
}

output "nat_gateway_ips" {
  description = "Elastic IPs of NAT gateways"
  value       = module.vpc.nat_public_ips
}

output "vpc_endpoint_s3_id" {
  description = "S3 VPC gateway endpoint ID"
  value       = aws_vpc_endpoint.s3.id
}

output "azs" {
  description = "Availability zones used"
  value       = local.azs
}
