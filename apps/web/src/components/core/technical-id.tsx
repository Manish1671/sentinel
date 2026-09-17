import { cn } from "@/lib/utils";

type TechnicalIdProps = {
  value: string;
  className?: string;
};

export function TechnicalId({ value, className }: TechnicalIdProps) {
  return (
    <code className={cn("type-code text-text-secondary", className)}>
      {value}
    </code>
  );
}
