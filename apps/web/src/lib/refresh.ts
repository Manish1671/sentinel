export const refreshIntervals = {
  overviewMs: 20_000,
  incidentActiveMs: 8_000,
  incidentSettledMs: 20_000,
  remediationActiveMs: 3_000,
  catalogMs: 30_000,
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
  return refreshIntervals.incidentSettledMs;
}
