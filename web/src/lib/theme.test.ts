import { describe, expect, it } from "vitest";
import { applyTheme, normalizeTheme, resolveTheme } from "./theme";

describe("theme", () => {
  it("normalizes unknown values to system", () => {
    expect(normalizeTheme("dark")).toBe("dark");
    expect(normalizeTheme("light")).toBe("light");
    expect(normalizeTheme("system")).toBe("system");
    expect(normalizeTheme(null)).toBe("system");
  });

  it("applies the dark class and color scheme", () => {
    const root = document.createElement("html");

    applyTheme("dark", root);
    expect(root.classList.contains("dark")).toBe(true);
    expect(root.style.colorScheme).toBe("dark");

    applyTheme("light", root);
    expect(root.classList.contains("dark")).toBe(false);
    expect(root.style.colorScheme).toBe("light");
  });

  it("resolves system to light when media queries are unavailable", () => {
    expect(resolveTheme("system")).toBe("light");
  });
});
