/**
 * Kubric Portal API client — typed wrappers around the Go REST services.
 * Base URLs are set from NEXT_PUBLIC_* env vars injected at build time.
 */

const API_BASE = process.env.NEXT_PUBLIC_API_BASE ?? "http://localhost:8080";
const VDR_BASE = process.env.NEXT_PUBLIC_VDR_URL ?? "http://localhost:8081";
const KIC_BASE = process.env.NEXT_PUBLIC_KIC_URL ?? "http://localhost:8082";
const NOC_BASE = process.env.NEXT_PUBLIC_NOC_URL ?? "http://localhost:8083";
const KAI_BASE = process.env.NEXT_PUBLIC_KAI_URL ?? "http://localhost:8100";

// ─── Generic fetch helper ──────────────────────────────────────────────────

async function apiFetch<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    headers: { "Content-Type": "application/json", ...init?.headers },
    ...init,
  });
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(`API ${res.status}: ${text}`);
  }
  return res.json() as Promise<T>;
}

// ─── Tenant service (K-SVC) ───────────────────────────────────────────────

export interface Tenant {
  tenant_id: string;
  name: string;
  plan: string;
  created_at: string;
}

export const tenantApi = {
  list: () =>
    apiFetch<{ items: Tenant[] }>(`${API_BASE}/v1/tenants`),
  get: (id: string) =>
    apiFetch<Tenant>(`${API_BASE}/v1/tenants/${id}`),
  create: (payload: Pick<Tenant, "name" | "plan">) =>
    apiFetch<Tenant>(`${API_BASE}/v1/tenants`, {
      method: "POST",
      body: JSON.stringify(payload),
    }),
};

// ─── VDR — vulnerability findings ─────────────────────────────────────────

export interface Finding {
  id: string;
  tenant_id: string;
  cve_id: string;
  severity: "critical" | "high" | "medium" | "low" | "info";
  status: "open" | "remediated" | "accepted" | "false_positive";
  asset: string;
  title: string;
  description: string;
  cvss_score: number;
  epss_score: number;
  created_at: string;
  updated_at: string;
}

export const vdrApi = {
  list: (tenantId: string, status?: string) => {
    const params = new URLSearchParams({ tenant_id: tenantId });
    if (status) params.set("status", status);
    return apiFetch<{ items: Finding[]; total: number }>(
      `${VDR_BASE}/v1/vdr/findings?${params}`
    );
  },
  get: (tenantId: string, id: string) =>
    apiFetch<Finding>(`${VDR_BASE}/v1/vdr/findings/${id}`, {
      headers: { "X-Tenant-ID": tenantId },
    }),
};

// ─── KIC — compliance assessments ─────────────────────────────────────────

export interface Assessment {
  id: string;
  tenant_id: string;
  framework: string;
  pass_rate: number;
  total_controls: number;
  passed: number;
  failed: number;
  created_at: string;
}

export const kicApi = {
  list: (tenantId: string) =>
    apiFetch<{ items: Assessment[] }>(
      `${KIC_BASE}/v1/kic/assessments?tenant_id=${tenantId}`
    ),
};

// ─── NOC — cluster + agent health ─────────────────────────────────────────

export interface Agent {
  id: string;
  tenant_id: string;
  hostname: string;
  agent_type: string;
  status: "online" | "offline" | "degraded";
  last_heartbeat: string;
}

export const nocApi = {
  agents: (tenantId: string) =>
    apiFetch<{ items: Agent[] }>(
      `${NOC_BASE}/v1/noc/agents?tenant_id=${tenantId}`
    ),
};

// ─── KAI — AI orchestration ────────────────────────────────────────────────

export interface KissScore {
  tenant_id: string;
  overall: number;
  vuln_score: number;
  compliance_score: number;
  detection_score: number;
  response_score: number;
  generated_at: string;
}

export const kaiApi = {
  score: (tenantId: string) =>
    apiFetch<KissScore>(`${KAI_BASE}/score/${tenantId}`),
  insights: (tenantId: string) =>
    apiFetch<{ insights: string[] }>(`${KAI_BASE}/insights/${tenantId}`),
};
