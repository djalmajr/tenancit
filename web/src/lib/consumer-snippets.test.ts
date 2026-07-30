import { describe, expect, it } from "vitest";
import { consumerSnippets } from "./consumer-snippets";

describe("consumerSnippets", () => {
  it("teaches the edge-safe identify then tenantId flow", () => {
    expect(consumerSnippets.identify).toContain("/v1/identify?hostname=<tenant-hostname>");
    expect(consumerSnippets.byTenantId).toContain("/v1/resolve?tenantId=<tenant-slug>");
    expect(consumerSnippets.byTenantId).toContain("If-None-Match");
  });

  it("uses placeholders and retains supported alternate paths", () => {
    for (const snippet of Object.values(consumerSnippets)) {
      expect(snippet).toContain("Bearer <token>");
      expect(snippet).not.toMatch(/Bearer tnc_[a-zA-Z0-9]+/);
    }
    expect(consumerSnippets.byHostname).toContain("resolve?hostname=");
    expect(consumerSnippets.resource).toContain("resources/<resource-alias>");
  });
});
