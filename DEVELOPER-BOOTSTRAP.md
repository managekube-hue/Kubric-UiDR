# Developer Bootstrap

This is the baseline provisioning guide for Kubric UIDR contributors.

## Scope

- Platform: Ubuntu/Codespaces baseline
- Purpose: install and verify required tooling without changing repository code paths
- Deployment note: Vercel builds from repository commands in `vercel.json`; it does **not** depend on local apt repository files.

## Required Toolchain

- Core: `git`, `make`, `curl`, `jq`
- Node/docs: `node >= 18`, `yarn 1.x`
- Python: `python3`, `pip3`
- Infra: `docker`, `kubectl`, `helm`, `terraform`, `ansible`
- Languages: `go`, `rustc`, `cargo`
- API/schema: `protoc`

## Install (Ubuntu)

```bash
sudo apt-get update -y
sudo apt-get install -y --no-install-recommends \
  ansible rustc cargo protobuf-compiler unzip

# Terraform (pin)
TF_VER="1.9.8"
ARCH="linux_amd64"
curl -fsSL -o /tmp/terraform.zip "https://releases.hashicorp.com/terraform/${TF_VER}/terraform_${TF_VER}_${ARCH}.zip"
unzip -o /tmp/terraform.zip -d /tmp
sudo install -m 0755 /tmp/terraform /usr/local/bin/terraform
```

## Verify

```bash
git --version
node --version
yarn --version
python3 --version
terraform --version | head -n 1
ansible --version | head -n 1
rustc --version
cargo --version
protoc --version
```

## Project Verification

```bash
cd docs
yarn install --frozen-lockfile
yarn build
```

## Vercel Dependency Clarification

Vercel uses repository-defined commands in `vercel.json`:

- `installCommand`: `cd docs && yarn install --frozen-lockfile`
- `buildCommand`: `cd docs && yarn install --frozen-lockfile && yarn build`

The local file `/etc/apt/sources.list.d/yarn.list.disabled` is a machine-level apt source toggle used during package provisioning and is **not** part of the repo, not pushed to Git, and not used by Vercel build containers.

## Operational Policy

- Keep one Git-connected Vercel project/domain for production.
- Keep DocuNotion sync committing docs content only (`docs/docs/**`).
- Do not trigger extra deploy paths from workflow scripts.
