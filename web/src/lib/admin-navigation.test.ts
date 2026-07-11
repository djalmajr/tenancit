import { describe, expect, test } from "vitest";
import { visibleNavGroups } from "./admin-navigation";

function routesFor(permissions: string[]) {
  return visibleNavGroups(new Set(permissions)).flatMap((group) => group.items.map((item) => item.to));
}

describe("admin navigation", () => {
  test("keeps read-only operators away from privileged security and integration routes", () => {
    expect(routesFor(["admin.read"])).toEqual([
      "/", "/tenants", "/resource-definitions", "/api-clients",
      "/usage", "/operations/health", "/operations/settings",
    ]);
  });

  test("shows only capability-backed operational and security destinations", () => {
    expect(routesFor(["admin.read", "integration.manage"])).toContain("/integrations/webhooks");
    expect(routesFor(["admin.read", "audit.read"])).toContain("/audit-events");
    expect(routesFor(["admin.read", "session.manage"])).toContain("/security/sessions");
    expect(routesFor(["admin.read"])).not.toContain("/audit-events");
  });
});
