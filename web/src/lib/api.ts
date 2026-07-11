// Thin client for the admin API. Same-origin (SPA served by the Go binary).
const BASE = "/v1/admin";
export const ADMIN_TOKEN_KEY = "tenancitAdminToken";
export const ADMIN_TOKEN_CHANGE_EVENT = "admin-token-change";
export const ADMIN_SESSION_CHANGE_EVENT = "admin-session-change";
const ADMIN_AUTH_REQUIRED_EVENT = "admin-auth-required";
export const REQUEST_TIMEOUT_MS = 10_000;
export type AdminAuthMessage = "auth.invalidToken" | "auth.requiredAccess" | "auth.sessionExpired";

export type AdminAuthMode = "oidc" | "legacy_shared_token";

export interface AdminAuthConfig {
  mode: AdminAuthMode;
  login_url?: string;
}

export interface AdminSession {
  kind: "oidc_user";
  issuer: string;
  subject: string;
  label: string;
  session_id: string;
  roles: string[];
  csrf_token: string;
  expires_at: string;
  idle_expires_at: string;
}

let currentAdminSession: AdminSession | undefined;

export function setAdminSession(session: AdminSession | undefined) {
  currentAdminSession = session;
  window.dispatchEvent(new Event(ADMIN_SESSION_CHANGE_EVENT));
}

export function getAdminSession(): AdminSession | undefined {
  return currentAdminSession;
}

let pendingAdminAuthMessage: AdminAuthMessage | undefined;

// A 401 can arrive before AppShell's effect subscribes (child query effects
// mount before the parent effect). Retain the event payload so the late
// subscriber can replay it instead of silently falling back to generic copy.
export function consumePendingAdminAuthMessage(): AdminAuthMessage | undefined {
  const message = pendingAdminAuthMessage;
  pendingAdminAuthMessage = undefined;
  return message;
}

export function clearAdminToken() {
  localStorage.removeItem(ADMIN_TOKEN_KEY);
  window.dispatchEvent(new Event(ADMIN_TOKEN_CHANGE_EVENT));
}

export function getAdminToken(): string {
  return localStorage.getItem(ADMIN_TOKEN_KEY) ?? "";
}

export function setAdminToken(token: string) {
  pendingAdminAuthMessage = undefined;
  localStorage.setItem(ADMIN_TOKEN_KEY, token);
  window.dispatchEvent(new Event(ADMIN_TOKEN_CHANGE_EVENT));
}

function notifyAdminAuthRequired(messageKey: AdminAuthMessage = "auth.requiredAccess") {
  pendingAdminAuthMessage = messageKey;
  window.dispatchEvent(new CustomEvent(ADMIN_AUTH_REQUIRED_EVENT, { detail: { messageKey } }));
}

/** Error thrown by the admin API client, carrying the HTTP status so the UI can
 *  map it to a human, localized message instead of surfacing raw backend text. */
export class ApiError extends Error {
  readonly status: number;
  readonly serverMessage: string;
  constructor(status: number, serverMessage = "") {
    super(serverMessage ? `${status}: ${serverMessage}` : String(status));
    this.name = "ApiError";
    this.status = status;
    this.serverMessage = serverMessage;
  }
}

export class ApiTimeoutError extends Error {
  readonly timeoutMs: number;

  constructor(timeoutMs = REQUEST_TIMEOUT_MS) {
    super(`request timed out after ${timeoutMs}ms`);
    this.name = "ApiTimeoutError";
    this.timeoutMs = timeoutMs;
  }
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAdminToken();
  const session = getAdminSession();
  const controller = new AbortController();
  const upstreamSignal = init?.signal;
  let timedOut = false;
  const timeoutID = window.setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, REQUEST_TIMEOUT_MS);
  const abortFromCaller = () => controller.abort(upstreamSignal?.reason);
  if (upstreamSignal?.aborted) abortFromCaller();
  else upstreamSignal?.addEventListener("abort", abortFromCaller, { once: true });

  try {
    const res = await fetch(BASE + path, {
      ...init,
      credentials: "same-origin",
      // headers must remain after init so callers cannot accidentally discard
      // the authentication boundary above.
      headers: {
        "Content-Type": "application/json",
        ...(!session && token ? { Authorization: `Bearer ${token}` } : {}),
        ...(session ? { "X-CSRF-Token": session.csrf_token } : {}),
        ...(init?.headers ?? {}),
      },
      signal: controller.signal,
    });
    if (res.status === 401) {
      if (session) {
        setAdminSession(undefined);
        notifyAdminAuthRequired("auth.sessionExpired");
      } else {
        notifyAdminAuthRequired("auth.invalidToken");
      }
      throw new ApiError(401);
    }
    if (!res.ok) {
      let serverMessage = "";
      try {
        const body = (await res.json()) as { error?: unknown };
        if (typeof body?.error === "string") serverMessage = body.error;
      } catch {
        // non-JSON error body; leave serverMessage empty
      }
      throw new ApiError(res.status, serverMessage);
    }
    if (res.status === 204) return undefined as T;
    return (await res.json()) as T;
  } catch (error) {
    if (timedOut) throw new ApiTimeoutError();
    throw error;
  } finally {
    window.clearTimeout(timeoutID);
    upstreamSignal?.removeEventListener("abort", abortFromCaller);
  }
}

