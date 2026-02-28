# ─────────────────────────────────────────────────────────────────────
# Terraform – EKS Managed Node Groups for Kubric UiDR (Large)
# ─────────────────────────────────────────────────────────────────────
# Companion to K-DEPLOY-LG-001 (EKS cluster).
# Defines detailed launch templates, auto-scaling policies and
# security groups for the three workload tiers:
#
#   1. general   – 10× m5.2xlarge  – Kubric services, NATS, API
#   2. inference  – 5× g5.2xlarge  – vLLM, GPU model serving
#   3. storage   – 3× r5.2xlarge  – ClickHouse, MinIO, PostgreSQL
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

variable "cluster_name" {
  description = "EKS cluster name (must match K-DEPLOY-LG-001)"
  type        = string
  default     = "kubric-prod"
}

variable "cluster_version" {
  description = "Kubernetes version"
  type        = string
  default     = "1.29"
}

variable "vpc_id" {
  description = "VPC ID from K-DEPLOY-LG-002"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs from K-DEPLOY-LG-002"
  type        = list(string)
}

variable "cluster_security_group_id" {
  description = "Cluster security group from K-DEPLOY-LG-001"
  type        = string
}

variable "cluster_endpoint" {
  description = "EKS API endpoint URL"
  type        = string
}

variable "cluster_ca_data" {
  description = "Base64-encoded cluster CA certificate"
  type        = string
  sensitive   = true
}

variable "environment" {
  description = "Deployment environment tag"
  type        = string
  default     = "production"
}

variable "kubric_agent_version" {
  description = "Kubric XRO agent version to bootstrap on nodes"
  type        = string
  default     = "2.4.1"
}

# ── Data Sources ─────────────────────────────────────────────────────

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

data "aws_ami" "eks_gpu" {
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

# ── Security Groups ─────────────────────────────────────────────────

# Shared node security group for all managed node groups.
resource "aws_security_group" "kubric_nodes" {
  name_prefix = "${var.cluster_name}-nodes-"
  description = "Security group for Kubric EKS worker nodes"
  vpc_id      = var.vpc_id

  # Intra-node communication (full mesh)
  ingress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
    self      = true
    description = "Node-to-node all traffic"
  }

  # Cluster → node (kubelet, kube-proxy)
  ingress {
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = [var.cluster_security_group_id]
    description     = "Cluster API to kubelet HTTPS"
  }

  ingress {
    from_port       = 1025
    to_port         = 65535
    protocol        = "tcp"
    security_groups = [var.cluster_security_group_id]
    description     = "Cluster to node high ports"
  }

  # NATS (client + cluster + leafnode)
  ingress {
    from_port = 4222
    to_port   = 4222
    protocol  = "tcp"
    self      = true
    description = "NATS client"
  }

  ingress {
    from_port = 6222
    to_port   = 6222
    protocol  = "tcp"
    self      = true
    description = "NATS cluster"
  }

  ingress {
    from_port = 7422
    to_port   = 7422
    protocol  = "tcp"
    self      = true
    description = "NATS leafnode"
  }

  # ClickHouse native + HTTP
  ingress {
    from_port = 9000
    to_port   = 9000
    protocol  = "tcp"
    self      = true
    description = "ClickHouse native"
  }

  ingress {
    from_port = 8123
    to_port   = 8123
    protocol  = "tcp"
    self      = true
    description = "ClickHouse HTTP"
  }

  # PostgreSQL
  ingress {
    from_port = 5432
    to_port   = 5432
    protocol  = "tcp"
    self      = true
    description = "PostgreSQL"
  }

  # MinIO
  ingress {
    from_port = 9000
    to_port   = 9001
    protocol  = "tcp"
    self      = true
    description = "MinIO API + Console"
  }

  # Egress – allow all (restricted by NACLs if needed)
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
    description = "All outbound"
  }

  tags = {
    Name        = "${var.cluster_name}-nodes-sg"
    Project     = "kubric-uidr"
    Environment = var.environment
  }

  lifecycle {
    create_before_destroy = true
  }
}

# ── User Data (Kubric agent bootstrap) ───────────────────────────────

