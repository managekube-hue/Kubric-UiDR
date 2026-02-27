import React, { useState } from "react";

type Step = "connect" | "configure" | "deploy" | "verify";

interface DeploymentWizardProps {
  tenantId: string;
  onComplete: (agentId: string) => void;
  onCancel: () => void;
}

const STEPS: Step[] = ["connect", "configure", "deploy", "verify"];

export default function DeploymentWizard({
  tenantId,
  onComplete,
  onCancel,
}: DeploymentWizardProps) {
  const [currentStep, setCurrentStep] = useState<Step>("connect");
  const [targetHost, setTargetHost] = useState("");

  const stepIndex = STEPS.indexOf(currentStep);

  function advance() {
    if (stepIndex < STEPS.length - 1) {
      setCurrentStep(STEPS[stepIndex + 1]);
    } else {
      onComplete(`agent-${tenantId}-${Date.now()}`);
    }
  }

  return (
    <div className="mx-auto max-w-lg rounded-xl border border-gray-200 bg-white p-6 shadow-md">
      <h2 className="text-lg font-semibold text-gray-900">Deploy Agent</h2>
      <p className="mt-1 text-sm text-gray-500">Tenant: {tenantId}</p>

      <ol className="mt-4 flex gap-2">
        {STEPS.map((step, i) => (
          <li
            key={step}
            className={`flex-1 rounded py-1 text-center text-xs font-medium ${
              i <= stepIndex
                ? "bg-blue-600 text-white"
                : "bg-gray-100 text-gray-400"
            }`}
          >
            {step}
          </li>
        ))}
      </ol>

      <div className="mt-6">
        {currentStep === "connect" && (
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Target host (IP or hostname)
            </label>
            <input
              type="text"
              value={targetHost}
              onChange={(e) => setTargetHost(e.target.value)}
              placeholder="192.168.1.100"
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
            />
          </div>
        )}
        {currentStep === "configure" && (
          <p className="text-sm text-gray-600">
            Agent will be configured with tenant ID <strong>{tenantId}</strong>{" "}
            and default detection profiles (EDR + NDR).
          </p>
        )}
        {currentStep === "deploy" && (
          <p className="text-sm text-gray-600">
            Deploying agent to <strong>{targetHost || "target"}</strong> via
            Ansible playbook…
          </p>
        )}
        {currentStep === "verify" && (
          <p className="text-sm text-green-600 font-medium">
            Agent deployed and reporting. Heartbeat received.
          </p>
        )}
      </div>

      <div className="mt-6 flex justify-between">
        <button
          onClick={onCancel}
          className="rounded-md border border-gray-300 px-4 py-2 text-sm text-gray-700 hover:bg-gray-50"
        >
          Cancel
        </button>
        <button
          onClick={advance}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          {stepIndex < STEPS.length - 1 ? "Next" : "Finish"}
        </button>
      </div>
    </div>
  );
}
