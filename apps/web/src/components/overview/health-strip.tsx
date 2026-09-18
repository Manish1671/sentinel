import { cn } from "@/lib/utils";
import Link from "next/link";

type HealthStripItem = {
  label: string;
  value: string;
  hint: string;
  alert?: boolean;
  href?: string;
};

export function HealthStrip({ items }: { items: HealthStripItem[] }) {
  return (
    <section aria-label="System health" className="grid gap-px overflow-hidden rounded-md border border-border bg-border sm:grid-cols-2 lg:grid-cols-4">
      {items.map((item) => {
        const inner = (
          <>
            <p className="type-label">{item.label}</p>
            <p className={cn("mt-1 text-[15px] font-medium tracking-tight capitalize", item.alert && "text-status-degraded")}>
              {item.value}
            </p>
            <p className="type-meta mt-1">{item.hint}</p>
          </>
        );
        return item.href ? (
          <Link key={item.label} href={item.href} className="bg-background px-4 py-3 hover:bg-surface/80">
            {inner}
          </Link>
        ) : (
          <div key={item.label} className="bg-background px-4 py-3">
            {inner}
          </div>
        );
      })}
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
