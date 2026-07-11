import { describe, expect, it, vi } from "vitest";
import { stableIdempotencyKey, type IdempotencyAttempt } from "./idempotency";

describe("stableIdempotencyKey", () => {
  it("reuses a key for the same payload and rotates it after an edit", () => {
    const randomUUID = vi.spyOn(crypto, "randomUUID")
      .mockReturnValueOnce("00000000-0000-4000-8000-000000000001")
      .mockReturnValueOnce("00000000-0000-4000-8000-000000000002");
    const ref: { current: IdempotencyAttempt } = { current: null };
    const first = stableIdempotencyKey(ref, { name: "same" });
    const retry = stableIdempotencyKey(ref, { name: "same" });
    const edited = stableIdempotencyKey(ref, { name: "edited" });
    expect(retry).toBe(first);
    expect(edited).not.toBe(first);
    expect(randomUUID).toHaveBeenCalledTimes(2);
    randomUUID.mockRestore();
  });
});
