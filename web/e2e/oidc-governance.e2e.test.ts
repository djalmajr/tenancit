import { expect, test } from "@playwright/test";

async function login(page: import("@playwright/test").Page) {
  await page.goto("/");
  await page.getByRole("button", { name: "Entrar com SSO" }).click();
  await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
}

test("OIDC settings are localized and another session can be revoked immediately", async ({ browser, page }) => {
  await login(page);

  await page.goto("/operations/settings");
  await expect(page.getByRole("heading", { name: "Configurações operacionais" })).toBeVisible();
  await page.getByRole("button", { name: "Idioma" }).click();
  await page.getByRole("menuitemradio", { name: "English" }).click();
  await expect(page.getByRole("heading", { name: "Operational settings" })).toBeVisible();

  const rpm = page.getByLabel("Default RPM for new keys");
  const originalRPM = await rpm.inputValue();
  await rpm.fill(String(Number(originalRPM) + 1));
  await page.getByRole("button", { name: "Save" }).click();
  await expect(page.getByText("Settings saved.")).toBeVisible();

  const secondContext = await browser.newContext();
  const secondPage = await secondContext.newPage();
  try {
    await login(secondPage);
    await page.goto("/security/sessions");
    await expect(page.getByRole("heading", { name: "Administrative sessions" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Revoke session" })).toHaveCount(1);
    await page.getByRole("button", { name: "Revoke session" }).click();
    await page.getByRole("dialog").getByRole("button", { name: "Revoke session" }).click();
    await expect(page.getByText("Session revoked.")).toBeVisible();

    await secondPage.reload();
    await expect(secondPage.getByRole("button", { name: "Entrar com SSO" })).toBeVisible();
  } finally {
    await secondContext.close();
  }
});
