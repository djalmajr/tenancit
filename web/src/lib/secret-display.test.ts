import { describe, expect, it } from "vitest";
import { displaySecretValue, SECRET_MASK } from "./secret-display";

describe("displaySecretValue", () => {
  it("never returns cleartext for a hidden secret", () => {
    expect(displaySecretValue({ isSecret: true, revealed: false, value: "hunter2" })).toBe(
      SECRET_MASK,
    );
  });

  it("returns cleartext only when a secret is revealed", () => {
    expect(displaySecretValue({ isSecret: true, revealed: true, value: "hunter2" })).toBe(
      "hunter2",
    );
  });

  it("keeps non-secret values visible", () => {
    expect(displaySecretValue({ isSecret: false, revealed: false, value: "db.internal" })).toBe(
      "db.internal",
    );
  });
});
