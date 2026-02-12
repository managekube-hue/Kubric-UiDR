#!/bin/bash
set -e

echo "🚀 Bootstrapping Kubric Kubernetes Cluster..."

# Check prerequisites
echo "✓ Checking prerequisites..."
command -v kubectl >/dev/null 2>&1 || { echo "kubectl not found"; exit 1; }
command -v helm >/dev/null 2>&1 || { echo "helm not found"; exit 1; }

# Create namespace
echo "✓ Creating kubric namespace..."
kubectl create namespace kubric --dry-run=client -o yaml | kubectl apply -f -

# Add Helm repos
echo "✓ Adding Helm repositories..."
helm repo add jetstack https://charts.jetstack.io
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo add temporal https://temporalio.github.io/helm-charts
helm repo update

# Install cert-manager
echo "✓ Installing cert-manager..."
helm upgrade --install cert-manager jetstack/cert-manager \
  --namespace cert-manager \
  --create-namespace \
  --set installCRDs=true \
  --wait

# Deploy Kubric stack via Kustomize
echo "✓ Deploying Kubric stack..."
kubectl apply -k deployments/k8s/

# Wait for statefulsets to be ready
echo "✓ Waiting for databases..."
kubectl wait --for=condition=ready pod \
  -l app=postgres \
  -n kubric \
  --timeout=300s

kubectl wait --for=condition=ready pod \
  -l app=nats \
  -n kubric \
  --timeout=300s

# Initialize databases
echo "✓ Initializing databases..."
./scripts/init-databases.sh

echo "✅ Cluster bootstrap complete!"
echo ""
echo "Next steps:"
echo "  1. Deploy agents: ./scripts/deploy-agents.sh"
echo "  2. Check status: kubectl get all -n kubric"
echo "  3. View logs: kubectl logs -n kubric -f deployment/api"
