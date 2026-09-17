import { describe, expect, it } from "vitest";
import { canApproveRemediation, canWriteIncidents, meetsApprovalRole } from "@/lib/auth/roles";

describe("role helpers", () => {
  it("hides approval for viewer and responder", () => {
    expect(canApproveRemediation("viewer")).toBe(false);
    expect(canApproveRemediation("responder")).toBe(false);
    expect(canApproveRemediation("approver")).toBe(true);
    expect(canApproveRemediation("admin")).toBe(true);
  });

  it("keeps write access off for viewers", () => {
    expect(canWriteIncidents("viewer")).toBe(false);
    expect(canWriteIncidents("responder")).toBe(true);
  });

  it("enforces admin-required approval", () => {
    expect(meetsApprovalRole("approver", "admin")).toBe(false);
    expect(meetsApprovalRole("admin", "admin")).toBe(true);
    expect(meetsApprovalRole("approver", "approver")).toBe(true);
  });
});
