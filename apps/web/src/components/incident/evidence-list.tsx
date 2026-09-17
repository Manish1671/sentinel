import { EvidenceBlock } from "@/components/ai/evidence-block";
import { EmptyState } from "@/components/core/empty-state";
import { TechnicalId } from "@/components/core/technical-id";
import { formatDateTime } from "@/lib/format";
import { evidenceCategoryLabel, groupEvidence } from "@/lib/evidence";
import type { EvidenceItem } from "@/lib/api/types";
import { FileSearch } from "lucide-react";

type EvidenceListProps = {
  evidence: EvidenceItem[];
};

function metaEntries(metadata: Record<string, unknown> | null): Array<[string, string]> {
  if (!metadata) return [];
  const out: Array<[string, string]> = [];
  for (const [key, value] of Object.entries(metadata)) {
    if (typeof value === "string" && value.length > 0 && value.length < 80) {
      out.push([key, value]);
    } else if (typeof value === "number" && Number.isFinite(value)) {
      out.push([key, String(value)]);
    }
  }
  return out.slice(0, 6);
}

export function EvidenceList({ evidence }: EvidenceListProps) {
  if (evidence.length === 0) {
    return (
      <EmptyState
        icon={FileSearch}
        title="No observed evidence collected"
        description="Telemetry and catalog facts appear here after investigation tools run. Hypotheses are shown separately."
      />
    );
  }

  const groups = groupEvidence(evidence);

  return (
    <EvidenceBlock title="Observed evidence">
      <div className="space-y-5">
        {groups.map((group) => (
          <section key={group.category} className="space-y-2">
            <h3 className="type-label">{evidenceCategoryLabel(group.category)}</h3>
            <ul className="space-y-3">
              {group.items.map((item) => {
                const reference = item.artifact_uri || (item.source_ref ? String(item.source_ref) : null);
                return (
                  <li key={item.id} className="space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <p className="type-card text-text-primary">{item.tool_name}</p>
                      <TechnicalId value={item.source_type} />
                      <time className="type-meta" dateTime={item.captured_at}>
                        {formatDateTime(item.captured_at)}
                      </time>
                    </div>
                    <p className="type-body">{item.summary}</p>
                    {reference ? (
                      <p className="type-meta">
                        Ref <TechnicalId value={reference} />
                      </p>
                    ) : null}
                    {metaEntries(item.metadata).length ? (
                      <div className="flex flex-wrap gap-x-3 gap-y-1">
                        {metaEntries(item.metadata).map(([key, value]) => (
                          <span key={key} className="inline-flex items-center gap-1">
                            <span className="type-label">{key.replaceAll("_", " ")}</span>
                            <TechnicalId value={value} />
                          </span>
                        ))}
                      </div>
                    ) : null}
                  </li>
                );
              })}
            </ul>
          </section>
        ))}
      </div>
    </EvidenceBlock>
  );
}
