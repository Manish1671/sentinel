import { EmptyState } from "@/components/core/empty-state";
import { afterSnapshotFromVerification, hasBeforeAfterPair } from "@/lib/verification";
import { Activity } from "lucide-react";

type HealthSnapshot = {
  latencyMs: number;
  errorRate: number;
  saturation: number;
};

type HealthComparisonProps = {
  before?: HealthSnapshot;
  after?: HealthSnapshot;
  verificationDetails?: Record<string, unknown> | null;
};

function Cell({ label, value, unit }: { label: string; value: string; unit?: string }) {
  return (
    <div>
      <p className="type-label">{label}</p>
      <p className="type-metric text-lg">
        {value}
        {unit ? <span className="ml-1 font-sans text-[11px] font-normal text-text-muted">{unit}</span> : null}
      </p>
    </div>
  );
}

export function HealthComparison({ before, after, verificationDetails }: HealthComparisonProps) {
  if (before && after) {
    return (
      <div className="space-y-3">
        <section className="rounded-md border border-border px-3 py-3">
          <p className="type-label mb-2">Before</p>
          <div className="grid grid-cols-3 gap-2">
            <Cell label="Latency" value={String(before.latencyMs)} unit="ms" />
            <Cell label="Errors" value={before.errorRate.toFixed(1)} unit="%" />
            <Cell label="DB util" value={String(before.saturation)} unit="%" />
          </div>
        </section>
        <section className="rounded-md border border-border px-3 py-3">
          <p className="type-label mb-2 text-success">After</p>
          <div className="grid grid-cols-3 gap-2">
            <Cell label="Latency" value={String(after.latencyMs)} unit="ms" />
            <Cell label="Errors" value={after.errorRate.toFixed(1)} unit="%" />
            <Cell label="DB util" value={String(after.saturation)} unit="%" />
          </div>
        </section>
      </div>
    );
  }

  const snapshot = afterSnapshotFromVerification(verificationDetails);
  const pair = hasBeforeAfterPair(verificationDetails ?? null);

  if (!pair && snapshot) {
    return (
      <div className="space-y-3">
        <p className="type-meta">Detailed recovery metrics are not currently available as a before/after pair.</p>
        <section className="rounded-md border border-border px-3 py-3">
          <p className="type-label mb-2">Verification snapshot</p>
          <div className="grid grid-cols-3 gap-2">
            <Cell
              label="Latency"
              value={snapshot.latencyMs != null ? String(snapshot.latencyMs) : "—"}
              unit={snapshot.latencyMs != null ? "ms" : undefined}
            />
            <Cell
              label="Errors"
              value={snapshot.errorRate != null ? snapshot.errorRate.toString() : "—"}
            />
            <Cell
              label="DB util"
              value={snapshot.dbUtilization != null ? snapshot.dbUtilization.toString() : "—"}
            />
          </div>
          {snapshot.healthStatus ? <p className="type-meta mt-2">Health {snapshot.healthStatus}</p> : null}
        </section>
      </div>
    );
  }

  return (
    <EmptyState
      icon={Activity}
      title="Detailed recovery metrics are not currently available."
      description="Before/after latency and error values are only shown when the control plane stores both snapshots."
    />
  );
}
