import { describe, expect, it } from "vitest";
import { formatDate } from "./date-format";
import { normalizeLocale, translate } from "./i18n";

describe("i18n", () => {
  it("normalizes supported locale aliases", () => {
    expect(normalizeLocale("en")).toBe("en-US");
    expect(normalizeLocale("es_ES")).toBe("es-ES");
    expect(normalizeLocale("pt-br")).toBe("pt-BR");
  });

  it("falls back to pt-BR for unsupported locales", () => {
    expect(normalizeLocale("fr-FR")).toBe("pt-BR");
  });

  it("translates and interpolates messages", () => {
    expect(translate("en-US", "auth.title")).toBe("Administrative access");
    expect(translate("es-ES", "overview.totalTenants", { count: 3 })).toBe("3 en total");
    expect(translate("pt-BR", "overview.resourceCount", { count: 2 })).toBe("2 recursos");
  });

  it("formats dates with the selected locale without shifting UTC calendar dates", () => {
    const value = "2026-06-30T00:00:00Z";
    expect(formatDate("pt-BR", value, { timeZone: "UTC" })).toBe("30/06/2026");
    expect(formatDate("en-US", value, { timeZone: "UTC" })).toBe("6/30/2026");
    expect(formatDate("es-ES", value, { timeZone: "UTC" })).toBe("30/6/2026");
  });
});
