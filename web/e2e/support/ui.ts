import { expect, type Page } from "@playwright/test";
import { adminToken } from "../fixtures/admin";
import {
  type APIClientRecord,
  CatalogCleanup,
  type DefinitionRecord,
  type TenantRecord,
} from "../fixtures/catalog";

export async function loginViaUI(page: Page, token = adminToken) {
  await page.goto("/");
  const heading = page.getByRole("heading", { name: "Acesso administrativo" });
  await expect(heading).toBeVisible();
  await page.getByRole("textbox", { name: "Token", exact: true }).fill(token);
  await page.getByRole("button", { name: "Entrar" }).click();
  if (token === adminToken) await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
}

export async function logoutViaUI(page: Page) {
  await page.getByRole("button", { name: /^(Conta|Account|Cuenta)$/ }).click();
  await page.getByRole("menuitem", { name: /^(Sair|Log out|Salir)$/ }).click();
  await expect(page.getByRole("heading", { name: "Acesso administrativo" })).toBeVisible();
}

export async function navigateFromSidebar(page: Page, name: "Visão geral" | "Tenants" | "Recursos" | "Chaves de API") {
  await page.getByRole("complementary").getByRole("link", { name, exact: true }).click();
  const heading = name === "Recursos" ? "Definições de recurso" : name;
  await expect(page.getByRole("heading", { name: heading, exact: true })).toBeVisible();
}

export async function createTenantViaUI(page: Page, cleanup: CatalogCleanup, name: string, slug: string) {
  await page.goto("/tenants");
  await expect(page.getByRole("heading", { name: "Tenants", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Novo tenant" }).click();
  const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Novo tenant" }) });
  await expect(dialog).toBeVisible();
  await dialog.getByLabel("Nome", { exact: true }).fill(name);
  await dialog.getByLabel("Slug", { exact: true }).fill(slug);
  const [response] = await Promise.all([
    page.waitForResponse((candidate) =>
      candidate.request().method() === "POST" && new URL(candidate.url()).pathname === "/v1/admin/tenants"),
    dialog.getByRole("button", { name: "Criar tenant" }).click(),
  ]);
  expect(response.ok(), "tenant creation response").toBe(true);
  const tenant = await response.json() as TenantRecord;
  cleanup.trackTenant(tenant.id);
  await expect(page.getByRole("heading", { name, exact: true })).toBeVisible();
  return tenant;
}

export async function createDefinitionViaUI(page: Page, cleanup: CatalogCleanup, name: string, key: string) {
  await page.goto("/resource-definitions");
  await expect(page.getByRole("heading", { name: "Definições de recurso" })).toBeVisible();
  await page.getByRole("button", { name: "Nova definição" }).first().click();
  const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Nova definição" }) });
  await dialog.getByLabel("Key", { exact: true }).fill(key);
  await dialog.getByLabel("Nome", { exact: true }).fill(name);
  await dialog.getByLabel("Descrição", { exact: true }).fill(`Definition ${key}`);
  const [response] = await Promise.all([
    page.waitForResponse((candidate) =>
      candidate.request().method() === "POST"
      && new URL(candidate.url()).pathname === "/v1/admin/resource-definitions"),
    dialog.getByRole("button", { name: "Criar definição" }).click(),
  ]);
  expect(response.ok(), "definition creation response").toBe(true);
  const definition = await response.json() as DefinitionRecord;
  cleanup.trackDefinition(definition.id);
  await expect(page.getByRole("heading", { name, exact: true })).toBeVisible();
  return definition;
}

export async function addFieldViaUI(
  page: Page,
  field: { key: string; label: string; required?: boolean; secret?: boolean; dataType?: "string" | "int" | "bool" },
) {
  await page.getByRole("button", { name: "Novo campo" }).click();
  const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Novo campo" }) });
  await dialog.getByLabel("Key", { exact: true }).fill(field.key);
  await dialog.getByLabel("Label", { exact: true }).fill(field.label);
  if (field.dataType && field.dataType !== "string") {
    await dialog.getByRole("combobox").click();
    await page.getByRole("option", { name: field.dataType, exact: true }).click();
  }
  if (field.required) {
    await dialog.getByRole("checkbox", { name: "Obrigatório" }).check();
  }
  if (field.secret) {
    await dialog.getByRole("checkbox", { name: "Segredo" }).check();
  }
  await dialog.getByRole("button", { name: "Adicionar campo" }).click();
  await expect(page.getByRole("cell", { name: field.key, exact: true })).toBeVisible();
}

export async function addDomainViaUI(page: Page, hostname: string) {
  await page.getByRole("tab", { name: "Domínios", exact: true }).click();
  await page.getByRole("button", { name: "Adicionar", exact: true }).click();
  const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Adicionar domínio" }) });
  await dialog.getByPlaceholder("app.cliente.com").fill(hostname);
  await dialog.getByRole("button", { name: "Adicionar", exact: true }).click();
  await expect(page.getByRole("cell", { name: hostname, exact: true })).toBeVisible();
}

export async function addResourceViaUI(
  page: Page,
  definitionName: string,
  values: Record<string, string>,
) {
  await page.getByRole("tab", { name: "Recursos", exact: true }).click();
  await page.getByRole("button", { name: "Adicionar recurso" }).click();
  const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Adicionar recurso" }) });
  await dialog.getByRole("combobox").click();
  await page.getByRole("option", { name: definitionName, exact: true }).click();
  for (const [key, value] of Object.entries(values)) {
    await dialog.getByPlaceholder(key, { exact: true }).fill(value);
  }
  await dialog.getByRole("button", { name: "Salvar recurso" }).click();
  await expect(dialog).toHaveCount(0);
  await expect(page.getByText("Recurso adicionado.", { exact: true }).last()).toBeVisible();
  await expect(page.getByText(definitionName, { exact: true })).toBeVisible();
}

export async function createAPIClientViaUI(page: Page, cleanup: CatalogCleanup, name: string) {
  await page.goto("/api-clients");
  await expect(page.getByRole("heading", { name: "Chaves de API" })).toBeVisible();
  await page.getByRole("button", { name: "Nova chave" }).click();
  const dialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Nova chave de API" }) });
  await dialog.getByPlaceholder("billing-service").fill(name);
  const [response] = await Promise.all([
    page.waitForResponse((candidate) =>
      candidate.request().method() === "POST" && new URL(candidate.url()).pathname === "/v1/admin/api-clients"),
    dialog.getByRole("button", { name: "Gerar token" }).click(),
  ]);
  expect(response.ok(), "API client creation response").toBe(true);
  const created = await response.json() as { client: APIClientRecord; token: string };
  cleanup.trackAPIClient(created.client.id);
  const tokenDialog = page.getByRole("dialog").filter({ has: page.getByRole("heading", { name: "Token gerado" }) });
  const token = await tokenDialog.getByRole("textbox", { name: "Token" }).inputValue();
  expect(token.startsWith("tnc_") && token.length > 4).toBe(true);
  return { client: created.client, dialog: tokenDialog, token };
}

export async function chooseLocale(page: Page, locale: "Português" | "English" | "Español") {
  await page.getByRole("button", { name: /^(Idioma|Language)/ }).click();
  await page.getByRole("menuitemradio", { name: locale }).click();
}

export async function chooseTheme(
  page: Page,
  theme: "Claro" | "Escuro" | "Sistema" | "Light" | "Dark" | "System" | "Oscuro",
) {
  await page.getByRole("button", { name: /^(Tema|Theme)/ }).click();
  await page.getByRole("menuitemradio", { name: theme }).click();
}
