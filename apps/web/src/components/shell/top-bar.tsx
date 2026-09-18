"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { IconButton } from "@/components/core/icon-button";
import { useAuth } from "@/components/auth/auth-provider";
import { SystemStatusIndicator } from "@/components/shell/system-status";
import { Menu, Search } from "lucide-react";
import { titleForPath } from "@/lib/navigation";
import { usePathname } from "next/navigation";

type TopBarProps = {
  onOpenCommand: () => void;
  onOpenMobileNav: () => void;
};

function initials(name: string) {
  return name
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
}

export function TopBar({ onOpenCommand, onOpenMobileNav }: TopBarProps) {
  const pathname = usePathname();
  const title = titleForPath(pathname);
  const { user, signOut } = useAuth();
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <header className="flex h-11 items-center gap-3 border-b border-border bg-background/80 px-3 backdrop-blur-sm md:px-6">
      <IconButton label="Open navigation" variant="ghost" className="md:hidden" onClick={onOpenMobileNav}>
        <Menu className="size-4" />
      </IconButton>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium tracking-tight">{title}</p>
      </div>
      <div className="hidden items-center gap-3 sm:flex">
        <span className="type-meta font-medium tracking-wide text-text-secondary">Production</span>
        <span className="text-border" aria-hidden>
          /
        </span>
        <SystemStatusIndicator compact />
        <span className="text-border" aria-hidden>
          /
        </span>
        <span className="type-meta text-text-muted capitalize">{user.role}</span>
      </div>
      <Button
        variant="outline"
        size="sm"
        className="hidden h-7 gap-2 border-border/80 bg-transparent text-text-muted md:inline-flex"
        onClick={onOpenCommand}
      >
        <Search className="size-3.5" aria-hidden />
        Search
        <kbd className="type-meta rounded border border-border px-1">⌘K</kbd>
      </Button>
      <IconButton label="Search" variant="ghost" className="md:hidden" onClick={onOpenCommand}>
        <Search className="size-4" />
      </IconButton>
      <div className="relative">
        <button
          type="button"
          className="inline-flex h-7 min-w-7 items-center justify-center rounded-md px-2 text-[12px] font-medium tracking-tight text-text-primary hover:bg-muted"
          aria-haspopup="menu"
          aria-expanded={menuOpen}
          onClick={() => setMenuOpen((open) => !open)}
        >
          {initials(user.display_name) || "OP"}
        </button>
        {menuOpen ? (
          <div
            role="menu"
            className="absolute right-0 z-50 mt-1 min-w-44 rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-md"
          >
            <p className="px-1.5 py-1 text-xs font-medium text-muted-foreground">{user.display_name}</p>
            <p className="type-meta px-1.5 py-1">{user.email}</p>
            <p className="type-meta px-1.5 py-1 capitalize">Signed in · {user.role}</p>
            <div className="my-1 h-px bg-border" />
            <button
              type="button"
              role="menuitem"
              className="flex w-full rounded-md px-1.5 py-1 text-left text-sm hover:bg-accent"
              onClick={() => void signOut()}
            >
              Sign out
            </button>
          </div>
        ) : null}
      </div>
    </header>
  );
}
