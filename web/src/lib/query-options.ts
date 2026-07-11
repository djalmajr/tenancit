import { queryOptions } from "@tanstack/react-query";
import { api } from "./api";
import { adminQueryKeys } from "./query-keys";

export const adminQueryOptions = {
  apiClients: () => queryOptions({
    queryFn: ({ signal }) => api.listAPIClients(signal),
    queryKey: adminQueryKeys.apiClients(),
  }),
  definition: (definitionId: string) => queryOptions({
    queryFn: ({ signal }) => api.getDefinition(definitionId, signal),
    queryKey: adminQueryKeys.definition(definitionId),
  }),
  definitions: () => queryOptions({
    queryFn: ({ signal }) => api.listDefinitions(signal),
    queryKey: adminQueryKeys.definitions(),
  }),
  overview: () => queryOptions({
    queryFn: ({ signal }) => api.overview(signal),
    queryKey: adminQueryKeys.overview(),
  }),
  tenant: (tenantId: string) => queryOptions({
    queryFn: ({ signal }) => api.getTenant(tenantId, signal),
    queryKey: adminQueryKeys.tenant(tenantId),
  }),
  tenantDomains: (tenantId: string) => queryOptions({
    queryFn: ({ signal }) => api.listDomains(tenantId, signal),
    queryKey: adminQueryKeys.tenantDomains(tenantId),
  }),
  tenantResources: (tenantId: string) => queryOptions({
    queryFn: ({ signal }) => api.listTenantResources(tenantId, false, signal),
    queryKey: adminQueryKeys.tenantResources(tenantId),
  }),
  tenants: () => queryOptions({
    queryFn: ({ signal }) => api.listTenants(signal),
    queryKey: adminQueryKeys.tenants(),
  }),
};
