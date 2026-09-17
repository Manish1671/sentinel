import type { HealthStatus, OperationalStatus, RemediationStatus } from "@/types/sentinel";

export function healthToOperational(status: HealthStatus | string): OperationalStatus {
  switch (status) {
    case "healthy":
      return "healthy";
    case "degraded":
      return "degraded";
    case "unhealthy":
      return "failed";
    default:
      return "pending";
  }
}

export function deploymentToOperational(status: string): OperationalStatus {
  switch (status) {
    case "succeeded":
      return "healthy";
    case "failed":
      return "failed";
    case "rolled_back":
      return "resolved";
    default:
      return "running";
  }
}

export function overallHealth(statuses: Array<HealthStatus | string>): {
  label: string;
  tone: "healthy" | "degraded" | "failed";
} {
  if (statuses.some((status) => status === "unhealthy")) {
    return { label: "Unhealthy", tone: "failed" };
  }
  if (statuses.some((status) => status === "degraded" || status === "unknown")) {
    return { label: "Degraded", tone: "degraded" };
  }
  if (statuses.length === 0) {
    return { label: "Unknown", tone: "degraded" };
  }
  return { label: "Healthy", tone: "healthy" };
}

export function parameterString(params: Record<string, unknown> | null | undefined, key: string): string | null {
  const value = params?.[key];
  return typeof value === "string" && value.length > 0 ? value : null;
}

const UUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

export function operatorParameters(
  params: Record<string, unknown> | null | undefined,
): Array<{ key: string; value: string }> {
  if (!params) return [];
  const out: Array<{ key: string; value: string }> = [];
  for (const [key, value] of Object.entries(params)) {
    if (typeof value === "number" && Number.isFinite(value)) {
      out.push({ key, value: String(value) });
      continue;
    }
    if (typeof value !== "string" || value.length === 0 || UUID.test(value)) continue;
    out.push({ key, value });
  }
  return out;
}

export function remediationLabel(status: RemediationStatus | string): string {
  return status.replaceAll("_", " ");
}

export function newIdempotencyKey(prefix: string): string {
  const id = typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : `${Date.now()}`;
  return `${prefix}-${id}`;
}

export function errorMessage(error: unknown, fallback: string): string {
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}

export function isActiveIncident(status: string): boolean {
  return status !== "resolved" && status !== "closed";
}
