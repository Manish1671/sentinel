import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

type EvidenceBlockProps = {
  title?: string;
  children: ReactNode;
  className?: string;
};

export function EvidenceBlock({ title = "Observed evidence", children, className }: EvidenceBlockProps) {
  return (
    <section className={cn("border-l border-border pl-4", className)}>
      {title ? <p className="type-label mb-2">{title}</p> : null}
      <div className="space-y-2 text-text-secondary">{children}</div>
    </section>
  );
}
