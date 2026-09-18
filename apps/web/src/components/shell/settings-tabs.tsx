"use client";

import { useEffect, useState } from "react";
import { SystemStatusIndicator } from "@/components/shell/system-status";
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
            <CardDescription>Dark-first theme is the only supported appearance.</CardDescription>
          </CardHeader>
          <CardContent className="type-meta">
            Session is stored in the HttpOnly sentinel_session cookie. Role management stays in the control plane.
          </CardContent>
        </Card>
      </TabsContent>
      <TabsContent value="system" className="pt-4">
        <section id="system-status" className="space-y-4">
          <SystemStatusIndicator />
          <p className="type-meta">
            This probe is GET /ready on apps/api. Ingestion, detection, and Kafka are not included and are not
            reported as healthy from this screen.
          </p>
        </section>
      </TabsContent>
    </Tabs>
  );
}
