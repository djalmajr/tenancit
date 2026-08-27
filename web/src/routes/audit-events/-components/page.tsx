import { useMutation, useQuery } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { Download, FileClock, Gavel, ShieldCheck } from "lucide-react";
import { useMemo, useState } from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AuditLegalHoldsDialog } from "./audit-legal-holds-dialog";
import { DomainStatus } from "@/components/domain-status";
import { Button } from "@/components/ui/button";
import { StatCard } from "@/components/stat-card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { useDataTable } from "@/hooks/use-data-table";
import { api, downloadAuditExport, getAdminSession, type AdminAuditEvent, type AuditExportJob } from "@/lib/api";
import { formatDate } from "@/lib/date-format";
import { apiErrorMessage, useI18n } from "@/lib/i18n";

const EMPTY_EVENTS: AdminAuditEvent[] = [];

function downloadBlob(blob: Blob, format: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `tenancit-audit.${format}`;
  link.click();
  URL.revokeObjectURL(url);
}

export default function AuditEventsPage() {
  const { locale, t } = useI18n();
  const session = getAdminSession();
  const canExport = !session || session.permissions.includes("audit.export");
  const [exportJob, setExportJob] = useState<AuditExportJob>();
  const [auditWindow] = useState(() => {
    const end = new Date();
    return { from: new Date(end.getTime() - 24 * 60 * 60 * 1000).toISOString(), to: end.toISOString() };
  });
  const { from, to } = auditWindow;
  const query = new URLSearchParams({ limit: "200", from, to });
  const eventsQuery = useQuery({ queryKey: ["admin", "audit-events", query.toString()], queryFn: ({ signal }) => api.listAuditEvents(query, signal) });
  const healthQuery = useQuery({ queryKey: ["admin", "audit-health"], queryFn: ({ signal }) => api.getAuditHealth(signal) });
  const exportStatusQuery = useQuery({
    queryKey: ["admin", "audit-export", exportJob?.id],
    queryFn: ({ signal }) => api.getAuditExport(exportJob?.id ?? "", signal),
    enabled: Boolean(exportJob && !["ready", "failed", "expired"].includes(exportJob.status)),
    refetchInterval: 2000,
  });
  const currentExport = exportStatusQuery.data ?? exportJob;
  const exportMutation = useMutation({
    mutationFn: () => api.createAuditExport({ filters: { from, to, target_type: undefined }, format: "csv" }, crypto.randomUUID()),
    onSuccess: setExportJob,
  });
  const events = eventsQuery.data?.events ?? EMPTY_EVENTS;
  const sortLabels = useMemo(() => ({ asc: t("dataTable.sortAsc"), desc: t("dataTable.sortDesc"), reset: t("dataTable.sortReset") }), [t]);
  const labels = useMemo(() => ({
    goToFirstPage: t("dataTable.firstPage"), goToLastPage: t("dataTable.lastPage"), goToNextPage: t("dataTable.nextPage"), goToPreviousPage: t("dataTable.previousPage"),
    item: t("dataTable.item"), items: t("dataTable.items"), noResults: t("audit.empty"), page: t("dataTable.page"), pageOf: t("dataTable.pageOf"), rowsPerPage: t("dataTable.rowsPerPage"),
  }), [t]);
  const columns: ColumnDef<AdminAuditEvent>[] = [
    { accessorKey: "occurred_at", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.when")} labels={sortLabels} />, cell: ({ row }) => new Date(row.original.occurred_at).toLocaleString(locale), meta: { label: t("audit.when") }, size: 170 },
    { accessorFn: (event) => `${event.actor_kind}/${event.actor_subject}`, id: "actor", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.actor")} labels={sortLabels} />, cell: ({ row }) => <code className="text-xs">{row.getValue("actor")}</code>, meta: { label: t("audit.actor") }, size: 210 },
    { accessorKey: "action", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.action")} labels={sortLabels} />, meta: { label: t("audit.action") }, size: 210 },
    { accessorFn: (event) => `${event.target_type}:${event.target_id}`, id: "target", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.target")} labels={sortLabels} />, cell: ({ row }) => <code className="text-xs">{row.getValue("target")}</code>, meta: { label: t("audit.target") }, size: 360 },
    { accessorKey: "result", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.result")} labels={sortLabels} />, cell: ({ row }) => <DomainStatus label={row.original.result} value={row.original.result} />, filterFn: (row, _id, value) => (value as string[]).includes(row.original.result), meta: { label: t("audit.result") }, size: 120 },
    { accessorKey: "request_id", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.request")} labels={sortLabels} />, cell: ({ row }) => <code className="text-xs">{row.original.request_id}</code>, meta: { label: t("audit.request") }, size: 240 },
  ];
  const { table } = useDataTable({ columns, data: events, initialState: { pagination: { pageIndex: 0, pageSize: 10 }, sorting: [{ desc: true, id: "occurred_at" }] }, visibilityStorageKey: "tenancit.audit.columns" });
  const overviewError = healthQuery.error ? apiErrorMessage(healthQuery.error, t) : "";
  const eventsError = eventsQuery.error || exportMutation.error ? apiErrorMessage(eventsQuery.error ?? exportMutation.error, t) : "";

  async function consumeExport(job: AuditExportJob) {
    const blob = await downloadAuditExport(job.id);
    downloadBlob(blob, job.format);
    setExportJob({ ...job, status: "expired", downloaded_at: new Date().toISOString() });
  }

  return <Tabs className="gap-4" defaultValue="overview">
    <TabsList>
      <TabsTrigger value="overview">{t("audit.overviewTab")}</TabsTrigger>
      <TabsTrigger value="events">{t("audit.eventsTab")}</TabsTrigger>
    </TabsList>
    <TabsContent className="flex flex-col gap-6" value="overview">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">{t("audit.actorNotice")}</p>
        <AuditLegalHoldsDialog />
      </div>
      {overviewError && <Alert variant="destructive"><AlertDescription>{overviewError}</AlertDescription></Alert>}
      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard hint={t("audit.partitionHealthHint")} icon={<ShieldCheck className="size-4" />} label={t("audit.partitionHealth")} value={healthQuery.data?.future_through ? formatDate(locale, healthQuery.data.future_through, { timeZone: "UTC" }) : "—"} />
        <StatCard hint={t("audit.defaultRowsHint")} icon={<FileClock className="size-4" />} label={t("audit.defaultRows")} value={healthQuery.data?.default_rows ?? "—"} />
        <StatCard hint={t("audit.legalHoldsHint")} icon={<Gavel className="size-4" />} label={t("audit.legalHolds")} value={healthQuery.data?.active_legal_holds ?? "—"} />
      </div>
    </TabsContent>
    <TabsContent className="flex flex-col gap-6" value="events">
      {currentExport && <Alert><AlertDescription className="flex flex-wrap items-center justify-between gap-3"><span>{t("audit.exportStatus", { status: currentExport.status, rows: currentExport.row_count ?? 0 })}</span>{currentExport.status === "ready" && <Button size="sm" onClick={() => { void consumeExport(currentExport); }}><Download />{t("audit.download")}</Button>}</AlertDescription></Alert>}
      {eventsError && <Alert variant="destructive"><AlertDescription>{eventsError}</AlertDescription></Alert>}
      {eventsQuery.isPending ? <div className="rounded-md border p-4 text-sm text-muted-foreground" role="status">{t("common.loading")}</div> : eventsError && events.length === 0 ? null : <DataTable labels={labels} table={table}>
        <DataTableToolbar
          clearLabel={t("dataTable.clearFilters")}
          columnsLabel={t("dataTable.columns")}
          emptyLabel={t("dataTable.noResults")}
          facets={[{ columnId: "result", multiple: true, options: ["success", "denied", "error"].map((value) => ({ value, label: value })), title: t("audit.result") }]}
          resetLabel={t("dataTable.resetPreferences")}
          searchLabel={`${t("audit.action")}, ${t("audit.target")}, ${t("audit.request")}`}
          table={table}
        >
          {canExport && <Button aria-label={t("audit.export")} disabled={exportMutation.isPending} onClick={() => exportMutation.mutate()} size="icon" title={t("audit.export")} variant="outline"><Download /></Button>}
        </DataTableToolbar>
      </DataTable>}
    </TabsContent>
  </Tabs>;
}
