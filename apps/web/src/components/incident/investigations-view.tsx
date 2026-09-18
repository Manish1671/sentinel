"use client";

import Link from "next/link";
import { HypothesisBlock } from "@/components/ai/hypothesis-block";
import { EmptyState } from "@/components/core/empty-state";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { TechnicalId } from "@/components/core/technical-id";
import { listInvestigations } from "@/lib/api/investigations";
import { formatDateTime, formatPercent } from "@/lib/format";
import { errorMessage } from "@/lib/health";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { refreshIntervals } from "@/lib/refresh";

export function InvestigationsView() {
  const resource = useApiResource((signal) => listInvestigations({ limit: 30 }, { signal }), {
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

  const items = resource.data.data;
  if (items.length === 0) {
    return (
      <EmptyState
        title="Investigation has not been run"
        description="Request an investigation from an incident workspace."
      />
    );
  }

  return (
    <ul className="divide-y divide-border">
      {items.map((item) => {
        const incidentHref = item.incident_reference
          ? `/incidents/${item.incident_reference}`
          : `/incidents/${item.incident_id}`;
        return (
          <li key={item.id} className="space-y-3 py-4">
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <div className="flex flex-wrap items-center gap-2">
                <TechnicalId value={item.id} />
                <Link href={incidentHref} className="hover:text-text-primary">
                  <TechnicalId value={item.incident_reference || item.incident_id} className="text-text-primary" />
                </Link>
                <span className="type-meta capitalize">{item.status.replaceAll("_", " ")}</span>
              </div>
              <span className="type-meta">
                {item.completed_at
                  ? `Completed ${formatDateTime(item.completed_at)}`
                  : `Requested ${formatDateTime(item.requested_at)}`}
              </span>
            </div>
            {item.root_cause_hypothesis ? (
              <HypothesisBlock title="Inference / hypothesis">
                <p className="text-[13px] leading-6 text-text-primary">{item.root_cause_hypothesis}</p>
                <p className="type-meta">Inference from observed evidence. Not verified fact.</p>
              </HypothesisBlock>
            ) : (
              <p className="type-meta">Hypothesis is not available yet.</p>
            )}
            {item.confidence != null ? (
              <p className="type-meta">
                Confidence {formatPercent(item.confidence)} · {item.status === "completed" ? "completed investigation" : item.status}
              </p>
            ) : (
              <p className="type-meta">Confidence has not been reported.</p>
            )}
            {item.status === "completed" ? (
              <Link href={incidentHref} className="inline-flex text-[13px] font-medium text-brand hover:underline">
                Open incident workspace
              </Link>
            ) : null}
          </li>
        );
      })}
    </ul>
  );
}
