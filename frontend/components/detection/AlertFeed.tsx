"use client";

/**
 * AlertFeed — Live detection alert feed with real-time NATS WebSocket updates.
 * Displays severity-colored alert cards with MITRE ATT&CK context.
 * Combines REST polling (SWR, every 10 s) with WebSocket push for low-latency delivery.
 */

import {
  useEffect,
  useState,
  useRef,
  useCallback,
  useMemo,
} from "react";
import useSWR from "swr";
import { useSession } from "next-auth/react";
import {
  Card,
  Title,
  Text,
  Badge,
  TextInput,
  Select,
  SelectItem,
  MultiSelect,
  MultiSelectItem,
  Button,
} from "@tremor/react";
import {
  Search,
  RefreshCw,
  CheckCircle,
  Wifi,
  WifiOff,
  PauseCircle,
} from "lucide-react";
import {
  getAlertTimeline,
  connectAlertStream,
  markAlertReviewed,
  type DetectionAlert,
  type AlertSeverity,
  type AlertSource,
  type WsDetectionAlert,
} from "@/lib/detection-api";

// ── Constants ─────────────────────────────────────────────────────────────────

const SEV_COLORS: Record<AlertSeverity, string> = {
  1: "gray",
  2: "blue",
  3: "yellow",
  4: "orange",
  5: "red",
} as const;

const SEV_LABELS: Record<AlertSeverity, string> = {
  1: "Info",
  2: "Low",
  3: "Medium",
  4: "High",
  5: "Critical",
} as const;

const SOURCE_ICONS: Record<AlertSource, string> = {
  coresec: "🛡️",
  netguard: "🌐",
  wazuh: "👁️",
  falco: "🦅",
  suricata: "⚡",
} as const;

const ALL_SOURCES: AlertSource[] = [
  "coresec",
  "netguard",
  "wazuh",
  "falco",
  "suricata",
];

const POLL_INTERVAL = 10_000; // 10 s
const MAX_ALERTS = 500;
const VISIBLE_WINDOW = 60; // rows to render at a time (virtual window)

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60_000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
  return `${Math.floor(diff / 86_400_000)}d ago`;
}

// ── Alert Card ────────────────────────────────────────────────────────────────

interface AlertCardProps {
  alert: DetectionAlert;
  onReview: (id: string) => Promise<void>;
}

function AlertCard({ alert, onReview }: AlertCardProps) {
  const [reviewing, setReviewing] = useState(false);
  const [done, setDone] = useState(alert.reviewed ?? false);

  async function handleReview() {
    if (done) return;
    setReviewing(true);
    try {
      await onReview(alert.id);
      setDone(true);
    } finally {
      setReviewing(false);
    }
  }

  return (
    <div
      className={`flex items-start gap-3 rounded-lg border p-3 transition-opacity ${
        done ? "opacity-50" : "opacity-100"
      } hover:bg-gray-50 dark:hover:bg-gray-800`}
    >
      {/* Source icon */}
      <span className="text-xl leading-none mt-0.5 flex-shrink-0">
        {SOURCE_ICONS[alert.source] ?? "❓"}
      </span>

      {/* Body */}
      <div className="flex-1 min-w-0 space-y-1">
        <div className="flex flex-wrap items-center gap-2">
          <Badge color={SEV_COLORS[alert.severity]}>
            {SEV_LABELS[alert.severity]}
          </Badge>
          <span className="text-xs font-mono text-gray-500 uppercase">
            {alert.source}
          </span>
          {alert.mitre_technique && (
            <span className="text-xs bg-gray-100 dark:bg-gray-700 rounded px-1.5 py-0.5 font-mono">
              {alert.mitre_technique}
            </span>
          )}
        </div>

        <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
          {alert.title}
        </p>

        <div className="flex items-center gap-3 text-xs text-gray-500">
          <span>{alert.hostname}</span>
          {alert.mitre_tactic && (
            <>
              <span aria-hidden>·</span>
              <span>{alert.mitre_tactic}</span>
            </>
          )}
        </div>
      </div>

      {/* Right side */}
      <div className="flex flex-col items-end gap-2 flex-shrink-0">
        <span className="text-xs text-gray-400">{timeAgo(alert.timestamp)}</span>
        <button
          onClick={handleReview}
          disabled={done || reviewing}
          title="Mark as reviewed"
          className="text-gray-400 hover:text-green-500 disabled:cursor-not-allowed transition-colors"
        >
          {reviewing ? (
            <RefreshCw className="h-4 w-4 animate-spin" />
          ) : (
            <CheckCircle
              className={`h-4 w-4 ${done ? "text-green-500" : ""}`}
            />
          )}
        </button>
      </div>
    </div>
  );
}

