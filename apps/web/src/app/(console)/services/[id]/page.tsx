import type { Metadata } from "next";
import { ServiceWorkspace } from "@/components/catalog/service-workspace";
import { PageFade } from "@/components/motion/page-fade";

export const metadata: Metadata = { title: "Service" };

export default async function ServiceDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return (
    <PageFade>
      <ServiceWorkspace serviceId={id} />
    </PageFade>
  );
}
