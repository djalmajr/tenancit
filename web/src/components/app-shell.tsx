import * as React from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import { Layers, LayoutDashboard, Building2, Boxes, KeyRound } from "lucide-react";
import {
  SidebarProvider,
  Sidebar,
  SidebarHeader,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarInset,
  SidebarTrigger,
} from "@/components/ui/sidebar";

const NAV = [
  { to: "/", label: "Visão geral", icon: LayoutDashboard, exact: true },
  { to: "/tenants", label: "Tenants", icon: Building2 },
  { to: "/resource-definitions", label: "Resource Definitions", icon: Boxes },
  { to: "/api-clients", label: "API Clients", icon: KeyRound },
];

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  return (
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader>
          <div className="flex items-center gap-2 px-1 py-1.5">
            <div className="flex size-8 shrink-0 items-center justify-center">
              <Layers className="size-[22px] text-primary" />
            </div>
            <div className="flex flex-col group-data-[state=collapsed]:hidden">
              <span className="text-sm font-semibold leading-tight">Resource Tenant</span>
              <span className="text-xs text-sidebar-foreground/60 leading-tight">Admin Console</span>
            </div>
          </div>
        </SidebarHeader>

        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupLabel>Gestão</SidebarGroupLabel>
            <SidebarMenu>
              {NAV.map((item) => {
                const active = item.exact ? pathname === item.to : pathname.startsWith(item.to);
                const Icon = item.icon;
                return (
                  <SidebarMenuItem key={item.to}>
                    <SidebarMenuButton asChild isActive={active} tooltip={item.label}>
                      <Link to={item.to}>
                        <Icon />
                        <span className="group-data-[state=collapsed]:hidden">{item.label}</span>
                      </Link>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                );
              })}
            </SidebarMenu>
          </SidebarGroup>
        </SidebarContent>

        <SidebarFooter>
          <div className="flex items-center gap-2 px-1 py-1.5">
            <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted text-xs font-medium">
              AD
            </div>
            <div className="flex flex-col group-data-[state=collapsed]:hidden">
              <span className="text-sm font-medium leading-tight">Admin</span>
              <span className="text-xs text-sidebar-foreground/60 leading-tight">admin@centralit</span>
            </div>
          </div>
        </SidebarFooter>
      </Sidebar>

      <SidebarInset>
        <header className="flex h-14 items-center gap-2 border-b px-4">
          <SidebarTrigger />
          <span className="text-sm font-medium">{pageLabel(pathname)}</span>
        </header>
        <main className="flex-1 overflow-auto p-6">{children}</main>
      </SidebarInset>
    </SidebarProvider>
  );
}

function pageLabel(pathname: string): string {
  if (pathname === "/") return "Visão geral";
  if (pathname.startsWith("/tenants")) return "Tenants";
  if (pathname.startsWith("/resource-definitions")) return "Resource Definitions";
  if (pathname.startsWith("/api-clients")) return "API Clients";
  return "";
}
