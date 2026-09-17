import { Badge } from "@/components/ui/badge";
import { statusClass, statusDotClass, statusLabel } from "@/lib/tokens";
import { cn } from "@/lib/utils";
import type { OperationalStatus } from "@/types/sentinel";

type StatusBadgeProps = {
  status: OperationalStatus;
  className?: string;
};

export function StatusBadge({ status, className }: StatusBadgeProps) {
  return (
    <Badge
      variant="ghost"
      className={cn(
        "h-5 gap-1.5 rounded-sm border-0 bg-transparent px-1.5 font-medium capitalize",
        statusClass[status],
        className,
      )}
    >
      <span className={cn("size-1.5 rounded-full", statusDotClass[status])} aria-hidden />
      <span className="sr-only">Status</span>
      {statusLabel[status]}
    </Badge>
  );
}
