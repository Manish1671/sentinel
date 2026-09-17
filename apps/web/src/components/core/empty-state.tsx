import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";
import { Inbox } from "lucide-react";
import type { ReactNode } from "react";

type EmptyStateProps = {
  title: string;
  description?: string;
  icon?: LucideIcon;
  action?: ReactNode;
  className?: string;
};

export function EmptyState({
  title,
  description,
  icon: Icon = Inbox,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div className={cn("flex flex-col items-start gap-2 py-6", className)}>
      <div className="flex items-center gap-2 text-text-secondary">
        <Icon className="size-3.5" aria-hidden />
        <p className="text-[13px] font-medium tracking-tight">{title}</p>
      </div>
      {description ? <p className="type-meta max-w-prose">{description}</p> : null}
      {action}
    </div>
  );
}
