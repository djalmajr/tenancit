import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  ApiTimeoutError,
  REQUEST_TIMEOUT_MS,
  api,
  consumePendingAdminAuthMessage,
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
    await api.createTenant({ slug: "acme", name: "Acme" });
    const [url, init] = spy.mock.calls[0];
    expect(url).toBe("/v1/admin/tenants");
    expect(init?.method).toBe("POST");
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

  it("setResourceStatus PUTs to the status sub-resource", async () => {
    const spy = mockFetch(() => new Response(null, { status: 204 }));
    await api.setResourceStatus("t1", "r1", "inactive");
    const [url, init] = spy.mock.calls[0];
    expect(url).toBe("/v1/admin/tenants/t1/resources/r1/status");
    expect(init?.method).toBe("PUT");
    if (typeof init?.body !== "string") throw new TypeError("expected a serialized request body");
    expect(JSON.parse(init.body)).toEqual({ status: "inactive" });
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
