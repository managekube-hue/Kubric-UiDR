import React from "react";

interface RiskItem {
  id: string;
  title: string;
  severity: "critical" | "high" | "medium" | "low";
  category: "vulnerability" | "compliance" | "detection" | "identity";
  cveId?: string;
  asset?: string;
  detectedAt: string;
}

interface RiskDashboardProps {
  tenantId: string;
  items: RiskItem[];
  onItemClick?: (id: string) => void;
}

const severityRank: Record<string, number> = {
  critical: 0,
  high: 1,
  medium: 2,
  low: 3,
};

const severityBadge: Record<string, string> = {
  critical: "bg-red-100 text-red-800 border-red-200",
  high: "bg-orange-100 text-orange-800 border-orange-200",
  medium: "bg-yellow-100 text-yellow-800 border-yellow-200",
  low: "bg-gray-100 text-gray-700 border-gray-200",
};

export default function RiskDashboard({
  tenantId,
  items,
  onItemClick,
}: RiskDashboardProps) {
  const sorted = [...items].sort(
    (a, b) => severityRank[a.severity] - severityRank[b.severity]
  );

  const counts = items.reduce<Record<string, number>>(
    (acc, i) => ({ ...acc, [i.severity]: (acc[i.severity] ?? 0) + 1 }),
    {}
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-base font-semibold text-gray-900">Risk Overview</h2>
        <span className="text-xs text-gray-400">Tenant {tenantId}</span>
      </div>

      <div className="flex gap-3">
        {(["critical", "high", "medium", "low"] as const).map((sev) => (
          <div
            key={sev}
            className={`flex-1 rounded-lg border px-3 py-2 text-center ${severityBadge[sev]}`}
          >
            <p className="text-lg font-bold">{counts[sev] ?? 0}</p>
            <p className="text-xs capitalize">{sev}</p>
          </div>
        ))}
      </div>

      <div className="rounded-lg border border-gray-200 bg-white">
        {sorted.length === 0 ? (
          <p className="p-4 text-center text-sm text-gray-400">
            No active risk items
          </p>
        ) : (
          <ul className="divide-y divide-gray-100">
            {sorted.map((item) => (
              <li
                key={item.id}
                onClick={() => onItemClick?.(item.id)}
                className="flex cursor-pointer items-start gap-3 px-4 py-3 hover:bg-gray-50"
              >
                <span
                  className={`mt-0.5 inline-flex shrink-0 rounded border px-1.5 py-0.5 text-xs font-medium ${severityBadge[item.severity]}`}
                >
                  {item.severity}
                </span>
                <div className="min-w-0 flex-1">
                  <p className="truncate text-sm font-medium text-gray-800">
                    {item.title}
                  </p>
                  <p className="mt-0.5 text-xs text-gray-400">
                    {item.category}
                    {item.cveId ? ` · ${item.cveId}` : ""}
                    {item.asset ? ` · ${item.asset}` : ""}
                    {" · "}
                    {item.detectedAt}
                  </p>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
