"use client";

import { useState } from "react";
import { SidebarNav } from "@/components/shell/sidebar-nav";
import { TopBar } from "@/components/shell/top-bar";
import { CommandPalette } from "@/components/shell/command-palette";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";

export function AppShell({ children }: { children: React.ReactNode }) {
  const [commandOpen, setCommandOpen] = useState(false);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);

  return (
    <div className="flex min-h-screen bg-background">
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:m-3 focus:rounded-md focus:bg-elevated focus:px-3 focus:py-2"
      >
        Skip to content
      </a>
      <aside className="hidden w-56 shrink-0 border-r border-sidebar-border bg-sidebar md:flex md:flex-col">
        <SidebarNav />
      </aside>
      <Sheet open={mobileNavOpen} onOpenChange={setMobileNavOpen}>
        <SheetContent side="left" className="w-64 p-0" showCloseButton>
          <SheetHeader className="sr-only">
            <SheetTitle>Navigation</SheetTitle>
            <SheetDescription>Sentinel operator console</SheetDescription>
          </SheetHeader>
          <SidebarNav onNavigate={() => setMobileNavOpen(false)} />
        </SheetContent>
      </Sheet>
      <div className="flex min-w-0 flex-1 flex-col">
        <TopBar onOpenCommand={() => setCommandOpen(true)} onOpenMobileNav={() => setMobileNavOpen(true)} />
        <main id="main-content" className="flex-1 overflow-x-hidden px-3 py-6 md:px-8 md:py-7">
          <div className="mx-auto w-full max-w-[1120px]">{children}</div>
        </main>
      </div>
      <CommandPalette open={commandOpen} onOpenChange={setCommandOpen} />
    </div>
  );
}
