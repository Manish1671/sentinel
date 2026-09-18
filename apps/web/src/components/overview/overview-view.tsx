"use client";

import Link from "next/link";
import { HealthStrip, TelemetryStrip } from "@/components/overview/health-strip";
import { FeaturedIncident } from "@/components/overview/featured-incident";
import { ActivityStream } from "@/components/overview/activity-stream";
import { EmptyState } from "@/components/core/empty-state";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { SeverityBadge } from "@/components/core/severity-badge";
import { StatusBadge } from "@/components/core/status-badge";
import { TechnicalId } from "@/components/core/technical-id";
import { IncidentStatus } from "@/components/incident/incident-status";
import { listIncidentInvestigations } from "@/lib/api/investigations";
import { getIncident, getIncidentAlerts, getIncidentTimeline, listIncidents } from "@/lib/api/incidents";
import { listRecommendations } from "@/lib/api/recommendations";
import { listIncidentRemediations, listRemediations } from "@/lib/api/remediations";
import { listServices } from "@/lib/api/services";
import { errorMessage, healthToOperational, isActiveIncident, overallHealth } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { refreshIntervals } from "@/lib/refresh";
import { timelineFromEvents } from "@/lib/timeline";
import type { IncidentDetail, IncidentSummary, ServiceSummary } from "@/lib/api/types";

const severityRank: Record<string, number> = { critical: 4, high: 3, medium: 2, low: 1 };

async function settle<T>(promise: Promise<T>): Promise<T | null> {
  try {
    return await promise;
  } catch {
    return null;
  }
}

async function loadOverview(signal: AbortSignal) {
  const [incidents, services, pending] = await Promise.all([
    listIncidents({ limit: 20 }, { signal }),
    listServices({ limit: 50 }, { signal }),
    listRemediations({ status: "pending_approval", limit: 20 }, { signal }),
  ]);

  const active = incidents.data.filter((incident) => isActiveIncident(incident.status));
  const featuredSummary = [...active].sort((a, b) => {
    const severity = (severityRank[b.severity] ?? 0) - (severityRank[a.severity] ?? 0);
    if (severity !== 0) return severity;
    return new Date(b.detected_at).getTime() - new Date(a.detected_at).getTime();
  })[0];

  let featured: {
    incident: IncidentDetail;
    alerts: string[];
    investigation: Awaited<ReturnType<typeof listIncidentInvestigations>>["data"][number] | null;
    recommendation: Awaited<ReturnType<typeof listRecommendations>>["data"][number] | null;
    remediation: Awaited<ReturnType<typeof listIncidentRemediations>>["data"][number] | null;
    activity: ReturnType<typeof timelineFromEvents>;
    deploymentVersion: string | null;
  } | null = null;

  if (featuredSummary) {
    const id = featuredSummary.reference;
    const [detail, alerts, investigations, recommendations, remediations, timeline] = await Promise.all([
      getIncident(id, { signal }),
      settle(getIncidentAlerts(id, { signal })),
      settle(listIncidentInvestigations(id, { limit: 5 }, { signal })),
      settle(listRecommendations(id, undefined, { signal })),
      settle(listIncidentRemediations(id, { signal })),
      settle(getIncidentTimeline(id, { limit: 12 }, { signal })),
    ]);
    featured = {
      incident: detail.data,
      alerts: alerts?.data.map((alert) => alert.detector_id) ?? [],
      investigation: investigations?.data[0] ?? null,
      recommendation: recommendations?.data[0] ?? null,
      remediation: remediations?.data[0] ?? null,
      activity: timelineFromEvents(timeline?.data ?? []),
      deploymentVersion: null,
    };
  }

  return {
    incidents: incidents.data,
    services: services.data,
    pendingApprovals: pending.data.length,
    featured,
  };
}

