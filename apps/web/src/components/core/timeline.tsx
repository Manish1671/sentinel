"use client";

import { motion } from "framer-motion";
import { TechnicalId } from "@/components/core/technical-id";
import { EmptyState } from "@/components/core/empty-state";
import { formatClock } from "@/lib/format";
import { cn } from "@/lib/utils";
import { severityDotClass, statusDotClass } from "@/lib/tokens";
import { groupTimeline } from "@/lib/timeline";
import type { TimelineEmphasis, TimelineItem } from "@/types/sentinel";

type TimelineProps = {
  items: TimelineItem[];
  className?: string;
};

const emphasisDot: Record<TimelineEmphasis, string> = {
  default: "bg-text-muted",
  lifecycle: "bg-info",
  deployment: "bg-warning",
  remediation: "bg-brand",
  investigation: "bg-info",
};

const emphasisRing: Record<TimelineEmphasis, string> = {
  default: "",
  lifecycle: "ring-2 ring-info/30",
  deployment: "ring-2 ring-warning/30",
  remediation: "ring-2 ring-brand/30",
  investigation: "ring-2 ring-info/25",
};

export function Timeline({ items, className }: TimelineProps) {
  if (items.length === 0) {
    return (
      <EmptyState
        title="No timeline events yet"
        description="Detection, investigation, and remediation events will stream here as the incident progresses."
      />
    );
  }

  const groups = groupTimeline(items);
  let visualIndex = 0;

  return (
    <ol className={cn("relative", className)}>
      {groups.map((group) => {
        const related = group.length > 1;
        return group.map((item, groupIndex) => {
          const index = visualIndex++;
          const emphasis = item.emphasis ?? "default";
          const dotClass = item.severity
            ? severityDotClass[item.severity]
            : item.status
              ? statusDotClass[item.status]
              : emphasisDot[emphasis];
          const isLast = index === items.length - 1;
          return (
            <motion.li
              key={item.id}
              initial={{ y: 4 }}
              animate={{ y: 0 }}
              transition={{ duration: 0.14, delay: Math.min(index * 0.02, 0.2), ease: "easeOut" }}
              className="relative flex gap-3 pb-4 last:pb-0"
            >
              {!isLast ? (
                <span className="absolute top-3 left-[5px] h-[calc(100%-4px)] w-px bg-border" aria-hidden />
              ) : null}
              <span
                className={cn(
                  "relative z-10 mt-1.5 size-2.5 shrink-0 rounded-full",
                  dotClass,
                  emphasisRing[emphasis],
                )}
                aria-hidden
              />
              <div className={cn("min-w-0 flex-1", related && groupIndex > 0 && "opacity-90")}>
                <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
                  <time dateTime={item.timestamp} className="type-code text-text-muted">
                    {formatClock(item.timestamp)}
                  </time>
                  {item.kind ? (
                    <span className="type-label uppercase tracking-[0.06em]">{item.kind.replaceAll("_", " ")}</span>
                  ) : null}
                  <p
                    className={cn(
                      "text-[13px] font-medium tracking-tight text-text-primary",
                      (emphasis === "lifecycle" || emphasis === "remediation") && "text-[13.5px]",
                    )}
                  >
                    {item.title}
                  </p>
                </div>
                {item.description ? <p className="type-meta mt-1">{item.description}</p> : null}
                <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1">
                  {item.source ? <span className="type-meta">{item.source}</span> : null}
                  {item.actor ? <span className="type-meta">{item.actor}</span> : null}
                  {item.metadata
                    ? Object.entries(item.metadata).map(([key, value]) => (
                        <span key={key} className="inline-flex items-center gap-1">
                          <span className="type-label">{key.replaceAll("_", " ")}</span>
                          <TechnicalId value={value} />
                        </span>
                      ))
                    : null}
                </div>
              </div>
            </motion.li>
          );
        });
      })}
    </ol>
  );
}
