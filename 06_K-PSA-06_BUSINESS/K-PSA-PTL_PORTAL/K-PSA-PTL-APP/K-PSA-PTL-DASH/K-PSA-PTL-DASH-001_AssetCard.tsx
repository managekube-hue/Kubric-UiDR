import React from "react";

interface AssetCardProps {
  assetId: string;
  hostname: string;
  agentType: string;
  status: "online" | "offline" | "degraded";
  lastSeen: string;
  tenantId: string;
}

export default function AssetCard({
  assetId,
  hostname,
  agentType,
  status,
  lastSeen,
}: AssetCardProps) {
  const statusColor =
    status === "online"
      ? "bg-green-500"
      : status === "degraded"
      ? "bg-yellow-500"
      : "bg-red-500";

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-900">{hostname}</p>
          <p className="text-xs text-gray-500">{agentType}</p>
        </div>
        <span
          className={`inline-flex items-center rounded-full px-2 py-1 text-xs font-medium text-white ${statusColor}`}
        >
          {status}
        </span>
      </div>
      <div className="mt-2 text-xs text-gray-400">
        <span>Last seen: {lastSeen}</span>
        <span className="ml-2 text-gray-300">|</span>
        <span className="ml-2 font-mono">{assetId.slice(0, 8)}…</span>
      </div>
    </div>
  );
}
