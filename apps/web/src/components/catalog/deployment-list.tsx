"use client";

import { DataTable, type DataTableColumn } from "@/components/core/data-table";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { StatusBadge } from "@/components/core/status-badge";
import { TechnicalId } from "@/components/core/technical-id";
import { listDeployments } from "@/lib/api/deployments";
import type { DeploymentSummary } from "@/lib/api/types";
import { formatDateTime } from "@/lib/format";
import { deploymentToOperational, errorMessage } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { refreshIntervals } from "@/lib/refresh";

const columns: DataTableColumn<DeploymentSummary>[] = [
  {
    id: "service",
    header: "Service",
    cell: (row) => <TechnicalId value={row.service_slug ?? row.service_id ?? "—"} />,
  },
  {
    id: "version",
    header: "Version",
    cell: (row) => <TechnicalId value={row.version} />,
  },
  {
    id: "sha",
    header: "Git",
    hideOnMobile: true,
    cell: (row) => (row.git_sha ? <TechnicalId value={row.git_sha} /> : <span className="type-meta">—</span>),
  },
  {
    id: "status",
    header: "Status",
    cell: (row) => <StatusBadge status={deploymentToOperational(row.status)} />,
  },
  {
    id: "started",
    header: "Started",
    hideOnMobile: true,
    cell: (row) => <span className="type-meta">{formatDateTime(row.started_at)}</span>,
  },
];

export function DeploymentList() {
  const resource = useApiResource((signal) => listDeployments({ limit: 50 }, { signal }), {
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

  return (
    <DataTable
      caption="Deployments"
      columns={columns}
      rows={resource.data.data}
      getRowId={(row) => row.id}
      emptyTitle="No deployments"
      emptyDescription="Deployment history will list here once catalog events are recorded."
    />
  );
}
