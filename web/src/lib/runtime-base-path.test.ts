import { afterEach, describe, expect, it } from "vitest";
import {
  buildAdminEndpoint,
  buildOIDCLoginURL,
  readRuntimeBasePath,
  RUNTIME_BASE_PATH_META_NAME,
} from "./runtime-base-path";

afterEach(() => {
  document.head.replaceChildren();
});

describe("runtime base path", () => {
  // Mutation captured during Red: ignoring the injected meta kept the router at "/".
  it("reads the server-injected base path from document metadata", () => {
    const meta = document.createElement("meta");
    meta.content = "/tenancit";
    meta.name = RUNTIME_BASE_PATH_META_NAME;
    document.head.append(meta);

    expect(readRuntimeBasePath()).toBe("/tenancit");
  });

  // Mutation captured: requiring deployment metadata would break root-hosted development.
  it("uses the root path when metadata is absent", () => {
    expect(readRuntimeBasePath()).toBe("/");
  });

  it.each([
    ["/v1/auth/config", "/tenancit/v1/auth/config"],
    ["/v1/admin/tenants?status=active", "/tenancit/v1/admin/tenants?status=active"],
  ])("prefixes the human endpoint %s without changing its query", (endpoint, expected) => {
    const meta = document.createElement("meta");
    meta.content = "/tenancit";
    meta.name = RUNTIME_BASE_PATH_META_NAME;
    document.head.append(meta);

    expect(buildAdminEndpoint(endpoint)).toBe(expected);
  });

  it("keeps human endpoints at root when runtime metadata is absent", () => {
    expect(buildAdminEndpoint("/v1/admin/tenants")).toBe("/v1/admin/tenants");
    expect(buildAdminEndpoint("/v1/auth/session")).toBe("/v1/auth/session");
  });

  it("refuses to prefix consumer endpoints", () => {
    expect(() => buildAdminEndpoint("/v1/resolve?hostname=tenant.example")).toThrow(
      "admin or auth endpoint",
    );
  });

  // Mutation captured: accepting an origin, query, or fragment turns configuration into URL input.
  it.each(["tenancit", "//other.example/tenancit", "/tenancit?mode=admin", "/tenancit#admin"])(
    "rejects unsafe metadata value %s",
    (basePath) => {
      const meta = document.createElement("meta");
      meta.content = basePath;
      meta.name = RUNTIME_BASE_PATH_META_NAME;
      document.head.append(meta);

      expect(() => readRuntimeBasePath()).toThrow("invalid runtime base path");
    },
  );
});

describe("OIDC login navigation", () => {
  // Mutation captured during Red: using the internal pathname dropped "/tenancit" after login.
  it("preserves the server-prefixed login endpoint and public routed URL", () => {
    const loginURL = buildOIDCLoginURL({
      loginURL: "/tenancit/v1/auth/login",
      origin: "https://admin-labdev.cloud4biz.com",
      returnTo: "/tenancit/tenants?status=active",
    });

    expect(loginURL).toBe(
      "https://admin-labdev.cloud4biz.com/tenancit/v1/auth/login?return_to=%2Ftenancit%2Ftenants%3Fstatus%3Dactive",
    );
  });

  it("rejects a login endpoint on another origin", () => {
    expect(() =>
      buildOIDCLoginURL({
        loginURL: "https://attacker.example/v1/auth/login",
        origin: "https://admin-labdev.cloud4biz.com",
        returnTo: "/tenancit/tenants",
      }),
    ).toThrow("same-origin");
  });
});
