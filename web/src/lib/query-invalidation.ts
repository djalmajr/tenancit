import type { QueryClient, QueryKey } from "@tanstack/react-query";
import { adminQueryKeys } from "./query-keys";

function invalidateExact(queryClient: QueryClient, queryKey: QueryKey) {
  return queryClient.invalidateQueries({ exact: true, queryKey });
}

export function invalidateOverview(queryClient: QueryClient) {
  return invalidateExact(queryClient, adminQueryKeys.overview());
}

export function invalidateTenants(queryClient: QueryClient) {
  return Promise.all([
    invalidateExact(queryClient, adminQueryKeys.tenants()),
    invalidateOverview(queryClient),
  ]);
}

export function invalidateTenant(queryClient: QueryClient, tenantId: string) {
  return Promise.all([
    invalidateExact(queryClient, adminQueryKeys.tenant(tenantId)),
    invalidateExact(queryClient, adminQueryKeys.tenants()),
    invalidateOverview(queryClient),
  ]);
}

export function invalidateTenantDomains(queryClient: QueryClient, tenantId: string) {
  return Promise.all([
    invalidateExact(queryClient, adminQueryKeys.tenantDomains(tenantId)),
    invalidateOverview(queryClient),
  ]);
}

export function invalidateTenantResources(queryClient: QueryClient, tenantId: string) {
  return Promise.all([
    invalidateExact(queryClient, adminQueryKeys.tenantResources(tenantId)),
    invalidateOverview(queryClient),
  ]);
}

export function invalidateAllTenantResources(queryClient: QueryClient) {
  return queryClient.invalidateQueries({
    queryKey: adminQueryKeys.tenantResourcesRoot(),
  });
}

export function invalidateDefinitions(queryClient: QueryClient) {
  return Promise.all([
    invalidateExact(queryClient, adminQueryKeys.definitions()),
    invalidateOverview(queryClient),
  ]);
}

export function invalidateDefinition(queryClient: QueryClient, definitionId: string) {
  return Promise.all([
    invalidateExact(queryClient, adminQueryKeys.definition(definitionId)),
    invalidateExact(queryClient, adminQueryKeys.definitions()),
    invalidateOverview(queryClient),
  ]);
}

export function invalidateApiClients(queryClient: QueryClient) {
  return Promise.all([
    invalidateExact(queryClient, adminQueryKeys.apiClients()),
    invalidateOverview(queryClient),
  ]);
}

export async function removeTenantQueries(queryClient: QueryClient, tenantId: string) {
  await Promise.all([
    queryClient.cancelQueries({ queryKey: adminQueryKeys.tenant(tenantId) }),
    queryClient.cancelQueries({ queryKey: adminQueryKeys.tenantResources(tenantId) }),
  ]);
  queryClient.removeQueries({ queryKey: adminQueryKeys.tenant(tenantId) });
  queryClient.removeQueries({ queryKey: adminQueryKeys.tenantResources(tenantId) });
  await invalidateTenants(queryClient);
}
