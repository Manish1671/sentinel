"use client";

import Link from "next/link";
import { useMemo, useState } from "react";
import { RecommendationCard } from "@/components/incident/recommendation-card";
import { RemediationCard } from "@/components/incident/remediation-card";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { EmptyState } from "@/components/core/empty-state";
import { TechnicalId } from "@/components/core/technical-id";
import { useAuth } from "@/components/auth/auth-provider";
import { getRecommendation } from "@/lib/api/recommendations";
import { listRemediations } from "@/lib/api/remediations";
import type { RecommendationDetail, RemediationDetail } from "@/lib/api/types";
import { errorMessage, remediationLabel } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { remediationRefreshMs } from "@/lib/refresh";
import { cn } from "@/lib/utils";

type RemediationItem = {
  remediation: RemediationDetail;
  recommendation: RecommendationDetail | null;
};

const GROUPS: Array<{ id: string; label: string; match: (status: string) => boolean }> = [
  { id: "pending_approval", label: "Pending approval", match: (status) => status === "pending_approval" },
  { id: "running", label: "Running", match: (status) => status === "approved" || status === "running" },
  { id: "verifying", label: "Verifying", match: (status) => status === "verifying" },
  { id: "succeeded", label: "Succeeded", match: (status) => status === "succeeded" },
  { id: "failed", label: "Failed", match: (status) => status === "failed" },
  { id: "rejected", label: "Rejected", match: (status) => status === "rejected" || status === "cancelled" },
];

async function loadRemediations(signal: AbortSignal): Promise<RemediationItem[]> {
  const list = await listRemediations({ limit: 40 }, { signal });
  return Promise.all(
    list.data.map(async (remediation) => {
      const recommendation = await getRecommendation(remediation.recommendation_id, { signal }).catch(() => null);
      return { remediation, recommendation: recommendation?.data ?? null };
    }),
  );
}

export function RemediationsView() {
  const { user } = useAuth();
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const resource = useApiResource(loadRemediations, {
    deps: [],
    refreshMs: (data) => {
      const active = data.find((item) =>
        ["pending_approval", "approved", "running", "verifying"].includes(item.remediation.status),
      );
      return remediationRefreshMs(active?.remediation.status ?? data[0]?.remediation.status);
    },
  });

  const grouped = useMemo(() => {
    if (resource.status !== "success") return [];
    return GROUPS.map((group) => ({
      ...group,
      items: resource.data.filter((item) => group.match(item.remediation.status)),
    })).filter((group) => group.items.length > 0);
  }, [resource]);

  if (resource.status === "loading") {
    return <LoadingState label="Loading remediations" />;
  }
  if (resource.status === "error") {
    return (
      <ErrorState
        title="Unable to load remediations"
        description={errorMessage(resource.error, "Remediation records could not be retrieved.")}
        onRetry={resource.reload}
      />
    );
  }

  if (resource.data.length === 0) {
    return (
      <EmptyState
        title="No remediations"
        description="Allowlisted actions appear here after an investigation proposes a change."
      />
    );
  }

  const pending = resource.data.filter((item) => item.remediation.status === "pending_approval");
  const selected =
    resource.data.find((item) => item.remediation.id === selectedId) ??
    pending[0] ??
    resource.data[0];

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1.15fr)_minmax(18rem,0.85fr)]">
      <div className="space-y-6">
        {grouped.map((group) => (
          <section key={group.id} className="space-y-2">
            <h2 className={cn("type-section", group.id === "pending_approval" && "text-warning")}>
              {group.label}
              <span className="ml-2 type-meta"> {group.items.length}</span>
            </h2>
            <ul className={cn(group.id === "pending_approval" && "border-l-2 border-warning/50 pl-3")}>
              {group.items.map((item) => {
                const active = item.remediation.id === selected.remediation.id;
                const incidentRef = item.remediation.incident_reference;
                return (
                  <li key={item.remediation.id}>
                    <button
                      type="button"
                      onClick={() => setSelectedId(item.remediation.id)}
                      className={cn(
                        "flex w-full items-start gap-3 py-3 text-left transition-colors hover:bg-surface/80",
                        active && "bg-surface",
                      )}
                    >
                      <div className="min-w-0 flex-1">
                        <p className="text-[13px] font-medium tracking-tight">
                          {item.recommendation?.title ?? item.remediation.action_type.replaceAll("_", " ")}
                        </p>
                        <p className="type-meta mt-1 capitalize">
                          {remediationLabel(item.remediation.status)}
                          {item.remediation.service_slug ? ` · ${item.remediation.service_slug}` : ""}
                        </p>
                      </div>
                      {incidentRef ? (
                        <Link
                          href={`/incidents/${incidentRef}`}
                          onClick={(event) => event.stopPropagation()}
                          className="shrink-0 hover:underline"
                        >
                          <TechnicalId value={incidentRef} />
                        </Link>
                      ) : null}
                    </button>
                  </li>
                );
              })}
            </ul>
          </section>
        ))}
      </div>
      <div className="space-y-5 xl:sticky xl:top-6 xl:self-start">
        <RecommendationCard
          recommendation={selected.recommendation}
          remediation={selected.remediation}
          role={user.role}
          onChanged={resource.reload}
        />
        <RemediationCard remediation={selected.remediation} />
      </div>
    </div>
  );
}
