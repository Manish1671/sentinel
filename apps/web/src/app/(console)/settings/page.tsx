import type { Metadata } from "next";
import { PageFade } from "@/components/motion/page-fade";
import { PageHeader } from "@/components/shell/page-header";
import { SettingsTabs } from "@/components/shell/settings-tabs";

export const metadata: Metadata = { title: "Settings" };

export default function SettingsPage() {
  return (
    <PageFade>
      <PageHeader
        title="Settings"
        description="Operator preferences and system status."
      />
      <SettingsTabs />
    </PageFade>
  );
}
