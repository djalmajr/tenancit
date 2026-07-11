import { expect, type APIRequestContext, type Page } from "@playwright/test";

export const adminToken = process.env.TENANCIT_E2E_ADMIN_TOKEN ?? "";

export function uniqueID(prefix: string) {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

export async function authenticate(page: Page) {
  if (!adminToken) throw new Error("TENANCIT_E2E_ADMIN_TOKEN is required");
  await page.addInitScript(
    ({ key, token }) => window.localStorage.setItem(key, token),
    { key: "tenancitAdminToken", token: adminToken },
  );
}

export async function adminRequest<T>(
  request: APIRequestContext,
  method: "get" | "post" | "put" | "delete",
  path: string,
  data?: unknown,
) {
  if (!adminToken) throw new Error("TENANCIT_E2E_ADMIN_TOKEN is required");
  const response = await request[method](path, {
    data,
    headers: {
      Authorization: `Bearer ${adminToken}`,
      ...(method === "post" ? { "Idempotency-Key": crypto.randomUUID() } : {}),
    },
  });
  expect(response.ok(), `${method.toUpperCase()} ${path}: ${response.status()}`).toBeTruthy();
  if (response.status() === 204) return undefined as T;
  return (await response.json()) as T;
}
