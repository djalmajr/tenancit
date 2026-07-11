import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { useCallback, useMemo, useState } from "react";
import { Plus, ShieldAlert, Copy, Check, Ban, RefreshCw, CircleHelp, Pencil, Trash2 } from "lucide-react";
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
import { apiClientCreatedAtSortValue } from "@/lib/api-client-sort";
import {
  canChangeAPIClientDialogOpen,
  isDuplicateAPIClientName,
} from "@/lib/api-client-dialog";
import { writeClipboardText } from "@/lib/clipboard";
import { consumerSnippets } from "@/lib/consumer-snippets";
import { useDataTable } from "@/hooks/use-data-table";
import { invalidateApiClients } from "@/lib/query-invalidation";
import { adminQueryOptions } from "@/lib/query-options";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { Combobox } from "@/components/ui/combobox";
import { Checkbox } from "@/components/ui/checkbox";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { DomainStatus } from "@/components/domain-status";

const EMPTY_API_CLIENTS: ApiClient[] = [];
const FILTER_REFERENCE_TIME = Date.now();

export default function ApiClients() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [helpOpen, setHelpOpen] = useState(false);
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [scopes, setScopes] = useState<Array<"tenant:identify" | "resource:resolve">>([
    "tenant:identify",
    "resource:resolve",
  ]);
  const [rpmLimit, setRpmLimit] = useState("300");
  const [expirationDays, setExpirationDays] = useState("90");
  const [editingClient, setEditingClient] = useState<ApiClient | null>(null);
  const [pendingDelete, setPendingDelete] = useState<ApiClient | null>(null);
  const [isLifecyclePending, setIsLifecyclePending] = useState(false);
  const [token, setToken] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [copied, setCopied] = useState(false);
  const [copiedSnippet, setCopiedSnippet] = useState<
    "identify" | "tenantId" | "hostname" | "resource" | ""
  >("");
  const [error, setError] = useState("");
  const clientsQuery = useQuery(adminQueryOptions.apiClients());
  // TanStack Table uses referential equality to detect data changes. A module
  // constant prevents its auto-reset logic from looping during query loading.
  const clients = clientsQuery.data ?? EMPTY_API_CLIENTS;
  const visibleError = error || (clientsQuery.error ? apiErrorMessage(clientsQuery.error, t) : "");
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
  const columns: ColumnDef<ApiClient>[] = [
    {
      accessorKey: "name",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} label={t("apiClients.name")} labels={sortLabels} />
      ),
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      meta: { label: t("apiClients.name") },
    },
    {
      accessorFn: (client) => client.key_preview ?? "tnc_••••••••",
      id: "token",
      header: ({ column }) => (
        <DataTableColumnHeader column={column} label={t("apiClients.token")} labels={sortLabels} />
      ),
      cell: ({ row }) => (
        <code className="text-xs text-muted-foreground">{row.original.key_preview ?? "tnc_••••••••"}</code>
      ),
      meta: { label: t("apiClients.token") },
    },
    {
      accessorFn: (client) => apiClientCreatedAtSortValue(client.created_at),
      id: "created_at",
      // Keep the header's asc -> desc -> reset action labels truthful even
      // though TanStack infers a descending-first cycle for numeric values.
      sortDescFirst: false,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} label={t("apiClients.createdAt")} labels={sortLabels} />
      ),
      cell: ({ row }) => (
        <span className="text-muted-foreground">{(row.original.created_at ?? "").slice(0, 10) || "—"}</span>
      ),
      meta: { label: t("apiClients.createdAt") },
    },
    {
      accessorFn: (client) => client.scopes.join(", "),
      id: "scopes",
      filterFn: (row, _id, value) => (value as Array<"tenant:identify" | "resource:resolve">).some((scope) => row.original.scopes.includes(scope)),
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("apiClients.scopes")} labels={sortLabels} />,
      cell: ({ row }) => <div className="flex flex-wrap gap-1">{row.original.scopes.map((scope) => <Badge key={scope} variant="outline">{scope}</Badge>)}</div>,
      meta: { label: t("apiClients.scopes") },
    },
    {
      accessorKey: "rpm_limit",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("apiClients.rpmLimit")} labels={sortLabels} />,
      cell: ({ row }) => row.original.rpm_limit ?? <Badge variant="secondary">{t("apiClients.legacy")}</Badge>,
      meta: { label: t("apiClients.rpmLimit") },
    },
    {
      accessorKey: "expires_at",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("apiClients.expiration")} labels={sortLabels} />,
      cell: ({ row }) => row.original.expires_at?.slice(0, 10) ?? <Badge variant="secondary">{t("apiClients.legacy")}</Badge>,
      meta: { label: t("apiClients.expiration") },
    },
    {
      accessorKey: "last_used_at",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("apiClients.lastUsed")} labels={sortLabels} />,
      cell: ({ row }) => row.original.last_used_at?.slice(0, 10) ?? "—",
      meta: { label: t("apiClients.lastUsed") },
    },
    {
      accessorKey: "status",
      filterFn: (row, _id, value) => (value as string[]).includes(row.original.status),
      header: ({ column }) => (
        <DataTableColumnHeader column={column} label={t("common.status")} labels={sortLabels} />
      ),
      cell: ({ row }) => (
        <DomainStatus label={formatStatus(row.original.status, t)} value={row.original.status} />
      ),
      meta: { label: t("common.status") },
    },
    {
      accessorFn: (client) => {
        const flags: string[] = [];
        const now = FILTER_REFERENCE_TIME;
        if (client.expires_at && Date.parse(client.expires_at) <= now + 30 * 24 * 60 * 60 * 1000) flags.push("expiring");
        if (client.legacy_unbounded) flags.push("legacy");
        if (!client.last_used_at || Date.parse(client.last_used_at) < now - 30 * 24 * 60 * 60 * 1000) flags.push("inactive");
        return flags.join(",");
      },
      enableHiding: false,
      enableSorting: false,
      filterFn: (row, _id, value) => (value as string[]).every((flag) => String(row.getValue("flags")).split(",").includes(flag)),
      header: t("apiClients.filters"),
      id: "flags",
      meta: { label: t("apiClients.filters") },
    },
    {
      enableSorting: false,
      enableHiding: false,
      header: t("apiClients.actions"),
      id: "actions",
      cell: ({ row }) => (
        <div className="flex justify-end gap-1">
          {row.original.status === "active" ? (
            <>
              <Button disabled={isLifecyclePending} onClick={() => startEdit(row.original)} size="icon-sm" title={t("common.edit")} variant="ghost"><Pencil className="size-4" /></Button>
              <Button disabled={isLifecyclePending} onClick={() => { void rotate(row.original); }} size="icon-sm" title={t("apiClients.rotate")} variant="ghost"><RefreshCw className="size-4" /></Button>
              <Button disabled={isLifecyclePending} onClick={() => { void revoke(row.original); }} size="icon-sm" title={t("apiClients.revoke")} variant="ghost"><Ban className="size-4" /></Button>
            </>
          ) : (
            <Button disabled={isLifecyclePending} onClick={() => setPendingDelete(row.original)} size="icon-sm" title={t("common.remove")} variant="ghost"><Trash2 className="size-4" /></Button>
          )}
        </div>
      ),
      meta: { align: "right", label: t("apiClients.actions") },
      size: 72,
    },
  ];
  const filterClients = useCallback((client: ApiClient, filterValue: string) => {
    const query = filterValue.trim().toLowerCase();
    if (!query) return true;
    return [
      client.name,
      client.key_preview ?? "",
      client.status,
      formatStatus(client.status, t),
      (client.created_at ?? "").slice(0, 10),
    ].some((value) => value.toLowerCase().includes(query));
  }, [t]);
  const initialTableState = useMemo(() => ({
    sorting: [{ desc: true, id: "created_at" }],
  }), []);
  const dataTable = useDataTable({
    columns,
    data: clients,
    globalFilterFn: filterClients,
    initialState: { ...initialTableState, columnVisibility: { flags: false } },
    visibilityStorageKey: "tenancit.api-clients.columns",
  });
  const { table } = dataTable;

  function start() {
    setEditingClient(null);
    setName("");
    setToken("");
    setCopied(false);
    setError("");
    setScopes(["tenant:identify", "resource:resolve"]);
    setRpmLimit("300");
    setExpirationDays("90");
    setOpen(true);
  }
  function startEdit(client: ApiClient) {
    setEditingClient(client);
    setName(client.name);
    setScopes(client.scopes as Array<"tenant:identify" | "resource:resolve">);
    setRpmLimit(String(client.rpm_limit ?? 300));
    const remainingDays = client.expires_at
      ? Math.max(30, Math.ceil((Date.parse(client.expires_at) - Date.now()) / (24 * 60 * 60 * 1000)))
      : 90;
    setExpirationDays(String([30, 90, 180, 365].find((days) => days >= remainingDays) ?? 365));
    setToken("");
    setError("");
    setOpen(true);
  }
  async function create() {
    const trimmedName = name.trim();
    if (!trimmedName) return;
    if (isDuplicateAPIClientName(trimmedName, clients.map((client) => client.name))) {
      setError(t("errors.conflict"));
      return;
    }
    setIsCreating(true);
    setError("");
    try {
      const policy = {
        name: trimmedName,
        scopes,
        rpm_limit: Number(rpmLimit),
        expires_at: new Date(Date.now() + Number(expirationDays) * 24 * 60 * 60 * 1000).toISOString(),
      };
      if (editingClient) {
        await api.updateAPIClient(editingClient.id, policy);
        setOpen(false);
        toast.success(t("apiClients.updated"));
      } else {
        const result = await api.createAPIClient(policy);
        setToken(result.token);
        setOpen(true);
      }
      await invalidateApiClients(queryClient);
    } catch (createError) {
      setError(apiErrorMessage(createError, t));
    } finally {
      setIsCreating(false);
    }
  }
  function changeDialogOpen(nextOpen: boolean) {
    if (!canChangeAPIClientDialogOpen(nextOpen, isCreating)) return;
    setOpen(nextOpen);
    if (!nextOpen) {
      setCopied(false);
      setError("");
      setName("");
      setToken("");
      setEditingClient(null);
    }
  }
  async function rotate(client: ApiClient) {
    setIsLifecyclePending(true);
    setError("");
    try {
      const result = await api.rotateAPIClient(client.id, 300);
      setEditingClient(null);
      setName(client.name);
      setToken(result.token);
      setCopied(false);
      setOpen(true);
      await invalidateApiClients(queryClient);
      toast.success(t("apiClients.rotated"));
    } catch (rotateError) {
      setError(apiErrorMessage(rotateError, t));
    } finally {
      setIsLifecyclePending(false);
    }
  }
  async function revoke(client: ApiClient) {
    setIsLifecyclePending(true);
    setError("");
    try {
      await api.revokeAPIClient(client.id);
      await invalidateApiClients(queryClient);
      toast.success(t("apiClients.statusRevoked"));
    } catch (revokeError) {
      setError(apiErrorMessage(revokeError, t));
    } finally {
      setIsLifecyclePending(false);
    }
  }
  async function removeClient() {
    if (!pendingDelete) return;
    setIsLifecyclePending(true);
    try {
      await api.deleteAPIClient(pendingDelete.id);
      setPendingDelete(null);
      await invalidateApiClients(queryClient);
      toast.success(t("apiClients.deleted"));
    } catch (deleteError) {
      setError(apiErrorMessage(deleteError, t));
    } finally {
      setIsLifecyclePending(false);
    }
  }
  function toggleScope(scope: "tenant:identify" | "resource:resolve") {
    setScopes((current) => current.includes(scope)
      ? current.filter((candidate) => candidate !== scope)
      : [...current, scope]);
  }
  async function copy() {
    try {
      await writeClipboardText(token);
      setCopied(true);
      setError("");
    } catch {
      setCopied(false);
      setError(t("apiClients.copyFailed"));
    }
  }
  async function copyResolveSnippet(
    snippet: string,
    snippetKey: "identify" | "tenantId" | "hostname" | "resource",
  ) {
    try {
      await writeClipboardText(snippet);
      setCopiedSnippet(snippetKey);
      setError("");
      toast.success(t("apiClients.resolveSnippet.copied"));
    } catch {
      setCopiedSnippet("");
      toast.error(t("apiClients.copyFailed"));
    }
  }
  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
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

      {visibleError && <Alert variant="destructive"><AlertDescription>{visibleError}</AlertDescription></Alert>}

      {clientsQuery.isPending ? (
        <div className="rounded-md border p-4 text-sm text-muted-foreground" role="status">
          {t("apiClients.loading")}
        </div>
      ) : (
        <DataTable labels={dataTableLabels} table={table}>
          <DataTableToolbar
            clearLabel={t("dataTable.clearFilters")}
            columnsLabel={t("dataTable.columns")}
            emptyLabel={t("dataTable.noResults")}
            facets={[
              { columnId: "status", multiple: true, options: ["active", "expired", "revoked"].map((value) => ({ value, label: formatStatus(value, t) })), title: t("apiClients.filterStatus") },
              { columnId: "scopes", multiple: true, options: ["tenant:identify", "resource:resolve"].map((value) => ({ value, label: value })), title: t("apiClients.filterScope") },
              { columnId: "flags", multiple: true, options: [{ value: "expiring", label: t("apiClients.expiringSoon") }, { value: "legacy", label: t("apiClients.legacy") }, { value: "inactive", label: t("apiClients.inactive") }], title: t("apiClients.filters") },
            ]}
            searchLabel={t("apiClients.search")}
            table={table}
          />
        </DataTable>
      )}

      <Dialog modal={false} open={helpOpen} onOpenChange={setHelpOpen}>
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
              copied={copiedSnippet === "identify"}
              description={t("apiClients.resolveSnippet.edgeDescription")}
              onCopy={() => { void copyResolveSnippet(consumerSnippets.identify, "identify"); }}
              snippet={consumerSnippets.identify}
              title={t("apiClients.resolveSnippet.edgeTitle")}
            />
            <SnippetExample
              copied={copiedSnippet === "tenantId"}
              description={t("apiClients.resolveSnippet.tenantIdDescription")}
              onCopy={() => { void copyResolveSnippet(consumerSnippets.byTenantId, "tenantId"); }}
              snippet={consumerSnippets.byTenantId}
              title={t("apiClients.resolveSnippet.tenantIdTitle")}
            />
            <SnippetExample
              copied={copiedSnippet === "hostname"}
              description={t("apiClients.resolveSnippet.hostnameDescription")}
              onCopy={() => { void copyResolveSnippet(consumerSnippets.byHostname, "hostname"); }}
              snippet={consumerSnippets.byHostname}
              title={t("apiClients.resolveSnippet.hostnameTitle")}
            />
            <SnippetExample
              copied={copiedSnippet === "resource"}
              description={t("apiClients.resolveSnippet.resourceDescription")}
              onCopy={() => { void copyResolveSnippet(consumerSnippets.resource, "resource"); }}
              snippet={consumerSnippets.resource}
              title={t("apiClients.resolveSnippet.resourceTitle")}
            />
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={open} onOpenChange={changeDialogOpen}>
        <DialogContent>
          {!token ? (
            <>
              <DialogHeader>
                <DialogTitle>{t(editingClient ? "apiClients.editTitle" : "apiClients.newDialog.title")}</DialogTitle>
                <DialogDescription>{t("apiClients.newDialog.description")}</DialogDescription>
              </DialogHeader>
              <div className="space-y-1.5">
                <label className="text-sm font-medium" htmlFor="new-api-client-name">{t("apiClients.name")}</label>
                <Input id="new-api-client-name" placeholder="billing-service" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <fieldset className="space-y-2">
                <legend className="text-sm font-medium">{t("apiClients.scopes")}</legend>
                <label className="flex items-start gap-2 rounded-lg border p-3 text-sm" htmlFor="api-client-scope-identify">
                  <Checkbox
                    aria-label={t("apiClients.scopeIdentify")}
                    id="api-client-scope-identify"
                    checked={scopes.includes("tenant:identify")}
                    className="mt-0.5"
                    onCheckedChange={() => toggleScope("tenant:identify")}
                  />
                  <span><strong>{t("apiClients.scopeIdentify")}</strong><span className="block text-muted-foreground">{t("apiClients.scopeIdentifyDescription")}</span></span>
                </label>
                <label className="flex items-start gap-2 rounded-lg border border-amber-500/30 bg-amber-500/5 p-3 text-sm" htmlFor="api-client-scope-resolve">
                  <Checkbox
                    aria-label={t("apiClients.scopeResolve")}
                    id="api-client-scope-resolve"
                    checked={scopes.includes("resource:resolve")}
                    className="mt-0.5"
                    onCheckedChange={() => toggleScope("resource:resolve")}
                  />
                  <span><strong>{t("apiClients.scopeResolve")}</strong><span className="block text-muted-foreground">{t("apiClients.scopeResolveDescription")}</span></span>
                </label>
              </fieldset>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-1.5">
                  <label className="text-sm font-medium" htmlFor="new-api-client-rpm">{t("apiClients.rpmLimit")}</label>
                  <Input id="new-api-client-rpm" min="1" onChange={(event) => setRpmLimit(event.target.value)} type="number" value={rpmLimit} />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium" htmlFor="new-api-client-expiration">{t("apiClients.expiration")}</label>
                  <Combobox aria-label={t("apiClients.expiration")} className="w-full" options={[30, 90, 180, 365].map((days) => ({ value: String(days), label: t("apiClients.expirationDays", { days }) }))} onValueChange={setExpirationDays} triggerClassName="w-full" value={expirationDays} />
                </div>
              </div>
              {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
              <DialogFooter>
                <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
                <Button disabled={!name.trim() || scopes.length === 0 || Number(rpmLimit) <= 0 || isCreating} onClick={() => { void create(); }}>{t(editingClient ? "common.save" : "apiClients.generateToken")}</Button>
              </DialogFooter>
            </>
          ) : (
            <>
              <DialogHeader>
                <DialogTitle>{t("apiClients.tokenGenerated.title")}</DialogTitle>
                <DialogDescription>{t("apiClients.tokenGenerated.description")}</DialogDescription>
              </DialogHeader>
              <div className="flex min-w-0 items-center gap-2 rounded-lg border bg-muted/50 px-3 py-2.5">
                <Input
                  aria-label={t("apiClients.token")}
                  className="min-w-0 flex-1 border-0 bg-transparent font-mono text-xs shadow-none focus-visible:ring-0 dark:bg-transparent"
                  readOnly
                  tabIndex={0}
                  value={token}
                />
                <Button
                  aria-label={copied ? t("apiClients.copied") : t("apiClients.copy")}
                  className="shrink-0"
                  onClick={() => { void copy(); }}
                  size="icon-sm"
                  title={copied ? t("apiClients.copied") : t("apiClients.copy")}
                  variant="ghost"
                >
                  {copied ? <Check data-icon="inline-start" /> : <Copy data-icon="inline-start" />}
                </Button>
              </div>
              {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
              <DialogFooter>
                <DialogClose render={<Button>{t("apiClients.done")}</Button>} />
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
      <ConfirmDialog
        cancelLabel={t("common.cancel")}
        confirmDisabled={isLifecyclePending}
        confirmLabel={t("common.removeConfirm")}
        description={t("apiClients.deleteDescription", { name: pendingDelete?.name ?? "" })}
        onConfirm={() => { void removeClient(); }}
        onOpenChange={(nextOpen) => { if (!nextOpen) setPendingDelete(null); }}
        open={pendingDelete !== null}
        title={t("apiClients.deleteTitle")}
      />
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
