"use client";

import { motion } from "framer-motion";
import { TechnicalId } from "@/components/core/technical-id";
import { EmptyState } from "@/components/core/empty-state";
import { formatClock } from "@/lib/format";
import { statusDotClass } from "@/lib/tokens";
import { cn } from "@/lib/utils";
import type { TimelineItem } from "@/types/sentinel";

export function ActivityStream({ items }: { items: TimelineItem[] }) {
  if (items.length === 0) {
    return (
      <EmptyState
        title="No recent activity"
        description="Timeline events appear here as detection, investigation, and remediation progress."
      />
    );
  }

  return (
    <ol className="relative">
      {items.map((item, index) => {
        const dotClass = item.status ? statusDotClass[item.status] : "bg-text-muted";
        return (
          <motion.li
            key={item.id}
            initial={{ y: 4 }}
            animate={{ y: 0 }}
            transition={{ duration: 0.16, delay: index * 0.03, ease: "easeOut" }}
            className="relative flex gap-4 py-3"
          >
            {index < items.length - 1 ? (
              <span className="absolute top-6 left-[5px] h-[calc(100%-8px)] w-px bg-border" aria-hidden />
            ) : null}
            <span className={cn("relative z-10 mt-1.5 size-2.5 shrink-0 rounded-full", dotClass)} aria-hidden />
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-baseline gap-x-3">
                <time dateTime={item.timestamp} className="type-code text-text-muted">
                  {formatClock(item.timestamp)}
                </time>
                <p className="text-[13px] font-medium tracking-tight text-text-primary">{item.title}</p>
              </div>
              {item.description ? <p className="type-meta mt-1">{item.description}</p> : null}
              <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1">
                {item.source ? <span className="type-meta">{item.source}</span> : null}
                {item.metadata
                  ? Object.entries(item.metadata).map(([key, value]) => <TechnicalId key={key} value={value} />)
                  : null}
              </div>
            </div>
          </motion.li>
        );
      })}
    </ol>
  );
}
