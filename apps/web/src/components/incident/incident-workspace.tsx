"use client";

import { IncidentHeader } from "@/components/incident/incident-header";
import { IncidentSummary } from "@/components/incident/incident-summary";
import { IncidentTimeline } from "@/components/incident/incident-timeline";
import { AlertList } from "@/components/incident/alert-list";
import { EvidenceList } from "@/components/incident/evidence-list";
import { AIInvestigationCard } from "@/components/incident/ai-investigation-card";
import { RecommendationCard } from "@/components/incident/recommendation-card";
import { RemediationCard } from "@/components/incident/remediation-card";
import { HealthComparison } from "@/components/incident/health-comparison";
import { CorrelationSignalsPanel } from "@/components/incident/correlation-signals";
import { ChartEmpty } from "@/components/charts/chart-empty";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { PageFade } from "@/components/motion/page-fade";
import { useAuth } from "@/components/auth/auth-provider";
import { getIncident, getIncidentAlerts, getIncidentTimeline } from "@/lib/api/incidents";
import { getInvestigation, listIncidentInvestigations } from "@/lib/api/investigations";
import { listRecommendations } from "@/lib/api/recommendations";
import { listIncidentRemediations } from "@/lib/api/remediations";
import { getService } from "@/lib/api/services";
import { ApiError } from "@/lib/api/errors";
import type { AlertDetail, IncidentTimelineEvent } from "@/lib/api/types";
import { resolveCorrelation } from "@/lib/correlation";
import { errorMessage, parameterString } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { incidentRefreshMs, remediationRefreshMs } from "@/lib/refresh";
import { timelineFromEvents } from "@/lib/timeline";

type Section<T> = { data: T | null; error: Error | null };

async function settle<T>(promise: Promise<T>): Promise<Section<T>> {
  try {
    return { data: await promise, error: null };
  } catch (error) {
    return { data: null, error: error instanceof Error ? error : new Error("Request failed") };
  }
}

function deploymentFromAlerts(alerts: AlertDetail[]): string | null {
  for (const alert of alerts) {
    const version =
      parameterString(alert.labels as Record<string, unknown> | null, "deployment_version") ??
      parameterString(alert.labels as Record<string, unknown> | null, "version");
    if (version) return version;
  }
  return null;
}

async function loadWorkspace(id: string, signal: AbortSignal) {
  const incident = (await getIncident(id, { signal })).data;
  const [timeline, alerts, investigations, recommendations, remediations, service] = await Promise.all([
    settle(getIncidentTimeline(incident.id, { limit: 50 }, { signal })),
    settle(getIncidentAlerts(incident.id, { signal })),
    settle(listIncidentInvestigations(incident.id, { limit: 5 }, { signal })),
    settle(listRecommendations(incident.id, undefined, { signal })),
    settle(listIncidentRemediations(incident.id, { signal })),
    settle(getService(incident.service_id, { signal })),
  ]);

  const latestInvestigation = investigations.data?.data[0] ?? null;
  const investigationDetail = latestInvestigation
    ? await settle(getInvestigation(latestInvestigation.id, { signal }))
    : { data: null, error: investigations.error };

  const events: IncidentTimelineEvent[] = timeline.data?.data ?? [];
  const alertItems = alerts.data?.data ?? [];
  const investigation = investigationDetail.data?.data ?? null;
  const recommendation = recommendations.data?.data[0] ?? null;
  const remediation = remediations.data?.data[0] ?? null;
  const correlation = resolveCorrelation(events, incident, alertItems);

  return {
    incident,
    timelineItems: timelineFromEvents(events),
    alerts: alertItems,
    investigation,
    recommendation,
    remediation,
    correlation,
    deploymentVersion:
      deploymentFromAlerts(alertItems) ??
      service.data?.data.recent_deployments[0]?.version ??
      service.data?.data.current_version ??
      null,
    errors: {
      timeline: timeline.error,
      alerts: alerts.error,
      investigation: investigationDetail.error,
      recommendation: recommendations.error,
      remediation: remediations.error,
    },
  };
}

