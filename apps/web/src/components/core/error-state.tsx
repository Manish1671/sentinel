"use client";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { CircleAlert } from "lucide-react";

type ErrorStateProps = {
  title: string;
  description?: string;
  onRetry?: () => void;
  className?: string;
};

export function ErrorState({ title, description, onRetry, className }: ErrorStateProps) {
  return (
    <div role="alert" className={cn("flex flex-col items-start gap-2 py-6", className)}>
      <div className="flex items-center gap-2 text-danger">
        <CircleAlert className="size-3.5" aria-hidden />
        <p className="text-[13px] font-medium tracking-tight">{title}</p>
      </div>
      {description ? <p className="type-meta">{description}</p> : null}
      {onRetry ? (
        <Button variant="outline" size="sm" onClick={onRetry}>
          Retry
        </Button>
      ) : null}
    </div>
  );
}
