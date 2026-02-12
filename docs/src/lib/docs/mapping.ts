export interface DocMapping {
  slug: string;
  platformPath: string;
  notionId: string;
  moduleCode: string;
  moduleName: string;
  githubPath: string;
}

export const documentationMap: DocMapping[] = [
  { slug: 'K-CORE-01_INFRASTRUCTURE/K-HW-R740_HARDWARE/index', platformPath: '/platform/core/hardware', notionId: '1a2b3c4d5e6f7g8h9i0j', moduleCode: 'K-HW-R740', moduleName: 'Hardware Cluster', githubPath: 'docs/K-CORE-01_INFRASTRUCTURE/K-HW-R740_HARDWARE' },
  { slug: 'K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/index', platformPath: '/platform/core/network', notionId: '2b3c4d5e6f7g8h9i0j1a', moduleCode: 'K-NET', moduleName: 'Networking Layer', githubPath: 'docs/K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING' },
  { slug: 'K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/index', platformPath: '/platform/core/hypervisor', notionId: '3c4d5e6f7g8h9i0j1a2b', moduleCode: 'K-HV', moduleName: 'Proxmox Hypervisor', githubPath: 'docs/K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR' },
  { slug: 'K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/index', platformPath: '/platform/core/kubernetes', notionId: '4d5e6f7g8h9i0j1a2b3c', moduleCode: 'K-K8S', moduleName: 'Kubernetes', githubPath: 'docs/K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES' },
  { slug: 'K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/index', platformPath: '/platform/core/data', notionId: '5e6f7g8h9i0j1a2b3c4d', moduleCode: 'K-DL', moduleName: 'Data Lakehouse', githubPath: 'docs/K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE' },
  { slug: 'K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/index', platformPath: '/platform/core/messaging', notionId: '6f7g8h9i0j1a2b3c4d5e', moduleCode: 'K-MB', moduleName: 'Message Bus', githubPath: 'docs/K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS' },
  { slug: 'K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/index', platformPath: '/platform/core/security', notionId: '7g8h9i0j1a2b3c4d5e6f', moduleCode: 'K-SEC', moduleName: 'Security Root', githubPath: 'docs/K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT' },

  { slug: 'K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/index', platformPath: '/platform/xro/coresec', notionId: '8h9i0j1a2b3c4d5e6f7g', moduleCode: 'K-XRO-CS', moduleName: 'CoreSec eBPF Agent', githubPath: 'docs/K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC' },
  { slug: 'K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/index', platformPath: '/platform/xro/netguard', notionId: '9i0j1a2b3c4d5e6f7g8h', moduleCode: 'K-XRO-NG', moduleName: 'NetGuard NDR Agent', githubPath: 'docs/K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD' },
  { slug: 'K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/index', platformPath: '/platform/xro/perftrace', notionId: '0j1a2b3c4d5e6f7g8h9i', moduleCode: 'K-XRO-PT', moduleName: 'PerfTrace Agent', githubPath: 'docs/K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE' },
  { slug: 'K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/index', platformPath: '/platform/xro/watchdog', notionId: '1a2b3c4d5e6f7g8h9i0j', moduleCode: 'K-XRO-WD', moduleName: 'Watchdog Orchestrator', githubPath: 'docs/K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG' },
  { slug: 'K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/index', platformPath: '/platform/xro/provisioning', notionId: '2b3c4d5e6f7g8h9i0j1a', moduleCode: 'K-XRO-PV', moduleName: 'Provisioning', githubPath: 'docs/K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING' },

  { slug: 'K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/index', platformPath: '/platform/kai/personas', notionId: '3c4d5e6f7g8h9i0j1a2b', moduleCode: 'K-KAI-CP', moduleName: 'CrewAI Personas', githubPath: 'docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS' },
  { slug: 'K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/index', platformPath: '/platform/kai/workflow', notionId: '4d5e6f7g8h9i0j1a2b3c', moduleCode: 'K-KAI-WF', moduleName: 'Workflow Engine', githubPath: 'docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW' },
  { slug: 'K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/index', platformPath: '/platform/kai/guardrails', notionId: '5e6f7g8h9i0j1a2b3c4d', moduleCode: 'K-KAI-GD', moduleName: 'Guardrails & Safety', githubPath: 'docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS' },
  { slug: 'K-KAI-03_ORCHESTRATION/K-KAI-RAG/index', platformPath: '/platform/kai/rag', notionId: '6f7g8h9i0j1a2b3c4d5e', moduleCode: 'K-KAI-RAG', moduleName: 'RAG Engine', githubPath: 'docs/K-KAI-03_ORCHESTRATION/K-KAI-RAG' },
  { slug: 'K-KAI-03_ORCHESTRATION/K-KAI-AUDIT/index', platformPath: '/platform/kai/audit', notionId: '7g8h9i0j1a2b3c4d5e6f', moduleCode: 'K-KAI-AUDIT', moduleName: 'Audit Trail', githubPath: 'docs/K-KAI-03_ORCHESTRATION/K-KAI-AUDIT' },

  { slug: 'K-SOC-04_SECURITY/K-SOC-DET_DETECTION/index', platformPath: '/platform/soc/detection', notionId: '8h9i0j1a2b3c4d5e6f7g', moduleCode: 'K-SOC-DET', moduleName: 'Detection Engine', githubPath: 'docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION' },
  { slug: 'K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/index', platformPath: '/platform/soc/threat-intel', notionId: '9i0j1a2b3c4d5e6f7g8h', moduleCode: 'K-SOC-TI', moduleName: 'Threat Intelligence', githubPath: 'docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL' },
  { slug: 'K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/index', platformPath: '/platform/soc/vulnerability', notionId: '0j1a2b3c4d5e6f7g8h9i', moduleCode: 'K-SOC-VULN', moduleName: 'Vulnerability Management', githubPath: 'docs/K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY' },
  { slug: 'K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/index', platformPath: '/platform/soc/incident-stitching', notionId: '1a2b3c4d5e6f7g8h9i0j', moduleCode: 'K-SOC-IS', moduleName: 'Incident Stitching', githubPath: 'docs/K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH' },
  { slug: 'K-SOC-04_SECURITY/K-SOC-FR_FORENSICS/index', platformPath: '/platform/soc/forensics', notionId: '2b3c4d5e6f7g8h9i0j1a', moduleCode: 'K-SOC-FR', moduleName: 'Forensics', githubPath: 'docs/K-SOC-04_SECURITY/K-SOC-FR_FORENSICS' },

  { slug: 'K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/index', platformPath: '/platform/noc/config-mgmt', notionId: '3c4d5e6f7g8h9i0j1a2b', moduleCode: 'K-NOC-CM', moduleName: 'Configuration Management', githubPath: 'docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT' },
  { slug: 'K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/index', platformPath: '/platform/noc/backup-dr', notionId: '4d5e6f7g8h9i0j1a2b3c', moduleCode: 'K-NOC-BR', moduleName: 'Backup & DR', githubPath: 'docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR' },
  { slug: 'K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/index', platformPath: '/platform/noc/performance', notionId: '5e6f7g8h9i0j1a2b3c4d', moduleCode: 'K-NOC-PM', moduleName: 'Performance Monitoring', githubPath: 'docs/K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE' },
  { slug: 'K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT/index', platformPath: '/platform/noc/patch-mgmt', notionId: '6f7g8h9i0j1a2b3c4d5e', moduleCode: 'K-NOC-PT', moduleName: 'Patch Management', githubPath: 'docs/K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT' },
  { slug: 'K-GRC-07_COMPLIANCE/K-GRC-OSCAL/index', platformPath: '/platform/grc/oscal', notionId: '7g8h9i0j1a2b3c4d5e6f', moduleCode: 'K-GRC-OSCAL', moduleName: 'OSCAL Frameworks', githubPath: 'docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL' },
  { slug: 'K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/index', platformPath: '/platform/grc/evidence-vault', notionId: '8h9i0j1a2b3c4d5e6f7g', moduleCode: 'K-GRC-EV', moduleName: 'Evidence Vault', githubPath: 'docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT' },
  { slug: 'K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/index', platformPath: '/platform/grc/supply-chain', notionId: '9i0j1a2b3c4d5e6f7g8h', moduleCode: 'K-GRC-SCS', moduleName: 'Supply Chain Security', githubPath: 'docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN' },

  { slug: 'K-PSA-06_BUSINESS/K-PSA-ITSM/index', platformPath: '/platform/psa/itsm', notionId: '0j1a2b3c4d5e6f7g8h9i', moduleCode: 'K-PSA-ITSM', moduleName: 'ITSM Engine', githubPath: 'docs/K-PSA-06_BUSINESS/K-PSA-ITSM' },
  { slug: 'K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/index', platformPath: '/platform/psa/billing', notionId: '1a2b3c4d5e6f7g8h9i0j', moduleCode: 'K-PSA-BILL', moduleName: 'Billing & Invoicing', githubPath: 'docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING' },
  { slug: 'K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/index', platformPath: '/platform/psa/crm-cpq', notionId: '2b3c4d5e6f7g8h9i0j1a', moduleCode: 'K-PSA-CRM', moduleName: 'CRM & CPQ', githubPath: 'docs/K-PSA-06_BUSINESS/K-PSA-CRM_CPQ' },
  { slug: 'K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/index', platformPath: '/platform/psa/portal', notionId: '3c4d5e6f7g8h9i0j1a2b', moduleCode: 'K-PSA-PTL', moduleName: 'Customer Portal', githubPath: 'docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL' },
  { slug: 'K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/index', platformPath: '/platform/psa/bi', notionId: '4d5e6f7g8h9i0j1a2b3c', moduleCode: 'K-PSA-BI', moduleName: 'Business Intelligence', githubPath: 'docs/K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL' },

  { slug: 'K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/index', platformPath: '/platform/dev/local-stack', notionId: '5e6f7g8h9i0j1a2b3c4d', moduleCode: 'K-DEV-LOCAL', moduleName: 'Local Development Stack', githubPath: 'docs/K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK' },
  { slug: 'K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/index', platformPath: '/platform/dev/build', notionId: '6f7g8h9i0j1a2b3c4d5e', moduleCode: 'K-DEV-BLD', moduleName: 'Build Toolchain', githubPath: 'docs/K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN' },
  { slug: 'K-DEV-08_DEVELOPMENT/K-DEV-CICD/index', platformPath: '/platform/dev/cicd', notionId: '7g8h9i0j1a2b3c4d5e6f', moduleCode: 'K-DEV-CICD', moduleName: 'CI/CD', githubPath: 'docs/K-DEV-08_DEVELOPMENT/K-DEV-CICD' },
  { slug: 'K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/index', platformPath: '/platform/dev/gitops', notionId: '8h9i0j1a2b3c4d5e6f7g', moduleCode: 'K-DEV-GIT', moduleName: 'GitOps', githubPath: 'docs/K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS' },
  { slug: 'K-API-09_API_REFERENCE/index', platformPath: '/platform/dev/api', notionId: '9i0j1a2b3c4d5e6f7g8h', moduleCode: 'K-API', moduleName: 'API Reference', githubPath: 'docs/K-API-09_API_REFERENCE' },

  { slug: 'K-ITIL-10_ITIL_MATRIX/K-ITIL-02_GMP_MAP/index', platformPath: '/platform/itil/gmp', notionId: '0j1a2b3c4d5e6f7g8h9i', moduleCode: 'K-ITIL-GMP', moduleName: 'General Management Practices', githubPath: 'docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-02_GMP_MAP' },
  { slug: 'K-ITIL-10_ITIL_MATRIX/K-ITIL-03_SMP_MAP/index', platformPath: '/platform/itil/smp', notionId: '1a2b3c4d5e6f7g8h9i0j', moduleCode: 'K-ITIL-SMP', moduleName: 'Service Management Practices', githubPath: 'docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-03_SMP_MAP' },
  { slug: 'K-ITIL-10_ITIL_MATRIX/K-ITIL-04_TMP_MAP/index', platformPath: '/platform/itil/tmp', notionId: '2b3c4d5e6f7g8h9i0j1a', moduleCode: 'K-ITIL-TMP', moduleName: 'Technical Management Practices', githubPath: 'docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-04_TMP_MAP' },
  { slug: 'K-ITIL-10_ITIL_MATRIX/K-ITIL-05_AUDIT_READINESS/index', platformPath: '/platform/itil/audit', notionId: '3c4d5e6f7g8h9i0j1a2b', moduleCode: 'K-ITIL-AUDIT', moduleName: 'Audit Readiness', githubPath: 'docs/K-ITIL-10_ITIL_MATRIX/K-ITIL-05_AUDIT_READINESS' },
];

function normalizeSlug(value: string): string {
  return value.replace(/^\//, '').replace(/\/index$/, '');
}

export function getDocBySlug(slug: string): DocMapping | undefined {
  const normalized = normalizeSlug(slug);
  return documentationMap.find((doc) => normalizeSlug(doc.slug) === normalized);
}

export function getDocByPlatformPath(path: string): DocMapping | undefined {
  return documentationMap.find((doc) => doc.platformPath === path);
}

export function getDocsByModule(moduleCode: string): DocMapping[] {
  return documentationMap.filter((doc) => doc.moduleCode.startsWith(moduleCode));
}

export function getNotionUrl(notionId: string): string {
  return `https://notion.so/kubric/${notionId}`;
}

export function getGithubUrl(githubPath: string): string {
  return `https://github.com/managekube-hue/Kubric-UiDR/tree/main/${githubPath}`;
}
