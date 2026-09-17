import { Badge } from "@/components/ui/badge";
import { severityClass, severityDotClass, severityLabel } from "@/lib/tokens";
import { cn } from "@/lib/utils";
import type { Severity } from "@/types/sentinel";

type SeverityBadgeProps = {
  severity: Severity;
  className?: string;
};

export function SeverityBadge({ severity, className }: SeverityBadgeProps) {
  return (
    <Badge
      variant="ghost"
      className={cn(
        "h-5 gap-1.5 rounded-sm border-0 bg-transparent px-1.5 font-medium",
        severityClass[severity],
        className,
      )}
    >
      <span className={cn("size-1.5 rounded-full", severityDotClass[severity])} aria-hidden />
      <span className="sr-only">Severity</span>
      {severityLabel[severity]}
    </Badge>
  );
}
