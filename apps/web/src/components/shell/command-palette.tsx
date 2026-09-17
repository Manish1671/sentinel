"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandShortcut,
} from "@/components/ui/command";
import { listIncidents } from "@/lib/api/incidents";
import { listServices } from "@/lib/api/services";
import { useApiResource } from "@/lib/hooks/use-api-resource";
import { Boxes, FileSearch, Rocket, Search, TriangleAlert, Wrench } from "lucide-react";

type CommandPaletteProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function CommandPalette({ open, onOpenChange }: CommandPaletteProps) {
  const router = useRouter();
  const incidents = useApiResource((signal) => listIncidents({ limit: 8 }, { signal }), {
    enabled: open,
    deps: [open],
  });
  const services = useApiResource((signal) => listServices({ limit: 8 }, { signal }), {
    enabled: open,
    deps: [open],
  });

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        onOpenChange(!open);
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [open, onOpenChange]);

  function go(href: string) {
    onOpenChange(false);
    router.push(href);
  }

  const firstIncident = incidents.status === "success" ? incidents.data.data[0] : null;

  return open ? (
    <CommandDialog open={open} onOpenChange={onOpenChange} title="Search Sentinel" description="Jump to a page or record">
      <Command>
        <CommandInput placeholder="Go to incident, service, investigation…" />
        <CommandList>
          <CommandEmpty>No matching destination.</CommandEmpty>
          <CommandGroup heading="Go to">
            <CommandItem
              value="Go to incident"
              onSelect={() => go(firstIncident ? `/incidents/${firstIncident.reference}` : "/incidents")}
            >
              <TriangleAlert className="size-3.5" aria-hidden />
              Go to incident
              {firstIncident ? <CommandShortcut>{firstIncident.reference}</CommandShortcut> : null}
            </CommandItem>
            <CommandItem value="Search service" onSelect={() => go("/services")}>
              <Boxes className="size-3.5" aria-hidden />
              Search service
            </CommandItem>
            <CommandItem value="Open investigation" onSelect={() => go("/investigations")}>
              <FileSearch className="size-3.5" aria-hidden />
              Open investigation
            </CommandItem>
            <CommandItem value="View deployment" onSelect={() => go("/deployments")}>
              <Rocket className="size-3.5" aria-hidden />
              View deployment
            </CommandItem>
            <CommandItem value="Review remediation" onSelect={() => go("/remediations")}>
              <Wrench className="size-3.5" aria-hidden />
              Review remediation
            </CommandItem>
          </CommandGroup>
          {incidents.status === "success" ? (
            <CommandGroup heading="Incidents">
              {incidents.data.data.map((incident) => (
                <CommandItem
                  key={incident.id}
                  value={`incident ${incident.reference} ${incident.title}`}
                  onSelect={() => go(`/incidents/${incident.reference}`)}
                >
                  <Search className="size-3.5" aria-hidden />
                  {incident.reference}
                </CommandItem>
              ))}
            </CommandGroup>
          ) : null}
          {services.status === "success" ? (
            <CommandGroup heading="Services">
              {services.data.data.map((service) => (
                <CommandItem key={service.id} value={`service ${service.slug}`} onSelect={() => go("/services")}>
                  <Search className="size-3.5" aria-hidden />
                  {service.slug}
                </CommandItem>
              ))}
            </CommandGroup>
          ) : null}
        </CommandList>
      </Command>
    </CommandDialog>
  ) : null;
}
