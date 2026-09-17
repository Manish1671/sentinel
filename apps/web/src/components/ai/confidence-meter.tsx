import { cn } from "@/lib/utils";
import { confidenceBand, formatPercent } from "@/lib/format";

type ConfidenceMeterProps = {
  value: number;
  basis?: string;
  className?: string;
};

export function ConfidenceMeter({ value, basis, className }: ConfidenceMeterProps) {
  const percent = Math.max(0, Math.min(1, value));
  const band = confidenceBand(percent);
  const ticks = 8;
  const filled = Math.round(percent * ticks);

  return (
    <div className={cn("space-y-2", className)}>
      <div className="flex items-end justify-between gap-3">
        <p className="font-mono text-[1.375rem] leading-none tracking-tight tabular-nums">
          {formatPercent(percent)}
        </p>
        <p className="type-meta">{band} confidence</p>
      </div>
      <div
        className="flex gap-0.5"
        role="meter"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(percent * 100)}
        aria-label={`${formatPercent(percent)} ${band} confidence, model estimate`}
      >
        {Array.from({ length: ticks }, (_, index) => (
          <span
            key={index}
            className={cn(
              "h-1 flex-1 rounded-full",
              index < filled ? "bg-info" : "bg-muted",
            )}
          />
        ))}
      </div>
      <p className="type-meta">
        {basis ?? "Numerical estimate only. Not a guarantee."}
      </p>
    </div>
  );
}
