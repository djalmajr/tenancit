import { expect, test } from "@playwright/test";
import { adminRequest, authenticate, uniqueID } from "./fixtures/admin";
import { CatalogCleanup } from "./fixtures/catalog";

type Entity = { id: string; name: string };

test("all business routes support navigation, deep links, and reload", async ({ page, request }) => {
  const slug = uniqueID("route-tenant");
  const key = uniqueID("route-definition");
  const cleanup = new CatalogCleanup();

  try {
    const tenant = await adminRequest<Entity>(request, "post", "/v1/admin/tenants", {
      name: `Route tenant ${slug}`,
      slug,
    });
    cleanup.trackTenant(tenant.id);
    const definition = await adminRequest<Entity>(
      request,
      "post",
      "/v1/admin/resource-definitions",
      { key, name: `Route definition ${key}` },
    );
    cleanup.trackDefinition(definition.id);

    await authenticate(page);
    const routes = [
      { path: "/", heading: "Visão geral" },
      { path: "/tenants", heading: "Tenants" },
      { path: `/tenants/${tenant.id}`, heading: tenant.name },
      { path: "/resource-definitions", heading: "Definições de recurso" },
      { path: `/resource-definitions/${definition.id}`, heading: definition.name },
      { path: "/api-clients", heading: "Chaves de API" },
      { path: "/usage", heading: "Uso das chaves de API" },
      { path: "/audit-events", heading: "Auditoria administrativa" },
      { path: "/security/sessions", heading: "Sessões administrativas" },
      { path: "/operations/settings", heading: "Configurações operacionais" },
    ];

    for (const route of routes) {
      await test.step(`deep link and reload ${route.path}`, async () => {
        await page.goto(route.path);
        await expect(page.getByRole("heading", { name: route.heading, exact: true })).toBeVisible();
        await page.reload();
        await expect(page.getByRole("heading", { name: route.heading, exact: true })).toBeVisible();
      });
    }
  } finally {
    await cleanup.run(request);
  }
});
