import type { Metadata } from "next";
import { ServiceList } from "@/components/catalog/service-list";
import { PageFade } from "@/components/motion/page-fade";
import { PageHeader } from "@/components/shell/page-header";

export const metadata: Metadata = { title: "Services" };

export default function ServicesPage() {
  return (
    <PageFade>
      <PageHeader title="Services" description="Watched production services and current health." />
      <ServiceList />
    </PageFade>
  );
}
