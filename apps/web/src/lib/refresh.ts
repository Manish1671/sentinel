export const refreshIntervals = {
  overviewMs: 20_000,
  incidentActiveMs: 8_000,
  incidentSettledMs: 20_000,
  remediationActiveMs: 3_000,
  catalogMs: 30_000,
  systemStatusMs: 20_000,
} as const;

export function incidentRefreshMs(status?: string | null): number {
  if (!status || status === "resolved" || status === "closed") {
    return refreshIntervals.incidentSettledMs;
  }
  return refreshIntervals.incidentActiveMs;
}

export function remediationRefreshMs(status?: string | null): number {
  if (status === "approved" || status === "running" || status === "verifying") {
    return refreshIntervals.remediationActiveMs;
  }
  if (status === "pending_approval") {
    return refreshIntervals.overviewMs;
  }
  return refreshIntervals.incidentSettledMs;
}

/**
 * Live updates stay on this polling abstraction so SSE can replace it later
 * without rewriting pages. The control plane does not currently expose
 * incident/investigation/remediation event streams.
 */
export const liveTransport = "polling" as const;
