export const adminQueryKeys = {
  all: ["admin"] as const,
  apiClients: () => [...adminQueryKeys.all, "api-clients"] as const,
  definition: (definitionId: string) =>
    [...adminQueryKeys.definitions(), "detail", definitionId] as const,
  definitions: () => [...adminQueryKeys.all, "definitions"] as const,
  overview: () => [...adminQueryKeys.all, "overview"] as const,
  operationalHealth: () => [...adminQueryKeys.all, "operational-health"] as const,
  sessions: () => [...adminQueryKeys.all, "sessions"] as const,
  settings: () => [...adminQueryKeys.all, "settings"] as const,
  webhookTargets: () => [...adminQueryKeys.all, "webhook-targets"] as const,
  webhookDeliveries: () => [...adminQueryKeys.all, "webhook-deliveries"] as const,
  webhookOverview: () => [...adminQueryKeys.all, "webhook-overview"] as const,
  tenant: (tenantId: string) =>
    [...adminQueryKeys.tenants(), "detail", tenantId] as const,
  tenantDomains: (tenantId: string) =>
    [...adminQueryKeys.tenant(tenantId), "domains"] as const,
  tenantResourcesRoot: () => [...adminQueryKeys.all, "tenant-resources"] as const,
  tenantResources: (tenantId: string) =>
    [...adminQueryKeys.tenantResourcesRoot(), tenantId] as const,
  tenants: () => [...adminQueryKeys.all, "tenants"] as const,
};
