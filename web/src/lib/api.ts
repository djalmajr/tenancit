// Thin client for the admin API. Same-origin (SPA served by the Go binary).
const BASE = "/v1/admin";
const ADMIN_TOKEN_KEY = "konvarioAdminToken";
const ADMIN_AUTH_REQUIRED_EVENT = "admin-auth-required";

export function clearAdminToken() {
  localStorage.removeItem(ADMIN_TOKEN_KEY);
  window.dispatchEvent(new Event("admin-token-change"));
}

export function getAdminToken(): string {
  return localStorage.getItem(ADMIN_TOKEN_KEY) ?? "";
}

export function setAdminToken(token: string) {
  localStorage.setItem(ADMIN_TOKEN_KEY, token);
  window.dispatchEvent(new Event("admin-token-change"));
}

function notifyAdminAuthRequired(messageKey = "auth.requiredAccess") {
  window.dispatchEvent(new CustomEvent(ADMIN_AUTH_REQUIRED_EVENT, { detail: { messageKey } }));
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAdminToken();
  const res = await fetch(BASE + path, {
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    ...init,
  });
  if (res.status === 401) {
    notifyAdminAuthRequired("auth.invalidToken");
    throw new Error("401: autenticação admin necessária");
  }
  if (!res.ok) throw new Error(`${res.status}: ${await res.text()}`);
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export interface Tenant {
  id: string;
  slug: string;
  name: string;
  status: string;
}

export interface TenantDomain {
  id: string;
  hostname: string;
}

export interface Definition {
  id: string;
  key: string;
  name: string;
  description: string;
  icon: string;
  status: string;
  fieldCount?: number;
  secretCount?: number;
}

export interface Field {
  id: string;
  resource_definition_id: string;
  key: string;
  label: string;
  hint: string;
  data_type: string;
  required: boolean;
  is_secret: boolean;
  sort_order: number;
}

export interface DefinitionDetail {
  definition: Definition;
  fields: Field[];
}

export interface ResourceFieldValue {
  key: string;
  label: string;
  dataType: string;
  required: boolean;
  isSecret: boolean;
  value: string;
}

export interface TenantResource {
  id: string;
  definitionKey: string;
  definitionId: string;
  name: string;
  status: string;
  fields: ResourceFieldValue[];
}

export interface ApiClient {
  id: string;
  name: string;
  key_preview?: string;
  status: string;
  created_at?: string;
}

export interface OverviewTenant {
  id: string;
  name: string;
  slug: string;
  status: string;
  primaryHost: string;
  resourceCount: number;
}

export interface Overview {
  tenants: number;
  activeTenants: number;
  domains: number;
  resources: number;
  definitions: number;
  activeDefinitions: number;
  apiClients: number;
  tenantCards: OverviewTenant[];
}

export const api = {
  overview: () => req<Overview>("/overview"),

  // tenants
  listTenants: () => req<Tenant[]>("/tenants"),
  getTenant: (id: string) => req<Tenant>(`/tenants/${id}`),
  createTenant: (body: { slug: string; name: string }) =>
    req<Tenant>("/tenants", { method: "POST", body: JSON.stringify(body) }),
  updateTenant: (id: string, body: { name: string; slug: string; status: string }) =>
    req<Tenant>(`/tenants/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  listDomains: (id: string) => req<TenantDomain[]>(`/tenants/${id}/domains`),
  addDomain: (id: string, hostname: string) =>
    req(`/tenants/${id}/domains`, { method: "POST", body: JSON.stringify({ hostname }) }),
  removeDomain: (id: string, domainId: string) =>
    req(`/tenants/${id}/domains/${domainId}`, { method: "DELETE" }),
  listTenantResources: (id: string, reveal = false) =>
    req<TenantResource[]>(`/tenants/${id}/resources${reveal ? "?reveal=true" : ""}`),
  createResource: (id: string, body: { definitionKey: string; values: Record<string, string> }) =>
    req(`/tenants/${id}/resources`, { method: "POST", body: JSON.stringify(body) }),
  setResourceStatus: (id: string, resourceId: string, status: string) =>
    req(`/tenants/${id}/resources/${resourceId}/status`, {
      method: "PUT",
      body: JSON.stringify({ status }),
    }),
  deleteResource: (id: string, resourceId: string) =>
    req(`/tenants/${id}/resources/${resourceId}`, { method: "DELETE" }),

  // definitions
  listDefinitions: () => req<Definition[]>("/resource-definitions"),
  getDefinition: (id: string) => req<DefinitionDetail>(`/resource-definitions/${id}`),
  createDefinition: (body: { key: string; name: string; description?: string; icon?: string }) =>
    req<Definition>("/resource-definitions", { method: "POST", body: JSON.stringify(body) }),
  setDefinitionStatus: (id: string, status: string) =>
    req(`/resource-definitions/${id}/status`, { method: "PUT", body: JSON.stringify({ status }) }),
  addField: (
    id: string,
    body: { key: string; label?: string; dataType?: string; required?: boolean; isSecret?: boolean },
  ) => req(`/resource-definitions/${id}/fields`, { method: "POST", body: JSON.stringify(body) }),
  deleteField: (id: string, fieldId: string) =>
    req(`/resource-definitions/${id}/fields/${fieldId}`, { method: "DELETE" }),

  // api clients
  listAPIClients: () => req<ApiClient[]>("/api-clients"),
  createAPIClient: (name: string) =>
    req<{ client: ApiClient; token: string }>("/api-clients", {
      method: "POST",
      body: JSON.stringify({ name }),
    }),
  setAPIClientStatus: (id: string, status: string) =>
    req(`/api-clients/${id}/status`, { method: "PUT", body: JSON.stringify({ status }) }),
};
