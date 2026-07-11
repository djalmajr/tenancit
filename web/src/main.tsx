import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { RouterProvider } from "@tanstack/react-router";
import { QueryClientProvider } from "@tanstack/react-query";
import { router } from "./router";
import { createAppQueryClient, registerAdminQueryCacheInvalidation } from "./lib/query-client";
import "./index.css";

const el = document.getElementById("root");
if (!el) throw new Error("root element not found");
const queryClient = createAppQueryClient();
registerAdminQueryCacheInvalidation(queryClient);

createRoot(el).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </StrictMode>,
);
