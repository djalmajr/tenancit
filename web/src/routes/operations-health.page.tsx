import { useQuery } from "@tanstack/react-query";
import { Activity, CircleAlert, Database, RefreshCw } from "lucide-react";
import { StatCard } from "@/components/stat-card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { api } from "@/lib/api";
import { apiErrorMessage, type TranslationKey, useI18n } from "@/lib/i18n";
import { adminQueryKeys } from "@/lib/query-keys";

function statusVariant(status: string): "default" | "secondary" | "destructive" {
  if (status === "healthy") return "default";
  if (status === "degraded" || status === "stale") return "secondary";
  return "destructive";
}

export default function OperationsHealthPage() {
  const { locale, t } = useI18n();
  const healthQuery = useQuery({
    queryKey: adminQueryKeys.operationalHealth(),
    queryFn: ({ signal }) => api.getOperationalHealth(signal),
    refetchInterval: () => document.visibilityState === "visible" ? 30_000 : false,
  });
  const health = healthQuery.data;
  const inFlight = (health?.queues.webhook_pending ?? 0) + (health?.queues.webhook_retry ?? 0);

  return <div className="space-y-8">
    <div>
      <h1 className="text-2xl font-semibold tracking-tight">{t("operationsHealth.title")}</h1>
      <p className="text-muted-foreground">{t("operationsHealth.description")}</p>
    </div>

    {healthQuery.error && <Alert variant="destructive">
      <AlertDescription>{apiErrorMessage(healthQuery.error, t)}</AlertDescription>
    </Alert>}

    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <StatCard icon={<Activity className="size-4" />} label={t("operationsHealth.overall")} value={t(`operationsHealth.status.${health?.status ?? "unavailable"}` as TranslationKey)} />
      <StatCard icon={<Database className="size-4" />} label={t("operationsHealth.dependencies")} value={health?.components.length ?? 0} />
      <StatCard icon={<RefreshCw className="size-4" />} label={t("operationsHealth.inFlight")} value={inFlight} />
      <StatCard icon={<CircleAlert className="size-4" />} label={t("operationsHealth.deadLetters")} value={health?.queues.webhook_dead_letter ?? 0} />
    </div>

    <Card>
      <CardHeader><CardTitle>{t("operationsHealth.dependencies")}</CardTitle></CardHeader>
      <CardContent>
        <Table>
          <TableHeader><TableRow>
            <TableHead>{t("operationsHealth.component")}</TableHead>
            <TableHead>{t("common.status")}</TableHead>
            <TableHead>{t("operationsHealth.latency")}</TableHead>
          </TableRow></TableHeader>
          <TableBody>
            {health?.components.map((component) => <TableRow key={component.name}>
              <TableCell className="font-medium">{component.name}</TableCell>
              <TableCell><Badge variant={statusVariant(component.status)}>{t(`operationsHealth.status.${component.status}` as TranslationKey)}</Badge></TableCell>
              <TableCell className="tabular-nums">{component.latency_ms} ms</TableCell>
            </TableRow>)}
            {!health?.components.length && <TableRow><TableCell colSpan={3} className="text-center text-muted-foreground">{t("operationsHealth.noDependencies")}</TableCell></TableRow>}
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <Card>
      <CardHeader><CardTitle>{t("operationsHealth.reports")}</CardTitle></CardHeader>
      <CardContent>
        <Table>
          <TableHeader><TableRow>
            <TableHead>{t("operationsHealth.kind")}</TableHead>
            <TableHead>{t("operationsHealth.source")}</TableHead>
            <TableHead>{t("common.status")}</TableHead>
            <TableHead>{t("operationsHealth.occurredAt")}</TableHead>
            <TableHead>{t("operationsHealth.freshUntil")}</TableHead>
          </TableRow></TableHeader>
          <TableBody>
            {health?.reports.map((report) => <TableRow key={report.id}>
              <TableCell className="font-medium">{t(`operationsHealth.kind.${report.kind}` as TranslationKey)}</TableCell>
              <TableCell><code className="text-xs">{report.source}</code></TableCell>
              <TableCell><Badge variant={statusVariant(report.effective_status)}>{t(`operationsHealth.status.${report.effective_status}` as TranslationKey)}</Badge></TableCell>
              <TableCell className="tabular-nums">{new Date(report.occurred_at).toLocaleString(locale)}</TableCell>
              <TableCell className="tabular-nums">{new Date(report.fresh_until).toLocaleString(locale)}</TableCell>
            </TableRow>)}
            {!health?.reports.length && <TableRow><TableCell colSpan={5} className="text-center text-muted-foreground">{t("operationsHealth.noReports")}</TableCell></TableRow>}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  </div>;
}
