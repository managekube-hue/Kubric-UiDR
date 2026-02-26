"use client";

/**
 * IncidentTimeline — Correlation engine incident timeline visualization.
 * Shows how individual alerts chain into correlated incidents on a vertical timeline.
 * Polls every 30 s via SWR. Incidents expand to reveal contributing events.
 */

import { useState, useCallback } from "react";
import useSWR from "swr";
import { useSession } from "next-auth/react";
import {
  Card,
  Title,
  Text,
  Badge,
  Select,
  SelectItem,
  Button,
} from "@tremor/react";
import {
  ExternalLink,
  Send,
  ChevronDown,
  ChevronRight,
  RefreshCw,
  AlertCircle,
  Clock,
} from "lucide-react";
import {
  listIncidents,
  dispatchIncident,
  type Incident,
  type IncidentStatus,
  type AlertSeverity,
} from "@/lib/detection-api";

// ── Constants ─────────────────────────────────────────────────────────────────

const THEHIVE_URL =
  process.env.NEXT_PUBLIC_THEHIVE_URL || "http://localhost:9000";

const POLL_INTERVAL = 30_000;

const SEV_BADGE_COLORS: Record<AlertSeverity, string> = {
  1: "gray",
  2: "blue",
  3: "yellow",
  4: "orange",
  5: "red",
};

const SEV_LABELS: Record<AlertSeverity, string> = {
  1: "Info",
  2: "Low",
  3: "Medium",
  4: "High",
  5: "Critical",
};

const STATUS_COLORS: Record<IncidentStatus, string> = {
  new: "red",
  investigating: "yellow",
  resolved: "green",
  closed: "gray",
};

const STATUS_LABELS: Record<IncidentStatus, string> = {
  new: "New",
  investigating: "Investigating",
  resolved: "Resolved",
  closed: "Closed",
};

type TimeRange = "1h" | "6h" | "24h" | "7d";

const TIME_RANGE_MS: Record<TimeRange, number> = {
  "1h": 3_600_000,
  "6h": 21_600_000,
  "24h": 86_400_000,
  "7d": 604_800_000,
};

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function formatDuration(first: string, last: string): string {
  const ms = new Date(last).getTime() - new Date(first).getTime();
  if (ms < 60_000) return `${Math.round(ms / 1000)}s`;
  if (ms < 3_600_000) return `${Math.round(ms / 60_000)}m`;
  return `${(ms / 3_600_000).toFixed(1)}h`;
}

// ── Status pulse dot ─────────────────────────────────────────────────────────

function StatusDot({ status }: { status: IncidentStatus }) {
  const colorClass =
    status === "new"
      ? "bg-red-500 animate-pulse"
      : status === "investigating"
        ? "bg-yellow-400 animate-pulse"
        : status === "resolved"
          ? "bg-green-500"
          : "bg-gray-400";
  return (
    <span
      className={`inline-block h-2.5 w-2.5 rounded-full flex-shrink-0 mt-1 ${colorClass}`}
    />
  );
}

// ── Incident row ─────────────────────────────────────────────────────────────

interface IncidentRowProps {
  incident: Incident;
  onDispatch: (id: string) => Promise<void>;
  isLast: boolean;
}

