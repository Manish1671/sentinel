import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { AIInvestigationCard } from "@/components/incident/ai-investigation-card";
import { AlertList } from "@/components/incident/alert-list";
import { CorrelationSignalsPanel } from "@/components/incident/correlation-signals";
import { EvidenceList } from "@/components/incident/evidence-list";
import { HealthComparison } from "@/components/incident/health-comparison";
import { IncidentSummary } from "@/components/incident/incident-summary";
import { RemediationCard } from "@/components/incident/remediation-card";
import { RecommendationCard } from "@/components/incident/recommendation-card";
import { ChartEmpty } from "@/components/charts/chart-empty";
import { ConfidenceMeter } from "@/components/ai/confidence-meter";
import { ErrorState } from "@/components/core/error-state";
import { EmptyState } from "@/components/core/empty-state";
import type { AlertDetail, InvestigationDetail, RecommendationDetail, RemediationDetail } from "@/lib/api/types";

const investigation: InvestigationDetail = {
  id: "inv-1",
  incident_id: "inc-1",
  status: "completed",
  requested_by_user_id: null,
  model_name: "sentinel-investigator",
  model_version: "2026.09.01",
  root_cause_hypothesis: "Deploy 1.18.0 introduced a 150ms reserve timeout.",
  reasoning_summary: "Latency and errors stepped after the deploy.",
  confidence: 0.72,
  risk_level: "high",
  tool_usage: [{ tool: "get_metrics", call_count: 2, status: "ok" }],
  error_message: null,
  requested_at: "2026-09-14T04:21:00Z",
  started_at: "2026-09-14T04:21:03Z",
  completed_at: "2026-09-14T04:27:00Z",
  evidence: [
    {
      id: "ev-1",
      tool_name: "get_metrics",
      source_type: "metrics",
      summary: "error_rate = 0.083",
      artifact_uri: null,
      source_ref: null,
      metadata: null,
      captured_at: "2026-09-14T04:21:20Z",
    },
  ],
};

const recommendation: RecommendationDetail = {
  id: "rec-1",
  incident_id: "inc-1",
  investigation_id: "inv-1",
  action_type: "rollback_deployment",
  title: "Roll back payments-api to 1.17.4",
  rationale: "Restore the previous version.",
  target_service_id: "svc-1",
  parameters: { from_version: "1.18.0", to_version: "1.17.4" },
  confidence: 0.88,
  risk_level: "high",
  required_approval_role: "approver",
  status: "proposed",
};

const remediation: RemediationDetail = {
  id: "rem-1",
  incident_id: "inc-1",
  recommendation_id: "rec-1",
  service_id: "svc-1",
  status: "pending_approval",
  action_type: "rollback_deployment",
  parameters: { from_version: "1.18.0", to_version: "1.17.4" },
  requested_by_user_id: null,
  attempt_number: 1,
  result_summary: null,
  verification_status: "pending",
  verification_details: null,
  error_message: null,
  started_at: null,
  completed_at: null,
  approval: { id: "appr-1", decision: "pending", actor_user_id: null, decided_at: null },
};

const alerts: AlertDetail[] = [
  {
    id: "a1",
    service_id: "svc-1",
    service_slug: "payments-api",
    detector_id: "high_error_rate",
    severity: "critical",
    status: "open",
    title: "Capture error rate",
    summary: "Checkout capture error rate 8.3%",
    started_at: "2026-09-14T04:19:00Z",
    labels: { deployment_version: "1.18.0" },
  },
];

describe("investigation rendering", () => {
  it("shows a polished empty state when no investigation exists", () => {
    render(<AIInvestigationCard />);
    expect(screen.getByText("No investigation has been run for this incident.")).toBeTruthy();
  });

  it("renders hypothesis and confidence without implying certainty", () => {
    render(<AIInvestigationCard investigation={investigation} />);
    expect(screen.getByText(/Deploy 1.18.0/)).toBeTruthy();
    expect(screen.getByText("72%")).toBeTruthy();
    expect(screen.getByText(/Not a guarantee/)).toBeTruthy();
    expect(screen.getByText(/Inference/)).toBeTruthy();
    expect(screen.getByText("get_metrics")).toBeTruthy();
  });

  it("renders an investigation error state independently", () => {
    render(<AIInvestigationCard state="error" />);
    expect(screen.getByText("Investigation unavailable")).toBeTruthy();
  });
});

describe("evidence states", () => {
  it("categorizes observed evidence and keeps it distinct from inference", () => {
    render(<EvidenceList evidence={investigation.evidence ?? []} />);
    expect(screen.getByText("Observed evidence")).toBeTruthy();
    expect(screen.getByText("Metrics")).toBeTruthy();
    expect(screen.getByText("error_rate = 0.083")).toBeTruthy();
  });

  it("shows an empty evidence state", () => {
    render(<EvidenceList evidence={[]} />);
    expect(screen.getByText("No observed evidence collected")).toBeTruthy();
  });
});