export async function fetchAdminAuthConfig(signal?: AbortSignal): Promise<AdminAuthConfig> {
  const response = await fetch("/v1/auth/config", { credentials: "same-origin", signal });
  if (!response.ok) throw new ApiError(response.status);
  return (await response.json()) as AdminAuthConfig;
}

export async function fetchAdminSession(signal?: AbortSignal): Promise<AdminSession | undefined> {
  const response = await fetch("/v1/auth/session", { credentials: "same-origin", signal });
  if (response.status === 401) {
    setAdminSession(undefined);
    return undefined;
  }
  if (!response.ok) throw new ApiError(response.status);
  const session = (await response.json()) as AdminSession;
  setAdminSession(session);
  return session;
}

export async function logoutAdminSession(): Promise<void> {
  const session = getAdminSession();
  if (!session) return;
  const response = await fetch("/v1/auth/logout", {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": session.csrf_token },
  });
  if (!response.ok && response.status !== 401) throw new ApiError(response.status);
  setAdminSession(undefined);
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
  scopes: string[];
  rpm_limit?: number;
  expires_at?: string;
  last_used_at?: string;
  revoked_at?: string;
  legacy_unbounded: boolean;
  status: string;
  created_at?: string;
  updated_at?: string;
}

export interface CreateAPIClientInput {
  name: string;
  scopes: Array<"tenant:identify" | "resource:resolve" | "events:read">;
  rpm_limit: number;
  expires_at: string;
}

export interface WebhookTarget {
  id: string; name: string; format: "generic" | "slack" | "discord" | "teams";
  status: "active" | "disabled"; endpoint: string; consecutive_failures: number;
  circuit_open_until?: string; created_at: string; updated_at: string;
}
export interface CreatedWebhookTarget extends WebhookTarget { signing_secret: string }
export interface WebhookDelivery {
  id: string; event_id: string; event_type: string; target_id: string; target_name: string;
  status: "pending" | "delivering" | "retry" | "delivered" | "dead_letter";
  attempt_count: number; next_attempt_at: string; last_http_status?: number;
  last_error_code?: string; delivered_at?: string; created_at: string;
}
export interface WebhookOverview { targets:number;pending:number;retry:number;delivered:number;dead_letter:number;open_circuits:number }

export interface OperationalComponent {
  name: string;
  status: "healthy" | "degraded" | "unavailable";
  latency_ms: number;
}
export interface OperationalReport {
  id: string;
  kind: "backup" | "restore" | "rewrap" | "migration";
  source: string;
  status: "healthy" | "degraded" | "failed";
  effective_status: "healthy" | "degraded" | "failed" | "stale";
  occurred_at: string;
  fresh_until: string;
  received_at: string;
  credential_version: string;
}
export interface OperationalHealth {
  status: "healthy" | "degraded" | "unavailable";
  checked_at: string;
  components: OperationalComponent[];
  reports: OperationalReport[];
  queues: {
    webhook_pending: number;
    webhook_retry: number;
    webhook_dead_letter: number;
    open_circuits: number;
  };
}

export interface APIClientUsageRecord {
  day: string;
  api_client_id: string;
  operation: "identify" | "resolve";
  status_class: number;
  request_count: number;
  rate_limited_count: number;
  client_name?: string;
}

export interface AdminAuditEvent {
  occurred_at: string;
  id: string;
  request_id: string;
  actor_kind: string;
  actor_subject: string;
  actor_label?: string;
  action: string;
  target_type: string;
  target_id: string;
  result: "success" | "denied" | "error";
  http_method: string;
  route_template: string;
  http_status: number;
  error_code?: string;
  metadata: Record<string, unknown>;
}

export interface AdminAuditPage {
  events: AdminAuditEvent[];
  next_cursor?: string;
}

export interface AdminSettingDefinition {
  key: string;
  type: "integer" | "enum";
  default_value: string;
  owner: string;
  minimum?: number;
  maximum?: number;
  options?: string[];
}

export interface AdminSettingsSnapshot {
  revision: number;
  values: Record<string, string>;
  definitions: AdminSettingDefinition[];
}

export interface AdminSessionView {
  id: string;
  issuer: string;
  subject: string;
  label: string;
  roles: string[];
  created_at: string;
  last_used_at: string;
  expires_at: string;
  idle_expires_at: string;
  revoked_at?: string;
  status: "active" | "revoked" | "expired" | "idle_expired";
  current: boolean;
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
  overview: (signal?: AbortSignal) => req<Overview>("/overview", { signal }),

