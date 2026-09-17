import { cn } from "@/lib/utils";

type SparklineProps = {
  values: number[];
  className?: string;
  color?: string;
};

export function Sparkline({ values, className, color = "var(--brand)" }: SparklineProps) {
  if (values.length < 2) return null;
  const width = 88;
  const height = 28;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;
  const path = values
    .map((value, index) => {
      const x = (index / (values.length - 1)) * width;
      const y = height - ((value - min) / range) * (height - 4) - 2;
      return `${index === 0 ? "M" : "L"}${x.toFixed(1)} ${y.toFixed(1)}`;
    })
    .join(" ");

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      className={cn("h-7 w-[5.5rem]", className)}
      aria-hidden
    >
      <path d={path} fill="none" stroke={color} strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}
