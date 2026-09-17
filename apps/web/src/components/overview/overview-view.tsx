"use client";

import { HealthStrip, TelemetryStrip } from "@/components/overview/health-strip";
import { FeaturedIncident } from "@/components/overview/featured-incident";
import { ActivityStream } from "@/components/overview/activity-stream";
import { EmptyState } from "@/components/core/empty-state";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { listIncidentInvestigations } from "@/lib/api/investigations";
import { getIncident, getIncidentAlerts, getIncidentTimeline, listIncidents } from "@/lib/api/incidents";
import { listRecommendations } from "@/lib/api/recommendations";
import { listIncidentRemediations, listRemediations } from "@/lib/api/remediations";
import { listServices } from "@/lib/api/services";
import { errorMessage, isActiveIncident, overallHealth } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { refreshIntervals } from "@/lib/refresh";
import { timelineFromEvents } from "@/lib/timeline";
import type { IncidentDetail } from "@/lib/api/types";

const severityRank: Record<string, number> = { critical: 4, high: 3, medium: 2, low: 1 };

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
      getIncidentAlerts(id, { signal }),
      listIncidentInvestigations(id, { limit: 5 }, { signal }),
      listRecommendations(id, undefined, { signal }),
      listIncidentRemediations(id, { signal }),
      getIncidentTimeline(id, { limit: 12 }, { signal }),
    ]);
    featured = {
      incident: detail.data,
      alerts: alerts.data.map((alert) => alert.detector_id),
      investigation: investigations.data[0] ?? null,
      recommendation: recommendations.data[0] ?? null,
      remediation: remediations.data[0] ?? null,
      activity: timelineFromEvents(timeline.data),
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
  const activeCount = incidents.filter((incident) => isActiveIncident(incident.status)).length;
  const health = overallHealth(services.map((service) => service.health_status));
  const attention = services.filter(
    (service) => service.health_status === "unhealthy" || service.health_status === "degraded",
  ).length;
  const latestInvestigation = featured?.investigation?.status ?? "None";

  return (
    <div className="space-y-8">
      <HealthStrip
        items={[
          {
            label: "System",
            value: health.label,
            hint:
              attention > 0
                ? `${attention} of ${services.length} services degraded or unhealthy`
                : `${services.length} watched services`,
            alert: health.tone !== "healthy",
          },
          {
            label: "Incidents",
            value: String(activeCount),
            hint: activeCount === 1 ? "Active incident" : "Active incidents",
          },
          {
            label: "Investigations",
            value: latestInvestigation,
            hint: featured?.investigation ? "Latest investigation on the featured incident" : "No investigation activity",
          },
          {
            label: "Approvals",
            value: String(pendingApprovals),
            hint: "Pending remediation approvals",
          },
          {
            label: "Services",
            value: String(services.length),
            hint: "Watched production services",
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
      <section>
        <h2 className="type-section mb-3">Recent activity</h2>
        <ActivityStream items={featured?.activity ?? []} />
      </section>
    </div>
  );
}
