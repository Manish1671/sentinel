"use client";

import { Progress } from "@/components/ui/progress";
import { cn } from "@/lib/utils";

type ProgressIndicatorProps = {
  value: number;
  label?: string;
  className?: string;
};

export function ProgressIndicator({ value, label, className }: ProgressIndicatorProps) {
  const clamped = Math.max(0, Math.min(100, value));
  return (
    <div className={cn("space-y-1.5", className)}>
      {label ? <p className="type-label">{label}</p> : null}
      <Progress value={clamped} aria-label={label ?? "Progress"} />
    </div>
  );
}
