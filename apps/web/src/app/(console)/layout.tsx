import { AppShell } from "@/components/shell/app-shell";
import { AuthProvider } from "@/components/auth/auth-provider";

export default function ConsoleLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      <AppShell>{children}</AppShell>
    </AuthProvider>
  );
}
