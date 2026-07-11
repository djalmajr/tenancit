import { useContext } from "react";
import { AdminCapabilitiesContext } from "@/lib/admin-capabilities-context";

export function useAdminCapabilities() {
  return useContext(AdminCapabilitiesContext);
}
