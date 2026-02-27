import React from "react";

interface SubScore {
  label: string;
  value: number;   // 0–100
  weight: number;  // fraction, should sum to 1.0
}

interface KissScorecardProps {
  tenantId: string;
  overallScore: number;  // 0–100
  subScores: SubScore[];
  generatedAt: string;
}

function scoreColor(v: number): string {
  if (v >= 80) return "text-green-600";
  if (v >= 60) return "text-yellow-600";
  return "text-red-600";
}

function barColor(v: number): string {
  if (v >= 80) return "bg-green-500";
  if (v >= 60) return "bg-yellow-500";
  return "bg-red-500";
}

export default function KissScorecard({
  tenantId,
  overallScore,
  subScores,
  generatedAt,
}: KissScorecardProps) {
  return (
    <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-base font-semibold text-gray-900">
            KiSS Score
          </h3>
          <p className="text-xs text-gray-400">Tenant {tenantId}</p>
        </div>
        <div className={`text-4xl font-bold ${scoreColor(overallScore)}`}>
          {overallScore}
        </div>
      </div>

      <p className="mt-1 text-xs text-gray-400">Generated {generatedAt}</p>

      <ul className="mt-5 space-y-3">
        {subScores.map((s) => (
          <li key={s.label}>
            <div className="flex justify-between text-xs">
              <span className="text-gray-700">{s.label}</span>
              <span className={`font-medium ${scoreColor(s.value)}`}>
                {s.value}
              </span>
            </div>
            <div className="mt-1 h-1.5 w-full rounded-full bg-gray-100">
              <div
                className={`h-1.5 rounded-full ${barColor(s.value)}`}
                style={{ width: `${s.value}%` }}
              />
            </div>
          </li>
        ))}
      </ul>

      <p className="mt-4 text-xs text-gray-400">
        Weighted: vuln 30% · compliance 25% · detection 25% · response 20%
      </p>
    </div>
  );
}
