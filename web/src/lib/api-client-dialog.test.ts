import { describe, expect, it } from "vitest";
import {
  canChangeAPIClientDialogOpen,
  isDuplicateAPIClientName,
} from "@/lib/api-client-dialog";

describe("API client create dialog", () => {
  it("cannot be dismissed while the one-time token request is pending", () => {
    expect(canChangeAPIClientDialogOpen(false, true)).toBe(false);
    expect(canChangeAPIClientDialogOpen(true, true)).toBe(true);
  });

  it("can close after the request settles", () => {
    expect(canChangeAPIClientDialogOpen(false, false)).toBe(true);
  });

  // Mutation captured: removing trim/lowercase normalization lets an existing client be submitted again.
  it("rejects an existing client name regardless of case and surrounding spaces", () => {
    expect(isDuplicateAPIClientName("  Billing Service ", ["billing service", "edge-worker"])).toBe(true);
    expect(isDuplicateAPIClientName("new-worker", ["billing service", "edge-worker"])).toBe(false);
  });
});
