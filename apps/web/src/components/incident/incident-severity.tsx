import { SeverityBadge } from "@/components/core/severity-badge";
import type { Severity } from "@/types/sentinel";

type IncidentSeverityProps = {
  severity: Severity;
};

export function IncidentSeverity({ severity }: IncidentSeverityProps) {
  return <SeverityBadge severity={severity} />;
}