locals {
  # Common user data that runs on every node to:
  # 1. Set EKS bootstrap args
  # 2. Install the Kubric Watchdog agent for bare-metal telemetry
  general_user_data = <<-USERDATA
    #!/bin/bash
    set -euo pipefail

    # EKS bootstrap (kubelet args)
    /etc/eks/bootstrap.sh '${var.cluster_name}' \
      --kubelet-extra-args '--node-labels=kubric.io/node-role=general --max-pods=110'

    # Install Kubric Watchdog agent
    curl -fsSL https://artifacts.kubric.io/agents/watchdog/${var.kubric_agent_version}/install.sh | \
      KUBRIC_CLUSTER=${var.cluster_name} \
      KUBRIC_NODE_ROLE=general \
      bash -s -- --version ${var.kubric_agent_version}

    # Harden node
    sysctl -w net.core.somaxconn=65535
    sysctl -w net.ipv4.tcp_max_syn_backlog=65535
    sysctl -w vm.max_map_count=262144
    echo 'never' > /sys/kernel/mm/transparent_hugepage/enabled
  USERDATA

  inference_user_data = <<-USERDATA
    #!/bin/bash
    set -euo pipefail

    /etc/eks/bootstrap.sh '${var.cluster_name}' \
      --kubelet-extra-args '--node-labels=kubric.io/node-role=inference,nvidia.com/gpu=true --register-with-taints=nvidia.com/gpu=true:NoSchedule --max-pods=58'

    # Install NVIDIA container toolkit
    distribution=$(. /etc/os-release; echo $ID$VERSION_ID)
    curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
    curl -s -L "https://nvidia.github.io/libnvidia-container/$distribution/libnvidia-container.list" | \
      sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
      tee /etc/apt/sources.list.d/nvidia-container-toolkit.list > /dev/null
    apt-get update && apt-get install -y nvidia-container-toolkit
    nvidia-ctk runtime configure --runtime=containerd
    systemctl restart containerd

    # Kubric Watchdog
    curl -fsSL https://artifacts.kubric.io/agents/watchdog/${var.kubric_agent_version}/install.sh | \
      KUBRIC_CLUSTER=${var.cluster_name} \
      KUBRIC_NODE_ROLE=inference \
      bash -s -- --version ${var.kubric_agent_version}
  USERDATA

  storage_user_data = <<-USERDATA
    #!/bin/bash
    set -euo pipefail

    /etc/eks/bootstrap.sh '${var.cluster_name}' \
      --kubelet-extra-args '--node-labels=kubric.io/node-role=storage --register-with-taints=kubric.io/storage=true:NoSchedule --max-pods=58'

    # Tune kernel for data workloads
    sysctl -w vm.max_map_count=262144
    sysctl -w vm.dirty_ratio=40
    sysctl -w vm.dirty_background_ratio=10
    sysctl -w net.core.rmem_max=16777216
    sysctl -w net.core.wmem_max=16777216
    echo 'never' > /sys/kernel/mm/transparent_hugepage/enabled
    echo 'never' > /sys/kernel/mm/transparent_hugepage/defrag

    # Kubric Watchdog
    curl -fsSL https://artifacts.kubric.io/agents/watchdog/${var.kubric_agent_version}/install.sh | \
      KUBRIC_CLUSTER=${var.cluster_name} \
      KUBRIC_NODE_ROLE=storage \
      bash -s -- --version ${var.kubric_agent_version}
  USERDATA
}

# ── Launch Templates ─────────────────────────────────────────────────

