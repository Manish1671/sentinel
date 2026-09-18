import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ErrorState } from "@/components/core/error-state";
import { EmptyState } from "@/components/core/empty-state";
import { LoadingState } from "@/components/core/loading-state";
import { RecommendationCard } from "@/components/incident/recommendation-card";
import { controlPlaneLabel } from "@/lib/api/health";
import { isNavActive, primaryNav, titleForPath } from "@/lib/navigation";
import type { RecommendationDetail, RemediationDetail } from "@/lib/api/types";

describe("control plane status", () => {
  it("does not claim operational when the probe fails", () => {
    expect(controlPlaneLabel(null, new Error("timeout"))).toEqual({ label: "Unavailable", tone: "failed" });
    expect(controlPlaneLabel({ status: "unavailable", checks: { postgres: "unavailable" } })).toEqual({
      label: "Degraded",
      tone: "degraded",
    });
    expect(controlPlaneLabel({ status: "ok", checks: { postgres: "ok" } })).toEqual({
      label: "Operational",
      tone: "healthy",
    });
  });
});

describe("localized error states", () => {
  it("renders 404 copy", () => {
    render(<ErrorState title="Service missing was not found" description="The control plane did not return this service." />);
    expect(screen.getByRole("alert").textContent).toMatch(/not found/i);
  });

  it("renders 401/unauthenticated copy", () => {
    render(<ErrorState title="Unable to load overview" description="Authentication required." />);
    expect(screen.getByText("Authentication required.")).toBeTruthy();
  });

  it("renders 403 copy", () => {
    render(<ErrorState title="Approval denied" description="You are not allowed to approve this remediation." />);
    expect(screen.getByText(/not allowed/)).toBeTruthy();
  });

  it("renders API unavailable copy", () => {
    render(
      <ErrorState
        title="Unable to load overview"
        description="Unable to reach the Sentinel control plane."
        onRetry={() => undefined}
      />,
    );
    expect(screen.getByText(/Unable to reach the Sentinel control plane/)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Retry" })).toBeTruthy();
  });
});

describe("empty and loading states", () => {
  it("renders a loading status", () => {
    render(<LoadingState label="Loading service" />);
    expect(screen.getByRole("status")).toBeTruthy();
  });

  it("renders empty lists", () => {
    render(<EmptyState title="No remediations" description="Allowlisted actions appear here after an investigation proposes a change." />);
    expect(screen.getByText("No remediations")).toBeTruthy();
  });
});

describe("authorization states", () => {
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
    incident_reference: "INC-2026-0004",
    recommendation_id: "rec-1",
    service_id: "svc-1",
    service_slug: "payments-api",
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

  it("hides approval for unauthorized roles", () => {
    render(<RecommendationCard recommendation={recommendation} remediation={remediation} role="viewer" />);
    expect(screen.queryByRole("button", { name: "Review remediation" })).toBeNull();
  });

  it("shows approval for authorized roles", () => {
    render(<RecommendationCard recommendation={recommendation} remediation={remediation} role="approver" />);
    expect(screen.getByRole("button", { name: "Review remediation" })).toBeTruthy();
  });
});

describe("navigation consistency", () => {
  it("keeps service detail on the Services nav item", () => {
    const services = primaryNav.find((item) => item.href === "/services")!;
    expect(isNavActive("/services/22222222-2222-4222-8222-222222222221", services)).toBe(true);
    expect(titleForPath("/services/abc")).toBe("Service");
  });

  it("keeps incident workspace on the Incidents nav item", () => {
    const incidents = primaryNav.find((item) => item.href === "/incidents")!;
    expect(isNavActive("/incidents/INC-2026-0012", incidents)).toBe(true);
    expect(titleForPath("/incidents/INC-2026-0012")).toBe("Incident");
  });
});
