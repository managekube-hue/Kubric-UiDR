#!/usr/bin/env python3
"""
Script to create all missing files according to the project tree structure.
Comprehensive file structure for Kubric Platform.
"""

import os
from pathlib import Path

# Template content for different file types
TEMPLATES = {
    '.md': "# {title}\n\n## Overview\n\nDocumentation for {description}\n\n## Configuration\n\n(Content to be filled)\n\n## References\n\nSee related documentation in the project.\n",
    '.go': "package main\n\n// {title}\n// {description}\n",
    '.rs': "// {title}\n// {description}\n\nfn main() {{\n    println!(\"Kubric {}\");\n}}\n",
    '.sql': "-- {title}\n-- {description}\n\n-- SQL content placeholder\n",
    '.py': "# {title}\n# {description}\n\nif __name__ == \"__main__\":\n    pass\n",
    '.toml': "# {title}\n# {description}\n\n[package]\n",
    '.yaml': "# {title}\n# {description}\n\napiVersion: v1\nkind: Config\nmetadata:\n  name: kubric\n",
    '.json': \"{{\\n  \\\"title\\\": \\\"{title}\\\",\\n  \\\"description\\\": \\\"{description}\\\"\\n}}\\n\",
    '.yml': "# {title}\n# {description}\n\nversion: '3.8'\nservices: {{}}\n",
    '.tsx': \"// {title}\\n// {description}\\n\\nexport function Component() {{\\n  return <div>Component</div>;\\n}}\\n\",
    '.ts': \"// {title}\\n// {description}\\n\\nexport class Component {{}\\n\",
    '.css': \"/* {title}\\n   {description}\\n*/\\n\",
    '.sh': \"#!/bin/bash\\n# {title}\\n# {description}\\n\",
}

