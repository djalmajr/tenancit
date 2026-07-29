import { createRouter, type RouterHistory } from "@tanstack/react-router";
import { Route as rootRoute } from "./routes/__root";
import { Route as indexRoute } from "./routes/index";
import { Route as tenantsRoute } from "./routes/tenants";
import { Route as tenantDetailRoute } from "./routes/tenant-detail";
import { Route as definitionsRoute } from "./routes/definitions";
import { Route as definitionDetailRoute } from "./routes/definition-detail";
import { Route as apiClientsRoute } from "./routes/api-clients";
import { Route as usageRoute } from "./routes/usage";
import { Route as auditEventsRoute } from "./routes/audit-events";
import { Route as sessionsRoute } from "./routes/sessions";
import { Route as settingsRoute } from "./routes/settings";
import { Route as integrationsRoute } from "./routes/integrations";
import { Route as operationsHealthRoute } from "./routes/operations-health";
import { RoutePending } from "./components/route-pending";
import { readRuntimeBasePath } from "./lib/runtime-base-path";

const routeTree = rootRoute.addChildren([
  indexRoute,
  tenantsRoute,
  tenantDetailRoute,
  definitionsRoute,
  definitionDetailRoute,
  apiClientsRoute,
  usageRoute,
  auditEventsRoute,
  sessionsRoute,
  settingsRoute,
  integrationsRoute,
  operationsHealthRoute,
]);

interface CreateAppRouterOptions {
  basepath?: string;
  history?: RouterHistory;
}

export function createAppRouter({
  basepath = readRuntimeBasePath(),
  history,
}: CreateAppRouterOptions = {}) {
  return createRouter({
    basepath,
    defaultPendingComponent: RoutePending,
    defaultPendingMinMs: 150,
    defaultPendingMs: 0,
    history,
    routeTree,
  });
}

export const router = createAppRouter();

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
