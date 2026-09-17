export type UserRole = "viewer" | "responder" | "approver" | "admin";

export function canApproveRemediation(role: UserRole | string | undefined): boolean {
  return role === "approver" || role === "admin";
}

export function canWriteIncidents(role: UserRole | string | undefined): boolean {
  return role === "responder" || role === "approver" || role === "admin";
}

export function meetsApprovalRole(role: UserRole | string | undefined, required?: string | null): boolean {
  if (!canApproveRemediation(role)) return false;
  if (required === "admin") return role === "admin";
  return true;
}
