import { useQuery } from "@tanstack/react-query";
import { Activity, BarChart3, Building2, CircleAlert, Database, FileKey2, Globe, Boxes } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { apiErrorMessage, useI18n, type TranslationKey } from "@/lib/i18n";
import { adminQueryOptions } from "@/lib/query-options";
import { StatCard } from "@/components/stat-card";
import { api } from "@/lib/api";
import { adminQueryKeys } from "@/lib/query-keys";
import { countExpiringAPIClients, currentUTCMonthRange, totalUsageRequests } from "@/lib/overview-pulse";

export default function OverviewPage() {
  const { t } = useI18n();
  const overviewQuery = useQuery(adminQueryOptions.overview());
  const apiClientsQuery = useQuery(adminQueryOptions.apiClients());
  const monthRange = currentUTCMonthRange();
  const usageQuery = useQuery({
    queryKey: adminQueryKeys.apiClientUsage(monthRange.from, monthRange.to),
    queryFn: ({ signal }) => api.listAPIClientUsage(monthRange.from, monthRange.to, signal),
    refetchInterval: () => document.visibilityState === "visible" ? 60_000 : false,
  });
  const healthQuery = useQuery({
    queryKey: adminQueryKeys.operationalHealth(),
    queryFn: ({ signal }) => api.getOperationalHealth(signal),
    refetchInterval: () => document.visibilityState === "visible" ? 30_000 : false,
  });
  const o = overviewQuery.data;
  const error = overviewQuery.error ? apiErrorMessage(overviewQuery.error, t) : "";
  const expiringClients = apiClientsQuery.data
    ? countExpiringAPIClients(apiClientsQuery.data)
    : "—";
  const monthlyRequests = usageQuery.data ? totalUsageRequests(usageQuery.data) : "—";
  const healthStatus = healthQuery.data
    ? t(`operationsHealth.status.${healthQuery.data.status}` as TranslationKey)
    : "—";

  return (
    <div className="flex flex-col gap-8">
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

          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard icon={<Activity className="size-4" />} label={t("overview.operationalHealth")} value={healthStatus} hint={t("overview.operationalHealthHint")} />
            <StatCard icon={<BarChart3 className="size-4" />} label={t("overview.monthRequests")} value={monthlyRequests} hint={t("overview.monthRequestsHint")} />
            <StatCard icon={<FileKey2 className="size-4" />} label={t("overview.expiringKeys")} value={expiringClients} hint={t("overview.expiringKeysHint")} />
            <StatCard icon={<CircleAlert className="size-4" />} label={t("overview.deadLetters")} value={healthQuery.data?.queues.webhook_dead_letter ?? "—"} hint={t("overview.deadLettersHint")} />
          </div>
        </>
      )}
    </div>
  );
}

function authOrLoadMessage(error: string, t: (key: TranslationKey) => string): string {
  if (error.startsWith("401:")) return t("overview.authRequired");
  return t("overview.loadError");
}
