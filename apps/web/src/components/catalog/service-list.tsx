"use client";

import { EmptyState } from "@/components/core/empty-state";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { StatusBadge } from "@/components/core/status-badge";
import { TechnicalId } from "@/components/core/technical-id";
import { listServices } from "@/lib/api/services";
import { errorMessage, healthToOperational } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { refreshIntervals } from "@/lib/refresh";
import { cn } from "@/lib/utils";
import { Boxes } from "lucide-react";

export function ServiceList() {
  const resource = useApiResource((signal) => listServices({ limit: 50 }, { signal }), {
    refreshMs: refreshIntervals.catalogMs,
    deps: [],
  });

  if (resource.status === "loading") {
    return <LoadingState label="Loading services" />;
  }
  if (resource.status === "error") {
    return (
      <ErrorState
        title="Unable to load services"
        description={errorMessage(resource.error, "The service catalog could not be retrieved.")}
        onRetry={resource.reload}
      />
    );
  }

  const services = resource.data.data;
  if (services.length === 0) {
    return (
      <EmptyState
        icon={Boxes}
        title="No watched services"
        description="When the catalog is connected, production services and their health will appear here."
      />
    );
  }

  return (
    <ul className="divide-y divide-border rounded-md border border-border">
      {services.map((service) => (
        <li
          key={service.id}
          className={cn(
            "grid gap-3 px-4 py-3 md:grid-cols-[minmax(0,1.6fr)_auto_auto_auto] md:items-center",
            service.health_status !== "healthy" && "bg-warning/[0.03]",
          )}
        >
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <p className="text-[13px] font-medium tracking-tight">{service.name}</p>
              <TechnicalId value={service.slug} />
            </div>
            <p className="type-meta mt-1 uppercase">{service.environment}</p>
          </div>
          <StatusBadge status={healthToOperational(service.health_status)} />
          {service.current_version ? <TechnicalId value={service.current_version} /> : <span className="type-meta">No version</span>}
          <span className="type-meta">
            {service.active_incident_count} inc
          </span>
        </li>
      ))}
    </ul>
  );
}