describe("confidence rendering", () => {
  it("uses the numerical value and a medium/high band without calling it a guarantee", () => {
    render(<ConfidenceMeter value={0.86} basis="Based on 4 evidence items. Not a guarantee." />);
    expect(screen.getByText("86%")).toBeTruthy();
    expect(screen.getByText("High confidence")).toBeTruthy();
    expect(screen.getByText(/Based on 4 evidence items/)).toBeTruthy();
  });
});

describe("correlation signals", () => {
  it("renders stored flags without a score", () => {
    render(
      <CorrelationSignalsPanel
        source="timeline"
        signals={{
          same_service: true,
          same_environment: true,
          within_window: true,
          compatible_category: true,
          deployment_context: true,
        }}
      />,
    );
    expect(screen.getByText("Same service")).toBeTruthy();
    expect(screen.getByText("Deployment context")).toBeTruthy();
    expect(screen.getByText(/Not a score/)).toBeTruthy();
    expect(screen.queryByText(/score/i)?.textContent).not.toMatch(/\d+%/);
  });
});

describe("alert rendering", () => {
  it("shows detector, severity, service, and deployment association", () => {
    render(<AlertList alerts={alerts} />);
    expect(screen.getByText("high_error_rate")).toBeTruthy();
    expect(screen.getByText("Capture error rate")).toBeTruthy();
    expect(screen.getByText("payments-api")).toBeTruthy();
    expect(screen.getByText("1.18.0")).toBeTruthy();
  });
});

describe("incident summary strip", () => {
  it("uses API-backed counts and statuses", () => {
    render(
      <IncidentSummary
        summary="Checkout capture failures after deployment 1.18.0"
        alertCount={4}
        investigationStatus="completed"
        recommendationStatus="proposed"
        remediationStatus="pending_approval"
      />,
    );
    expect(screen.getByText("4")).toBeTruthy();
    expect(screen.getByText("completed")).toBeTruthy();
    expect(screen.getByText("pending approval")).toBeTruthy();
  });
});

describe("recommendation states", () => {
  it("renders missing recommendation independently", () => {
    render(<RecommendationCard />);
    expect(screen.getByText("No recommendation yet")).toBeTruthy();
  });
});

describe("remediation lifecycle", () => {
  it("shows pending approval state", () => {
    render(<RemediationCard remediation={remediation} />);
    expect(screen.getByText("pending approval")).toBeTruthy();
    expect(screen.getByText("Pending approval")).toBeTruthy();
    expect(screen.getByText("Verification pending")).toBeTruthy();
  });

  it("shows succeeded terminal state", () => {
    render(
      <RemediationCard
        remediation={{
          ...remediation,
          status: "succeeded",
          verification_status: "passed",
          result_summary: "rollback completed",
          started_at: "2026-09-16T02:10:00Z",
          completed_at: "2026-09-16T02:17:00Z",
          approval: { id: "appr-1", decision: "approved", actor_user_id: "user-1", decided_at: "2026-09-16T02:09:00Z" },
        }}
      />,
    );
    expect(screen.getByText("succeeded")).toBeTruthy();
    expect(screen.getByText("rollback completed")).toBeTruthy();
  });
});

describe("role-aware recommendation controls", () => {
  it("hides approval for viewers", () => {
    render(
      <RecommendationCard recommendation={recommendation} remediation={remediation} role="viewer" />,
    );
    expect(screen.getByText(/Approval requires an approver/)).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Review remediation" })).toBeNull();
  });

  it("shows approval for approvers", () => {
    render(
      <RecommendationCard recommendation={recommendation} remediation={remediation} role="approver" />,
    );
    expect(screen.getByRole("button", { name: "Review remediation" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Reject" })).toBeTruthy();
  });
});

describe("telemetry unavailable", () => {
  it("explains the control-plane gap instead of drawing fake charts", () => {
    render(<ChartEmpty />);
    expect(screen.getByText(/Telemetry history is not exposed by the control plane/)).toBeTruthy();
  });

  it("does not fabricate before/after recovery metrics", () => {
    render(<HealthComparison />);
    expect(screen.getByText("Detailed recovery metrics are not currently available.")).toBeTruthy();
  });

  it("shows a verification snapshot when only after values exist", () => {
    render(
      <HealthComparison
        verificationDetails={{
          verification: { checks: { latency_ms: 180, error_rate: 0.01, db_utilization: 0.4, health_status: "healthy" } },
        }}
      />,
    );
    expect(screen.getByText("Verification snapshot")).toBeTruthy();
    expect(screen.getByText("180")).toBeTruthy();
  });
});

describe("empty and error states", () => {
  it("renders useful error copy with retry", () => {
    render(
      <ErrorState
        title="Unable to load incident INC-2026-0004"
        description="The control plane did not return this incident."
        onRetry={() => undefined}
      />,
    );
    expect(screen.getByRole("alert")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Retry" })).toBeTruthy();
  });

  it("renders product empty copy", () => {
    render(
      <EmptyState title="No watched services" description="When the catalog is connected, production services appear here." />,
    );
    expect(screen.getByText("No watched services")).toBeTruthy();
  });
});
