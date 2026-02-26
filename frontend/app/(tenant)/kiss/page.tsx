"use client";

import { useSession } from "next-auth/react";
import { useEffect, useState } from "react";
import {
  Card,
  Title,
  Text,
  Metric,
  DonutChart,
  Flex,
  Grid,
  Badge,
} from "@tremor/react";
import {
  Fingerprint,
  Monitor,
  Network,
  Cloud,
  ShieldCheck,
} from "lucide-react";
import { getKissScores, type KissScores } from "@/lib/api-client";

interface DomainConfig {
  key: keyof Omit<KissScores, "tenant_id" | "overall" | "last_updated">;
  label: string;
  icon: React.ComponentType<{ className?: string }>;
  description: string;
}

const domains: DomainConfig[] = [
  {
    key: "identity",
    label: "Identity",
    icon: Fingerprint,
    description: "IAM posture, MFA coverage, privilege creep",
  },
  {
    key: "endpoint",
    label: "Endpoint",
    icon: Monitor,
    description: "EDR coverage, patching cadence, encryption",
  },
  {
    key: "network",
    label: "Network",
    icon: Network,
    description: "Segmentation, firewall rules, DNS security",
  },
  {
    key: "cloud",
    label: "Cloud",
    icon: Cloud,
    description: "CSP posture, misconfiguration, public exposure",
  },
  {
    key: "compliance",
    label: "Compliance",
    icon: ShieldCheck,
    description: "Framework adherence, policy gaps, audit readiness",
  },
];

function scoreColor(score: number): string {
  if (score >= 80) return "emerald";
  if (score >= 60) return "yellow";
  return "red";
}

function scoreLabel(score: number): string {
  if (score >= 90) return "Excellent";
  if (score >= 80) return "Good";
  if (score >= 60) return "Fair";
  if (score >= 40) return "Poor";
  return "Critical";
}

export default function KissPage() {
  const { data: session } = useSession();
  const [scores, setScores] = useState<KissScores | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!session?.accessToken) return;
    setLoading(true);
    getKissScores({ token: session.accessToken, tenantId: session.tenantId })
      .then(setScores)
      .catch(() => setScores(null))
      .finally(() => setLoading(false));
  }, [session?.accessToken, session?.tenantId]);

  // Fallback demo scores when API unreachable
  const data: KissScores = scores ?? {
    tenant_id: session?.tenantId ?? "",
    identity: 0,
    endpoint: 0,
    network: 0,
    cloud: 0,
    compliance: 0,
    overall: 0,
    last_updated: new Date().toISOString(),
  };

  return (
    <div className="space-y-6">
      <div>
        <Title>Kubric Integrated Security Score (KiSS)</Title>
        <Text>
          Composite security posture across 5 domains. Updated{" "}
          {data.last_updated
            ? new Date(data.last_updated).toLocaleString()
            : "—"}
        </Text>
      </div>

      {/* Overall score */}
      <Card className="max-w-sm mx-auto text-center">
        <Text>Overall Score</Text>
        <Metric className="text-5xl mt-2">{data.overall}</Metric>
        <Badge color={scoreColor(data.overall)} className="mt-2">
          {scoreLabel(data.overall)}
        </Badge>
        <DonutChart
          className="mt-4 h-40"
          data={domains.map((d) => ({
            name: d.label,
            value: data[d.key] as number,
          }))}
          category="value"
          index="name"
          colors={["blue", "cyan", "violet", "amber", "emerald"]}
          showAnimation
        />
      </Card>

      {/* Domain score cards */}
      <Grid numItemsMd={2} numItemsLg={3} className="gap-4">
        {domains.map((d) => {
          const val = data[d.key] as number;
          return (
            <Card key={d.key}>
              <Flex alignItems="start">
                <div>
                  <Text className="font-semibold">{d.label}</Text>
                  <Text className="text-xs text-gray-500 mt-1">
                    {d.description}
                  </Text>
                </div>
                <d.icon className="h-6 w-6 text-gray-400" />
              </Flex>
              <Flex className="mt-4" alignItems="end">
                <Metric>{val}</Metric>
                <Badge color={scoreColor(val)}>{scoreLabel(val)}</Badge>
              </Flex>
              <DonutChart
                className="mt-4 h-28"
                data={[
                  { name: "Score", value: val },
                  { name: "Gap", value: 100 - val },
                ]}
                category="value"
                index="name"
                colors={[scoreColor(val), "gray"]}
                showAnimation
                showLabel={false}
              />
            </Card>
          );
        })}
      </Grid>

      {loading && (
        <Text className="text-center text-gray-400">
          Loading live scores from API...
        </Text>
      )}
    </div>
  );
}
