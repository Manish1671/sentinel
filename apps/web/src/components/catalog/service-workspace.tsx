"use client";

import Link from "next/link";
import { AlertList } from "@/components/incident/alert-list";
import { IncidentStatus } from "@/components/incident/incident-status";
import { SeverityBadge } from "@/components/core/severity-badge";
import { StatusBadge } from "@/components/core/status-badge";
import { TechnicalId } from "@/components/core/technical-id";
import { EmptyState } from "@/components/core/empty-state";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { ChartEmpty } from "@/components/charts/chart-empty";
import { getIncidentAlerts, listIncidents } from "@/lib/api/incidents";
import { getService } from "@/lib/api/services";
import type { AlertDetail, IncidentSummary } from "@/lib/api/types";
import { ApiError } from "@/lib/api/errors";
import { formatDateTime } from "@/lib/format";
import { deploymentToOperational, errorMessage, healthToOperational, isActiveIncident } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { refreshIntervals } from "@/lib/refresh";

async function loadServiceWorkspace(id: string, signal: AbortSignal) {
  const service = (await getService(id, { signal })).data;
  const incidents = await listIncidents({ service_id: service.id, limit: 20 }, { signal }).catch(() => null);
  const incidentRows = incidents?.data ?? [];
  const active = incidentRows.filter((incident) => isActiveIncident(incident.status));
  const primary = active[0] ?? incidentRows[0] ?? null;
  let alerts: AlertDetail[] = [];
  if (primary) {
    const attached = await getIncidentAlerts(primary.id, { signal }).catch(() => null);
    alerts = attached?.data ?? [];
  }
  return {
    service,
    incidents: incidentRows,
    alerts,
    alertsSource: primary,
    incidentsError: incidents ? null : new Error("Incidents could not be loaded for this service."),
  };
}