function IncidentRow({ incident: inc, onDispatch, isLast }: IncidentRowProps) {
  const [expanded, setExpanded] = useState(false);
  const [dispatching, setDispatching] = useState(false);
  const [dispatched, setDispatched] = useState(false);

  async function handleDispatch() {
    setDispatching(true);
    try {
      await onDispatch(inc.id);
      setDispatched(true);
    } finally {
      setDispatching(false);
    }
  }

  return (
    <div className="relative flex gap-4">
      {/* Timeline track */}
      <div className="flex flex-col items-center">
        <StatusDot status={inc.status} />
        {!isLast && (
          <div className="flex-1 w-px bg-gray-200 dark:bg-gray-700 min-h-[40px]" />
        )}
      </div>

      {/* Content */}
      <div className="flex-1 pb-6 min-w-0">
        {/* Header row */}
        <div className="flex flex-wrap items-start gap-2 mb-1">
          <button
            onClick={() => setExpanded((v) => !v)}
            className="flex items-center gap-1 text-sm font-semibold text-gray-900 dark:text-gray-100 hover:text-kubric-600 transition-colors text-left"
          >
            {expanded ? (
              <ChevronDown className="h-4 w-4 flex-shrink-0" />
            ) : (
              <ChevronRight className="h-4 w-4 flex-shrink-0" />
            )}
            <span>{inc.rule_name}</span>
          </button>
          <Badge color={SEV_BADGE_COLORS[inc.severity]}>
            {SEV_LABELS[inc.severity]}
          </Badge>
          <Badge color={STATUS_COLORS[inc.status]}>
            {STATUS_LABELS[inc.status]}
          </Badge>
        </div>

        {/* Meta row */}
        <div className="flex flex-wrap items-center gap-3 text-xs text-gray-500 mb-2 ml-5">
          {inc.mitre_tactic && (
            <span className="font-mono">{inc.mitre_tactic}</span>
          )}
          {inc.mitre_technique && (
            <>
              <span aria-hidden>·</span>
              <span className="font-mono bg-gray-100 dark:bg-gray-700 rounded px-1.5 py-0.5">
                {inc.mitre_technique}
              </span>
            </>
          )}
          <span aria-hidden>·</span>
          <Clock className="h-3 w-3" />
          <span>
            {formatDate(inc.first_seen)} –{" "}
            {formatDate(inc.last_seen)} (
            {formatDuration(inc.first_seen, inc.last_seen)})
          </span>
          <span aria-hidden>·</span>
          <span>{inc.contributing_events} event{inc.contributing_events !== 1 ? "s" : ""}</span>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2 ml-5">
          {inc.thehive_case_id && (
            <a
              href={`${THEHIVE_URL}/cases/${inc.thehive_case_id}`}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-1 text-xs text-kubric-500 hover:text-kubric-700 transition-colors"
            >
              <ExternalLink className="h-3.5 w-3.5" />
              Open in TheHive
            </a>
          )}
          <button
            onClick={handleDispatch}
            disabled={dispatching || dispatched}
            className={`inline-flex items-center gap-1 text-xs transition-colors ${
              dispatched
                ? "text-green-500 cursor-default"
                : "text-gray-400 hover:text-kubric-500"
            } disabled:opacity-60`}
          >
            {dispatching ? (
              <RefreshCw className="h-3.5 w-3.5 animate-spin" />
            ) : (
              <Send className="h-3.5 w-3.5" />
            )}
            {dispatched ? "Dispatched" : "Dispatch to SOAR"}
          </button>
        </div>

        {/* Expanded detail */}
        {expanded && (
          <div className="ml-5 mt-3 rounded-lg border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 p-3 text-xs space-y-1">
            <div className="grid grid-cols-2 gap-x-4 gap-y-1 text-gray-600 dark:text-gray-300">
              <span className="text-gray-400">Incident ID</span>
              <span className="font-mono truncate">{inc.id}</span>
              <span className="text-gray-400">Rule ID</span>
              <span className="font-mono truncate">{inc.rule_id}</span>
              {inc.thehive_case_id && (
                <>
                  <span className="text-gray-400">TheHive Case</span>
                  <span className="font-mono">{inc.thehive_case_id}</span>
                </>
              )}
              <span className="text-gray-400">Contributing events</span>
              <span>{inc.contributing_events}</span>
              <span className="text-gray-400">First seen</span>
              <span>{formatDate(inc.first_seen)}</span>
              <span className="text-gray-400">Last seen</span>
              <span>{formatDate(inc.last_seen)}</span>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ── Main Component ────────────────────────────────────────────────────────────

export function IncidentTimeline() {
  const { data: session } = useSession();
  const token = (session as { accessToken?: string } | null)?.accessToken ?? "";
  const tenantId = (session as { tenantId?: string } | null)?.tenantId ?? "";

  const [timeRange, setTimeRange] = useState<TimeRange>("24h");
  const [severityFilter, setSeverityFilter] = useState<string>("all");

  const since = new Date(
    Date.now() - TIME_RANGE_MS[timeRange]
  ).toISOString();

  const {
    data: incidents,
    isLoading,
    isValidating,
    mutate,
    error,
  } = useSWR(
    token && tenantId ? ["incidents", tenantId, timeRange, severityFilter] : null,
    () =>
      listIncidents(
        {
          since,
          ...(severityFilter !== "all"
            ? { severity: Number(severityFilter) as AlertSeverity }
            : {}),
          limit: 100,
        },
        { token, tenantId }
      ),
    { refreshInterval: POLL_INTERVAL, revalidateOnFocus: false }
  );

  const handleDispatch = useCallback(
    async (id: string) => {
      await dispatchIncident(id, { token, tenantId });
      await mutate();
    },
    [token, tenantId, mutate]
  );

  const list = incidents ?? [];

  return (
    <Card>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <Title>Incident Timeline</Title>
          {isValidating && !isLoading && (
            <RefreshCw className="h-3.5 w-3.5 text-gray-400 animate-spin" />
          )}
        </div>
        <div className="flex items-center gap-2">
          {/* Time range filter */}
          <div className="w-28">
            <Select
              value={timeRange}
              onValueChange={(v) => setTimeRange(v as TimeRange)}
            >
              <SelectItem value="1h">Last 1 h</SelectItem>
              <SelectItem value="6h">Last 6 h</SelectItem>
              <SelectItem value="24h">Last 24 h</SelectItem>
              <SelectItem value="7d">Last 7 d</SelectItem>
            </Select>
          </div>
          {/* Severity filter */}
          <div className="w-36">
            <Select
              value={severityFilter}
              onValueChange={setSeverityFilter}
            >
              <SelectItem value="all">All severities</SelectItem>
              <SelectItem value="5">Critical</SelectItem>
              <SelectItem value="4">High</SelectItem>
              <SelectItem value="3">Medium</SelectItem>
              <SelectItem value="2">Low</SelectItem>
              <SelectItem value="1">Info</SelectItem>
            </Select>
          </div>
        </div>
      </div>

      {/* Summary bar */}
      {list.length > 0 && (
        <div className="flex flex-wrap gap-3 mb-4 text-xs text-gray-500">
          {(["new", "investigating", "resolved"] as IncidentStatus[]).map(
            (s) => {
              const count = list.filter((i) => i.status === s).length;
              return count > 0 ? (
                <span key={s} className="flex items-center gap-1">
                  <StatusDot status={s} />
                  <span>
                    {count} {STATUS_LABELS[s]}
                  </span>
                </span>
              ) : null;
            }
          )}
        </div>
      )}

      {/* Timeline body */}
      <div className="overflow-auto max-h-[640px] pr-1">
        {isLoading ? (
          <div className="py-12 flex items-center justify-center gap-2 text-gray-400">
            <RefreshCw className="h-5 w-5 animate-spin" />
            <Text>Loading incidents…</Text>
          </div>
        ) : error ? (
          <div className="py-12 flex items-center justify-center gap-2 text-red-400">
            <AlertCircle className="h-5 w-5" />
            <Text>Failed to load incidents. Retrying…</Text>
          </div>
        ) : list.length === 0 ? (
          <Text className="py-8 text-center text-gray-400">
            No incidents in the selected time range.
          </Text>
        ) : (
          <div className="mt-2">
            {list.map((inc, idx) => (
              <IncidentRow
                key={inc.id}
                incident={inc}
                onDispatch={handleDispatch}
                isLast={idx === list.length - 1}
              />
            ))}
          </div>
        )}
      </div>
    </Card>
  );
}
