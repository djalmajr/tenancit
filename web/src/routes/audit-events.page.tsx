import { useQuery } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { DomainStatus } from "@/components/domain-status";
import { Button } from "@/components/ui/button";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { useDataTable } from "@/hooks/use-data-table";
import { api, type AdminAuditEvent } from "@/lib/api";
import { apiErrorMessage, useI18n } from "@/lib/i18n";

const EMPTY_EVENTS: AdminAuditEvent[] = [];

export default function AuditEventsPage() {
  const { t } = useI18n();
  const [cursor, setCursor] = useState("");
  const query = new URLSearchParams({ limit: "200" });
  if (cursor) query.set("cursor", cursor);
  const eventsQuery = useQuery({ queryKey: ["admin", "audit-events", query.toString()], queryFn: ({ signal }) => api.listAuditEvents(query, signal) });
  const events = eventsQuery.data?.events ?? EMPTY_EVENTS;
  const sortLabels = useMemo(() => ({ asc: t("dataTable.sortAsc"), desc: t("dataTable.sortDesc"), reset: t("dataTable.sortReset") }), [t]);
  const labels = useMemo(() => ({
    goToFirstPage: t("dataTable.firstPage"), goToLastPage: t("dataTable.lastPage"), goToNextPage: t("dataTable.nextPage"), goToPreviousPage: t("dataTable.previousPage"),
    item: t("dataTable.item"), items: t("dataTable.items"), noResults: t("audit.empty"), page: t("dataTable.page"), pageOf: t("dataTable.pageOf"), rowsPerPage: t("dataTable.rowsPerPage"),
  }), [t]);
  const columns: ColumnDef<AdminAuditEvent>[] = [
    { accessorKey: "occurred_at", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.when")} labels={sortLabels} />, cell: ({ row }) => row.original.occurred_at.slice(0, 19).replace("T", " "), meta: { label: t("audit.when") }, size: 150 },
    { accessorFn: (event) => `${event.actor_kind}/${event.actor_subject}`, id: "actor", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.actor")} labels={sortLabels} />, cell: ({ row }) => <code className="text-xs">{row.getValue("actor")}</code>, meta: { label: t("audit.actor") }, size: 190 },
    { accessorKey: "action", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.action")} labels={sortLabels} />, meta: { label: t("audit.action") }, size: 190 },
    { accessorFn: (event) => `${event.target_type}:${event.target_id}`, id: "target", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.target")} labels={sortLabels} />, cell: ({ row }) => <code className="text-xs">{row.getValue("target")}</code>, meta: { label: t("audit.target") }, size: 360 },
    { accessorKey: "result", filterFn: (row, _id, value) => (value as string[]).includes(row.original.result), header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.result")} labels={sortLabels} />, cell: ({ row }) => <DomainStatus label={row.original.result} value={row.original.result} />, meta: { label: t("audit.result") }, size: 120 },
    { accessorKey: "request_id", header: ({ column }) => <DataTableColumnHeader column={column} label={t("audit.request")} labels={sortLabels} />, cell: ({ row }) => <code className="text-xs">{row.original.request_id}</code>, meta: { label: t("audit.request") }, size: 240 },
  ];
  const { table } = useDataTable({ columns, data: events, initialState: { pagination: { pageIndex: 0, pageSize: 10 }, sorting: [{ desc: true, id: "occurred_at" }] }, visibilityStorageKey: "tenancit.audit.columns" });
  const error = eventsQuery.error ? apiErrorMessage(eventsQuery.error, t) : "";

  return <div className="flex flex-col gap-8">
    <div><h1 className="text-2xl font-semibold tracking-tight">{t("audit.title")}</h1><p className="text-muted-foreground">{t("audit.description")}</p></div>
    <p className="-mt-4 text-sm text-muted-foreground">{t("audit.actorNotice")}</p>
    {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
    {eventsQuery.isPending ? <div className="rounded-md border p-4 text-sm text-muted-foreground" role="status">{t("common.loading")}</div> : error && events.length === 0 ? null : <DataTable labels={labels} table={table}>
      <DataTableToolbar clearLabel={t("dataTable.clearFilters")} columnsLabel={t("dataTable.columns")} emptyLabel={t("dataTable.noResults")} facets={[{ columnId: "result", multiple: true, options: ["success", "denied", "error"].map((value) => ({ value, label: value })), title: t("audit.result") }]} searchLabel={`${t("audit.action")}, ${t("audit.target")}, ${t("audit.request")}`} table={table}>
        {eventsQuery.data?.next_cursor && <Button onClick={() => setCursor(eventsQuery.data?.next_cursor ?? "")} variant="outline">{t("audit.next")}</Button>}
      </DataTableToolbar>
    </DataTable>}
  </div>;
}
