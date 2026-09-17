"use client";

import { useEffect, useState } from "react";
import { StatusBadge } from "@/components/core/status-badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

export function SettingsTabs() {
  const [tab, setTab] = useState("preferences");

  useEffect(() => {
    if (window.location.hash === "#system-status") {
      setTab("system");
    }
  }, []);

  return (
    <Tabs value={tab} onValueChange={setTab}>
      <TabsList>
        <TabsTrigger value="preferences">Preferences</TabsTrigger>
        <TabsTrigger value="system">System status</TabsTrigger>
      </TabsList>
      <TabsContent value="preferences" className="pt-4">
        <Card className="rounded-lg">
          <CardHeader>
            <CardTitle className="type-card">Console</CardTitle>
            <CardDescription>Dark-first theme is the only supported appearance in Phase 8A.</CardDescription>
          </CardHeader>
          <CardContent className="type-meta">
            Authentication, notification routing, and role management are not implemented in this phase.
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="system" className="pt-4">
        <section id="system-status" className="grid gap-3 sm:grid-cols-2">
          {[
            { name: "API", status: "healthy" as const },
            { name: "Ingestion", status: "healthy" as const },
            { name: "Detection", status: "healthy" as const },
            { name: "Remediation", status: "healthy" as const },
          ].map((item) => (
            <Card key={item.name} size="sm" className="rounded-lg">
              <CardHeader>
                <CardTitle className="type-card">{item.name}</CardTitle>
                <StatusBadge status={item.status} />
              </CardHeader>
              <CardContent className="type-meta">Placeholder indicator. Not a live health probe.</CardContent>
            </Card>
          ))}
        </section>
      </TabsContent>
    </Tabs>
  );
}
