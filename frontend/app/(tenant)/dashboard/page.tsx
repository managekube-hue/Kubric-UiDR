"use client";

import { useSession } from "next-auth/react";
import { useEffect, useState, useCallback } from "react";
import {
  Card,
  Title,
  Text,
  AreaChart,
  Metric,
  Flex,
  Badge,
  Grid,
} from "@tremor/react";
import { AlertTriangle, Bug, ShieldCheck, Cpu } from "lucide-react";
import { subscribeAlerts, type NatsAlert } from "@/lib/nats-client";

interface AlertStat {
  critical: number;
  high: number;
  medium: number;
  low: number;
}

interface TimePoint {
  time: string;
  Critical: number;
  High: number;
  Medium: number;
}

export default function DashboardPage() {
  const { data: session } = useSession();
  const [alerts, setAlerts] = useState<NatsAlert[]>([]);
  const [stats, setStats] = useState<AlertStat>({
    critical: 0,
    high: 0,
    medium: 0,
    low: 0,
  });
  const [chartData, setChartData] = useState<TimePoint[]>([]);

  const handleAlert = useCallback((alert: NatsAlert) => {
    setAlerts((prev) => [alert, ...prev].slice(0, 100));
    setStats((prev) => {
      const sev = alert.severity?.toLowerCase() ?? "low";
      return { ...prev, [sev]: (prev[sev as keyof AlertStat] ?? 0) + 1 };
    });
    setChartData((prev) => {
      const now = new Date().toLocaleTimeString("en-US", {
        hour: "2-digit",
        minute: "2-digit",
      });
      const last = prev[prev.length - 1];
      if (last?.time === now) {
        const updated = { ...last };
        const key = (alert.severity?.charAt(0).toUpperCase() +
          alert.severity?.slice(1).toLowerCase()) as keyof TimePoint;
        if (key in updated && typeof updated[key] === "number") {
          (updated[key] as number) += 1;
        }
        return [...prev.slice(0, -1), updated];
      }
      return [
        ...prev.slice(-23),
        {
          time: now,
          Critical: alert.severity === "critical" ? 1 : 0,
          High: alert.severity === "high" ? 1 : 0,
          Medium: alert.severity === "medium" ? 1 : 0,
        },
      ];
    });
  }, []);

  useEffect(() => {
    if (!session?.tenantId) return;
    let sub: Awaited<ReturnType<typeof subscribeAlerts>> | undefined;

    subscribeAlerts(session.tenantId, handleAlert)
      .then((s) => {
        sub = s;
      })
      .catch(() => {
        // NATS not available — silent fallback
      });

    return () => {
      sub?.unsubscribe();
    };
  }, [session?.tenantId, handleAlert]);

  const statCards = [
    {
      label: "Critical",
      value: stats.critical,
      icon: AlertTriangle,
      color: "red" as const,
    },
    { label: "High", value: stats.high, icon: Bug, color: "orange" as const },
    {
      label: "Medium",
      value: stats.medium,
      icon: ShieldCheck,
      color: "yellow" as const,
    },
    { label: "Agents Online", value: "--", icon: Cpu, color: "emerald" as const },
  ];

  return (
    <div className="space-y-6">
      <div>
        <Title>Security Dashboard</Title>
        <Text>Real-time threat monitoring and alert triage</Text>
      </div>

      {/* KPI cards */}
      <Grid numItemsMd={2} numItemsLg={4} className="gap-4">
        {statCards.map((s) => (
          <Card key={s.label} decoration="top" decorationColor={s.color}>
            <Flex alignItems="start">
              <div>
                <Text>{s.label}</Text>
                <Metric>{s.value}</Metric>
              </div>
              <s.icon className="h-6 w-6 text-gray-400" />
            </Flex>
          </Card>
        ))}
      </Grid>

      {/* Alert trend chart */}
      <Card>
        <Title>Alert Trend (Live)</Title>
        <AreaChart
          className="mt-4 h-72"
          data={chartData.length > 0 ? chartData : [{ time: "Now", Critical: 0, High: 0, Medium: 0 }]}
          index="time"
          categories={["Critical", "High", "Medium"]}
          colors={["red", "orange", "yellow"]}
          showAnimation
        />
      </Card>

      {/* Recent alerts feed */}
      <Card>
        <Title>Recent Alerts</Title>
        <div className="mt-4 space-y-2 max-h-96 overflow-auto">
          {alerts.length === 0 ? (
            <Text>Waiting for events. Connect NATS WebSocket to see live alerts.</Text>
          ) : (
            alerts.map((a, i) => (
              <div
                key={`${a.timestamp}-${i}`}
                className="flex items-center gap-3 rounded-md border p-3"
              >
                <Badge
                  color={
                    a.severity === "critical"
                      ? "red"
                      : a.severity === "high"
                        ? "orange"
                        : "yellow"
                  }
                >
                  {a.severity}
                </Badge>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium truncate">{a.title}</p>
                  <p className="text-xs text-gray-500">{a.source}</p>
                </div>
                <span className="text-xs text-gray-400 whitespace-nowrap">
                  {new Date(a.timestamp).toLocaleTimeString()}
                </span>
              </div>
            ))
          )}
        </div>
      </Card>
    </div>
  );
}
