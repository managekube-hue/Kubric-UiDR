"use client";

/**
 * IntegrationHealth — Real-time health status panel for all security tool integrations.
 * Shows connectivity for: Wazuh, Velociraptor, TheHive, Cortex, Falco,
 * BloodHound, osquery, and Shuffle. Polls every 30 s via SWR.
 */

import { useState } from "react";
import useSWR from "swr";
import { useSession } from "next-auth/react";
import {
  Card,
  Title,
  Text,
  Grid,
  Badge,
} from "@tremor/react";
import { RefreshCw, AlertCircle, PlugZap } from "lucide-react";
import {
  getIntegrationsHealth,
  reconnectIntegration,
  type IntegrationStatus,
  type IntegrationHealth as IH,
} from "@/lib/detection-api";

// ── Tool metadata ─────────────────────────────────────────────────────────────

interface ToolMeta {
  label: string;
  description: string;
  icon: string;
  docUrl?: string;
}

const TOOL_META: Record<string, ToolMeta> = {
  wazuh: {
    label: "Wazuh",
    description: "EDR / SIEM",
    icon: "👁️",
    docUrl: "https://wazuh.com/docs",
  },
  velociraptor: {
    label: "Velociraptor",
    description: "DFIR",
    icon: "🦖",
    docUrl: "https://docs.velociraptor.app",
  },
  thehive: {
    label: "TheHive",
    description: "Case Management",
    icon: "🐝",
    docUrl: "https://thehive-project.org",
  },
  cortex: {
    label: "Cortex",
    description: "Analysis / Automation",
    icon: "🧠",
  },
  falco: {
    label: "Falco",
    description: "Runtime Security",
    icon: "🦅",
    docUrl: "https://falco.org/docs",
  },
  bloodhound: {
    label: "BloodHound",
    description: "AD Attack Paths",
    icon: "🐕",
    docUrl: "https://bloodhound.readthedocs.io",
  },
  osquery: {
    label: "osquery",
    description: "Endpoint Telemetry",
    icon: "🔍",
    docUrl: "https://osquery.readthedocs.io",
  },
  shuffle: {
    label: "Shuffle",
    description: "SOAR",
    icon: "🔀",
    docUrl: "https://shuffler.io/docs",
  },
};

const POLL_INTERVAL = 30_000;

// ── Status helpers ────────────────────────────────────────────────────────────

const STATUS_BADGE_COLORS: Record<IntegrationStatus, string> = {
  connected: "green",
  degraded: "yellow",
  offline: "red",
};

const STATUS_LABELS: Record<IntegrationStatus, string> = {
  connected: "Connected",
  degraded: "Degraded",
  offline: "Offline",
};

function StatusPulse({ status }: { status: IntegrationStatus }) {
  const colorClass =
    status === "connected"
      ? "bg-green-500 animate-pulse"
      : status === "degraded"
        ? "bg-yellow-400 animate-pulse"
        : "bg-red-500";
  return (
    <span
      className={`inline-block h-2.5 w-2.5 rounded-full flex-shrink-0 ${colorClass}`}
      aria-label={STATUS_LABELS[status]}
    />
  );
}

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60_000) return "just now";
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  return `${Math.floor(diff / 3_600_000)}h ago`;
}

// ── Integration Card ──────────────────────────────────────────────────────────

interface IntegrationCardProps {
  integration: IH;
  onReconnect: (name: string) => Promise<void>;
}

