import { createRoute, useNavigate } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Plus, ChevronRight } from "lucide-react";
import { Route as rootRoute } from "./__root";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { formatStatus, useI18n } from "@/lib/i18n";
import { api, type Tenant } from "@/lib/api";
import { useDataTable } from "@/hooks/use-data-table";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tenants",
  component: Tenants,
});

function Tenants() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ slug: "", name: "" });
  const [error, setError] = useState("");
  const [pageError, setPageError] = useState("");
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
        <Badge variant={row.original.status === "active" ? "default" : "secondary"}>
          {formatStatus(row.original.status, t)}
        </Badge>
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
    return [
      tenant.name,
      tenant.slug,
      tenant.status,
      formatStatus(tenant.status, t),
    ].some((value) => value.toLowerCase().includes(query));
  }, [t]);
  const initialTableState = useMemo(() => ({
    sorting: [{ desc: false, id: "name" }],
  }), []);
  const { globalFilter, setGlobalFilter, table } = useDataTable({
    columns,
    data: tenants,
    globalFilterFn: filterTenants,
    initialState: initialTableState,
  });

  const load = () => api.listTenants().then((t) => {
    setPageError("");
    setTenants(t ?? []);
  }).catch((e) => setPageError(String(e)));
  useEffect(() => void load(), []);

  async function create() {
    setError("");
    try {
      const created = await api.createTenant(form);
      setForm({ slug: "", name: "" });
      setOpen(false);
      navigate({ to: "/tenants/$id", params: { id: created.id } });
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("tenants.title")}</h1>
          <p className="text-muted-foreground">{t("tenants.description")}</p>
        </div>
        <Button onClick={() => { setError(""); setOpen(true); }}><Plus className="size-4" /> {t("tenants.new")}</Button>
      </div>

      {pageError && <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{pageError}</div>}

      <DataTable
        labels={dataTableLabels}
        onRowClick={(tenant) => navigate({ to: "/tenants/$id", params: { id: tenant.id } })}
        table={table}
      >
        <div className="flex items-center justify-between gap-2">
          <Input
            aria-label={t("tenants.search")}
            className="max-w-xs"
            onChange={(event) => setGlobalFilter(event.target.value)}
            placeholder={t("tenants.search")}
            value={globalFilter}
          />
        </div>
      </DataTable>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("tenants.newDialog.title")}</DialogTitle>
            <DialogDescription>{t("tenants.newDialog.description")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.name")}</label>
              <Input placeholder="Acme Corp" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.slug")}</label>
              <Input placeholder="acme" value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value })} />
            </div>
            {error && <div className="text-sm text-destructive">{error}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!form.slug || !form.name} onClick={create}>{t("tenants.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
