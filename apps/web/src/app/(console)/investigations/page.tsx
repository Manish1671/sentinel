import type { Metadata } from "next";
import { InvestigationsView } from "@/components/incident/investigations-view";
import { PageFade } from "@/components/motion/page-fade";
import { PageHeader } from "@/components/shell/page-header";

export const metadata: Metadata = { title: "Investigations" };

export default function InvestigationsPage() {
  return (
    <PageFade>
      <PageHeader
        title="Investigations"
        description="Bounded AI investigations attached to incidents."
      />
      <InvestigationsView />
    </PageFade>
  );
}
