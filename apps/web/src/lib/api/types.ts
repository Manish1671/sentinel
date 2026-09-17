import type {
  Environment,
  HealthStatus,
  IncidentStatus,
  InvestigationStatus,
  RecommendationStatus,
  RemediationStatus,
  Severity,
  VerificationStatus,
} from "@/types/sentinel";

export type PageInfo = {
  next_cursor: string | null;
  limit: number;
};

export type ListResponse<T> = {
  data: T[];
  page?: PageInfo;
};

export type ItemResponse<T> = {
  data: T;
};

export type FetchInit = {
  signal?: AbortSignal;
};

export type ServiceSummary = {
  id: string;
  slug: string;
  name: string;
  environment: Environment;
  health_status: HealthStatus;
  owner_user_id: string | null;
  current_version: string | null;
  active_incident_count: number;
};

export type DeploymentSummary = {
  id: string;
  service_id?: string;
  service_slug?: string;
  version: string;
  git_sha: string | null;
  status: string;
  started_at: string;
  completed_at: string | null;
};

export type ServiceDetail = ServiceSummary & {
  description: string;
  recent_deployments: DeploymentSummary[];
};

export type IncidentSummary = {
  id: string;
  reference: string;
  service_id: string;
  service_slug: string;
  service_name: string;
  environment: Environment;
  title: string;
  severity: Severity;
  status: IncidentStatus;
  commander_user_id: string | null;
  detected_at: string;
  version: number;
};

export type IncidentDetail = IncidentSummary & {
  summary: string | null;
  created_by_user_id: string | null;
  resolved_at: string | null;
  closed_at: string | null;
  alert_ids: string[];
};

export type IncidentTimelineEvent = {
  id: string;
  kind: string;
  event_type?: string;
  actor_user_id: string | null;
  occurred_at: string;
  timestamp?: string;
  summary?: string;
  source?: string;
  payload: Record<string, unknown> | null;
  metadata?: Record<string, unknown> | null;
};

export type EvidenceItem = {
  id: string;
  tool_name: string;
  source_type: string;
  summary: string;
  artifact_uri: string | null;
  source_ref: string | null;
  metadata: Record<string, unknown> | null;
  captured_at: string;
};

export type ToolUsage = {
  tool: string;
  call_count?: number;
  status?: string;
};

export type InvestigationDetail = {
  id: string;
  incident_id: string;
  status: InvestigationStatus;
  requested_by_user_id: string | null;
  model_name: string | null;
  model_version: string | null;
  root_cause_hypothesis: string | null;
  reasoning_summary: string | null;
  confidence: number | null;
  risk_level: Severity | null;
  tool_usage: ToolUsage[] | null;
  error_message: string | null;
  requested_at: string;
  started_at: string | null;
  completed_at: string | null;
  evidence?: EvidenceItem[];
};

export type RecommendationDetail = {
  id: string;
  incident_id: string;
  investigation_id: string | null;
  action_type: string;
  title: string;
  rationale: string;
  target_service_id: string;
  parameters: Record<string, unknown> | null;
  confidence: number;
  risk_level: Severity;
  required_approval_role: string;
  status: RecommendationStatus;
};

export type RemediationDetail = {
  id: string;
  incident_id: string;
  recommendation_id: string;
  service_id: string;
  status: RemediationStatus;
  action_type: string;
  parameters: Record<string, unknown> | null;
  requested_by_user_id: string | null;
  attempt_number: number;
  result_summary: string | null;
  verification_status: VerificationStatus;
  verification_details: Record<string, unknown> | null;
  error_message: string | null;
  started_at: string | null;
  completed_at: string | null;
  approval: {
    id: string;
    decision: "pending" | "approved" | "rejected";
    actor_user_id: string | null;
    comment?: string | null;
    decided_at: string | null;
  } | null;
};

export type AlertDetail = {
  id: string;
  service_id: string;
  service_slug?: string;
  detector_id: string;
  severity: Severity;
  status: string;
  title: string;
  summary: string;
  started_at: string;
  labels?: Record<string, unknown> | null;
};

export type ListQuery = {
  limit?: number;
  cursor?: string;
};
