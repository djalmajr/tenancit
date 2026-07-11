import { useMutation, useQuery } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { Download, FileClock, Gavel, ShieldCheck } from "lucide-react";
import { useMemo, useState } from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AuditLegalHoldsDialog } from "@/components/audit-legal-holds-dialog";
import { DomainStatus } from "@/components/domain-status";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Combobox } from "@/components/ui/combobox";
import { Input } from "@/components/ui/input";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { useDataTable } from "@/hooks/use-data-table";
import { api, downloadAuditExport, getAdminSession, type AdminAuditEvent, type AuditExportJob } from "@/lib/api";
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
  const { t } = useI18n();
  const session = getAdminSession();
  const canExport = !session || session.permissions.includes("audit.export");
  const [cursor, setCursor] = useState("");
  const [cursorHistory, setCursorHistory] = useState<string[]>([]);
  const [actionDraft, setActionDraft] = useState("");
  const [targetDraft, setTargetDraft] = useState("");
  const [resultDraft, setResultDraft] = useState("");
  const [filters, setFilters] = useState({ action: "", target: "", result: "" });
  const [exportJob, setExportJob] = useState<AuditExportJob>();
  const [auditWindow, setAuditWindow] = useState(() => {
    const end = new Date();
    return { from: new Date(end.getTime() - 24 * 60 * 60 * 1000).toISOString(), to: end.toISOString() };
  });
  const { from, to } = auditWindow;
  const query = new URLSearchParams({ limit: "50", from, to });
  if (cursor) query.set("cursor", cursor);
  if (filters.action) query.set("action", filters.action);
  if (filters.target) query.set("target_id", filters.target);
  if (filters.result) query.set("result", filters.result);
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
    mutationFn: () => api.createAuditExport({ filters: { from, to, action: filters.action || undefined, target_type: undefined, result: filters.result || undefined }, format: "csv" }, crypto.randomUUID()),
    onSuccess: setExportJob,
  });
  const events = eventsQuery.data?.events ?? EMPTY_EVENTS;
  const sortLabels = useMemo(() => ({ asc: t("dataTable.sortAsc"), desc: t("dataTable.sortDesc"), reset: t("dataTable.sortReset") }), [t]);
  const labels = useMemo(() => ({
    goToFirstPage: t("dataTable.firstPage"), goToLastPage: t("dataTable.lastPage"), goToNextPage: t("dataTable.nextPage"), goToPreviousPage: t("dataTable.previousPage"),
    item: t("dataTable.item"), items: t("dataTable.items"), noResults: t("audit.empty"), page: t("dataTable.page"), pageOf: t("dataTable.pageOf"), rowsPerPage: t("dataTable.rowsPerPage"),
  }), [t]);
  const columns: ColumnDef<AdminAuditEvent>[] = [
    { accessorKey: "occurred_at", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.when")} labels={sortLabels} />, cell: ({ row }) => new Date(row.original.occurred_at).toLocaleString(), meta: { label: t("audit.when") }, size: 170 },
    { accessorFn: (event) => `${event.actor_kind}/${event.actor_subject}`, id: "actor", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.actor")} labels={sortLabels} />, cell: ({ row }) => <code className="text-xs">{row.getValue("actor")}</code>, meta: { label: t("audit.actor") }, size: 210 },
    { accessorKey: "action", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.action")} labels={sortLabels} />, meta: { label: t("audit.action") }, size: 210 },
    { accessorFn: (event) => `${event.target_type}:${event.target_id}`, id: "target", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.target")} labels={sortLabels} />, cell: ({ row }) => <code className="text-xs">{row.getValue("target")}</code>, meta: { label: t("audit.target") }, size: 360 },
    { accessorKey: "result", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.result")} labels={sortLabels} />, cell: ({ row }) => <DomainStatus label={row.original.result} value={row.original.result} />, meta: { label: t("audit.result") }, size: 120 },
    { accessorKey: "request_id", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.request")} labels={sortLabels} />, cell: ({ row }) => <code className="text-xs">{row.original.request_id}</code>, meta: { label: t("audit.request") }, size: 240 },
  ];
  const { table } = useDataTable({ columns, data: events, initialState: { pagination: { pageIndex: 0, pageSize: 10 }, sorting: [{ desc: true, id: "occurred_at" }] }, visibilityStorageKey: "tenancit.audit.columns" });
  const error = eventsQuery.error || healthQuery.error || exportMutation.error ? apiErrorMessage(eventsQuery.error ?? healthQuery.error ?? exportMutation.error, t) : "";

  function applyFilters() {
    const end = new Date();
    setAuditWindow({ from: new Date(end.getTime() - 24 * 60 * 60 * 1000).toISOString(), to: end.toISOString() });
    setFilters({ action: actionDraft.trim(), target: targetDraft.trim(), result: resultDraft });
    setCursor(""); setCursorHistory([]);
  }

  async function consumeExport(job: AuditExportJob) {
    const blob = await downloadAuditExport(job.id);
    downloadBlob(blob, job.format);
    setExportJob({ ...job, status: "expired", downloaded_at: new Date().toISOString() });
  }

  return <div className="flex flex-col gap-8">
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div><h1 className="text-2xl font-semibold tracking-tight">{t("audit.title")}</h1><p className="text-muted-foreground">{t("audit.description")}</p></div>
      <div className="flex flex-wrap gap-2"><AuditLegalHoldsDialog />{canExport && <Button disabled={exportMutation.isPending} onClick={() => exportMutation.mutate()}><Download />{t("audit.export")}</Button>}</div>
    </div>
    <p className="-mt-4 text-sm text-muted-foreground">{t("audit.actorNotice")}</p>
    <div className="grid gap-4 sm:grid-cols-3">
      <Card><CardHeader className="flex-row items-center justify-between"><CardTitle>{t("audit.partitionHealth")}</CardTitle><ShieldCheck className="size-4" /></CardHeader><CardContent className="text-2xl font-semibold">{healthQuery.data?.future_through ? new Date(healthQuery.data.future_through).toLocaleDateString() : "—"}</CardContent></Card>
      <Card><CardHeader className="flex-row items-center justify-between"><CardTitle>{t("audit.defaultRows")}</CardTitle><FileClock className="size-4" /></CardHeader><CardContent className="text-2xl font-semibold">{healthQuery.data?.default_rows ?? "—"}</CardContent></Card>
      <Card><CardHeader className="flex-row items-center justify-between"><CardTitle>{t("audit.legalHolds")}</CardTitle><Gavel className="size-4" /></CardHeader><CardContent className="text-2xl font-semibold">{healthQuery.data?.active_legal_holds ?? "—"}</CardContent></Card>
    </div>
    {currentExport && <Alert><AlertDescription className="flex flex-wrap items-center justify-between gap-3"><span>{t("audit.exportStatus", { status: currentExport.status, rows: currentExport.row_count ?? 0 })}</span>{currentExport.status === "ready" && <Button size="sm" onClick={() => { void consumeExport(currentExport); }}><Download />{t("audit.download")}</Button>}</AlertDescription></Alert>}
    {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
    <div className="grid gap-3 md:grid-cols-[1fr_1fr_14rem_auto]">
      <Input aria-label={t("audit.action")} onChange={(event) => setActionDraft(event.target.value)} placeholder={t("audit.action")} value={actionDraft} />
      <Input aria-label={t("audit.target")} onChange={(event) => setTargetDraft(event.target.value)} placeholder={t("audit.target")} value={targetDraft} />
      <Combobox aria-label={t("audit.result")} options={[{ value: "", label: t("audit.allResults") }, ...["success", "denied", "error"].map((value) => ({ value, label: value }))]} onValueChange={setResultDraft} value={resultDraft} />
      <Button onClick={applyFilters} variant="outline">{t("audit.applyFilters")}</Button>
    </div>
    {eventsQuery.isPending ? <div className="rounded-md border p-4 text-sm text-muted-foreground" role="status">{t("common.loading")}</div> : error && events.length === 0 ? null : <DataTable labels={labels} table={table}>
      <DataTableToolbar clearLabel={t("dataTable.clearFilters")} columnsLabel={t("dataTable.columns")} emptyLabel={t("dataTable.noResults")} resetLabel={t("dataTable.resetPreferences")} searchLabel={`${t("audit.action")}, ${t("audit.target")}, ${t("audit.request")}`} table={table}>
        <div className="flex gap-2">
          <Button disabled={cursorHistory.length === 0} onClick={() => { const history = [...cursorHistory]; setCursor(history.pop() ?? ""); setCursorHistory(history); }} variant="outline">{t("audit.previous")}</Button>
          <Button disabled={!eventsQuery.data?.next_cursor} onClick={() => { setCursorHistory((history) => [...history, cursor]); setCursor(eventsQuery.data?.next_cursor ?? ""); }} variant="outline">{t("audit.next")}</Button>
        </div>
      </DataTableToolbar>
    </DataTable>}
  </div>;
}
