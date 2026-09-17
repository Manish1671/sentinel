import type { Metadata } from "next";
import { RemediationsView } from "@/components/incident/remediations-view";
import { PageFade } from "@/components/motion/page-fade";
import { PageHeader } from "@/components/shell/page-header";

export const metadata: Metadata = { title: "Remediations" };

export default function RemediationsPage() {
  return (
    <PageFade>
      <PageHeader
        title="Remediations"
        description="Allowlisted actions require explicit human approval before execution."
      />
      <RemediationsView />
    </PageFade>
  );
}
