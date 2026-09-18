"use client";

import Link from "next/link";
import { EmptyState } from "@/components/core/empty-state";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { StatusBadge } from "@/components/core/status-badge";
import { TechnicalId } from "@/components/core/technical-id";
import { listDeployments } from "@/lib/api/deployments";
import { listIncidents } from "@/lib/api/incidents";
import type { DeploymentSummary, IncidentSummary } from "@/lib/api/types";
import { formatDateTime } from "@/lib/format";
import { deploymentToOperational, errorMessage } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { refreshIntervals } from "@/lib/refresh";

export function associatedIncident(
  deployment: DeploymentSummary,
  incidents: IncidentSummary[],
): IncidentSummary | null {
  const sameService = incidents.filter((incident) => {
    if (deployment.service_id && incident.service_id === deployment.service_id) return true;
    if (deployment.service_slug && incident.service_slug === deployment.service_slug) return true;
    return false;
  });
  const byTitle = sameService.find((incident) => incident.title.includes(deployment.version));
  if (byTitle) return byTitle;
  const started = Date.parse(deployment.started_at);
  if (!Number.isFinite(started)) return null;
  const completed = deployment.completed_at ? Date.parse(deployment.completed_at) : started;
  const windowEnd = (Number.isFinite(completed) ? completed : started) + 36 * 60 * 60 * 1000;
  return (
    sameService
      .filter((incident) => {
        const detected = Date.parse(incident.detected_at);
        return detected >= started && detected <= windowEnd;
      })
      .sort((a, b) => Date.parse(a.detected_at) - Date.parse(b.detected_at))[0] ?? null
  );
}

async function loadDeployments(signal: AbortSignal) {
  const [deployments, incidents] = await Promise.all([
    listDeployments({ limit: 50 }, { signal }),
    listIncidents({ limit: 50 }, { signal }).catch(() => null),
  ]);
  return {
    deployments: deployments.data,
    incidents: incidents?.data ?? [],
  };
}

export function DeploymentList() {
  const resource = useApiResource(loadDeployments, {
    refreshMs: refreshIntervals.catalogMs,
    deps: [],
  });

  if (resource.status === "loading") {
    return <LoadingState label="Loading deployments" />;
  }
  if (resource.status === "error") {
    return (
      <ErrorState
        title="Unable to load deployments"
        description={errorMessage(resource.error, "Deployment history could not be retrieved.")}
        onRetry={resource.reload}
      />
    );
  }

  const { deployments, incidents } = resource.data;
  if (deployments.length === 0) {
    return (
      <EmptyState
        title="No deployments"
        description="Deployment history will list here once catalog events are recorded."
      />
    );
  }

  return (
    <ul className="divide-y divide-border">
      {deployments.map((row) => {
        const incident = associatedIncident(row, incidents);
        return (
          <li key={row.id} className="flex flex-col gap-2 py-3 md:flex-row md:items-center md:justify-between">
            <div className="space-y-1">
              <div className="flex flex-wrap items-center gap-2">
                <TechnicalId value={row.version} className="text-text-primary" />
                {row.service_slug ? (
                  <Link href={`/services/${row.service_id ?? row.service_slug}`} className="hover:underline">
                    <TechnicalId value={row.service_slug} />
                  </Link>
                ) : (
                  <span className="type-meta">Service not available</span>
                )}
                <span className="type-meta uppercase">{row.environment ?? "Not recorded"}</span>
                <StatusBadge status={deploymentToOperational(row.status)} />
              </div>
              <p className="type-meta">
                Started {formatDateTime(row.started_at)}
                {row.completed_at ? ` · Completed ${formatDateTime(row.completed_at)}` : " · Completion not recorded"}
              </p>
              {row.git_sha ? (
                <p className="type-meta">
                  Git <TechnicalId value={row.git_sha} />
                </p>
              ) : (
                <p className="type-meta">Deployment details unavailable.</p>
              )}
            </div>
            {incident ? (
              <Link href={`/incidents/${incident.reference}`} className="text-[13px] font-medium text-brand hover:underline">
                {incident.reference}
              </Link>
            ) : (
              <span className="type-meta">No associated incident recorded</span>
            )}
          </li>
        );
      })}
    </ul>
  );
}
