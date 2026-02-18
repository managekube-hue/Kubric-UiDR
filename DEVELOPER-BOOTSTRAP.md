# Developer Bootstrap

This is the baseline provisioning guide for Kubric UIDR contributors.

## Scope

- Platform: Ubuntu/Codespaces baseline
- Purpose: install and verify required tooling without changing repository code paths

## Required Toolchain

- Core: `git`, `make`, `curl`, `jq`
- Node: `node >= 18`
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
python3 --version
terraform --version | head -n 1
ansible --version | head -n 1
rustc --version
cargo --version
protoc --version
```
