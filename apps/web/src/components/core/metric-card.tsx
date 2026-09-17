import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { cn } from "@/lib/utils";

type MetricCardProps = {
  label: string;
  value: string;
  hint?: string;
  className?: string;
};

export function MetricCard({ label, value, hint, className }: MetricCardProps) {
  return (
    <Card size="sm" className={cn("rounded-lg bg-surface", className)}>
      <CardHeader className="gap-2">
        <CardDescription className="type-label">{label}</CardDescription>
        <CardTitle className="type-metric">{value}</CardTitle>
      </CardHeader>
      {hint ? <CardContent className="type-meta pt-0">{hint}</CardContent> : null}
    </Card>
  );
}
