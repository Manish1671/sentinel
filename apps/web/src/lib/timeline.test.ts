import { describe, expect, it } from "vitest";
import { timelineFromEvents } from "@/lib/timeline";

describe("timelineFromEvents", () => {
  it("keeps operator fields and drops internal UUIDs", () => {
    const items = timelineFromEvents([
      {
        id: "evt-1",
        kind: "alert_attached",
        actor_user_id: null,
        occurred_at: "2026-09-14T03:11:00Z",
        summary: "Alert attached",
        source: "services.incident",
        payload: {
          source: "services.incident",
          audit_event: "Alert attached",
          alert_id: "12619da2-93ac-4b56-a3d2-6a04dc856024",
          service_id: "8f220c09-28e9-4d13-a649-09c2e270ac9b",
          version: "1.18.0",
          detector_id: "high_latency",
        },
      },
    ]);

    expect(items[0]?.title).toBe("Alert attached");
    expect(items[0]?.source).toBe("system");
    expect(items[0]?.description).toBeUndefined();
    expect(items[0]?.metadata).toEqual({ version: "1.18.0", detector_id: "high_latency" });
    expect(items[0]?.emphasis).toBe("default");
  });

  it("emphasizes lifecycle and remediation transitions", () => {
    const items = timelineFromEvents([
      { id: "1", kind: "created", actor_user_id: null, occurred_at: "2026-09-14T04:19:00Z", payload: {} },
      { id: "2", kind: "remediation_succeeded", actor_user_id: "u1", occurred_at: "2026-09-16T02:17:00Z", payload: {} },
    ]);
    expect(items[0]?.emphasis).toBe("lifecycle");
    expect(items[1]?.emphasis).toBe("remediation");
    expect(items[1]?.actor).toBe("operator");
  });
});
