import { expect, test } from "@playwright/test";
import { adminToken, uniqueID } from "./fixtures/admin";
import { CatalogCleanup, createDefinitionFixture } from "./fixtures/catalog";
import { flowStep } from "./support/flow-step";

test("critical admin retries produce one effect and the same one-shot token", { tag: "@pr-critical" }, async ({ request }) => {
  const cleanup = new CatalogCleanup();
  const headers = { Authorization: `Bearer ${adminToken}` };
  try {
    const tenantKey = crypto.randomUUID();
    const slug = uniqueID("retry-tenant");
    const tenantPayload = { slug, name: `E2E ${slug}` };
    let tenantId = "";
    await flowStep("admin-idempotent-retry", 1, "repete criação de tenant sem duplicar", async () => {
      const first = await request.post("/v1/admin/tenants", { data: tenantPayload, headers: { ...headers, "Idempotency-Key": tenantKey } });
      const retry = await request.post("/v1/admin/tenants", { data: tenantPayload, headers: { ...headers, "Idempotency-Key": tenantKey } });
      expect(first.status()).toBe(201);
      expect(retry.status()).toBe(201);
      expect(retry.headers()["idempotency-replayed"]).toBe("true");
      const firstBody = await first.json() as { id: string };
      const retryBody = await retry.json() as { id: string };
      expect(retryBody.id).toBe(firstBody.id);
      tenantId = firstBody.id;
      cleanup.trackTenant(tenantId);
    });
    await flowStep("admin-idempotent-retry", 2, "rejeita a mesma chave com payload diferente", async () => {
      const mismatch = await request.post("/v1/admin/tenants", {
        data: { ...tenantPayload, name: "changed" },
        headers: { ...headers, "Idempotency-Key": tenantKey },
      });
      expect(mismatch.status()).toBe(409);
      await expect(mismatch.json()).resolves.toMatchObject({ error: "idempotency_mismatch" });
    });
    let clientId = "";
    await flowStep("admin-idempotent-retry", 3, "recupera o mesmo token de create e rotate", async () => {
      const key = crypto.randomUUID();
      const body = {
        name: uniqueID("retry-client"), scopes: ["tenant:identify"], rpm_limit: 300,
        expires_at: new Date(Date.now() + 90 * 86400000).toISOString(),
      };
      const first = await request.post("/v1/admin/api-clients", { data: body, headers: { ...headers, "Idempotency-Key": key } });
      const retry = await request.post("/v1/admin/api-clients", { data: body, headers: { ...headers, "Idempotency-Key": key } });
      const created = await first.json() as { client: { id: string }; token: string };
      const replayed = await retry.json() as typeof created;
      expect(replayed.token === created.token).toBe(true);
      clientId = created.client.id;
      cleanup.trackAPIClient(clientId);
      const rotateKey = crypto.randomUUID();
      const rotated = await request.post(`/v1/admin/api-clients/${clientId}/rotate`, { data: { grace_seconds: 0 }, headers: { ...headers, "Idempotency-Key": rotateKey } });
      const rotateRetry = await request.post(`/v1/admin/api-clients/${clientId}/rotate`, { data: { grace_seconds: 0 }, headers: { ...headers, "Idempotency-Key": rotateKey } });
      const rotatedBody = await rotated.json() as { token: string };
      const rotateRetryBody = await rotateRetry.json() as { token: string };
      expect(rotateRetryBody.token === rotatedBody.token).toBe(true);
    });
    await flowStep("admin-idempotent-retry", 4, "repete provisionamento e preserva uma instância", async () => {
      const { definition } = await createDefinitionFixture(request, cleanup, { prefix: "retry-definition" });
      const key = crypto.randomUUID();
      const body = { definitionKey: definition.key, values: {} };
      const first = await request.post(`/v1/admin/tenants/${tenantId}/resources`, { data: body, headers: { ...headers, "Idempotency-Key": key } });
      const retry = await request.post(`/v1/admin/tenants/${tenantId}/resources`, { data: body, headers: { ...headers, "Idempotency-Key": key } });
      expect(first.status()).toBe(201);
      expect(retry.headers()["idempotency-replayed"]).toBe("true");
      expect((await retry.json() as { id: string }).id).toBe((await first.json() as { id: string }).id);
    });
  } finally {
    await cleanup.run(request);
  }
});
