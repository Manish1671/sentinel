import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { associatedIncident, DeploymentList } from "@/components/catalog/deployment-list";
import { ServiceWorkspace } from "@/components/catalog/service-workspace";
import { IncidentList } from "@/components/incident/incident-list";
import { InvestigationsView } from "@/components/incident/investigations-view";
import { RemediationsView } from "@/components/incident/remediations-view";
import { CommandPalette } from "@/components/shell/command-palette";
import { SystemStatusIndicator } from "@/components/shell/system-status";
import { ApiError } from "@/lib/api/errors";
import type {
  DeploymentSummary,
  IncidentSummary,
  InvestigationDetail,
  RecommendationDetail,
  RemediationDetail,
  ServiceDetail,
  ServiceSummary,
} from "@/lib/api/types";

const push = vi.fn();

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push, replace: vi.fn() }),
  usePathname: () => "/overview",
}));

vi.mock("@/components/auth/auth-provider", () => ({
  useAuth: () => ({
    user: {
      id: "u-approver",
      email: "jordan.hale@sentinel.dev",
      display_name: "Jordan Hale",
      role: "approver",
    },
    reload: async () => undefined,
    signOut: async () => undefined,
  }),
}));

vi.mock("@/lib/api/services", () => ({
  getService: vi.fn(),
  listServices: vi.fn(),
}));

vi.mock("@/lib/api/incidents", () => ({
  listIncidents: vi.fn(),
  getIncidentAlerts: vi.fn(),
}));

vi.mock("@/lib/api/deployments", () => ({
  listDeployments: vi.fn(),
}));

vi.mock("@/lib/api/investigations", () => ({
  listInvestigations: vi.fn(),
}));

vi.mock("@/lib/api/remediations", () => ({
  listRemediations: vi.fn(),
}));

vi.mock("@/lib/api/recommendations", () => ({
  getRecommendation: vi.fn(),
}));

vi.mock("@/lib/api/health", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api/health")>("@/lib/api/health");
  return {
    ...actual,
    getControlPlaneReady: vi.fn(),
  };
});

import { getService, listServices } from "@/lib/api/services";
import { getIncidentAlerts, listIncidents } from "@/lib/api/incidents";
import { listDeployments } from "@/lib/api/deployments";
import { listInvestigations } from "@/lib/api/investigations";
import { listRemediations } from "@/lib/api/remediations";
import { getRecommendation } from "@/lib/api/recommendations";
import { getControlPlaneReady } from "@/lib/api/health";

const service: ServiceDetail = {
  id: "svc-1",
  slug: "payments-api",
  name: "Payments API",
  environment: "production",
  health_status: "degraded",
  owner_user_id: "owner-1",
  current_version: "1.18.0",
  active_incident_count: 1,
  description: "Authorization, capture, and refunds.",
  recent_deployments: [
    {
      id: "dep-1",
      service_id: "svc-1",
      service_slug: "payments-api",
      environment: "production",
      version: "1.18.0",
      git_sha: "abc123",
      status: "healthy",
      started_at: "2026-09-14T04:12:00Z",
      completed_at: "2026-09-14T04:17:00Z",
    },
  ],
};

const catalogService: ServiceSummary = {
  id: "svc-1",
  slug: "payments-api",
  name: "Payments API",
  environment: "production",
  health_status: "degraded",
  owner_user_id: "owner-1",
  current_version: "1.18.0",
  active_incident_count: 1,
};

const openIncident: IncidentSummary = {
  id: "inc-1",
  reference: "INC-2026-0012",
  service_id: "svc-1",
  service_slug: "payments-api",
  service_name: "Payments API",
  environment: "production",
  title: "payments-api production performance incident",
  severity: "critical",
  status: "open",
  commander_user_id: null,
  detected_at: "2026-09-15T03:11:00Z",
  version: 1,
};

