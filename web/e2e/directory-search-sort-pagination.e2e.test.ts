import { expect, test, type Locator } from "@playwright/test";
import {
  CatalogCleanup,
  createAPIClientFixture,
  createTenantFixture,
  type APIClientRecord,
} from "./fixtures/catalog";
import { adminRequest, uniqueID } from "./fixtures/admin";
import { flowStep } from "./support/flow-step";
import { loginViaUI, navigateFromSidebar } from "./support/ui";

test.use({ screenshot: "off", trace: "off" });

async function expectColumnOrder(
  cells: Locator,
  direction: "asc" | "desc",
  numericValueByLabel?: Map<string, number>,
) {
  await expect.poll(async () => {
    const values = (await cells.allTextContents()).map((value) => value.trim());
    if (numericValueByLabel) {
      const numericValues = values.map((value) => numericValueByLabel.get(value));
      if (numericValues.some((value) => value === undefined)) return false;
      return numericValues.slice(1).every((value, index) =>
        direction === "asc"
          ? numericValues[index]! <= value!
          : numericValues[index]! >= value!);
    }
    const sorted = [...values].sort((left, right) => left.localeCompare(right, "pt-BR"));
    if (direction === "desc") sorted.reverse();
    return JSON.stringify(values) === JSON.stringify(sorted);
  }).toBe(true);
}

async function cycleSort(
  button: Locator,
  cells?: Locator,
  numericValueByLabel?: Map<string, number>,
) {
  for (let attempt = 0; attempt < 3; attempt += 1) {
    const before = await button.getAttribute("title");
    if (before === "Ordenar crescente") break;
    await button.click();
    await expect.poll(() => button.getAttribute("title")).not.toBe(before);
  }
  await expect.poll(() => button.getAttribute("title")).toBe("Ordenar crescente");

  await button.click();
  await expect(button).toHaveAttribute("title", "Ordenar decrescente");
  if (cells) await expectColumnOrder(cells, "asc", numericValueByLabel);

  await button.click();
  await expect(button).toHaveAttribute("title", "Limpar ordenação");
  if (cells) await expectColumnOrder(cells, "desc", numericValueByLabel);

  await button.click();
  await expect(button).toHaveAttribute("title", "Ordenar crescente");
}

