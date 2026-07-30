// @vitest-environment node
import { describe, expect, it } from "vitest";
import viteConfig from "../vite.config";

describe("Vite deployment base", () => {
  // Mutation captured during Red: the absent base emitted host-root asset URLs.
  it("emits relative asset URLs for runtime base-path injection", () => {
    expect(viteConfig).toMatchObject({ base: "./" });
  });
});
