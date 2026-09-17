export type AfterSnapshot = {
  latencyMs?: number;
  errorRate?: number;
  dbUtilization?: number;
  healthStatus?: string;
  version?: string;
};

function asNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function asString(value: unknown): string | undefined {
  return typeof value === "string" && value.length > 0 ? value : undefined;
}

function fromRecord(record: Record<string, unknown> | null | undefined): AfterSnapshot | null {
  if (!record) return null;
  const snapshot: AfterSnapshot = {
    latencyMs: asNumber(record.latency_ms),
    errorRate: asNumber(record.error_rate),
    dbUtilization: asNumber(record.db_utilization),
    healthStatus: asString(record.health_status),
    version: asString(record.current_version),
  };
  if (
    snapshot.latencyMs == null &&
    snapshot.errorRate == null &&
    snapshot.dbUtilization == null &&
    !snapshot.healthStatus &&
    !snapshot.version
  ) {
    return null;
  }
  return snapshot;
}

export function afterSnapshotFromVerification(
  details: Record<string, unknown> | null | undefined,
): AfterSnapshot | null {
  if (!details) return null;
  const verification = details.verification;
  if (verification && typeof verification === "object" && !Array.isArray(verification)) {
    const nested = verification as Record<string, unknown>;
    const fromChecks = fromRecord(
      nested.checks && typeof nested.checks === "object" && !Array.isArray(nested.checks)
        ? (nested.checks as Record<string, unknown>)
        : nested,
    );
    if (fromChecks) return fromChecks;
  }
  const execution = details.execution;
  if (execution && typeof execution === "object" && !Array.isArray(execution)) {
    const fromExec = fromRecord(execution as Record<string, unknown>);
    if (fromExec) return fromExec;
  }
  return fromRecord(details);
}

export function hasBeforeAfterPair(details: Record<string, unknown> | null | undefined): boolean {
  if (!details) return false;
  return Boolean(details.before) && Boolean(details.after);
}
