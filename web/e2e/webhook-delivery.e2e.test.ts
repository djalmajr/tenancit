import { createHmac } from "node:crypto";
import { expect, test, type APIRequestContext } from "@playwright/test";
import { adminRequest, authenticate, uniqueID } from "./fixtures/admin";

interface CreatedWebhookTarget {
  id: string;
  name: string;
  signing_secret: string;
}

interface ReceiverEvent {
  body: string;
  headers: Record<string, string>;
}

interface WebhookDelivery {
  event_type: string;
  status: "pending" | "delivering" | "retry" | "delivered" | "dead_letter";
  target_id: string;
  target_name: string;
}

interface WebhookTarget {
  endpoint: string;
  id: string;
  name: string;
  status: "active" | "disabled";
}

const RECEIVER_ORIGIN = "http://127.0.0.1:9090";
const RECEIVER_URL = `${RECEIVER_ORIGIN}/hook`;
const TARGET_PREFIX = "e2e-webhook-receiver";

async function cleanupWebhookFixtures(
  request: APIRequestContext,
  { targetID, tenantID }: { targetID: string; tenantID: string },
) {
  if (targetID) {
    await adminRequest(request, "put", `/v1/admin/webhook-targets/${targetID}/status`, { status: "disabled" });
  }
  if (tenantID) {
    await adminRequest(request, "delete", `/v1/admin/tenants/${tenantID}`);
  }
}

async function disableStaleReceiverTargets(request: APIRequestContext) {
  const targets = await adminRequest<WebhookTarget[]>(request, "get", "/v1/admin/webhook-targets");
  for (const target of targets) {
    if (target.status === "active" && target.endpoint === RECEIVER_ORIGIN && target.name.startsWith(`${TARGET_PREFIX}-`)) {
      await adminRequest(request, "put", `/v1/admin/webhook-targets/${target.id}/status`, { status: "disabled" });
    }
  }
}

function receiverBaseURLFromEnvironment() {
  const value = process.env.TENANCIT_E2E_WEBHOOK_RECEIVER_BASE_URL;
  if (!value) throw new Error("TENANCIT_E2E_WEBHOOK_RECEIVER_BASE_URL is required");
  return value;
}

test("webhook target receives a signed domain event and exposes delivery status", async ({ page, request }) => {
  const receiverBaseURL = receiverBaseURLFromEnvironment();
  await disableStaleReceiverTargets(request);
  expect((await request.delete(`${receiverBaseURL}/events`)).ok()).toBe(true);

  const targetName = uniqueID(TARGET_PREFIX);
  let targetID = "";
  let tenantID = "";
  try {
    await authenticate(page);
    await page.goto("/integrations/webhooks");
    await page.getByRole("button", { name: "Novo webhook" }).click();
    await page.getByRole("textbox", { name: "Nome", exact: true }).fill(targetName);
    await page.getByRole("textbox", { name: "URL do receiver", exact: true }).fill(RECEIVER_URL);
    const createResponsePromise = page.waitForResponse((response) =>
      response.request().method() === "POST"
      && new URL(response.url()).pathname === "/v1/admin/webhook-targets");
    await page.getByRole("button", { name: "Salvar" }).click();
    const createResponse = await createResponsePromise;
    expect(createResponse.status()).toBe(201);
    const createdTarget = await createResponse.json() as CreatedWebhookTarget;
    targetID = createdTarget.id;

    const secretInput = page.getByRole("textbox", { name: "Signing secret", exact: true });
    await expect(secretInput).toHaveValue(createdTarget.signing_secret);
    const secret = createdTarget.signing_secret;
    await page.getByRole("button", { name: "Concluído" }).click();
    await expect(secretInput).toHaveCount(0);

    const slug = uniqueID("webhook-tenant");
    const tenant = await adminRequest<{ id: string }>(request, "post", "/v1/admin/tenants", {
      slug,
      name: `Webhook ${slug}`,
    });
    tenantID = tenant.id;

    // The terminal state proves the worker finished before receiver evidence is counted.
    // Returning the whole status array also catches duplicate delivery rows.
    await expect.poll(async () => {
      const deliveries = await adminRequest<WebhookDelivery[]>(
        request,
        "get",
        "/v1/admin/webhook-deliveries?limit=200",
      );
      return deliveries
        .filter((delivery) =>
          delivery.target_id === targetID
          && delivery.event_type === "tenancit.tenant.created")
        .map((delivery) => delivery.status)
        .sort();
    }, { timeout: 15_000 }).toEqual(["delivered"]);

    const receiverResponse = await request.get(`${receiverBaseURL}/events`);
    expect(receiverResponse.ok()).toBe(true);
    const receivedEvents = await receiverResponse.json() as ReceiverEvent[];
    const matchingEvents = receivedEvents.filter((candidate) => {
      const body = JSON.parse(candidate.body) as { aggregate_id?: string; type?: string };
      return body.aggregate_id === tenant.id && body.type === "tenancit.tenant.created";
    });
    expect(matchingEvents).toHaveLength(1);
    const [event] = matchingEvents;
    expect(JSON.parse(event.body).type).toBe("tenancit.tenant.created");
    const timestamp = event.headers["tenancit-webhook-timestamp"];
    const expected = `v1=${createHmac("sha256", secret).update(`${timestamp}.${event.body}`).digest("hex")}`;
    expect(event.headers["tenancit-webhook-signature"]).toBe(expected);

    await page.reload();
    await page.getByRole("tab", { name: "Entregas", exact: true }).click();
    await page.getByPlaceholder("Buscar entregas…").fill(targetName);
    const deliveryRow = page.getByRole("row").filter({ hasText: targetName });
    await expect(deliveryRow).toHaveCount(1);
    await expect(deliveryRow).toContainText("tenancit.tenant.created");
    await expect(deliveryRow).toContainText("delivered");
  } finally {
    await cleanupWebhookFixtures(request, { targetID, tenantID });
  }
});
