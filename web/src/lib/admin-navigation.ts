import {
  BarChart3, Boxes, Building2, HeartPulse, KeyRound, LayoutDashboard,
  MonitorDot, ScrollText, Settings, Webhook, type LucideIcon,
} from "lucide-react";
import type { TranslationKey } from "@/lib/i18n";
import type { AdminPermission } from "@/lib/admin-permissions";

type NavItem = { exact?: boolean; icon: LucideIcon; labelKey: TranslationKey; permission: AdminPermission; to: string };

export const NAV_GROUPS: Array<{ labelKey: TranslationKey; items: NavItem[] }> = [
  { labelKey: "nav.management", items: [
    { exact: true, icon: LayoutDashboard, labelKey: "nav.overview", permission: "admin.read", to: "/" },
    { icon: Building2, labelKey: "nav.tenants", permission: "admin.read", to: "/tenants" },
    { icon: Boxes, labelKey: "nav.definitions", permission: "admin.read", to: "/resource-definitions" },
    { icon: KeyRound, labelKey: "nav.apiClients", permission: "admin.read", to: "/api-clients" },
    { icon: BarChart3, labelKey: "nav.usage", permission: "admin.read", to: "/usage" },
  ] },
  { labelKey: "nav.operations", items: [
    { icon: HeartPulse, labelKey: "nav.health", permission: "admin.read", to: "/operations/health" },
    { icon: Webhook, labelKey: "nav.integrations", permission: "integration.manage", to: "/integrations/webhooks" },
    { icon: ScrollText, labelKey: "nav.audit", permission: "audit.read", to: "/audit-events" },
  ] },
  { labelKey: "nav.system", items: [
    { icon: MonitorDot, labelKey: "nav.sessions", permission: "session.manage", to: "/security/sessions" },
    { icon: Settings, labelKey: "nav.settings", permission: "admin.read", to: "/operations/settings" },
  ] },
];

export function visibleNavGroups(permissions: ReadonlySet<string>) {
  return NAV_GROUPS.map((group) => ({ ...group, items: group.items.filter((item) => permissions.has(item.permission)) }))
    .filter((group) => group.items.length > 0);
}
