const timeFormatter = new Intl.DateTimeFormat("en-GB", {
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
  timeZone: "UTC",
});

const dateTimeFormatter = new Intl.DateTimeFormat("en-GB", {
  day: "2-digit",
  month: "short",
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
  timeZone: "UTC",
});

export function formatClock(iso: string): string {
  return timeFormatter.format(new Date(iso));
}

export function formatDateTime(iso: string): string {
  return `${dateTimeFormatter.format(new Date(iso))} UTC`;
}

export function formatPercent(value: number): string {
  return `${Math.round(value * 100)}%`;
}

export function confidenceBand(value: number): "Low" | "Medium" | "High" {
  if (value >= 0.8) return "High";
  if (value >= 0.55) return "Medium";
  return "Low";
}

export function formatConfidence(value: number): string {
  return `${Math.round(value * 100)}% model estimate`;
}
