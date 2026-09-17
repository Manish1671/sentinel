"use client";

import { RecommendationCard } from "@/components/incident/recommendation-card";
import { RemediationCard } from "@/components/incident/remediation-card";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { EmptyState } from "@/components/core/empty-state";
import { StatusBadge } from "@/components/core/status-badge";
import { useAuth } from "@/components/auth/auth-provider";
import { getRecommendation } from "@/lib/api/recommendations";
import { listRemediations } from "@/lib/api/remediations";
import type { RecommendationDetail, RemediationDetail } from "@/lib/api/types";
import { errorMessage, remediationLabel } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { remediationRefreshMs } from "@/lib/refresh";
import { cn } from "@/lib/utils";
import { useState } from "react";

type RemediationItem = {
  remediation: RemediationDetail;
  recommendation: RecommendationDetail | null;
};

function rank(status: string): number {
  if (status === "pending_approval") return 0;
  if (status === "approved" || status === "running" || status === "verifying") return 1;
  return 2;
}

async function loadRemediations(signal: AbortSignal): Promise<RemediationItem[]> {
  const list = await listRemediations({ limit: 20 }, { signal });
  const items = await Promise.all(
    list.data.map(async (remediation) => {
      const recommendation = await getRecommendation(remediation.recommendation_id, { signal }).catch(() => null);
      return { remediation, recommendation: recommendation?.data ?? null };
    }),
  );
  return [...items].sort((a, b) => rank(a.remediation.status) - rank(b.remediation.status));
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

  const selected =
    resource.data.find((item) => item.remediation.id === selectedId) ?? resource.data[0];

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_minmax(18rem,0.9fr)]">
      <ul className="divide-y divide-border rounded-md border border-border">
        {resource.data.map((item) => {
          const active = item.remediation.id === selected.remediation.id;
          return (
            <li key={item.remediation.id}>
              <button
                type="button"
                onClick={() => setSelectedId(item.remediation.id)}
                className={cn(
                  "flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-surface/80",
                  active && "bg-surface",
                )}
              >
                <div className="min-w-0 flex-1">
                  <p className="text-[13px] font-medium tracking-tight">
                    {item.recommendation?.title ?? item.remediation.action_type.replaceAll("_", " ")}
                  </p>
                  <p className="type-meta mt-1 capitalize">{remediationLabel(item.remediation.status)}</p>
                </div>
                <StatusBadge
                  status={
                    item.remediation.status === "succeeded"
                      ? "healthy"
                      : item.remediation.status === "failed" || item.remediation.status === "rejected"
                        ? "failed"
                        : item.remediation.status === "pending_approval"
                          ? "pending"
                          : "running"
                  }
                />
              </button>
            </li>
          );
        })}
      </ul>
      <div className="space-y-5">
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
