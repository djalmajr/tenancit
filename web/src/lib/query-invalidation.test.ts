import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import { adminQueryKeys } from "./query-keys";
import {
  invalidateApiClients,
  invalidateAllTenantResources,
  invalidateDefinition,
  invalidateTenantDomains,
  invalidateTenantResources,
  removeTenantQueries,
} from "./query-invalidation";

function createClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false } } });
}

describe("admin query invalidation", () => {
  it("invalidates a resource mutation's masked resources and overview", async () => {
    const queryClient = createClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");

    await invalidateTenantResources(queryClient, "tenant-1");

    expect(invalidate).toHaveBeenCalledWith({
      exact: true,
      queryKey: adminQueryKeys.tenantResources("tenant-1"),
    });
    expect(invalidate).toHaveBeenCalledWith({
      exact: true,
      queryKey: adminQueryKeys.overview(),
    });
  });

  it("invalidates dependent lists after definition, domain, and client mutations", async () => {
    const queryClient = createClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");

    await invalidateDefinition(queryClient, "definition-1");
    await invalidateTenantDomains(queryClient, "tenant-1");
    await invalidateApiClients(queryClient);

    const invalidatedKeys = invalidate.mock.calls.map(([filters]) => filters?.queryKey);
    expect(invalidatedKeys).toContainEqual(adminQueryKeys.definition("definition-1"));
    expect(invalidatedKeys).toContainEqual(adminQueryKeys.definitions());
    expect(invalidatedKeys).toContainEqual(adminQueryKeys.tenantDomains("tenant-1"));
    expect(invalidatedKeys).toContainEqual(adminQueryKeys.apiClients());
    expect(invalidatedKeys).toContainEqual(adminQueryKeys.overview());
  });

  it("removes deleted tenant detail data before invalidating collection views", async () => {
    const queryClient = createClient();
    queryClient.setQueryData(adminQueryKeys.tenant("tenant-1"), { id: "tenant-1" });
    queryClient.setQueryData(adminQueryKeys.tenantDomains("tenant-1"), [{ id: "domain-1" }]);
    queryClient.setQueryData(adminQueryKeys.tenantResources("tenant-1"), [{ id: "resource-1" }]);

    await removeTenantQueries(queryClient, "tenant-1");

    expect(queryClient.getQueryData(adminQueryKeys.tenant("tenant-1"))).toBeUndefined();
    expect(queryClient.getQueryData(adminQueryKeys.tenantDomains("tenant-1"))).toBeUndefined();
    expect(queryClient.getQueryData(adminQueryKeys.tenantResources("tenant-1"))).toBeUndefined();
  });

  it("invalidates every masked resource list after a definition field changes", async () => {
    const queryClient = createClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");

    await invalidateAllTenantResources(queryClient);

    expect(invalidate).toHaveBeenCalledWith({
      queryKey: adminQueryKeys.tenantResourcesRoot(),
    });
  });
});
