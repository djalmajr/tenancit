import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { Building2, Globe, Database, Boxes, ChevronRight } from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { Empty, EmptyDescription, EmptyHeader } from "@/components/ui/empty";
import { DomainStatus } from "@/components/domain-status";
import { apiErrorMessage, formatStatus, useI18n, type TranslationKey } from "@/lib/i18n";
import { adminQueryOptions } from "@/lib/query-options";
import { StatCard } from "@/components/stat-card";

export default function OverviewPage() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const overviewQuery = useQuery(adminQueryOptions.overview());
  const o = overviewQuery.data;
  const error = overviewQuery.error ? apiErrorMessage(overviewQuery.error, t) : "";

  return (
    <div className="flex flex-col gap-8">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{t("overview.title")}</h1>
        <p className="text-muted-foreground">{t("overview.description")}</p>
      </div>

      {!o ? (
        <Card>
          <CardContent className="text-sm text-muted-foreground">
            {overviewQuery.isPending ? t("overview.loading") : authOrLoadMessage(error, t)}
          </CardContent>
        </Card>
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard icon={<Building2 className="size-4" />} label={t("overview.activeTenants")} value={o.activeTenants} hint={t("overview.totalTenants", { count: o.tenants })} />
            <StatCard icon={<Globe className="size-4" />} label={t("overview.domains")} value={o.domains} hint={t("overview.domainHint")} />
            <StatCard icon={<Database className="size-4" />} label={t("overview.resources")} value={o.resources} hint={t("overview.resourceHint")} />
            <StatCard icon={<Boxes className="size-4" />} label={t("overview.activeDefinitions")} value={o.activeDefinitions} hint={t("overview.typeHint")} />
          </div>

          <Card size="sm">
            <CardHeader>
              <CardTitle>{t("overview.tenantsCard.title")}</CardTitle>
              <CardDescription>{t("overview.tenantsCard.description")}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              {o.tenantCards.length === 0 ? (
                <Empty className="py-8"><EmptyHeader><EmptyDescription>{t("overview.emptyTenants")}</EmptyDescription></EmptyHeader></Empty>
              ) : (
                o.tenantCards.map((tenant) => (
                  <button
                    key={tenant.id}
                    onClick={() => { void navigate({ to: "/tenants/$id", params: { id: tenant.id } }); }}
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
                      <DomainStatus label={formatStatus(tenant.status, t)} value={tenant.status} />
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
