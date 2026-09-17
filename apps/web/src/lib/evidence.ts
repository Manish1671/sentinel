import type { EvidenceItem } from "@/lib/api/types";

export const EVIDENCE_CATEGORIES = [
  "logs",
  "metrics",
  "traces",
  "deployment",
  "runbooks",
  "historical_incidents",
] as const;

export type EvidenceCategory = (typeof EVIDENCE_CATEGORIES)[number] | "other";

const CATEGORY_LABEL: Record<EvidenceCategory, string> = {
  logs: "Logs",
  metrics: "Metrics",
  traces: "Traces",
  deployment: "Deployment",
  runbooks: "Runbooks",
  historical_incidents: "Historical incidents",
  other: "Other",
};

const ALIASES: Record<string, EvidenceCategory> = {
  log: "logs",
  logs: "logs",
  metric: "metrics",
  metrics: "metrics",
  trace: "traces",
  traces: "traces",
  span: "traces",
  deployment: "deployment",
  deploy: "deployment",
  catalog: "deployment",
  runbook: "runbooks",
  runbooks: "runbooks",
  historical: "historical_incidents",
  historical_incidents: "historical_incidents",
  incident: "historical_incidents",
};

export function evidenceCategory(item: EvidenceItem): EvidenceCategory {
  const raw = (item.source_type || item.tool_name || "").toLowerCase().replaceAll(" ", "_");
  return ALIASES[raw] ?? ALIASES[raw.split("_")[0] ?? ""] ?? "other";
}

export function evidenceCategoryLabel(category: EvidenceCategory): string {
  return CATEGORY_LABEL[category];
}

export function groupEvidence(items: EvidenceItem[]): Array<{ category: EvidenceCategory; items: EvidenceItem[] }> {
  const buckets = new Map<EvidenceCategory, EvidenceItem[]>();
  for (const item of items) {
    const category = evidenceCategory(item);
    const list = buckets.get(category) ?? [];
    list.push(item);
    buckets.set(category, list);
  }
  const ordered: Array<{ category: EvidenceCategory; items: EvidenceItem[] }> = [];
  for (const category of [...EVIDENCE_CATEGORIES, "other"] as EvidenceCategory[]) {
    const grouped = buckets.get(category);
    if (grouped?.length) ordered.push({ category, items: grouped });
  }
  return ordered;
}
