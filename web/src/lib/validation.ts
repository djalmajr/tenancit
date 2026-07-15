export const TENANT_SLUG_PATTERN = /^[a-z0-9]+(?:-[a-z0-9]+)*$/;
export const DEFINITION_KEY_PATTERN = /^[a-z][a-z0-9_-]{0,62}$/;
export const RESOURCE_ALIAS_PATTERN = /^[a-z0-9][a-z0-9._-]{0,62}$/;

export function isValidTenantSlug(value: string): boolean {
  const slug = value.trim();
  return slug.length <= 63 && TENANT_SLUG_PATTERN.test(slug);
}

export function isValidDefinitionKey(value: string): boolean {
  return DEFINITION_KEY_PATTERN.test(value.trim());
}

export function isValidResourceAlias(value: string): boolean {
  return RESOURCE_ALIAS_PATTERN.test(value.trim());
}

export function isValidHostname(value: string): boolean {
  const hostname = value.trim().toLowerCase().replace(/\.$/, "");
  if (!hostname || hostname.length > 253) return false;

  return hostname.split(".").every((label) => (
    label.length > 0
    && label.length <= 63
    && !label.startsWith("-")
    && !label.endsWith("-")
    && /^[a-z0-9-]+$/.test(label)
  ));
}

export function isValidWebhookURL(value: string): boolean {
  try {
    const parsed = new URL(value.trim());
    if (parsed.username || parsed.password || parsed.hash || !parsed.hostname) return false;
    if (parsed.protocol === "https:") return true;
    return parsed.protocol === "http:"
      && ["localhost", "127.0.0.1", "::1"].includes(parsed.hostname);
  } catch {
    return false;
  }
}

export function isIntegerInRange(value: string, minimum: number, maximum: number): boolean {
  if (!/^-?\d+$/.test(value.trim())) return false;
  const parsed = Number(value);
  return Number.isSafeInteger(parsed) && parsed >= minimum && parsed <= maximum;
}

export function isValidDateRange(from: string, to: string): boolean {
  const fromTime = Date.parse(from);
  const toTime = Date.parse(to);
  return Number.isFinite(fromTime) && Number.isFinite(toTime) && fromTime < toTime;
}

export function isValidTypedResourceValue(dataType: string, value: string): boolean {
  if (value === "") return true;
  if (dataType === "int") return /^-?\d+$/.test(value);
  if (dataType === "bool") return value === "true" || value === "false";
  return true;
}
