import { apiFetch } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";
import type { ItemResponse } from "@/lib/api/types";

export type CurrentUser = {
  id: string;
  email: string;
  display_name: string;
  role: "viewer" | "responder" | "approver" | "admin";
  status: "active" | "disabled";
};

export type LoginResponse = {
  user: Omit<CurrentUser, "status">;
};

export function login(email: string, password: string) {
  return apiFetch<ItemResponse<LoginResponse>>("/api/v1/auth/login", {
    method: "POST",
    body: { email, password },
  });
}

export async function logout() {
  try {
    await apiFetch<null>("/api/v1/auth/logout", { method: "POST" });
  } catch (error) {
    if (error instanceof ApiError && (error.status === 401 || error.status === 204)) {
      return;
    }
    throw error;
  }
}

export function getCurrentUser() {
  return apiFetch<ItemResponse<CurrentUser>>("/api/v1/me");
}
