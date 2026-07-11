import { describe, expect, it } from "vitest";
import { matchesTenantSlug } from "./tenant-delete";

describe("matchesTenantSlug", () => {
  it("requires the exact tenant slug", () => {
    expect(matchesTenantSlug("acme", "acme")).toBe(true);
    expect(matchesTenantSlug("ACME", "acme")).toBe(false);
    expect(matchesTenantSlug("other", "acme")).toBe(false);
  });

  it("ignores accidental surrounding whitespace", () => {
    expect(matchesTenantSlug("  acme  ", "acme")).toBe(true);
  });
});