export function ServiceWorkspace({ serviceId }: { serviceId: string }) {
  const resource = useApiResource((signal) => loadServiceWorkspace(serviceId, signal), {
    deps: [serviceId],
    refreshMs: refreshIntervals.catalogMs,
  });

  if (resource.status === "loading") {
    return <LoadingState label="Loading service" />;
  }
  if (resource.status === "error") {
    const status = resource.error instanceof ApiError ? resource.error.status : 0;
    const notFound = status === 404;
    const invalid = status === 400;
    return (
      <ErrorState
        title={
          notFound
            ? `Service ${serviceId} was not found`
            : invalid
              ? `Service ${serviceId} is not a valid identifier`
              : "Unable to load service"
        }
        description={errorMessage(
          resource.error,
          notFound
            ? "The control plane did not return this service."
            : "The service record could not be retrieved.",
        )}
        onRetry={resource.reload}
      />
    );
  }

  const { service, incidents, alerts, alertsSource, incidentsError } = resource.data;
  const active = incidents.filter((incident) => isActiveIncident(incident.status));
  const primary = active[0] ?? null;

  return (
    <div className="space-y-7">
      <header className="space-y-4 border-b border-border pb-5">
        <nav aria-label="Service context" className="flex flex-wrap items-center gap-x-2 gap-y-1 type-meta">
          <Link href="/services" className="hover:text-text-primary">
            Services
          </Link>
          <span aria-hidden>/</span>
          <TechnicalId value={service.slug} className="text-text-primary" />
        </nav>
        <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
          <div className="space-y-2">
            <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
              <TechnicalId value={service.slug} className="text-[0.9375rem] text-text-primary" />
              <span className="type-meta uppercase tracking-[0.08em]">{service.environment}</span>
            </div>
            <h1 className="type-page-title">{service.name}</h1>
            {service.description ? <p className="type-body max-w-3xl">{service.description}</p> : null}
          </div>
          <div className="flex flex-wrap items-center gap-3 rounded-md border border-border px-3 py-2">
            <StatusBadge status={healthToOperational(service.health_status)} />
            {service.current_version ? <TechnicalId value={service.current_version} /> : <span className="type-meta">Version not recorded</span>}
          </div>
        </div>
      </header>

      <section className="flex flex-wrap gap-x-8 gap-y-3 border-b border-border pb-5">
        <div>
          <p className="type-label">Health</p>
          <p className="text-[13px] font-medium capitalize text-text-primary">{service.health_status}</p>
        </div>
        <div>
          <p className="type-label">Active incidents</p>
          <p className="text-[13px] font-medium text-text-primary">{service.active_incident_count}</p>
        </div>
        <div>
          <p className="type-label">Owner</p>
          <p className="type-meta">{service.owner_user_id ? <TechnicalId value={service.owner_user_id} /> : "Not recorded"}</p>
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="type-section">Current incident context</h2>
        {incidentsError ? (
          <ErrorState title="Incidents unavailable" description={incidentsError.message} onRetry={resource.reload} />
        ) : primary ? (
          <PrimaryIncident incident={primary} />
        ) : (
          <EmptyState
            title="No active incident"
            description="When detection correlates alerts for this service, the primary incident appears here."
          />
        )}
        {active.length > 1 ? (
          <ul className="divide-y divide-border">
            {active.slice(1).map((incident) => (
              <li key={incident.id} className="py-2">
                <Link href={`/incidents/${incident.reference}`} className="flex flex-wrap items-center gap-3 hover:text-text-primary">
                  <TechnicalId value={incident.reference} />
                  <span className="text-[13px]">{incident.title}</span>
                  <SeverityBadge severity={incident.severity} />
                </Link>
              </li>
            ))}
          </ul>
        ) : null}
      </section>

      <section className="space-y-3">
        <h2 className="type-section">Recent alerts</h2>
        {alertsSource ? (
          <p className="type-meta">
            From incident <Link href={`/incidents/${alertsSource.reference}`} className="hover:text-text-primary"><TechnicalId value={alertsSource.reference} /></Link>
            . Service-wide alert history is not exposed as a standalone control-plane list.
          </p>
        ) : (
          <p className="type-meta">No recorded data. Alerts appear after they attach to an incident for this service.</p>
        )}
        <AlertList alerts={alerts} serviceSlug={service.slug} />
      </section>

      <section className="space-y-3">
        <h2 className="type-section">Recent deployments</h2>
        {service.recent_deployments.length === 0 ? (
          <EmptyState title="No recorded data" description="Deployment history is not available for this service." />
        ) : (
          <ul className="divide-y divide-border">
            {service.recent_deployments.map((deployment) => (
              <li key={deployment.id} className="flex flex-wrap items-center justify-between gap-3 py-3">
                <div className="flex flex-wrap items-center gap-3">
                  <TechnicalId value={deployment.version} className="text-text-primary" />
                  <StatusBadge status={deploymentToOperational(deployment.status)} />
                </div>
                <div className="flex flex-wrap gap-x-4 gap-y-1">
                  <span className="type-meta">Started {formatDateTime(deployment.started_at)}</span>
                  <span className="type-meta">
                    Completed {deployment.completed_at ? formatDateTime(deployment.completed_at) : "not recorded"}
                  </span>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <section className="space-y-3">
        <h2 className="type-section">Telemetry</h2>
        <ChartEmpty />
      </section>
    </div>
  );
}

function PrimaryIncident({ incident }: { incident: IncidentSummary }) {
  return (
    <div className="rounded-md border border-border border-l-2 border-l-severity-critical px-4 py-4">
      <div className="flex flex-wrap items-center gap-2">
        <TechnicalId value={incident.reference} className="text-text-primary" />
        <SeverityBadge severity={incident.severity} />
        <IncidentStatus status={incident.status} />
      </div>
      <p className="mt-2 text-[15px] font-medium tracking-tight">{incident.title}</p>
      <p className="type-meta mt-1">Detected {formatDateTime(incident.detected_at)}</p>
      <Link
        href={`/incidents/${incident.reference}`}
        className="mt-3 inline-flex text-[13px] font-medium text-brand hover:underline"
      >
        Open incident workspace
      </Link>
    </div>
  );
}
