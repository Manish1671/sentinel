import { ConfidenceMeter } from "@/components/ai/confidence-meter";
import { HypothesisBlock } from "@/components/ai/hypothesis-block";
import { EmptyState } from "@/components/core/empty-state";
import { ErrorState } from "@/components/core/error-state";
import { LoadingState } from "@/components/core/loading-state";
import { TechnicalId } from "@/components/core/technical-id";
import type { InvestigationDetail } from "@/lib/api/types";
import { SearchX } from "lucide-react";

type AIInvestigationCardProps = {
  investigation?: InvestigationDetail | null;
  state?: "loading" | "empty" | "error" | "success";
};

function confidenceBasis(investigation: InvestigationDetail): string {
  const count = investigation.evidence?.length ?? 0;
  if (count === 0) {
    return "Numerical estimate only. Not a guarantee of root cause.";
  }
  return `Based on ${count} evidence item${count === 1 ? "" : "s"} collected by investigation tools. Not a guarantee.`;
}

export function AIInvestigationCard({
  investigation,
  state = investigation ? "success" : "empty",
}: AIInvestigationCardProps) {
  if (state === "loading") {
    return <LoadingState label="Loading investigation" />;
  }
  if (state === "error") {
    return (
      <ErrorState
        title="Investigation unavailable"
        description="The control plane did not return an investigation for this incident."
      />
    );
  }
  if (state === "empty" || !investigation) {
    return (
      <EmptyState
        icon={SearchX}
        title="No investigation has been run for this incident."
        description="A completed investigation produces a bounded hypothesis from observed evidence. That is inference, not verified fact."
      />
    );
  }

  const tools = (investigation.tool_usage ?? []).filter((item) => item.tool);

  return (
    <section aria-labelledby="investigation-heading" className="space-y-4">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 id="investigation-heading" className="type-section">
          AI investigation
        </h2>
        <p className="type-meta capitalize">Status {investigation.status.replaceAll("_", " ")}</p>
      </div>

      <HypothesisBlock title="Root cause hypothesis">
        {investigation.root_cause_hypothesis ? (
          <p className="text-[13px] leading-6 text-text-primary">{investigation.root_cause_hypothesis}</p>
        ) : (
          <p className="type-body">Hypothesis is not available yet.</p>
        )}
        <p className="type-meta">Inference from observed evidence. Not verified fact.</p>
      </HypothesisBlock>

      {investigation.confidence != null ? (
        <ConfidenceMeter value={investigation.confidence} basis={confidenceBasis(investigation)} />
      ) : (
        <p className="type-meta">Confidence has not been reported.</p>
      )}

      <div>
        <p className="type-label mb-1">Reasoning summary</p>
        {investigation.reasoning_summary ? (
          <p className="type-body">{investigation.reasoning_summary}</p>
        ) : (
          <p className="type-meta">No reasoning summary was stored.</p>
        )}
      </div>

      <div>
        <p className="type-label mb-1">Tools used</p>
        {tools.length ? (
          <ul className="flex flex-wrap gap-2">
            {tools.map((item) => (
              <li key={item.tool}>
                <TechnicalId value={item.tool} />
                {item.call_count != null ? <span className="type-meta ml-1">×{item.call_count}</span> : null}
              </li>
            ))}
          </ul>
        ) : (
          <p className="type-meta">No tool usage recorded.</p>
        )}
      </div>
    </section>
  );
}
