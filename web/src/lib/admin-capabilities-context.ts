import { createContext } from "react";
import { ALL_ADMIN_PERMISSIONS, type AdminPermission } from "@/lib/admin-permissions";

export type AdminCapabilities = {
  can: (permission: AdminPermission) => boolean;
  permissions: ReadonlySet<string>;
};

const fullCapabilities = new Set<string>(ALL_ADMIN_PERMISSIONS);

export const AdminCapabilitiesContext = createContext<AdminCapabilities>({
  can: (permission) => fullCapabilities.has(permission),
  permissions: fullCapabilities,
});
