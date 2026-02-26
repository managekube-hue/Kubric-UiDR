"use client";

import { useSession } from "next-auth/react";
import { useEffect, useState } from "react";
import {
  Card,
  Title,
  Text,
  Table,
  TableHead,
  TableHeaderCell,
  TableBody,
  TableRow,
  TableCell,
  Badge,
  Grid,
  Metric,
  Flex,
} from "@tremor/react";
import { Cpu, Wifi, WifiOff, Clock } from "lucide-react";
import { listAgents, type Agent } from "@/lib/api-client";

function agentStatus(lastSeen: string): {
  label: string;
  color: "emerald" | "yellow" | "red";
  icon: React.ComponentType<{ className?: string }>;
} {
  const elapsed = Date.now() - new Date(lastSeen).getTime();
  const minutes = elapsed / 60_000;
  if (minutes < 5)
    return { label: "Healthy", color: "emerald", icon: Wifi };
  if (minutes < 30)
    return { label: "Stale", color: "yellow", icon: Clock };
  return { label: "Offline", color: "red", icon: WifiOff };
}

const typeLabel: Record<string, string> = {
  coresec: "CoreSec EDR",
  netguard: "NetGuard NDR",
  perftrace: "PerfTrace APM",
};

export default function AgentsPage() {
  const { data: session } = useSession();
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!session?.accessToken) return;
    setLoading(true);
    listAgents({ token: session.accessToken, tenantId: session.tenantId })
      .then(setAgents)
      .catch(() => setAgents([]))
      .finally(() => setLoading(false));
  }, [session?.accessToken, session?.tenantId]);

  const healthy = agents.filter(
    (a) => agentStatus(a.last_seen_at).label === "Healthy"
  ).length;
  const stale = agents.filter(
    (a) => agentStatus(a.last_seen_at).label === "Stale"
  ).length;
  const offline = agents.filter(
    (a) => agentStatus(a.last_seen_at).label === "Offline"
  ).length;

  return (
    <div className="space-y-6">
      <div>
        <Title>Agent Health</Title>
        <Text>Enrolled agent status and heartbeat monitoring</Text>
      </div>

      {/* Summary cards */}
      <Grid numItemsMd={3} className="gap-4">
        <Card decoration="top" decorationColor="emerald">
          <Flex alignItems="start">
            <div>
              <Text>Healthy</Text>
              <Metric>{healthy}</Metric>
            </div>
            <Wifi className="h-6 w-6 text-emerald-500" />
          </Flex>
        </Card>
        <Card decoration="top" decorationColor="yellow">
          <Flex alignItems="start">
            <div>
              <Text>Stale</Text>
              <Metric>{stale}</Metric>
            </div>
            <Clock className="h-6 w-6 text-yellow-500" />
          </Flex>
        </Card>
        <Card decoration="top" decorationColor="red">
          <Flex alignItems="start">
            <div>
              <Text>Offline</Text>
              <Metric>{offline}</Metric>
            </div>
            <WifiOff className="h-6 w-6 text-red-500" />
          </Flex>
        </Card>
      </Grid>

      {/* Agent table */}
      <Card>
        {loading ? (
          <Text>Loading agents...</Text>
        ) : (
          <Table>
            <TableHead>
              <TableRow>
                <TableHeaderCell>Hostname</TableHeaderCell>
                <TableHeaderCell>Type</TableHeaderCell>
                <TableHeaderCell>Version</TableHeaderCell>
                <TableHeaderCell>Status</TableHeaderCell>
                <TableHeaderCell>Last Seen</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {agents.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5}>
                    <Text className="text-center">
                      No agents enrolled. Deploy an agent to get started.
                    </Text>
                  </TableCell>
                </TableRow>
              ) : (
                agents.map((a) => {
                  const status = agentStatus(a.last_seen_at);
                  return (
                    <TableRow key={a.id}>
                      <TableCell>
                        <Flex className="gap-2">
                          <Cpu className="h-4 w-4 text-gray-400" />
                          <span className="font-medium">{a.hostname}</span>
                        </Flex>
                      </TableCell>
                      <TableCell>
                        {typeLabel[a.agent_type] ?? a.agent_type}
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-sm">{a.version}</span>
                      </TableCell>
                      <TableCell>
                        <Badge color={status.color}>{status.label}</Badge>
                      </TableCell>
                      <TableCell>
                        {new Date(a.last_seen_at).toLocaleString()}
                      </TableCell>
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
}
