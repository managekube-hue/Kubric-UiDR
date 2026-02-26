"use client";

/**
 * Detection & Response — main MDR dashboard page.
 * Route: /detection  (inside the (tenant) route group)
 *
 * Layout:
 *   - Page header with title + subtitle
 *   - KPI stats row: Total Incidents (24h), Critical Alerts, Active Agents, MTTR
 *   - Tab switcher: Live Feed | Incidents | Integrations
 *   - Tab content panels rendered via the three detection components
 */

import { useState, useCallback } from "react";
import useSWR from "swr";
import { useSession } from "next-auth/react";
import {
  Card,
  Title,
  Text,
  Metric,
  Flex,
  Grid,
  Badge,
} from "@tremor/react";
import {
  ShieldAlert,
  AlertTriangle,
  Cpu,
  Clock,
  Radio,
  GitBranch,
  Plug,
} from "lucide-react";
import { getDetectionStats } from "@/lib/detection-api";
import { AlertFeed } from "@/components/detection/AlertFeed";
import { IncidentTimeline } from "@/components/detection/IncidentTimeline";
import { IntegrationHealth } from "@/components/detection/IntegrationHealth";

// ── Tab definition ────────────────────────────────────────────────────────────

type Tab = "feed" | "incidents" | "integrations";

interface TabItem {
  id: Tab;
  label: string;
  icon: React.ElementType;
}

const TABS: TabItem[] = [
  { id: "feed", label: "Live Feed", icon: Radio },
  { id: "incidents", label: "Incidents", icon: GitBranch },
  { id: "integrations", label: "Integrations", icon: Plug },
];

// ── Stat card ─────────────────────────────────────────────────────────────────

interface StatCardProps {
  label: string;
  value: string | number;
  icon: React.ElementType;
  decorationColor: string;
  loading?: boolean;
}

function StatCard({
  label,
  value,
  icon: Icon,
  decorationColor,
  loading,
}: StatCardProps) {
  return (
    <Card decoration="top" decorationColor={decorationColor}>
      <Flex alignItems="start">
        <div>
          <Text>{label}</Text>
          {loading ? (
            <div className="h-8 w-16 mt-1 bg-gray-200 dark:bg-gray-700 rounded animate-pulse" />
          ) : (
            <Metric>{value}</Metric>
          )}
        </div>
        <Icon className="h-6 w-6 text-gray-400 flex-shrink-0" />
      </Flex>
    </Card>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function DetectionPage() {
  const { data: session } = useSession();
  const token = (session as { accessToken?: string } | null)?.accessToken ?? "";
  const tenantId = (session as { tenantId?: string } | null)?.tenantId ?? "";

  const [activeTab, setActiveTab] = useState<Tab>("feed");

  // Stats row — refresh every 60 s
  const { data: stats, isLoading: statsLoading } = useSWR(
    token && tenantId ? ["detection-stats", tenantId] : null,
    () => getDetectionStats({ token, tenantId }),
    { refreshInterval: 60_000, revalidateOnFocus: false }
  );

  // ── Stat cards config ───────────────────────────────────────────────────────

  const statCards: StatCardProps[] = [
    {
      label: "Total Incidents (24 h)",
      value: stats?.incidents_24h ?? 0,
      icon: ShieldAlert,
      decorationColor: "blue",
      loading: statsLoading,
    },
    {
      label: "Critical Alerts",
      value: stats?.critical_alerts ?? 0,
      icon: AlertTriangle,
      decorationColor: "red",
      loading: statsLoading,
    },
    {
      label: "Active Agents",
      value: stats?.active_agents ?? 0,
      icon: Cpu,
      decorationColor: "emerald",
      loading: statsLoading,
    },
    {
      label: "MTTR",
      value:
        stats?.mttr_minutes != null
          ? stats.mttr_minutes < 60
            ? `${stats.mttr_minutes}m`
            : `${(stats.mttr_minutes / 60).toFixed(1)}h`
          : "--",
      icon: Clock,
      decorationColor: "violet",
      loading: statsLoading,
    },
  ];

  return (
    <div className="space-y-6">
      {/* Page header */}
      <div className="flex items-start justify-between">
        <div>
          <Title>Detection &amp; Response</Title>
          <Text>
            Real-time threat detection, incident correlation, and response
            orchestration
          </Text>
        </div>
        <Badge color="green" className="mt-1">
          MDR Active
        </Badge>
      </div>

      {/* KPI stats row */}
      <Grid numItemsMd={2} numItemsLg={4} className="gap-4">
        {statCards.map((s) => (
          <StatCard key={s.label} {...s} />
        ))}
      </Grid>

      {/* Tab switcher */}
      <div className="border-b border-gray-200 dark:border-gray-700">
        <nav className="-mb-px flex gap-0" aria-label="Detection tabs">
          {TABS.map((tab) => {
            const active = tab.id === activeTab;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`
                  inline-flex items-center gap-2 px-5 py-3 text-sm font-medium border-b-2 transition-colors
                  ${
                    active
                      ? "border-kubric-500 text-kubric-600 dark:text-kubric-400"
                      : "border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:hover:text-gray-300"
                  }
                `}
                aria-selected={active}
                role="tab"
              >
                <tab.icon className="h-4 w-4" />
                {tab.label}
              </button>
            );
          })}
        </nav>
      </div>

      {/* Tab panels */}
      <div role="tabpanel">
        {activeTab === "feed" && <AlertFeed />}
        {activeTab === "incidents" && <IncidentTimeline />}
        {activeTab === "integrations" && <IntegrationHealth />}
      </div>
    </div>
  );
}
