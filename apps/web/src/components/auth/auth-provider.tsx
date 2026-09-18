"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { getCurrentUser, logout, type CurrentUser } from "@/lib/api/auth";
import { ApiError } from "@/lib/api/errors";
import { LoadingState } from "@/components/core/loading-state";

type AuthContextValue = {
  user: CurrentUser;
  reload: () => Promise<void>;
  signOut: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return value;
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const response = await getCurrentUser();
      setUser(response.data);
      setError(null);
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        const next = encodeURIComponent(pathname || "/overview");
        router.replace(`/login?next=${next}`);
        return;
      }
      if (err instanceof ApiError && err.status >= 500) {
        setError("Unable to reach the Sentinel control plane. Retry in a moment.");
        return;
      }
      setError(err instanceof Error ? err.message : "Unable to load the current user.");
    } finally {
      setLoading(false);
    }
  }, [pathname, router]);

  useEffect(() => {
    void load();
  }, [load]);

  const signOut = useCallback(async () => {
    try {
      await logout();
    } finally {
      router.replace("/login");
    }
  }, [router]);

  const value = useMemo(
    () => (user ? { user, reload: load, signOut } : null),
    [user, load, signOut],
  );

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <LoadingState label="Loading session" className="w-80" />
      </div>
    );
  }

  if (error || !value) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-6">
        <div className="max-w-md space-y-3">
          <p className="text-[15px] font-medium tracking-tight">Unable to load operator session</p>
          <p className="type-meta">{error ?? "Authentication required."}</p>
          <button
            type="button"
            className="rounded-md border border-border px-3 py-1.5 text-[13px] hover:bg-muted"
            onClick={() => {
              setLoading(true);
              void load();
            }}
          >
            Retry
          </button>
        </div>
      </div>
    );
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
