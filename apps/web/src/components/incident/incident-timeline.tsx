import { Timeline } from "@/components/core/timeline";
import type { TimelineItem } from "@/types/sentinel";

type IncidentTimelineProps = {
  items: TimelineItem[];
};

export function IncidentTimeline({ items }: IncidentTimelineProps) {
  return <Timeline items={items} />;
}
