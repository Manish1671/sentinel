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
import { listDeployments } from "@/lib/api/deployments";
import { listIncidents } from "@/lib/api/incidents";
import { listInvestigations } from "@/lib/api/investigations";
import { listRemediations } from "@/lib/api/remediations";
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
  const investigations = useApiResource((signal) => listInvestigations({ limit: 6 }, { signal }), {
    enabled: open,
    deps: [open],
  });
  const remediations = useApiResource((signal) => listRemediations({ limit: 6 }, { signal }), {
    enabled: open,
    deps: [open],
  });
  const deployments = useApiResource((signal) => listDeployments({ limit: 6 }, { signal }), {
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
  const firstService = services.status === "success" ? services.data.data[0] : null;
  const firstInvestigation = investigations.status === "success" ? investigations.data.data[0] : null;
  const firstRemediation = remediations.status === "success" ? remediations.data.data[0] : null;
  const firstDeployment = deployments.status === "success" ? deployments.data.data[0] : null;

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
            <CommandItem
              value="Go to service"
              onSelect={() => go(firstService ? `/services/${firstService.id}` : "/services")}
            >
              <Boxes className="size-3.5" aria-hidden />
              Go to service
              {firstService ? <CommandShortcut>{firstService.slug}</CommandShortcut> : null}
            </CommandItem>
            <CommandItem
              value="Go to investigation"
              onSelect={() =>
                go(
                  firstInvestigation?.incident_reference
                    ? `/incidents/${firstInvestigation.incident_reference}`
                    : "/investigations",
                )
              }
            >
              <FileSearch className="size-3.5" aria-hidden />
              Go to investigation
            </CommandItem>
            <CommandItem value="Go to remediation" onSelect={() => go("/remediations")}>
              <Wrench className="size-3.5" aria-hidden />
              Go to remediation
              {firstRemediation ? <CommandShortcut>{firstRemediation.status.replaceAll("_", " ")}</CommandShortcut> : null}
            </CommandItem>
            <CommandItem
              value="Go to deployment"
              onSelect={() => go(firstDeployment?.service_id ? `/services/${firstDeployment.service_id}` : "/deployments")}
            >
              <Rocket className="size-3.5" aria-hidden />
              Go to deployment
            </CommandItem>
          </CommandGroup>
          {incidents.status === "success" ? (
            <CommandGroup heading="Incidents">
              {incidents.data.data.map((incident) => (
                <CommandItem
                  key={incident.id}
                  value={`incident ${incident.reference} ${incident.title} ${incident.service_slug}`}
                  onSelect={() => go(`/incidents/${incident.reference}`)}
                >
                  <Search className="size-3.5" aria-hidden />
                  {incident.reference} · {incident.service_slug}
                </CommandItem>
              ))}
            </CommandGroup>
          ) : null}
          {services.status === "success" ? (
            <CommandGroup heading="Services">
              {services.data.data.map((service) => (
                <CommandItem
                  key={service.id}
                  value={`service ${service.slug} ${service.name}`}
                  onSelect={() => go(`/services/${service.id}`)}
                >
                  <Search className="size-3.5" aria-hidden />
                  {service.slug}
                </CommandItem>
              ))}
            </CommandGroup>
          ) : null}
          {investigations.status === "success" ? (
            <CommandGroup heading="Investigations">
              {investigations.data.data.map((item) => (
                <CommandItem
                  key={item.id}
                  value={`investigation ${item.incident_reference ?? item.incident_id} ${item.status}`}
                  onSelect={() =>
                    go(item.incident_reference ? `/incidents/${item.incident_reference}` : "/investigations")
                  }
                >
                  <Search className="size-3.5" aria-hidden />
                  {item.incident_reference ?? item.id} · {item.status}
                </CommandItem>
              ))}
            </CommandGroup>
          ) : null}
        </CommandList>
      </Command>
    </CommandDialog>
  ) : null;
}