export function OverviewView() {
  const resource = useApiResource(loadOverview, {
    refreshMs: refreshIntervals.overviewMs,
    deps: [],
  });

  if (resource.status === "loading") {
    return <LoadingState label="Loading overview" />;
  }
  if (resource.status === "error") {
    return (
      <ErrorState
        title="Unable to load overview"
        description={errorMessage(resource.error, "The control plane did not return system state.")}
        onRetry={resource.reload}
      />
    );
  }

  const { incidents, services, pendingApprovals, featured } = resource.data;
  const active = incidents.filter((incident) => isActiveIncident(incident.status));
  const health = overallHealth(services.map((service) => service.health_status));
  const attention = services.filter(
    (service) => service.health_status === "unhealthy" || service.health_status === "degraded",
  );
  const latestInvestigation = featured?.investigation?.status ?? "None";

  return (
    <div className="space-y-6">
      <HealthStrip
        items={[
          {
            label: "System",
            value: health.label,
            hint:
              attention.length > 0
                ? `${attention.length} of ${services.length} services need attention`
                : `${services.length} watched services`,
            alert: health.tone !== "healthy",
          },
          {
            label: "Incidents",
            value: String(active.length),
            hint: active.length === 1 ? "Active incident" : "Active incidents",
          },
          {
            label: "Investigations",
            value: latestInvestigation.replaceAll("_", " "),
            hint: featured?.investigation ? "Featured incident investigation" : "No investigation activity",
          },
          {
            label: "Approvals",
            value: String(pendingApprovals),
            hint: "Pending remediation approvals",
            href: "/remediations",
          },
        ]}
      />
      <TelemetryStrip />
      {featured ? (
        <FeaturedIncident
          incident={featured.incident}
          alerts={featured.alerts}
          investigation={featured.investigation}
          recommendation={featured.recommendation}
          remediation={featured.remediation}
          deploymentVersion={featured.deploymentVersion}
        />
      ) : (
        <EmptyState
          title="No active incidents"
          description="When detection correlates alerts, the primary incident will appear here."
        />
      )}
      <div className="grid gap-8 lg:grid-cols-[minmax(0,1.4fr)_minmax(16rem,0.8fr)]">
        <section>
          <h2 className="type-section mb-3">Active incidents</h2>
          <ActiveIncidentList
            incidents={active.filter((incident) => incident.id !== featured?.incident.id).slice(0, 6)}
          />
        </section>
        <section>
          <h2 className="type-section mb-3">Service health</h2>
          <AttentionServices services={attention.slice(0, 8)} />
        </section>
      </div>
      <section>
        <h2 className="type-section mb-3">Recent activity</h2>
        <ActivityStream items={featured?.activity.slice(0, 8) ?? []} />
      </section>
    </div>
  );
}

function ActiveIncidentList({ incidents }: { incidents: IncidentSummary[] }) {
  if (incidents.length === 0) {
    return <p className="type-meta">No additional active incidents.</p>;
  }
  return (
    <ul className="divide-y divide-border">
      {incidents.map((incident) => (
        <li key={incident.id}>
          <Link href={`/incidents/${incident.reference}`} className="flex flex-wrap items-center justify-between gap-2 py-2.5 hover:bg-surface/40">
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <TechnicalId value={incident.reference} className="text-text-primary" />
                <TechnicalId value={incident.service_slug} />
              </div>
              <p className="mt-1 truncate text-[13px] text-text-primary">{incident.title}</p>
            </div>
            <div className="flex items-center gap-2">
              <SeverityBadge severity={incident.severity} />
              <IncidentStatus status={incident.status} />
            </div>
          </Link>
        </li>
      ))}
    </ul>
  );
}

function AttentionServices({ services }: { services: ServiceSummary[] }) {
  if (services.length === 0) {
    return <p className="type-meta">Watched services are healthy.</p>;
  }
  return (
    <ul className="divide-y divide-border">
      {services.map((service) => (
        <li key={service.id}>
          <Link href={`/services/${service.id}`} className="flex items-center justify-between gap-3 py-2.5 hover:bg-surface/40">
            <div>
              <p className="text-[13px] font-medium tracking-tight">{service.name}</p>
              <TechnicalId value={service.slug} />
            </div>
            <StatusBadge status={healthToOperational(service.health_status)} />
          </Link>
        </li>
      ))}
    </ul>
  );
}