test("directory tables search, sort, and paginate without losing records", { tag: "@full" }, async ({ page, request }) => {
  test.slow();
  const cleanup = new CatalogCleanup();
  const tenantBatch = uniqueID("directory-tenants");
  const clientBatch = uniqueID("directory-clients");
  const tenantNames: string[] = [];
  const tenantSlugs: string[] = [];
  const tenants: Array<Awaited<ReturnType<typeof createTenantFixture>>> = [];
  const clients: Array<Awaited<ReturnType<typeof createAPIClientFixture>>> = [];

  try {
    for (let index = 0; index < 12; index += 1) {
      const suffix = index.toString().padStart(2, "0");
      const tenant = await createTenantFixture(request, cleanup, `${tenantBatch}-${suffix}`);
      tenants.push(tenant);
      tenantNames.push(tenant.name);
      tenantSlugs.push(tenant.slug);
      clients.push(
        await createAPIClientFixture(
          request,
          cleanup,
          `${clientBatch}-${suffix}`,
        ),
      );
    }
    await adminRequest(request, "put", `/v1/admin/tenants/${tenants[0].id}`, {
      name: tenants[0].name,
      slug: tenants[0].slug,
      status: "inactive",
    });

    await flowStep("directory-search-sort-pagination", 1, "autentica o operador", async () => {
      await loginViaUI(page);
      await expect(page.getByRole("heading", { name: "Visão geral" })).toBeVisible();
    });
    await flowStep("directory-search-sort-pagination", 2, "abre o diretório populado de tenants", async () => {
      await navigateFromSidebar(page, "Tenants");
      await expect(page.getByText("12 itens", { exact: true })).toBeVisible();
      await expect(page.getByText("Página 1 de 3", { exact: true })).toBeVisible();
      await expect(page.locator("tbody tr")).toHaveCount(5);
      await expect(page.getByRole("button", { name: "Próxima página" })).toBeEnabled();
    });
    await flowStep("directory-search-sort-pagination", 3, "respeita limites em todos os controles", async () => {
      const first = page.getByRole("button", { name: "Primeira página" });
      const previous = page.getByRole("button", { name: "Página anterior" });
      const next = page.getByRole("button", { name: "Próxima página" });
      const last = page.getByRole("button", { name: "Última página" });
      await expect(first).toBeDisabled();
      await expect(previous).toBeDisabled();
      await next.click();
      await expect(page.getByText("Página 2 de 3", { exact: true })).toBeVisible();
      await last.click();
      await expect(page.getByText("Página 3 de 3", { exact: true })).toBeVisible();
      await expect(next).toBeDisabled();
      await expect(last).toBeDisabled();
      await previous.click();
      await expect(page.getByText("Página 2 de 3", { exact: true })).toBeVisible();
      await first.click();
      await expect(page.getByText("Página 1 de 3", { exact: true })).toBeVisible();
      await expect(first).toBeDisabled();
      await expect(previous).toBeDisabled();
    });
    await flowStep("directory-search-sort-pagination", 4, "persiste e restaura preferências da tabela", async () => {
      await page.getByRole("combobox").click();
      await page.getByRole("option", { name: "10", exact: true }).click();
      await expect(page.getByText("Página 1 de 2", { exact: true })).toBeVisible();
      await expect(page.locator("tbody tr")).toHaveCount(10);
      await page.reload();
      await expect(page.getByText("Página 1 de 2", { exact: true })).toBeVisible();
      await expect(page.locator("tbody tr")).toHaveCount(10);
      await page.getByRole("button", { name: "Colunas" }).click();
      await page.getByText("Restaurar tabela", { exact: true }).click();
      await expect(page.getByText("Página 1 de 3", { exact: true })).toBeVisible();
      await page.getByRole("combobox").click();
      await page.getByRole("option", { name: "10", exact: true }).click();
      await expect(page.getByText("Página 1 de 2", { exact: true })).toBeVisible();
    });
    await flowStep("directory-search-sort-pagination", 5, "ordena nome slug e status em três estados", async () => {
      await cycleSort(
        page.getByRole("button", { name: /Nome:/ }),
        page.locator("tbody tr td:nth-child(1)"),
      );
      await cycleSort(
        page.getByRole("button", { name: /Slug:/ }),
        page.locator("tbody tr td:nth-child(2)"),
      );
      await cycleSort(
        page.getByRole("button", { name: /Status:/ }),
        page.locator("tbody tr td:nth-child(3)"),
      );
      await expect(page.locator("tbody tr")).toHaveCount(10);
    });
    await flowStep("directory-search-sort-pagination", 6, "filtra por múltiplos campos e restaura o conjunto", async () => {
      const search = page.getByRole("textbox", { name: /Buscar por nome ou slug/ });
      await search.fill(tenantNames[0]);
      await expect(page.getByRole("row").filter({ hasText: tenantNames[0] })).toBeVisible();
      await expect(page.locator("tbody tr")).toHaveCount(1);
      await search.fill(tenantSlugs[11]);
      await expect(page.locator("tbody tr")).toHaveCount(1);
      await expect(page.getByRole("row").filter({ hasText: tenantSlugs[11] })).toBeVisible();
      await search.fill("ativo");
      await expect(page.getByText("11 itens", { exact: true })).toBeVisible();
      await expect(page.locator("tbody tr")).toHaveCount(10);
      await search.clear();
      await expect(page.getByText("12 itens", { exact: true })).toBeVisible();
      await expect(page.getByText("Página 1 de 2", { exact: true })).toBeVisible();
    });
    await flowStep("directory-search-sort-pagination", 7, "repete controles equivalentes em chaves", async () => {
      await navigateFromSidebar(page, "Chaves de API");
      await expect(page.getByRole("button", { name: /Criado em: Limpar ordenação/ })).toBeVisible();
      const search = page.getByRole("textbox", {
        name: "Buscar por nome, data ou status...",
        exact: true,
      });
      await search.fill(clientBatch);
      await expect(page.getByText("12 itens", { exact: true })).toBeVisible();
      await expect(page.getByText("Página 1 de 3", { exact: true })).toBeVisible();
      await cycleSort(
        page.getByRole("button", { name: /Nome:/ }),
        page.locator("tbody tr td:nth-child(1)"),
      );
      const listedClients = (await adminRequest<APIClientRecord[] | null>(
        request,
        "get",
        "/v1/admin/api-clients",
      )) ?? [];
      const createdAtByName = new Map(
        listedClients
          .filter((client) => client.name.startsWith(clientBatch))
          .map((client) => [client.name, Date.parse(client.created_at ?? "")] as const),
      );
      await cycleSort(
        page.getByRole("button", { name: /Criado em:/ }),
        page.locator("tbody tr td:nth-child(1)"),
        createdAtByName,
      );
      await page.getByRole("button", { name: "Próxima página" }).click();
      await expect(page.getByText("Página 2 de 3", { exact: true })).toBeVisible();
      await page.getByRole("button", { name: "Última página" }).click();
      await expect(page.getByText("Página 3 de 3", { exact: true })).toBeVisible();
      await page.getByRole("button", { name: "Página anterior" }).click();
      await expect(page.getByText("Página 2 de 3", { exact: true })).toBeVisible();
      await page.getByRole("button", { name: "Primeira página" }).click();
      await expect(page.getByText("Página 1 de 3", { exact: true })).toBeVisible();

      const target = clients[clients.length - 1];
      await search.fill(target.name);
      const targetRow = page.getByRole("row").filter({ hasText: target.name });
      await expect(targetRow).toBeVisible();
      await expect(page.locator("tbody tr")).toHaveCount(1);
      await targetRow.getByRole("button", { name: "Revogar" }).click();
      await expect(targetRow).toContainText("revogado");

      const createdDate = (target.client.created_at ?? "").slice(0, 10);
      expect(createdDate).toMatch(/^\d{4}-\d{2}-\d{2}$/);
      await search.fill(createdDate);
      await expect(page.getByRole("row").filter({ hasText: target.name })).toBeVisible();

      await search.fill("revogado");
      await expect(page.getByRole("row").filter({ hasText: target.name })).toBeVisible();
      const revokedRows = page.locator("tbody tr");
      await expect(revokedRows.first()).toBeVisible();
      await expect.poll(async () =>
        (await revokedRows.allTextContents()).every((row) => row.includes("revogado")),
      ).toBe(true);
      await search.fill(clientBatch);
      await expect(page.getByText("12 itens", { exact: true })).toBeVisible();
      await cycleSort(
        page.getByRole("button", { name: /Status:/ }),
        page.locator("tbody tr td:nth-child(4)"),
      );
    });
  } finally {
    await cleanup.run(request);
  }
});
