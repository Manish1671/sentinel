type IncidentSummaryProps = {
  summary: string;
  alertCount: number;
  investigationStatus: string | null;
  recommendationStatus: string | null;
  remediationStatus: string | null;
};

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-[8rem]">
      <p className="type-label">{label}</p>
      <p className="text-[13px] font-medium capitalize tracking-tight text-text-primary">{value}</p>
    </div>
  );
}

export function IncidentSummary({
  summary,
  alertCount,
  investigationStatus,
  recommendationStatus,
  remediationStatus,
}: IncidentSummaryProps) {
  return (
    <section aria-labelledby="what-happened" className="space-y-4 border-b border-border pb-5">
      <div className="space-y-2">
        <h2 id="what-happened" className="type-section">
          What happened
        </h2>
        <p className="type-body max-w-3xl text-[0.9375rem] leading-6 text-text-primary">{summary}</p>
      </div>
      <div className="flex flex-wrap gap-x-8 gap-y-3">
        <Stat label="Correlated alerts" value={String(alertCount)} />
        <Stat label="Investigation" value={investigationStatus?.replaceAll("_", " ") ?? "Not started"} />
        <Stat label="Recommendation" value={recommendationStatus?.replaceAll("_", " ") ?? "None"} />
        <Stat label="Remediation" value={remediationStatus?.replaceAll("_", " ") ?? "None"} />
      </div>
    </section>
  );
}
