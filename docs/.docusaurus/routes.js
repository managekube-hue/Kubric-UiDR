import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/__docusaurus/debug',
    component: ComponentCreator('/__docusaurus/debug', '5ff'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/config',
    component: ComponentCreator('/__docusaurus/debug/config', '5ba'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/content',
    component: ComponentCreator('/__docusaurus/debug/content', 'a2b'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/globalData',
    component: ComponentCreator('/__docusaurus/debug/globalData', 'c3c'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/metadata',
    component: ComponentCreator('/__docusaurus/debug/metadata', '156'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/registry',
    component: ComponentCreator('/__docusaurus/debug/registry', '88c'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/routes',
    component: ComponentCreator('/__docusaurus/debug/routes', '000'),
    exact: true
  },
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
    component: ComponentCreator('/docs', 'cdf'),
    routes: [
      {
        path: '/docs',
        component: ComponentCreator('/docs', 'c3b'),
        routes: [
          {
            path: '/docs',
            component: ComponentCreator('/docs', 'c38'),
            routes: [
              {
                path: '/docs/intro',
                component: ComponentCreator('/docs/intro', '544'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-NATS-001_subject_hierarchy',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-NATS-001_subject_hierarchy', '4fc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-001_provisioning.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-001_provisioning.yaml', '12a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-002_triage.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-002_triage.yaml', '131'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-003_billing.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-003_billing.yaml', 'ca8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-004_vdr_scan.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-004_vdr_scan.yaml', 'b82'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-005_grc_compliance.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-005_grc_compliance.yaml', '44c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-006_identity_graph.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-006_identity_graph.yaml', 'b6c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-007_ndr_flow.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-007_ndr_flow.yaml', '425'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-008_health.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-008_health.yaml', '247'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-009_alerts.yaml',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-OPENAPI/K-API-OPEN-009_alerts.yaml', '09e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-001_ocsf_schema.proto',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-001_ocsf_schema.proto', '3c3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-002_build_rs.rs',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-002_build_rs.rs', '269'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-003_ocsf_deploy.proto',
                component: ComponentCreator('/docs/K-API-09_API_REFERENCE/K-API-PB_PROTOBUF/K-API-PB-003_ocsf_deploy.proto', '259'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-001_Cluster_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-001_Cluster_Config', '8c7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-002_OCSF_Schema.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-002_OCSF_Schema.sql', '37d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-003_TTL_Cold_Storage',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-003_TTL_Cold_Storage', '4e1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-004_Agent_Decision_History.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-004_Agent_Decision_History.sql', '592'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-005_Arrow_Bulk_Insert',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-CLICKHOUSE/K-DL-CH-005_Arrow_Bulk_Insert', 'bfd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-DUCKDB/K-DL-DUCK-001_embedded_analytics',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-DUCKDB/K-DL-DUCK-001_embedded_analytics', '9f1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-DUCKDB/K-DL-DUCK-002_ml_feature_compute.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-DUCKDB/K-DL-DUCK-002_ml_feature_compute.sql', '136'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-001_golang_migrate_setup',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-001_golang_migrate_setup', 'b2a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-002_liquibase_k8s.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-002_liquibase_k8s.yaml', 'a33'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-003_atlas_ci_sync',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-MIGRATIONS/K-DL-MIG-003_atlas_ci_sync', '81c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-001_UAR_Asset_Table.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-001_UAR_Asset_Table.sql', '107'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-002_RLS_Policies.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-002_RLS_Policies.sql', 'a6b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-003_Contract_Rate_Tables.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-003_Contract_Rate_Tables.sql', '5f1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-004_OSCAL_Ingestion.sql',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-004_OSCAL_Ingestion.sql', '572'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-005_Atlas_Schema_HCL.hcl',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/K-DL-POSTGRES/K-DL-PG-005_Atlas_Schema_HCL.hcl', 'cbf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-001_Cluster_Bootstrap',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-001_Cluster_Bootstrap', 'f0f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-002_Ceph_Storage',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-002_Ceph_Storage', 'b7d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-003_Chrony_PTP_TimeSync',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-003_Chrony_PTP_TimeSync', 'ccc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-001_Gitea',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-001_Gitea', '042'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-002_n8n',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-002_n8n', '04e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-003_Caddy',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-003_Caddy', '4a4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-004_Woodpecker_CI',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-LXC_CONTAINERS/K-HV-LXC-004_Woodpecker_CI', '287'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-001_Go_API_CloudInit',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-001_Go_API_CloudInit', '647'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-002_ClickHouse_StatefulSet',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-002_ClickHouse_StatefulSet', '0a1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-003_PostgreSQL_RLS',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-003_PostgreSQL_RLS', '3c2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-004_Ollama_LLM',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-004_Ollama_LLM', 'b41'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-005_vLLM_GPU_Node',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/K-HV-VM_TEMPLATES/K-HV-VM-005_vLLM_GPU_Node', 'c5c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-001_Node1_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-001_Node1_Config', '879'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-002_Node2_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-002_Node2_Config', 'e10'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-003_Node3_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-003_Node3_Config', 'dff'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-004_iDRAC9_Network',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-004_iDRAC9_Network', '734'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-005_RAM_Expansion',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-HW-R630_HARDWARE/K-HW-005_RAM_Expansion', '4cf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-001_ClickHouse_StatefulSet.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-001_ClickHouse_StatefulSet.yaml', 'f71'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-002_NATS_StatefulSet.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-002_NATS_StatefulSet.yaml', '217'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-003_PostgreSQL_StatefulSet.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-003_PostgreSQL_StatefulSet.yaml', 'e87'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-004_API_Deployment.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-004_API_Deployment.yaml', '444'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-001_ArgoCD_Application.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-001_ArgoCD_Application.yaml', '8e5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-002_Flux_GitRepository.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-002_Flux_GitRepository.yaml', 'c5a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-003_Helm_Values.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-003_Helm_Values.yaml', '7fc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-004_Kustomize_Overlays.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-004_Kustomize_Overlays.yaml', 'baf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-005_Gatekeeper_OPA.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-GITOPS/K-K8S-GO-005_Gatekeeper_OPA.yaml', 'e05'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-001_Istio_ServiceMesh.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-001_Istio_ServiceMesh.yaml', '89d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-002_Linkerd_Config.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-002_Linkerd_Config.yaml', '8a2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-003_Cilium_CNI.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-003_Cilium_CNI.yaml', '1ca'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-004_Hubble_ServiceMap.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-004_Hubble_ServiceMap.yaml', '909'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-005_CertManager_Vault.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-005_CertManager_Vault.yaml', '4e1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-006_ExternalSecrets_Operator.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-MESH_SERVICE_MESH/K-K8S-MESH-006_ExternalSecrets_Operator.yaml', '33c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-001_Prometheus_Operator.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-001_Prometheus_Operator.yaml', '63e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-002_Thanos_Sidecar.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-002_Thanos_Sidecar.yaml', '5d2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-003_Grafana_Deployment.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-003_Grafana_Deployment.yaml', 'd42'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-004_Loki_Stack.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-004_Loki_Stack.yaml', '700'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-005_Tempo_Tracing.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-OBS_OBSERVABILITY/K-K8S-OBS-005_Tempo_Tracing.yaml', '68f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-001_SealedSecrets_Controller.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-001_SealedSecrets_Controller.yaml', '3f7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-002_NetworkPolicy_Defaults.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-002_NetworkPolicy_Defaults.yaml', 'd4a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-003_ResourceQuota.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/K-K8S-POLICY/K-K8S-POL-003_ResourceQuota.yaml', 'e38'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-001_NATS_Cluster.yaml',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-001_NATS_Cluster.yaml', '7bb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-002_JetStream_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-002_JetStream_Config', '322'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-003_mTLS_Cert_Rotation',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-003_mTLS_Cert_Rotation', '18c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-004_ZeroMQ_IPC',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-004_ZeroMQ_IPC', 'fc1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-001_edr.process.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-001_edr.process.v1', '5cd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-002_edr.file.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-002_edr.file.v1', '244'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-003_ndr.flow.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-003_ndr.flow.v1', 'ac0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-004_ndr.beacon.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-004_ndr.beacon.v1', 'bd5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-005_itdr.auth.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-005_itdr.auth.v1', '6f0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-006_vdr.vuln.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-006_vdr.vuln.v1', '3ad'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-007_grc.drift.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-007_grc.drift.v1', '83d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-008_svc.ticket.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-008_svc.ticket.v1', 'd7e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-009_billing.usage.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-009_billing.usage.v1', '78c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-010_health.score.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-010_health.score.v1', 'd02'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-011_ti.ioc.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-011_ti.ioc.v1', '176'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-012_comm.alert.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-012_comm.alert.v1', 'b24'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-013_security.alert.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-013_security.alert.v1', '3f7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-014_remediation.task.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-014_remediation.task.v1', '001'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-015_asset.provisioned.v1',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/K-MB-SUBJECT_MAPPING/K-MB-SUB-015_asset.provisioned.v1', 'aa4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-001_10G_SFP_Config',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-001_10G_SFP_Config', '0b1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-002_Corosync_Heartbeat',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-002_Corosync_Heartbeat', 'ec5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-003_Virtual_IP_Failover',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-003_Virtual_IP_Failover', '9a0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-004_HAProxy_Keepalived',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/K-NET-004_HAProxy_Keepalived', 'bf5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-001_HashiCorp_Vault',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-001_HashiCorp_Vault', 'd80'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-002_TPM_Root_of_Trust',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-002_TPM_Root_of_Trust', 'd7e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-003_Blake3_Fingerprint.go',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-003_Blake3_Fingerprint.go', '9a1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-004_CA_Setup',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-004_CA_Setup', '2c5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-005_Vault_Policies.hcl',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-005_Vault_Policies.hcl', '86e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-006_Vault_K8s_Auth.go',
                component: ComponentCreator('/docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/K-SEC-006_Vault_K8s_Auth.go', '11b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-000_INDEX',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-000_INDEX', '958'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-001_grafana_overview.json',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-001_grafana_overview.json', '86c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-002_prometheus_rules.yaml',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-002_prometheus_rules.yaml', 'f61'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-003_tuf_root.json',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-DASH-003_tuf_root.json', '8f4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-001_terraform_aws_eks.tf',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-001_terraform_aws_eks.tf', 'c48'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-002_vpc_config.tf',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-002_vpc_config.tf', 'b54'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-003_node_groups.tf',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-LARGE/K-DEPLOY-LG-003_node_groups.tf', '094'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-MEDIUM/K-DEPLOY-MD-001_kustomize_overlay.yaml',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-MEDIUM/K-DEPLOY-MD-001_kustomize_overlay.yaml', '962'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-MEDIUM/K-DEPLOY-MD-002_scale_config.yaml',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-MEDIUM/K-DEPLOY-MD-002_scale_config.yaml', '58f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-SMALL/K-DEPLOY-SM-001_docker-compose.yml',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-SMALL/K-DEPLOY-SM-001_docker-compose.yml', '895'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-SMALL/K-DEPLOY-SM-002_nats_single.conf',
                component: ComponentCreator('/docs/K-DEPLOY-12_TOPOLOGIES/K-DEPLOY-SMALL/K-DEPLOY-SM-002_nats_single.conf', '54e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-001_Makefile',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-001_Makefile', '062'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-002_rust-toolchain.toml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-002_rust-toolchain.toml', '2f8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-003_go.mod',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-003_go.mod', '850'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-004_package.json',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-004_package.json', '030'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-005_cobra_cli.go',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-005_cobra_cli.go', '705'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-006_chi_cors_middleware.go',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-006_chi_cors_middleware.go', 'dec'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-007_chi_jwt_auth.go',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-007_chi_jwt_auth.go', 'fdb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-008_workspace_cargo.toml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-008_workspace_cargo.toml', '51d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-009_buf_protobuf.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-009_buf_protobuf.yaml', '5ce'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-010_requirements_kai_core.txt',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-010_requirements_kai_core.txt', '568'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-011_requirements_kai_full.txt',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/K-DEV-BLD-011_requirements_kai_full.txt', '299'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-001_self_hosted_runner',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-001_self_hosted_runner', '0ec'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-002_woodpecker_pipeline.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-002_woodpecker_pipeline.yml', '30e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-003_jenkins_x_config.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-003_jenkins_x_config.yaml', '595'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-001_build-agents.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-001_build-agents.yml', '7c9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-002_test-api.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-002_test-api.yml', '8e6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-003_deploy-k8s.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-003_deploy-k8s.yml', '88f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-004_drone_config.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-004_drone_config.yml', '443'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-005_tekton_pipeline.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-005_tekton_pipeline.yaml', '821'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-006_concourse_pipeline.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-006_concourse_pipeline.yml', '145'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-007_dagger_ci.go',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-007_dagger_ci.go', '4de'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-008_earthly_Earthfile',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-008_earthly_Earthfile', '847'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-009_cosign_signing.sh',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-009_cosign_signing.sh', 'd0e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-010_snyk_scan.sh',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-010_snyk_scan.sh', '27e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-011_sonarqube_scanner.sh',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD/K-DEV-CICD-GHA_WORKFLOWS/K-DEV-CICD-GHA-011_sonarqube_scanner.sh', 'c60'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-001_architecture',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-001_architecture', 'f14'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-002_README',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-002_README', '44b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-003_LICENSE',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-003_LICENSE', '957'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-004_NOTICE',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-004_NOTICE', '4dd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-005_license_compliance_matrix',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-DOC_DOCUMENTATION/K-DEV-DOC-005_license_compliance_matrix', 'dae'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-001_gitea_setup',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-001_gitea_setup', 'cfb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-002_pre-commit.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-002_pre-commit.yaml', '2d4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-003_molt-scanner.sh',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-003_molt-scanner.sh', '7e2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-004_branch_protection',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-004_branch_protection', 'c6d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-005_ruff_config.toml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-005_ruff_config.toml', 'ea9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-006_golangci_lint.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-006_golangci_lint.yml', '91a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-007_clippy_config.toml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-007_clippy_config.toml', '2c1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-008_eslint_config.js',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-008_eslint_config.js', '483'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-009_pre_commit_config.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-009_pre_commit_config.yaml', 'eb0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-010_commitlint_config.js',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-010_commitlint_config.js', '1de'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-011_semantic_release.json',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-011_semantic_release.json', '334'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-012_black_config.toml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-012_black_config.toml', '307'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-013_isort_config.cfg',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-013_isort_config.cfg', '953'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-014_mypy_config.ini',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-014_mypy_config.ini', '236'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-015_pylintrc',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-015_pylintrc', '8e0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-016_bandit_config.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-016_bandit_config.yaml', '0d0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-017_safety_policy.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/K-DEV-GIT-017_safety_policy.yml', '5d9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-001_docker-compose.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-001_docker-compose.yml', '4a9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-002_docker-compose-small.yml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-002_docker-compose-small.yml', '9a6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-001_clickhouse_users.xml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-001_clickhouse_users.xml', 'd5d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-002_postgres_init.sql',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-002_postgres_init.sql', 'a37'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-003_nats_cluster.conf',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-003_nats_cluster.conf', '9ca'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-004_vault_dev.hcl',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/K-DEV-LOCAL-CONFIG/K-DEV-LOCAL-CFG-004_vault_dev.hcl', 'a6c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-001_k6_load_test.js',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-001_k6_load_test.js', '50a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-002_vegeta_attack.sh',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-002_vegeta_attack.sh', '2d2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-003_kube_burner_config.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-003_kube_burner_config.yaml', 'fbb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-004_chaos_mesh_experiment.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-004_chaos_mesh_experiment.yaml', '9bd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-005_litmus_chaos_engine.yaml',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-005_litmus_chaos_engine.yaml', '7f0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-006_pytest_xdist_config.ini',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-006_pytest_xdist_config.ini', '698'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-007_factory_boy_factories.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-007_factory_boy_factories.py', 'f0d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-008_faker_data_gen.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-008_faker_data_gen.py', '5fe'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-009_hypothesis_property_tests.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TEST/K-DEV-TEST-009_hypothesis_property_tests.py', 'd61'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-001_click_cli.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-001_click_cli.py', 'bec'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-002_typer_cli.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-002_typer_cli.py', '526'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-003_rich_output.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-003_rich_output.py', '622'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-004_tqdm_progress.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-004_tqdm_progress.py', '67f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-005_colorama_windows.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-005_colorama_windows.py', 'e37'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-006_tabulate_tables.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-006_tabulate_tables.py', '9e1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-007_jsonpath_extractor.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-007_jsonpath_extractor.py', 'b92'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-008_jmespath_querier.py',
                component: ComponentCreator('/docs/K-DEV-08_DEVELOPMENT/K-DEV-TOOLS/K-DEV-TOOLS-008_jmespath_querier.py', '488'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-001_lula_validator.go',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-001_lula_validator.go', '5a5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-002_openscap_binding.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-002_openscap_binding.py', '02e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-003_kyverno_policy.go',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-CA_COMPLIANCE_AUTO/K-GRC-CA-003_kyverno_policy.go', '723'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-001_immutable_audit.sql',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-001_immutable_audit.sql', '0af'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-002_blake3_signer.go',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-002_blake3_signer.go', 'e4b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-003_legal_hold',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-003_legal_hold', '330'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-004_evidence_export',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/K-GRC-EV-004_evidence_export', '1ad'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-001_nist_800_53_oscal',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-001_nist_800_53_oscal', '9a9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-002_pci_dss_oscal',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-002_pci_dss_oscal', 'a4f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-003_iso_27001_oscal',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-003_iso_27001_oscal', '800'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-004_soc2_oscal',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-FW-004_soc2_oscal', 'b0b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-001_nist_ingest.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-001_nist_ingest.py', '944'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-002_soc2_mapper.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-002_soc2_mapper.py', '32c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-003_iso_mapping.sql',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-003_iso_mapping.sql', 'ef7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-004_compliance_trestle.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-004_compliance_trestle.py', '48e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-005_regscale_ingest.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL/K-GRC-OSCAL-005_regscale_ingest.py', '0ce'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-001_sbom_syft.go',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-001_sbom_syft.go', '7b9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-002_grype_scanner.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-002_grype_scanner.py', 'e9f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-003_openssf_scorecard',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-003_openssf_scorecard', '6c4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-004_sbom_generation',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-004_sbom_generation', '136'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-005_sigstore_cosign.sh',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-005_sigstore_cosign.sh', '8ba'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-006_osv_api_check.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-006_osv_api_check.py', 'c92'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-007_dependency_track.go',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-007_dependency_track.go', '3b2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-008_cyclonedx_sbom.py',
                component: ComponentCreator('/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/K-GRC-SCS-008_cyclonedx_sbom.py', '5cf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-001_KIC_evidence_map',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-001_KIC_evidence_map', 'edd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-002_soc2_iso_crosswalk.cs',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-AUDIT_AUDIT_READINESS/K-ITIL-AUD-002_soc2_iso_crosswalk.cs', '6c5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-001_GMP1_Strategy',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-001_GMP1_Strategy', 'a50'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-002_GMP5_Risk',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-002_GMP5_Risk', '121'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-003_GMP6_InfoSec',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-003_GMP6_InfoSec', 'f08'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-004_SMP1_Incident',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-004_SMP1_Incident', 'd2f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-005_SMP10_Change',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-005_SMP10_Change', 'acb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-006_SMP12_Deployment',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-006_SMP12_Deployment', '9ba'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-007_TMP2_Infrastructure',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-007_TMP2_Infrastructure', '6a3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-008_SMP3_Problem',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-008_SMP3_Problem', '427'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-009_SMP7_ServiceLevel',
                component: ComponentCreator('/docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-MATRIX_PRACTICE_MAP/K-ITIL-MAT-009_SMP7_ServiceLevel', '6f8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-001_fastapi_server.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-001_fastapi_server.py', '1ff'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-002_asyncpg_client.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-002_asyncpg_client.py', '718'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-003_psycopg2_fallback.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-003_psycopg2_fallback.py', 'f4d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-004_clickhouse_connect.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-004_clickhouse_connect.py', '6e6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-005_aiokafka_consumer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-005_aiokafka_consumer.py', '28a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-006_nats_py_client.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-006_nats_py_client.py', 'c2f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-007_anyio_backend.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-007_anyio_backend.py', 'd9e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-008_asgiref_sync.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-008_asgiref_sync.py', 'ecc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-009_socketio_realtime.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-API/K-KAI-API-009_socketio_realtime.py', '886'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-AUDIT/K-KAI-AUD-001_decision_history.sql',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-AUDIT/K-KAI-AUD-001_decision_history.sql', '80a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-AUDIT/K-KAI-AUD-002_merkle_signer.go',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-AUDIT/K-KAI-AUD-002_merkle_signer.go', '0fc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-001_cortex_analyzer_chain.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-001_cortex_analyzer_chain.py', 'f13'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-002_observable_enrichment.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-ANALYST/K-KAI-AN-002_observable_enrichment.py', '5eb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-001_billing_clerk.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-001_billing_clerk.py', '8e2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-002_clickhouse_audit.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-002_clickhouse_audit.py', 'fad'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-003_hle_calculator.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-003_hle_calculator.py', '694'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-004_invoice_renderer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-BILL/K-KAI-BL-004_invoice_renderer.py', 'd72'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-001_comm_agent.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-001_comm_agent.py', '530'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-002_vapi_phone.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-002_vapi_phone.py', 'ad2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-003_twilio_sms.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-COMM/K-KAI-CM-003_twilio_sms.py', '1ad'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-001_deploy_agent.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-001_deploy_agent.py', '2cc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-002_saltstack_client.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-002_saltstack_client.py', '7a0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-003_fleet_rollout.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-DEPLOY/K-KAI-DEP-003_fleet_rollout.py', 'ffb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-001_lstm_baseline.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-001_lstm_baseline.py', '283'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-002_epss_enrichment.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-002_epss_enrichment.py', '855'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-003_hikari_trainer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-FORESIGHT/K-KAI-FS-003_hikari_trainer.py', '709'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-001_housekeeper.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-001_housekeeper.py', 'be7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-002_ansible_runner.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-002_ansible_runner.py', 'bcf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-003_criticality_check.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-003_criticality_check.py', '32e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-004_rollback.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HOUSE/K-KAI-HS-004_rollback.py', 'b5b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/K-KAI-HU-001_velociraptor_artifacts.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/K-KAI-HU-001_velociraptor_artifacts.py', '76b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/K-KAI-HU-002_sigma_hunting_runner.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-HUNTER/K-KAI-HU-002_sigma_hunting_runner.py', 'a0c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-001_misp_galaxy_query.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-001_misp_galaxy_query.py', 'd61'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-002_graph_investigation.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-INVEST/K-KAI-IV-002_graph_investigation.py', '337'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-001_remediation_planner.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-001_remediation_planner.py', 'f05'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-002_cortex_subprocess.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-002_cortex_subprocess.py', '34a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-003_vault_secret_fetcher.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-KEEPER/K-KAI-KP-003_vault_secret_fetcher.py', '36d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-001_pyfair_model.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-001_pyfair_model.py', '1a4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-002_epss_scorer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-002_epss_scorer.py', '2c4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-003_ssvc_decision.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-RISK/K-KAI-RISK-003_ssvc_decision.py', '7b8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-001_health_score_publisher.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-001_health_score_publisher.py', 'eec'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-002_churn_risk_model.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-002_churn_risk_model.py', 'a32'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-003_hibp_credential_score.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SENTINEL/K-KAI-SEN-003_hibp_credential_score.py', 'ec9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-001_ltv_predictor.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-001_ltv_predictor.py', 'c40'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-002_churn_simulator.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-002_churn_simulator.py', '4ad'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-003_dynamic_pricing.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-SIMULATE/K-KAI-SIM-003_dynamic_pricing.py', '930'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-001_triage_agent.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-001_triage_agent.py', '902'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-002_llama3_reasoning.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-002_llama3_reasoning.py', '53f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-003_ocsf_analyzer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-003_ocsf_analyzer.py', 'c2a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-004_kiss_calculator.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/K-KAI-TRIAGE/K-KAI-TR-004_kiss_calculator.py', 'ae8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-001_human_mfa.go',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-001_human_mfa.go', 'a71'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-002_action_queue.sql',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-002_action_queue.sql', '8cf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-003_criticality_5.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-003_criticality_5.py', '401'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-004_prompt_injection.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/K-KAI-GD-004_prompt_injection.py', 'b9d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-001_polars_dataframe.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-001_polars_dataframe.py', 'c05'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-002_pyarrow_parquet.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-002_pyarrow_parquet.py', '26b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-003_fastparquet_io.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-003_fastparquet_io.py', '8d5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-004_orjson_serializer.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-004_orjson_serializer.py', 'cb6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-005_ujson_fallback.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-005_ujson_fallback.py', 'cb0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-006_msgpack_encoder.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-006_msgpack_encoder.py', '7ef'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-007_dpkt_pcap_parser.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-007_dpkt_pcap_parser.py', '5bd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-008_scapy_probe.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-008_scapy_probe.py', '5df'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-009_pcap_capture.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-009_pcap_capture.py', '977'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-010_geoip2_resolver.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-LIBS/K-KAI-LIBS-010_geoip2_resolver.py', '712'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-001_tensorboard_logger.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-001_tensorboard_logger.py', '282'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-002_clearml_experiment.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-002_clearml_experiment.py', 'aa0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-003_pyspark_distributed.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-003_pyspark_distributed.py', 'c4f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-004_openai_fallback.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-004_openai_fallback.py', '3a3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-005_anthropic_long_context.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-005_anthropic_long_context.py', '194'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-006_cohere_embeddings.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-006_cohere_embeddings.py', '483'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-007_hikari_preprocessor.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-007_hikari_preprocessor.py', '949'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-008_ember_xgboost.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-008_ember_xgboost.py', 'b77'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-009_unswnb15_random_forest.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-009_unswnb15_random_forest.py', 'd7b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-010_mordor_lstm_baseline.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-010_mordor_lstm_baseline.py', 'b43'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-011_vllm_server.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-011_vllm_server.py', '00c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-012_model_tiering',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-ML-012_model_tiering', '223'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-001_vector_search.sql',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-001_vector_search.sql', 'a2b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-002_oscal_embeddings.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-002_oscal_embeddings.py', '490'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-003_ciso_assistant.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-003_ciso_assistant.py', '4c5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-004_cohere_embeddings.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG/K-KAI-RAG-004_cohere_embeddings.py', 'a9a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-001_security_triage.json',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-001_security_triage.json', '807'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-002_drift_housekeeper.json',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-002_drift_housekeeper.json', '058'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-003_heartbeat_billing.json',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-n8n/K-KAI-WF-n8n-003_heartbeat_billing.json', '5f7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-001_patch_workflow.go',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-001_patch_workflow.go', 'd51'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-002_retry_state.go',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-002_retry_state.go', '780'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-003_celery_tasks.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-003_celery_tasks.py', 'fd7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-004_flower_monitor.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-004_flower_monitor.py', '192'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-005_dramatiq_worker.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-005_dramatiq_worker.py', '76a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-006_huey_scheduler.py',
                component: ComponentCreator('/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/K-KAI-WF-TEMPORAL/K-KAI-WF-TEMP-006_huey_scheduler.py', '0d9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-000_MASTER_INDEX',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-000_MASTER_INDEX', '8e3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-001_EDR_Endpoint',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-001_EDR_Endpoint', '206'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-002_ITDR_Identity',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-002_ITDR_Identity', '83e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-003_NDR_Network',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-003_NDR_Network', '015'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-004_CDR_Cloud',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-004_CDR_Cloud', '8b1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-005_SDR_SaaS',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-005_SDR_SaaS', 'a2d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-006_ADR_Application',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-006_ADR_Application', '559'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-007_DDR_Data',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-007_DDR_Data', 'e47'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-008_VDR_Vulnerability',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-008_VDR_Vulnerability', '24f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-009_MDR_Managed',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-009_MDR_Managed', 'c64'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-010_TI_ThreatIntel',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-010_TI_ThreatIntel', 'bf9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-011_CFDR_ConfigDrift',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-011_CFDR_ConfigDrift', '4de'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-012_BDR_Backup',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-012_BDR_Backup', '157'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-013_NPM_NetworkPerf',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-013_NPM_NetworkPerf', 'c7f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-014_UEM_EndpointMgmt',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-014_UEM_EndpointMgmt', '000'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-015_MDM_Mobile',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-015_MDM_Mobile', 'c38'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-016_APM_AppPerf',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-016_APM_AppPerf', 'f76'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-017_GRC_Governance',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-017_GRC_Governance', '43b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-018_KAI_AILayer',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-018_KAI_AILayer', '748'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-019_PSA_Business',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-019_PSA_Business', 'b1f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-020_LICENSE_COMPLIANCE',
                component: ComponentCreator('/docs/K-MAP-11_DR_MODULE_MAPPING/K-MAP-020_LICENSE_COMPLIANCE', 'e12'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-001_restic_scheduler.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-001_restic_scheduler.go', 'f70'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-002_kopia_snapshots.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-002_kopia_snapshots.go', '174'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-003_s3_cold_lifecycle.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-003_s3_cold_lifecycle.go', 'd96'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-004_backup_verify',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-004_backup_verify', '4dc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-005_velero_backup.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-005_velero_backup.go', '331'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-006_velero_restore.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-006_velero_restore.go', 'cdc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-007_proxmox_vm_backup.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-007_proxmox_vm_backup.go', 'd78'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-008_minio_object_store.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-008_minio_object_store.go', 'd18'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-009_bareos_config',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/K-NOC-BR-009_bareos_config', 'db0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-001_osquery_drift.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-001_osquery_drift.go', '0b6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-002_desired_state',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-002_desired_state', '959'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-003_saltstack_reactor',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-003_saltstack_reactor', '0fe'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-004_rudder_drift',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-004_rudder_drift', 'e24'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-001_isolate_host.yml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-001_isolate_host.yml', 'cd1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-002_patch_cve.yml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-002_patch_cve.yml', '3af'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-003_restart_service.yml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-003_restart_service.yml', 'e48'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-004_rollback.yml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-ANSIBLE/K-NOC-CM-ANS-004_rollback.yml', '0af'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-001_reactor_setup',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-001_reactor_setup', 'cde'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-002_state_apply.py',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-002_state_apply.py', '224'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-003_sls_templates',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/K-NOC-CM-SALTSTACK/K-NOC-CM-SALT-003_sls_templates', '66a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-001_osquery_go_sdk.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-001_osquery_go_sdk.go', '16f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-002_fleetdm_policies.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-002_fleetdm_policies.go', 'a54'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-003_netbox_topology.py',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-003_netbox_topology.py', '202'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-004_docker_sdk.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-INV_INVENTORY/K-NOC-INV-004_docker_sdk.go', '449'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-001_micromdm_ios.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-001_micromdm_ios.go', 'b02'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-002_headwind_android.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-002_headwind_android.go', 'fe2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-003_android_enterprise',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-MDM-003_android_enterprise', 'd37'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-001_otel_config.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-001_otel_config.yaml', '863'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-002_anomaly_model.pkl',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-002_anomaly_model.pkl', 'e4a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-003_baseline_profiling',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-003_baseline_profiling', 'c64'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-004_prometheus_recording_rules.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-004_prometheus_recording_rules.yaml', 'f45'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-005_thanos_compactor.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-005_thanos_compactor.yaml', 'fcf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-006_grafana_datasources.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-006_grafana_datasources.yaml', 'bdb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-007_loki_promtail.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-007_loki_promtail.yaml', '127'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-008_tempo_otlp_receiver.yaml',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-008_tempo_otlp_receiver.yaml', 'a68'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-009_victoriametrics_tsdb',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/K-NOC-PM-009_victoriametrics_tsdb', 'f06'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT/K-NOC-PT-001_delta_generator.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT/K-NOC-PT-001_delta_generator.go', '61b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT/K-NOC-PT-002_manifest_signer.go',
                component: ComponentCreator('/docs/K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT/K-NOC-PT-002_manifest_signer.go', '30b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-001_qbr_engine',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-001_qbr_engine', 'bb3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-002_profitability',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-002_profitability', 'e00'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-003_grafana_qbr_export',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/K-PSA-BI-003_grafana_qbr_export', '541'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-001_usage_aggregator.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-001_usage_aggregator.go', '371'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-002_pdf_renderer.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-002_pdf_renderer.go', 'c51'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-003_hle_constants.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-003_hle_constants.go', 'e97'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-004_contract_rates.sql',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-004_contract_rates.sql', 'e5c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-005_pdf_generator',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-005_pdf_generator', '65e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-006_stripe_payments.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/K-PSA-BILL-006_stripe_payments.go', '059'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-001_contract_tables.sql',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-001_contract_tables.sql', 'c76'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-002_risk_quoting.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-002_risk_quoting.go', '7f4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-003_pyfair_risk_model.py',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-003_pyfair_risk_model.py', '42d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-004_ltv_model.py',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/K-PSA-CRM-004_ltv_model.py', 'c9f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-001_ticket_state.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-001_ticket_state.go', '987'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-002_sla_tracker.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-002_sla_tracker.go', '373'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-003_service_desk.sql',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-003_service_desk.sql', '192'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-004_multi_channel',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-004_multi_channel', 'cad'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-005_zammad_bridge.go',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-ITSM/K-PSA-ITSM-005_zammad_bridge.go', '4c0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-001_kiss_scorecard',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-001_kiss_scorecard', 'fcb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-002_white_label',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-002_white_label', '297'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-003_reasoning_playback',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-003_reasoning_playback', '26f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-001_AssetCard.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-001_AssetCard.tsx', '5e4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-002_DeploymentWizard.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-002_DeploymentWizard.tsx', 'c22'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-003_ActionApproval.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-003_ActionApproval.tsx', 'c7a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-004_KissScorecard.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-004_KissScorecard.tsx', 'aa6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-005_RiskDashboard.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-005_RiskDashboard.tsx', '1a2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-006_BillingChart.tsx',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-DASH/K-PSA-PTL-DASH-006_BillingChart.tsx', 'dbe'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-LIB-001_api_client.ts',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-LIB-001_api_client.ts', 'e22'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-LIB-002_nats_eventsource.ts',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-APP/K-PSA-PTL-LIB-002_nats_eventsource.ts', '9db'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-THEME/K-PSA-PTL-THEME-001_tenant_branding.css',
                component: ComponentCreator('/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/K-PSA-PTL-THEME/K-PSA-PTL-THEME-001_tenant_branding.css', '6e1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-001_sigma_compiler.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-001_sigma_compiler.go', '051'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-002_sigma_sync.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-002_sigma_sync.py', '3d1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-003_mitre_mapper.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-003_mitre_mapper.py', '103'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-004_yara_integration',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-004_yara_integration', 'fc1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-005_suricata_rules',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-005_suricata_rules', '732'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-006_custom_detections',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-006_custom_detections', '439'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-007_sigma_rust_eval',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-007_sigma_rust_eval', '5d9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-008_tetragon_k8s_ebpf',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-008_tetragon_k8s_ebpf', '84a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-009_zeek_subprocess',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION/K-SOC-DET-009_zeek_subprocess', '0db'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-FR_FORENSICS/K-SOC-FR-001_evidence_capture.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-FR_FORENSICS/K-SOC-FR-001_evidence_capture.go', '877'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-FR_FORENSICS/K-SOC-FR-002_blake3_evidence.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-FR_FORENSICS/K-SOC-FR-002_blake3_evidence.go', '5ec'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-001_bloodhound_analysis.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-001_bloodhound_analysis.go', '0fe'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-002_neo4j_graph.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-002_neo4j_graph.go', 'f0b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-003_cypher_ad_paths',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-003_cypher_ad_paths', '55b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-004_azure_oauth_queries',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-ID_IDENTITY/K-SOC-ID-004_azure_oauth_queries', 'a42'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-001_redis_state.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-001_redis_state.go', 'ca1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-002_graph_correlation.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-002_graph_correlation.py', 'e91'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-003_incident_stitching',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-003_incident_stitching', '72f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-004_forensic_chain',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/K-SOC-IS-004_forensic_chain', 'bcd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-001_otx_puller.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-001_otx_puller.py', '838'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-002_abuseipdb.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-002_abuseipdb.py', '6bc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-003_malware_bazaar.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-003_malware_bazaar.py', '412'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-004_phishing_tank.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-004_phishing_tank.py', '551'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-005_hibp',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-005_hibp', 'cb3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-006_cisa_kev',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-006_cisa_kev', '58c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-007_stix2_parser.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-007_stix2_parser.py', '7e3'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-008_stix2_validator.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-008_stix2_validator.py', '68c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-009_shodan_enrich.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-009_shodan_enrich.py', '558'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-010_censys_discovery.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-010_censys_discovery.py', 'bd8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-011_greynoise_filter.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-011_greynoise_filter.py', '57c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-012_wiz_cloud_ti.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-012_wiz_cloud_ti.py', 'e17'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-013_misp_pymisp_client.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-013_misp_pymisp_client.py', '646'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-014_opencti_connector',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-014_opencti_connector', '2e7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-015_ipsum_blocklist.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/K-SOC-TI-015_ipsum_blocklist.py', '30f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-001_nuclei_engine.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-001_nuclei_engine.go', 'eb9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-002_epss_worker.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-002_epss_worker.py', 'a10'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-003_cve_priority.sql',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-003_cve_priority.sql', '824'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-004_trivy_scanner.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-004_trivy_scanner.go', 'ba0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-005_grype_db.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-005_grype_db.go', '376'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-006_syft_sbom_gen.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-006_syft_sbom_gen.go', 'e8b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-007_openvas_rest',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-007_openvas_rest', 'c1b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-008_checkov_iac.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-008_checkov_iac.py', '92a'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-009_kics_engine.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-009_kics_engine.go', 'a27'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-010_ssvc_decision_tree.py',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-010_ssvc_decision_tree.py', 'fe8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-011_nvd_api_puller.go',
                component: ComponentCreator('/docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/K-SOC-VULN-011_nvd_api_puller.go', '5fa'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-000_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-000_INDEX', 'b2e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-001_INDEX', 'cec'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-002_windows_cypher',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-002_windows_cypher', '027'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-003_azure_cypher',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-BLOODHOUND/K-VENDOR-BH-003_azure_cypher', 'c5c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-001_INDEX', '1ce'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-002_analyzers',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-002_analyzers', '259'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-003_responders',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-003_responders', 'cec'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-004_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-CORTEX/K-VENDOR-COR-004_license_boundary', '5c6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-001_INDEX', 'a9f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-002_falco_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-002_falco_rules', '955'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-003_k8s_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-FALCO/K-VENDOR-FAL-003_k8s_rules', '506'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-001_INDEX', 'fdc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-002_taxonomies',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-002_taxonomies', '0f4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-003_galaxies',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-003_galaxies', '51c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-004_warninglists',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-004_warninglists', '6eb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-005_objects',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-005_objects', '3f5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-006_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MISP/K-VENDOR-MISP-006_sync_script.sh', 'fa9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-001_INDEX', '6e0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-002_enterprise_attack',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-002_enterprise_attack', 'ebe'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-003_cwe_stix2',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-003_cwe_stix2', '08e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-004_capec_stix2',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-004_capec_stix2', '610'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-005_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-MITRE/K-VENDOR-MIT-005_sync_script.sh', '4ab'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-001_INDEX', '3d4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-002_cve_templates',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-002_cve_templates', 'c65'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-003_cloud_templates',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-003_cloud_templates', '927'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-004_http_api_templates',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-004_http_api_templates', '505'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-005_saas_templates',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-005_saas_templates', '5a4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-006_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-NUCLEI/K-VENDOR-NUC-006_sync_script.sh', 'fc6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-001_INDEX', 'd2f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-002_cis_benchmarks',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-002_cis_benchmarks', '417'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-003_stig_content',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OPENSCAP/K-VENDOR-OSP-003_stig_content', 'dec'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-001_INDEX', '791'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-002_nist_800_53',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-002_nist_800_53', 'f9d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-003_pci_dss',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-003_pci_dss', '3bf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-004_iso_27001',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-004_iso_27001', 'c07'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-005_soc2',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSCAL/K-VENDOR-OSC-005_soc2', '3f5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-001_INDEX', '1ac'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-002_incident_response',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-002_incident_response', '88c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-003_fim_packs',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-003_fim_packs', 'e33'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-004_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-OSQUERY/K-VENDOR-OSQ-004_sync_script.sh', '85b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-001_INDEX', '415'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-002_techniques',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-002_techniques', '156'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-003_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-RUD-003_license_boundary', 'acb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-001_INDEX', '648'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-002_soar_workflows',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-002_soar_workflows', 'd1c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-003_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SHUFFLE/K-VENDOR-SHF-003_license_boundary', '825'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-001_INDEX', '87f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-002_windows_builtin',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-002_windows_builtin', 'aa8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-003_cloud_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-003_cloud_rules', '877'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-004_saas_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-004_saas_rules', 'd6f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-005_hunting_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-005_hunting_rules', '9e2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-006_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SIGMA/K-VENDOR-SIG-006_sync_script.sh', '309'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-001_INDEX', '189'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-002_emerging_malware',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-002_emerging_malware', '77e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-003_emerging_c2',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-003_emerging_c2', 'c63'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-004_emerging_web',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-004_emerging_web', '39d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-005_emerging_data',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-005_emerging_data', '45c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-006_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-SURICATA/K-VENDOR-SUR-006_sync_script.sh', '3e4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-001_INDEX', '7fb'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-002_case_schema',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-002_case_schema', 'af5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-003_alert_schema',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-003_alert_schema', 'b8f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-004_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-THEHIVE/K-VENDOR-THV-004_license_boundary', '392'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-001_INDEX', '2ff'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-002_threat_hunting',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-002_threat_hunting', 'b5e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-003_forensic_artifacts',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-003_forensic_artifacts', 'eb2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-004_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-VELOCIRAPTOR/K-VENDOR-VEL-004_license_boundary', 'b66'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-001_INDEX', 'ec0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-002_process_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-002_process_rules', '969'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-003_ad_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-003_ad_rules', 'a75'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-004_sca_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-004_sca_rules', 'c6b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-005_license_boundary',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-WAZUH/K-VENDOR-WAZ-005_license_boundary', '6b8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-001_INDEX', '4fc'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-002_malware_sigs',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-002_malware_sigs', '6dd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-003_pii_rules',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-003_pii_rules', '79e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-004_sync_script.sh',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-YARA/K-VENDOR-YAR-004_sync_script.sh', 'ef6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-001_INDEX',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-001_INDEX', 'ce7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-002_base_protocols',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-002_base_protocols', '4e0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-003_intel_framework',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-003_intel_framework', 'f79'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-004_http_scripts',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-004_http_scripts', '4e8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-005_ja3_tls',
                component: ComponentCreator('/docs/K-VENDOR-00_DETECTION_ASSETS/K-VENDOR-ZEEK/K-VENDOR-ZEK-005_ja3_tls', '7b8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-001_Cargo.toml',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-001_Cargo.toml', '334'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-002_eBPF_Compatibility',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-002_eBPF_Compatibility', '9b0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-001_main.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-001_main.rs', '01c'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-001_execve_hook.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-001_execve_hook.rs', '03d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-002_openat2_hook.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-002_openat2_hook.rs', '8c7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-003_map_pressure.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-EBPF/K-XRO-CS-EBPF-003_map_pressure.rs', 'ec2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-001_inotify_watcher.rs', 'b57'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-002_blake3_baseline.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FIM/K-XRO-CS-FIM-002_blake3_baseline.rs', 'f56'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FORENSIC/K-XRO-CS-FR-001_memory_snapshot.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-FORENSIC/K-XRO-CS-FR-001_memory_snapshot.rs', 'f5f'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-GOVERNOR/K-XRO-CS-GV-001_token_bucket.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-GOVERNOR/K-XRO-CS-GV-001_token_bucket.rs', 'da8'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-ML/K-XRO-CS-ML-001_candle_inference.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-ML/K-XRO-CS-ML-001_candle_inference.rs', '2b0'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-ML/K-XRO-CS-ML-002_tinyllama_loader.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-ML/K-XRO-CS-ML-002_tinyllama_loader.rs', '255'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-001_sigma_evaluator.rs', '950'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-002_ocsf_event_bridge.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-SIGMA/K-XRO-CS-SIG-002_ocsf_event_bridge.rs', 'd76'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-001_yara_compiler.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-001_yara_compiler.rs', 'f72'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-002_malware_scanner.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/K-XRO-CS-SRC/K-XRO-CS-YARA/K-XRO-CS-YAR-002_malware_scanner.rs', 'ca5'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-001_Cargo.toml',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-001_Cargo.toml', '257'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-002_10G_Validation',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-002_10G_Validation', '497'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-001_main.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-001_main.rs', 'ef1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-001_ndpi_ffi.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-001_ndpi_ffi.rs', '44e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-002_l7_classifier.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-DPI/K-XRO-NG-DPI-002_l7_classifier.rs', 'acf'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-001_rule_loader.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-001_rule_loader.rs', '3a1'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-002_alert_publisher.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-IDS/K-XRO-NG-IDS-002_alert_publisher.rs', '061'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-001_flow_analyzer.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-001_flow_analyzer.rs', 'd09'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-002_tls_sni.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-002_tls_sni.rs', '3e6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-003_af_packet_ring.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-003_af_packet_ring.rs', 'fb2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-004_dpdk_bypass.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-PCAP/K-XRO-NG-PCAP-004_dpdk_bypass.rs', '92b'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-001_beacon_detector.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-001_beacon_detector.go', '284'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-002_dns_tunnel.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-002_dns_tunnel.go', '4a4'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-003_exfil_detector.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-RITA/K-XRO-NG-RITA-003_exfil_detector.go', '7be'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-TI/K-XRO-NG-TI-001_ipsum_lookup.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/K-XRO-NG-SRC/K-XRO-NG-TI/K-XRO-NG-TI-001_ipsum_lookup.rs', 'aa7'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-001_Cargo.toml',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-001_Cargo.toml', '46e'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-006_Baseline_Schema.json',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-006_Baseline_Schema.json', '4cd'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-001_main.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-001_main.rs', 'bb9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-002_perf_event_open.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-002_perf_event_open.rs', '1a6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-003_prometheus.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-003_prometheus.rs', '576'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-004_otel_collector.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-004_otel_collector.rs', '2ba'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-005_sysinfo_host_metrics.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/K-XRO-PT-SRC/K-XRO-PT-005_sysinfo_host_metrics.rs', 'e50'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-001_registration_handler.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-001_registration_handler.go', '5de'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-002_install_script_gen.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-002_install_script_gen.go', '1e9'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-003_blake3_fingerprinter.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/K-XRO-PV-003_blake3_fingerprinter.go', 'a46'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-SD_SIDECARS/K-XRO-SD-001_rustdesk_remote',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-SD_SIDECARS/K-XRO-SD-001_rustdesk_remote', '033'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-SD_SIDECARS/K-XRO-SD-002_tetragon_ebpf',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-SD_SIDECARS/K-XRO-SD-002_tetragon_ebpf', '56d'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-001_agent_orchestrator.rs', 'cc6'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-002_zstd_delta.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-002_zstd_delta.go', 'f13'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-003_manifest_signer.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-003_manifest_signer.go', '6c2'),
                exact: true,
                sidebar: "docs"
              },
              {
                path: '/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-004_go_tuf_updater.go',
                component: ComponentCreator('/docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/K-XRO-WD-004_go_tuf_updater.go', '583'),
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
