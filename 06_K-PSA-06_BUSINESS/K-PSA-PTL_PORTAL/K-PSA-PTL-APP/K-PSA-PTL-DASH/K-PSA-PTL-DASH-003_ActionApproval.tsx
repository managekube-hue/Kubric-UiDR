import React, { useState } from "react";

interface RemediationStep {
  id: string;
  description: string;
  riskLevel: "low" | "medium" | "high";
  ansibleTask?: string;
}

interface ActionApprovalProps {
  planId: string;
  tenantId: string;
  findingId: string;
  steps: RemediationStep[];
  onApprove: (planId: string) => void;
  onReject: (planId: string, reason: string) => void;
}

const riskBadge: Record<string, string> = {
  low: "bg-green-100 text-green-800",
  medium: "bg-yellow-100 text-yellow-800",
  high: "bg-red-100 text-red-800",
};

export default function ActionApproval({
  planId,
  tenantId,
  findingId,
  steps,
  onApprove,
  onReject,
}: ActionApprovalProps) {
  const [rejectReason, setRejectReason] = useState("");
  const [showRejectForm, setShowRejectForm] = useState(false);

  return (
    <div className="rounded-xl border border-amber-200 bg-amber-50 p-5">
      <div className="flex items-start justify-between">
        <div>
          <h3 className="text-base font-semibold text-amber-900">
            Remediation Approval Required
          </h3>
          <p className="mt-1 text-xs text-amber-700">
            Plan {planId.slice(0, 8)} · Finding {findingId.slice(0, 8)} ·
            Tenant {tenantId}
          </p>
        </div>
        <span className="rounded-full bg-amber-200 px-2 py-0.5 text-xs font-medium text-amber-900">
          {steps.length} step{steps.length !== 1 ? "s" : ""}
        </span>
      </div>

      <ul className="mt-4 space-y-2">
        {steps.map((step) => (
          <li
            key={step.id}
            className="flex items-center gap-3 rounded-lg bg-white px-3 py-2 text-sm"
          >
            <span
              className={`rounded px-1.5 py-0.5 text-xs font-medium ${riskBadge[step.riskLevel]}`}
            >
              {step.riskLevel}
            </span>
            <span className="flex-1 text-gray-800">{step.description}</span>
            {step.ansibleTask && (
              <span className="font-mono text-xs text-gray-400">
                {step.ansibleTask}
              </span>
            )}
          </li>
        ))}
      </ul>

      {showRejectForm ? (
        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700">
            Rejection reason
          </label>
          <textarea
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            rows={2}
            className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
          />
          <div className="mt-2 flex gap-2">
            <button
              onClick={() => onReject(planId, rejectReason)}
              className="rounded-md bg-red-600 px-3 py-1.5 text-sm text-white hover:bg-red-700"
            >
              Confirm Reject
            </button>
            <button
              onClick={() => setShowRejectForm(false)}
              className="rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-700"
            >
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <div className="mt-4 flex gap-3">
          <button
            onClick={() => onApprove(planId)}
            className="rounded-md bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700"
          >
            Approve & Execute
          </button>
          <button
            onClick={() => setShowRejectForm(true)}
            className="rounded-md border border-red-300 px-4 py-2 text-sm font-medium text-red-700 hover:bg-red-50"
          >
            Reject
          </button>
        </div>
      )}
    </div>
  );
}
