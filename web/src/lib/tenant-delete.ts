export function matchesTenantSlug(typedValue: string, tenantSlug: string): boolean {
  return typedValue.trim() === tenantSlug;
}
