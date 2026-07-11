import { expect, test } from "@playwright/test";
import { authenticate, uniqueID } from "./fixtures/admin";

test("operational reporter publishes fresh evidence with its own credential", async ({ page, request }) => {
  const reporterToken = process.env.TENANCIT_E2E_OPERATIONS_TOKEN ?? "";
  const source = uniqueID("backup-agent");
  const response = await request.post("/v1/operations/reports", {
    headers: {
      Authorization: `Bearer ${reporterToken}`,
      "Idempotency-Key": uniqueID("backup"),
    },
    data: {
      kind: "backup",
      source,
      status: "healthy",
      occurred_at: new Date().toISOString(),
      fresh_for_seconds: 3600,
    },
  });
  expect(response.status()).toBe(201);

  await authenticate(page);
  await page.goto("/operations/health");
  await expect(page.getByRole("heading", { name: "Saúde operacional", exact: true })).toBeVisible();
  await expect(page.getByText(source, { exact: true })).toBeVisible();
  await expect(page.getByText("saudável", { exact: true }).last()).toBeVisible();
});
