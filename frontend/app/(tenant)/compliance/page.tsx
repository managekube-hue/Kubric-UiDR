"use client";

import { useSession } from "next-auth/react";
import { useEffect, useState } from "react";
import {
  Card,
  Title,
  Text,
  Badge,
  Grid,
  Flex,
  Metric,
  ProgressBar,
} from "@tremor/react";
import { listAssessments, type Assessment } from "@/lib/api-client";

const statusColor = (s: string) => {
  switch (s.toLowerCase()) {
    case "passed":
      return "emerald";
    case "failed":
      return "red";
    case "in_progress":
      return "yellow";
    default:
      return "gray";
  }
};

const frameworkLabels: Record<string, string> = {
  "nist-csf": "NIST CSF 2.0",
  "iso-27001": "ISO 27001:2022",
  "soc2": "SOC 2 Type II",
  "pci-dss": "PCI DSS v4.0",
  "hipaa": "HIPAA",
  "cis": "CIS Benchmarks",
};

export default function CompliancePage() {
  const { data: session } = useSession();
  const [assessments, setAssessments] = useState<Assessment[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!session?.accessToken) return;
    setLoading(true);
    listAssessments({ token: session.accessToken, tenantId: session.tenantId })
      .then(setAssessments)
      .catch(() => setAssessments([]))
      .finally(() => setLoading(false));
  }, [session?.accessToken, session?.tenantId]);

  return (
    <div className="space-y-6">
      <div>
        <Title>Compliance Assessments</Title>
        <Text>Framework compliance status and gap analysis</Text>
      </div>

      {loading ? (
        <Text>Loading assessments...</Text>
      ) : assessments.length === 0 ? (
        <Card>
          <Text>No compliance assessments found. Create one to get started.</Text>
        </Card>
      ) : (
        <Grid numItemsMd={2} numItemsLg={3} className="gap-4">
          {assessments.map((a) => (
            <Card key={a.id}>
              <Flex alignItems="start">
                <div className="space-y-1">
                  <Text className="font-semibold">
                    {frameworkLabels[a.framework] ?? a.framework}
                  </Text>
                  <Badge color={statusColor(a.status)}>{a.status}</Badge>
                </div>
              </Flex>
              <Metric className="mt-4">{a.score}%</Metric>
              <ProgressBar
                value={a.score}
                className="mt-2"
                color={a.score >= 80 ? "emerald" : a.score >= 50 ? "yellow" : "red"}
              />
              <Flex className="mt-4">
                <Text className="text-xs text-gray-500">
                  {a.findings_count} finding{a.findings_count !== 1 ? "s" : ""}
                </Text>
                <Text className="text-xs text-gray-400">
                  {new Date(a.updated_at).toLocaleDateString()}
                </Text>
              </Flex>
            </Card>
          ))}
        </Grid>
      )}
    </div>
  );
}
