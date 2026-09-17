import { cn } from "@/lib/utils";

type HealthStripItem = {
  label: string;
  value: string;
  hint: string;
  alert?: boolean;
};

export function HealthStrip({ items }: { items: HealthStripItem[] }) {
  return (
    <section aria-label="System health" className="grid gap-px overflow-hidden rounded-md border border-border bg-border sm:grid-cols-5">
      {items.map((item) => (
        <div key={item.label} className="bg-background px-4 py-3">
          <p className="type-label">{item.label}</p>
          <p className={cn("mt-1.5 text-[15px] font-medium tracking-tight", item.alert && "text-status-degraded")}>
            {item.value}
          </p>
          <p className="type-meta mt-1">{item.hint}</p>
        </div>
      ))}
    </section>
  );
}

export function TelemetryStrip() {
  return (
    <section aria-label="Telemetry" className="rounded-md border border-border/80 px-4 py-3">
      <p className="type-label">Telemetry</p>
      <p className="mt-2 text-[15px] font-medium tracking-tight">Telemetry history unavailable</p>
      <p className="type-meta mt-1">
        The control plane does not yet expose historical time-series. Current health, alerts, and the incident
        timeline remain the source of truth.
      </p>
    </section>
  );
}
