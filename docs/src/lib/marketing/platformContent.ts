export type PlatformPageKey = 'overview' | 'soc' | 'noc' | 'grc' | 'psa' | 'kai';

export interface PlatformFeature {
  title: string;
  description: string;
  href: string;
}

export interface PlatformPageContent {
  key: PlatformPageKey;
  title: string;
  description: string;
  eyebrow: string;
  headline: string;
  subheadline: string;
  icon: string;
  ctaLabel: string;
  ctaHref: string;
  secondaryLabel?: string;
  secondaryHref?: string;
  bandStatement: string;
  moduleWhy: string;
  roadmap: string[];
  features: PlatformFeature[];
}

const pages: Record<PlatformPageKey, PlatformPageContent> = {
  overview: {
    key: 'overview',
    title: 'Platform',
    description: 'Kubric platform overview across SOC, NOC, GRC, PSA, and KAI domains.',
    eyebrow: 'Platform',
    headline: 'The Unified IDR Architecture',
    subheadline:
      'Kubric architecture spans infrastructure, security operations, governance, and autonomous execution.',
    icon: '📋',
    ctaLabel: 'Open Technical Documentation',
    ctaHref: '/docs/intro',
    secondaryLabel: 'Contributors',
    secondaryHref: '/contributors',
    bandStatement: 'Six execution modules, one unified operational fabric for detection, remediation, compliance, and business continuity.',
    moduleWhy: 'This platform layer matters because enterprise teams need one operating model across SOC, NOC, GRC, PSA, and KAI to avoid fragmented controls and duplicated workflows.',
    roadmap: [
      'Define target operating model and ownership for each module.',
      'Map critical workflows to implementation pages and execution runbooks.',
      'Harden controls and telemetry with production-grade reliability gates.',
      'Operationalize governance and continuous improvement through measurable SLIs.',
    ],
    features: [
      {
        title: 'Security Operations (SOC)',
        description: 'Detection, threat intelligence, vulnerability workflows, incident stitching, and forensics.',
        href: '/platform/soc',
      },
      {
        title: 'Network Operations (NOC)',
        description: 'Configuration, backup/DR, patching, and performance operations for resilient infrastructure.',
        href: '/platform/noc',
      },
      {
        title: 'Governance & Compliance (GRC)',
        description: 'OSCAL alignment, evidence workflows, supply chain controls, and audit readiness.',
        href: '/platform/grc',
      },
      {
        title: 'Business Services (PSA)',
        description: 'ITSM, billing, CRM/CPQ, portals, and business intelligence operations.',
        href: '/platform/psa',
      },
      {
        title: 'AI Orchestration (KAI)',
        description: 'Workflow execution, guardrails, RAG, and auditable autonomous operations.',
        href: '/platform/kai',
      },
    ],
  },
  soc: {
    key: 'soc',
    title: 'Platform SOC',
    description: 'SOC capabilities in Kubric Platform.',
    eyebrow: 'Platform / SOC',
    headline: 'Security Operations',
    subheadline: 'Detection engineering, threat intelligence, vulnerability operations, and forensics in one control plane.',
    icon: '🛡️',
    ctaLabel: 'Open SOC Docs',
    ctaHref: '/docs/K-SOC-04_SECURITY',
    secondaryLabel: 'Platform Overview',
    secondaryHref: '/platform',
    bandStatement: 'SOC outcomes accelerate when detection engineering, intelligence, and forensics run on a single evidence-aware control plane.',
    moduleWhy: 'SOC is mission-critical because faster high-confidence detection and triage directly reduce blast radius and recovery time in live incidents.',
    roadmap: [
      'Standardize detection taxonomy and severity across all sources.',
      'Automate enrichment and triage workflows with policy-based routing.',
      'Integrate response playbooks with immutable evidence capture.',
      'Continuously tune detection precision with post-incident feedback loops.',
    ],
    features: [
      {title: 'Detection', description: 'Engineering and tuning for SOC detections.', href: '/docs/K-SOC-04_SECURITY/K-SOC-DET_DETECTION'},
      {title: 'Threat Intel', description: 'Curated threat intelligence pipelines.', href: '/docs/K-SOC-04_SECURITY/K-SOC-TI_THREAT_INTEL'},
      {title: 'Forensics', description: 'Investigation-ready evidence and workflows.', href: '/docs/K-SOC-04_SECURITY/K-SOC-FR_FORENSICS'},
    ],
  },
  noc: {
    key: 'noc',
    title: 'Platform NOC',
    description: 'NOC capabilities in Kubric Platform.',
    eyebrow: 'Platform / NOC',
    headline: 'Network & Operations Control',
    subheadline: 'Operate enterprise infrastructure with deterministic runbooks and observability-first workflows.',
    icon: '📊',
    ctaLabel: 'Open NOC Docs',
    ctaHref: '/docs/K-NOC-05_OPERATIONS',
    secondaryLabel: 'Platform Overview',
    secondaryHref: '/platform',
    bandStatement: 'NOC reliability depends on deterministic operations, resilient recovery, and measurable infrastructure performance.',
    moduleWhy: 'NOC is essential because every security and business workflow depends on healthy, observable, and recoverable infrastructure.',
    roadmap: [
      'Baseline infrastructure SLOs and operational ownership boundaries.',
      'Implement automated drift detection and safe remediation paths.',
      'Codify backup, restore, and patch workflows with approval controls.',
      'Scale observability for proactive performance and failure prevention.',
    ],
    features: [
      {title: 'Config Management', description: 'Versioned config workflows and drift control.', href: '/docs/K-NOC-05_OPERATIONS/K-NOC-CM_CONFIG_MGMT'},
      {title: 'Backup & DR', description: 'Recovery automation and restoration readiness.', href: '/docs/K-NOC-05_OPERATIONS/K-NOC-BR_BACKUP_DR'},
      {title: 'Patch Management', description: 'Patch compliance and staged rollouts.', href: '/docs/K-NOC-05_OPERATIONS/K-NOC-PT_PATCH_MGMT'},
    ],
  },
  grc: {
    key: 'grc',
    title: 'Platform GRC',
    description: 'GRC capabilities in Kubric Platform.',
    eyebrow: 'Platform / GRC',
    headline: 'Governance, Risk, and Compliance',
    subheadline: 'Build controls and evidence into day-to-day execution instead of bolt-on audit preparation.',
    icon: '⚖️',
    ctaLabel: 'Open GRC Docs',
    ctaHref: '/docs/K-GRC-07_COMPLIANCE',
    secondaryLabel: 'Platform Overview',
    secondaryHref: '/platform',
    bandStatement: 'GRC is strongest when control intent, execution evidence, and audit narratives are generated from the same operational workflows.',
    moduleWhy: 'GRC matters because compliance must be continuously provable, not periodically reconstructed from disconnected systems.',
    roadmap: [
      'Map frameworks to operational controls and implementation artifacts.',
      'Automate evidence lifecycle collection with retention policies.',
      'Introduce continuous validation checks for control effectiveness.',
      'Establish audit-ready reporting pipelines tied to live system state.',
    ],
    features: [
      {title: 'OSCAL', description: 'Control frameworks and machine-readable mappings.', href: '/docs/K-GRC-07_COMPLIANCE/K-GRC-OSCAL'},
      {title: 'Evidence Vault', description: 'Lifecycle-managed compliance artifacts.', href: '/docs/K-GRC-07_COMPLIANCE/K-GRC-EV_EVIDENCE_VAULT'},
      {title: 'Supply Chain', description: 'Software and infrastructure supply-chain controls.', href: '/docs/K-GRC-07_COMPLIANCE/K-GRC-SCS_SUPPLY_CHAIN'},
    ],
  },
  psa: {
    key: 'psa',
    title: 'Platform PSA',
    description: 'PSA capabilities in Kubric Platform.',
    eyebrow: 'Platform / PSA',
    headline: 'Business Service Operations',
    subheadline: 'Unify service delivery and revenue workflows with technical operations and audit context.',
    icon: '💰',
    ctaLabel: 'Open PSA Docs',
    ctaHref: '/docs/K-PSA-06_BUSINESS',
    secondaryLabel: 'Platform Overview',
    secondaryHref: '/platform',
    bandStatement: 'PSA aligns service delivery, commercial workflows, and operational telemetry into one accountable business execution layer.',
    moduleWhy: 'PSA is important because business reliability and customer trust require service and billing flows to remain auditable and technically grounded.',
    roadmap: [
      'Consolidate service, billing, and portal data contracts.',
      'Implement workflow automation for quote-to-cash and service lifecycle.',
      'Embed compliance checkpoints into customer-facing operations.',
      'Deliver executive reporting for margin, SLA health, and customer outcomes.',
    ],
    features: [
      {title: 'ITSM', description: 'Service operations and request workflows.', href: '/docs/K-PSA-06_BUSINESS/K-PSA-ITSM'},
      {title: 'Billing', description: 'Revenue operations and invoicing controls.', href: '/docs/K-PSA-06_BUSINESS/K-PSA-BILL_BILLING'},
      {title: 'Portal', description: 'Customer-facing service and reporting access.', href: '/docs/K-PSA-06_BUSINESS/K-PSA-PTL_PORTAL'},
    ],
  },
  kai: {
    key: 'kai',
    title: 'Platform KAI',
    description: 'KAI orchestration capabilities in Kubric Platform.',
    eyebrow: 'Platform / KAI',
    headline: 'AI Orchestration',
    subheadline: 'Coordinate autonomous agents with guardrails, retrieval pipelines, and auditable execution.',
    icon: '🧠',
    ctaLabel: 'Open KAI Docs',
    ctaHref: '/docs/K-KAI-03_ORCHESTRATION',
    secondaryLabel: 'Platform Overview',
    secondaryHref: '/platform',
    bandStatement: 'KAI orchestrates autonomous workflows with guardrails, transparent reasoning, and policy-aligned execution trails.',
    moduleWhy: 'KAI is critical because AI-enabled operations must remain controllable, explainable, and aligned with production safety requirements.',
    roadmap: [
      'Define role-scoped personas and tool access boundaries.',
      'Harden workflow orchestration with retries, approvals, and checkpoints.',
      'Integrate retrieval and context pipelines with provenance tracing.',
      'Operationalize governance metrics for autonomy quality and safety.',
    ],
    features: [
      {title: 'CrewAI Personas', description: 'Role-scoped autonomous personas for operations.', href: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-CP_CREWAI_PERSONAS'},
      {title: 'Workflow Engine', description: 'Deterministic execution and routing.', href: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-WF_WORKFLOW'},
      {title: 'Guardrails', description: 'Safety checks and policy enforcement.', href: '/docs/K-KAI-03_ORCHESTRATION/K-KAI-GD_GUARDRAILS'},
    ],
  },
};

export function platformPageByKey(key: PlatformPageKey): PlatformPageContent {
  return pages[key];
}
