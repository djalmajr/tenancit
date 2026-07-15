import { describe, expect, it } from "vitest";
import type { APIClientUsageRecord, ApiClient } from "./api";
import { countExpiringAPIClients, currentUTCMonthRange, totalUsageRequests } from "./overview-pulse";

describe("overview operational pulse", () => {
  it("builds an inclusive UTC month range", () => {
    expect(currentUTCMonthRange(new Date("2026-02-14T23:00:00-03:00"))).toEqual({
      from: "2026-02-01",
      to: "2026-02-28",
    });
  });

  it("counts only active clients expiring inside the configured window", () => {
    const now = new Date("2026-07-14T12:00:00Z");
    const client = (status: string, expires_at: string): ApiClient => ({
      expires_at,
      id: crypto.randomUUID(),
      key_preview: "tnc_test",
      name: "test",
      rpm_limit: 60,
      scopes: ["tenant:identify"],
      status,
    });
    expect(countExpiringAPIClients([
      client("active", "2026-07-14T12:00:00Z"),
      client("active", "2026-08-13T12:00:00Z"),
      client("active", "2026-08-14T12:00:01Z"),
      client("revoked", "2026-07-20T12:00:00Z"),
    ], now)).toBe(2);
  });

  it("sums request volume across operations and status classes", () => {
    const record = (request_count: number): APIClientUsageRecord => ({
      api_client_id: crypto.randomUUID(),
      day: "2026-07-14",
      operation: "identify",
      rate_limited_count: 0,
      request_count,
      status_class: 2,
    });
    expect(totalUsageRequests([record(7), record(5)])).toBe(12);
  });
});
