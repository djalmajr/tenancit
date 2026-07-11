import { expect, type APIRequestContext } from "@playwright/test";
import { adminRequest, adminToken, uniqueID } from "./admin";

export type TenantRecord = { id: string; name: string; slug: string; status: string };
export type DefinitionRecord = { id: string; key: string; name: string; status: string };
export type FieldRecord = { id: string; key: string; label: string; required: boolean; is_secret: boolean };
export type DomainRecord = { id: string; hostname: string };
export type ResourceRecord = { id: string; definitionKey: string; status: string };
export type APIClientRecord = {
  created_at?: string;
  id: string;
  key_preview?: string;
  name: string;
  status: string;
};

export class CatalogCleanup {
  private readonly tenants = new Set<string>();
  private readonly definitions = new Set<string>();
  private readonly apiClients = new Set<string>();
  private readonly definitionStatusRestores = new Map<string, string>();

  trackTenant(id: string) {
    this.tenants.add(id);
  }

  trackDefinition(id: string) {
    this.definitions.add(id);
  }

  trackAPIClient(id: string) {
    this.apiClients.add(id);
  }

  restoreDefinitionStatus(id: string, status: string) {
    if (!this.definitionStatusRestores.has(id)) {
      this.definitionStatusRestores.set(id, status);
    }
  }

  async run(request: APIRequestContext) {
    const headers = { Authorization: `Bearer ${adminToken}` };
    const failures: string[] = [];
    for (const id of this.apiClients) {
      const response = await request.post(`/v1/admin/api-clients/${id}/revoke`, {
        headers,
      });
      if (![200, 404].includes(response.status())) failures.push(`revoke api client ${id}: ${response.status()}`);
      if (response.status() === 200) {
        const deleted = await request.delete(`/v1/admin/api-clients/${id}`, { headers });
        if (![204, 404].includes(deleted.status())) failures.push(`delete api client ${id}: ${deleted.status()}`);
      }
    }
    for (const id of this.tenants) {
      const response = await request.delete(`/v1/admin/tenants/${id}`, { headers });
      if (![204, 404].includes(response.status())) failures.push(`delete tenant ${id}: ${response.status()}`);
    }
    for (const id of this.definitions) {
      const response = await request.put(`/v1/admin/resource-definitions/${id}/status`, {
        data: { status: "inactive" },
        headers,
      });
      if (![200, 404].includes(response.status())) failures.push(`deactivate definition ${id}: ${response.status()}`);
    }
    for (const [id, status] of this.definitionStatusRestores) {
      const response = await request.put(`/v1/admin/resource-definitions/${id}/status`, {
        data: { status },
        headers,
      });
      if (![200, 404].includes(response.status())) failures.push(`restore definition ${id}: ${response.status()}`);
    }
    expect(failures, `catalog cleanup failures:\n${failures.join("\n")}`).toEqual([]);
  }
}

export async function createTenantFixture(
  request: APIRequestContext,
  cleanup: CatalogCleanup,
  prefix = "tenant",
) {
  const slug = uniqueID(prefix);
  const tenant = await adminRequest<TenantRecord>(request, "post", "/v1/admin/tenants", {
    name: `E2E ${slug}`,
    slug,
  });
  cleanup.trackTenant(tenant.id);
  return tenant;
}

export async function createDefinitionFixture(
  request: APIRequestContext,
  cleanup: CatalogCleanup,
  options: {
    fields?: Array<{ key: string; label: string; dataType?: string; required?: boolean; isSecret?: boolean }>;
    prefix?: string;
  } = {},
) {
  const key = uniqueID(options.prefix ?? "definition");
  const definition = await adminRequest<DefinitionRecord>(
    request,
    "post",
    "/v1/admin/resource-definitions",
    { key, name: `E2E ${key}`, description: `Definition ${key}` },
  );
  cleanup.trackDefinition(definition.id);
  const fields: FieldRecord[] = [];
  for (const field of options.fields ?? []) {
    fields.push(
      await adminRequest<FieldRecord>(
        request,
        "post",
        `/v1/admin/resource-definitions/${definition.id}/fields`,
        field,
      ),
    );
  }
  return { definition, fields };
}

export async function suspendActiveDefinitions(
  request: APIRequestContext,
  cleanup: CatalogCleanup,
  keepIds: Iterable<string> = [],
) {
  const kept = new Set(keepIds);
  const definitions = (await adminRequest<DefinitionRecord[] | null>(
    request,
    "get",
    "/v1/admin/resource-definitions",
  )) ?? [];
  for (const definition of definitions) {
    if (definition.status !== "active" || kept.has(definition.id)) continue;
    cleanup.restoreDefinitionStatus(definition.id, definition.status);
    await adminRequest<DefinitionRecord>(
      request,
      "put",
      `/v1/admin/resource-definitions/${definition.id}/status`,
      { status: "inactive" },
    );
  }
}

export async function createDomainFixture(
  request: APIRequestContext,
  tenantId: string,
  hostname = `${uniqueID("host")}.e2e.local`,
) {
  return adminRequest<DomainRecord>(request, "post", `/v1/admin/tenants/${tenantId}/domains`, { hostname });
}

export async function createResourceFixture(
  request: APIRequestContext,
  tenantId: string,
  definitionKey: string,
  values: Record<string, string>,
) {
  return adminRequest<ResourceRecord>(request, "post", `/v1/admin/tenants/${tenantId}/resources`, {
    definitionKey,
    values,
  });
}

export async function createAPIClientFixture(
  request: APIRequestContext,
  cleanup: CatalogCleanup,
  prefix = "client",
) {
  const name = uniqueID(prefix);
  const created = await adminRequest<{ client: APIClientRecord; token: string }>(
    request,
    "post",
    "/v1/admin/api-clients",
    {
      name,
      scopes: ["tenant:identify", "resource:resolve"],
      rpm_limit: 300,
      expires_at: new Date(Date.now() + 90 * 24 * 60 * 60 * 1000).toISOString(),
    },
  );
  cleanup.trackAPIClient(created.client.id);
  return { ...created, name };
}

export async function findTenantBySlug(request: APIRequestContext, slug: string) {
  let tenant: TenantRecord | undefined;
  await expect.poll(async () => {
    const tenants = (await adminRequest<TenantRecord[] | null>(request, "get", "/v1/admin/tenants")) ?? [];
    tenant = tenants.find((candidate) => candidate.slug === slug);
    return tenant !== undefined;
  }, { message: `tenant ${slug} created by UI` }).toBe(true);
  return tenant!;
}

export async function findDefinitionByKey(request: APIRequestContext, key: string) {
  let definition: DefinitionRecord | undefined;
  await expect.poll(async () => {
    const definitions = (await adminRequest<DefinitionRecord[] | null>(
      request,
      "get",
      "/v1/admin/resource-definitions",
    )) ?? [];
    definition = definitions.find((candidate) => candidate.key === key);
    return definition !== undefined;
  }, { message: `definition ${key} created by UI` }).toBe(true);
  return definition!;
}

export async function findAPIClientByName(request: APIRequestContext, name: string) {
  let client: APIClientRecord | undefined;
  await expect.poll(async () => {
    const clients = (await adminRequest<APIClientRecord[] | null>(request, "get", "/v1/admin/api-clients")) ?? [];
    client = clients.find((candidate) => candidate.name === name);
    return client !== undefined;
  }, { message: `API client ${name} created by UI` }).toBe(true);
  return client!;
}
