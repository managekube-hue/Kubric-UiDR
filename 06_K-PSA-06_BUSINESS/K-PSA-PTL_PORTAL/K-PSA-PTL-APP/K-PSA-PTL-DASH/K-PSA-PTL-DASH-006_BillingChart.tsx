import React from "react";

interface BillingPeriod {
  month: string;       // e.g. "2025-11"
  amountCents: number;
  invoiceId?: string;
  status: "paid" | "pending" | "overdue";
}

interface BillingChartProps {
  tenantId: string;
  periods: BillingPeriod[];
  currency?: string;
}

const statusBadge: Record<string, string> = {
  paid: "bg-green-100 text-green-700",
  pending: "bg-yellow-100 text-yellow-700",
  overdue: "bg-red-100 text-red-700",
};

function formatAmount(cents: number, currency = "USD"): string {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
  }).format(cents / 100);
}

export default function BillingChart({
  tenantId,
  periods,
  currency = "USD",
}: BillingChartProps) {
  const maxCents = Math.max(...periods.map((p) => p.amountCents), 1);

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5">
      <div className="flex items-center justify-between">
        <h3 className="text-base font-semibold text-gray-900">Billing</h3>
        <span className="text-xs text-gray-400">Tenant {tenantId}</span>
      </div>

      <div className="mt-4 flex items-end gap-2">
        {periods.map((p) => (
          <div key={p.month} className="flex flex-1 flex-col items-center gap-1">
            <div
              className="w-full rounded-t bg-blue-500"
              style={{ height: `${(p.amountCents / maxCents) * 80}px` }}
              title={formatAmount(p.amountCents, currency)}
            />
            <span className="text-xs text-gray-500">
              {p.month.slice(5)}
            </span>
          </div>
        ))}
      </div>

      <ul className="mt-4 space-y-2">
        {[...periods].reverse().map((p) => (
          <li
            key={p.month}
            className="flex items-center justify-between text-sm"
          >
            <span className="text-gray-700">{p.month}</span>
            <div className="flex items-center gap-2">
              <span className="font-medium text-gray-900">
                {formatAmount(p.amountCents, currency)}
              </span>
              <span
                className={`rounded px-1.5 py-0.5 text-xs font-medium ${statusBadge[p.status]}`}
              >
                {p.status}
              </span>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
