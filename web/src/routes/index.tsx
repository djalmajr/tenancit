import { createRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { Building2, Globe, Database, Boxes, ChevronRight } from "lucide-react";
import { Route as rootRoute } from "./__root";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { api, type Overview } from "@/lib/api";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: OverviewPage,
});

function Stat({
  icon,
  label,
  value,
  hint,
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
  hint: string;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2 border-b-0">
        <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
        <span className="text-muted-foreground">{icon}</span>
      </CardHeader>
      <CardContent className="pt-0">
        <div className="text-2xl font-bold">{value}</div>
        <p className="text-xs text-muted-foreground">{hint}</p>
      </CardContent>
    </Card>
  );
}

function OverviewPage() {
  const navigate = useNavigate();
  const [o, setO] = useState<Overview | null>(null);

  useEffect(() => {
    api.overview().then(setO).catch(() => setO(null));
  }, []);

  const d = o ?? {
    tenants: 0, activeTenants: 0, domains: 0, resources: 0,
    definitions: 0, activeDefinitions: 0, apiClients: 0, tenantCards: [],
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Visão geral</h1>
        <p className="text-muted-foreground">Configuração multi-tenant de recursos por cliente.</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Stat icon={<Building2 className="size-4" />} label="Tenants ativos" value={d.activeTenants} hint={`${d.tenants} no total`} />
        <Stat icon={<Globe className="size-4" />} label="Domínios" value={d.domains} hint="hostnames mapeados" />
        <Stat icon={<Database className="size-4" />} label="Recursos provisionados" value={d.resources} hint="instâncias ativas" />
        <Stat icon={<Boxes className="size-4" />} label="Definitions ativas" value={d.activeDefinitions} hint="tipos de recurso" />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Tenants</CardTitle>
          <p className="text-sm text-muted-foreground">Atalho para os clientes cadastrados.</p>
        </CardHeader>
        <CardContent className="space-y-2">
          {d.tenantCards.length === 0 ? (
            <p className="text-sm text-muted-foreground">Nenhum tenant ainda.</p>
          ) : (
            d.tenantCards.map((t) => (
              <button
                key={t.id}
                onClick={() => navigate({ to: "/tenants/$id", params: { id: t.id } })}
                className="flex w-full items-center justify-between rounded-md border p-3 text-left transition-colors hover:bg-accent"
              >
                <div className="flex items-center gap-3">
                  <div className="flex size-9 items-center justify-center rounded-md bg-muted">
                    <Building2 className="size-4" />
                  </div>
                  <div>
                    <div className="font-medium">{t.name}</div>
                    <div className="text-xs text-muted-foreground">{t.primaryHost || "sem domínio"}</div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-xs text-muted-foreground">{t.resourceCount} recursos</span>
                  <Badge variant={t.status === "active" ? "default" : "secondary"}>{t.status}</Badge>
                  <ChevronRight className="size-4 text-muted-foreground" />
                </div>
              </button>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  );
}
