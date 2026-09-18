import { apiFetch } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";

export type ReadyStatus = {
  status: "ok" | "unavailable" | string;
  checks?: Record<string, string>;
};

export async function getControlPlaneReady(init: { signal?: AbortSignal } = {}): Promise<ReadyStatus> {
  try {
    return await apiFetch<ReadyStatus>("/api/ready", { signal: init.signal, timeoutMs: 4000 });
  } catch (error) {
    if (error instanceof ApiError && error.body && typeof error.body === "object" && error.body !== null && "status" in error.body) {
      return error.body as ReadyStatus;
    }
    throw error;
  }
}

export function controlPlaneLabel(ready: ReadyStatus | null, error?: Error | null): {
  label: string;
  tone: "healthy" | "degraded" | "failed";
} {
  if (error || !ready) {
    return { label: "Unavailable", tone: "failed" };
  }
  if (ready.status === "ok") {
    return { label: "Operational", tone: "healthy" };
  }
  return { label: "Degraded", tone: "degraded" };
}