const resolvedIncident: IncidentSummary = {
  ...openIncident,
  id: "inc-2",
  reference: "INC-2026-0001",
  title: "Resolved capture errors",
  severity: "high",
  status: "resolved",
  detected_at: "2026-09-10T03:11:00Z",
};

const investigation: InvestigationDetail = {
  id: "inv-1",
  incident_id: "inc-1",
  incident_reference: "INC-2026-0012",
  status: "completed",
  requested_by_user_id: null,
  model_name: "sentinel-investigator",
  model_version: "2026.09.01",
  root_cause_hypothesis: "Database connection saturation coinciding with deployment 1.18.0.",
  reasoning_summary: "Latency and errors stepped after the deploy.",
  confidence: 0.72,
  risk_level: "high",
  tool_usage: [],
  error_message: null,
  requested_at: "2026-09-14T04:21:00Z",
  started_at: "2026-09-14T04:21:03Z",
  completed_at: "2026-09-14T04:27:00Z",
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

const pendingRemediation: RemediationDetail = {
  id: "rem-1",
  incident_id: "inc-1",
  incident_reference: "INC-2026-0012",
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

const deployment: DeploymentSummary = {
  id: "dep-1",
  service_id: "svc-1",
  service_slug: "payments-api",
  environment: "production",
  version: "1.18.0",
  git_sha: null,
  status: "healthy",
  started_at: "2026-09-14T04:12:00Z",
  completed_at: "2026-09-14T04:17:00Z",
};

beforeEach(() => {
  push.mockReset();
  vi.mocked(listServices).mockResolvedValue({ data: [catalogService] });
  vi.mocked(listIncidents).mockResolvedValue({ data: [openIncident, resolvedIncident], page: { next_cursor: "next", limit: 25 } });
  vi.mocked(getService).mockResolvedValue({ data: service });
  vi.mocked(getIncidentAlerts).mockResolvedValue({
    data: [
      {
        id: "a1",
        service_id: "svc-1",
        service_slug: "payments-api",
        detector_id: "high_latency",
        severity: "high",
        status: "open",
        title: "High request latency",
        summary: "p99 exceeded threshold",
        started_at: "2026-09-15T03:11:00Z",
        labels: { service: "payments-api" },
      },
    ],
  });
  vi.mocked(listDeployments).mockResolvedValue({ data: [deployment] });
  vi.mocked(listInvestigations).mockResolvedValue({ data: [investigation] });
  vi.mocked(listRemediations).mockResolvedValue({ data: [pendingRemediation] });
  vi.mocked(getRecommendation).mockResolvedValue({ data: recommendation });
  vi.mocked(getControlPlaneReady).mockResolvedValue({ status: "ok", checks: { postgres: "ok" } });
});

describe("service detail", () => {
  it("renders catalog fields and incident context from the control plane", async () => {
    render(<ServiceWorkspace serviceId="svc-1" />);
    expect(screen.getByText("Loading service")).toBeTruthy();
    await screen.findByRole("heading", { name: "Payments API" });
    expect(screen.getAllByText("payments-api").length).toBeGreaterThan(0);
    expect(screen.getAllByText("1.18.0").length).toBeGreaterThan(0);
    expect(screen.getByText("Open incident workspace").closest("a")?.getAttribute("href")).toBe(
      "/incidents/INC-2026-0012",
    );
    expect(screen.getByText("High request latency")).toBeTruthy();
    expect(screen.getByText("Telemetry history is not exposed by the control plane")).toBeTruthy();
  });

  it("shows a localized 404 when the service is missing", async () => {
    vi.mocked(getService).mockRejectedValue(
      new ApiError({ message: "Service not found", status: 404, code: "not_found" }),
    );
    render(<ServiceWorkspace serviceId="missing" />);
    await screen.findByRole("alert");
    expect(screen.getByText(/Service missing was not found/)).toBeTruthy();
  });
});

describe("incident filters", () => {
  it("filters by status using the real list query", async () => {
    vi.mocked(listIncidents).mockImplementation(async (query) => {
      if (query.status === "open") {
        return { data: [openIncident] };
      }
      return { data: [openIncident, resolvedIncident], page: { next_cursor: "c2", limit: 25 } };
    });
    render(<IncidentList />);
    await screen.findByText("INC-2026-0012");
    expect(screen.getByText("INC-2026-0001")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Load more" })).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "open" }));
    await waitFor(() => {
      expect(screen.queryByText("INC-2026-0001")).toBeNull();
    });
    expect(screen.getByText("INC-2026-0012")).toBeTruthy();
    expect(screen.getAllByText("payments-api").length).toBeGreaterThan(0);
  });

  it("renders an empty filter result", async () => {
    vi.mocked(listIncidents).mockResolvedValue({ data: [] });
    render(<IncidentList />);
    await screen.findByText("No incidents match these filters");
  });

  it("renders an incident list error without crashing", async () => {
    vi.mocked(listIncidents).mockRejectedValue(new Error("timeout"));
    render(<IncidentList />);
    await screen.findByRole("alert");
    expect(screen.getByText("Unable to load incidents")).toBeTruthy();
  });
});

describe("investigation list", () => {
  it("shows hypothesis, confidence, and a path to the incident workspace", async () => {
    render(<InvestigationsView />);
    await screen.findByText(/Database connection saturation/);
    expect(screen.getByText(/Confidence 72%/)).toBeTruthy();
    expect(screen.getByText("Open incident workspace").closest("a")?.getAttribute("href")).toBe(
      "/incidents/INC-2026-0012",
    );
  });
});

describe("remediation queue", () => {
  it("groups pending approval and keeps the review control", async () => {
    render(<RemediationsView />);
    await screen.findByRole("heading", { name: /Pending approval/ });
    expect(screen.getAllByText("Roll back payments-api to 1.17.4").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Review remediation" })).toBeTruthy();
  });
});

describe("deployment navigation", () => {
  it("links the associated incident when service and time align", async () => {
    render(<DeploymentList />);
    const link = await screen.findByRole("link", { name: "INC-2026-0012" });
    expect(link.getAttribute("href")).toBe("/incidents/INC-2026-0012");
    expect(screen.getByText("Deployment details unavailable.")).toBeTruthy();
  });

  it("associates by version title or detection window without fabricating records", () => {
    expect(associatedIncident(deployment, [openIncident])?.reference).toBe("INC-2026-0012");
    expect(associatedIncident({ ...deployment, service_id: "other", service_slug: "other" }, [openIncident])).toBeNull();
  });
});

describe("system status", () => {
  it("shows operational when readiness succeeds", async () => {
    render(<SystemStatusIndicator />);
    await screen.findByText(/PostgreSQL/);
    expect(screen.getByText("Operational")).toBeTruthy();
  });

  it("does not claim operational when the probe fails", async () => {
    vi.mocked(getControlPlaneReady).mockRejectedValue(new Error("timeout"));
    render(<SystemStatusIndicator />);
    await screen.findByText(/could not be reached/);
    expect(screen.getByText("Unavailable")).toBeTruthy();
  });
});

describe("command palette", () => {
  it("offers go-to destinations from live lists", async () => {
    render(<CommandPalette open onOpenChange={() => undefined} />);
    await screen.findByText("Go to incident");
    expect(screen.getByText("Go to service")).toBeTruthy();
    expect(screen.getByText("Go to investigation")).toBeTruthy();
    expect(screen.getByText("Go to remediation")).toBeTruthy();
    expect(screen.getByText("Go to deployment")).toBeTruthy();
    fireEvent.click(screen.getByText("Go to incident"));
    expect(push).toHaveBeenCalledWith("/incidents/INC-2026-0012");
  });
});

describe("responsive layout classes", () => {
  it("uses stacked then row layout for incident rows", async () => {
    const { container } = render(<IncidentList />);
    await screen.findByText("INC-2026-0012");
    expect(container.querySelector("li")?.className).toMatch(/md:flex-row/);
  });
});
