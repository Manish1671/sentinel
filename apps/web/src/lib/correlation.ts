import type { AlertDetail, IncidentDetail, IncidentTimelineEvent } from "@/lib/api/types";

export type CorrelationSignals = {
  same_service: boolean;
  same_environment: boolean;
  within_window: boolean;
  compatible_category: boolean;
  deployment_context: boolean;
};

export const CORRELATION_SIGNAL_ORDER: Array<{ key: keyof CorrelationSignals; label: string }> = [
  { key: "same_service", label: "Same service" },
  { key: "same_environment", label: "Same environment" },
  { key: "within_window", label: "Within correlation window" },
  { key: "compatible_category", label: "Compatible degradation category" },
  { key: "deployment_context", label: "Deployment context" },
];

const DEGRADATION = new Set([
  "high_latency",
  "high_error_rate",
  "db_connection_saturation",
  "error_log_burst",
]);

function asBool(value: unknown): boolean {
  return value === true;
}

function fromPayload(payload: Record<string, unknown> | null | undefined): CorrelationSignals | null {
  const raw = payload?.correlation;
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;
  const record = raw as Record<string, unknown>;
  return {
    same_service: asBool(record.same_service),
    same_environment: asBool(record.same_environment),
    within_window: asBool(record.within_window),
    compatible_category: asBool(record.compatible_category),
    deployment_context: asBool(record.deployment_context),
  };
}

function merge(base: CorrelationSignals, next: CorrelationSignals): CorrelationSignals {
  return {
    same_service: base.same_service || next.same_service,
    same_environment: base.same_environment || next.same_environment,
    within_window: base.within_window || next.within_window,
    compatible_category: base.compatible_category || next.compatible_category,
    deployment_context: base.deployment_context || next.deployment_context,
  };
}

function deploymentAssociated(alert: AlertDetail): boolean {
  const labels = alert.labels ?? {};
  const associated = labels.deployment_associated;
  return associated === true || associated === "true" || Boolean(labels.deployment_id) || Boolean(labels.deployment_version);
}

export function correlationFromTimeline(events: IncidentTimelineEvent[]): CorrelationSignals | null {
  let acc: CorrelationSignals | null = null;
  for (const event of events) {
    const next = fromPayload(event.payload ?? event.metadata ?? undefined);
    if (!next) continue;
    acc = acc ? merge(acc, next) : next;
  }
  return acc;
}

export function correlationFromIncident(
  incident: IncidentDetail,
  alerts: AlertDetail[],
): CorrelationSignals {
  const sameService = alerts.length > 0 && alerts.every((alert) => alert.service_id === incident.service_id);
  const compatible = alerts.length > 0 && alerts.every((alert) => DEGRADATION.has(alert.detector_id));
  return {
    same_service: sameService,
    same_environment: Boolean(incident.environment) && sameService,
    within_window: false,
    compatible_category: compatible,
    deployment_context: alerts.some(deploymentAssociated),
  };
}

export function resolveCorrelation(
  events: IncidentTimelineEvent[],
  incident: IncidentDetail,
  alerts: AlertDetail[],
): { signals: CorrelationSignals | null; source: "timeline" | "derived" } {
  const stored = correlationFromTimeline(events);
  if (stored) return { signals: stored, source: "timeline" };
  if (alerts.length === 0) return { signals: null, source: "derived" };
  return { signals: correlationFromIncident(incident, alerts), source: "derived" };
}