  // tenants
  listTenants: (signal?: AbortSignal) => req<Tenant[]>("/tenants", { signal }),
  getTenant: (id: string, signal?: AbortSignal) => req<Tenant>(`/tenants/${id}`, { signal }),
  createTenant: (body: { slug: string; name: string }) =>
    req<Tenant>("/tenants", { method: "POST", body: JSON.stringify(body) }),
  updateTenant: (id: string, body: { name: string; slug: string; status: string }) =>
    req<Tenant>(`/tenants/${id}`, { method: "PUT", body: JSON.stringify(body) }),
  deleteTenant: (id: string) => req<void>(`/tenants/${id}`, { method: "DELETE" }),
  listDomains: (id: string, signal?: AbortSignal) =>
    req<TenantDomain[]>(`/tenants/${id}/domains`, { signal }),
  addDomain: (id: string, hostname: string) =>
    req(`/tenants/${id}/domains`, { method: "POST", body: JSON.stringify({ hostname }) }),
  removeDomain: (id: string, domainId: string) =>
    req(`/tenants/${id}/domains/${domainId}`, { method: "DELETE" }),
  listTenantResources: (id: string, reveal = false, signal?: AbortSignal) =>
    req<TenantResource[]>(
      `/tenants/${id}/resources${reveal ? "?reveal=true" : ""}`,
      { ...(reveal ? { cache: "no-store" as const } : {}), signal },
    ),
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
  listDefinitions: (signal?: AbortSignal) =>
    req<Definition[]>("/resource-definitions", { signal }),
  getDefinition: (id: string, signal?: AbortSignal) =>
    req<DefinitionDetail>(`/resource-definitions/${id}`, { signal }),
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
  listAPIClients: (signal?: AbortSignal) => req<ApiClient[]>("/api-clients", { signal }),
  createAPIClient: (body: CreateAPIClientInput) =>
    req<{ client: ApiClient; token: string }>("/api-clients", {
      method: "POST",
      body: JSON.stringify(body),
    }),
  updateAPIClient: (id: string, body: CreateAPIClientInput) =>
    req<ApiClient>(`/api-clients/${id}`, { method: "PATCH", body: JSON.stringify(body) }),
  rotateAPIClient: (id: string, graceSeconds: number) =>
    req<{ client: ApiClient; token: string }>(`/api-clients/${id}/rotate`, {
      method: "POST",
      body: JSON.stringify({ grace_seconds: graceSeconds }),
    }),
  revokeAPIClient: (id: string) =>
    req<ApiClient>(`/api-clients/${id}/revoke`, { method: "POST" }),
  deleteAPIClient: (id: string) => req<void>(`/api-clients/${id}`, { method: "DELETE" }),
  getAPIClientUsage: (id: string, from?: string, to?: string) => {
    const query = new URLSearchParams();
    if (from) query.set("from", from);
    if (to) query.set("to", to);
    return req<APIClientUsageRecord[]>(`/api-clients/${id}/usage?${query}`);
  },
  listAPIClientUsage: (from?: string, to?: string, signal?: AbortSignal) => {
    const query = new URLSearchParams();
    if (from) query.set("from", from);
    if (to) query.set("to", to);
    return req<APIClientUsageRecord[]>(`/api-client-usage?${query}`, { signal });
  },
  listAuditEvents: (query: URLSearchParams, signal?: AbortSignal) =>
    req<AdminAuditPage>(`/audit-events?${query}`, { signal }),

  getSettings: (signal?: AbortSignal) => req<AdminSettingsSnapshot>("/settings", { signal }),
  updateSettings: (values: Record<string, string>, revision: number) =>
    req<AdminSettingsSnapshot>("/settings", {
      method: "PATCH",
      headers: { "If-Match": `"settings-${revision}"` },
      body: JSON.stringify({ values }),
    }),
  listAdminSessions: (signal?: AbortSignal) => req<AdminSessionView[]>("/sessions", { signal }),
  revokeAdminSession: (id: string) => req<void>(`/sessions/${id}`, { method: "DELETE" }),
  revokeAdminPrincipalSessions: (issuer: string, subject: string) =>
    req<{ revoked: number }>("/sessions/revoke-principal", {
      method: "POST",
      body: JSON.stringify({ issuer, subject }),
    }),
  listWebhookTargets: (signal?: AbortSignal) => req<WebhookTarget[]>("/webhook-targets", { signal }),
  createWebhookTarget: (body: { name: string; url: string; format: WebhookTarget["format"] }) =>
    req<CreatedWebhookTarget>("/webhook-targets", { method: "POST", cache: "no-store", body: JSON.stringify(body) }),
  setWebhookTargetStatus: (id: string, status: WebhookTarget["status"]) =>
    req<void>(`/webhook-targets/${id}/status`, { method: "PUT", body: JSON.stringify({ status }) }),
  listWebhookDeliveries: (status = "", signal?: AbortSignal) =>
    req<WebhookDelivery[]>(`/webhook-deliveries${status ? `?status=${status}` : ""}`, { signal }),
  replayWebhookDelivery: (id: string) => req<void>(`/webhook-deliveries/${id}/replay`, { method: "POST" }),
  getWebhookOverview: (signal?: AbortSignal) => req<WebhookOverview>("/webhook-overview", { signal }),
  getOperationalHealth: (signal?: AbortSignal) => req<OperationalHealth>("/operational-health", { signal, cache: "no-store" }),
};
