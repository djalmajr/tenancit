import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { useCallback, useMemo, useRef, useState } from "react";
import { Plus, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DomainStatus } from "@/components/domain-status";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { apiErrorMessage, formatStatus, useI18n } from "@/lib/i18n";
import { api, type Tenant } from "@/lib/api";
import { stableIdempotencyKey, type IdempotencyAttempt } from "@/lib/idempotency";
import { useDataTable } from "@/hooks/use-data-table";
import { invalidateTenants } from "@/lib/query-invalidation";
import { adminQueryOptions } from "@/lib/query-options";
import { useAdminCapabilities } from "@/hooks/use-admin-capabilities";

const EMPTY_TENANTS: Tenant[] = [];

export default function Tenants() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { can } = useAdminCapabilities();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ slug: "", name: "" });
  const [error, setError] = useState("");
  const createAttempt = useRef<IdempotencyAttempt>(null);
  const tenantsQuery = useQuery(adminQueryOptions.tenants());
  // TanStack Table treats a new data reference as a data change and schedules
  // pagination resets. Keep the loading fallback stable across renders so an
  // interaction while the query is in flight cannot enter a render loop.
  const tenants = tenantsQuery.data ?? EMPTY_TENANTS;
  const createTenantMutation = useMutation({
    mutationFn: ({ body, key }: { body: { slug: string; name: string }; key: string }) => api.createTenant(body, key),
  });
  const pageError = tenantsQuery.error ? apiErrorMessage(tenantsQuery.error, t) : "";
  const sortLabels = useMemo(() => ({
    asc: t("dataTable.sortAsc"),
    desc: t("dataTable.sortDesc"),
    reset: t("dataTable.sortReset"),
  }), [t]);
  const dataTableLabels = useMemo(() => ({
    goToFirstPage: t("dataTable.firstPage"),
    goToLastPage: t("dataTable.lastPage"),
    goToNextPage: t("dataTable.nextPage"),
    goToPreviousPage: t("dataTable.previousPage"),
    item: t("dataTable.item"),
    items: t("dataTable.items"),
    noResults: t("tenants.empty"),
    page: t("dataTable.page"),
    pageOf: t("dataTable.pageOf"),
    rowsPerPage: t("dataTable.rowsPerPage"),
  }), [t]);
  const columns = useMemo<ColumnDef<Tenant>[]>(() => [
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} label={t("common.name")} labels={sortLabels} />
      ),
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      meta: { label: t("common.name") },
    },
    {
      accessorKey: "slug",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} label={t("common.slug")} labels={sortLabels} />
      ),
      cell: ({ row }) => <code className="text-xs">{row.original.slug}</code>,
      meta: { label: t("common.slug") },
    },
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} label={t("common.status")} labels={sortLabels} />
      ),
      cell: ({ row }) => (
        <DomainStatus label={formatStatus(row.original.status, t)} value={row.original.status} />
      ),
      meta: { label: t("common.status") },
    },
    {
      enableSorting: false,
      header: "",
      id: "actions",
      cell: () => (
        <div className="flex justify-end">
          <ChevronRight className="size-4 text-muted-foreground" />
        </div>
      ),
      meta: { align: "right", label: t("common.actions") },
      size: 40,
    },
  ], [sortLabels, t]);
  const filterTenants = useCallback((tenant: Tenant, filterValue: string) => {
    const query = filterValue.trim().toLowerCase();
    if (!query) return true;
    const statusValues = [tenant.status, formatStatus(tenant.status, t)]
      .map((value) => value.toLowerCase());
    return tenant.name.toLowerCase().includes(query)
      || tenant.slug.toLowerCase().includes(query)
      || statusValues.includes(query);
  }, [t]);
  const initialTableState = useMemo(() => ({
    sorting: [{ desc: false, id: "name" }],
  }), []);
  const dataTable = useDataTable({
    columns,
    data: tenants,
    globalFilterFn: filterTenants,
    initialState: initialTableState,
    visibilityStorageKey: "tenancit.tenants.table",
  });
  const { table } = dataTable;

  async function create() {
    setError("");
    try {
      const created = await createTenantMutation.mutateAsync({ body: form, key: stableIdempotencyKey(createAttempt, form) });
      await invalidateTenants(queryClient);
      setForm({ slug: "", name: "" });
      setOpen(false);
      await navigate({ to: "/tenants/$id", params: { id: created.id } });
      createAttempt.current = null;
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }

  return (
    <div className="space-y-6">
      {pageError && <Alert variant="destructive"><AlertDescription>{pageError}</AlertDescription></Alert>}

      {tenantsQuery.isPending ? <div className="rounded-md border p-4 text-sm text-muted-foreground" role="status">{t("common.loading")}</div> : pageError && tenants.length === 0 ? null : <DataTable
        labels={dataTableLabels}
        onRowClick={(tenant) => { void navigate({ to: "/tenants/$id", params: { id: tenant.id } }); }}
        table={table}
      >
        <DataTableToolbar
          clearLabel={t("dataTable.clearFilters")}
          columnsLabel={t("dataTable.columns")}
          emptyLabel={t("dataTable.noResults")}
          resetLabel={t("dataTable.resetPreferences")}
          searchLabel={t("tenants.search")}
          table={table}
          trailing={can("tenant.write") ? <Button onClick={() => { setError(""); setOpen(true); }}><Plus className="size-4" /> {t("tenants.new")}</Button> : undefined}
        />
      </DataTable>}

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("tenants.newDialog.title")}</DialogTitle>
            <DialogDescription>{t("tenants.newDialog.description")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="new-tenant-name">{t("common.name")}</label>
              <Input id="new-tenant-name" placeholder="Acme Corp" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="new-tenant-slug">{t("common.slug")}</label>
              <Input id="new-tenant-slug" placeholder="acme" value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value })} />
            </div>
            {error && <div className="text-sm text-destructive">{error}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!form.slug || !form.name || createTenantMutation.isPending} onClick={() => { void create(); }}>{t("tenants.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
