import { describe, expect, it } from "vitest";
import type { TenantResource } from "@/lib/api";
import { summarizeTenantOverview } from "./tenant-overview";

function resource(overrides: Partial<TenantResource> = {}): TenantResource {
  return {
    definitionId: "definition-1",
    definitionKey: "postgres",
    fields: [{ dataType: "string", isSecret: false, key: "host", label: "Host", required: true, value: "db.local" }],
    id: "resource-1",
    name: "PostgreSQL",
    status: "active",
    ...overrides,
  };
}

describe("summarizeTenantOverview", () => {
  it("marks a consumable tenant as ready", () => {
    expect(summarizeTenantOverview({ domainCount: 1, resources: [resource()], tenantStatus: "active" }))
      .toMatchObject({
        activeResourceCount: 1,
        attentionCodes: [],
        incompleteResourceCount: 0,
        readiness: "ready",
        readyRequirementCount: 3,
        totalResourceCount: 1,
      });
  });

  it("reports missing requirements and incomplete configuration", () => {
    const incomplete = resource({
      fields: [{ dataType: "string", isSecret: false, key: "host", label: "Host", required: true, value: "" }],
    });

    expect(summarizeTenantOverview({ domainCount: 0, resources: [incomplete], tenantStatus: "inactive" }))
      .toMatchObject({
        attentionCodes: ["inactive_tenant", "missing_domain", "incomplete_resources"],
        incompleteResourceCount: 1,
        readiness: "incomplete",
        readyRequirementCount: 1,
      });
  });

  it("uses attention when the essentials are ready but inactive resources remain", () => {
    const inactive = resource({ id: "resource-2", status: "inactive" });

    expect(summarizeTenantOverview({ domainCount: 1, resources: [resource(), inactive], tenantStatus: "active" }))
      .toMatchObject({
        attentionCodes: ["inactive_resources"],
        inactiveResourceCount: 1,
        readiness: "attention",
        readyRequirementCount: 3,
      });
  });
});
