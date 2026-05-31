import { createRouter } from "@tanstack/react-router";
import { Route as rootRoute } from "./routes/__root";
import { Route as indexRoute } from "./routes/index";
import { Route as tenantsRoute } from "./routes/tenants";
import { Route as tenantDetailRoute } from "./routes/tenant-detail";
import { Route as definitionsRoute } from "./routes/definitions";
import { Route as definitionDetailRoute } from "./routes/definition-detail";
import { Route as apiClientsRoute } from "./routes/api-clients";

const routeTree = rootRoute.addChildren([
  indexRoute,
  tenantsRoute,
  tenantDetailRoute,
  definitionsRoute,
  definitionDetailRoute,
  apiClientsRoute,
]);

export const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
