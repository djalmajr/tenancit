import { createHmac } from "node:crypto";
import { expect, test } from "@playwright/test";
import { adminRequest, authenticate, uniqueID } from "./fixtures/admin";

function receiverBaseURLFromEnvironment() {
  const value = process.env.TENANCIT_E2E_WEBHOOK_RECEIVER_BASE_URL;
  if (!value) throw new Error("TENANCIT_E2E_WEBHOOK_RECEIVER_BASE_URL is required");
  return value;
}

test("webhook target receives a signed domain event and exposes delivery status", async ({ page, request }) => {
  const receiverBaseURL = receiverBaseURLFromEnvironment();
  await request.delete(`${receiverBaseURL}/events`);
  await authenticate(page);
  await page.goto("/integrations/webhooks");
  await page.getByRole("button", { name: "Novo webhook" }).click();
  await page.getByRole("textbox", { name: "Nome", exact: true }).fill(uniqueID("receiver"));
  await page.getByRole("textbox", { name: "URL do receiver", exact: true }).fill("http://127.0.0.1:9090/hook");
  await page.getByRole("button", { name: "Salvar" }).click();
  const secretInput = page.getByRole("textbox", { name: "Signing secret", exact: true });
  await expect(secretInput).not.toHaveValue("");
  const secret = await secretInput.evaluate((element: HTMLInputElement) => element.value);
  await page.getByRole("button", { name: "Concluído" }).click();

  const slug = uniqueID("webhook-tenant");
  const tenant = await adminRequest<{ id: string }>(request, "post", "/v1/admin/tenants", {
    slug,
    name: `Webhook ${slug}`,
  });
  try {
    await expect.poll(async () => {
      const response = await request.get(`${receiverBaseURL}/events`);
      return await response.json() as Array<{ body: string; headers: Record<string, string> }>;
    }, { timeout: 15_000 }).toHaveLength(1);
    const response = await request.get(`${receiverBaseURL}/events`);
    const [event] = await response.json() as Array<{ body: string; headers: Record<string, string> }>;
    expect(JSON.parse(event.body).type).toBe("tenancit.tenant.created");
    const timestamp = event.headers["tenancit-webhook-timestamp"];
    const expected = `v1=${createHmac("sha256", secret).update(`${timestamp}.${event.body}`).digest("hex")}`;
    expect(event.headers["tenancit-webhook-signature"]).toBe(expected);

    await page.reload();
    await expect(page.getByText("delivered", { exact: true })).toBeVisible();
  } finally {
    await adminRequest(request, "delete", `/v1/admin/tenants/${tenant.id}`);
  }
});
