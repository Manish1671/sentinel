"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { DataTable, type DataTableColumn } from "@/components/core/data-table";
import { SeverityBadge } from "@/components/core/severity-badge";
import { TechnicalId } from "@/components/core/technical-id";
import { IncidentStatus } from "@/components/incident/incident-status";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { Button } from "@/components/ui/button";
import { listIncidents } from "@/lib/api/incidents";
import type { IncidentSummary } from "@/lib/api/types";
import { formatDateTime } from "@/lib/format";
import { errorMessage } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { refreshIntervals } from "@/lib/refresh";
import { cn } from "@/lib/utils";

const columns: DataTableColumn<IncidentSummary>[] = [
  {
    id: "reference",
    header: "Incident",
    cell: (row) => <TechnicalId value={row.reference} />,
  },
  {
    id: "title",
    header: "Title",
    cell: (row) => <span className="text-[13px] text-text-primary">{row.title}</span>,
  },
  {
    id: "service",
    header: "Service",
    hideOnMobile: true,
    cell: (row) => <TechnicalId value={row.service_slug} />,
  },
  {
    id: "severity",
    header: "Severity",
    cell: (row) => <SeverityBadge severity={row.severity} />,
  },
  {
    id: "status",
    header: "Status",
    hideOnMobile: true,
    cell: (row) => <IncidentStatus status={row.status} />,
  },
  {
    id: "detected",
    header: "Detected",
    hideOnMobile: true,
    cell: (row) => <span className="type-meta">{formatDateTime(row.detected_at)}</span>,
  },
];

const STATUSES = ["", "open", "investigating", "remediating", "verifying", "resolved", "closed"] as const;
const SEVERITIES = ["", "critical", "high", "medium", "low"] as const;

export function IncidentList() {
  const router = useRouter();
  const [status, setStatus] = useState("");
  const [severity, setSeverity] = useState("");
  const [extra, setExtra] = useState<IncidentSummary[]>([]);
  const [extraCursor, setExtraCursor] = useState<string | null>(null);
  const [loadingMore, setLoadingMore] = useState(false);

  const resource = useApiResource(
    (signal) =>
      listIncidents(
        {
          limit: 25,
          status: status || undefined,
          severity: severity || undefined,
        },
        { signal },
      ),
    {
      refreshMs: refreshIntervals.catalogMs,
      deps: [status, severity],
    },
  );

  if (resource.status === "loading") {
    return <LoadingState label="Loading incidents" />;
  }
  if (resource.status === "error") {
    return (
      <ErrorState
        title="Unable to load incidents"
        description={errorMessage(resource.error, "The incident list could not be retrieved.")}
        onRetry={resource.reload}
      />
    );
  }

  const firstPage = resource.data.data;
  const seen = new Set(firstPage.map((row) => row.id));
  const rows = [...firstPage, ...extra.filter((row) => !seen.has(row.id))];
  const nextCursor = extra.length === 0 ? (resource.data.page?.next_cursor ?? null) : extraCursor;

  async function loadMore() {
    if (!nextCursor) return;
    setLoadingMore(true);
    try {
      const page = await listIncidents({
        limit: 25,
        status: status || undefined,
        severity: severity || undefined,
        cursor: nextCursor,
      });
      setExtra((current) => [...current, ...page.data]);
      setExtraCursor(page.page?.next_cursor ?? null);
    } finally {
      setLoadingMore(false);
    }
  }

  function applyFilter(kind: "status" | "severity", value: string) {
    setExtra([]);
    setExtraCursor(null);
    if (kind === "status") setStatus(value);
    else setSeverity(value);
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2">
        {STATUSES.map((value) => (
          <Button
            key={value || "all-status"}
            type="button"
            variant={status === value ? "default" : "outline"}
            size="sm"
            className={cn("capitalize", status === value && "pointer-events-none")}
            onClick={() => applyFilter("status", value)}
          >
            {value || "All statuses"}
          </Button>
        ))}
      </div>
      <div className="flex flex-wrap gap-2">
        {SEVERITIES.map((value) => (
          <Button
            key={value || "all-severity"}
            type="button"
            variant={severity === value ? "default" : "outline"}
            size="sm"
            className={cn("capitalize", severity === value && "pointer-events-none")}
            onClick={() => applyFilter("severity", value)}
          >
            {value || "All severities"}
          </Button>
        ))}
      </div>
      <DataTable
        caption="Incidents"
        columns={columns}
        rows={rows}
        getRowId={(row) => row.id}
        emptyTitle="No incidents match these filters"
        emptyDescription="When detection correlates alerts, incidents will appear here."
        onRowClick={(row) => router.push(`/incidents/${row.reference}`)}
      />
      {nextCursor ? (
        <div className="flex justify-center">
          <Button variant="outline" size="sm" disabled={loadingMore} onClick={() => void loadMore()}>
            {loadingMore ? "Loading…" : "Load more"}
          </Button>
        </div>
      ) : null}
    </div>
  );
}
