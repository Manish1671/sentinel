/**
 * MOCK / DEMO DATA — UI DEVELOPMENT ONLY
 * See `@/lib/mock/notice`. Never import this from API clients.
 */
import { IS_MOCK_DATA, MOCK_DATA_NOTICE } from "@/lib/mock/notice";
import type {
  AlertName,
  Environment,
  IncidentStatus,
  InvestigationStatus,
  RecommendationStatus,
  RemediationStatus,
  Severity,
  TimelineItem,
  VerificationStatus,
} from "@/types/sentinel";

export { IS_MOCK_DATA, MOCK_DATA_NOTICE };

export type MockAlert = {
  id: string;
  name: AlertName;
  title: string;
  severity: Severity;
  serviceSlug: string;
  detectedAt: string;
  summary: string;
};

export type MockEvidence = {
  id: string;
  title: string;
  source: string;
  observedAt: string;
  detail: string;
};

export type MockIncident = {
  id: string;
  reference: string;
  title: string;
  serviceSlug: string;
  serviceName: string;
  environment: Environment;
  severity: Severity;
  status: IncidentStatus;
  summary: string;
  detectedAt: string;
  commander: string;
};

export type MockInvestigation = {
  id: string;
  incidentId: string;
  status: InvestigationStatus;
  hypothesis: string;
  reasoning: string;
  confidence: number;
  confidenceBasis: string;
  completedAt: string;
  modelName: string;
};

export type MockRecommendation = {
  id: string;
  incidentId: string;
  title: string;
  actionType: string;
  rationale: string;
  confidence: number;
  confidenceBasis: string;
  risk: "low" | "medium" | "high";
  expectedEffect: { label: string; direction: "down" | "up" }[];
  status: RecommendationStatus;
  targetVersion: string;
  fromVersion: string;
};

export type MockRemediation = {
  id: string;
  incidentId: string;
  actionType: string;
  status: RemediationStatus;
  verificationStatus: VerificationStatus;
  fromVersion: string;
  toVersion: string;
  completedAt: string;
};

export type MockService = {
  id: string;
  slug: string;
  name: string;
  environment: Environment;
  health: "healthy" | "degraded" | "failed";
  version: string;
  latencyMs: number;
  errorRate: number;
  availability: number;
  incidentCount: number;
  trend: number[];
};

export type MockDeployment = {
  id: string;
  serviceSlug: string;
  version: string;
  gitSha: string;
  status: "running" | "succeeded" | "failed";
  startedAt: string;
};

export const mockService: MockService = {
  id: "svc-payments-api",
  slug: "payments-api",
  name: "Payments API",
  environment: "production",
  health: "degraded",
  version: "1.17.4",
  latencyMs: 190,
  errorRate: 0.5,
  availability: 99.1,
  incidentCount: 1,
  trend: [160, 170, 175, 210, 1640, 1510, 420, 190],
};

export const mockServices: MockService[] = [
  mockService,
  {
    id: "svc-checkout-api",
    slug: "checkout-api",
    name: "Checkout API",
    environment: "production",
    health: "healthy",
    version: "2.4.1",
    latencyMs: 84,
    errorRate: 0.2,
    availability: 99.98,
    incidentCount: 0,
    trend: [90, 88, 86, 92, 85, 84, 83, 84],
  },
  {
    id: "svc-inventory-api",
    slug: "inventory-api",
    name: "Inventory API",
    environment: "production",
    health: "degraded",
    version: "1.9.3",
    latencyMs: 310,
    errorRate: 1.4,
    availability: 99.4,
    incidentCount: 0,
    trend: [180, 190, 210, 240, 280, 320, 315, 310],
  },
];

export const mockIncident: MockIncident = {
  id: "inc-2026-0012",
  reference: "INC-2026-0012",
  title: "Checkout capture failures after payments-api 1.18.0",
  serviceSlug: "payments-api",
  serviceName: "Payments API",
  environment: "production",
  severity: "critical",
  status: "resolved",
  summary:
    "Capture error rate and p99 latency rose immediately after deployment 1.18.0. Correlated alerts: high_latency, high_error_rate, and db_connection_saturation.",
  detectedAt: "2026-09-14T10:44:00Z",
  commander: "A. Rao",
};

export const mockAlerts: MockAlert[] = [
  {
    id: "alert-high-latency",
    name: "high_latency",
    title: "High latency",
    severity: "high",
    serviceSlug: "payments-api",
    detectedAt: "2026-09-14T10:42:00Z",
    summary: "p99 capture latency crossed 1.6s.",
  },
  {
    id: "alert-high-error-rate",
    name: "high_error_rate",
    title: "High error rate",
    severity: "critical",
    serviceSlug: "payments-api",
    detectedAt: "2026-09-14T10:43:00Z",
    summary: "Capture error rate reached 8.3%.",
  },
  {
    id: "alert-db-sat",
    name: "db_connection_saturation",
    title: "DB connection saturation",
    severity: "high",
    serviceSlug: "payments-api",
    detectedAt: "2026-09-14T10:43:20Z",
    summary: "Checkout pool utilization exceeded saturation threshold.",
  },
];

