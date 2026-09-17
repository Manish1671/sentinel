import type { Metadata } from "next";
import { DeploymentList } from "@/components/catalog/deployment-list";
import { PageFade } from "@/components/motion/page-fade";
import { PageHeader } from "@/components/shell/page-header";

export const metadata: Metadata = { title: "Deployments" };

export default function DeploymentsPage() {
  return (
    <PageFade>
      <PageHeader title="Deployments" description="Recent catalog deployments." />
      <DeploymentList />
    </PageFade>
  );
}
