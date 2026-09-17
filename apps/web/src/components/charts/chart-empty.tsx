import { EmptyState } from "@/components/core/empty-state";
import { LineChart } from "lucide-react";

type ChartEmptyProps = {
  title?: string;
  description?: string;
};

export function ChartEmpty({
  title = "Telemetry history is not exposed by the control plane",
  description = "apps/api does not serve a coherent historical series for latency, error rate, or DB utilization. Sentinel does not invent chart values.",
}: ChartEmptyProps) {
  return <EmptyState icon={LineChart} title={title} description={description} />;
}
