import { afterEach, describe, expect, it, vi } from "vitest";
import { apiFetch, withQuery } from "@/lib/api/client";
import { ApiError } from "@/lib/api/errors";

describe("apiFetch", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("parses a successful JSON payload", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        headers: new Headers({ "X-Request-Id": "req-1" }),
        text: async () => JSON.stringify({ data: { id: "1" } }),
      }),
    );
    const result = await apiFetch<{ data: { id: string } }>("/api/v1/me");
    expect(result.data.id).toBe("1");
    expect(fetch).toHaveBeenCalledWith(
      "/api/v1/me",
      expect.objectContaining({ credentials: "include" }),
    );
  });

  it("throws ApiError on unauthorized responses", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: false,
        status: 401,
        headers: new Headers({ "X-Request-Id": "req-2" }),
        text: async () =>
          JSON.stringify({
            error: { code: "unauthenticated", message: "Authentication required.", request_id: "req-2" },
          }),
      }),
    );
    await expect(apiFetch("/api/v1/me")).rejects.toBeInstanceOf(ApiError);
    await expect(apiFetch("/api/v1/me")).rejects.toMatchObject({
      status: 401,
      code: "unauthenticated",
    });
  });
});

describe("withQuery", () => {
  it("omits empty values", () => {
    expect(withQuery("/api/v1/incidents", { status: "open", cursor: undefined })).toBe(
      "/api/v1/incidents?status=open",
    );
  });
});
