import { CORRELATION_SIGNAL_ORDER, type CorrelationSignals } from "@/lib/correlation";
import { cn } from "@/lib/utils";
import { EmptyState } from "@/components/core/empty-state";
import { GitMerge } from "lucide-react";

type CorrelationSignalsPanelProps = {
  signals: CorrelationSignals | null;
  source?: "timeline" | "derived";
};

export function CorrelationSignalsPanel({ signals, source }: CorrelationSignalsPanelProps) {
  if (!signals) {
    return (
      <EmptyState
        icon={GitMerge}
        title="Correlation signals unavailable"
        description="Alert correlation explanations appear when the incident service records them on the timeline."
        className="py-3"
      />
    );
  }

  return (
    <section aria-labelledby="correlation-heading" className="space-y-3">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 id="correlation-heading" className="type-section">
          Correlation signals
        </h2>
        {source === "derived" ? (
          <p className="type-meta">Derived from attached alerts. Not a score.</p>
        ) : (
          <p className="type-meta">From incident-service correlation metadata. Not a score.</p>
        )}
      </div>
      <ul className="grid gap-2 sm:grid-cols-2">
        {CORRELATION_SIGNAL_ORDER.map(({ key, label }) => {
          const present = signals[key];
          return (
            <li key={key} className="flex items-start gap-2 text-[13px]">
              <span
                className={cn(
                  "mt-0.5 font-mono text-[11px]",
                  present ? "text-success" : "text-text-muted",
                )}
                aria-hidden
              >
                {present ? "✓" : "–"}
              </span>
              <span className={present ? "text-text-primary" : "text-text-muted"}>
                {label}
                <span className="sr-only">{present ? ", matched" : ", not recorded"}</span>
              </span>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
