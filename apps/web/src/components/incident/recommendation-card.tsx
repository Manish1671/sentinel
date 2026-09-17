"use client";

import { useState, type ReactNode } from "react";
import { RecommendationBlock } from "@/components/ai/recommendation-block";
import { ConfirmationDialog } from "@/components/core/confirmation-dialog";
import { EmptyState } from "@/components/core/empty-state";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { TechnicalId } from "@/components/core/technical-id";
import { Button } from "@/components/ui/button";
import { approveRemediation, rejectRemediation } from "@/lib/api/remediations";
import type { RecommendationDetail, RemediationDetail } from "@/lib/api/types";
import { ApiError } from "@/lib/api/errors";
import { meetsApprovalRole } from "@/lib/auth/roles";
import { errorMessage, newIdempotencyKey, operatorParameters, parameterString } from "@/lib/health";
import { Lightbulb } from "lucide-react";

type RecommendationCardProps = {
  recommendation?: RecommendationDetail | null;
  remediation?: RemediationDetail | null;
  role?: string;
  onChanged?: () => void;
  state?: "loading" | "empty" | "error" | "success";
};

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <dt className="type-label">{label}</dt>
      <dd className="text-[13px] text-text-primary">{children}</dd>
    </div>
  );
}

export function RecommendationCard({
  recommendation,
  remediation,
  role,
  onChanged,
  state = recommendation ? "success" : "empty",
}: RecommendationCardProps) {
  const [open, setOpen] = useState(false);
  const [pending, setPending] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  if (state === "loading") {
    return <LoadingState label="Loading recommendation" />;
  }
  if (state === "error") {
    return (
      <ErrorState
        title="Recommendation unavailable"
        description="The control plane did not return a recommendation for this incident."
      />
    );
  }
  if (!recommendation) {
    return (
      <EmptyState
        icon={Lightbulb}
        title="No recommendation yet"
        description="A recommendation appears after a completed investigation proposes an allowlisted action."
      />
    );
  }

  const fromVersion = parameterString(recommendation.parameters, "from_version");
  const toVersion =
    parameterString(recommendation.parameters, "to_version") ??
    parameterString(recommendation.parameters, "version_hint");
  const extraParams = operatorParameters(recommendation.parameters).filter(
    (item) => item.key !== "from_version" && item.key !== "to_version" && item.key !== "version_hint",
  );
  const canApprove = meetsApprovalRole(role, recommendation.required_approval_role);
  const pendingApproval = remediation?.status === "pending_approval";
  const showControls = Boolean(remediation) && pendingApproval;

  async function decide(approve: boolean) {
    if (!remediation) return;
    setPending(true);
    setActionError(null);
    try {
      const options = {
        idempotencyKey: newIdempotencyKey(approve ? "approve" : "reject"),
        comment: approve ? "Approved from Sentinel operator console." : "Rejected from Sentinel operator console.",
      };
      if (approve) {
        await approveRemediation(remediation.id, options);
      } else {
        await rejectRemediation(remediation.id, options);
      }
      setOpen(false);
      onChanged?.();
    } catch (error) {
      if (error instanceof ApiError && error.status === 403) {
        setActionError("You are not allowed to approve this remediation. Backend authorization is authoritative.");
      } else {
        setActionError(errorMessage(error, "The approval request failed. The control plane did not change state."));
      }
    } finally {
      setPending(false);
    }
  }

  return (
    <RecommendationBlock>
      <p className="text-[15px] font-medium tracking-tight text-text-primary">{recommendation.title}</p>
      <p className="type-meta capitalize">
        Status {recommendation.status.replaceAll("_", " ")} · Action {recommendation.action_type.replaceAll("_", " ")}
      </p>
      <dl className="grid grid-cols-2 gap-3">
        <Field label="Action">
          <span className="capitalize">{recommendation.action_type.replaceAll("_", " ")}</span>
        </Field>
        <Field label="Target">
          <TechnicalId value={recommendation.target_service_id} />
        </Field>
        <Field label="From">{fromVersion ? <TechnicalId value={fromVersion} /> : <span className="type-meta">Unavailable</span>}</Field>
        <Field label="To">{toVersion ? <TechnicalId value={toVersion} /> : <span className="type-meta">Unavailable</span>}</Field>
        <Field label="Risk">
          <span className="capitalize">{recommendation.risk_level}</span>
        </Field>
        <Field label="Required role">
          <span className="capitalize">{recommendation.required_approval_role}</span>
        </Field>
      </dl>
      {extraParams.length ? (
        <dl className="grid gap-2">
          {extraParams.map((item) => (
            <Field key={item.key} label={item.key.replaceAll("_", " ")}>
              <TechnicalId value={item.value} />
            </Field>
          ))}
        </dl>
      ) : null}
      <div>
        <p className="type-label mb-1">Why Sentinel recommends this</p>
        <p className="type-body">{recommendation.rationale}</p>
      </div>
      {recommendation.investigation_id ? (
        <p className="type-meta">
          Evidence references investigation <TechnicalId value={recommendation.investigation_id} />
        </p>
      ) : (
        <p className="type-meta">No investigation reference stored on this recommendation.</p>
      )}
      {showControls ? (
        canApprove ? (
          <>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Button
                variant="outline"
                className="mt-1 flex-1 border-brand/40 bg-brand/10 text-brand hover:bg-brand/15"
                onClick={() => setOpen(true)}
              >
                Review remediation
              </Button>
              <Button variant="outline" disabled={pending} onClick={() => void decide(false)}>
                Reject
              </Button>
            </div>
            <ConfirmationDialog
              open={open}
              onOpenChange={setOpen}
              title="Approve production remediation"
              description="This records approval in the Go control plane. The remediation service then executes the allowlisted action. Success is not assumed until the backend returns the new state."
              confirmLabel={pending ? "Submitting…" : "Approve"}
              cancelLabel="Cancel"
              destructive
              pending={pending}
              onConfirm={() => void decide(true)}
            >
              <dl className="grid gap-3 border-y border-border py-3">
                <Field label="Action">
                  <span className="capitalize">{recommendation.action_type.replaceAll("_", " ")}</span>
                </Field>
                <Field label="Target">
                  <TechnicalId value={recommendation.target_service_id} />
                </Field>
                <Field label="Risk">
                  <span className="capitalize">{recommendation.risk_level}</span>
                </Field>
                <Field label="Required approval role">
                  <span className="capitalize">{recommendation.required_approval_role}</span>
                </Field>
                <Field label="Why Sentinel recommends it">
                  <span className="type-body">{recommendation.rationale}</span>
                </Field>
                <Field label="Expected effect">
                  <span className="type-meta">Expected effect is not recorded by the control plane.</span>
                </Field>
              </dl>
            </ConfirmationDialog>
          </>
        ) : (
          <p className="type-meta">
            Approval requires an {recommendation.required_approval_role}. Your current role cannot submit this decision.
            Backend authorization remains authoritative.
          </p>
        )
      ) : null}
      {pending ? <p className="type-meta">Waiting for the control plane to record the decision…</p> : null}
      {actionError ? (
        <p role="alert" className="type-meta text-danger">
          {actionError}
        </p>
      ) : null}
    </RecommendationBlock>
  );
}
