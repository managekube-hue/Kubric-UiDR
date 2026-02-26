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
  TextInput,
} from "@tremor/react";
import { Search } from "lucide-react";
import { listFindings, type Finding } from "@/lib/api-client";

const severityColor = (s: string) => {
  switch (s.toLowerCase()) {
    case "critical":
      return "red";
    case "high":
      return "orange";
    case "medium":
      return "yellow";
    case "low":
      return "emerald";
    default:
      return "gray";
  }
};

const statusColor = (s: string) => {
  switch (s.toLowerCase()) {
    case "open":
      return "red";
    case "in_progress":
      return "yellow";
    case "resolved":
      return "emerald";
    default:
      return "gray";
  }
};

export default function VulnsPage() {
  const { data: session } = useSession();
  const [findings, setFindings] = useState<Finding[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!session?.accessToken) return;
    setLoading(true);
    listFindings({ token: session.accessToken, tenantId: session.tenantId })
      .then(setFindings)
      .catch(() => setFindings([]))
      .finally(() => setLoading(false));
  }, [session?.accessToken, session?.tenantId]);

  const filtered = findings.filter(
    (f) =>
      f.title.toLowerCase().includes(search.toLowerCase()) ||
      f.cve_id?.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="space-y-6">
      <div>
        <Title>Vulnerability Findings</Title>
        <Text>All discovered vulnerabilities with EPSS enrichment</Text>
      </div>

      <div className="max-w-md">
        <TextInput
          icon={Search}
          placeholder="Search CVE or title..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <Card>
        {loading ? (
          <Text>Loading findings...</Text>
        ) : (
          <Table>
            <TableHead>
              <TableRow>
                <TableHeaderCell>CVE</TableHeaderCell>
                <TableHeaderCell>Title</TableHeaderCell>
                <TableHeaderCell>Severity</TableHeaderCell>
                <TableHeaderCell>Status</TableHeaderCell>
                <TableHeaderCell>EPSS Score</TableHeaderCell>
                <TableHeaderCell>Source</TableHeaderCell>
                <TableHeaderCell>Created</TableHeaderCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filtered.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7}>
                    <Text className="text-center">No findings.</Text>
                  </TableCell>
                </TableRow>
              ) : (
                filtered.map((f) => (
                  <TableRow key={f.id}>
                    <TableCell>
                      <span className="font-mono text-sm">{f.cve_id || "N/A"}</span>
                    </TableCell>
                    <TableCell>{f.title}</TableCell>
                    <TableCell>
                      <Badge color={severityColor(f.severity)}>{f.severity}</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge color={statusColor(f.status)}>{f.status}</Badge>
                    </TableCell>
                    <TableCell>
                      {f.epss_score != null
                        ? `${(f.epss_score * 100).toFixed(1)}%`
                        : "—"}
                    </TableCell>
                    <TableCell>{f.source}</TableCell>
                    <TableCell>
                      {new Date(f.created_at).toLocaleDateString()}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
}
