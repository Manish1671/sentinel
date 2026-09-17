import { EmptyState } from "@/components/core/empty-state";
import { SeverityBadge } from "@/components/core/severity-badge";
import { TechnicalId } from "@/components/core/technical-id";
import { formatDateTime } from "@/lib/format";
import type { AlertDetail } from "@/lib/api/types";
import { BellOff } from "lucide-react";

type AlertListProps = {
  alerts: AlertDetail[];
  serviceSlug?: string;
};

function labelString(labels: Record<string, unknown> | null | undefined, key: string): string | null {
  const value = labels?.[key];
  if (typeof value === "string" && value.length > 0) return value;
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  return null;
}

function deploymentHint(alert: AlertDetail): string | null {
  return (
    labelString(alert.labels, "deployment_version") ??
    labelString(alert.labels, "version") ??
    (alert.labels?.deployment_associated === true || alert.labels?.deployment_associated === "true"
      ? "associated"
      : null)
  );
}

export function AlertList({ alerts, serviceSlug }: AlertListProps) {
  if (alerts.length === 0) {
    return (
      <EmptyState
        icon={BellOff}
        title="No alerts attached"
        description="Correlated alerts appear here when detection links them to this incident."
      />
    );
  }

  return (
    <ul className="divide-y divide-border border-l-2 border-danger/40 pl-4">
      {alerts.map((alert) => {
        const deploy = deploymentHint(alert);
        return (
          <li key={alert.id} className="py-3 first:pt-0 last:pb-0">
            <div className="flex flex-wrap items-center gap-2">
              <TechnicalId value={alert.detector_id} className="text-text-primary" />
              <SeverityBadge severity={alert.severity} />
              <time className="type-code text-text-muted" dateTime={alert.started_at}>
                {formatDateTime(alert.started_at)}
              </time>
            </div>
            <p className="mt-1 text-[13px] font-medium tracking-tight text-text-primary">{alert.title}</p>
            <p className="type-body mt-0.5">{alert.summary}</p>
            <div className="mt-1.5 flex flex-wrap gap-x-4 gap-y-1">
              <span className="type-meta">
                Service <TechnicalId value={alert.service_slug || serviceSlug || alert.service_id} />
              </span>
              <span className="type-meta">
                Deploy {deploy ? <TechnicalId value={deploy} /> : <span>not associated</span>}
              </span>
            </div>
          </li>
        );
      })}
    </ul>
  );
}