// ── Main Component ────────────────────────────────────────────────────────────

export function AlertFeed() {
  const { data: session } = useSession();
  const token = (session as { accessToken?: string } | null)?.accessToken ?? "";
  const tenantId = (session as { tenantId?: string } | null)?.tenantId ?? "";

  // Live alerts accumulated from REST poll + WebSocket push
  const [liveAlerts, setLiveAlerts] = useState<DetectionAlert[]>([]);
  const [wsStatus, setWsStatus] = useState<"connecting" | "open" | "closed">(
    "closed"
  );

  // Scroll / pause state
  const scrollRef = useRef<HTMLDivElement>(null);
  const [autoScroll, setAutoScroll] = useState(true);
  const isHovered = useRef(false);

  // Scroll offset for virtual window
  const [scrollTop, setScrollTop] = useState(0);
  const [itemHeight] = useState(72); // px per row

  // Filters
  const [search, setSearch] = useState("");
  const [severityFilter, setSeverityFilter] = useState<string>("all");
  const [sourceFilter, setSourceFilter] = useState<string[]>([]);

  // ── REST polling ────────────────────────────────────────────────────────────

  const { mutate } = useSWR(
    token && tenantId ? ["detection-timeline", tenantId] : null,
    async () => {
      const alerts = await getAlertTimeline(
        { limit: 200 },
        { token, tenantId }
      );
      setLiveAlerts((prev) => {
        const existingIds = new Set(prev.map((a) => a.id));
        const newOnes = alerts.filter((a) => !existingIds.has(a.id));
        if (newOnes.length === 0) return prev;
        return [
          ...newOnes,
          ...prev,
        ].slice(0, MAX_ALERTS);
      });
      return alerts;
    },
    { refreshInterval: POLL_INTERVAL, revalidateOnFocus: false }
  );

  // ── WebSocket push ──────────────────────────────────────────────────────────

  useEffect(() => {
    if (!token || !tenantId) return;

    const conn = connectAlertStream(
      tenantId,
      token,
      (alert: WsDetectionAlert) => {
        setLiveAlerts((prev) => {
          if (prev.some((a) => a.id === alert.id)) return prev;
          return [alert, ...prev].slice(0, MAX_ALERTS);
        });
        // Trigger SWR refresh to sync reviewed states
        mutate();
      },
      setWsStatus
    );

    return () => conn.close();
  }, [token, tenantId, mutate]);

  // ── Auto-scroll ─────────────────────────────────────────────────────────────

  useEffect(() => {
    if (autoScroll && !isHovered.current && scrollRef.current) {
      scrollRef.current.scrollTop = 0;
    }
  }, [liveAlerts, autoScroll]);

  // ── Mark reviewed ───────────────────────────────────────────────────────────

  const handleReview = useCallback(
    async (id: string) => {
      await markAlertReviewed(id, { token, tenantId });
      setLiveAlerts((prev) =>
        prev.map((a) => (a.id === id ? { ...a, reviewed: true } : a))
      );
    },
    [token, tenantId]
  );

  // ── Filtering ───────────────────────────────────────────────────────────────

  const filtered = useMemo(() => {
    return liveAlerts.filter((a) => {
      if (
        severityFilter !== "all" &&
        a.severity !== Number(severityFilter)
      )
        return false;
      if (sourceFilter.length > 0 && !sourceFilter.includes(a.source))
        return false;
      if (search) {
        const q = search.toLowerCase();
        if (
          !a.title.toLowerCase().includes(q) &&
          !a.hostname.toLowerCase().includes(q) &&
          !(a.mitre_technique?.toLowerCase().includes(q) ?? false)
        )
          return false;
      }
      return true;
    });
  }, [liveAlerts, severityFilter, sourceFilter, search]);

  // ── Virtual window ──────────────────────────────────────────────────────────

  const startIndex = Math.floor(scrollTop / itemHeight);
  const visibleAlerts = filtered.slice(startIndex, startIndex + VISIBLE_WINDOW);
  const paddingTop = startIndex * itemHeight;
  const paddingBottom = Math.max(
    0,
    (filtered.length - startIndex - VISIBLE_WINDOW) * itemHeight
  );

  // ── Render ──────────────────────────────────────────────────────────────────

  return (
    <Card>
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <Title>Live Alert Feed</Title>
          <span className="text-xs text-gray-500">
            {filtered.length} / {liveAlerts.length}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {wsStatus === "open" ? (
            <span className="flex items-center gap-1 text-xs text-green-500">
              <Wifi className="h-3.5 w-3.5" />
              Live
            </span>
          ) : wsStatus === "connecting" ? (
            <span className="flex items-center gap-1 text-xs text-yellow-500">
              <RefreshCw className="h-3.5 w-3.5 animate-spin" />
              Connecting
            </span>
          ) : (
            <span className="flex items-center gap-1 text-xs text-gray-400">
              <WifiOff className="h-3.5 w-3.5" />
              Polling
            </span>
          )}
          <button
            onClick={() => setAutoScroll((v) => !v)}
            className="text-gray-400 hover:text-gray-600 transition-colors"
            title={autoScroll ? "Pause auto-scroll" : "Resume auto-scroll"}
          >
            <PauseCircle
              className={`h-4 w-4 ${autoScroll ? "text-kubric-400" : "text-gray-300"}`}
            />
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap gap-2 mb-4">
        <div className="flex-1 min-w-[180px]">
          <TextInput
            icon={Search}
            placeholder="Search title, host, technique…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
        <div className="w-36">
          <Select
            value={severityFilter}
            onValueChange={setSeverityFilter}
            placeholder="Severity"
          >
            <SelectItem value="all">All severities</SelectItem>
            <SelectItem value="5">Critical</SelectItem>
            <SelectItem value="4">High</SelectItem>
            <SelectItem value="3">Medium</SelectItem>
            <SelectItem value="2">Low</SelectItem>
            <SelectItem value="1">Info</SelectItem>
          </Select>
        </div>
        <div className="w-52">
          <MultiSelect
            value={sourceFilter}
            onValueChange={setSourceFilter}
            placeholder="All sources"
          >
            {ALL_SOURCES.map((s) => (
              <MultiSelectItem key={s} value={s}>
                {SOURCE_ICONS[s]} {s}
              </MultiSelectItem>
            ))}
          </MultiSelect>
        </div>
      </div>

      {/* Virtual list */}
      <div
        ref={scrollRef}
        className="overflow-auto max-h-[600px] space-y-0"
        onScroll={(e) => setScrollTop(e.currentTarget.scrollTop)}
        onMouseEnter={() => {
          isHovered.current = true;
        }}
        onMouseLeave={() => {
          isHovered.current = false;
        }}
      >
        {filtered.length === 0 ? (
          <Text className="py-8 text-center text-gray-400">
            {liveAlerts.length === 0
              ? "Waiting for detection events…"
              : "No alerts match the current filters."}
          </Text>
        ) : (
          <>
            <div style={{ height: paddingTop }} />
            <div className="space-y-1.5">
              {visibleAlerts.map((alert) => (
                <AlertCard
                  key={alert.id}
                  alert={alert}
                  onReview={handleReview}
                />
              ))}
            </div>
            <div style={{ height: paddingBottom }} />
          </>
        )}
      </div>
    </Card>
  );
}
