import { createRootRoute, Outlet } from "@tanstack/react-router";
import { AppShell } from "@/components/app-shell";
import { Toaster } from "@/components/ui/sonner";
import { I18nProvider } from "@/lib/i18n";
import { ThemeProvider } from "@/lib/theme";

export const Route = createRootRoute({ component: RootLayout });

function RootLayout() {
  return (
    <I18nProvider>
      <ThemeProvider>
        <AppShell>
          <Outlet />
        </AppShell>
        <Toaster />
      </ThemeProvider>
    </I18nProvider>
  );
}
