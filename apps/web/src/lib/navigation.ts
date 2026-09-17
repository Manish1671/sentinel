import type { LucideIcon } from "lucide-react";
import {
  Activity,
  Boxes,
  LayoutDashboard,
  Rocket,
  Search,
  Settings,
  TriangleAlert,
  Wrench,
} from "lucide-react";

export type NavItem = {
  href: string;
  label: string;
  icon: LucideIcon;
  match?: "exact" | "prefix";
};

export const primaryNav: NavItem[] = [
  { href: "/overview", label: "Overview", icon: LayoutDashboard, match: "prefix" },
  { href: "/services", label: "Services", icon: Boxes, match: "prefix" },
  { href: "/incidents", label: "Incidents", icon: TriangleAlert, match: "prefix" },
  { href: "/investigations", label: "Investigations", icon: Search, match: "prefix" },
  { href: "/remediations", label: "Remediations", icon: Wrench, match: "prefix" },
  { href: "/deployments", label: "Deployments", icon: Rocket, match: "prefix" },
];

export const secondaryNav: NavItem[] = [
  { href: "/settings", label: "Settings", icon: Settings, match: "exact" },
  {
    href: "/settings#system-status",
    label: "System status",
    icon: Activity,
    match: "exact",
  },
];

export const pageTitles: Record<string, string> = {
  "/overview": "Overview",
  "/services": "Services",
  "/incidents": "Incidents",
  "/investigations": "Investigations",
  "/remediations": "Remediations",
  "/deployments": "Deployments",
  "/settings": "Settings",
};

export function isNavActive(pathname: string, item: NavItem): boolean {
  const href = item.href.split("#")[0];
  if (item.match === "exact") {
    return pathname === href;
  }
  if (href === "/incidents") {
    return pathname === "/incidents" || pathname.startsWith("/incidents/");
  }
  return pathname === href || pathname.startsWith(`${href}/`);
}

export function titleForPath(pathname: string): string {
  if (pathname.startsWith("/incidents/") && pathname !== "/incidents") {
    return "Incident";
  }
  return pageTitles[pathname] ?? "Sentinel";
}
