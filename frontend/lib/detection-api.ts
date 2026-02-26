/**
 * Kubric Detection & Response API client.
 * Typed fetch wrappers for detection, incidents, integrations, and agent endpoints.
 * Extends the base api-client.ts patterns — always pass { token, tenantId } from useSession.
 */

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8080";

// ── Shared fetch helper (mirrors base api-client.ts) ──────────────────────────

export interface ApiOptions {
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
    throw new ApiError(res.status, `API ${res.status}: ${body}`, path);
  }
  return res.json();
}

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
    public readonly path: string
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// ── Detection — Alerts ────────────────────────────────────────────────────────

export type AlertSeverity = 1 | 2 | 3 | 4 | 5;
export type AlertSource = "coresec" | "netguard" | "wazuh" | "falco" | "suricata";

export interface DetectionAlert {
  id: string;
  tenant_id: string;
  source: AlertSource;
  event_type: string;
  severity: AlertSeverity;
  title: string;
  description: string;
  agent_id: string;
  hostname: string;
  timestamp: string;
  mitre_tactic?: string;
  mitre_technique?: string;
  indicators?: string[];
  reviewed?: boolean;
}

export interface AlertTimelineParams {
  since?: string;       // ISO 8601
  until?: string;
  severity?: AlertSeverity;
  source?: AlertSource;
  limit?: number;
  offset?: number;
}

export async function getAlertTimeline(
  params: AlertTimelineParams,
  opts: ApiOptions
): Promise<DetectionAlert[]> {
  const qs = new URLSearchParams();
  if (params.since) qs.set("since", params.since);
  if (params.until) qs.set("until", params.until);
  if (params.severity) qs.set("severity", String(params.severity));
  if (params.source) qs.set("source", params.source);
  if (params.limit) qs.set("limit", String(params.limit));
  if (params.offset) qs.set("offset", String(params.offset));
  const query = qs.toString() ? `?${qs}` : "";
  return apiFetch(`/v1/detection/timeline${query}`, { ...opts, method: "GET" });
}

export async function markAlertReviewed(
  id: string,
  opts: ApiOptions
): Promise<DetectionAlert> {
  return apiFetch(`/v1/detection/incidents/${id}`, {
    ...opts,
    method: "PATCH",
    body: JSON.stringify({ reviewed: true }),
  });
}

// ── Detection — Incidents ────────────────────────────────────────────────────

export type IncidentStatus = "new" | "investigating" | "resolved" | "closed";

export interface Incident {
  id: string;
  tenant_id: string;
  rule_id: string;
  rule_name: string;
  severity: AlertSeverity;
  status: IncidentStatus;
  mitre_tactic?: string;
  mitre_technique?: string;
  contributing_events: number;
  first_seen: string;
  last_seen: string;
  thehive_case_id?: string;
}

export interface IncidentListParams {
  since?: string;
  severity?: AlertSeverity;
  status?: IncidentStatus;
  limit?: number;
}

export async function listIncidents(
  params: IncidentListParams,
  opts: ApiOptions
): Promise<Incident[]> {
  const qs = new URLSearchParams();
  if (params.since) qs.set("since", params.since);
  if (params.severity) qs.set("severity", String(params.severity));
  if (params.status) qs.set("status", params.status);
  if (params.limit) qs.set("limit", String(params.limit));
  const query = qs.toString() ? `?${qs}` : "";
  return apiFetch(`/v1/detection/incidents${query}`, { ...opts, method: "GET" });
}

export async function getIncident(
  id: string,
  opts: ApiOptions
): Promise<Incident> {
  return apiFetch(`/v1/detection/incidents/${id}`, { ...opts, method: "GET" });
}

export async function updateIncidentStatus(
  id: string,
  status: IncidentStatus,
  opts: ApiOptions
): Promise<Incident> {
  return apiFetch(`/v1/detection/incidents/${id}`, {
    ...opts,
    method: "PATCH",
    body: JSON.stringify({ status }),
  });
}

export async function dispatchIncident(
  id: string,
  opts: ApiOptions
): Promise<{ dispatched: boolean; soar_case_id?: string }> {
  return apiFetch(`/v1/detection/incidents/${id}/dispatch`, {
    ...opts,
    method: "POST",
  });
}

// ── Detection — Stats (summary for header row) ───────────────────────────────

export interface DetectionStats {
  incidents_24h: number;
  critical_alerts: number;
  active_agents: number;
  mttr_minutes: number;
}

export async function getDetectionStats(opts: ApiOptions): Promise<DetectionStats> {
  return apiFetch("/v1/detection/stats", { ...opts, method: "GET" });
}

// ── Integrations — Health ─────────────────────────────────────────────────────

export type IntegrationStatus = "connected" | "degraded" | "offline";

export interface IntegrationHealth {
  name: string;
  status: IntegrationStatus;
  last_check: string;
  version?: string;
  error?: string;
  latency_ms?: number;
}

export interface IntegrationsHealthResponse {
  integrations: IntegrationHealth[];
  healthy_count: number;
  total_count: number;
  checked_at: string;
}

export async function getIntegrationsHealth(
  opts: ApiOptions
): Promise<IntegrationsHealthResponse> {
  return apiFetch("/v1/integrations/health", { ...opts, method: "GET" });
}

export async function reconnectIntegration(
  name: string,
  opts: ApiOptions
): Promise<{ triggered: boolean }> {
  return apiFetch(`/v1/integrations/${name}/reconnect`, {
    ...opts,
    method: "POST",
  });
}

// ── WebSocket — Detection alert stream ───────────────────────────────────────

const WS_BASE =
  process.env.NEXT_PUBLIC_WS_BASE ||
  (typeof window !== "undefined"
    ? `ws://${window.location.hostname}:8080`
    : "ws://localhost:8080");

export interface WsDetectionAlert extends DetectionAlert {
  _ws_seq: number;
}

export interface WsConnection {
  close(): void;
}

/**
 * Opens a WebSocket connection to /api/ws/alerts scoped to a tenant.
 * Automatically reconnects with exponential back-off on disconnect.
 * Returns a handle with a close() method to tear down the connection.
 */
export function connectAlertStream(
  tenantId: string,
  token: string,
  onAlert: (alert: WsDetectionAlert) => void,
  onStatusChange?: (status: "connecting" | "open" | "closed") => void
): WsConnection {
  let ws: WebSocket | null = null;
  let closed = false;
  let retryDelay = 1000;
  let seq = 0;

  function connect() {
    if (closed) return;
    onStatusChange?.("connecting");

    ws = new WebSocket(
      `${WS_BASE}/api/ws/alerts?tenant_id=${tenantId}&token=${encodeURIComponent(token)}`
    );

    ws.onopen = () => {
      retryDelay = 1000;
      onStatusChange?.("open");
    };

    ws.onmessage = (ev) => {
      try {
        const alert = JSON.parse(ev.data as string) as DetectionAlert;
        onAlert({ ...alert, _ws_seq: ++seq });
      } catch {
        // skip malformed frames
      }
    };

    ws.onerror = () => {
      ws?.close();
    };

    ws.onclose = () => {
      onStatusChange?.("closed");
      if (!closed) {
        setTimeout(connect, retryDelay);
        retryDelay = Math.min(retryDelay * 2, 30_000);
      }
    };
  }

  connect();

  return {
    close() {
      closed = true;
      ws?.close();
    },
  };
}
