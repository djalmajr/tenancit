import { expect, test } from "@playwright/test";
import { adminRequest, adminToken, authenticate, uniqueID } from "./fixtures/admin";
import { type APIClientRecord, CatalogCleanup } from "./fixtures/catalog";
import { flowStep } from "./support/flow-step";

test.use({ screenshot: "off", trace: "off" });

test("operational governance catalog", { tag: "@full" }, async ({ page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();
  const headers = { Authorization: `Bearer ${adminToken}` };
  const governedName = uniqueID("governed");
  const rateName = uniqueID("rate");
  let governed!: { client: APIClientRecord; token: string };
  let rateClient!: { client: APIClientRecord; token: string };
  let limitedResponse!: Awaited<ReturnType<typeof request.get>>;

  try {
    await flowStep("api-client-governance", 1, "cria política completa e token one-shot", async () => {
      governed = await adminRequest(request, "post", "/v1/admin/api-clients", {
        name: governedName, scopes: ["tenant:identify"], rpm_limit: 60,
        expires_at: new Date(Date.now() + 30 * 86_400_000).toISOString(),
      });
      cleanup.trackAPIClient(governed.client.id);
      expect(governed.token.startsWith("tnc_")).toBe(true);
    });
    await flowStep("api-client-governance", 2, "edita política governada", async () => {
      const response = await request.patch(`/v1/admin/api-clients/${governed.client.id}`, {
        headers, data: { name: `${governedName}-edited`, scopes: ["tenant:identify", "resource:resolve"], rpm_limit: 120, expires_at: new Date(Date.now() + 90 * 86_400_000).toISOString() },
      });
      expect(response.ok()).toBe(true);
      expect((await response.json() as APIClientRecord).name).toBe(`${governedName}-edited`);
    });
    await flowStep("api-client-governance", 3, "rotaciona sucessor one-shot", async () => {
      const rotated = await adminRequest<{ client: APIClientRecord; token: string }>(request, "post", `/v1/admin/api-clients/${governed.client.id}/rotate`, { grace_period_seconds: 0 });
      expect(rotated.token.startsWith("tnc_")).toBe(true);
      expect(rotated.token === governed.token).toBe(false);
    });
    await flowStep("api-client-governance", 4, "revoga terminalmente", async () => {
      await adminRequest(request, "post", `/v1/admin/api-clients/${governed.client.id}/revoke`);
      const denied = await request.get("/v1/identify?hostname=terminal.e2e.local", { headers: { Authorization: `Bearer ${governed.token}` } });
      expect(denied.status()).toBe(401);
    });
    await flowStep("api-client-governance", 5, "remove somente após revogar", async () => {
      const deleted = await request.delete(`/v1/admin/api-clients/${governed.client.id}`, { headers });
      expect(deleted.status()).toBe(204);
    });

    await flowStep("api-client-rate-limit", 1, "esgota bucket global", async () => {
      rateClient = await adminRequest(request, "post", "/v1/admin/api-clients", {
        name: rateName, scopes: ["tenant:identify"], rpm_limit: 1,
        expires_at: new Date(Date.now() + 30 * 86_400_000).toISOString(),
      });
      cleanup.trackAPIClient(rateClient.client.id);
      const first = await request.get("/v1/identify?hostname=rate-limit.e2e.local", { headers: { Authorization: `Bearer ${rateClient.token}` } });
      expect(first.status()).toBe(404);
      limitedResponse = await request.get("/v1/identify?hostname=rate-limit.e2e.local", { headers: { Authorization: `Bearer ${rateClient.token}` } });
      expect(limitedResponse.status()).toBe(429);
    });
    await flowStep("api-client-rate-limit", 2, "expõe contrato de recuperação", async () => {
      expect(await limitedResponse.json()).toMatchObject({ error: "rate_limited" });
      expect(limitedResponse.headers()["retry-after"]).toBeTruthy();
      expect(limitedResponse.headers()["ratelimit-limit"]).toBe("1");
    });
    await flowStep("api-client-rate-limit", 3, "torna limitação observável", async () => {
      await expect.poll(async () => {
        const rows = await adminRequest<Array<{ api_client_id: string; rate_limited_count: number }>>(request, "get", `/v1/admin/api-client-usage?api_client_id=${rateClient.client.id}`);
        return rows.some((row) => row.api_client_id === rateClient.client.id && row.rate_limited_count > 0);
      }, { timeout: 20_000 }).toBe(true);
    });
    await flowStep("api-client-rate-limit", 4, "mantém limiter externo saudável", async () => {
      const health = await request.get("/healthz");
      expect(health.ok()).toBe(true);
    });

    await flowStep("api-client-usage-audit", 1, "abre dashboard de uso", async () => {
      await authenticate(page);
      await page.goto("/usage");
      await expect(page.getByRole("heading", { name: "Uso das chaves de API" })).toBeVisible();
      await page.getByRole("combobox", { name: "Chave", exact: true }).click();
      await expect(page.getByText(rateName, { exact: true })).toBeVisible();
      await page.keyboard.press("Escape");
    });
    await flowStep("api-client-usage-audit", 2, "filtra por chave e operação", async () => {
      await page.getByRole("combobox", { name: "Chave", exact: true }).click();
      await page.getByText(rateName, { exact: true }).click();
      await page.getByRole("combobox", { name: "Operação" }).click();
      await page.getByRole("option", { name: "Identificação", exact: true }).click();
      await expect(page.getByText("Limitadas", { exact: true }).first()).toBeVisible();
    });
    await flowStep("api-client-usage-audit", 3, "abre auditoria com ator técnico", async () => {
      await page.goto("/audit-events");
      await expect(page.getByRole("heading", { name: "Auditoria administrativa" })).toBeVisible();
      await expect(page.getByText(/token compartilhado/i)).toBeVisible();
    });
    await flowStep("api-client-usage-audit", 4, "localiza evento de lifecycle", async () => {
      await expect(page.getByText("api_client.created").first()).toBeVisible();
    });
  } finally {
    await cleanup.run(request);
  }
});
