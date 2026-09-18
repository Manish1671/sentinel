"use client";

import { StatusBadge } from "@/components/core/status-badge";
import { controlPlaneLabel, getControlPlaneReady } from "@/lib/api/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { refreshIntervals } from "@/lib/refresh";
import { cn } from "@/lib/utils";

export function SystemStatusIndicator({ compact = false }: { compact?: boolean }) {
  const resource = useApiResource((signal) => getControlPlaneReady({ signal }), {
    refreshMs: refreshIntervals.systemStatusMs,
    deps: [],
  });

  const ready = resource.status === "success" ? resource.data : null;
  const error = resource.status === "error" ? resource.error : null;
  const checking = resource.status === "loading";
  const { label, tone } = checking
    ? { label: "Checking", tone: "pending" as const }
    : controlPlaneLabel(ready, error);
  const status = tone === "healthy" ? "healthy" : tone === "degraded" ? "degraded" : tone === "pending" ? "pending" : "failed";

  if (compact) {
    return (
      <span className="inline-flex items-center gap-1.5" title={`Control plane ${label}`}>
        <StatusBadge status={status} />
        <span className="sr-only">Control plane {label}</span>
        <span className={cn("type-meta hidden lg:inline", tone === "failed" && "text-danger")}>{label}</span>
      </span>
    );
  }

  return (
    <section aria-labelledby="system-status-heading" className="space-y-3">
      <div className="flex flex-wrap items-center gap-2">
        <h2 id="system-status-heading" className="type-section">
          Control plane
        </h2>
        <StatusBadge status={status} />
        <span className="type-meta">{label}</span>
      </div>
      <p className="type-body">
        {label === "Operational"
          ? "PostgreSQL (and Redis, when enabled) responded to the readiness probe."
          : label === "Degraded"
            ? "The API process is up but a dependency failed the readiness probe."
            : label === "Checking"
              ? "Probing GET /ready on the control plane."
              : "The control-plane readiness endpoint could not be reached. Do not treat the console as operational."}
      </p>
      {ready?.checks ? (
        <dl className="grid gap-2 sm:grid-cols-2">
          {Object.entries(ready.checks).map(([name, value]) => (
            <div key={name} className="flex items-center justify-between gap-3 border-b border-border py-2">
              <dt className="type-label">{name}</dt>
              <dd className="type-meta capitalize">{value}</dd>
            </div>
          ))}
        </dl>
      ) : null}
    </section>
  );
}
