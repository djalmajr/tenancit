import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { useMemo, useState } from "react";
import { LogOut, Shield, UserRound } from "lucide-react";
import { toast } from "sonner";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { useDataTable } from "@/hooks/use-data-table";
import { api, type AdminSessionView } from "@/lib/api";
import { apiErrorMessage, useI18n, type TranslationKey } from "@/lib/i18n";
import { adminQueryKeys } from "@/lib/query-keys";

const EMPTY_SESSIONS: AdminSessionView[] = [];

export default function SessionsPage() {
  const { locale, t } = useI18n();
  const queryClient = useQueryClient();
  const sessionsQuery = useQuery({
    queryKey: adminQueryKeys.sessions(),
    queryFn: ({ signal }) => api.listAdminSessions(signal),
    refetchInterval: 30_000,
  });
  const sessions = sessionsQuery.data ?? EMPTY_SESSIONS;
  const [revokeTarget, setRevokeTarget] = useState<AdminSessionView>();
  const [pending, setPending] = useState(false);
  const sortLabels = useMemo(() => ({
    asc: t("dataTable.sortAsc"), desc: t("dataTable.sortDesc"), reset: t("dataTable.sortReset"),
  }), [t]);
  const labels = useMemo(() => ({
    goToFirstPage: t("dataTable.firstPage"), goToLastPage: t("dataTable.lastPage"),
    goToNextPage: t("dataTable.nextPage"), goToPreviousPage: t("dataTable.previousPage"),
    item: t("dataTable.item"), items: t("dataTable.items"), noResults: t("sessions.empty"),
    page: t("dataTable.page"), pageOf: t("dataTable.pageOf"), rowsPerPage: t("dataTable.rowsPerPage"),
  }), [t]);

  const columns: ColumnDef<AdminSessionView>[] = [
    {
      accessorFn: (session) => session.label || session.subject,
      id: "principal",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("sessions.principal")} labels={sortLabels} />,
      cell: ({ row }) => <div className="flex min-w-48 items-center gap-2"><span className="font-medium">{row.original.label || row.original.subject}</span>{row.original.current && <Badge variant="secondary">{t("sessions.current")}</Badge>}</div>,
      meta: { label: t("sessions.principal") },
    },
    {
      accessorKey: "roles",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("sessions.roles")} labels={sortLabels} />,
      cell: ({ row }) => <div className="flex gap-1">{row.original.roles.map((role) => <Badge key={role} variant="outline">{role}</Badge>)}</div>,
      meta: { label: t("sessions.roles") },
    },
    {
      accessorKey: "last_used_at",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("sessions.lastUsed")} labels={sortLabels} />,
      cell: ({ row }) => <span className="tabular-nums text-muted-foreground">{new Date(row.original.last_used_at).toLocaleString(locale)}</span>,
      meta: { label: t("sessions.lastUsed") }, size: 190,
    },
    {
      accessorKey: "expires_at",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("sessions.expires")} labels={sortLabels} />,
      cell: ({ row }) => <span className="tabular-nums text-muted-foreground">{new Date(row.original.expires_at).toLocaleString(locale)}</span>,
      meta: { label: t("sessions.expires") }, size: 190,
    },
    {
      accessorKey: "status",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("common.status")} labels={sortLabels} />,
      cell: ({ row }) => <Badge variant={row.original.status === "active" ? "default" : "secondary"}>{t(`sessions.status.${row.original.status}` as TranslationKey)}</Badge>,
      meta: { label: t("common.status") }, size: 130,
    },
    {
      id: "actions", enableHiding: false, enableSorting: false, size: 70,
      header: () => <span className="sr-only">{t("common.actions")}</span>,
      cell: ({ row }) => row.original.current || row.original.status !== "active" ? null : <Button aria-label={t("sessions.revoke")} onClick={() => setRevokeTarget(row.original)} size="icon-sm" variant="ghost"><LogOut className="size-4" /></Button>,
    },
  ];
  const { table } = useDataTable({
    columns, data: sessions, initialState: { pagination: { pageIndex: 0, pageSize: 10 }, sorting: [{ id: "last_used_at", desc: true }] },
    visibilityStorageKey: "tenancit.sessions.columns",
  });
  const error = sessionsQuery.error ? apiErrorMessage(sessionsQuery.error, t) : "";

  async function revoke() {
    if (!revokeTarget) return;
    setPending(true);
    try {
      await api.revokeAdminSession(revokeTarget.id);
      toast.success(t("sessions.revoked"));
      setRevokeTarget(undefined);
      await queryClient.invalidateQueries({ queryKey: adminQueryKeys.sessions() });
    } catch (cause) {
      toast.error(apiErrorMessage(cause, t));
    } finally {
      setPending(false);
    }
  }

  return <div className="flex flex-col gap-8">
    {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
    {sessionsQuery.isPending ? <div className="rounded-md border p-4 text-sm text-muted-foreground" role="status">{t("common.loading")}</div> : <DataTable labels={labels} table={table}>
      <DataTableToolbar clearLabel={t("dataTable.clearFilters")} columnsLabel={t("dataTable.columns")} emptyLabel={t("sessions.empty")} resetLabel={t("dataTable.resetPreferences")} searchLabel={t("sessions.search")} table={table} />
    </DataTable>}
    <ConfirmDialog cancelLabel={t("common.cancel")} confirmDisabled={pending} confirmLabel={t("sessions.revoke")} description={t("sessions.revokeDescription", { name: revokeTarget?.label || revokeTarget?.subject || "" })} onConfirm={() => void revoke()} onOpenChange={(open) => { if (!open) setRevokeTarget(undefined); }} open={Boolean(revokeTarget)} title={t("sessions.revokeTitle")}>
      <div className="flex items-center gap-2 rounded-md bg-muted p-3 text-sm"><Shield className="size-4" /><UserRound className="size-4" />{revokeTarget?.issuer} / {revokeTarget?.subject}</div>
    </ConfirmDialog>
  </div>;
}
