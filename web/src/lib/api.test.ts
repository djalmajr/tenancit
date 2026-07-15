import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  ApiTimeoutError,
  REQUEST_TIMEOUT_MS,
  api,
  consumePendingAdminAuthMessage,
  fetchAdminAuthConfig,
  fetchAdminSession,
  logoutAdminSession,
  setAdminSession,
  setAdminToken,
} from "./api";

// Contract tests for the admin API client: correct method/path/body, and that
// a non-ok response surfaces an error instead of silently resolving.

const originalFetch = globalThis.fetch;

function mockFetch(impl: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  const spy = vi.fn(impl);
  globalThis.fetch = spy as unknown as typeof fetch;
  return spy;
}

beforeEach(() => {
  localStorage.clear();
  setAdminSession(undefined);
  consumePendingAdminAuthMessage();
  vi.restoreAllMocks();
});
afterEach(() => {
  globalThis.fetch = originalFetch;
});

describe("api client", () => {
  // Mutation captured: changing the verb to GET or the path prefix would make
  // these assertions fail — locks the request contract.
  it("createTenant POSTs to /v1/admin/tenants with the body", async () => {
    const spy = mockFetch(() => new Response(JSON.stringify({ id: "1" }), { status: 201 }));
    await api.createTenant({ slug: "acme", name: "Acme" }, "00000000-0000-4000-8000-000000000001");
    const [url, init] = spy.mock.calls[0];
    expect(url).toBe("/v1/admin/tenants");
    expect(init?.method).toBe("POST");
    expect(new Headers(init?.headers).get("Idempotency-Key")).toBe("00000000-0000-4000-8000-000000000001");
    if (typeof init?.body !== "string") throw new TypeError("expected a serialized request body");
    expect(JSON.parse(init.body)).toEqual({ slug: "acme", name: "Acme" });
  });

  it("sends the configured admin token", async () => {
    setAdminToken("tenancit_admin_dev");
    const spy = mockFetch(() => new Response(JSON.stringify({ id: "1" }), { status: 201 }));
    await api.createTenant({ slug: "acme", name: "Acme" });
    const [, init] = spy.mock.calls[0];
    expect(new Headers(init?.headers).get("Authorization")).toBe("Bearer tenancit_admin_dev");
  });

  it("discovers OIDC auth without sending a legacy credential", async () => {
    setAdminToken("legacy-token-that-must-not-leak");
    const spy = mockFetch(() =>
      new Response(JSON.stringify({ mode: "oidc", login_url: "/v1/auth/login" }), { status: 200 }),
    );

    await expect(fetchAdminAuthConfig()).resolves.toEqual({
      mode: "oidc",
      login_url: "/v1/auth/login",
    });

    const [url, init] = spy.mock.calls[0];
    expect(url).toBe("/v1/auth/config");
    expect(new Headers(init?.headers).get("Authorization")).toBeNull();
    expect(init?.credentials).toBe("same-origin");
  });

  it("uses the in-memory CSRF token for cookie-authenticated mutations", async () => {
    setAdminSession({
      kind: "oidc_user",
      issuer: "https://id.example.test",
      subject: "user-1",
      label: "Ada",
      session_id: "session-1",
      roles: ["operator"],
      permissions: ["admin.read", "tenant.write"],
      csrf_token: "csrf-token",
      expires_at: "2026-07-11T20:00:00Z",
      idle_expires_at: "2026-07-11T12:30:00Z",
    });
    const spy = mockFetch(() => new Response(JSON.stringify({ id: "1" }), { status: 201 }));

    await api.createTenant({ slug: "acme", name: "Acme" });

    const [, init] = spy.mock.calls[0];
    const headers = new Headers(init?.headers);
    expect(headers.get("X-CSRF-Token")).toBe("csrf-token");
    expect(headers.get("Authorization")).toBeNull();
    expect(init?.credentials).toBe("same-origin");
  });

  it("loads and logs out a server-side session", async () => {
    const session = {
      kind: "oidc_user" as const,
      issuer: "https://id.example.test",
      subject: "user-1",
      label: "Ada",
      session_id: "session-1",
      roles: ["operator"],
      permissions: ["admin.read", "tenant.write"],
      csrf_token: "csrf-token",
      expires_at: "2026-07-11T20:00:00Z",
      idle_expires_at: "2026-07-11T12:30:00Z",
    };
    const spy = mockFetch((url) =>
      url === "/v1/auth/session"
        ? new Response(JSON.stringify(session), { status: 200 })
        : new Response(null, { status: 204 }),
    );

    await expect(fetchAdminSession()).resolves.toEqual(session);
    await logoutAdminSession();

    expect(spy.mock.calls[1][0]).toBe("/v1/auth/logout");
    expect(spy.mock.calls[1][1]?.method).toBe("POST");
    expect(new Headers(spy.mock.calls[1][1]?.headers).get("X-CSRF-Token")).toBe("csrf-token");
  });

  it("setResourceStatus PUTs to the status sub-resource", async () => {
    const spy = mockFetch(() => new Response(null, { status: 204 }));
    await api.setResourceStatus("t1", "r1", "inactive");
    const [url, init] = spy.mock.calls[0];
    expect(url).toBe("/v1/admin/tenants/t1/resources/r1/status");
    expect(init?.method).toBe("PUT");
    if (typeof init?.body !== "string") throw new TypeError("expected a serialized request body");
    expect(JSON.parse(init.body)).toEqual({ status: "inactive" });
  });

  it("updateResourceField PUTs one value without exposing it in the URL", async () => {
    const spy = mockFetch(() => new Response(null, { status: 204 }));
    await api.updateResourceField("t1", "r1", "secret key", "s3cr3t");
    const [url, init] = spy.mock.calls[0];
    expect(url).toBe("/v1/admin/tenants/t1/resources/r1/fields/secret%20key");
    expect(url).not.toContain("s3cr3t");
    expect(init?.method).toBe("PUT");
    if (typeof init?.body !== "string") throw new TypeError("expected a serialized request body");
    expect(JSON.parse(init.body)).toEqual({ value: "s3cr3t" });
  });

  it("listTenantResources appends ?reveal=true only when asked", async () => {
    const spy = mockFetch(() => new Response("[]", { status: 200 }));
    await api.listTenantResources("t1", true);
    expect(spy.mock.calls[0][0]).toBe("/v1/admin/tenants/t1/resources?reveal=true");
    expect(spy.mock.calls[0][1]?.cache).toBe("no-store");
  });

  it("deleteTenant uses the hard-delete tenant endpoint", async () => {
    const spy = mockFetch(() => new Response(null, { status: 204 }));
    await api.deleteTenant("t1");
    const [url, init] = spy.mock.calls[0];
    expect(url).toBe("/v1/admin/tenants/t1");
    expect(init?.method).toBe("DELETE");
  });

  it("updates settings with an optimistic concurrency precondition", async () => {
    const spy = mockFetch(() => new Response(JSON.stringify({ revision: 4, values: {}, definitions: [] }), { status: 200 }));
    await api.updateSettings({ session_idle_minutes: "45" }, 3);
    const [url, init] = spy.mock.calls[0];
    expect(url).toBe("/v1/admin/settings");
    expect(init?.method).toBe("PATCH");
    expect(new Headers(init?.headers).get("If-Match")).toBe('"settings-3"');
  });

  it("creates webhook signing secrets through a no-store one-shot request", async () => {
    const spy = mockFetch(() => new Response(JSON.stringify({ id: "target-1", signing_secret: "once" }), { status: 201 }));
    await api.createWebhookTarget({ name: "receiver", url: "https://receiver.example/hook", format: "generic" });
    const [url, init] = spy.mock.calls[0];
    expect(url).toBe("/v1/admin/webhook-targets");
    expect(init?.method).toBe("POST");
    expect(init?.cache).toBe("no-store");
  });

  it("aborts stalled requests with a typed timeout error", async () => {
    vi.useFakeTimers();
    try {
      mockFetch((_url, init) =>
        new Promise<Response>((_resolve, reject) => {
          init?.signal?.addEventListener(
            "abort",
            () => reject(new DOMException("aborted", "AbortError")),
            { once: true },
          );
        }),
      );

      const request = api.listTenants();
      await Promise.all([
        vi.advanceTimersByTimeAsync(REQUEST_TIMEOUT_MS),
        expect(request).rejects.toBeInstanceOf(ApiTimeoutError),
      ]);
    } finally {
      vi.useRealTimers();
    }
  });

  it("forwards caller cancellation to every admin read", async () => {
    const calls: Array<(signal: AbortSignal) => Promise<unknown>> = [
      (signal) => api.overview(signal),
      (signal) => api.listTenants(signal),
      (signal) => api.getTenant("tenant-1", signal),
      (signal) => api.listDomains("tenant-1", signal),
      (signal) => api.listTenantResources("tenant-1", false, signal),
      (signal) => api.listDefinitions(signal),
      (signal) => api.getDefinition("definition-1", signal),
      (signal) => api.listAPIClients(signal),
      (signal) => api.getSettings(signal),
      (signal) => api.listAdminSessions(signal),
      (signal) => api.listWebhookTargets(signal),
      (signal) => api.listWebhookDeliveries("", signal),
      (signal) => api.getOperationalHealth(signal),
    ];

    for (const read of calls) {
      let fetchSignal: AbortSignal | undefined;
      mockFetch((_url, init) => {
        fetchSignal = init?.signal ?? undefined;
        return new Promise<Response>((_resolve, reject) => {
          init?.signal?.addEventListener(
            "abort",
            () => reject(new DOMException("aborted", "AbortError")),
            { once: true },
          );
        });
      });
      const controller = new AbortController();
      const request = read(controller.signal);

      controller.abort("route changed");

      await expect(request).rejects.toMatchObject({ name: "AbortError" });
      expect(fetchSignal?.aborted).toBe(true);
    }
  });

  // Mutation captured: dropping the `if (!res.ok) throw` guard would resolve
  // with garbage instead of throwing → this expectation fails.
  it("throws on a non-ok response", async () => {
    mockFetch(() => new Response("boom", { status: 500 }));
    await expect(api.listTenants()).rejects.toThrow(/500/);
  });

  it("dispatches admin auth required on 401", async () => {
    const listener = vi.fn();
    window.addEventListener("admin-auth-required", listener);
    mockFetch(() => new Response("missing bearer token", { status: 401 }));

    // A 401 rejects with an ApiError carrying the status; the human-readable
    // message is derived at display time (apiErrorMessage), not baked in here.
    await expect(api.listTenants()).rejects.toMatchObject({ status: 401 });
    expect(listener).toHaveBeenCalledTimes(1);
    expect(listener.mock.calls[0][0]).toMatchObject({
      detail: { messageKey: "auth.invalidToken" },
    });

    window.removeEventListener("admin-auth-required", listener);
  });

  it("retains a 401 message until a late auth-boundary subscriber consumes it", async () => {
    mockFetch(() => new Response("invalid bearer token", { status: 401 }));

    await expect(api.listTenants()).rejects.toMatchObject({ status: 401 });
    expect(consumePendingAdminAuthMessage()).toBe("auth.invalidToken");
    expect(consumePendingAdminAuthMessage()).toBeUndefined();
  });
});
