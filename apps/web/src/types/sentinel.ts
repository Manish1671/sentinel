export type Severity = "critical" | "high" | "medium" | "low";

export type IncidentStatus =
  | "open"
  | "investigating"
  | "remediating"
  | "verifying"
  | "resolved"
  | "closed";

export type HealthStatus = "healthy" | "degraded" | "unhealthy" | "unknown";

export type OperationalStatus =
  | "healthy"
  | "degraded"
  | "failed"
  | "running"
  | "pending"
  | "resolved";

export type Environment = "production" | "staging" | "development";

export type InvestigationStatus =
  | "requested"
  | "running"
  | "completed"
  | "failed";

export type RecommendationStatus =
  | "proposed"
  | "accepted"
  | "rejected"
  | "superseded";

export type RemediationStatus =
  | "pending_approval"
  | "approved"
  | "rejected"
  | "running"
  | "verifying"
  | "succeeded"
  | "failed"
  | "cancelled";

export type VerificationStatus = "pending" | "passed" | "failed" | "skipped";

export type AlertName =
  | "high_latency"
  | "high_error_rate"
  | "db_connection_saturation";

export type PageState = "loading" | "empty" | "error" | "success";

export type TimelineEmphasis = "default" | "lifecycle" | "deployment" | "remediation" | "investigation";

export type TimelineItem = {
  id: string;
  timestamp: string;
  title: string;
  kind?: string;
  description?: string;
  severity?: Severity;
  status?: OperationalStatus;
  metadata?: Record<string, string>;
  actor?: string;
  source?: string;
  emphasis?: TimelineEmphasis;
};
