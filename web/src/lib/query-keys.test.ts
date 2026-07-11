import { describe, expect, it } from "vitest";
import { adminQueryKeys } from "./query-keys";

describe("admin query keys", () => {
  it("namespaces every protected query below the admin root", () => {
    const keys = [
      adminQueryKeys.overview(),
      adminQueryKeys.tenants(),
      adminQueryKeys.tenant("tenant-1"),
      adminQueryKeys.tenantDomains("tenant-1"),
      adminQueryKeys.tenantResources("tenant-1"),
      adminQueryKeys.definitions(),
      adminQueryKeys.definition("definition-1"),
      adminQueryKeys.apiClients(),
      adminQueryKeys.sessions(),
      adminQueryKeys.settings(),
      adminQueryKeys.webhookTargets(),
      adminQueryKeys.webhookDeliveries(),
      adminQueryKeys.webhookOverview(),
    ];

    expect(keys.every(([root]) => root === "admin")).toBe(true);
  });

  it("keeps list and detail keys distinct and deterministic", () => {
    expect(adminQueryKeys.tenants()).toEqual(["admin", "tenants"]);
    expect(adminQueryKeys.tenant("tenant-1")).toEqual([
      "admin",
      "tenants",
      "detail",
      "tenant-1",
    ]);
    expect(adminQueryKeys.tenant("tenant-1")).toEqual(adminQueryKeys.tenant("tenant-1"));
    expect(adminQueryKeys.tenant("tenant-1")).not.toEqual(adminQueryKeys.tenant("tenant-2"));
  });

  it("provides a shared prefix for every tenant resource list", () => {
    expect(adminQueryKeys.tenantResourcesRoot()).toEqual(["admin", "tenant-resources"]);
    expect(adminQueryKeys.tenantResources("tenant-1")).toEqual([
      "admin",
      "tenant-resources",
      "tenant-1",
    ]);
  });
});
