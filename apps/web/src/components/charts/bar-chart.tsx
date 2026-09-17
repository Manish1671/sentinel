"use client";

import { ChartEmpty } from "@/components/charts/chart-empty";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

type BarChartCardProps<T extends Record<string, string | number>> = {
  title: string;
  description?: string;
  data: T[];
  xKey: string;
  yKey: string;
  unit?: string;
  color?: string;
};

export function BarChartCard<T extends Record<string, string | number>>({
  title,
  description,
  data,
  xKey,
  yKey,
  unit,
  color = "var(--warning)",
}: BarChartCardProps<T>) {
  return (
    <Card size="sm" className="rounded-lg">
      <CardHeader>
        <CardTitle className="type-card">{title}</CardTitle>
        {description ? <CardDescription>{description}</CardDescription> : null}
      </CardHeader>
      <CardContent>
        {data.length === 0 ? (
          <ChartEmpty />
        ) : (
          <div className="h-52 w-full">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
                <CartesianGrid stroke="var(--border)" strokeDasharray="3 3" />
                <XAxis dataKey={xKey} tick={{ fill: "var(--text-muted)", fontSize: 11 }} axisLine={{ stroke: "var(--border)" }} />
                <YAxis
                  tick={{ fill: "var(--text-muted)", fontSize: 11 }}
                  axisLine={{ stroke: "var(--border)" }}
                  unit={unit ? ` ${unit}` : undefined}
                  width={48}
                />
                <Tooltip
                  contentStyle={{
                    background: "var(--elevated)",
                    border: "1px solid var(--border)",
                    borderRadius: 8,
                    color: "var(--text-primary)",
                    fontSize: 12,
                  }}
                />
                <Bar dataKey={yKey} fill={color} radius={[3, 3, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
