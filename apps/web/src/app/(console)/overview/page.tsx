import type { Metadata } from "next";
import { OverviewView } from "@/components/overview/overview-view";
import { PageFade } from "@/components/motion/page-fade";
import { PageHeader } from "@/components/shell/page-header";

export const metadata: Metadata = { title: "Overview" };

export default function OverviewPage() {
  return (
    <PageFade>
      <PageHeader title="Overview" description="Production reliability control center." />
      <OverviewView />
    </PageFade>
  );
}
