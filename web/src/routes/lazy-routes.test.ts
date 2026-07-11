import { describe, expect, it } from "vitest";
import { Route as overviewRoute } from "./index";
import { Route as tenantsRoute } from "./tenants";
import { Route as tenantDetailRoute } from "./tenant-detail";
import { Route as definitionsRoute } from "./definitions";
import { Route as definitionDetailRoute } from "./definition-detail";
import { Route as apiClientsRoute } from "./api-clients";
import { Route as usageRoute } from "./usage";
import { Route as auditEventsRoute } from "./audit-events";
import { Route as sessionsRoute } from "./sessions";
import { Route as settingsRoute } from "./settings";
import { Route as integrationsRoute } from "./integrations";
import { Route as operationsHealthRoute } from "./operations-health";

const routes = [
  ["overview", overviewRoute],
  ["tenants", tenantsRoute],
  ["tenant detail", tenantDetailRoute],
  ["definitions", definitionsRoute],
  ["definition detail", definitionDetailRoute],
  ["API clients", apiClientsRoute],
  ["usage", usageRoute],
  ["audit events", auditEventsRoute],
  ["sessions", sessionsRoute],
  ["settings", settingsRoute],
  ["integrations", integrationsRoute],
  ["operational health", operationsHealthRoute],
] as const;

describe("business route loading boundary", () => {
  it.each(routes)("loads %s through a lazy component", async (_name, route) => {
    const component = route.options.component as
      | { preload?: () => Promise<unknown> }
      | undefined;

    expect(component?.preload).toBeTypeOf("function");
    await component?.preload?.();
  });
});
