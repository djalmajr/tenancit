import { createMemoryHistory } from "@tanstack/react-router";
import { afterEach, describe, expect, it } from "vitest";
import { RUNTIME_BASE_PATH_META_NAME } from "./lib/runtime-base-path";
import { createAppRouter } from "./router";

afterEach(() => {
  document.head.replaceChildren();
});

describe("application router base path", () => {
  // Mutation captured during Red: omitting createRouter.basepath made this route unmatched.
  it("matches internal routes while exposing base-prefixed public URLs", async () => {
    const meta = document.createElement("meta");
    meta.content = "/tenancit";
    meta.name = RUNTIME_BASE_PATH_META_NAME;
    document.head.append(meta);
    const history = createMemoryHistory({
      initialEntries: ["/tenancit/tenants?status=active"],
    });
    const appRouter = createAppRouter({ history });

    await appRouter.load();

    expect(appRouter.state.location.pathname).toBe("/tenants");
    expect(appRouter.state.location.publicHref).toBe("/tenancit/tenants?status=active");
    expect(appRouter.buildLocation({ to: "/resource-definitions" }).publicHref).toBe(
      "/tenancit/resource-definitions",
    );
  });
});
