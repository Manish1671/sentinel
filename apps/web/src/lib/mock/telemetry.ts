/**
 * MOCK / DEMO DATA — UI DEVELOPMENT ONLY
 * Deterministic series for chart primitives. Not live telemetry.
 */
import { MOCK_DATA_NOTICE } from "@/lib/mock/notice";

export { MOCK_DATA_NOTICE };

export type TelemetryPoint = {
  time: string;
  latencyMs: number;
  errorRate: number;
  saturation: number;
};

export const emptyTelemetry: TelemetryPoint[] = [];

export const demoTelemetry: TelemetryPoint[] = [
  { time: "10:35", latencyMs: 160, errorRate: 0.3, saturation: 42 },
  { time: "10:37", latencyMs: 170, errorRate: 0.4, saturation: 44 },
  { time: "10:39", latencyMs: 175, errorRate: 0.4, saturation: 46 },
  { time: "10:41", latencyMs: 210, errorRate: 0.9, saturation: 61 },
  { time: "10:43", latencyMs: 1640, errorRate: 8.3, saturation: 93 },
  { time: "10:45", latencyMs: 1510, errorRate: 7.8, saturation: 91 },
  { time: "10:47", latencyMs: 420, errorRate: 2.1, saturation: 58 },
  { time: "10:49", latencyMs: 190, errorRate: 0.5, saturation: 47 },
];

export const demoHealthComparison = {
  before: { latencyMs: 1640, errorRate: 8.3, saturation: 93 },
  after: { latencyMs: 190, errorRate: 0.5, saturation: 47 },
};

export type ChartMarker = {
  x: string;
  label: string;
  tone: "neutral" | "danger" | "warning" | "success" | "info";
};

export const incidentChartMarkers: ChartMarker[] = [
  { x: "10:41", label: "Deploy", tone: "neutral" },
  { x: "10:42", label: "Alert", tone: "warning" },
  { x: "10:44", label: "Incident", tone: "danger" },
  { x: "10:47", label: "Rollback", tone: "info" },
  { x: "10:49", label: "Recovery", tone: "success" },
];
