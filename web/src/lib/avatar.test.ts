import { describe, expect, it } from "vitest";
import { avatarHue, initials } from "@/lib/avatar";

describe("avatar helpers", () => {
  it("builds compact initials for people and shared credentials", () => {
    expect(initials("Djalma Junior")).toBe("DJ");
    expect(initials("Credencial")).toBe("CR");
    expect(initials(" ")).toBe("?");
  });

  it("returns a stable color for the same identity", () => {
    expect(avatarHue("subject-1")).toBe(avatarHue("subject-1"));
    expect(avatarHue("subject-1")).not.toBe(avatarHue("subject-2"));
  });
});
