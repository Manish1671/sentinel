import { describe, expect, it } from "vitest";
import { correlationFromIncident, correlationFromTimeline } from "@/lib/correlation";
import type { AlertDetail, IncidentDetail, IncidentTimelineEvent } from "@/lib/api/types";

const incident: IncidentDetail = {
  id: "inc-1",
  reference: "INC-2026-0004",
  service_id: "svc-1",
  service_slug: "payments-api",
  service_name: "Payments API",
  environment: "production",
  title: "Payments capture failures",
  severity: "critical",
  status: "resolved",
  commander_user_id: null,
  detected_at: "2026-09-14T04:19:00Z",
  version: 1,
  summary: "Capture errors after 1.18.0",
  created_by_user_id: null,
  resolved_at: "2026-09-16T02:17:00Z",
  closed_at: null,
  alert_ids: ["a1"],
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
    summary: "8.3%",
    started_at: "2026-09-14T04:19:00Z",
    labels: { deployment_associated: "true", deployment_version: "1.18.0" },
  },
];

describe("correlationFromTimeline", () => {
  it("reads stored correlation explanations without scoring", () => {
    const events: IncidentTimelineEvent[] = [
      {
        id: "e1",
        kind: "alert_attached",
        actor_user_id: null,
        occurred_at: "2026-09-14T04:20:00Z",
        payload: {
          correlation: {
            same_service: true,
            same_environment: true,
            within_window: true,
            compatible_category: true,
            deployment_context: false,
          },
        },
      },
    ];
    expect(correlationFromTimeline(events)).toEqual({
      same_service: true,
      same_environment: true,
      within_window: true,
      compatible_category: true,
      deployment_context: false,
    });
  });
});

describe("correlationFromIncident", () => {
  it("derives service, category, and deployment context from real alerts", () => {
    expect(correlationFromIncident(incident, alerts)).toMatchObject({
      same_service: true,
      compatible_category: true,
      deployment_context: true,
    });
  });
});