resource "aws_launch_template" "general" {
  name_prefix   = "${var.cluster_name}-general-"
  description   = "Launch template for Kubric general-purpose nodes"
  image_id      = data.aws_ami.eks_optimized.id
  instance_type = "m5.2xlarge"

  vpc_security_group_ids = [aws_security_group.kubric_nodes.id]

  user_data = base64encode(local.general_user_data)

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size           = 100
      volume_type           = "gp3"
      iops                  = 3000
      throughput            = 125
      encrypted             = true
      delete_on_termination = true
    }
  }

  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required" # IMDSv2
    http_put_response_hop_limit = 2
  }

  monitoring {
    enabled = true
  }

  tag_specifications {
    resource_type = "instance"
    tags = {
      Name        = "${var.cluster_name}-general"
      Project     = "kubric-uidr"
      Environment = var.environment
      NodeGroup   = "general"
    }
  }

  tag_specifications {
    resource_type = "volume"
    tags = {
      Name        = "${var.cluster_name}-general-vol"
      Project     = "kubric-uidr"
      Environment = var.environment
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_launch_template" "inference" {
  name_prefix   = "${var.cluster_name}-inference-"
  description   = "Launch template for Kubric GPU inference nodes"
  image_id      = data.aws_ami.eks_gpu.id
  instance_type = "g5.2xlarge"

  vpc_security_group_ids = [aws_security_group.kubric_nodes.id]

  user_data = base64encode(local.inference_user_data)

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size           = 200
      volume_type           = "gp3"
      iops                  = 6000
      throughput            = 250
      encrypted             = true
      delete_on_termination = true
    }
  }

  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 2
  }

  monitoring {
    enabled = true
  }

  tag_specifications {
    resource_type = "instance"
    tags = {
      Name        = "${var.cluster_name}-inference"
      Project     = "kubric-uidr"
      Environment = var.environment
      NodeGroup   = "inference"
    }
  }

  tag_specifications {
    resource_type = "volume"
    tags = {
      Name        = "${var.cluster_name}-inference-vol"
      Project     = "kubric-uidr"
      Environment = var.environment
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_launch_template" "storage" {
  name_prefix   = "${var.cluster_name}-storage-"
  description   = "Launch template for Kubric storage-optimised nodes"
  image_id      = data.aws_ami.eks_optimized.id
  instance_type = "r5.2xlarge"

  vpc_security_group_ids = [aws_security_group.kubric_nodes.id]

  user_data = base64encode(local.storage_user_data)

  # OS volume
  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size           = 100
      volume_type           = "gp3"
      iops                  = 3000
      throughput            = 125
      encrypted             = true
      delete_on_termination = true
    }
  }

  # Data volume for ClickHouse / MinIO
  block_device_mappings {
    device_name = "/dev/xvdb"
    ebs {
      volume_size           = 500
      volume_type           = "gp3"
      iops                  = 10000
      throughput            = 500
      encrypted             = true
      delete_on_termination = false # Preserve data on termination
    }
  }

  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 2
  }

  monitoring {
    enabled = true
  }

  tag_specifications {
    resource_type = "instance"
    tags = {
      Name        = "${var.cluster_name}-storage"
      Project     = "kubric-uidr"
      Environment = var.environment
      NodeGroup   = "storage"
    }
  }

  tag_specifications {
    resource_type = "volume"
    tags = {
      Name        = "${var.cluster_name}-storage-vol"
      Project     = "kubric-uidr"
      Environment = var.environment
    }
  }

  lifecycle {
    create_before_destroy = true
  }
}

# ── EKS Managed Node Groups ─────────────────────────────────────────

resource "aws_eks_node_group" "general" {
  cluster_name    = var.cluster_name
  node_group_name = "${var.cluster_name}-general"
  node_role_arn   = aws_iam_role.node_group.arn
  subnet_ids      = var.private_subnet_ids

  launch_template {
    id      = aws_launch_template.general.id
    version = aws_launch_template.general.latest_version
  }

  scaling_config {
    desired_size = 10
    min_size     = 3
    max_size     = 15
  }

  update_config {
    max_unavailable_percentage = 25
  }

  labels = {
    "kubric.io/node-role" = "general"
    "kubric.io/workload"  = "services"
  }

  tags = {
    Name        = "${var.cluster_name}-general"
    Project     = "kubric-uidr"
    Environment = var.environment
    AutoScaling = "enabled"
  }

  lifecycle {
    ignore_changes = [scaling_config[0].desired_size]
  }

  depends_on = [
    aws_iam_role_policy_attachment.node_AmazonEKSWorkerNodePolicy,
    aws_iam_role_policy_attachment.node_AmazonEKS_CNI_Policy,
    aws_iam_role_policy_attachment.node_AmazonEC2ContainerRegistryReadOnly,
    aws_iam_role_policy_attachment.node_AmazonSSMManagedInstanceCore,
  ]
}

resource "aws_eks_node_group" "inference" {
  cluster_name    = var.cluster_name
  node_group_name = "${var.cluster_name}-inference"
  node_role_arn   = aws_iam_role.node_group.arn
  subnet_ids      = var.private_subnet_ids

  launch_template {
    id      = aws_launch_template.inference.id
    version = aws_launch_template.inference.latest_version
  }

  scaling_config {
    desired_size = 5
    min_size     = 2
    max_size     = 10
  }

  update_config {
    max_unavailable = 1
  }

  labels = {
    "kubric.io/node-role" = "inference"
    "kubric.io/workload"  = "gpu"
    "nvidia.com/gpu"      = "true"
  }

  taint {
    key    = "nvidia.com/gpu"
    value  = "true"
    effect = "NO_SCHEDULE"
  }

  tags = {
    Name        = "${var.cluster_name}-inference"
    Project     = "kubric-uidr"
    Environment = var.environment
    AutoScaling = "enabled"
  }

  lifecycle {
    ignore_changes = [scaling_config[0].desired_size]
  }

  depends_on = [
    aws_iam_role_policy_attachment.node_AmazonEKSWorkerNodePolicy,
    aws_iam_role_policy_attachment.node_AmazonEKS_CNI_Policy,
    aws_iam_role_policy_attachment.node_AmazonEC2ContainerRegistryReadOnly,
    aws_iam_role_policy_attachment.node_AmazonSSMManagedInstanceCore,
  ]
}