export const mockEvidence: MockEvidence[] = [
  {
    id: "ev-deploy",
    title: "Deployment 1.18.0 completed",
    source: "catalog",
    observedAt: "2026-09-14T10:41:00Z",
    detail: "payments-api rolled to 1.18.0 (git e4b21aa) at 10:41 UTC.",
  },
  {
    id: "ev-latency",
    title: "Latency step-change",
    source: "telemetry",
    observedAt: "2026-09-14T10:42:00Z",
    detail: "p99 capture latency moved from 180ms to 1640ms.",
  },
  {
    id: "ev-errors",
    title: "Error-rate step-change",
    source: "telemetry",
    observedAt: "2026-09-14T10:43:00Z",
    detail: "Capture error rate moved from 0.4% to 8.3%.",
  },
];

export const mockInvestigation: MockInvestigation = {
  id: "inv-0012",
  incidentId: mockIncident.id,
  status: "completed",
  hypothesis:
    "payments-api 1.18.0 added a synchronous inventory.reserve call with a 150ms timeout. Inventory p99 exceeds that budget, so capture fails and checkout degrades.",
  reasoning:
    "Error rate and latency stepped at 10:41 after 1.18.0. Traces show inventory.reserve status=error at 150ms.",
  confidence: 0.86,
  confidenceBasis:
    "Based on 3 evidence items across telemetry, deployment history, and correlated alerts.",
  completedAt: "2026-09-14T10:45:00Z",
  modelName: "sentinel-investigator",
};

export const mockRecommendation: MockRecommendation = {
  id: "rec-0012",
  incidentId: mockIncident.id,
  title: "Roll back payments-api to 1.17.4",
  actionType: "rollback_deployment",
  rationale: "Rollback removes the synchronous reserve path introduced in 1.18.0.",
  confidence: 0.88,
  confidenceBasis: "Based on the completed investigation and allowlisted rollback path.",
  risk: "medium",
  expectedEffect: [
    { label: "Latency", direction: "down" },
    { label: "Errors", direction: "down" },
    { label: "DB utilization", direction: "down" },
  ],
  status: "accepted",
  targetVersion: "1.17.4",
  fromVersion: "1.18.0",
};

export const mockRemediation: MockRemediation = {
  id: "rem-0012",
  incidentId: mockIncident.id,
  actionType: "rollback_deployment",
  status: "succeeded",
  verificationStatus: "passed",
  fromVersion: "1.18.0",
  toVersion: "1.17.4",
  completedAt: "2026-09-14T10:49:00Z",
};

export const mockDeployment: MockDeployment = {
  id: "dep-1180",
  serviceSlug: "payments-api",
  version: "1.18.0",
  gitSha: "e4b21aa",
  status: "succeeded",
  startedAt: "2026-09-14T10:41:00Z",
};

export const mockDeployments: MockDeployment[] = [
  mockDeployment,
  {
    id: "dep-1174",
    serviceSlug: "payments-api",
    version: "1.17.4",
    gitSha: "9c18d02",
    status: "succeeded",
    startedAt: "2026-09-13T18:04:00Z",
  },
];

export const mockTimeline: TimelineItem[] = [
  {
    id: "tl-1",
    timestamp: "2026-09-14T10:41:00Z",
    title: "Deployment 1.18.0",
    description: "payments-api rolled out to production.",
    source: "catalog",
    metadata: { version: "1.18.0" },
  },
  {
    id: "tl-2",
    timestamp: "2026-09-14T10:42:00Z",
    title: "High latency detected",
    description: "p99 capture latency crossed the high_latency threshold.",
    severity: "high",
    source: "detection",
  },
  {
    id: "tl-3",
    timestamp: "2026-09-14T10:43:00Z",
    title: "Error rate increased",
    description: "high_error_rate and db_connection_saturation fired.",
    severity: "critical",
    source: "detection",
  },
  {
    id: "tl-4",
    timestamp: "2026-09-14T10:44:00Z",
    title: "Incident created",
    description: "INC-2026-0012 opened from correlated alerts.",
    source: "incident",
    metadata: { reference: "INC-2026-0012" },
  },
  {
    id: "tl-5",
    timestamp: "2026-09-14T10:45:00Z",
    title: "AI investigation completed",
    description: "Hypothesis points to the 1.18.0 reserve timeout.",
    source: "investigation",
    actor: "sentinel-investigator",
  },
  {
    id: "tl-6",
    timestamp: "2026-09-14T10:47:00Z",
    title: "Rollback approved",
    description: "Approver accepted rollback of deployment 1.18.0.",
    source: "remediation",
    actor: "A. Rao",
  },
  {
    id: "tl-7",
    timestamp: "2026-09-14T10:49:00Z",
    title: "Recovery verified",
    description: "Health comparison passed. Incident resolved.",
    status: "resolved",
    source: "verification",
  },
];

export type MockIncidentWorkspace = {
  incident: MockIncident;
  alerts: MockAlert[];
  timeline: TimelineItem[];
  evidence: MockEvidence[];
  investigation: MockInvestigation;
  recommendation: MockRecommendation;
  remediation: MockRemediation;
};

export const mockIncidentWorkspace: MockIncidentWorkspace = {
  incident: mockIncident,
  alerts: mockAlerts,
  timeline: mockTimeline,
  evidence: mockEvidence,
  investigation: mockInvestigation,
  recommendation: mockRecommendation,
  remediation: mockRemediation,
};

export function getMockIncidentWorkspace(id: string): MockIncidentWorkspace | null {
  const key = id.toLowerCase();
  if (key === mockIncident.id || key === mockIncident.reference.toLowerCase()) {
    return mockIncidentWorkspace;
  }
  return mockIncidentWorkspace;
}

export const mockInvestigations = [mockInvestigation];
export const mockRecommendations = [mockRecommendation];
export const mockRemediations = [mockRemediation];
export const mockIncidents = [mockIncident];
