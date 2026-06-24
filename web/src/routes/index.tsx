import { createRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState, type ReactNode } from "react";
import { Building2, Globe, Database, Boxes, ChevronRight } from "lucide-react";
import { Route as rootRoute } from "./__root";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { formatStatus, useI18n, type TranslationKey } from "@/lib/i18n";
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
  icon: ReactNode;
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
  const { t } = useI18n();
  const navigate = useNavigate();
  const [o, setO] = useState<Overview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    let alive = true;
    setLoading(true);
    api.overview()
      .then((overview) => {
        if (!alive) return;
        setO(overview);
        setError("");
      })
      .catch((err) => {
        if (!alive) return;
        setO(null);
        setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        if (alive) setLoading(false);
      });
    return () => {
      alive = false;
    };
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{t("overview.title")}</h1>
        <p className="text-muted-foreground">{t("overview.description")}</p>
      </div>

      {!o ? (
        <Card>
          <CardContent className="text-sm text-muted-foreground">
            {loading ? t("overview.loading") : authOrLoadMessage(error, t)}
          </CardContent>
        </Card>
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Stat icon={<Building2 className="size-4" />} label={t("overview.activeTenants")} value={o.activeTenants} hint={t("overview.totalTenants", { count: o.tenants })} />
            <Stat icon={<Globe className="size-4" />} label={t("overview.domains")} value={o.domains} hint={t("overview.domainHint")} />
            <Stat icon={<Database className="size-4" />} label={t("overview.resources")} value={o.resources} hint={t("overview.resourceHint")} />
            <Stat icon={<Boxes className="size-4" />} label={t("overview.activeDefinitions")} value={o.activeDefinitions} hint={t("overview.typeHint")} />
          </div>

          <Card>
            <CardHeader>
              <CardTitle>{t("overview.tenantsCard.title")}</CardTitle>
              <p className="text-sm text-muted-foreground">{t("overview.tenantsCard.description")}</p>
            </CardHeader>
            <CardContent className="space-y-2">
              {o.tenantCards.length === 0 ? (
                <p className="text-sm text-muted-foreground">{t("overview.emptyTenants")}</p>
              ) : (
                o.tenantCards.map((tenant) => (
                  <button
                    key={tenant.id}
                    onClick={() => navigate({ to: "/tenants/$id", params: { id: tenant.id } })}
                    className="flex w-full items-center justify-between rounded-md border p-3 text-left transition-colors hover:bg-accent"
                  >
                    <div className="flex items-center gap-3">
                      <div className="flex size-9 items-center justify-center rounded-md bg-muted">
                        <Building2 className="size-4" />
                      </div>
                      <div>
                        <div className="font-medium">{tenant.name}</div>
                        <div className="text-xs text-muted-foreground">{tenant.primaryHost || t("overview.noDomain")}</div>
                      </div>
                    </div>
                    <div className="flex items-center gap-3">
                      <span className="text-xs text-muted-foreground">{t("overview.resourceCount", { count: tenant.resourceCount })}</span>
                      <Badge variant={tenant.status === "active" ? "default" : "secondary"}>{formatStatus(tenant.status, t)}</Badge>
                      <ChevronRight className="size-4 text-muted-foreground" />
                    </div>
                  </button>
                ))
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  );
}

function authOrLoadMessage(error: string, t: (key: TranslationKey) => string): string {
  if (error.startsWith("401:")) return t("overview.authRequired");
  return t("overview.loadError");
}
