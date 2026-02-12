terraform {
  required_version = ">= 1.2"
  required_providers {
    proxmox = {
      source  = "bpg/proxmox"
      version = "~> 0.45.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.11.0"
    }
  }
}

provider "proxmox" {
  endpoint = var.proxmox_endpoint
  insecure = var.proxmox_insecure
  username = var.proxmox_username
  password = var.proxmox_password
}

provider "helm" {
  kubernetes {
    config_path = var.kubeconfig_path
  }
}

module "proxmox_infrastructure" {
  source = "./modules/proxmox"

  environment = var.environment
  cluster_name = var.cluster_name
  
  vm_count = var.proxmox_vm_count
  cpu_cores = var.proxmox_cpu_cores
  memory_mb = var.proxmox_memory_mb
}

module "ceph_storage" {
  source = "./modules/ceph"

  environment = var.environment
  proxmox_nodes = module.proxmox_infrastructure.node_ids
  osd_count = var.ceph_osd_count
}

module "networking" {
  source = "./modules/networking"

  environment = var.environment
  cluster_name = var.cluster_name
  vpc_cidr = var.vpc_cidr
}

resource "helm_release" "nats" {
  name       = "nats"
  repository = "https://nats-io.github.io/k8s/helm/charts/"
  chart      = "nats"
  namespace  = "kubric"

  values = [
    file("${path.module}/../../config/nats/nats-helm-values.yaml")
  ]
}

resource "helm_release" "temporal" {
  name       = "temporal"
  repository = "https://temporalio.github.io/helm-charts"
  chart      = "temporal"
  namespace  = "kubric"

  values = [
    file("${path.module}/../../config/temporal/temporal-helm-values.yaml")
  ]
}
