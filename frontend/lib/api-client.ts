/**
 * Kubric API client — typed fetch wrapper for Go backend services.
 * All calls include Authorization: Bearer {jwt} from the session.
 * Tenant ID is extracted from JWT — never trusted from URL params.
 */

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";

interface ApiOptions {
  token: string;
  tenantId?: string;
}

async function apiFetch<T>(
  path: string,
  opts: ApiOptions & RequestInit = { token: "" }
): Promise<T> {
  const { token, tenantId, ...fetchOpts } = opts;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${token}`,
    ...(tenantId ? { "X-Kubric-Tenant-Id": tenantId } : {}),
    ...(fetchOpts.headers as Record<string, string>),
  };
  const res = await fetch(`${API_BASE}${path}`, { ...fetchOpts, headers });
  if (!res.ok) {
    const body = await res.text().catch(() => "");
    throw new Error(`API ${res.status}: ${body}`);
  }
  return res.json();
}

// ── K-SVC (Tenants) ────────────────────────────────────────────────────────

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  subscription_tier: string;
  created_at: string;
  updated_at: string;
}

export async function listTenants(opts: ApiOptions): Promise<Tenant[]> {
  return apiFetch("/v1/tenants", { ...opts, method: "GET" });
}

export async function getTenant(id: string, opts: ApiOptions): Promise<Tenant> {
  return apiFetch(`/v1/tenants/${id}`, { ...opts, method: "GET" });
}

// ── K-VDR (Vulnerability Findings) ─────────────────────────────────────────

export interface Finding {
  id: string;
  tenant_id: string;
  cve_id: string;
  title: string;
  severity: string;
  status: string;
  source: string;
  epss_score?: number;
  epss_percentile?: number;
  created_at: string;
  updated_at: string;
}

export async function listFindings(opts: ApiOptions): Promise<Finding[]> {
  return apiFetch("/v1/findings", { ...opts, method: "GET" });
}

export async function createFinding(
  data: Partial<Finding>,
  opts: ApiOptions
): Promise<Finding> {
  return apiFetch("/v1/findings", { ...opts, method: "POST", body: JSON.stringify(data) });
}

// ── K-IC (Compliance Assessments) ──────────────────────────────────────────

export interface Assessment {
  id: string;
  tenant_id: string;
  framework: string;
  status: string;
  score: number;
  findings_count: number;
  created_at: string;
  updated_at: string;
}

export async function listAssessments(opts: ApiOptions): Promise<Assessment[]> {
  return apiFetch("/v1/assessments", { ...opts, method: "GET" });
}

// ── K-NOC (Agents) ─────────────────────────────────────────────────────────

export interface Agent {
  id: string;
  tenant_id: string;
  hostname: string;
  agent_type: string;
  version: string;
  active: boolean;
  last_seen_at: string;
}

export async function listAgents(opts: ApiOptions): Promise<Agent[]> {
  return apiFetch("/v1/agents", { ...opts, method: "GET" });
}

// ── Billing ────────────────────────────────────────────────────────────────

export interface BillingUsage {
  tenant_id: string;
  period: string;
  events_count: number;
  agents_count: number;
  total_amount: number;
}

export async function getBillingUsage(opts: ApiOptions): Promise<BillingUsage> {
  return apiFetch(`/v1/billing/usage/${opts.tenantId}`, { ...opts, method: "GET" });
}

export async function getBillingPortalUrl(opts: ApiOptions): Promise<{ url: string }> {
  return apiFetch("/v1/billing/portal", { ...opts, method: "POST" });
}

// ── KiSS Scores ────────────────────────────────────────────────────────────

export interface KissScores {
  tenant_id: string;
  identity: number;
  endpoint: number;
  network: number;
  cloud: number;
  compliance: number;
  overall: number;
  last_updated: string;
}

export async function getKissScores(opts: ApiOptions): Promise<KissScores> {
  return apiFetch(`/v1/kiss/${opts.tenantId}`, { ...opts, method: "GET" });
}

// ── CISO-Assistant (GRC AI Queries) ────────────────────────────────────────

export interface CISOQuestion {
  question: string;
  frameworks?: string[];
  include_refs?: boolean;
}

export interface CISOAnswer {
  answer: string;
  sources: string[];
  confidence: number;
  frameworks: string[];
  posture?: PostureSummary;
  retrieved_docs?: string[];
  responded_at: string;
}

export interface PostureSummary {
  tenant_id: string;
  overall_score: number;
  by_framework: FrameworkPosture[];
  assessed_at: string;
}

export interface FrameworkPosture {
  framework: string;
  score: number;
  total_controls: number;
  compliant_count: number;
  non_compliant_count: number;
  partial_count: number;
  last_assessed_at: string;
}

export interface ComplianceFramework {
  id: string;
  name: string;
  category: string;
}

export async function askCISO(
  question: CISOQuestion,
  opts: ApiOptions
): Promise<CISOAnswer> {
  return apiFetch("/v1/ciso/ask", {
    ...opts,
    method: "POST",
    body: JSON.stringify(question),
  });
}

export async function listComplianceFrameworks(
  opts: ApiOptions
): Promise<{ frameworks: ComplianceFramework[] }> {
  return apiFetch("/v1/ciso/frameworks", { ...opts, method: "GET" });
}

export async function getCompliancePosture(
  opts: ApiOptions
): Promise<PostureSummary> {
  return apiFetch("/v1/ciso/posture", { ...opts, method: "GET" });
}
