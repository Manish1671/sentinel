import { cn } from "@/lib/utils";
import { statusDotClass } from "@/lib/tokens";
import type { IncidentStatus as IncidentStatusValue, OperationalStatus } from "@/types/sentinel";

const incidentToOperational: Record<IncidentStatusValue, OperationalStatus> = {
  open: "pending",
  investigating: "running",
  remediating: "running",
  verifying: "running",
  resolved: "resolved",
  closed: "resolved",
};

const incidentLabel: Record<IncidentStatusValue, string> = {
  open: "Open",
  investigating: "Investigating",
  remediating: "Remediating",
  verifying: "Verifying",
  resolved: "Resolved",
  closed: "Closed",
};

type IncidentStatusProps = {
  status: IncidentStatusValue;
  className?: string;
};

export function IncidentStatus({ status, className }: IncidentStatusProps) {
  const visual = incidentToOperational[status];
  return (
    <span className={cn("inline-flex items-center gap-1.5 text-xs text-text-secondary", className)}>
      <span className={cn("size-1.5 rounded-full", statusDotClass[visual])} aria-hidden />
      <span className="sr-only">Incident status</span>
      {incidentLabel[status]}
    </span>
  );
}
