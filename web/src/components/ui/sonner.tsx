import { Toaster as SonnerToaster } from "sonner";
import { useTheme } from "@/lib/theme";

/** App-wide toast region. Syncs with the app's own theme (no next-themes). */
export function Toaster() {
  const { resolvedTheme } = useTheme();
  return (
    <SonnerToaster
      theme={resolvedTheme}
      position="bottom-right"
      closeButton
      toastOptions={{ duration: 4000 }}
    />
  );
}
