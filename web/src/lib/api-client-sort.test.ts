import { describe, expect, it } from "vitest";
import { apiClientCreatedAtSortValue } from "./api-client-sort";

describe("API client created-at sort", () => {
  it("orders ISO timestamps chronologically even when fractional precision differs", () => {
    const older = apiClientCreatedAtSortValue("2026-07-10T12:00:00.071593Z");
    const newer = apiClientCreatedAtSortValue("2026-07-10T12:00:00.08512Z");

    expect(older).toBeLessThan(newer);
  });

  it("places missing or malformed timestamps at the zero fallback", () => {
    expect(apiClientCreatedAtSortValue(undefined)).toBe(0);
    expect(apiClientCreatedAtSortValue("not-a-date")).toBe(0);
  });
});
