import Link from "next/link";
import { IncidentStatus } from "@/components/incident/incident-status";
import { SeverityBadge } from "@/components/core/severity-badge";
import { TechnicalId } from "@/components/core/technical-id";
import type { IncidentDetail, InvestigationDetail, RecommendationDetail, RemediationDetail } from "@/lib/api/types";
import { ArrowRight } from "lucide-react";

type FeaturedIncidentProps = {
  incident: IncidentDetail;
  alerts: string[];
  investigation?: InvestigationDetail | null;
  recommendation?: RecommendationDetail | null;
  remediation?: RemediationDetail | null;
  deploymentVersion?: string | null;
};

export function FeaturedIncident({
  incident,
  alerts,
  investigation,
  recommendation,
  remediation,
  deploymentVersion,
}: FeaturedIncidentProps) {
  return (
    <section
      aria-labelledby="featured-incident"
      className="relative overflow-hidden rounded-md border border-border border-l-2 border-l-severity-critical bg-surface/40 px-5 py-5"
    >
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p id="featured-incident" className="type-label">
          Featured incident
        </p>
        <div className="flex items-center gap-3">
          <SeverityBadge severity={incident.severity} />
          <IncidentStatus status={incident.status} />
        </div>
      </div>
      <div className="mt-3 flex flex-wrap items-baseline gap-x-3 gap-y-1">
        <TechnicalId value={incident.reference} className="text-[13px] text-text-primary" />
        <TechnicalId value={incident.service_slug} />
        {deploymentVersion ? <TechnicalId value={deploymentVersion} /> : null}
      </div>
      <h2 className="mt-3 max-w-3xl text-[1.5rem] font-medium tracking-tight text-text-primary">{incident.title}</h2>
      {incident.summary ? <p className="type-body mt-2 max-w-3xl">{incident.summary}</p> : null}
      <dl className="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <dt className="type-label">Alerts</dt>
          <dd className="type-meta mt-1">{alerts.length ? alerts.join(" · ") : "None attached"}</dd>
        </div>
        <div>
          <dt className="type-label">Investigation</dt>
          <dd className="type-meta mt-1">{investigation?.status ?? "Not run"}</dd>
        </div>
        <div>
          <dt className="type-label">Recommendation</dt>
          <dd className="type-meta mt-1">{recommendation?.title ?? "None"}</dd>
        </div>
        <div>
          <dt className="type-label">Remediation</dt>
          <dd className="type-meta mt-1">
            {remediation
              ? `${remediation.status.replaceAll("_", " ")} · verification ${remediation.verification_status}`
              : "None"}
          </dd>
        </div>
      </dl>
      <div className="mt-6">
        <Link
          href={`/incidents/${incident.reference}`}
          className="inline-flex h-8 items-center gap-1.5 rounded-md bg-primary px-3 text-[13px] font-medium text-primary-foreground transition-opacity hover:opacity-90"
        >
          Open incident workspace
          <ArrowRight className="size-3.5" aria-hidden />
        </Link>
      </div>
    </section>
  );
}