function IntegrationCard({ integration: ig, onReconnect }: IntegrationCardProps) {
  const [reconnecting, setReconnecting] = useState(false);
  const meta = TOOL_META[ig.name.toLowerCase()] ?? {
    label: ig.name,
    description: "Security Tool",
    icon: "🔧",
  };

  async function handleReconnect() {
    setReconnecting(true);
    try {
      await onReconnect(ig.name);
    } finally {
      setReconnecting(false);
    }
  }

  return (
    <div
      className={`rounded-lg border p-4 space-y-3 transition-colors ${
        ig.status === "connected"
          ? "border-green-200 dark:border-green-900"
          : ig.status === "degraded"
            ? "border-yellow-200 dark:border-yellow-900"
            : "border-red-200 dark:border-red-900 bg-red-50 dark:bg-red-950/20"
      }`}
    >
      {/* Header */}
      <div className="flex items-start justify-between gap-2">
        <div className="flex items-center gap-2">
          <span className="text-2xl leading-none">{meta.icon}</span>
          <div>
            <p className="text-sm font-semibold text-gray-900 dark:text-gray-100">
              {meta.label}
            </p>
            <p className="text-xs text-gray-500">{meta.description}</p>
          </div>
        </div>
        <StatusPulse status={ig.status} />
      </div>

      {/* Status badge + latency */}
      <div className="flex items-center gap-2 flex-wrap">
        <Badge color={STATUS_BADGE_COLORS[ig.status]}>
          {STATUS_LABELS[ig.status]}
        </Badge>
        {ig.version && (
          <span className="text-xs text-gray-400 font-mono">v{ig.version}</span>
        )}
        {ig.latency_ms != null && ig.status === "connected" && (
          <span className="text-xs text-gray-400">{ig.latency_ms} ms</span>
        )}
      </div>

      {/* Error message if degraded/offline */}
      {ig.error && ig.status !== "connected" && (
        <p className="text-xs text-red-500 truncate" title={ig.error}>
          {ig.error}
        </p>
      )}

      {/* Footer: last check + reconnect */}
      <div className="flex items-center justify-between pt-1 border-t border-gray-100 dark:border-gray-800">
        <span className="text-xs text-gray-400">
          Checked {timeAgo(ig.last_check)}
        </span>
        {ig.status !== "connected" && (
          <button
            onClick={handleReconnect}
            disabled={reconnecting}
            className="inline-flex items-center gap-1 text-xs text-kubric-500 hover:text-kubric-700 disabled:opacity-50 transition-colors"
          >
            {reconnecting ? (
              <RefreshCw className="h-3 w-3 animate-spin" />
            ) : (
              <PlugZap className="h-3 w-3" />
            )}
            Reconnect
          </button>
        )}
      </div>
    </div>
  );
}

// ── Health Score Bar ──────────────────────────────────────────────────────────

function HealthScoreBar({
  healthy,
  total,
}: {
  healthy: number;
  total: number;
}) {
  const pct = total === 0 ? 0 : Math.round((healthy / total) * 100);
  const color =
    pct === 100
      ? "bg-green-500"
      : pct >= 50
        ? "bg-yellow-400"
        : "bg-red-500";

  return (
    <div className="flex items-center gap-3">
      <div className="flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
        <div
          className={`h-full ${color} transition-all duration-700`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-sm font-semibold text-gray-700 dark:text-gray-300 whitespace-nowrap">
        {healthy} / {total} connected
      </span>
    </div>
  );
}

// ── Main Component ────────────────────────────────────────────────────────────

export function IntegrationHealth() {
  const { data: session } = useSession();
  const token = (session as { accessToken?: string } | null)?.accessToken ?? "";
  const tenantId = (session as { tenantId?: string } | null)?.tenantId ?? "";

  const {
    data,
    isLoading,
    isValidating,
    mutate,
    error,
  } = useSWR(
    token && tenantId ? ["integrations-health", tenantId] : null,
    () => getIntegrationsHealth({ token, tenantId }),
    { refreshInterval: POLL_INTERVAL, revalidateOnFocus: false }
  );

  async function handleReconnect(name: string) {
    await reconnectIntegration(name, { token, tenantId });
    // Allow backend a moment to process then re-validate
    setTimeout(() => mutate(), 3_000);
  }

  const integrations = data?.integrations ?? [];
  const healthy = data?.healthy_count ?? 0;
  const total = data?.total_count ?? integrations.length;

  return (
    <Card>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <Title>Integration Health</Title>
          {isValidating && !isLoading && (
            <RefreshCw className="h-3.5 w-3.5 text-gray-400 animate-spin" />
          )}
        </div>
        {data?.checked_at && (
          <span className="text-xs text-gray-400">
            Last checked {timeAgo(data.checked_at)}
          </span>
        )}
      </div>

      {/* Health score */}
      {!isLoading && integrations.length > 0 && (
        <div className="mb-5">
          <HealthScoreBar healthy={healthy} total={total} />
        </div>
      )}

      {/* Grid */}
      {isLoading ? (
        <div className="py-12 flex items-center justify-center gap-2 text-gray-400">
          <RefreshCw className="h-5 w-5 animate-spin" />
          <Text>Checking integrations…</Text>
        </div>
      ) : error ? (
        <div className="py-12 flex items-center justify-center gap-2 text-red-400">
          <AlertCircle className="h-5 w-5" />
          <Text>Failed to load integration status. Retrying…</Text>
        </div>
      ) : integrations.length === 0 ? (
        <Text className="py-8 text-center text-gray-400">
          No integrations configured.
        </Text>
      ) : (
        <Grid numItemsMd={2} numItemsLg={4} className="gap-3">
          {integrations.map((ig) => (
            <IntegrationCard
              key={ig.name}
              integration={ig}
              onReconnect={handleReconnect}
            />
          ))}
        </Grid>
      )}
    </Card>
  );
}
