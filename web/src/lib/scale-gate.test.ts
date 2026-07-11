import { describe, expect, it } from "vitest";
import { decideScaleGate, nearestRank, parseObservedOperationalVolume } from "./scale-gate";

describe("scale pagination gate", () => {
  it("uses nearest-rank percentiles", () => {
    expect(nearestRank(Array.from({ length: 100 }, (_, index) => index + 1), 0.95)).toBe(95);
  });

  it("does not confuse a synthetic capacity breakpoint with operational volume", () => {
    expect(
      decideScaleGate(100, [
        { cardinality: 100, hardTriggers: [], softItemCount: false, softPayload: false },
        { cardinality: 5_000, hardTriggers: ["payload"], softItemCount: true, softPayload: true },
      ]),
    ).toBe("KEEP_FULL_LISTS");
  });

  it("opens the epic at an observed hard cardinality", () => {
    expect(decideScaleGate(2_000, [])).toBe("OPEN_PAGINATION_EPIC");
  });

  it("opens the epic when both soft triggers persist at the operational point", () => {
    expect(
      decideScaleGate(500, [
        { cardinality: 500, hardTriggers: [], softItemCount: true, softPayload: true },
      ]),
    ).toBe("OPEN_PAGINATION_EPIC");
  });

  it("defaults an omitted operational volume to zero", () => {
    expect(parseObservedOperationalVolume(undefined)).toBe(0);
  });

  it.each(["", "-1", "1.5", "not-a-number", "Infinity"])(
    "rejects invalid operational volume %j instead of failing open",
    (value) => {
      expect(() => parseObservedOperationalVolume(value)).toThrow(/non-negative integer/);
    },
  );
});
