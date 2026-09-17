import type { OperationalStatus, Severity } from "@/types/sentinel";

export const severityLabel: Record<Severity, string> = {
  critical: "Critical",
  high: "High",
  medium: "Medium",
  low: "Low",
};

export const severityClass: Record<Severity, string> = {
  critical: "text-severity-critical",
  high: "text-severity-high",
  medium: "text-severity-medium",
  low: "text-severity-low",
};

export const severityDotClass: Record<Severity, string> = {
  critical: "bg-severity-critical",
  high: "bg-severity-high",
  medium: "bg-severity-medium",
  low: "bg-severity-low",
};

export const statusLabel: Record<OperationalStatus, string> = {
  healthy: "Healthy",
  degraded: "Degraded",
  failed: "Failed",
  running: "Running",
  pending: "Pending",
  resolved: "Resolved",
};

export const statusClass: Record<OperationalStatus, string> = {
  healthy: "text-status-healthy",
  degraded: "text-status-degraded",
  failed: "text-status-failed",
  running: "text-status-running",
  pending: "text-status-pending",
  resolved: "text-status-resolved",
};

export const statusDotClass: Record<OperationalStatus, string> = {
  healthy: "bg-status-healthy",
  degraded: "bg-status-degraded",
  failed: "bg-status-failed",
  running: "bg-status-running",
  pending: "bg-status-pending",
  resolved: "bg-status-resolved",
};
