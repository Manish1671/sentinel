import Link from "next/link";
import { TechnicalId } from "@/components/core/technical-id";
import { SeverityBadge } from "@/components/core/severity-badge";
import { IncidentStatus } from "@/components/incident/incident-status";
import { formatDateTime } from "@/lib/format";
import type { IncidentDetail } from "@/lib/api/types";
import { cn } from "@/lib/utils";

type IncidentHeaderProps = {
  incident: IncidentDetail;
  deploymentVersion?: string | null;
};

export function IncidentHeader({ incident, deploymentVersion }: IncidentHeaderProps) {
  return (
    <header className="space-y-4 border-b border-border pb-5">
      <nav aria-label="Incident context" className="flex flex-wrap items-center gap-x-2 gap-y-1 type-meta">
        <Link href="/incidents" className="hover:text-text-primary">
          Incidents
        </Link>
        <span aria-hidden>/</span>
        <TechnicalId value={incident.reference} className="text-text-primary" />
        <span aria-hidden className="text-border">
          ·
        </span>
        <Link href="/services" className="hover:text-text-primary">
          <TechnicalId value={incident.service_slug} />
        </Link>
        <span aria-hidden className="text-border">
          ·
        </span>
        <Link href="/deployments" className="hover:text-text-primary">
          Deployments
        </Link>
        <span aria-hidden className="text-border">
          ·
        </span>
        <Link href="/investigations" className="hover:text-text-primary">
          Investigations
        </Link>
        <span aria-hidden className="text-border">
          ·
        </span>
        <Link href="/remediations" className="hover:text-text-primary">
          Remediations
        </Link>
      </nav>

      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="space-y-3 min-w-0">
          <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
            <TechnicalId value={incident.reference} className="text-[0.9375rem] text-text-primary" />
            <TechnicalId value={incident.service_slug} className="text-[0.9375rem]" />
            <span className="type-meta uppercase tracking-[0.08em]">{incident.environment}</span>
          </div>
          <h1 className="type-page-title max-w-3xl text-[1.375rem] leading-snug">{incident.title}</h1>
        </div>
        <div
          className={cn(
            "flex flex-wrap items-center gap-x-3 gap-y-2 rounded-md border border-border px-3 py-2",
          )}
        >
          <SeverityBadge severity={incident.severity} />
          <span className="type-meta" aria-hidden>
            ·
          </span>
          <IncidentStatus status={incident.status} className="text-[13px] font-medium text-text-primary" />
        </div>
      </div>

      <dl className="flex flex-wrap gap-x-6 gap-y-2">
        <div>
          <dt className="type-label">Detected</dt>
          <dd className="type-code text-text-secondary">{formatDateTime(incident.detected_at)}</dd>
        </div>
        {incident.resolved_at ? (
          <div>
            <dt className="type-label">Resolved</dt>
            <dd className="type-code text-text-secondary">{formatDateTime(incident.resolved_at)}</dd>
          </div>
        ) : null}
        <div>
          <dt className="type-label">Deployment context</dt>
          <dd>
            {deploymentVersion ? (
              <TechnicalId value={deploymentVersion} />
            ) : (
              <span className="type-meta">Not recorded on this incident</span>
            )}
          </dd>
        </div>
      </dl>
    </header>
  );
}
