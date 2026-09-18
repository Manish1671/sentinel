import type { ReactNode } from "react";

type PageHeaderProps = {
  title: string;
  description?: string;
  actions?: ReactNode;
};

export function PageHeader({ title, description, actions }: PageHeaderProps) {
  return (
    <div className="mb-6 flex flex-col gap-2 sm:flex-row sm:items-end sm:justify-between">
      <div className="space-y-1">
        <h1 className="type-page-title">{title}</h1>
        {description ? <p className="type-meta max-w-2xl">{description}</p> : null}
      </div>
      {actions}
    </div>
  );
}
