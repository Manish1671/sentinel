import type { Metadata } from "next";
import { IncidentList } from "@/components/incident/incident-list";
import { PageFade } from "@/components/motion/page-fade";
import { PageHeader } from "@/components/shell/page-header";

export const metadata: Metadata = { title: "Incidents" };

export default function IncidentsPage() {
  return (
    <PageFade>
      <PageHeader title="Incidents" description="Correlated production incidents." />
      <IncidentList />
    </PageFade>
  );
}
