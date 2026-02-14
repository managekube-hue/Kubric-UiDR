/**
 * Kubric Enterprise Documentation Sidebar
 * Organized by operational domains in logical order
 */

// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  docs: [
    'intro',
    {
      type: 'category',
      label: 'PLATFORM',
      collapsible: true,
      collapsed: false,
      items: [
        {
          type: 'category',
          label: 'Core Infrastructure',
          items: [
            'K-CORE-01_INFRASTRUCTURE/index',
            'K-CORE-01_INFRASTRUCTURE/K-HW-R740_HARDWARE/index',
            'K-CORE-01_INFRASTRUCTURE/K-NET-NETWORKING/index',
            'K-CORE-01_INFRASTRUCTURE/K-HV-PROXMOX_HYPERVISOR/index',
            'K-CORE-01_INFRASTRUCTURE/K-K8S-KUBERNETES/index',
            'K-CORE-01_INFRASTRUCTURE/K-DL-DATA_LAKEHOUSE/index',
            'K-CORE-01_INFRASTRUCTURE/K-MB-MESSAGE_BUS/index',
            'K-CORE-01_INFRASTRUCTURE/K-SEC-SECURITY_ROOT/index',
          ],
        },
        {
          type: 'category',
          label: 'XRO Super Agent',
          items: [
            'K-XRO-02_SUPER_AGENT/index',
            'K-XRO-02_SUPER_AGENT/K-XRO-CS_CORESEC/index',
            'K-XRO-02_SUPER_AGENT/K-XRO-NG_NETGUARD/index',
            'K-XRO-02_SUPER_AGENT/K-XRO-PT_PERFTRACE/index',
            'K-XRO-02_SUPER_AGENT/K-XRO-WD_WATCHDOG/index',
            'K-XRO-02_SUPER_AGENT/K-XRO-PV_PROVISIONING/index',
          ],
        },
        {
          type: 'category',
          label: 'KAI Orchestration',
          items: [
            'K-KAI-03_ORCHESTRATION/index',
            'K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS/index',
            'K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW/index',
            'K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS/index',
            'K-KAI-03_ORCHESTRATION/K-KAI-RAG/index',
            'K-KAI-03_ORCHESTRATION/K-KAI-AUDIT/index',
          ],
        },
      ],
    },
    {
      type: 'category',
      label: 'SECURITY OPERATIONS',
      items: [
        'K-SOC-04_SECURITY/index',
        'K-SOC-04_SECURITY/K-SOC-DET_DETECTION/index',
        'K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL/index',
        'K-SOC-04_SECURITY/K-SOC-VULN_VULNERABILITY/index',
        'K-SOC-04_SECURITY/K-SOC-IS_INCIDENT_STITCH/index',
        'K-SOC-04_SECURITY/K-SOC-FR_FORENSICS/index',
      ],
    },
    {
      type: 'category',
      label: 'OPERATIONS & COMPLIANCE',
      items: [
        'K-NOC-05_OPERATIONS/index',
        'K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT/index',
        'K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR/index',
        'K-NOC-05_OPERATIONS/K-NOC-PM_PERFORMANCE/index',
        'K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT/index',
        'K-GRC-07_COMPLIANCE/index',
        'K-GRC-07_COMPLIANCE/K-GRC-OSCAL/index',
        'K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT/index',
        'K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN/index',
      ],
    },
    {
      type: 'category',
      label: 'BUSINESS CAPABILITIES',
      items: [
        'K-PSA-06_BUSINESS/index',
        'K-PSA-06_BUSINESS/K-PSA-ITSM/index',
        'K-PSA-06_BUSINESS/K-PSA-BILL_BILLING/index',
        'K-PSA-06_BUSINESS/K-PSA-CRM_CPQ/index',
        'K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL/index',
        'K-PSA-06_BUSINESS/K-PSA-BI_BUSINESS_INTEL/index',
      ],
    },
    {
      type: 'category',
      label: 'DEVELOPMENT & APIS',
      items: [
        'K-DEV-08_DEVELOPMENT/index',
        'K-DEV-08_DEVELOPMENT/K-DEV-LOCAL_LOCAL_STACK/index',
        'K-DEV-08_DEVELOPMENT/K-DEV-BLD_BUILD_TOOLCHAIN/index',
        'K-DEV-08_DEVELOPMENT/K-DEV-CICD/index',
        'K-DEV-08_DEVELOPMENT/K-DEV-GIT_GITOPS/index',
        'K-API-09_API_REFERENCE/index',
      ],
    },
    {
      type: 'category',
      label: 'ITIL FRAMEWORK',
      items: [
        'K-ITIL-10_ITIL_MATRIX/index',
        'K-ITIL-10_ITIL_MATRIX/K-ITIL-02_GMP_MAP/index',
        'K-ITIL-10_ITIL_MATRIX/K-ITIL-03_SMP_MAP/index',
        'K-ITIL-10_ITIL_MATRIX/K-ITIL-04_TMP_MAP/index',
        'K-ITIL-10_ITIL_MATRIX/K-ITIL-05_AUDIT_READINESS/index',
      ],
    },
  ],
};

module.exports = sidebars;
