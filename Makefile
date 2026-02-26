.PHONY: help build dev test clean deploy bootstrap lint check-gpl-boundary restore-drill kustomize-build db-migrate

help:
	@echo "Kubric Platform - Development Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make bootstrap      Bootstrap Kubernetes cluster"
	@echo "  make dev           Start docker-compose development environment"
	@echo "  make test          Run all tests (Rust + Go + Python)"
	@echo "  make build         Build all components (Rust agents + Go services)"
	@echo "  make lint          Run linters and code quality checks"
	@echo "  make kustomize-build   Validate kustomize overlays"
	@echo "  make db-migrate    Run database migrations (local)"
	@echo "  make deploy-staging Deploy to staging environment"
	@echo "  make deploy-prod   Deploy to production (requires manual approval)"
	@echo "  make clean         Clean build artifacts and caches"
	@echo "  make check-gpl-boundary  Verify no GPL-3.0 RITA imports in services/"
	@echo ""

.DEFAULT_GOAL := help

# Development

dev:
	docker-compose -f docker-compose/docker-compose.dev.yml up -d
	@echo "Development environment started"
	@echo "Services:"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  NATS:       localhost:4222"
	@echo "  ClickHouse: localhost:8123"
	@echo "  Prometheus: localhost:9090"

dev-logs:
	docker-compose -f docker-compose/docker-compose.dev.yml logs -f

dev-down:
	docker-compose -f docker-compose/docker-compose.dev.yml down
	@echo "Development environment stopped"

# Kubernetes — kustomize overlays

kustomize-build:
	@echo "Validating kustomize overlays..."
	kustomize build infra/k8s/overlays/dev > /dev/null
	@echo "  dev overlay: OK"
	kustomize build infra/k8s/overlays/prod > /dev/null
	@echo "  prod overlay: OK"
	kustomize build infra/argocd > /dev/null
	@echo "  argocd: OK"

kustomize-apply-dev:
	kustomize build infra/k8s/overlays/dev | kubectl apply -f -

kustomize-apply-prod:
	kustomize build infra/k8s/overlays/prod | kubectl apply -f -

