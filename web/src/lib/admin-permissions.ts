export type AdminPermission =
  | "admin.read" | "audit.read" | "audit.export" | "audit.manage"
  | "api_client.manage" | "integration.manage" | "resource.write"
  | "secret.reveal" | "session.manage" | "settings.manage"
  | "tenant.hard_delete" | "tenant.write";

export const ALL_ADMIN_PERMISSIONS: AdminPermission[] = [
  "admin.read", "audit.read", "audit.export", "audit.manage",
  "api_client.manage", "integration.manage", "resource.write",
  "secret.reveal", "session.manage", "settings.manage",
  "tenant.hard_delete", "tenant.write",
];
