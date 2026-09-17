import { cn } from "@/lib/utils";

type DataHintProps = {
  className?: string;
};

export function DataHint({ className }: DataHintProps) {
  return (
    <span
      className={cn("type-meta text-text-muted", className)}
      title="UI development fixtures only. Not live Sentinel telemetry."
    >
      Fixture data
    </span>
  );
}

/** @deprecated Use DataHint in chrome instead of a page banner. */
export function MockBanner({ className }: { className?: string }) {
  return <DataHint className={className} />;
}
