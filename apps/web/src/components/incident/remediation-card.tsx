import { EmptyState } from "@/components/core/empty-state";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { TechnicalId } from "@/components/core/technical-id";
import type { RemediationDetail } from "@/lib/api/types";
import type { RemediationStatus } from "@/types/sentinel";
import { formatDateTime } from "@/lib/format";
import { parameterString, remediationLabel } from "@/lib/health";
import { cn } from "@/lib/utils";
import { Wrench } from "lucide-react";

const LIFECYCLE: RemediationStatus[] = [
  "pending_approval",
  "approved",
  "running",
  "verifying",
  "succeeded",
];

type RemediationCardProps = {
  remediation?: RemediationDetail | null;
  state?: "loading" | "empty" | "error" | "success";
};

function stepReached(status: string, step: RemediationStatus): boolean {
  if (status === "rejected" || status === "cancelled" || status === "failed") {
    return step === "pending_approval";
  }
  const current = LIFECYCLE.indexOf(status as RemediationStatus);
  const target = LIFECYCLE.indexOf(step);
  if (current === -1 || target === -1) return false;
  return target <= current;
}

export function RemediationCard({
  remediation,
  state = remediation ? "success" : "empty",
}: RemediationCardProps) {
  if (state === "loading") {
    return <LoadingState label="Loading remediation" />;
  }
  if (state === "error") {
    return (
      <ErrorState
        title="Remediation unavailable"
        description="The control plane did not return a remediation for this incident."
      />
    );
  }
  if (!remediation) {
    return (
      <EmptyState
        icon={Wrench}
        title="No remediation in progress"
        description="Approved allowlisted actions and verification results appear here."
      />
    );
  }

  const fromVersion = parameterString(remediation.parameters, "from_version");
  const toVersion =
    parameterString(remediation.parameters, "to_version") ?? parameterString(remediation.parameters, "version_hint");
  const terminalFailed = remediation.status === "failed" || remediation.status === "rejected" || remediation.status === "cancelled";
  const steps: Array<{ id: RemediationStatus; label: string }> = [
    { id: "pending_approval", label: "Pending approval" },
    { id: "approved", label: "Approved" },
    { id: "running", label: "Running" },
    { id: "verifying", label: "Verifying" },
    { id: "succeeded", label: terminalFailed ? remediationLabel(remediation.status) : "Succeeded" },
  ];

  return (
    <section aria-labelledby="remediation-heading" className="space-y-3 rounded-md border border-border px-4 py-3">
      <div className="flex items-center justify-between gap-2">
        <h2 id="remediation-heading" className="type-label">
          Remediation
        </h2>
        <p className="type-meta capitalize">{remediationLabel(remediation.status)}</p>
      </div>

      <ol className="flex flex-wrap gap-2" aria-label="Remediation lifecycle">
        {steps.map((step, index) => {
          const active = remediation.status === step.id || (step.id === "succeeded" && terminalFailed);
          const reached = stepReached(remediation.status, step.id) || active;
          return (
            <li key={step.id} className="flex items-center gap-2">
              <span
                className={cn(
                  "rounded-sm px-2 py-1 text-[11px]",
                  active && terminalFailed ? "bg-danger/15 text-danger" : null,
                  active && !terminalFailed ? "bg-brand/15 text-brand" : null,
                  !active && reached ? "text-text-primary" : null,
                  !active && !reached ? "text-text-muted" : null,
                )}
              >
                {step.label}
              </span>
              {index < steps.length - 1 ? (
                <span className="type-meta" aria-hidden>
                  →
                </span>
              ) : null}
            </li>
          );
        })}
      </ol>

      <p className="text-[13px] font-medium tracking-tight capitalize">
        {remediation.action_type.replaceAll("_", " ")}
      </p>
      <div className="flex flex-wrap gap-x-4 gap-y-1">
        <span className="type-meta">
          Target <TechnicalId value={remediation.service_id} />
        </span>
        {fromVersion ? (
          <span className="type-meta">
            From <TechnicalId value={fromVersion} />
          </span>
        ) : null}
        {toVersion ? (
          <span className="type-meta">
            To <TechnicalId value={toVersion} />
          </span>
        ) : null}
      </div>
      <p className="type-meta">
        Approver{" "}
        {remediation.approval?.actor_user_id ? (
          <TechnicalId value={remediation.approval.actor_user_id} />
        ) : (
          "not recorded"
        )}
        {remediation.approval?.decided_at ? ` · ${formatDateTime(remediation.approval.decided_at)}` : null}
      </p>
      <p className="type-meta">
        Started {remediation.started_at ? formatDateTime(remediation.started_at) : "not started"}
        {remediation.completed_at ? ` · Completed ${formatDateTime(remediation.completed_at)}` : ""}
      </p>
      <p className="type-meta">Verification {remediation.verification_status.replaceAll("_", " ")}</p>
      {remediation.result_summary ? <p className="type-body">{remediation.result_summary}</p> : null}
      {remediation.error_message ? <p className="type-meta text-danger">{remediation.error_message}</p> : null}
    </section>
  );
}
