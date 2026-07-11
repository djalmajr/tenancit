import { createRoute, lazyRouteComponent } from "@tanstack/react-router";
import { Route as rootRoute } from "./__root";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tenants/$id",
  component: lazyRouteComponent(() => import("./tenant-detail.page")),
});
