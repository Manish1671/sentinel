"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Shield } from "lucide-react";
import { isNavActive, primaryNav, secondaryNav, type NavItem } from "@/lib/navigation";
import { cn } from "@/lib/utils";

type SidebarNavProps = {
  onNavigate?: () => void;
};

function NavLink({ item, pathname, onNavigate }: { item: NavItem; pathname: string; onNavigate?: () => void }) {
  const active = isNavActive(pathname, item);
  const Icon = item.icon;
  return (
    <Link
      href={item.href}
      onClick={onNavigate}
      aria-current={active ? "page" : undefined}
      className={cn(
        "group relative flex items-center gap-2 rounded-md px-2 py-1.5 text-[13px] text-text-muted transition-colors duration-150 hover:bg-sidebar-accent hover:text-sidebar-foreground",
        active && "bg-sidebar-accent text-sidebar-foreground",
      )}
    >
      <span
        className={cn(
          "absolute inset-y-1 left-0 w-px rounded-full bg-brand opacity-0 transition-opacity duration-150",
          active && "opacity-100",
        )}
        aria-hidden
      />
      <Icon className="size-3.5 shrink-0" aria-hidden />
      {item.label}
    </Link>
  );
}

export function SidebarNav({ onNavigate }: SidebarNavProps) {
  const pathname = usePathname();

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center gap-2.5 px-3 py-4">
        <span className="flex size-6 items-center justify-center rounded-md bg-brand/12 text-brand">
          <Shield className="size-3.5" aria-hidden />
        </span>
        <div>
          <p className="text-[13px] font-medium tracking-tight">Sentinel</p>
          <p className="type-meta">Reliability</p>
        </div>
      </div>
      <nav aria-label="Primary" className="flex flex-1 flex-col gap-6 px-2">
        <div className="space-y-0.5">
          {primaryNav.map((item) => (
            <NavLink key={item.href} item={item} pathname={pathname} onNavigate={onNavigate} />
          ))}
        </div>
        <div className="mt-auto space-y-0.5 border-t border-sidebar-border pt-3 pb-8">
          {secondaryNav.map((item) => (
            <NavLink key={item.href} item={item} pathname={pathname} onNavigate={onNavigate} />
          ))}
        </div>
      </nav>
    </div>
  );
}
