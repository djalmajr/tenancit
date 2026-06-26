import { createRoute } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Plus, ShieldAlert, Copy, Check, Ban, RotateCcw, CircleHelp } from "lucide-react";
import { Route as rootRoute } from "./__root";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { toast } from "sonner";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { apiErrorMessage, formatStatus, useI18n } from "@/lib/i18n";
import { api, type ApiClient } from "@/lib/api";
import { useDataTable } from "@/hooks/use-data-table";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/api-clients",
  component: ApiClients,
});

function ApiClients() {
  const { t } = useI18n();
  const [clients, setClients] = useState<ApiClient[]>([]);
  const [helpOpen, setHelpOpen] = useState(false);
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [token, setToken] = useState("");
  const [copied, setCopied] = useState(false);
  const [copiedSnippet, setCopiedSnippet] = useState<"resource" | "tenant" | "">("");
  const [error, setError] = useState("");
  const tenantResolveSnippet = `curl -H "Authorization: Bearer <token>" \\
  "/v1/resolve?hostname=<tenant-hostname>"`;
  const resourceResolveSnippet = `curl -H "Authorization: Bearer <token>" \\
  "/v1/resolve/<tenant-hostname>/resources/<definition-key>"`;
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
    noResults: t("apiClients.emptyActive"),
    page: t("dataTable.page"),
    pageOf: t("dataTable.pageOf"),
    rowsPerPage: t("dataTable.rowsPerPage"),
  }), [t]);
  const columns = useMemo<ColumnDef<ApiClient>[]>(() => [
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} label={t("apiClients.name")} labels={sortLabels} />
      ),
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      meta: { label: t("apiClients.name") },
    },
    {
      accessorFn: (client) => client.key_preview ?? "rt_live_••••••••",
      id: "token",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} label={t("apiClients.token")} labels={sortLabels} />
      ),
      cell: ({ row }) => (
        <code className="text-xs text-muted-foreground">{row.original.key_preview ?? "rt_live_••••••••"}</code>
      ),
      meta: { label: t("apiClients.token") },
    },
    {
      accessorKey: "created_at",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} label={t("apiClients.createdAt")} labels={sortLabels} />
      ),
      cell: ({ row }) => (
        <span className="text-muted-foreground">{(row.original.created_at ?? "").slice(0, 10) || "—"}</span>
      ),
      meta: { label: t("apiClients.createdAt") },
    },
    {
      accessorKey: "status",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} label={t("common.status")} labels={sortLabels} />
      ),
      cell: ({ row }) => (
        <Badge variant={row.original.status === "active" ? "default" : "destructive"}>
          {formatStatus(row.original.status, t)}
        </Badge>
      ),
      meta: { label: t("common.status") },
    },
    {
      enableSorting: false,
      header: t("apiClients.actions"),
      id: "actions",
      cell: ({ row }) => (
        <div className="text-right">
          {row.original.status === "active" ? (
            <Button
              onClick={() => revoke(row.original)}
              size="icon-sm"
              title={t("apiClients.revoke")}
              variant="ghost"
            >
              <Ban className="size-4" />
            </Button>
          ) : (
            <Button
              onClick={() => reactivate(row.original)}
              size="icon-sm"
              title={t("apiClients.reactivate")}
              variant="ghost"
            >
              <RotateCcw className="size-4" />
            </Button>
          )}
        </div>
      ),
      meta: { align: "right", label: t("apiClients.actions") },
      size: 72,
    },
  ], [sortLabels, t]);
  const filterClients = useCallback((client: ApiClient, filterValue: string) => {
    const query = filterValue.trim().toLowerCase();
    if (!query) return true;
    return [
      client.name,
      client.key_preview ?? "rt_live_••••••••",
      client.status,
      formatStatus(client.status, t),
      (client.created_at ?? "").slice(0, 10),
    ].some((value) => value.toLowerCase().includes(query));
  }, [t]);
  const initialTableState = useMemo(() => ({
    sorting: [{ desc: true, id: "created_at" }],
  }), []);
  const { globalFilter, setGlobalFilter, table } = useDataTable({
    columns,
    data: clients,
    globalFilterFn: filterClients,
    initialState: initialTableState,
  });

  const load = () => api.listAPIClients().then((c) => {
    setClients(c ?? []);
    setError("");
  }).catch((e) => setError(apiErrorMessage(e, t)));
  useEffect(() => void load(), []);

  function start() {
    setName("");
    setToken("");
    setCopied(false);
    setError("");
    setOpen(true);
  }
  async function create() {
    if (!name.trim()) return;
    try {
      const res = await api.createAPIClient(name.trim());
      setError("");
      setToken(res.token);
      load();
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }
  function copy() {
    navigator.clipboard?.writeText(token).then(() => setCopied(true)).catch(() => setCopied(true));
  }
  function copyResolveSnippet(snippet: string, snippetKey: "resource" | "tenant") {
    navigator.clipboard?.writeText(snippet).then(() => {
      setCopiedSnippet(snippetKey);
      toast.success(t("apiClients.resolveSnippet.copied"));
    }).catch(() => {
      setCopiedSnippet(snippetKey);
      toast.success(t("apiClients.resolveSnippet.copied"));
    });
  }
  async function revoke(c: ApiClient) {
    try {
      await api.setAPIClientStatus(c.id, "revoked");
      toast.success(t("apiClients.statusRevoked"));
      load();
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }
  async function reactivate(c: ApiClient) {
    try {
      await api.setAPIClientStatus(c.id, "active");
      toast.success(t("apiClients.statusReactivated"));
      load();
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("apiClients.title")}</h1>
          <p className="text-muted-foreground">{t("apiClients.description")}</p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            aria-label={t("apiClients.help")}
            onClick={() => setHelpOpen(true)}
            size="icon"
            title={t("apiClients.help")}
            variant="ghost"
          >
            <CircleHelp className="size-4" />
          </Button>
          <Button onClick={start}><Plus className="size-4" /> {t("apiClients.new")}</Button>
        </div>
      </div>

      {error && <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}

      <DataTable labels={dataTableLabels} table={table}>
        <div className="flex items-center justify-between gap-2">
          <Input
            aria-label={t("apiClients.search")}
            className="max-w-xs"
            onChange={(event) => setGlobalFilter(event.target.value)}
            placeholder={t("apiClients.search")}
            value={globalFilter}
          />
        </div>
      </DataTable>

      <Dialog open={helpOpen} onOpenChange={setHelpOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{t("apiClients.helpTitle")}</DialogTitle>
            <DialogDescription>{t("apiClients.resolveSnippet.description")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="flex items-start gap-3 rounded-md border border-amber-500/20 bg-amber-500/10 p-3">
              <ShieldAlert className="mt-0.5 size-4 shrink-0 text-amber-500" />
              <div className="text-sm">
                <div className="font-medium">{t("apiClients.clearSecretAccess.title")}</div>
                <p className="text-muted-foreground">{t("apiClients.clearSecretAccess.description")}</p>
              </div>
            </div>
            <SnippetExample
              copied={copiedSnippet === "tenant"}
              description={t("apiClients.resolveSnippet.fullDescription")}
              onCopy={() => copyResolveSnippet(tenantResolveSnippet, "tenant")}
              snippet={tenantResolveSnippet}
              title={t("apiClients.resolveSnippet.fullTitle")}
            />
            <SnippetExample
              copied={copiedSnippet === "resource"}
              description={t("apiClients.resolveSnippet.resourceDescription")}
              onCopy={() => copyResolveSnippet(resourceResolveSnippet, "resource")}
              snippet={resourceResolveSnippet}
              title={t("apiClients.resolveSnippet.resourceTitle")}
            />
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          {!token ? (
            <>
              <DialogHeader>
                <DialogTitle>{t("apiClients.newDialog.title")}</DialogTitle>
                <DialogDescription>{t("apiClients.newDialog.description")}</DialogDescription>
              </DialogHeader>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">{t("apiClients.name")}</label>
                <Input placeholder="billing-service" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <DialogFooter>
                <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
                <Button disabled={!name.trim()} onClick={create}>{t("apiClients.generateToken")}</Button>
              </DialogFooter>
            </>
          ) : (
            <>
              <DialogHeader>
                <DialogTitle>{t("apiClients.tokenGenerated.title")}</DialogTitle>
                <DialogDescription>{t("apiClients.tokenGenerated.description")}</DialogDescription>
              </DialogHeader>
              <div className="flex items-center gap-2 rounded-md border bg-muted/50 p-3">
                <code className="flex-1 break-all text-xs">{token}</code>
                <Button variant="outline" size="sm" onClick={copy}>
                  {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                  {copied ? t("apiClients.copied") : t("apiClients.copy")}
                </Button>
              </div>
              <DialogFooter>
                <DialogClose render={<Button>{t("apiClients.done")}</Button>} />
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function SnippetExample({
  copied,
  description,
  onCopy,
  snippet,
  title,
}: {
  copied: boolean;
  description: string;
  onCopy: () => void;
  snippet: string;
  title: string;
}) {
  const { t } = useI18n();
  return (
    <div className="space-y-2 rounded-md border bg-muted/20 p-3">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-sm font-medium">{title}</div>
          <p className="text-xs text-muted-foreground">{description}</p>
        </div>
        <Button onClick={onCopy} size="sm" variant="outline">
          {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
          {copied ? t("apiClients.resolveSnippet.copied") : t("apiClients.resolveSnippet.copy")}
        </Button>
      </div>
      <pre className="overflow-x-auto rounded-md bg-muted/60 p-3 text-xs">
        <code>{snippet}</code>
      </pre>
    </div>
  );
}
