import { useQuery } from "@tanstack/react-query";
import { Activity, Ban, BarChart3, CircleAlert } from "lucide-react";
import { useMemo, useState } from "react";
import { Bar, BarChart, CartesianGrid, XAxis } from "recharts";
import { MonthYearNav } from "./month-year-nav";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Card, CardContent } from "@/components/ui/card";
import { StatCard } from "@/components/stat-card";
import { ChartContainer, ChartLegend, ChartLegendContent, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";
import { Combobox } from "@/components/ui/combobox";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { api } from "@/lib/api";
import { apiErrorMessage, useI18n } from "@/lib/i18n";
import { adminQueryOptions } from "@/lib/query-options";
import { adminQueryKeys } from "@/lib/query-keys";

const EMPTY_USAGE: Awaited<ReturnType<typeof api.listAPIClientUsage>> = [];
const EMPTY_CLIENTS: Awaited<ReturnType<typeof api.listAPIClients>> = [];
const ALL = "all";
const monthToString = (date: Date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
const stringToMonth = (month: string) => { const [year, value] = month.split("-").map(Number); return new Date(year, value - 1, 1); };
function monthRange(month: string) { const [year, value] = month.split("-").map(Number); return { from: `${month}-01`, to: new Date(Date.UTC(year, value, 0)).toISOString().slice(0, 10) }; }

export default function UsagePage() {
  const { t } = useI18n();
  const [month, setMonth] = useState(() => monthToString(new Date()));
  const [clientID, setClientID] = useState(ALL);
  const [operation, setOperation] = useState(ALL);
  const range = monthRange(month);
  const usageQuery = useQuery({
    queryKey: adminQueryKeys.apiClientUsage(range.from, range.to),
    queryFn: ({ signal }) => api.listAPIClientUsage(range.from, range.to, signal),
    refetchInterval: () => document.visibilityState === "visible" ? 60_000 : false,
  });
  const apiClientsQuery = useQuery(adminQueryOptions.apiClients());
  const rows = usageQuery.data ?? EMPTY_USAGE;
  const clients = apiClientsQuery.data ?? EMPTY_CLIENTS;
  const selectedClientID = clientID === ALL || clients.some((client) => client.id === clientID) ? clientID : ALL;
  const filtered = rows.filter((row) => (selectedClientID === ALL || row.api_client_id === selectedClientID) && (operation === ALL || row.operation === operation));
  const clientOptions = useMemo(() => [{ value: ALL, label: t("usage.allClients") }, ...clients.map((client) => ({ value: client.id, label: client.name, keywords: `${client.id} ${client.key_preview ?? ""}` }))], [clients, t]);
  const operationOptions = useMemo(() => [{ value: ALL, label: t("usage.allOperations") }, { value: "identify", label: t("usage.operationIdentify") }, { value: "resolve", label: t("usage.operationResolve") }], [t]);
  const chart = useMemo(() => {
    const [year, monthNumber] = month.split("-").map(Number);
    const daysInMonth = new Date(year, monthNumber, 0).getDate();
    const byDay = new Map<string, { identify: number; resolve: number }>();
    for (const row of filtered) {
      const current = byDay.get(row.day) ?? { identify: 0, resolve: 0 };
      current[row.operation] += row.request_count;
      byDay.set(row.day, current);
    }
    return Array.from({ length: daysInMonth }, (_, index) => {
      const day = String(index + 1).padStart(2, "0");
      return { day, identify: 0, resolve: 0, ...byDay.get(`${month}-${day}`) };
    });
  }, [filtered, month]);
  const totals = filtered.reduce((acc, row) => ({ requests: acc.requests + row.request_count, errors: acc.errors + (row.status_class >= 4 ? row.request_count : 0), limited: acc.limited + row.rate_limited_count }), { requests: 0, errors: 0, limited: 0 });
  const chartConfig: ChartConfig = {
    identify: { label: t("usage.operationIdentify"), color: "var(--chart-1)" },
    resolve: { label: t("usage.operationResolve"), color: "var(--chart-2)" },
  };
  const showIdentify = operation === ALL || operation === "identify";
  const showResolve = operation === ALL || operation === "resolve";
  const chartTotal = chart.reduce((total, day) => total + (showIdentify ? day.identify : 0) + (showResolve ? day.resolve : 0), 0);
  const queryError = usageQuery.error ?? apiClientsQuery.error;
  const error = queryError ? apiErrorMessage(queryError, t) : "";

  return <div className="flex flex-col gap-6">
    <div className="flex flex-wrap items-center gap-2">
      <MonthYearNav value={stringToMonth(month)} onChange={(date) => setMonth(monthToString(date))} />
      <div className="ml-auto flex flex-wrap items-center gap-2">
        <Combobox aria-label={t("usage.client")} options={clientOptions} searchable searchPlaceholder={t("usage.client")} triggerClassName="w-52" value={selectedClientID} onValueChange={setClientID} />
        <Combobox aria-label={t("usage.operation")} options={operationOptions} triggerClassName="w-52" value={operation} onValueChange={setOperation} />
      </div>
    </div>
    {error && <Alert variant="destructive"><CircleAlert /><AlertTitle>{t("errors.server")}</AlertTitle><AlertDescription>{error}</AlertDescription></Alert>}
    <div className="grid gap-4 sm:grid-cols-3">
      <StatCard hint={t("usage.requestsHint")} icon={<Activity className="size-4" />} label={t("usage.requests")} value={totals.requests} />
      <StatCard hint={t("usage.errorsHint")} icon={<CircleAlert className="size-4" />} label={t("usage.errors")} value={totals.errors} />
      <StatCard hint={t("usage.limitedHint")} icon={<Ban className="size-4" />} label={t("usage.limited")} value={totals.limited} />
    </div>
    <Card>
      <CardContent className="pt-6">
        {usageQuery.isPending ? <p className="text-sm text-muted-foreground">{t("common.loading")}</p> : chartTotal === 0 ?
          <Empty><EmptyHeader><EmptyMedia variant="icon"><BarChart3 /></EmptyMedia><EmptyTitle>{t("usage.daily")}</EmptyTitle><EmptyDescription>{t("usage.empty")}</EmptyDescription></EmptyHeader></Empty> :
          <ChartContainer config={chartConfig} className="aspect-auto h-64 w-full">
            <BarChart accessibilityLayer data={chart} margin={{ left: 4, right: 4 }}>
              <CartesianGrid vertical strokeDasharray="2 4" className="stroke-border/60" />
              <XAxis
                axisLine={{ className: "stroke-border" }}
                dataKey="day"
                interval={0}
                minTickGap={0}
                tickFormatter={(value: string) => {
                  const day = Number(value);
                  return day === 1 || day === chart.length || day % 5 === 0 ? value : "";
                }}
                tickLine
                tickMargin={8}
              />
              <ChartTooltip cursor={false} content={<ChartTooltipContent />} />
              <ChartLegend content={<ChartLegendContent />} />
              {showIdentify && <Bar dataKey="identify" fill="var(--color-identify)" radius={2} />}
              {showResolve && <Bar dataKey="resolve" fill="var(--color-resolve)" radius={2} />}
            </BarChart>
          </ChartContainer>}
      </CardContent>
    </Card>
  </div>;
}
