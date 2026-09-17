"use client";

import { ChartEmpty } from "@/components/charts/chart-empty";
import {
  CartesianGrid,
  Line,
  LineChart,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { cn } from "@/lib/utils";

export type ChartMarker = {
  x: string;
  label: string;
  tone: "neutral" | "danger" | "warning" | "success" | "info";
};

const markerStroke: Record<ChartMarker["tone"], string> = {
  neutral: "var(--text-muted)",
  danger: "var(--danger)",
  warning: "var(--warning)",
  success: "var(--success)",
  info: "var(--info)",
};

type NarrativeChartProps = {
  title: string;
  description?: string;
  data: Record<string, string | number>[];
  xKey: string;
  yKey: string;
  unit?: string;
  color?: string;
  markers?: ChartMarker[];
  className?: string;
};

export function NarrativeChart({
  title,
  description,
  data,
  xKey,
  yKey,
  unit,
  color = "var(--brand)",
  markers = [],
  className,
}: NarrativeChartProps) {
  return (
    <section className={cn("space-y-2", className)}>
      <div className="flex items-baseline justify-between gap-3">
        <div>
          <h3 className="type-card">{title}</h3>
          {description ? <p className="type-meta">{description}</p> : null}
        </div>
      </div>
      {data.length === 0 ? (
        <ChartEmpty title="Telemetry data unavailable" />
      ) : (
        <>
          <div className="h-44 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={data} margin={{ top: 12, right: 8, left: 0, bottom: 0 }}>
                <CartesianGrid stroke="var(--border)" strokeDasharray="2 4" vertical={false} />
                <XAxis
                  dataKey={xKey}
                  tick={{ fill: "var(--text-muted)", fontSize: 10 }}
                  axisLine={{ stroke: "var(--border)" }}
                  tickLine={false}
                />
                <YAxis
                  tick={{ fill: "var(--text-muted)", fontSize: 10 }}
                  axisLine={false}
                  tickLine={false}
                  width={40}
                  unit={unit ? ` ${unit}` : undefined}
                />
                <Tooltip
                  contentStyle={{
                    background: "var(--elevated)",
                    border: "1px solid var(--border)",
                    borderRadius: 6,
                    color: "var(--text-primary)",
                    fontSize: 12,
                  }}
                />
                {markers.map((marker) => (
                  <ReferenceLine
                    key={`${marker.x}-${marker.label}`}
                    x={marker.x}
                    stroke={markerStroke[marker.tone]}
                    strokeDasharray="3 3"
                    strokeOpacity={0.7}
                  />
                ))}
                <Line type="monotone" dataKey={yKey} stroke={color} strokeWidth={1.6} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
          {markers.length > 0 ? (
            <ul className="flex flex-wrap gap-x-3 gap-y-1">
              {markers.map((marker) => (
                <li key={`${marker.x}-${marker.label}`} className="type-meta flex items-center gap-1.5">
                  <span className="size-1.5 rounded-full" style={{ background: markerStroke[marker.tone] }} aria-hidden />
                  {marker.x} {marker.label}
                </li>
              ))}
            </ul>
          ) : null}
        </>
      )}
    </section>
  );
}
