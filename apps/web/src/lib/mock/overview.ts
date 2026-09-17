/**
 * MOCK / DEMO DATA — UI DEVELOPMENT ONLY
 * Isolated composition data for the Overview placeholder.
 */
import { mockIncident, mockServices } from "@/lib/mock/catalog";
import { demoTelemetry, MOCK_DATA_NOTICE } from "@/lib/mock/telemetry";

export { MOCK_DATA_NOTICE };

export const overviewComposition = {
  notice: MOCK_DATA_NOTICE,
  environment: "PRODUCTION" as const,
  systemState: "Degraded" as const,
  health: {
    overall: "degraded" as const,
    overallLabel: "Degraded",
    overallHint: "2 of 3 watched services are outside baseline",
    activeIncidents: 0,
    activeIncidentsHint: "INC-2026-0012 resolved after verification",
    investigation: "Idle",
    investigationHint: "Last completed 10:45 UTC",
    pendingApprovals: 0,
    pendingApprovalsHint: "No remediations waiting",
    watchedServices: mockServices.length,
    watchedServicesHint: "Production catalog",
  },
  telemetry: [
    {
      id: "latency",
      label: "Request latency",
      value: "190",
      unit: "ms p99",
      tone: "warning" as const,
      series: demoTelemetry.map((point) => point.latencyMs),
    },
    {
      id: "errors",
      label: "Error rate",
      value: "0.5",
      unit: "%",
      tone: "success" as const,
      series: demoTelemetry.map((point) => point.errorRate),
    },
    {
      id: "saturation",
      label: "DB saturation",
      value: "47",
      unit: "%",
      tone: "success" as const,
      series: demoTelemetry.map((point) => point.saturation),
    },
    {
      id: "availability",
      label: "Availability",
      value: "99.1",
      unit: "%",
      tone: "warning" as const,
      series: [99.98, 99.97, 99.96, 99.4, 98.2, 98.4, 99.0, 99.1],
    },
  ],
  featuredIncident: mockIncident,
  services: mockServices,
};