# Define the complete file structure - ALL SECTIONS
FILES_TO_CREATE = {
    # K-CORE-01_INFRASTRUCTURE
    "K-CORE-01_INFRASTRUCTURE/K-HW-R740_HARDWARE": [
        ("K-HW-001_Node1_Config.md", "# Dell R740 Node1 Configuration\n\n## Overview\nNode 1 configuration for Kubric platform\n\n## Hardware Specs\n- Processor: 2x Intel Xeon\n- RAM: 256GB\n- Storage: NVMe SSD\n\n## Network Configuration\n- Interface 1: 10GbE\n- Interface 2: 1GbE mgmt\n\n## See Also\n- K-HW-002_Node2_Config.md\n- K-HW-003_Node3_Config.md\n"),
        ("K-HW-002_Node2_Config.md", "# Dell R740 Node2 Configuration\n\n## Overview\nNode 2 configuration for Kubric platform\n\n## Hardware Specs\n- Processor: 2x Intel Xeon\n- RAM: 256GB\n- Storage: NVMe SSD\n\n## Network Configuration\n- Interface 1: 10GbE\n- Interface 2: 1GbE mgmt\n\n## See Also\n- K-HW-001_Node1_Config.md\n- K-HW-003_Node3_Config.md\n"),
        ("K-HW-003_Node3_Config.md", "# Dell R740 Node3 Configuration\n\n## Overview\nNode 3 configuration for Kubric platform\n\n## Hardware Specs\n- Processor: 2x Intel Xeon\n- RAM: 256GB\n- Storage: NVMe SSD\n\n## Network Configuration\n- Interface 1: 10GbE\n- Interface 2: 1GbE mgmt\n\n## See Also\n- K-HW-001_Node1_Config.md\n- K-HW-002_Node2_Config.md\n"),
        ("K-HW-004_iDRAC9_Network.md", "# iDRAC9 Network Configuration\n\n## Overview\nDell iDRAC9 remote access configuration\n\n## Network Setup\n- Static IP: 192.168.1.x/24\n- Gateway: 192.168.1.1\n- DNS: 8.8.8.8\n\n## Security\n- IPMI v2 enabled with authentication\n- SSH access configured\n- SSL/TLS encryption\n\n## See Also\n- K-HW-001_Node1_Config.md\n- K-HW-002_Node2_Config.md\n- K-HW-003_Node3_Config.md\n"),
        ("K-HW-005_RAM_Expansion.md", "# RAM Expansion Planning\n\n## Current Configuration\n- 3x Dell R740 nodes\n- 256GB RAM per node\n- Total: 768GB\n\n## Expansion Path\n- Target: 512GB per node\n- Total capacity: 1.5TB\n- DIMM type: 32GB RDIMM\n\n## Timeline\nQ2-Q3 2024 expansion\n"),
    ],
    "K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING": [
        ("K-NET-001_10G_SFP_Config.md", "# 10G SFP+ Network Configuration\n\n## Overview\nDirect-attached copper and fiber SFP+ configuration\n\n## Physical Layer\n- 4x 10GBase-SR fiber optic\n- 4x 10GBase-Cu copper DAC\n- MTU: 9000 bytes (jumbo frames)\n\n## IP Configuration\n- Subnet: 10.0.0.0/24\n- Node1: 10.0.0.1\n- Node2: 10.0.0.2\n- Node3: 10.0.0.3\n\n## Performance\n- Line rate: 10Gbps aggregate\n"),
        ("K-NET-002_Corosync_Heartbeat.md", "# Corosync Heartbeat Configuration\n\n## Overview\nCluster heartbeat and membership service\n\n## Corosync Config\n```yaml\ncluster_name: kubric_cluster\ntoken: 3000\ntoken_retransmits_before_loss_const: 10\njoin: 50\nexpect_votes: 3\nactual_votes: 3\n```\n\n## Heartbeat Interval\n- 1000ms heartbeat\n- Failure detection: 3 seconds\n\n## Network\n- Dedicated heartbeat VLAN: 100\n"),
        ("K-NET-003_Virtual_IP_Failover.md", "# Virtual IP Failover Configuration\n\n## Overview\nFloating VIP for HA services\n\n## VIP Configuration\n- Primary VIP: 10.1.1.100\n- Secondary VIP: 10.1.1.101\n- Netmask: 255.255.255.0\n\n## Failover Mechanism\n- Uses keepalived\n- Priority: Node1(100) > Node2(50) > Node3(10)\n- Health check interval: 2 seconds\n\n## Services\n- Kubernetes API\n- n8n\n- Caddy\n"),
    ],
    "K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR": [
        ("K-HV-001_Cluster_Bootstrap.md", "# Proxmox Cluster Bootstrap\n\n## Overview\nInitial Proxmox VE cluster setup and configuration\n\n## Cluster Setup\n- 3 nodes\n- Quorum: 2 of 3\n- Cluster network: 10.0.0.0/24\n\n## Initialization Steps\n1. Install Proxmox VE 8.x\n2. Configure networking\n3. Join cluster\n4. Configure corosync\n\n## SSL Certificates\n- Generated during cluster creation\n- Valid for 365 days\n"),
        ("K-HV-002_Ceph_Storage.md", "# Ceph Storage Configuration\n\n## Overview\nObject-based storage for VM backing\n\n## Pool Configuration\n- vm-images: 3x replication\n- containers: 2x replication\n- data: 1x replication\n\n## PG Count\n- vm-images: 256 PGs\n- containers: 128 PGs\n- data: 64 PGs\n\n## Capacity\n- Total: 9TB usable (3x3TB SATA SSDs)\n"),
    ],
    "K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES": [
        ("K-HV-VM-001_Go_API_CloudInit.md", "# Go API CloudInit Template\n\n## Overview\nGo-based API server CloudInit template\n\n## Base Image\n- Debian 12\n- 4 vCPU\n- 8GB RAM\n- 40GB disk\n\n## CloudInit Configuration\n- Install Go 1.21\n- Install dependencies\n- Configure systemd service\n\n## See Also\n- K-HV-VM-002_ClickHouse_StatefulSet.md\n"),
        ("K-HV-VM-002_ClickHouse_StatefulSet.md", "# ClickHouse StatefulSet Template\n\n## Overview\nClickHouse OLAP database VM template\n\n## Configuration\n- Base: Ubuntu 22.04\n- vCPU: 8\n- RAM: 32GB\n- Storage: 500GB SSD + 1TB HDD archive\n\n## ClickHouse Setup\n- Distributed cluster mode\n- Replication enabled\n- TTL cold storage to HDD\n"),
        ("K-HV-VM-003_PostgreSQL_RLS.md", "# PostgreSQL RLS Template\n\n## Overview\nPostgreSQL with Row-Level Security\n\n## VM Spec\n- Base: Ubuntu 22.04\n- vCPU: 4\n- RAM: 16GB\n- Storage: 200GB SSD (ext4)\n\n## RLS Policies\n- Tenant isolation\n- Role-based access\n- Field-level encryption\n"),
        ("K-HV-VM-004_Ollama_LLM.md", "# Ollama LLM Template\n\n## Overview\nOllama local LLM inference VM\n\n## Hardware\n- vCPU: 16\n- RAM: 64GB\n- GPU: NVIDIA A100 passthrough\n- Storage: 500GB NVMe\n\n## Models\n- Llama 2 70B\n- Mistral 7B\n- Neural chat\n"),
    ],
    "K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS": [
        ("K-HV-LXC-001_Gitea.md", "# Gitea LXC Container\n\n## Overview\nGit service over HTTP(S)\n\n## Container Spec\n- Base: Debian 12\n- vCPU: 2\n- RAM: 4GB\n- Storage: 100GB\n\n## Services\n- Gitea web UI\n- SSH git access\n- Webhook support\n"),
        ("K-HV-LXC-002_n8n.md", "# n8n Workflow Automation LXC\n\n## Overview\nn8n workflow orchestration engine\n\n## Container Spec\n- Base: Ubuntu 22.04\n- vCPU: 4\n- RAM: 8GB\n- Storage: 100GB\n\n## Configuration\n- PostgreSQL backend\n- Webhook endpoints\n- Credential vault\n"),
        ("K-HV-LXC-003_Caddy.md", "# Caddy Reverse Proxy LXC\n\n## Overview\nAutomatic HTTPS reverse proxy\n\n## Container Spec\n- Base: Alpine Linux\n- vCPU: 2\n- RAM: 2GB\n- Storage: 20GB\n\n## Features\n- Let's Encrypt auto\n- Load balancing\n- API gateway\n"),
    ],
    "K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES": [
        ("K-K8S-001_ClickHouse_StatefulSet.yaml", "apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  name: clickhouse\n  namespace: kubric\nspec:\n  serviceName: clickhouse\n  replicas: 3\n  selector:\n    matchLabels:\n      app: clickhouse\n  template:\n    metadata:\n      labels:\n        app: clickhouse\n    spec:\n      containers:\n      - name: clickhouse\n        image: clickhouse/clickhouse-server:latest\n        ports:\n        - containerPort: 9000\n          name: native\n        - containerPort: 8123\n          name: http\n        env:\n        - name: CLICKHOUSE_DEFAULT_ACCESS_MANAGEMENT\n          value: '1'\n"),
        ("K-K8S-002_NATS_StatefulSet.yaml", "apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  name: nats\n  namespace: kubric\nspec:\n  serviceName: nats\n  replicas: 3\n  selector:\n    matchLabels:\n      app: nats\n  template:\n    metadata:\n      labels:\n        app: nats\n    spec:\n      containers:\n      - name: nats\n        image: nats:latest\n        ports:\n        - containerPort: 4222\n          name: client\n        - containerPort: 6222\n          name: cluster\n        - containerPort: 8222\n          name: monitor\n"),
        ("K-K8S-003_PostgreSQL_StatefulSet.yaml", "apiVersion: apps/v1\nkind: StatefulSet\nmetadata:\n  name: postgres\n  namespace: kubric\nspec:\n  serviceName: postgres\n  replicas: 3\n  selector:\n    matchLabels:\n      app: postgres\n  template:\n    metadata:\n      labels:\n        app: postgres\n    spec:\n      containers:\n      - name: postgres\n        image: postgres:15-alpine\n        ports:\n        - containerPort: 5432\n        env:\n        - name: POSTGRES_DB\n          value: kubric\n        - name: POSTGRES_PASSWORD\n          valueFrom:\n            secretKeyRef:\n              name: postgres-secret\n              key: password\n"),
        ("K-K8S-004_API_Deployment.yaml", "apiVersion: apps/v1\nkind: Deployment\nmetadata:\n  name: api\n  namespace: kubric\nspec:\n  replicas: 3\n  selector:\n    matchLabels:\n      app: api\n  template:\n    metadata:\n      labels:\n        app: api\n    spec:\n      containers:\n      - name: api\n        image: kubric/api:latest\n        ports:\n        - containerPort: 8080\n        env:\n        - name: LOG_LEVEL\n          value: 'info'\n        - name: DATABASE_URL\n          valueFrom:\n            secretKeyRef:\n              name: api-secret\n              key: database_url\n"),
    ],
    "K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE": [
        ("K-DL-CH-001_Cluster_Config.md", "# ClickHouse Cluster Configuration\n\n## Overview\nClickHouse distributed cluster setup\n\n## Cluster Topology\n- 3 shards\n- 2 replicas per shard\n- Quorum replication\n\n## Configuration\n```xml\n<remote_servers>\n  <kubric>\n    <shard>\n      <replica><host>ch1</host><port>9000</port></replica>\n      <replica><host>ch2</host><port>9000</port></replica>\n    </shard>\n  </kubric>\n</remote_servers>\n```\n"),
        ("K-DL-CH-002_OCSF_Schema.sql", "-- OCSF Event Schema for ClickHouse\n\nCREATE TABLE IF NOT EXISTS ocsf_events (\n    metadata_event_code String,\n    metadata_product String,\n    metadata_version String,\n    metadata_profiles Array(String),\n    event_time DateTime,\n    severity_id UInt8,\n    status_id UInt8,\n    event_data String,\n    raw_event String\n) ENGINE = MergeTree()\nORDER BY (event_time, severity_id)\nPARTITION BY toYYYYMM(event_time);\n"),
        ("K-DL-CH-003_TTL_Cold_Storage.md", "# TTL and Cold Storage Configuration\n\n## Overview\nAutomatic data tiering strategy\n\n## TTL Rules\n- Hot: 30 days (NVMe SSD)\n- Warm: 90 days (SATA SSD)\n- Cold: 365 days (HDD archive)\n- Delete: 2 years\n\n## Implementation\n```sql\nALTER TABLE ocsf_events MODIFY TTL event_time + INTERVAL 30 DAY TO DISK 'ssd';\n```\n"),
        ("K-DL-CH-004_Agent_Decision_History.sql", "-- Agent Decision History Table\n\nCREATE TABLE IF NOT EXISTS agent_decisions (\n    decision_id UUID,\n    agent_name String,\n    action_type String,\n    timestamp DateTime,\n    input_context String,\n    decision_output String,\n    confidence_score Float32,\n    verification_status String,\n    approval_status String\n) ENGINE = MergeTree()\nORDER BY (timestamp, agent_name)\nPARTITION BY toYYYYMM(timestamp);\n"),
    ],
    "K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES": [
        ("K-DL-PG-001_UAR_Asset_Table.sql", "-- User Asset Registry (UAR)\n\nCREATE TABLE IF NOT EXISTS user_assets (\n    asset_id UUID PRIMARY KEY,\n    asset_name VARCHAR(255) NOT NULL,\n    asset_type VARCHAR(100),\n    owner_id UUID,\n    criticality_level INT,\n    deployment_env VARCHAR(50),\n    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,\n    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP\n);\n\nCREATE INDEX idx_owner_id ON user_assets(owner_id);\nCREATE INDEX idx_asset_type ON user_assets(asset_type);\n"),
        ("K-DL-PG-002_RLS_Policies.sql", "-- Row-Level Security Policies\n\nALTER TABLE user_assets ENABLE ROW LEVEL SECURITY;\n\nCREATE POLICY user_asset_access ON user_assets\n    FOR SELECT\n    USING (owner_id = current_user_id());\n\nCREATE POLICY user_asset_update ON user_assets\n    FOR UPDATE\n    USING (owner_id = current_user_id())\n    WITH CHECK (owner_id = current_user_id());\n"),
        ("K-DL-PG-003_Contract_Rate_Tables.sql", "-- Billing Contract and Rate Tables\n\nCREATE TABLE IF NOT EXISTS contracts (\n    contract_id UUID PRIMARY KEY,\n    customer_id UUID,\n    contract_type VARCHAR(50),\n    start_date DATE,\n    end_date DATE,\n    status VARCHAR(20)\n);\n\nCREATE TABLE IF NOT EXISTS rate_cards (\n    rate_id UUID PRIMARY KEY,\n    contract_id UUID REFERENCES contracts(contract_id),\n    service_type VARCHAR(100),\n    unit_price DECIMAL(10, 4),\n    effective_date DATE\n);\n"),
        ("K-DL-PG-004_OSCAL_Ingestion.sql", "-- OSCAL Compliance Data Ingestion\n\nCREATE TABLE IF NOT EXISTS oscal_controls (\n    control_id VARCHAR(50) PRIMARY KEY,\n    control_name VARCHAR(255),\n    control_class VARCHAR(20),\n    framework VARCHAR(50),\n    family VARCHAR(100),\n    description TEXT,\n    implementation_status VARCHAR(50)\n);\n\nCREATE INDEX idx_framework ON oscal_controls(framework);\nCREATE INDEX idx_status ON oscal_controls(implementation_status);\n"),
    ],
    "K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS": [
        ("K-MB-001_NATS_Cluster.yaml", "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: nats-config\n  namespace: kubric\ndata:\n  nats.conf: |\n    port: 4222\n    http_port: 8222\n    \n    cluster {\n      port: 6222\n      routes [\n        nats://nats-0.nats:6222\n        nats://nats-1.nats:6222\n        nats://nats-2.nats:6222\n      ]\n    }\n    \n    jetstream {\n      store_dir: /data/jetstream\n      max_memory: 10GB\n    }\n"),
        ("K-MB-002_JetStream_Config.md", "# NATS JetStream Configuration\n\n## Overview\nEvent streaming and replay capability\n\n## Streams\n- security.alerts\n- remediation.tasks\n- asset.provisioning\n- billing.metering\n\n## Retention Policy\n- Max age: 30 days\n- Max msgs: 1,000,000\n- Max bytes: 100GB\n\n## Consumer Groups\n- SOC processing\n- Billing aggregation\n- Asset inventory\n"),
        ("K-MB-003_mTLS_Cert_Rotation.md", "# mTLS Certificate Rotation\n\n## Overview\nAutomated TLS certificate management\n\n## Rotation Schedule\n- Frequency: Every 90 days\n- Grace period: 30 days before expiry\n- Automated renewal via cert-manager\n\n## Implementation\n- OpenSSL for CSR generation\n- HashiCorp Vault for CA\n- kubectl rollout for pod restart\n"),
    ],
    "K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING": [
        ("K-MB-SUB-001_security.alert.v1", "# Security Alert Subject Schema v1\n\nmetadata:\n  version: 1\n  namespace: security.alert.v1\n  \npayload:\n  alert_id: string\n  severity: enum:[critical, high, medium, low]\n  event_type: string\n  detection_timestamp: timestamp\n  source: string\n  target: string\n  rule_id: string\n  indicators: array[string]\n"),
        ("K-MB-SUB-002_remediation.task.v1", "# Remediation Task Subject Schema v1\n\nmetadata:\n  version: 1\n  namespace: remediation.task.v1\n\npayload:\n  task_id: string\n  alert_id: string\n  remediation_type: string\n  target_asset: string\n  priority: enum:[p0, p1, p2, p3]\n  automation_level: enum:[manual, semi-auto, full-auto]\n  status: enum:[pending, executing, completed, failed]\n"),
        ("K-MB-SUB-003_asset.provisioned.v1", "# Asset Provisioned Subject Schema v1\n\nmetadata:\n  version: 1\n  namespace: asset.provisioned.v1\n\npayload:\n  asset_id: string\n  asset_type: string\n  deployment_env: string\n  provisioning_timestamp: timestamp\n  configuration: object\n  deployment_status: enum:[success, failed]\n"),
        ("K-MB-SUB-004_billing.meter.v1", "# Billing Meter Subject Schema v1\n\nmetadata:\n  version: 1\n  namespace: billing.meter.v1\n\npayload:\n  meter_id: string\n  customer_id: string\n  service_type: string\n  quantity: number\n  unit: string\n  timestamp: timestamp\n  meter_value: number\n"),
    ],
    "K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT": [
        ("K-SEC-001_HashiCorp_Vault.md", "# HashiCorp Vault Configuration\n\n## Overview\nSecrets management and encryption\n\n## Setup\n- HA cluster with 3 nodes\n- Integrated storage (Raft)\n- Auto unsealing with GCP KMS\n\n## Secrets Engines\n- KV v2: Application secrets\n- PKI: Certificate generation\n- SSH: SSH key management\n- Transit: Encryption as a service\n\n## Access Control\n- OIDC/OAuth2 auth method\n- Role-based policies\n- Audit logging enabled\n"),
        ("K-SEC-002_TPM_Root_of_Trust.md", "# TPM Root of Trust Configuration\n\n## Overview\nTrusted Platform Module for hardware security\n\n## Implementation\n- TPM 2.0 on all R740 nodes\n- TPM-backed key storage\n- Measured boot (UEFI Secure Boot)\n\n## Use Cases\n- VM image validation\n- Host attestation\n- Cluster bootstrap keys\n\n## Tools\n- tpm2-tools for crypto ops\n- tboot for measured launch\n"),
        ("K-SEC-003_Blake3_Fingerprint.go", "package security\n\nimport (\n\t\"crypto/blake3\"\n\t\"encoding/hex\"\n\t\"io\"\n)\n\n// Blake3Fingerprint generates a BLAKE3 hash of data\nfunc Blake3Fingerprint(data []byte) string {\n\th := blake3.Sum256(data)\n\treturn hex.EncodeToString(h[:])\n}\n\n// Blake3FingerprintStream handles streaming large files\nfunc Blake3FingerprintStream(r io.Reader) (string, error) {\n\th := blake3.New()\n\tif _, err := io.Copy(h, r); err != nil {\n\t\treturn \"\", err\n\t}\n\treturn hex.EncodeToString(h.Sum(nil)), nil\n}\n"),
        ("K-SEC-004_CA_Setup.md", "# Certificate Authority Setup\n\n## Overview\nInternal PKI infrastructure\n\n## Hierarchy\n- Root CA (offline, air-gapped)\n- Intermediate CA (online)\n- Leaf certificates for services\n\n## Storage\n- Root key: HSM-backed\n- Intermediate: HashiCorp Vault\n- CRL: S3 bucket with CDN\n\n## Validity Periods\n- Root CA: 10 years\n- Intermediate: 5 years\n- Leaf: 1 year (auto-renewal at 30 days)\n"),
    ],
}

def create_file_structure():
    """Create all files according to the defined structure."""
    base_path = Path("/workspaces/Kubric-UiDR")
    
    for dir_path, files in FILES_TO_CREATE.items():
        dir_full_path = base_path / dir_path
        
        # Create directory if it doesn't exist
        dir_full_path.mkdir(parents=True, exist_ok=True)
        
        for filename, content in files:
            file_path = dir_full_path / filename
            
            # Skip if file already exists
            if file_path.exists():
                print(f"SKIP: {file_path} (already exists)")
                continue
                
            try:
                file_path.write_text(content, encoding='utf-8')
                print(f"CREATE: {file_path}")
            except Exception as e:
                print(f"ERROR: {file_path} - {e}")

if __name__ == "__main__":
    create_file_structure()
    print("\nFile creation complete!")
