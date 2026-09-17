import type { Metadata } from "next";
import { IncidentWorkspace } from "@/components/incident/incident-workspace";

export const metadata: Metadata = { title: "Incident" };

export default async function IncidentDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  return <IncidentWorkspace incidentId={id} />;
}
