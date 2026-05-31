import { describe, it, expect } from "vitest";
import { cn } from "./utils";

describe("cn", () => {
  // Mutation captured: replacing twMerge with plain join would keep both
  // conflicting classes instead of the last winning → "px-2 px-4" not "px-4".
  it("dedupes conflicting tailwind classes (last wins)", () => {
    expect(cn("px-2", "px-4")).toBe("px-4");
  });

  it("drops falsy values and merges the rest", () => {
    expect(cn("a", false && "b", undefined, "c")).toBe("a c");
  });
});