type WorkspaceData = Awaited<ReturnType<typeof loadWorkspace>>;

function sectionState(
  error: Error | null,
  present: unknown,
): "loading" | "empty" | "error" | "success" {
  if (error) return "error";
  if (!present) return "empty";
  return "success";
}

export function IncidentWorkspace({ incidentId }: { incidentId: string }) {
  const { user } = useAuth();
  const resource = useApiResource((signal) => loadWorkspace(incidentId, signal), {
    deps: [incidentId],
    refreshMs: (data: WorkspaceData) =>
      Math.min(incidentRefreshMs(data.incident.status), remediationRefreshMs(data.remediation?.status)),
  });

  if (resource.status === "loading") {
    return <LoadingState label="Loading incident" />;
  }
  if (resource.status === "error") {
    const notFound = resource.error instanceof ApiError && resource.error.status === 404;
    return (
      <ErrorState
        title={notFound ? `Incident ${incidentId} was not found` : "Unable to load incident"}
        description={errorMessage(resource.error, "Retry to fetch the incident from the control plane.")}
        onRetry={resource.reload}
      />
    );
  }

  const data = resource.data;

  return (
    <PageFade>
      <div className="space-y-7">
        <IncidentHeader incident={data.incident} deploymentVersion={data.deploymentVersion} />
        <IncidentSummary
          summary={data.incident.summary || "No summary has been recorded for this incident."}
          alertCount={data.alerts.length || data.incident.alert_ids.length}
          investigationStatus={data.investigation?.status ?? null}
          recommendationStatus={data.recommendation?.status ?? null}
          remediationStatus={data.remediation?.status ?? null}
        />

        <div className="grid gap-8 xl:grid-cols-[minmax(0,1.55fr)_minmax(18rem,0.85fr)]">
          <div className="space-y-8">
            {data.errors.timeline ? (
              <ErrorState
                title="Timeline unavailable"
                description={errorMessage(data.errors.timeline, "The control plane did not return timeline events.")}
                onRetry={resource.reload}
              />
            ) : (
              <section className="space-y-3">
                <h2 className="type-section">Timeline</h2>
                <IncidentTimeline items={data.timelineItems} />
              </section>
            )}

            <CorrelationSignalsPanel signals={data.correlation.signals} source={data.correlation.source} />

            {data.errors.alerts ? (
              <ErrorState
                title="Alerts unavailable"
                description={errorMessage(data.errors.alerts, "Correlated alerts could not be loaded.")}
                onRetry={resource.reload}
              />
            ) : (
              <section className="space-y-3">
                <h2 className="type-section">Correlated alerts</h2>
                <p className="type-meta">Incident → alerts attached by the incident service.</p>
                <AlertList alerts={data.alerts} serviceSlug={data.incident.service_slug} />
              </section>
            )}

            <section className="space-y-3">
              <h2 className="type-section">Telemetry</h2>
              <ChartEmpty />
            </section>

            <section className="space-y-3">
              <h2 className="type-section">Observed evidence</h2>
              {data.errors.investigation ? (
                <ErrorState
                  title="Evidence unavailable"
                  description="Evidence is loaded with the investigation. That request failed."
                  onRetry={resource.reload}
                />
              ) : (
                <EvidenceList evidence={data.investigation?.evidence ?? []} />
              )}
            </section>
          </div>

          <aside className="space-y-6 xl:sticky xl:top-6 xl:self-start">
            <AIInvestigationCard
              investigation={data.investigation}
              state={sectionState(data.errors.investigation, data.investigation)}
            />
            <RecommendationCard
              recommendation={data.recommendation}
              remediation={data.remediation}
              role={user.role}
              onChanged={resource.reload}
              state={sectionState(data.errors.recommendation, data.recommendation)}
            />
            <RemediationCard
              remediation={data.remediation}
              state={sectionState(data.errors.remediation, data.remediation)}
            />
            <section className="space-y-3">
              <h2 className="type-section">Recovery</h2>
              <HealthComparison verificationDetails={data.remediation?.verification_details ?? null} />
            </section>
          </aside>
        </div>
      </div>
    </PageFade>
  );
}

