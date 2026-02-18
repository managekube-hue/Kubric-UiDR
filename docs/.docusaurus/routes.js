import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/compliance',
    component: ComponentCreator('/compliance', 'f48'),
    exact: true
  },
  {
    path: '/contact',
    component: ComponentCreator('/contact', 'b83'),
    exact: true
  },
  {
    path: '/contributors',
    component: ComponentCreator('/contributors', '9cd'),
    exact: true
  },
  {
    path: '/modules',
    component: ComponentCreator('/modules', 'ea4'),
    exact: true
  },
  {
    path: '/open-source',
    component: ComponentCreator('/open-source', 'b96'),
    exact: true
  },
  {
    path: '/platform',
    component: ComponentCreator('/platform', '5da'),
    exact: true
  },
  {
    path: '/platform/grc',
    component: ComponentCreator('/platform/grc', 'ee3'),
    exact: true
  },
  {
    path: '/platform/kai',
    component: ComponentCreator('/platform/kai', 'a9b'),
    exact: true
  },
  {
    path: '/platform/noc',
    component: ComponentCreator('/platform/noc', '68e'),
    exact: true
  },
  {
    path: '/platform/psa',
    component: ComponentCreator('/platform/psa', 'eed'),
    exact: true
  },
  {
    path: '/platform/soc',
    component: ComponentCreator('/platform/soc', 'b0c'),
    exact: true
  },
  {
    path: '/search',
    component: ComponentCreator('/search', '5de'),
    exact: true
  },
  {
    path: '/docs',
    component: ComponentCreator('/docs', 'f95'),
    routes: [
      {
        path: '/docs',
        component: ComponentCreator('/docs', 'f56'),
        routes: [
          {
            path: '/docs',
            component: ComponentCreator('/docs', '881'),
            routes: [
              {
                path: '/docs/intro',
                component: ComponentCreator('/docs/intro', '677'),
                exact: true
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-NATS-001_subject_hierarchy',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-NATS-001_subject_hierarchy', '969'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-001_provisioning.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-001_provisioning.yaml', '9c2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-002_triage.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-002_triage.yaml', '424'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-003_billing.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-003_billing.yaml', '425'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-004_vdr_scan.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-004_vdr_scan.yaml', '78d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-005_grc_compliance.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-005_grc_compliance.yaml', '86e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-006_identity_graph.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-006_identity_graph.yaml', 'cc1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-007_ndr_flow.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-007_ndr_flow.yaml', '392'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-008_health.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-008_health.yaml', 'ed3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-009_alerts.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-009_alerts.yaml', '55a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-001_ocsf_schema.proto',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-001_ocsf_schema.proto', '472'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-002_build_rs.rs',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-002_build_rs.rs', '8dd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-003_ocsf_deploy.proto',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-003_ocsf_deploy.proto', '541'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-001_Cluster_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-001_Cluster_Config', 'a98'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-002_OCSF_Schema.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-002_OCSF_Schema.sql', '3df'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-003_TTL_Cold_Storage',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-003_TTL_Cold_Storage', '418'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-004_Agent_Decision_History.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-004_Agent_Decision_History.sql', 'f4b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-005_Arrow_Bulk_Insert',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-005_Arrow_Bulk_Insert', '682'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-DUCKDB/K-DL-DUCK-001_embedded_analytics',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-DUCKDB/K-DL-DUCK-001_embedded_analytics', '4e7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-DUCKDB/K-DL-DUCK-002_ml_feature_compute.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-DUCKDB/K-DL-DUCK-002_ml_feature_compute.sql', '441'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-001_golang_migrate_setup',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-001_golang_migrate_setup', 'b41'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-002_liquibase_k8s.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-002_liquibase_k8s.yaml', 'c14'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-003_atlas_ci_sync',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-003_atlas_ci_sync', 'd14'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-001_UAR_Asset_Table.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-001_UAR_Asset_Table.sql', '5cc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-002_RLS_Policies.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-002_RLS_Policies.sql', '48d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-003_Contract_Rate_Tables.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-003_Contract_Rate_Tables.sql', 'b1e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-004_OSCAL_Ingestion.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-004_OSCAL_Ingestion.sql', '291'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-005_Atlas_Schema_HCL.hcl',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-005_Atlas_Schema_HCL.hcl', 'bd9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-001_Cluster_Bootstrap',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-001_Cluster_Bootstrap', '920'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-002_Ceph_Storage',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-002_Ceph_Storage', '657'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-003_Chrony_PTP_TimeSync',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-003_Chrony_PTP_TimeSync', '842'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-001_Gitea',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-001_Gitea', '67b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-002_n8n',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-002_n8n', '6cc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-003_Caddy',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-003_Caddy', '2a8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-004_Woodpecker_CI',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-004_Woodpecker_CI', 'e10'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-001_Go_API_CloudInit',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-001_Go_API_CloudInit', 'c27'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-002_ClickHouse_StatefulSet',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-002_ClickHouse_StatefulSet', '7be'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-003_PostgreSQL_RLS',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-003_PostgreSQL_RLS', '27f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-004_Ollama_LLM',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-004_Ollama_LLM', 'd32'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-005_vLLM_GPU_Node',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-005_vLLM_GPU_Node', 'b29'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-001_Node1_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-001_Node1_Config', 'a49'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-002_Node2_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-002_Node2_Config', 'bdd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-003_Node3_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-003_Node3_Config', 'f06'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-004_iDRAC9_Network',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-004_iDRAC9_Network', '504'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-005_RAM_Expansion',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-005_RAM_Expansion', 'a03'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-001_ClickHouse_StatefulSet.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-001_ClickHouse_StatefulSet.yaml', '0ef'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-002_NATS_StatefulSet.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-002_NATS_StatefulSet.yaml', 'f1b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-003_PostgreSQL_StatefulSet.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-003_PostgreSQL_StatefulSet.yaml', 'c14'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-004_API_Deployment.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-004_API_Deployment.yaml', 'a36'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-001_ArgoCD_Application.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-001_ArgoCD_Application.yaml', '908'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-002_Flux_GitRepository.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-002_Flux_GitRepository.yaml', 'e4e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-003_Helm_Values.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-003_Helm_Values.yaml', '8da'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-004_Kustomize_Overlays.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-004_Kustomize_Overlays.yaml', 'ecc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-005_Gatekeeper_OPA.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-005_Gatekeeper_OPA.yaml', '065'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-001_Istio_ServiceMesh.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-001_Istio_ServiceMesh.yaml', '23d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-002_Linkerd_Config.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-002_Linkerd_Config.yaml', 'fd9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-003_Cilium_CNI.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-003_Cilium_CNI.yaml', '7f3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-004_Hubble_ServiceMap.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-004_Hubble_ServiceMap.yaml', 'dcb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-005_CertManager_Vault.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-005_CertManager_Vault.yaml', 'd53'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-006_ExternalSecrets_Operator.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-006_ExternalSecrets_Operator.yaml', '006'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-001_Prometheus_Operator.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-001_Prometheus_Operator.yaml', 'a77'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-002_Thanos_Sidecar.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-002_Thanos_Sidecar.yaml', '830'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-003_Grafana_Deployment.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-003_Grafana_Deployment.yaml', 'a56'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-004_Loki_Stack.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-004_Loki_Stack.yaml', '767'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-005_Tempo_Tracing.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-005_Tempo_Tracing.yaml', '9cf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-001_SealedSecrets_Controller.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-001_SealedSecrets_Controller.yaml', '300'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-002_NetworkPolicy_Defaults.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-002_NetworkPolicy_Defaults.yaml', 'ee2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-003_ResourceQuota.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-003_ResourceQuota.yaml', 'c98'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-001_NATS_Cluster.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-001_NATS_Cluster.yaml', 'e59'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-002_JetStream_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-002_JetStream_Config', '062'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-003_mTLS_Cert_Rotation',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-003_mTLS_Cert_Rotation', '30f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-004_ZeroMQ_IPC',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-004_ZeroMQ_IPC', '7a1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-001_edr.process.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-001_edr.process.v1', '53e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-002_edr.file.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-002_edr.file.v1', '461'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-003_ndr.flow.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-003_ndr.flow.v1', 'ed2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-004_ndr.beacon.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-004_ndr.beacon.v1', 'ea1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-005_itdr.auth.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-005_itdr.auth.v1', '4f0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-006_vdr.vuln.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-006_vdr.vuln.v1', '536'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-007_grc.drift.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-007_grc.drift.v1', '9b5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-008_svc.ticket.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-008_svc.ticket.v1', '3a8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-009_billing.usage.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-009_billing.usage.v1', 'edb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-010_health.score.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-010_health.score.v1', '1d6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-011_ti.ioc.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-011_ti.ioc.v1', '6c7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-012_comm.alert.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-012_comm.alert.v1', '688'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-013_security.alert.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-013_security.alert.v1', '3cf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-014_remediation.task.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-014_remediation.task.v1', '567'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-015_asset.provisioned.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-015_asset.provisioned.v1', 'ce3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-001_10G_SFP_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-001_10G_SFP_Config', '5c9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-002_Corosync_Heartbeat',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-002_Corosync_Heartbeat', '2cc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-003_Virtual_IP_Failover',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-003_Virtual_IP_Failover', 'fee'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-004_HAProxy_Keepalived',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-004_HAProxy_Keepalived', '22d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-001_HashiCorp_Vault',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-001_HashiCorp_Vault', '1b9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-002_TPM_Root_of_Trust',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-002_TPM_Root_of_Trust', '809'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-003_Blake3_Fingerprint.go',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-003_Blake3_Fingerprint.go', '8c3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-004_CA_Setup',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-004_CA_Setup', '6b7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-005_Vault_Policies.hcl',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-005_Vault_Policies.hcl', '1d7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-006_Vault_K8s_Auth.go',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-006_Vault_K8s_Auth.go', '363'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-000_INDEX',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-000_INDEX', 'b99'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-001_grafana_overview.json',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-001_grafana_overview.json', '42f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-002_prometheus_rules.yaml',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-002_prometheus_rules.yaml', 'c2e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-003_tuf_root.json',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-003_tuf_root.json', 'a0b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-001_terraform_aws_eks.tf',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-001_terraform_aws_eks.tf', '013'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-002_vpc_config.tf',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-002_vpc_config.tf', '1cf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-003_node_groups.tf',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-003_node_groups.tf', '088'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-MEDIUM/K-DEPLOY-MD-001_kustomize_overlay.yaml',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-MEDIUM/K-DEPLOY-MD-001_kustomize_overlay.yaml', 'd60'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-MEDIUM/K-DEPLOY-MD-002_scale_config.yaml',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-MEDIUM/K-DEPLOY-MD-002_scale_config.yaml', 'a68'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-SMALL/K-DEPLOY-SM-001_docker-compose.yml',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-SMALL/K-DEPLOY-SM-001_docker-compose.yml', '2f0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-SMALL/K-DEPLOY-SM-002_nats_single.conf',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-SMALL/K-DEPLOY-SM-002_nats_single.conf', 'fa9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-001_Makefile',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-001_Makefile', '2be'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-002_rust-toolchain.toml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-002_rust-toolchain.toml', '965'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-003_go.mod',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-003_go.mod', 'c8c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-004_package.json',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-004_package.json', '785'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-005_cobra_cli.go',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-005_cobra_cli.go', '421'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-006_chi_cors_middleware.go',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-006_chi_cors_middleware.go', 'df2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-007_chi_jwt_auth.go',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-007_chi_jwt_auth.go', '3bf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-008_workspace_cargo.toml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-008_workspace_cargo.toml', '9ea'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-009_buf_protobuf.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-009_buf_protobuf.yaml', '4be'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-010_requirements_kai_core.txt',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-010_requirements_kai_core.txt', '590'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-011_requirements_kai_full.txt',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-011_requirements_kai_full.txt', '5c7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-001_self_hosted_runner',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-001_self_hosted_runner', 'f56'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-002_woodpecker_pipeline.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-002_woodpecker_pipeline.yml', '6e1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-003_jenkins_x_config.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-003_jenkins_x_config.yaml', 'df4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-001_build-agents.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-001_build-agents.yml', 'b3b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-002_test-api.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-002_test-api.yml', 'f3b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-003_deploy-k8s.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-003_deploy-k8s.yml', '51b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-004_drone_config.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-004_drone_config.yml', '5d8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-005_tekton_pipeline.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-005_tekton_pipeline.yaml', '52a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-006_concourse_pipeline.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-006_concourse_pipeline.yml', 'de7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-007_dagger_ci.go',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-007_dagger_ci.go', 'f8e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-008_earthly_Earthfile',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-008_earthly_Earthfile', 'b7d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-009_cosign_signing.sh',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-009_cosign_signing.sh', '685'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-010_snyk_scan.sh',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-010_snyk_scan.sh', 'f72'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-011_sonarqube_scanner.sh',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-011_sonarqube_scanner.sh', 'd0c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-001_architecture',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-001_architecture', '677'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-002_README',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-002_README', '353'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-003_LICENSE',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-003_LICENSE', 'd1d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-004_NOTICE',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-004_NOTICE', 'a14'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-005_license_compliance_matrix',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-005_license_compliance_matrix', '28c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-001_gitea_setup',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-001_gitea_setup', 'd1f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-002_pre-commit.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-002_pre-commit.yaml', '714'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-003_molt-scanner.sh',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-003_molt-scanner.sh', '7ea'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-004_branch_protection',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-004_branch_protection', 'e31'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-005_ruff_config.toml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-005_ruff_config.toml', '163'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-006_golangci_lint.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-006_golangci_lint.yml', '00d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-007_clippy_config.toml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-007_clippy_config.toml', '84e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-008_eslint_config.js',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-008_eslint_config.js', 'ff5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-009_pre_commit_config.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-009_pre_commit_config.yaml', '9ad'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-010_commitlint_config.js',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-010_commitlint_config.js', '1ba'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-011_semantic_release.json',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-011_semantic_release.json', '20f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-012_black_config.toml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-012_black_config.toml', '57b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-013_isort_config.cfg',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-013_isort_config.cfg', 'a98'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-014_mypy_config.ini',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-014_mypy_config.ini', '667'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-015_pylintrc',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-015_pylintrc', 'ff5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-016_bandit_config.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-016_bandit_config.yaml', 'f85'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-017_safety_policy.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-017_safety_policy.yml', '9e5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-001_docker-compose.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-001_docker-compose.yml', 'e45'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-002_docker-compose-small.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-002_docker-compose-small.yml', '235'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-001_clickhouse_users.xml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-001_clickhouse_users.xml', '469'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-002_postgres_init.sql',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-002_postgres_init.sql', '6c7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-003_nats_cluster.conf',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-003_nats_cluster.conf', '171'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-004_vault_dev.hcl',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-004_vault_dev.hcl', 'df8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-001_k6_load_test.js',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-001_k6_load_test.js', 'a3f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-002_vegeta_attack.sh',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-002_vegeta_attack.sh', 'c6a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-003_kube_burner_config.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-003_kube_burner_config.yaml', '706'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-004_chaos_mesh_experiment.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-004_chaos_mesh_experiment.yaml', '630'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-005_litmus_chaos_engine.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-005_litmus_chaos_engine.yaml', '7ec'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-006_pytest_xdist_config.ini',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-006_pytest_xdist_config.ini', '32f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-007_factory_boy_factories.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-007_factory_boy_factories.py', '61b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-008_faker_data_gen.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-008_faker_data_gen.py', '485'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-009_hypothesis_property_tests.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-009_hypothesis_property_tests.py', 'f04'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-001_click_cli.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-001_click_cli.py', '0de'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-002_typer_cli.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-002_typer_cli.py', '6bc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-003_rich_output.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-003_rich_output.py', '20d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-004_tqdm_progress.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-004_tqdm_progress.py', '545'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-005_colorama_windows.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-005_colorama_windows.py', 'c9c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-006_tabulate_tables.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-006_tabulate_tables.py', '657'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-007_jsonpath_extractor.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-007_jsonpath_extractor.py', 'aa8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-008_jmespath_querier.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-008_jmespath_querier.py', 'd2d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-001_lula_validator.go',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-001_lula_validator.go', 'a6d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-002_openscap_binding.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-002_openscap_binding.py', '205'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-003_kyverno_policy.go',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-003_kyverno_policy.go', 'e22'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-001_immutable_audit.sql',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-001_immutable_audit.sql', 'a98'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-002_blake3_signer.go',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-002_blake3_signer.go', '645'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-003_legal_hold',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-003_legal_hold', 'd8b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-004_evidence_export',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-004_evidence_export', 'd50'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-001_nist_800_53_oscal',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-001_nist_800_53_oscal', '09a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-002_pci_dss_oscal',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-002_pci_dss_oscal', '6d6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-003_iso_27001_oscal',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-003_iso_27001_oscal', '8a4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-004_soc2_oscal',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-004_soc2_oscal', '707'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-001_nist_ingest.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-001_nist_ingest.py', '8c0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-002_soc2_mapper.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-002_soc2_mapper.py', '3c3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-003_iso_mapping.sql',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-003_iso_mapping.sql', '9bd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-004_compliance_trestle.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-004_compliance_trestle.py', '26e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-005_regscale_ingest.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-005_regscale_ingest.py', 'bc1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-001_sbom_syft.go',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-001_sbom_syft.go', 'c5f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-002_grype_scanner.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-002_grype_scanner.py', '3f8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-003_openssf_scorecard',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-003_openssf_scorecard', '2a6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-004_sbom_generation',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-004_sbom_generation', 'eda'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-005_sigstore_cosign.sh',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-005_sigstore_cosign.sh', 'a9d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-006_osv_api_check.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-006_osv_api_check.py', 'faf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-007_dependency_track.go',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-007_dependency_track.go', 'ef8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-008_cyclonedx_sbom.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-008_cyclonedx_sbom.py', 'ee1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-001_KIC_evidence_map',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-001_KIC_evidence_map', '262'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-002_soc2_iso_crosswalk.cs',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-002_soc2_iso_crosswalk.cs', '2d3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-001_GMP1_Strategy',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-001_GMP1_Strategy', '68c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-002_GMP5_Risk',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-002_GMP5_Risk', 'd41'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-003_GMP6_InfoSec',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-003_GMP6_InfoSec', 'dec'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-004_SMP1_Incident',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-004_SMP1_Incident', 'be1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-005_SMP10_Change',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-005_SMP10_Change', 'c0d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-006_SMP12_Deployment',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-006_SMP12_Deployment', '3ca'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-007_TMP2_Infrastructure',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-007_TMP2_Infrastructure', 'e94'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-008_SMP3_Problem',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-008_SMP3_Problem', '673'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-009_SMP7_ServiceLevel',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-009_SMP7_ServiceLevel', '52d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-001_fastapi_server.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-001_fastapi_server.py', 'fb0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-002_asyncpg_client.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-002_asyncpg_client.py', '362'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-003_psycopg2_fallback.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-003_psycopg2_fallback.py', 'b8b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-004_clickhouse_connect.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-004_clickhouse_connect.py', '3c1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-005_aiokafka_consumer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-005_aiokafka_consumer.py', '55a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-006_nats_py_client.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-006_nats_py_client.py', '815'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-007_anyio_backend.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-007_anyio_backend.py', 'e1f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-008_asgiref_sync.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-008_asgiref_sync.py', 'cc4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-009_socketio_realtime.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-009_socketio_realtime.py', '6c3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-AUDIT/K-KAI-AUD-001_decision_history.sql',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-AUDIT/K-KAI-AUD-001_decision_history.sql', '9f5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-AUDIT/K-KAI-AUD-002_merkle_signer.go',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-AUDIT/K-KAI-AUD-002_merkle_signer.go', '9a6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-001_cortex_analyzer_chain.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-001_cortex_analyzer_chain.py', '12b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-002_observable_enrichment.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-002_observable_enrichment.py', '576'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-001_billing_clerk.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-001_billing_clerk.py', '0c9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-002_clickhouse_audit.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-002_clickhouse_audit.py', 'fd8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-003_hle_calculator.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-003_hle_calculator.py', 'd9f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-004_invoice_renderer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-004_invoice_renderer.py', '521'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-001_comm_agent.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-001_comm_agent.py', 'b12'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-002_vapi_phone.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-002_vapi_phone.py', '605'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-003_twilio_sms.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-003_twilio_sms.py', '669'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-001_deploy_agent.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-001_deploy_agent.py', 'e3b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-002_saltstack_client.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-002_saltstack_client.py', '3cb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-003_fleet_rollout.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-003_fleet_rollout.py', 'ce5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-001_lstm_baseline.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-001_lstm_baseline.py', '652'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-002_epss_enrichment.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-002_epss_enrichment.py', 'e02'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-003_hikari_trainer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-003_hikari_trainer.py', 'b40'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-001_housekeeper.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-001_housekeeper.py', 'f63'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-002_ansible_runner.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-002_ansible_runner.py', 'de2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-003_criticality_check.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-003_criticality_check.py', 'cd6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-004_rollback.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-004_rollback.py', '2dd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/K-KAI-HU-001_velociraptor_artifacts.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/K-KAI-HU-001_velociraptor_artifacts.py', '742'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/K-KAI-HU-002_sigma_hunting_runner.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/K-KAI-HU-002_sigma_hunting_runner.py', '68d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-001_misp_galaxy_query.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-001_misp_galaxy_query.py', '5cc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-002_graph_investigation.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-002_graph_investigation.py', '4ea'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-001_remediation_planner.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-001_remediation_planner.py', '0a0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-002_cortex_subprocess.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-002_cortex_subprocess.py', '523'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-003_vault_secret_fetcher.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-003_vault_secret_fetcher.py', '16a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-001_pyfair_model.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-001_pyfair_model.py', '6da'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-002_epss_scorer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-002_epss_scorer.py', '6d2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-003_ssvc_decision.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-003_ssvc_decision.py', 'a67'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-001_health_score_publisher.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-001_health_score_publisher.py', 'b8b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-002_churn_risk_model.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-002_churn_risk_model.py', '0c2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-003_hibp_credential_score.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-003_hibp_credential_score.py', '203'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-001_ltv_predictor.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-001_ltv_predictor.py', 'f39'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-002_churn_simulator.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-002_churn_simulator.py', 'd34'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-003_dynamic_pricing.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-003_dynamic_pricing.py', '309'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-001_triage_agent.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-001_triage_agent.py', 'c99'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-002_llama3_reasoning.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-002_llama3_reasoning.py', '797'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-003_ocsf_analyzer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-003_ocsf_analyzer.py', '387'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-004_kiss_calculator.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-004_kiss_calculator.py', '56e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-001_human_mfa.go',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-001_human_mfa.go', 'a84'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-002_action_queue.sql',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-002_action_queue.sql', '42a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-003_criticality_5.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-003_criticality_5.py', 'b09'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-004_prompt_injection.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-004_prompt_injection.py', '326'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-001_polars_dataframe.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-001_polars_dataframe.py', '91a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-002_pyarrow_parquet.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-002_pyarrow_parquet.py', 'ca7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-003_fastparquet_io.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-003_fastparquet_io.py', 'a9c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-004_orjson_serializer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-004_orjson_serializer.py', 'd8a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-005_ujson_fallback.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-005_ujson_fallback.py', 'b59'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-006_msgpack_encoder.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-006_msgpack_encoder.py', '31e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-007_dpkt_pcap_parser.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-007_dpkt_pcap_parser.py', 'ff8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-008_scapy_probe.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-008_scapy_probe.py', '4a6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-009_pcap_capture.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-009_pcap_capture.py', 'ed4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-010_geoip2_resolver.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-010_geoip2_resolver.py', 'd5d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-001_tensorboard_logger.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-001_tensorboard_logger.py', 'e00'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-002_clearml_experiment.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-002_clearml_experiment.py', 'b1e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-003_pyspark_distributed.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-003_pyspark_distributed.py', 'fba'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-004_openai_fallback.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-004_openai_fallback.py', '4d5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-005_anthropic_long_context.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-005_anthropic_long_context.py', '00f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-006_cohere_embeddings.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-006_cohere_embeddings.py', 'b62'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-007_hikari_preprocessor.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-007_hikari_preprocessor.py', '481'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-008_ember_xgboost.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-008_ember_xgboost.py', '931'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-009_unswnb15_random_forest.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-009_unswnb15_random_forest.py', 'b90'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-010_mordor_lstm_baseline.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-010_mordor_lstm_baseline.py', '27f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-011_vllm_server.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-011_vllm_server.py', '8e2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-012_model_tiering',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-012_model_tiering', '939'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-001_vector_search.sql',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-001_vector_search.sql', '18b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-002_oscal_embeddings.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-002_oscal_embeddings.py', '5f0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-003_ciso_assistant.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-003_ciso_assistant.py', '920'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-004_cohere_embeddings.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-004_cohere_embeddings.py', '81a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-001_security_triage.json',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-001_security_triage.json', 'db3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-002_drift_housekeeper.json',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-002_drift_housekeeper.json', '001'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-003_heartbeat_billing.json',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-003_heartbeat_billing.json', 'e66'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-001_patch_workflow.go',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-001_patch_workflow.go', '011'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-002_retry_state.go',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-002_retry_state.go', '192'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-003_celery_tasks.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-003_celery_tasks.py', '50c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-004_flower_monitor.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-004_flower_monitor.py', '611'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-005_dramatiq_worker.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-005_dramatiq_worker.py', 'ee4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-006_huey_scheduler.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-006_huey_scheduler.py', '261'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-000_MASTER_INDEX',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-000_MASTER_INDEX', '7d0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-001_EDR_Endpoint',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-001_EDR_Endpoint', 'bc9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-002_ITDR_Identity',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-002_ITDR_Identity', '4dd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-003_NDR_Network',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-003_NDR_Network', '4f5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-004_CDR_Cloud',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-004_CDR_Cloud', '27f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-005_SDR_SaaS',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-005_SDR_SaaS', 'cd0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-006_ADR_Application',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-006_ADR_Application', 'c74'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-007_DDR_Data',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-007_DDR_Data', '72c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-008_VDR_Vulnerability',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-008_VDR_Vulnerability', 'fff'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-009_MDR_Managed',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-009_MDR_Managed', 'bc7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-010_TI_ThreatIntel',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-010_TI_ThreatIntel', '98c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-011_CFDR_ConfigDrift',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-011_CFDR_ConfigDrift', 'a5e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-012_BDR_Backup',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-012_BDR_Backup', '5da'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-013_NPM_NetworkPerf',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-013_NPM_NetworkPerf', 'cb7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-014_UEM_EndpointMgmt',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-014_UEM_EndpointMgmt', '2c5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-015_MDM_Mobile',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-015_MDM_Mobile', '0fe'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-016_APM_AppPerf',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-016_APM_AppPerf', '692'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-017_GRC_Governance',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-017_GRC_Governance', 'd6d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-018_KAI_AILayer',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-018_KAI_AILayer', '632'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-019_PSA_Business',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-019_PSA_Business', 'e1b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-020_LICENSE_COMPLIANCE',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-020_LICENSE_COMPLIANCE', 'faf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-001_restic_scheduler.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-001_restic_scheduler.go', '7d1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-002_kopia_snapshots.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-002_kopia_snapshots.go', 'cd5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-003_s3_cold_lifecycle.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-003_s3_cold_lifecycle.go', 'cc5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-004_backup_verify',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-004_backup_verify', 'abd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-005_velero_backup.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-005_velero_backup.go', 'c1a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-006_velero_restore.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-006_velero_restore.go', '9d9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-007_proxmox_vm_backup.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-007_proxmox_vm_backup.go', 'f7e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-008_minio_object_store.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-008_minio_object_store.go', '3a0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-009_bareos_config',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-009_bareos_config', 'af5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-001_osquery_drift.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-001_osquery_drift.go', 'f3a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-002_desired_state',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-002_desired_state', '7fd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-003_saltstack_reactor',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-003_saltstack_reactor', '67d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-004_rudder_drift',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-004_rudder_drift', 'ee1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-001_isolate_host.yml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-001_isolate_host.yml', 'd1a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-002_patch_cve.yml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-002_patch_cve.yml', '353'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-003_restart_service.yml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-003_restart_service.yml', 'fa6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-004_rollback.yml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-004_rollback.yml', 'f26'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-001_reactor_setup',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-001_reactor_setup', '49e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-002_state_apply.py',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-002_state_apply.py', 'af2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-003_sls_templates',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-003_sls_templates', '146'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-001_osquery_go_sdk.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-001_osquery_go_sdk.go', '917'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-002_fleetdm_policies.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-002_fleetdm_policies.go', 'f77'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-003_netbox_topology.py',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-003_netbox_topology.py', '85e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-004_docker_sdk.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-004_docker_sdk.go', 'de2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-001_micromdm_ios.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-001_micromdm_ios.go', '95a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-002_headwind_android.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-002_headwind_android.go', '35a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-003_android_enterprise',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-003_android_enterprise', '91e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-001_otel_config.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-001_otel_config.yaml', '9f9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-002_anomaly_model.pkl',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-002_anomaly_model.pkl', '8d9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-003_baseline_profiling',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-003_baseline_profiling', '96f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-004_prometheus_recording_rules.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-004_prometheus_recording_rules.yaml', '415'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-005_thanos_compactor.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-005_thanos_compactor.yaml', '705'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-006_grafana_datasources.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-006_grafana_datasources.yaml', 'c14'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-007_loki_promtail.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-007_loki_promtail.yaml', '559'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-008_tempo_otlp_receiver.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-008_tempo_otlp_receiver.yaml', '2e7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-009_victoriametrics_tsdb',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-009_victoriametrics_tsdb', '3df'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT/K-NOC-PT-001_delta_generator.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT/K-NOC-PT-001_delta_generator.go', 'e21'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT/K-NOC-PT-002_manifest_signer.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT/K-NOC-PT-002_manifest_signer.go', '4a2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-001_qbr_engine',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-001_qbr_engine', '8d3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-002_profitability',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-002_profitability', 'f3c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-003_grafana_qbr_export',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-003_grafana_qbr_export', '932'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-001_usage_aggregator.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-001_usage_aggregator.go', 'f06'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-002_pdf_renderer.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-002_pdf_renderer.go', '091'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-003_hle_constants.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-003_hle_constants.go', '082'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-004_contract_rates.sql',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-004_contract_rates.sql', '669'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-005_pdf_generator',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-005_pdf_generator', '61b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-006_stripe_payments.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-006_stripe_payments.go', '3eb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-001_contract_tables.sql',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-001_contract_tables.sql', 'ecd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-002_risk_quoting.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-002_risk_quoting.go', '66c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-003_pyfair_risk_model.py',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-003_pyfair_risk_model.py', 'ff7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-004_ltv_model.py',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-004_ltv_model.py', 'f89'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-001_ticket_state.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-001_ticket_state.go', '3bb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-002_sla_tracker.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-002_sla_tracker.go', '75d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-003_service_desk.sql',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-003_service_desk.sql', 'e1d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-004_multi_channel',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-004_multi_channel', '846'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-005_zammad_bridge.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-005_zammad_bridge.go', '6e9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-001_kiss_scorecard',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-001_kiss_scorecard', 'cf9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-002_white_label',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-002_white_label', 'd91'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-003_reasoning_playback',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-003_reasoning_playback', '17b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-001_AssetCard.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-001_AssetCard.tsx', '926'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-002_DeploymentWizard.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-002_DeploymentWizard.tsx', 'bd5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-003_ActionApproval.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-003_ActionApproval.tsx', 'dd2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-004_KissScorecard.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-004_KissScorecard.tsx', 'bf8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-005_RiskDashboard.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-005_RiskDashboard.tsx', '0a7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-006_BillingChart.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-006_BillingChart.tsx', '090'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-LIB-001_api_client.ts',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-LIB-001_api_client.ts', 'ae5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-LIB-002_nats_eventsource.ts',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-LIB-002_nats_eventsource.ts', '15d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-THEME/K-PSA-PTL-THEME-001_tenant_branding.css',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-THEME/K-PSA-PTL-THEME-001_tenant_branding.css', '217'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-001_sigma_compiler.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-001_sigma_compiler.go', '012'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-002_sigma_sync.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-002_sigma_sync.py', 'f9b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-003_mitre_mapper.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-003_mitre_mapper.py', '184'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-004_yara_integration',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-004_yara_integration', '641'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-005_suricata_rules',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-005_suricata_rules', '110'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-006_custom_detections',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-006_custom_detections', 'c80'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-007_sigma_rust_eval',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-007_sigma_rust_eval', 'ace'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-008_tetragon_k8s_ebpf',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-008_tetragon_k8s_ebpf', '5a6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-009_zeek_subprocess',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-009_zeek_subprocess', 'f66'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-FR_FORENSICS/K-SOC-FR-001_evidence_capture.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-FR_FORENSICS/K-SOC-FR-001_evidence_capture.go', '67b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-FR_FORENSICS/K-SOC-FR-002_blake3_evidence.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-FR_FORENSICS/K-SOC-FR-002_blake3_evidence.go', '172'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-001_bloodhound_analysis.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-001_bloodhound_analysis.go', 'caa'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-002_neo4j_graph.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-002_neo4j_graph.go', 'd2d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-003_cypher_ad_paths',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-003_cypher_ad_paths', 'c75'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-004_azure_oauth_queries',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-004_azure_oauth_queries', '74c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-001_redis_state.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-001_redis_state.go', 'a29'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-002_graph_correlation.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-002_graph_correlation.py', 'c61'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-003_incident_stitching',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-003_incident_stitching', '7ce'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-004_forensic_chain',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-004_forensic_chain', '861'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-001_otx_puller.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-001_otx_puller.py', '110'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-002_abuseipdb.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-002_abuseipdb.py', 'c13'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-003_malware_bazaar.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-003_malware_bazaar.py', '9f6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-004_phishing_tank.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-004_phishing_tank.py', 'e87'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-005_hibp',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-005_hibp', '90e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-006_cisa_kev',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-006_cisa_kev', 'c77'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-007_stix2_parser.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-007_stix2_parser.py', '423'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-008_stix2_validator.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-008_stix2_validator.py', 'f11'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-009_shodan_enrich.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-009_shodan_enrich.py', 'c61'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-010_censys_discovery.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-010_censys_discovery.py', '27c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-011_greynoise_filter.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-011_greynoise_filter.py', '1f6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-012_wiz_cloud_ti.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-012_wiz_cloud_ti.py', '627'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-013_misp_pymisp_client.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-013_misp_pymisp_client.py', 'e51'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-014_opencti_connector',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-014_opencti_connector', 'fff'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-015_ipsum_blocklist.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-015_ipsum_blocklist.py', '3b5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-001_nuclei_engine.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-001_nuclei_engine.go', '8d5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-002_epss_worker.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-002_epss_worker.py', '538'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-003_cve_priority.sql',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-003_cve_priority.sql', 'd2f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-004_trivy_scanner.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-004_trivy_scanner.go', '81f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-005_grype_db.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-005_grype_db.go', 'dc6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-006_syft_sbom_gen.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-006_syft_sbom_gen.go', 'd72'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-007_openvas_rest',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-007_openvas_rest', '62a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-008_checkov_iac.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-008_checkov_iac.py', '05b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-009_kics_engine.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-009_kics_engine.go', '0d2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-010_ssvc_decision_tree.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-010_ssvc_decision_tree.py', '181'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-011_nvd_api_puller.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-011_nvd_api_puller.go', '5b6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-000_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-000_INDEX', 'edf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-001_INDEX', '265'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-002_windows_cypher',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-002_windows_cypher', 'e60'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-003_azure_cypher',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-003_azure_cypher', '4f4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-001_INDEX', 'bb2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-002_analyzers',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-002_analyzers', '549'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-003_responders',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-003_responders', '97e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-004_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-004_license_boundary', '2c0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-001_INDEX', '44b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-002_falco_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-002_falco_rules', '725'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-003_k8s_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-003_k8s_rules', 'f3e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-001_INDEX', '148'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-002_taxonomies',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-002_taxonomies', '82c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-003_galaxies',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-003_galaxies', 'f86'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-004_warninglists',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-004_warninglists', 'd9e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-005_objects',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-005_objects', '7ef'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-006_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-006_sync_script.sh', '232'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-001_INDEX', 'c7b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-002_enterprise_attack',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-002_enterprise_attack', '9be'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-003_cwe_stix2',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-003_cwe_stix2', '433'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-004_capec_stix2',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-004_capec_stix2', 'a16'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-005_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-005_sync_script.sh', 'dbd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-001_INDEX', '63a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-002_cve_templates',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-002_cve_templates', 'b4c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-003_cloud_templates',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-003_cloud_templates', 'f61'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-004_http_api_templates',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-004_http_api_templates', 'e47'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-005_saas_templates',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-005_saas_templates', 'ab2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-006_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-006_sync_script.sh', '688'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-001_INDEX', '789'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-002_cis_benchmarks',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-002_cis_benchmarks', '7ae'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-003_stig_content',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-003_stig_content', '56c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-001_INDEX', '855'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-002_nist_800_53',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-002_nist_800_53', '386'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-003_pci_dss',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-003_pci_dss', '8df'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-004_iso_27001',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-004_iso_27001', '040'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-005_soc2',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-005_soc2', '404'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-001_INDEX', 'c39'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-002_incident_response',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-002_incident_response', '43d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-003_fim_packs',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-003_fim_packs', '497'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-004_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-004_sync_script.sh', '599'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-001_INDEX', '54e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-002_techniques',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-002_techniques', 'f89'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-003_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-003_license_boundary', '6c0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-001_INDEX', 'a3c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-002_soar_workflows',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-002_soar_workflows', '5be'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-003_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-003_license_boundary', '472'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-001_INDEX', 'de9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-002_windows_builtin',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-002_windows_builtin', '167'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-003_cloud_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-003_cloud_rules', 'b6c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-004_saas_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-004_saas_rules', '9cc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-005_hunting_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-005_hunting_rules', '56b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-006_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-006_sync_script.sh', 'd33'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-001_INDEX', '51c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-002_emerging_malware',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-002_emerging_malware', '116'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-003_emerging_c2',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-003_emerging_c2', 'a03'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-004_emerging_web',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-004_emerging_web', 'da2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-005_emerging_data',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-005_emerging_data', '79e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-006_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-006_sync_script.sh', 'aa6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-001_INDEX', '8c1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-002_case_schema',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-002_case_schema', '2c6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-003_alert_schema',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-003_alert_schema', '568'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-004_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-004_license_boundary', 'cdc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-001_INDEX', 'e9d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-002_threat_hunting',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-002_threat_hunting', 'a54'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-003_forensic_artifacts',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-003_forensic_artifacts', '1ba'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-004_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-004_license_boundary', '199'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-001_INDEX', '435'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-002_process_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-002_process_rules', 'c14'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-003_ad_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-003_ad_rules', '72a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-004_sca_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-004_sca_rules', 'd55'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-005_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-005_license_boundary', '3b2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-001_INDEX', 'b98'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-002_malware_sigs',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-002_malware_sigs', '58e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-003_pii_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-003_pii_rules', '630'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-004_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-004_sync_script.sh', '504'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-001_INDEX', 'a7d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-002_base_protocols',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-002_base_protocols', '8a6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-003_intel_framework',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-003_intel_framework', '118'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-004_http_scripts',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-004_http_scripts', '242'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-005_ja3_tls',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-005_ja3_tls', '665'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-001_Cargo.toml',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-001_Cargo.toml', '6eb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-002_eBPF_Compatibility',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-002_eBPF_Compatibility', 'd0a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-001_main.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-001_main.rs', '751'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-001_execve_hook.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-001_execve_hook.rs', '761'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-002_openat2_hook.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-002_openat2_hook.rs', '339'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-003_map_pressure.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-003_map_pressure.rs', '7d7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs', 'd67'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-002_blake3_baseline.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-002_blake3_baseline.rs', '152'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FORENSIC/K-XRO-CS-FR-001_memory_snapshot.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FORENSIC/K-XRO-CS-FR-001_memory_snapshot.rs', 'bc8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-GOVERNOR/K-XRO-CS-GV-001_token_bucket.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-GOVERNOR/K-XRO-CS-GV-001_token_bucket.rs', '2ca'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-ML/K-XRO-CS-ML-001_candle_inference.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-ML/K-XRO-CS-ML-001_candle_inference.rs', '915'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-ML/K-XRO-CS-ML-002_tinyllama_loader.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-ML/K-XRO-CS-ML-002_tinyllama_loader.rs', '173'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs', 'd61'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-002_ocsf_event_bridge.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-002_ocsf_event_bridge.rs', '8c3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-001_yara_compiler.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-001_yara_compiler.rs', 'd62'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-002_malware_scanner.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-002_malware_scanner.rs', '984'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-001_Cargo.toml',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-001_Cargo.toml', '75c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-002_10G_Validation',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-002_10G_Validation', 'ca0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-001_main.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-001_main.rs', 'b8d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-001_ndpi_ffi.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-001_ndpi_ffi.rs', '9e8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-002_l7_classifier.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-002_l7_classifier.rs', '0de'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-001_rule_loader.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-001_rule_loader.rs', '4ce'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-002_alert_publisher.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-002_alert_publisher.rs', '25b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-001_flow_analyzer.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-001_flow_analyzer.rs', '429'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-002_tls_sni.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-002_tls_sni.rs', 'd55'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-003_af_packet_ring.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-003_af_packet_ring.rs', '037'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-004_dpdk_bypass.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-004_dpdk_bypass.rs', 'f08'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-001_beacon_detector.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-001_beacon_detector.go', 'd8f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-002_dns_tunnel.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-002_dns_tunnel.go', '698'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-003_exfil_detector.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-003_exfil_detector.go', '1d5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-TI/K-XRO-NG-TI-001_ipsum_lookup.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-TI/K-XRO-NG-TI-001_ipsum_lookup.rs', '941'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-001_Cargo.toml',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-001_Cargo.toml', '823'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-006_Baseline_Schema.json',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-006_Baseline_Schema.json', '601'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-001_main.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-001_main.rs', 'bda'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-002_perf_event_open.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-002_perf_event_open.rs', '934'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-003_prometheus.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-003_prometheus.rs', 'fa1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-004_otel_collector.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-004_otel_collector.rs', '005'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-005_sysinfo_host_metrics.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-005_sysinfo_host_metrics.rs', '94f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-001_registration_handler.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-001_registration_handler.go', '717'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-002_install_script_gen.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-002_install_script_gen.go', '2a5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-003_blake3_fingerprinter.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-003_blake3_fingerprinter.go', 'bfb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-SD_SIDECARS/K-XRO-SD-001_rustdesk_remote',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-SD_SIDECARS/K-XRO-SD-001_rustdesk_remote', 'c73'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-SD_SIDECARS/K-XRO-SD-002_tetragon_ebpf',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-SD_SIDECARS/K-XRO-SD-002_tetragon_ebpf', '4db'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs', 'b37'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-002_zstd_delta.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-002_zstd_delta.go', '294'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-003_manifest_signer.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-003_manifest_signer.go', 'b74'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-004_go_tuf_updater.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-004_go_tuf_updater.go', 'faa'),
                exact: true,
                sidebar: "docs"
              }
            ]
          }
        ]
      }
    ]
  },
  {
    path: '/',
    component: ComponentCreator('/', 'fd5'),
    exact: true
  },
  {
    path: '*',
    component: ComponentCreator('*'),
  },
];
