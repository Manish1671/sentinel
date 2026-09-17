import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

type RecommendationBlockProps = {
  title?: string;
  children: ReactNode;
  actions?: ReactNode;
  className?: string;
};

export function RecommendationBlock({
  title = "Recommendation",
  children,
  actions,
  className,
}: RecommendationBlockProps) {
  return (
    <section className={cn("rounded-md border border-brand/25 bg-brand/[0.04] px-4 py-3", className)}>
      <p className="type-label mb-2 text-brand">{title}</p>
      <div className="space-y-3">{children}</div>
      {actions ? <div className="mt-3 flex flex-wrap gap-2">{actions}</div> : null}
    </section>
  );
}