resource "aws_eks_node_group" "storage" {
  cluster_name    = var.cluster_name
  node_group_name = "${var.cluster_name}-storage"
  node_role_arn   = aws_iam_role.node_group.arn
  subnet_ids      = var.private_subnet_ids

  launch_template {
    id      = aws_launch_template.storage.id
    version = aws_launch_template.storage.latest_version
  }

  scaling_config {
    desired_size = 3
    min_size     = 3
    max_size     = 6
  }

  update_config {
    max_unavailable = 1
  }

  labels = {
    "kubric.io/node-role" = "storage"
    "kubric.io/workload"  = "data"
  }

  taint {
    key    = "kubric.io/storage"
    value  = "true"
    effect = "NO_SCHEDULE"
  }

  tags = {
    Name        = "${var.cluster_name}-storage"
    Project     = "kubric-uidr"
    Environment = var.environment
    AutoScaling = "enabled"
  }

  lifecycle {
    ignore_changes = [scaling_config[0].desired_size]
  }

  depends_on = [
    aws_iam_role_policy_attachment.node_AmazonEKSWorkerNodePolicy,
    aws_iam_role_policy_attachment.node_AmazonEKS_CNI_Policy,
    aws_iam_role_policy_attachment.node_AmazonEC2ContainerRegistryReadOnly,
    aws_iam_role_policy_attachment.node_AmazonSSMManagedInstanceCore,
  ]
}

# ── Node Group IAM Role ─────────────────────────────────────────────

resource "aws_iam_role" "node_group" {
  name = "${var.cluster_name}-node-group"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Project     = "kubric-uidr"
    Environment = var.environment
  }
}

resource "aws_iam_role_policy_attachment" "node_AmazonEKSWorkerNodePolicy" {
  policy_arn = "arn:aws:iam::policy/AmazonEKSWorkerNodePolicy"
  role       = aws_iam_role.node_group.name
}

resource "aws_iam_role_policy_attachment" "node_AmazonEKS_CNI_Policy" {
  policy_arn = "arn:aws:iam::policy/AmazonEKS_CNI_Policy"
  role       = aws_iam_role.node_group.name
}

resource "aws_iam_role_policy_attachment" "node_AmazonEC2ContainerRegistryReadOnly" {
  policy_arn = "arn:aws:iam::policy/AmazonEC2ContainerRegistryReadOnly"
  role       = aws_iam_role.node_group.name
}

resource "aws_iam_role_policy_attachment" "node_AmazonSSMManagedInstanceCore" {
  policy_arn = "arn:aws:iam::policy/AmazonSSMManagedInstanceCore"
  role       = aws_iam_role.node_group.name
}

# ── Cluster Autoscaler IAM (for ASG scaling) ────────────────────────

resource "aws_iam_policy" "cluster_autoscaler" {
  name        = "${var.cluster_name}-cluster-autoscaler"
  description = "Cluster Autoscaler permissions for EKS node groups"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "autoscaling:DescribeAutoScalingGroups",
          "autoscaling:DescribeAutoScalingInstances",
          "autoscaling:DescribeLaunchConfigurations",
          "autoscaling:DescribeScalingActivities",
          "autoscaling:DescribeTags",
          "autoscaling:SetDesiredCapacity",
          "autoscaling:TerminateInstanceInAutoScalingGroup",
          "ec2:DescribeImages",
          "ec2:DescribeInstanceTypes",
          "ec2:DescribeLaunchTemplateVersions",
          "ec2:GetInstanceTypesFromInstanceRequirements",
          "eks:DescribeNodegroup",
        ]
        Resource = "*"
      }
    ]
  })
}

# ── Outputs ──────────────────────────────────────────────────────────

output "general_node_group_name" {
  description = "Name of the general-purpose node group"
  value       = aws_eks_node_group.general.node_group_name
}

output "inference_node_group_name" {
  description = "Name of the GPU inference node group"
  value       = aws_eks_node_group.inference.node_group_name
}

output "storage_node_group_name" {
  description = "Name of the storage-optimised node group"
  value       = aws_eks_node_group.storage.node_group_name
}

output "node_security_group_id" {
  description = "Security group ID for all node groups"
  value       = aws_security_group.kubric_nodes.id
}

output "node_role_arn" {
  description = "IAM role ARN for node groups"
  value       = aws_iam_role.node_group.arn
}
