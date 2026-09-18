"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { SeverityBadge } from "@/components/core/severity-badge";
import { TechnicalId } from "@/components/core/technical-id";
import { IncidentStatus } from "@/components/incident/incident-status";
import { EmptyState } from "@/components/core/empty-state";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { Button } from "@/components/ui/button";
import { listIncidents } from "@/lib/api/incidents";
import { listServices } from "@/lib/api/services";
import type { IncidentSummary } from "@/lib/api/types";
import { formatDateTime } from "@/lib/format";
import { errorMessage, isActiveIncident } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { refreshIntervals } from "@/lib/refresh";
import { cn } from "@/lib/utils";

const STATUSES = ["", "open", "investigating", "remediating", "verifying", "resolved", "closed"] as const;
const SEVERITIES = ["", "critical", "high", "medium", "low"] as const;
const severityRank: Record<string, number> = { critical: 4, high: 3, medium: 2, low: 1 };

function sortIncidents(rows: IncidentSummary[]): IncidentSummary[] {
  return [...rows].sort((a, b) => {
    const active = Number(isActiveIncident(b.status)) - Number(isActiveIncident(a.status));
    if (active !== 0) return active;
    const severity = (severityRank[b.severity] ?? 0) - (severityRank[a.severity] ?? 0);
    if (severity !== 0) return severity;
    return new Date(b.detected_at).getTime() - new Date(a.detected_at).getTime();
  });
}

export function IncidentList() {
  const [status, setStatus] = useState("");
  const [severity, setSeverity] = useState("");
  const [serviceId, setServiceId] = useState("");
  const [extra, setExtra] = useState<IncidentSummary[]>([]);
  const [extraCursor, setExtraCursor] = useState<string | null>(null);
  const [loadingMore, setLoadingMore] = useState(false);

  const services = useApiResource((signal) => listServices({ limit: 50 }, { signal }), {
    refreshMs: refreshIntervals.catalogMs,
    deps: [],
  });

  const resource = useApiResource(
    (signal) =>
      listIncidents(
        {
          limit: 25,
          status: status || undefined,
          severity: severity || undefined,
          service_id: serviceId || undefined,
        },
        { signal },
      ),
    {
      refreshMs: refreshIntervals.catalogMs,
      deps: [status, severity, serviceId],
    },
  );

  const serviceOptions = services.status === "success" ? services.data.data : [];

  const rows = useMemo(() => {
    if (resource.status !== "success") return [];
    const firstPage = resource.data.data;
    const seen = new Set(firstPage.map((row) => row.id));
    return sortIncidents([...firstPage, ...extra.filter((row) => !seen.has(row.id))]);
  }, [resource, extra]);

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

  const nextCursor = extra.length === 0 ? (resource.data.page?.next_cursor ?? null) : extraCursor;

  async function loadMore() {
    if (!nextCursor) return;
    setLoadingMore(true);
    try {
      const page = await listIncidents({
        limit: 25,
        status: status || undefined,
        severity: severity || undefined,
        service_id: serviceId || undefined,
        cursor: nextCursor,
      });
      setExtra((current) => [...current, ...page.data]);
      setExtraCursor(page.page?.next_cursor ?? null);
    } finally {
      setLoadingMore(false);
    }
  }

  function resetPaging() {
    setExtra([]);
    setExtraCursor(null);
  }

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap gap-2" role="group" aria-label="Filter by status">
        {STATUSES.map((value) => (
          <Button
            key={value || "all-status"}
            type="button"
            variant={status === value ? "default" : "outline"}
            size="sm"
            className={cn("capitalize", status === value && "pointer-events-none")}
            aria-pressed={status === value}
            onClick={() => {
              resetPaging();
              setStatus(value);
            }}
          >
            {value || "All statuses"}
          </Button>
        ))}
      </div>
      <div className="flex flex-wrap gap-2" role="group" aria-label="Filter by severity">
        {SEVERITIES.map((value) => (
          <Button
            key={value || "all-severity"}
            type="button"
            variant={severity === value ? "default" : "outline"}
            size="sm"
            className={cn("capitalize", severity === value && "pointer-events-none")}
            aria-pressed={severity === value}
            onClick={() => {
              resetPaging();
              setSeverity(value);
            }}
          >
            {value || "All severities"}
          </Button>
        ))}
      </div>
      {serviceOptions.length ? (
        <div className="flex flex-wrap gap-2" role="group" aria-label="Filter by service">
          <Button
            type="button"
            variant={serviceId === "" ? "default" : "outline"}
            size="sm"
            onClick={() => {
              resetPaging();
              setServiceId("");
            }}
          >
            All services
          </Button>
          {serviceOptions
            .filter((service) => service.active_incident_count > 0)
            .slice(0, 10)
            .map((service) => (
              <Button
                key={service.id}
                type="button"
                variant={serviceId === service.id ? "default" : "outline"}
                size="sm"
                onClick={() => {
                  resetPaging();
                  setServiceId(service.id);
                }}
              >
                {service.slug}
              </Button>
            ))}
        </div>
      ) : null}

      {rows.length === 0 ? (
        <EmptyState
          title="No incidents match these filters"
          description="When detection correlates alerts, incidents will appear here."
        />
      ) : (
        <ul className="divide-y divide-border">
          {rows.map((row) => (
            <li key={row.id} className="flex flex-col gap-2 py-3 md:flex-row md:items-center md:justify-between">
              <div className="min-w-0 space-y-1">
                <div className="flex flex-wrap items-center gap-2">
                  <Link href={`/incidents/${row.reference}`} className="hover:underline">
                    <TechnicalId value={row.reference} className="text-text-primary" />
                  </Link>
                  <Link href={`/services/${row.service_id}`} className="hover:underline">
                    <TechnicalId value={row.service_slug} />
                  </Link>
                  <span className="type-meta uppercase">{row.environment}</span>
                </div>
                <Link href={`/incidents/${row.reference}`} className="text-[13px] font-medium tracking-tight text-text-primary hover:underline">
                  {row.title}
                </Link>
              </div>
              <div className="flex flex-wrap items-center gap-3">
                <SeverityBadge severity={row.severity} />
                <IncidentStatus status={row.status} />
                <time className="type-meta" dateTime={row.detected_at}>
                  {formatDateTime(row.detected_at)}
                </time>
              </div>
            </li>
          ))}
        </ul>
      )}
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
