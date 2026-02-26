.PHONY: help build dev test clean deploy bootstrap lint check-gpl-boundary

help:
	@echo "Kubric Platform - Development Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make bootstrap      Bootstrap Kubernetes cluster"
	@echo "  make dev           Start docker-compose development environment"
	@echo "  make test          Run all tests"
	@echo "  make lint          Run linters and code quality checks"
	@echo "  make build         Build all components"
	@echo "  make deploy-staging Deploy to staging environment"
	@echo "  make deploy-prod   Deploy to production (requires manual approval)"
	@echo "  make clean         Clean build artifacts and caches"
	@echo "  make check-gpl-boundary  Verify no GPL-3.0 RITA imports in services/"
	@echo ""

.DEFAULT_GOAL := help

# Development

dev:
	docker-compose -f docker-compose/docker-compose.dev.yml up -d
	@echo "✅ Development environment started"
	@echo "Services:"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  NATS:       localhost:4222"
	@echo "  ClickHouse: localhost:8123"
	@echo "  Prometheus: localhost:9090"

dev-logs:
	docker-compose -f docker-compose/docker-compose.dev.yml logs -f

dev-down:
	docker-compose -f docker-compose/docker-compose.dev.yml down
	@echo "✅ Development environment stopped"

# Kubernetes

bootstrap:
	@echo "🚀 Bootstrapping Kubric Kubernetes cluster..."
	@chmod +x scripts/*.sh
	./scripts/bootstrap-cluster.sh

bootstrap-apply:
	kubectl apply -k deployments/k8s/

bootstrap-delete:
	kubectl delete -k deployments/k8s/ || true
	kubectl delete namespace kubric || true

# Database

db-init:
	@echo "🗄️  Initializing databases..."
	@chmod +x scripts/init-databases.sh
	./scripts/init-databases.sh

db-backup:
	@echo "💾 Backing up ClickHouse..."
	@chmod +x scripts/backup-clickhouse.sh
	./scripts/backup-clickhouse.sh

db-restore:
	@echo "♻️  Restore not yet implemented"

# Testing

test:
	@echo "🧪 Running tests..."
	cd agents/coresec && cargo test
	cd agents/netguard && cargo test
	pytest ./tests/ -v

test-unit:
	cd agents/coresec && cargo test --lib
	cd agents/netguard && cargo test --lib

test-integration:
	@echo "Running integration tests on docker-compose..."
	docker-compose -f docker-compose/docker-compose.dev.yml up -d
	pytest ./tests/integration/ -v --timeout=60
	docker-compose -f docker-compose/docker-compose.dev.yml down

test-coverage:
	cargo tarpaulin --out Html --output-dir coverage/

# Linting & Code Quality

lint:
	@echo "🔍 Running linters..."
	cargo fmt -- --check
	cargo clippy --all-targets -- -D warnings
	black --check .
	flake8 .
	yamllint -r .github/
	terraform fmt -check -recursive deployments/terraform/

lint-fix:
	cargo fmt
	black .
	autopep8 -i -r .
	yamllint -f parsable .github/ | head -20
	terraform fmt -recursive deployments/terraform/

security-scan:
	@echo "🔐 Running security scanners..."
	grype dir:. --fail-on high
	syft dir:. -o json > sbom.json
	@echo "✅ SBOM generated: sbom.json"

# GPL 3.0 boundary enforcement — RITA must never be imported as a Go package.
# Run this after every change to services/ to verify the boundary holds.
check-gpl-boundary:
	@echo "Checking GPL 3.0 boundary: scanning for activecm/rita imports in services/..."
	@grep -r '"github.com/activecm/rita' services/ 2>/dev/null && (echo 'GPL VIOLATION DETECTED — remove activecm/rita imports from services/' && exit 1) || echo 'GPL boundary clean'

# Building

build:
	@echo "🏗️  Building components..."
	cd agents/coresec && cargo build --release
	cd agents/netguard && cargo build --release

build-docker:
	docker build -t kubric/api:latest -f Dockerfile.api .
	docker build -t kubric/kai:latest -f Dockerfile.kai .
	docker build -t kubric/web:latest -f Dockerfile.web .

# Deployment

deploy-staging:
	@echo "📦 Deploying to staging..."
	kubectl apply -k deployments/k8s/ --kubeconfig=~/.kube/staging
	kubectl rollout status deployment/api -n kubric --kubeconfig=~/.kube/staging

deploy-prod:
	@echo "⚠️  PRODUCTION DEPLOYMENT - Requires approval"
	@echo "This will be triggered by GitHub Actions"
	@echo "Push to main branch to start deployment workflow"

# Infrastructure

tf-init:
	cd deployments/terraform && terraform init

tf-plan:
	cd deployments/terraform && terraform plan -out=tfplan

tf-apply:
	cd deployments/terraform && terraform apply tfplan

tf-destroy:
	cd deployments/terraform && terraform destroy

# Utilities

shell-k8s:
	kubectl exec -it -n kubric $(shell kubectl get pod -n kubric -l app=postgres -o jsonpath='{.items[0].metadata.name}') -- bash

logs-api:
	kubectl logs -n kubric -f deployment/api

logs-kai:
	kubectl logs -n kubric -f deployment/kai-orchestration

status:
	@echo "📊 Kubric Status"
	kubectl get all -n kubric
	kubectl get pvc -n kubric

# Cleanup

clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -rf build/ dist/ coverage/ .pytest_cache/
	find . -type d -name __pycache__ -exec rm -rf {} +
	find . -type d -name .terraform -exec rm -rf {} +
	cd agents/coresec && cargo clean
	cd agents/netguard && cargo clean
	@echo "✅ Cleanup complete"

clean-all: clean
	docker-compose -f docker-compose/docker-compose.dev.yml down -v || true
	kubectl delete namespace kubric || true
	@echo "✅ Full cleanup complete"

# Pre-commit

pre-commit-install:
	pre-commit install
	@echo "✅ Pre-commit hooks installed"

pre-commit-run:
	pre-commit run --all-files

# Version

version:
	@echo "Kubric Platform v1.0.0"
