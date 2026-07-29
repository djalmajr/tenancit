import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";
import { render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RUNTIME_BASE_PATH_META_NAME } from "@/lib/runtime-base-path";
import { createAppRouter } from "@/router";

const originalFetch = globalThis.fetch;

beforeEach(() => {
  const meta = document.createElement("meta");
  meta.content = "/tenancit";
  meta.name = RUNTIME_BASE_PATH_META_NAME;
  document.head.append(meta);
  globalThis.fetch = vi.fn((input: RequestInfo | URL) => {
    const url = typeof input === "string"
      ? input
      : input instanceof URL
        ? input.toString()
        : input.url;
    if (url === "/tenancit/v1/auth/config") {
      return Promise.resolve(
        new Response(
          JSON.stringify({ mode: "oidc", login_url: "/tenancit/v1/auth/login" }),
          { status: 200 },
        ),
      );
    }
    if (url === "/tenancit/v1/auth/session") {
      return Promise.resolve(new Response(null, { status: 401 }));
    }
    return Promise.reject(new Error(`unexpected request: ${url}`));
  });
});

afterEach(() => {
  document.head.replaceChildren();
  globalThis.fetch = originalFetch;
  vi.restoreAllMocks();
});

describe("OIDC access navigation", () => {
  // Mutation captured: selecting location.pathname would remove the public "/tenancit" prefix.
  it("returns to the public base-prefixed route after authentication", async () => {
    const history = createMemoryHistory({
      initialEntries: ["/tenancit/tenants?status=active"],
    });
    const appRouter = createAppRouter({ history });

    await appRouter.load();
    render(<RouterProvider router={appRouter} />);

    const loginLink = await screen.findByRole("link", { name: /SSO/ });
    const loginURL = new URL(loginLink.getAttribute("href") ?? "", window.location.origin);
    expect(loginURL.pathname).toBe("/tenancit/v1/auth/login");
    expect(loginURL.searchParams.get("return_to")).toBe("/tenancit/tenants?status=active");
  });
});
