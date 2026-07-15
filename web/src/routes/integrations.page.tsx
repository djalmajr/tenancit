import { useCallback, useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { Plus, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Combobox } from "@/components/ui/combobox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useDataTable } from "@/hooks/use-data-table";
import {
  api,
  type CreatedWebhookTarget,
  type WebhookDelivery,
  type WebhookTarget,
} from "@/lib/api";
import { apiErrorMessage, useI18n } from "@/lib/i18n";
import { adminQueryKeys } from "@/lib/query-keys";
import { isValidWebhookURL } from "@/lib/validation";

const EMPTY_TARGETS: WebhookTarget[] = [];
const EMPTY_DELIVERIES: WebhookDelivery[] = [];

export default function IntegrationsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [url, setURL] = useState("");
  const [format, setFormat] = useState<WebhookTarget["format"]>("generic");
  const [createdTarget, setCreatedTarget] = useState<CreatedWebhookTarget>();
  const [pending, setPending] = useState(false);
  const webhookURLValid = isValidWebhookURL(url);

  const targetsQuery = useQuery({
    queryKey: adminQueryKeys.webhookTargets(),
    queryFn: ({ signal }) => api.listWebhookTargets(signal),
  });
  const deliveriesQuery = useQuery({
    queryKey: adminQueryKeys.webhookDeliveries(),
    queryFn: ({ signal }) => api.listWebhookDeliveries("", signal),
    refetchInterval: 30_000,
  });

  const sortLabels = useMemo(() => ({
    asc: t("dataTable.sortAsc"),
    desc: t("dataTable.sortDesc"),
    reset: t("dataTable.sortReset"),
  }), [t]);
  const tableLabels = useMemo(() => ({
    goToFirstPage: t("dataTable.firstPage"),
    goToLastPage: t("dataTable.lastPage"),
    goToNextPage: t("dataTable.nextPage"),
    goToPreviousPage: t("dataTable.previousPage"),
    item: t("dataTable.item"),
    items: t("dataTable.items"),
    noResults: t("dataTable.noResults"),
    page: t("dataTable.page"),
    pageOf: t("dataTable.pageOf"),
    rowsPerPage: t("dataTable.rowsPerPage"),
  }), [t]);

  const refreshWebhookQueries = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: adminQueryKeys.webhookTargets() }),
      queryClient.invalidateQueries({ queryKey: adminQueryKeys.webhookDeliveries() }),
    ]);
  }, [queryClient]);

  async function createTarget() {
    if (!name.trim() || !webhookURLValid) return;
    setPending(true);
    try {
      const created = await api.createWebhookTarget({ name, url, format });
      setCreatedTarget(created);
      setCreateOpen(false);
      setName("");
      setURL("");
      await refreshWebhookQueries();
      toast.success(t("integrations.created"));
    } catch (error) {
      toast.error(apiErrorMessage(error, t));
    } finally {
      setPending(false);
    }
  }

  const toggleTarget = useCallback(async (target: WebhookTarget) => {
    try {
      await api.setWebhookTargetStatus(target.id, target.status === "active" ? "disabled" : "active");
      await refreshWebhookQueries();
    } catch (error) {
      toast.error(apiErrorMessage(error, t));
    }
  }, [refreshWebhookQueries, t]);

  const replayDelivery = useCallback(async (id: string) => {
    try {
      await api.replayWebhookDelivery(id);
      await refreshWebhookQueries();
      toast.success(t("integrations.replayed"));
    } catch (error) {
      toast.error(apiErrorMessage(error, t));
    }
  }, [refreshWebhookQueries, t]);

  const targetColumns = useMemo<ColumnDef<WebhookTarget>[]>(() => [
    {
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("integrations.name")} labels={sortLabels} />,
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
      meta: { label: t("integrations.name") },
    },
    {
      accessorKey: "endpoint",
      header: t("integrations.endpoint"),
      cell: ({ row }) => <code className="text-xs">{row.original.endpoint}</code>,
      meta: { label: t("integrations.endpoint") },
    },
    {
      accessorKey: "format",
      header: t("integrations.format"),
      cell: ({ row }) => <Badge variant="outline">{row.original.format}</Badge>,
      meta: { label: t("integrations.format") },
    },
    {
      accessorKey: "status",
      header: t("common.status"),
      cell: ({ row }) => <Badge variant={row.original.status === "active" ? "default" : "secondary"}>{row.original.status}</Badge>,
      meta: { label: t("common.status") },
    },
    {
      id: "actions",
      enableHiding: false,
      enableSorting: false,
      cell: ({ row }) => <Button size="sm" variant="outline" onClick={() => void toggleTarget(row.original)}>
        {t(row.original.status === "active" ? "integrations.disable" : "integrations.enable")}
      </Button>,
    },
  ], [sortLabels, t, toggleTarget]);

  const deliveryColumns = useMemo<ColumnDef<WebhookDelivery>[]>(() => [
    {
      accessorKey: "event_type",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("integrations.event")} labels={sortLabels} />,
      meta: { label: t("integrations.event") },
    },
    { accessorKey: "target_name", header: t("integrations.target"), meta: { label: t("integrations.target") } },
    {
      accessorKey: "status",
      header: t("common.status"),
      cell: ({ row }) => <Badge variant={row.original.status === "delivered" ? "default" : row.original.status === "dead_letter" ? "destructive" : "secondary"}>{row.original.status}</Badge>,
      meta: { label: t("common.status") },
    },
    { accessorKey: "attempt_count", header: t("integrations.attempts"), meta: { label: t("integrations.attempts") } },
    {
      id: "actions",
      enableHiding: false,
      enableSorting: false,
      cell: ({ row }) => row.original.status === "dead_letter" ? <Button
        aria-label={t("integrations.replay")}
        size="icon-sm"
        variant="ghost"
        onClick={() => void replayDelivery(row.original.id)}
      ><RefreshCw /></Button> : null,
    },
  ], [replayDelivery, sortLabels, t]);

  const { table: targetsTable } = useDataTable({
    columns: targetColumns,
    data: targetsQuery.data ?? EMPTY_TARGETS,
    initialState: { pagination: { pageIndex: 0, pageSize: 10 } },
    visibilityStorageKey: "tenancit.webhooks.targets.columns",
  });
  const { table: deliveriesTable } = useDataTable({
    columns: deliveryColumns,
    data: deliveriesQuery.data ?? EMPTY_DELIVERIES,
    initialState: { pagination: { pageIndex: 0, pageSize: 10 } },
    visibilityStorageKey: "tenancit.webhooks.deliveries.columns",
  });

  const queryError = targetsQuery.error || deliveriesQuery.error;

  return <div className="flex flex-col gap-4">
    {queryError && <Alert variant="destructive"><AlertDescription>{apiErrorMessage(queryError, t)}</AlertDescription></Alert>}

    <Tabs className="gap-4" defaultValue="targets">
      <TabsList>
        <TabsTrigger value="targets">{t("integrations.targets")}</TabsTrigger>
        <TabsTrigger value="deliveries">{t("integrations.deliveries")}</TabsTrigger>
      </TabsList>
      <TabsContent value="targets">
        <DataTable labels={tableLabels} table={targetsTable}>
          <DataTableToolbar
            clearLabel={t("dataTable.clearFilters")}
            columnsLabel={t("dataTable.columns")}
            emptyLabel={t("integrations.emptyTargets")}
            resetLabel={t("dataTable.resetPreferences")}
            searchLabel={t("integrations.searchTargets")}
            table={targetsTable}
            trailing={<Button onClick={() => setCreateOpen(true)}>
              <Plus data-icon="inline-start" />
              {t("integrations.new")}
            </Button>}
          />
        </DataTable>
      </TabsContent>
      <TabsContent value="deliveries">
        <DataTable labels={tableLabels} table={deliveriesTable}>
          <DataTableToolbar
            clearLabel={t("dataTable.clearFilters")}
            columnsLabel={t("dataTable.columns")}
            emptyLabel={t("integrations.emptyDeliveries")}
            resetLabel={t("dataTable.resetPreferences")}
            searchLabel={t("integrations.searchDeliveries")}
            table={deliveriesTable}
          />
        </DataTable>
      </TabsContent>
    </Tabs>

    <Dialog open={createOpen} onOpenChange={setCreateOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("integrations.new")}</DialogTitle>
          <DialogDescription>{t("integrations.newDescription")}</DialogDescription>
        </DialogHeader>
        <label className="space-y-1.5 text-sm">
          {t("integrations.name")}
          <Input aria-label={t("integrations.name")} value={name} onChange={(event) => setName(event.target.value)} />
        </label>
        <label className="space-y-1.5 text-sm">
          {t("integrations.url")}
          <Input
            aria-describedby="webhook-url-hint"
            aria-invalid={url.length > 0 && !webhookURLValid}
            aria-label={t("integrations.url")}
            autoComplete="url"
            placeholder="https://receiver.example/webhooks/tenancit"
            type="url"
            value={url}
            onChange={(event) => setURL(event.target.value)}
          />
          <span className={url.length > 0 && !webhookURLValid ? "block text-xs text-destructive" : "block text-xs text-muted-foreground"} id="webhook-url-hint">
            {t("validation.webhookURL")}
          </span>
        </label>
        <Combobox
          aria-label={t("integrations.format")}
          options={[
            { value: "generic", label: t("integrations.formatGeneric") },
            { value: "slack", label: "Slack" },
            { value: "discord", label: "Discord" },
            { value: "teams", label: "Microsoft Teams" },
          ]}
          searchable={false}
          value={format}
          onValueChange={(value) => setFormat(value as WebhookTarget["format"])}
        />
        <DialogFooter>
          <Button variant="outline" onClick={() => setCreateOpen(false)}>{t("common.cancel")}</Button>
          <Button disabled={pending || !name.trim() || !webhookURLValid} onClick={() => void createTarget()}>{t("common.save")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <Dialog open={Boolean(createdTarget)} onOpenChange={(open) => { if (!open) setCreatedTarget(undefined); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t("integrations.secretTitle")}</DialogTitle>
          <DialogDescription>{t("integrations.secretDescription")}</DialogDescription>
        </DialogHeader>
        <Input aria-label={t("integrations.signingSecret")} className="font-mono" readOnly value={createdTarget?.signing_secret ?? ""} />
        <DialogFooter><Button onClick={() => setCreatedTarget(undefined)}>{t("integrations.done")}</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  </div>;
}
