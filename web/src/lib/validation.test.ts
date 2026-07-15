import { describe, expect, it } from "vitest";
import {
  isValidDefinitionKey,
  isValidDateRange,
  isValidHostname,
  isIntegerInRange,
  isValidResourceAlias,
  isValidTenantSlug,
  isValidTypedResourceValue,
  isValidWebhookURL,
} from "./validation";

describe("input validation", () => {
  it.each(["mydesk", "my-desk", "tenant23"])("accepts tenant slug %s", (value) => {
    expect(isValidTenantSlug(value)).toBe(true);
  });

  it.each(["MyDesk", "my_desk", "mydesk|", "-mydesk", "mydesk-", "my--desk", ""])(
    "rejects tenant slug %s",
    (value) => expect(isValidTenantSlug(value)).toBe(false),
  );

  it.each(["postgres", "postgres_primary", "postgres-primary"])("accepts definition key %s", (value) => {
    expect(isValidDefinitionKey(value)).toBe(true);
  });

  it.each(["Postgres", "2postgres", "postgres.main", "postgres|"])("rejects definition key %s", (value) => {
    expect(isValidDefinitionKey(value)).toBe(false);
  });

  it("keeps resource aliases compatible with the server contract", () => {
    expect(isValidResourceAlias("postgres.agility")).toBe(true);
    expect(isValidResourceAlias("postgres|agility")).toBe(false);
  });

  it.each(["app.example.com", "api-2.example.com", "APP.EXAMPLE.COM.", "localhost"])("accepts hostname %s", (value) => {
    expect(isValidHostname(value)).toBe(true);
  });

  it.each(["-app.example.com", "app_.example.com", "https://example.com"])(
    "rejects hostname %s",
    (value) => expect(isValidHostname(value)).toBe(false),
  );

  it("accepts HTTPS webhook URLs and loopback HTTP only", () => {
    expect(isValidWebhookURL("https://receiver.example/hooks")).toBe(true);
    expect(isValidWebhookURL("http://localhost:8080/hooks")).toBe(true);
    expect(isValidWebhookURL("http://receiver.example/hooks")).toBe(false);
    expect(isValidWebhookURL("https://user:secret@receiver.example/hooks")).toBe(false);
    expect(isValidWebhookURL("https://receiver.example/hooks#secret")).toBe(false);
  });

  it("accepts only whole numbers inside an explicit range", () => {
    expect(isIntegerInRange("300", 1, 10_000)).toBe(true);
    expect(isIntegerInRange("1.5", 1, 10_000)).toBe(false);
    expect(isIntegerInRange("0", 1, 10_000)).toBe(false);
    expect(isIntegerInRange("10001", 1, 10_000)).toBe(false);
  });

  it("requires a valid chronological date range", () => {
    expect(isValidDateRange("2026-07-01T10:00", "2026-07-01T11:00")).toBe(true);
    expect(isValidDateRange("2026-07-01T11:00", "2026-07-01T10:00")).toBe(false);
    expect(isValidDateRange("not-a-date", "2026-07-01T11:00")).toBe(false);
  });

  it("validates resource values using their declared data type", () => {
    expect(isValidTypedResourceValue("int", "-42")).toBe(true);
    expect(isValidTypedResourceValue("int", "4.2")).toBe(false);
    expect(isValidTypedResourceValue("bool", "true")).toBe(true);
    expect(isValidTypedResourceValue("bool", "yes")).toBe(false);
    expect(isValidTypedResourceValue("string", "any value")).toBe(true);
  });
});
