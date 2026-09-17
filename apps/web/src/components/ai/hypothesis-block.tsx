import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

type HypothesisBlockProps = {
  title?: string;
  children: ReactNode;
  className?: string;
};

export function HypothesisBlock({
  title = "Inference",
  children,
  className,
}: HypothesisBlockProps) {
  return (
    <section className={cn("rounded-md border border-info/20 bg-info/[0.04] px-4 py-3", className)}>
      <p className="type-label mb-2 text-info">{title}</p>
      <div className="space-y-3">{children}</div>
    </section>
  );
}
