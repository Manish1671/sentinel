import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

type LoadingStateProps = {
  label?: string;
  className?: string;
};

export function LoadingState({ label = "Loading", className }: LoadingStateProps) {
  return (
    <div role="status" aria-live="polite" className={cn("space-y-2", className)}>
      <span className="sr-only">{label}</span>
      <Skeleton className="h-4 w-40" />
      <Skeleton className="h-16 w-full" />
      <Skeleton className="h-16 w-full" />
    </div>
  );
}
