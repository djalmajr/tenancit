import { useMemo, type ReactNode } from "react";
import { AdminCapabilitiesContext } from "@/lib/admin-capabilities-context";
import type { AdminPermission } from "@/lib/admin-permissions";

export function AdminCapabilitiesProvider({ children, permissions }: { children: ReactNode; permissions: string[] }) {
  const value = useMemo(() => {
    const set = new Set(permissions);
    return { can: (permission: AdminPermission) => set.has(permission), permissions: set };
  }, [permissions]);
  return <AdminCapabilitiesContext.Provider value={value}>{children}</AdminCapabilitiesContext.Provider>;
}
