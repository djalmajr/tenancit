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

  test("groups daily management, operations, and system destinations like the appliance", () => {
    const groups = visibleNavGroups(new Set([
      "admin.read", "integration.manage", "audit.read", "session.manage",
    ]));
    expect(groups.map((group) => ({
      label: group.labelKey,
      routes: group.items.map((item) => item.to),
    }))).toEqual([
      {
        label: "nav.management",
        routes: ["/", "/tenants", "/resource-definitions", "/api-clients", "/usage"],
      },
      {
        label: "nav.operations",
        routes: ["/operations/health", "/integrations/webhooks", "/audit-events"],
      },
      {
        label: "nav.system",
        routes: ["/security/sessions", "/operations/settings"],
      },
    ]);
  });
});
