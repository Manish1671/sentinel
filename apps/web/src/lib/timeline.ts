import type { IncidentTimelineEvent } from "@/lib/api/types";
import type { TimelineEmphasis, TimelineItem } from "@/types/sentinel";

const UUID = /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/i;

const SKIP_KEYS = new Set([
  "source",
  "audit_event",
  "incident_id",
  "alert_id",
  "service_id",
  "correlation_id",
  "event_id",
  "request_id",
  "correlation",
  "event_key",
]);

function isOperatorValue(key: string, value: unknown): value is string {
  if (typeof value !== "string" || value.length === 0 || value.length > 80) return false;
  if (SKIP_KEYS.has(key)) return false;
  if (UUID.test(value)) return false;
  if (key.endsWith("_id") && key !== "detector_id") return false;
  return true;
}

function displaySource(source?: string): string | undefined {
  if (!source) return undefined;
  if (source.startsWith("services.")) return "system";
  return source;
}

export function timelineEmphasis(kind: string): TimelineEmphasis {
  if (kind.startsWith("remediation") || kind === "approval_recorded") return "remediation";
  if (kind.includes("investigation") || kind === "recommendation_added") return "investigation";
  if (kind === "created" || kind === "status_changed" || kind === "closed") return "lifecycle";
  if (kind.includes("deploy")) return "deployment";
  return "default";
}

export function timelineFromEvents(events: IncidentTimelineEvent[]): TimelineItem[] {
  return events.map((event) => {
    const payload = event.payload ?? event.metadata ?? {};
    const metadata: Record<string, string> = {};
    for (const [key, value] of Object.entries(payload)) {
      if (isOperatorValue(key, value)) {
        metadata[key] = value;
      }
    }
    const audit = typeof payload.audit_event === "string" ? payload.audit_event : undefined;
    const title = event.summary || event.kind.replaceAll("_", " ");
    return {
      id: event.id,
      kind: event.kind,
      timestamp: event.occurred_at ?? event.timestamp ?? "",
      title,
      description: audit && audit !== title ? audit : undefined,
      source: displaySource(event.source),
      actor: event.actor_user_id ? "operator" : undefined,
      metadata: Object.keys(metadata).length ? metadata : undefined,
      emphasis: timelineEmphasis(event.kind),
    };
  });
}

export function groupTimeline(items: TimelineItem[]): TimelineItem[][] {
  const groups: TimelineItem[][] = [];
  for (const item of items) {
    const last = groups.at(-1);
    if (last && last[0]?.kind === item.kind && item.kind === "alert_attached") {
      last.push(item);
    } else {
      groups.push([item]);
    }
  }
  return groups;
}
