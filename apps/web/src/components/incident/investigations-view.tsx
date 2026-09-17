"use client";

import { AIInvestigationCard } from "@/components/incident/ai-investigation-card";
import { EmptyState } from "@/components/core/empty-state";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { getInvestigation, listInvestigations } from "@/lib/api/investigations";
import { errorMessage } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { refreshIntervals } from "@/lib/refresh";

async function loadInvestigations(signal: AbortSignal) {
  const list = await listInvestigations({ limit: 20 }, { signal });
  const latest = list.data[0];
  if (!latest) return { items: list.data, latest: null };
  const detail = await getInvestigation(latest.id, { signal });
  return { items: list.data, latest: detail.data };
}

export function InvestigationsView() {
  const resource = useApiResource(loadInvestigations, {
    refreshMs: refreshIntervals.catalogMs,
    deps: [],
  });

  if (resource.status === "loading") {
    return <LoadingState label="Loading investigations" />;
  }
  if (resource.status === "error") {
    return (
      <ErrorState
        title="Unable to load investigations"
        description={errorMessage(resource.error, "Investigation records could not be retrieved.")}
        onRetry={resource.reload}
      />
    );
  }

  const { items, latest } = resource.data;
  if (!latest) {
    return (
      <EmptyState
        title="Investigation has not been run"
        description="Request an investigation from an incident workspace."
      />
    );
  }

  return (
    <div className="max-w-xl space-y-3">
      <p className="type-meta">{items.length} investigation{items.length === 1 ? "" : "s"} recorded</p>
      <AIInvestigationCard investigation={latest} />
    </div>
  );
}