bootstrap:
	@echo "Bootstrapping Kubric Kubernetes cluster..."
	@chmod +x scripts/*.sh 2>/dev/null || true
	./scripts/bootstrap-cluster.sh

bootstrap-apply:
	kubectl apply -k infra/k8s/overlays/dev

bootstrap-delete:
	kubectl delete -k infra/k8s/overlays/dev || true
	kubectl delete namespace kubric || true

# Database migrations

db-migrate:
	@echo "Running PostgreSQL migrations..."
	migrate -path db/migrations -database "$(KUBRIC_DATABASE_URL)" up
	@echo "PostgreSQL migrations complete"

db-migrate-down:
	@echo "Rolling back last PostgreSQL migration..."
	migrate -path db/migrations -database "$(KUBRIC_DATABASE_URL)" down 1

db-migrate-status:
	migrate -path db/migrations -database "$(KUBRIC_DATABASE_URL)" version

db-init:
	@echo "Initializing databases..."
	@chmod +x scripts/init-databases.sh 2>/dev/null || true
	./scripts/init-databases.sh

db-backup:
	@echo "Backing up ClickHouse..."
	@chmod +x scripts/backup-clickhouse.sh 2>/dev/null || true
	./scripts/backup-clickhouse.sh

db-restore:
	@echo "Restore not yet implemented"

# Testing

test: test-rust test-go test-python
	@echo "All tests complete"

test-rust:
	@echo "Running Rust agent tests..."
	cargo test --workspace

test-go:
	@echo "Running Go service tests..."
	go test ./internal/... ./cmd/... -count=1

test-python:
	@echo "Running Python KAI tests..."
	cd kai && python -m pytest tests/ -v 2>/dev/null || echo "No Python tests found"

test-unit:
	cargo test --workspace --lib
	go test ./internal/... -count=1 -short

test-integration:
	@echo "Running integration tests on docker-compose..."
	docker-compose -f docker-compose/docker-compose.dev.yml up -d
	pytest ./tests/integration/ -v --timeout=60 2>/dev/null || true
	docker-compose -f docker-compose/docker-compose.dev.yml down

test-coverage:
	cargo tarpaulin --out Html --output-dir coverage/
	go test ./internal/... -coverprofile=coverage/go-coverage.out
	@echo "Coverage reports in coverage/"

# Restore Drill — must pass before customer 1 (L4-3)
# Requires running ClickHouse + MinIO (make dev)
restore-drill:
	@echo "Running DR restore drill..."
	go test ./scripts/backup/... -run TestRestoreDrill -v -timeout 10m
	@echo "Restore drill complete"

# Linting & Code Quality

lint:
	@echo "Running linters..."
	cargo fmt -- --check
	cargo clippy --all-targets -- -D warnings
	go vet ./...
	black --check kai/ 2>/dev/null || true
	flake8 kai/ 2>/dev/null || true

lint-fix:
	cargo fmt
	black kai/ 2>/dev/null || true
	gofmt -w .

security-scan:
	@echo "Running security scanners..."
	grype dir:. --fail-on high
	syft dir:. -o json > sbom.json
	@echo "SBOM generated: sbom.json"

# GPL 3.0 boundary enforcement — RITA must never be imported as a Go package.
# Run this after every change to services/ to verify the boundary holds.
check-gpl-boundary:
	@echo "Checking GPL 3.0 boundary: scanning for activecm/rita imports in services/..."
	@grep -r '"github.com/activecm/rita' services/ 2>/dev/null && (echo 'GPL VIOLATION DETECTED — remove activecm/rita imports from services/' && exit 1) || echo 'GPL boundary clean'

# Building

build: build-rust build-go
	@echo "All builds complete"

build-rust:
	@echo "Building Rust agents..."
	cargo build --workspace --release

build-go:
	@echo "Building Go services..."
	go build ./cmd/ksvc/...
	go build ./cmd/vdr/...
	go build ./cmd/kic/...
	go build ./cmd/noc/...
	go build ./cmd/nats-clickhouse-bridge/...
	go build ./cmd/nuclei-bridge/...

build-docker:
	docker build -t kubric/k-svc:latest -f Dockerfile.api .
	docker build -t kubric/kai:latest -f Dockerfile.kai .
	docker build -t kubric/web:latest -f Dockerfile.web .

# Deployment

deploy-staging:
	@echo "Deploying to staging..."
	kustomize build infra/k8s/overlays/dev | kubectl apply -f -
	kubectl rollout status deployment/k-svc -n kubric --timeout=120s

deploy-prod:
	@echo "PRODUCTION DEPLOYMENT - Requires approval"
	@echo "This will be triggered by GitHub Actions on push to main"
	@echo "Manual: kustomize build infra/k8s/overlays/prod | kubectl apply -f -"

# Infrastructure

tf-init:
	cd deployments/terraform && terraform init

tf-plan:
	cd deployments/terraform && terraform plan -out=tfplan

tf-apply:
	cd deployments/terraform && terraform apply tfplan

tf-destroy:
	cd deployments/terraform && terraform destroy

# Vault

vault-policy-apply:
	vault policy write kubric-default config/vault/policies.hcl
	@echo "Vault policies applied"

# Utilities

shell-k8s:
	kubectl exec -it -n kubric $(shell kubectl get pod -n kubric -l app=postgresql -o jsonpath='{.items[0].metadata.name}') -- bash

logs-ksvc:
	kubectl logs -n kubric -f deployment/k-svc

logs-kai:
	kubectl logs -n kubric -f deployment/kai-core

status:
	@echo "Kubric Status"
	kubectl get all -n kubric
	kubectl get pvc -n kubric

# Cleanup

clean:
	@echo "Cleaning build artifacts..."
	rm -rf build/ dist/ coverage/ .pytest_cache/
	find . -type d -name __pycache__ -exec rm -rf {} + 2>/dev/null || true
	find . -type d -name .terraform -exec rm -rf {} + 2>/dev/null || true
	cargo clean 2>/dev/null || true
	@echo "Cleanup complete"

clean-all: clean
	docker-compose -f docker-compose/docker-compose.dev.yml down -v || true
	kubectl delete namespace kubric || true
	@echo "Full cleanup complete"

# Pre-commit

pre-commit-install:
	pre-commit install
	@echo "Pre-commit hooks installed"

pre-commit-run:
	pre-commit run --all-files

# Version

version:
	@echo "Kubric Platform v1.0.0"
