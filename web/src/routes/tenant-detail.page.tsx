import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getRouteApi, Link, useNavigate } from "@tanstack/react-router";
import type { ColumnDef } from "@tanstack/react-table";
import { useMemo, useRef, useState } from "react";
import {
  CheckCircle2,
  Box, Copy, EllipsisVertical, Eye, EyeOff, Link2, Pencil, Plus, RotateCcw, Trash2,
  CircleAlert, Power, PowerOff, Settings2, TriangleAlert,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { AlertAction, AlertTitle } from "@/components/ui/alert";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { DomainStatus } from "@/components/domain-status";
import { StatCard } from "@/components/stat-card";
import { DataTable } from "@/components/data-table/data-table";
import { DataTableColumnHeader } from "@/components/data-table/data-table-column-header";
import { DataTableToolbar } from "@/components/data-table/data-table-toolbar";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { toast } from "sonner";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { apiErrorMessage, formatStatus, useI18n } from "@/lib/i18n";
import { displaySecretValue } from "@/lib/secret-display";
import { matchesTenantSlug } from "@/lib/tenant-delete";
import {
  api, type ResourceFieldValue, type TenantDomain, type TenantResource,
} from "@/lib/api";
import { stableIdempotencyKey, type IdempotencyAttempt } from "@/lib/idempotency";
import { summarizeTenantOverview } from "@/lib/tenant-overview";
import {
  invalidateTenant,
  invalidateTenantDomains,
  invalidateTenantResources,
  removeTenantQueries,
} from "@/lib/query-invalidation";
import { adminQueryOptions } from "@/lib/query-options";
import { useTransientResourceReveal } from "@/lib/use-transient-resource-reveal";
import { useAdminCapabilities } from "@/hooks/use-admin-capabilities";
import { useDataTable } from "@/hooks/use-data-table";
import {
  isValidHostname,
  isValidResourceAlias,
  isValidTenantSlug,
  isValidTypedResourceValue,
} from "@/lib/validation";

const routeApi = getRouteApi("/tenants/$id");
const EMPTY_RESOURCES: TenantResource[] = [];

export default function TenantDetail() {
  const { t } = useI18n();
  const { id } = routeApi.useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { can } = useAdminCapabilities();
  const [tab, setTab] = useState<"overview" | "resources" | "domains">("overview");

  const [hostname, setHostname] = useState("");
  const [newDomainOpen, setNewDomainOpen] = useState(false);

  const [editOpen, setEditOpen] = useState(false);
  const [edit, setEdit] = useState({ name: "", slug: "", status: "active" });

  const [resOpen, setResOpen] = useState(false);
  const [pick, setPick] = useState("");
  const [resourceName, setResourceName] = useState("");
  const [resourceAlias, setResourceAlias] = useState("");
  const [sourceResourceId, setSourceResourceId] = useState("");
  const [values, setValues] = useState<Record<string, string>>({});
  const [domainError, setDomainError] = useState("");
  const [editError, setEditError] = useState("");
  const [isCreatingResource, setIsCreatingResource] = useState(false);
  const [pageError, setPageError] = useState("");
  const [resourceError, setResourceError] = useState("");
  const resourceCreateAttempt = useRef<IdempotencyAttempt>(null);
  const [pendingDomain, setPendingDomain] = useState<TenantDomain | null>(null);
  const [editingDomain, setEditingDomain] = useState<TenantDomain | null>(null);
  const [editingHostname, setEditingHostname] = useState("");
  const [editDomainError, setEditDomainError] = useState("");
  const [pendingResource, setPendingResource] = useState<Pick<TenantResource, "id" | "name"> | null>(null);
  const [selectedResourceId, setSelectedResourceId] = useState("");
  const [editingResource, setEditingResource] = useState<Pick<TenantResource, "id" | "name" | "alias"> | null>(null);
  const [editingResourceName, setEditingResourceName] = useState("");
  const [editingResourceAlias, setEditingResourceAlias] = useState("");
  const [editResourceError, setEditResourceError] = useState("");
  const [editingField, setEditingField] = useState<ResourceFieldValue | null>(null);
  const [editingFieldValue, setEditingFieldValue] = useState("");
  const [fieldEditError, setFieldEditError] = useState("");
  const [isUpdatingField, setIsUpdatingField] = useState(false);
  const [duplicateOpen, setDuplicateOpen] = useState(false);
  const [duplicateTarget, setDuplicateTarget] = useState<Pick<TenantResource, "id" | "alias"> | null>(null);
  const [duplicateAlias, setDuplicateAlias] = useState("");
  const [duplicateError, setDuplicateError] = useState("");
  const duplicateAttempt = useRef<IdempotencyAttempt>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteConfirmation, setDeleteConfirmation] = useState("");

  const tenantQuery = useQuery(adminQueryOptions.tenant(id));
  const domainsQuery = useQuery(adminQueryOptions.tenantDomains(id));
  const resourcesQuery = useQuery(adminQueryOptions.tenantResources(id));
  const definitionsQuery = useQuery(adminQueryOptions.definitions());
  const reveal = useTransientResourceReveal(id);
  const tenant = tenantQuery.data;
  const domains = domainsQuery.data ?? [];
  const resources = reveal.resources ?? resourcesQuery.data ?? EMPTY_RESOURCES;
  const definitions = definitionsQuery.data ?? [];
  const selectedResource = resources.find((resource) => resource.id === selectedResourceId);

  const resourceSortLabels = useMemo(() => ({
    asc: t("dataTable.sortAsc"),
    desc: t("dataTable.sortDesc"),
    reset: t("dataTable.sortReset"),
  }), [t]);
  const resourceTableLabels = useMemo(() => ({
    goToFirstPage: t("dataTable.firstPage"),
    goToLastPage: t("dataTable.lastPage"),
    goToNextPage: t("dataTable.nextPage"),
    goToPreviousPage: t("dataTable.previousPage"),
    item: t("dataTable.item"),
    items: t("dataTable.items"),
    noResults: t("tenantDetail.noResource"),
    page: t("dataTable.page"),
    pageOf: t("dataTable.pageOf"),
    rowsPerPage: t("dataTable.rowsPerPage"),
  }), [t]);
  const resourceColumns: ColumnDef<TenantResource>[] = [
    {
      accessorKey: "name",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("common.name")} labels={resourceSortLabels} />,
      cell: ({ row }) => <span className="flex items-center gap-2 font-medium">
        {row.original.linked ? <Link2 aria-label={t("tenantDetail.linked")} /> : <Box aria-label={t("tenantDetail.independent")} />}
        {row.original.name}
      </span>,
      meta: { label: t("common.name") },
    },
    {
      accessorKey: "alias",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("tenantDetail.resourceAlias")} labels={resourceSortLabels} />,
      cell: ({ row }) => <code className="text-xs">{row.original.alias}</code>,
      meta: { label: t("tenantDetail.resourceAlias") },
    },
    {
      accessorKey: "definitionKey",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("common.type")} labels={resourceSortLabels} />,
      cell: ({ row }) => <code className="text-xs">{row.original.definitionKey}</code>,
      meta: { label: t("common.type") },
    },
    {
      accessorFn: (resource) => resource.fields.length,
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("tenantDetail.fieldsColumn")} labels={resourceSortLabels} />,
      id: "fields",
      meta: { label: t("tenantDetail.fieldsColumn") },
      size: 110,
    },
    {
      accessorFn: (resource) => resource.fields.filter((field) => field.isSecret).length,
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("tenantDetail.secretsColumn")} labels={resourceSortLabels} />,
      id: "secrets",
      meta: { label: t("tenantDetail.secretsColumn") },
      size: 110,
    },
    {
      accessorKey: "status",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("common.status")} labels={resourceSortLabels} />,
      cell: ({ row }) => <DomainStatus label={formatStatus(row.original.status, t)} value={row.original.status} />,
      meta: { label: t("common.status") },
      size: 120,
    },
    {
      cell: ({ row }) => {
        const resource = row.original;

        return <DropdownMenu>
          <DropdownMenuTrigger
            aria-label={t("common.actions")}
            onClick={(event) => event.stopPropagation()}
            render={<Button size="icon-sm" title={t("common.actions")} variant="ghost" />}
          >
            <EllipsisVertical />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {can("resource.write") && <DropdownMenuItem onClick={(event) => {
              event.stopPropagation();
              openResourceIdentity(resource);
            }}>
              <Pencil /> {t("common.edit")}
            </DropdownMenuItem>}
            {can("resource.write") && <>
              <DropdownMenuItem onClick={(event) => {
                event.stopPropagation();
                void toggleResource(resource);
              }}>
                {resource.status === "active" ? <PowerOff /> : <Power />}
                {resource.status === "active" ? t("tenantDetail.deactivate") : t("tenantDetail.reactivate")}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={(event) => {
                event.stopPropagation();
                openDuplicateResource(resource);
              }}>
                <Copy /> {t("tenantDetail.duplicate")}
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={(event) => {
                event.stopPropagation();
                requestResourceRemoval(resource);
              }} variant="destructive">
                <Trash2 /> {t("common.remove")}
              </DropdownMenuItem>
            </>}
          </DropdownMenuContent>
        </DropdownMenu>;
      },
      enableHiding: false,
      enableSorting: false,
      header: t("common.actions"),
      id: "actions",
      meta: { align: "right", label: t("common.actions") },
      size: 64,
    },
  ];
  const { table: resourceTable } = useDataTable({
    columns: resourceColumns,
    data: resources,
    globalFilterFn: (resource, filterValue) => {
      const query = filterValue.trim().toLowerCase();
      return !query || [resource.alias, resource.name, resource.definitionKey, resource.sourceAlias ?? "", resource.status, formatStatus(resource.status, t)]
        .some((value) => value.toLowerCase().includes(query));
    },
    initialState: { sorting: [{ desc: false, id: "alias" }] },
    visibilityStorageKey: "tenancit.tenant-resources.table",
  });
  const domainTableLabels = useMemo(() => ({
    ...resourceTableLabels,
    noResults: t("tenantDetail.domainsEmpty"),
  }), [resourceTableLabels, t]);
  const domainColumns: ColumnDef<TenantDomain>[] = [
    {
      accessorKey: "hostname",
      header: ({ column }) => <DataTableColumnHeader column={column} label={t("common.hostname")} labels={resourceSortLabels} />,
      cell: ({ row }) => <code className="text-xs font-medium">{row.original.hostname}</code>,
      meta: { label: t("common.hostname") },
    },
    {
      cell: ({ row }) => can("tenant.write") ? <DropdownMenu>
        <DropdownMenuTrigger
          aria-label={t("common.actions")}
          onClick={(event) => event.stopPropagation()}
          render={<Button size="icon-sm" title={t("common.actions")} variant="ghost" />}
        >
          <EllipsisVertical />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          <DropdownMenuItem onClick={(event) => {
            event.stopPropagation();
            setEditingDomain(row.original);
            setEditingHostname(row.original.hostname);
            setEditDomainError("");
          }}>
            <Pencil /> {t("tenantDetail.renameDomain")}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={(event) => {
            event.stopPropagation();
            setPendingDomain(row.original);
          }} variant="destructive">
            <Trash2 /> {t("common.remove")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu> : null,
      enableHiding: false,
      enableSorting: false,
      header: t("common.actions"),
      id: "actions",
      meta: { align: "right", label: t("common.actions") },
      size: 80,
    },
  ];
  const { table: domainTable } = useDataTable({
    columns: domainColumns,
    data: domains,
    globalFilterFn: (domain, filterValue) => domain.hostname.toLowerCase().includes(filterValue.trim().toLowerCase()),
    initialState: { sorting: [{ desc: false, id: "hostname" }] },
    visibilityStorageKey: "tenancit.tenant-domains.table",
  });

  const overview = summarizeTenantOverview({
    domainCount: domains.length,
    resources,
    tenantStatus: tenant?.status ?? "inactive",
  });
  const available = definitions.filter((d) => d.status === "active");
  const picked = available.find((d) => d.key === pick) ?? null;
  const pickedDefinitionQuery = useQuery({
    ...adminQueryOptions.definition(picked?.id ?? ""),
    enabled: Boolean(picked),
  });
  const pickedFields = pickedDefinitionQuery.data?.fields ?? [];
  const sourceCandidates = resources.filter((resource) =>
    resource.status === "active" && !resource.linked && resource.definitionKey === picked?.key);
  const requiredFilled =
    picked && (Boolean(sourceResourceId)
      || pickedFields.filter((f) => f.required).every((f) => (values[f.key] ?? "").trim()));
  const resourceValuesValid = pickedFields.every((field) =>
    isValidTypedResourceValue(field.data_type, values[field.key] ?? ""));
  const resourceAliasValid = isValidResourceAlias(resourceAlias);
  const editingResourceAliasValid = isValidResourceAlias(editingResourceAlias);
  const duplicateAliasValid = isValidResourceAlias(duplicateAlias);
  const editSlugValid = isValidTenantSlug(edit.slug);
  const hostnameValid = isValidHostname(hostname);
  const editingHostnameValid = isValidHostname(editingHostname);
  const editingFieldValueValid = !editingField
    || isValidTypedResourceValue(editingField.dataType, editingFieldValue);
  const queryError = [
    tenantQuery.error,
    domainsQuery.error,
    resourcesQuery.error,
    definitionsQuery.error,
    reveal.error,
  ].find(Boolean);
  const visiblePageError = pageError || (queryError ? apiErrorMessage(queryError, t) : "");
  const visibleResourceError = resourceError
    || (pickedDefinitionQuery.error ? apiErrorMessage(pickedDefinitionQuery.error, t) : "");

  const updateTenantMutation = useMutation({
    mutationFn: (body: { name: string; slug: string; status: string }) => api.updateTenant(id, body),
    onSuccess: async () => {
      reveal.hide();
      await invalidateTenant(queryClient, id);
    },
  });
  const resourceStatusMutation = useMutation({
    mutationFn: ({ resourceId, status }: { resourceId: string; status: string }) =>
      api.setResourceStatus(id, resourceId, status),
    onSuccess: async () => {
      reveal.hide();
      await invalidateTenantResources(queryClient, id);
    },
  });
  const resourceIdentityMutation = useMutation({
    mutationFn: ({ resourceId, name, alias }: { resourceId: string; name: string; alias: string }) =>
      api.updateResourceIdentity(id, resourceId, { name, alias }),
    onSuccess: async () => {
      reveal.hide();
      await invalidateTenantResources(queryClient, id);
    },
  });
  const deleteResourceMutation = useMutation({
    mutationFn: (resourceId: string) => api.deleteResource(id, resourceId),
    onSuccess: async () => {
      reveal.hide();
      await invalidateTenantResources(queryClient, id);
    },
  });
  const duplicateResourceMutation = useMutation({
    mutationFn: ({ resourceId, alias }: { resourceId: string; alias: string }) =>
      api.duplicateResource(id, resourceId, alias, stableIdempotencyKey(duplicateAttempt, { resourceId, alias })),
    onSuccess: async () => {
      reveal.hide();
      await invalidateTenantResources(queryClient, id);
    },
  });
  const addDomainMutation = useMutation({
    mutationFn: (nextHostname: string) => api.addDomain(id, nextHostname),
    onSuccess: () => invalidateTenantDomains(queryClient, id),
  });
  const updateDomainMutation = useMutation({
    mutationFn: ({ domainId, hostname: nextHostname }: { domainId: string; hostname: string }) =>
      api.updateDomain(id, domainId, nextHostname),
    onSuccess: () => invalidateTenantDomains(queryClient, id),
  });
  const removeDomainMutation = useMutation({
    mutationFn: (domainId: string) => api.removeDomain(id, domainId),
    onSuccess: () => invalidateTenantDomains(queryClient, id),
  });
  const deleteTenantMutation = useMutation({
    mutationFn: () => api.deleteTenant(id),
    onSuccess: () => removeTenantQueries(queryClient, id),
  });

  function openEdit() {
    if (!tenant) return;
    setEdit({ name: tenant.name, slug: tenant.slug, status: tenant.status });
    setEditError("");
    setEditOpen(true);
  }
  async function saveEdit() {
    if (!edit.name.trim() || !editSlugValid) return;
    setEditError("");
    try {
      await updateTenantMutation.mutateAsync(edit);
      setEditOpen(false);
      toast.success(t("tenantDetail.updated"));
    } catch (e) {
      setEditError(apiErrorMessage(e, t));
    }
  }

  function choose(key: string) {
    setPick(key);
    setResourceName(available.find((definition) => definition.key === key)?.name ?? key);
    setResourceAlias(key);
    setSourceResourceId("");
    setValues({});
    setResourceError("");
  }
  async function saveResource() {
    if (!picked || !resourceName.trim() || !requiredFilled || !resourceAliasValid) return;
    setResourceError("");
    setIsCreatingResource(true);
    reveal.hide();
    try {
      // Secret-bearing values stay out of MutationCache and are erased after use.
      const request = {
        name: resourceName.trim(),
        alias: resourceAlias.trim(),
        definitionKey: picked.key,
        ...(sourceResourceId ? { sourceResourceId } : {}),
        values,
      };
      await api.createResource(id, request, stableIdempotencyKey(resourceCreateAttempt, request));
      await invalidateTenantResources(queryClient, id);
      setResOpen(false);
      setPick("");
      setResourceName("");
      setResourceAlias("");
      setSourceResourceId("");
      setValues({});
      toast.success(t("tenantDetail.resourceAdded"));
      resourceCreateAttempt.current = null;
    } catch (e) {
      setResourceError(apiErrorMessage(e, t));
    } finally {
      setIsCreatingResource(false);
    }
  }

  function openResourceIdentity(resource: TenantResource) {
    setEditingResource({ id: resource.id, name: resource.name, alias: resource.alias });
    setEditingResourceName(resource.name);
    setEditingResourceAlias(resource.alias);
    setEditResourceError("");
    setSelectedResourceId("");
  }

  async function saveResourceIdentity() {
    if (!editingResource || !editingResourceName.trim() || !editingResourceAliasValid) return;
    setEditResourceError("");
    try {
      await resourceIdentityMutation.mutateAsync({
        resourceId: editingResource.id,
        name: editingResourceName.trim(),
        alias: editingResourceAlias.trim(),
      });
      setEditingResource(null);
      toast.success(t("tenantDetail.resourceUpdated"));
    } catch (error) {
      setEditResourceError(apiErrorMessage(error, t));
    }
  }

  async function toggleResource(r: TenantResource) {
    const next = r.status === "active" ? "inactive" : "active";
    try {
      await resourceStatusMutation.mutateAsync({ resourceId: r.id, status: next });
      toast.success(next === "active" ? t("tenantDetail.resourceStatusActivated") : t("tenantDetail.resourceStatusDeactivated"));
    } catch (e) {
      setPageError(apiErrorMessage(e, t));
    }
  }

  function openDuplicateResource(resource: TenantResource) {
    setDuplicateAlias(`${resource.alias}.copy`);
    setDuplicateTarget({ id: resource.id, alias: resource.alias });
    setDuplicateError("");
    setSelectedResourceId("");
    setDuplicateOpen(true);
  }

  function requestResourceRemoval(resource: TenantResource) {
    setPendingResource({ id: resource.id, name: resource.alias });
    setSelectedResourceId("");
  }

  async function removeResource() {
    if (!pendingResource) return;
    try {
      await deleteResourceMutation.mutateAsync(pendingResource.id);
      setPendingResource(null);
      toast.success(t("tenantDetail.resourceRemoved"));
    } catch (e) {
      setPageError(apiErrorMessage(e, t));
    }
  }

  function openFieldEdit(field: ResourceFieldValue) {
    setEditingField(field);
    setEditingFieldValue(field.isSecret ? "" : field.value);
    setFieldEditError("");
  }

  async function saveFieldEdit() {
    if (!selectedResource || !editingField || !editingFieldValueValid || (editingField.required && !editingFieldValue)) return;
    setFieldEditError("");
    setIsUpdatingField(true);
    reveal.hide();
    try {
      await api.updateResourceField(id, selectedResource.id, editingField.key, editingFieldValue);
      await invalidateTenantResources(queryClient, id);
      setEditingField(null);
      setEditingFieldValue("");
      toast.success(t("tenantDetail.resourceFieldUpdated"));
    } catch (error) {
      setFieldEditError(apiErrorMessage(error, t));
    } finally {
      setIsUpdatingField(false);
    }
  }

  async function clearFieldOverride(field: ResourceFieldValue) {
    if (!selectedResource) return;
    try {
      reveal.hide();
      await api.clearResourceFieldOverride(id, selectedResource.id, field.key);
      await invalidateTenantResources(queryClient, id);
      toast.success(t("tenantDetail.resourceFieldInherited"));
    } catch (error) {
      setPageError(apiErrorMessage(error, t));
    }
  }

  async function duplicateSelectedResource() {
    if (!duplicateTarget || !duplicateAliasValid) return;
    setDuplicateError("");
    try {
      await duplicateResourceMutation.mutateAsync({ resourceId: duplicateTarget.id, alias: duplicateAlias.trim() });
      setDuplicateOpen(false);
      setDuplicateTarget(null);
      setDuplicateAlias("");
      duplicateAttempt.current = null;
      toast.success(t("tenantDetail.resourceDuplicated"));
    } catch (error) {
      setDuplicateError(apiErrorMessage(error, t));
    }
  }

  async function saveDomain() {
    if (!hostnameValid) return;
    setDomainError("");
    try {
      await addDomainMutation.mutateAsync(hostname.trim());
      setHostname("");
      setNewDomainOpen(false);
      toast.success(t("tenantDetail.domainAdded"));
    } catch (e) {
      setDomainError(apiErrorMessage(e, t));
    }
  }
  async function saveDomainEdit() {
    if (!editingDomain || !editingHostnameValid) return;
    setEditDomainError("");
    try {
      await updateDomainMutation.mutateAsync({
        domainId: editingDomain.id,
        hostname: editingHostname.trim(),
      });
      setEditingDomain(null);
      setEditingHostname("");
      toast.success(t("tenantDetail.domainUpdated"));
    } catch (error) {
      setEditDomainError(apiErrorMessage(error, t));
    }
  }
  async function removeDomain() {
    if (!pendingDomain) return;
    try {
      await removeDomainMutation.mutateAsync(pendingDomain.id);
      setPendingDomain(null);
      toast.success(t("tenantDetail.domainRemoved"));
    } catch (e) {
      setPageError(apiErrorMessage(e, t));
    }
  }

  async function toggleSecrets() {
    if (reveal.isRevealed) {
      reveal.hide();
      toast.success(t("tenantDetail.secretsDisabled"));
      return;
    }

    if (await reveal.show()) {
      toast.success(t("tenantDetail.secretsEnabled"));
    }
  }

  async function deleteTenantPermanently() {
    if (!tenant || !matchesTenantSlug(deleteConfirmation, tenant.slug)) return;
    try {
      reveal.hide();
      await deleteTenantMutation.mutateAsync();
      toast.success(t("tenantDetail.deleted"));
      await navigate({ to: "/tenants" });
    } catch (e) {
      setPageError(apiErrorMessage(e, t));
    }
  }

  if (!tenant) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16">
        {visiblePageError && <Alert variant="destructive"><AlertDescription>{visiblePageError}</AlertDescription></Alert>}
        <p className="text-muted-foreground">
          {tenantQuery.isPending
            ? t("common.loading")
            : visiblePageError
              ? t("tenantDetail.loadError")
              : t("tenantDetail.notFound")}
        </p>
        <Link to="/tenants"><Button variant="outline">{t("tenantDetail.backToTenants")}</Button></Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {visiblePageError && <Alert variant="destructive"><AlertDescription>{visiblePageError}</AlertDescription></Alert>}
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-2xl font-semibold tracking-tight">{tenant.name}</h1>
            <DomainStatus label={formatStatus(tenant.status, t)} value={tenant.status} />
          </div>
          <p className="text-sm text-muted-foreground">{t("tenantDetail.headerDescription", { slug: tenant.slug })}</p>
        </div>
        {can("tenant.write") && <Button variant="outline" onClick={openEdit}>
          <Settings2 data-icon="inline-start" /> {t("common.edit")}
        </Button>}
      </div>

      <Tabs
        className="gap-4"
        value={tab}
        onValueChange={(value) => setTab(value as "overview" | "resources" | "domains")}
      >
        <TabsList>
          <TabsTrigger value="overview">{t("tenantDetail.overviewTab")}</TabsTrigger>
          <TabsTrigger value="resources">{t("tenantDetail.resourcesTab")}</TabsTrigger>
          <TabsTrigger value="domains">{t("tenantDetail.domainsTab")}</TabsTrigger>
        </TabsList>

        <TabsContent className="flex flex-col gap-6" value="overview">
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
              hint={t("tenantDetail.readiness.requirementsHint", { count: overview.readyRequirementCount, total: 3 })}
              icon={overview.readiness === "ready" ? <CheckCircle2 /> : <CircleAlert />}
              label={t("tenantDetail.readiness.title")}
              value={t(`tenantDetail.readiness.${overview.readiness}`)}
            />
            <StatCard
              hint={domains.length > 0 ? t("overview.domainHint") : t("tenantDetail.readiness.missingDomain")}
              icon={domains.length > 0 ? <CheckCircle2 /> : <CircleAlert />}
              label={t("tenantDetail.readiness.domains")}
              value={domains.length}
            />
            <StatCard
              hint={overview.totalResourceCount > 0
                ? t("tenantDetail.readiness.resourcesHint")
                : t("tenantDetail.readiness.missingResource")}
              icon={overview.activeResourceCount > 0 ? <CheckCircle2 /> : <CircleAlert />}
              label={t("tenantDetail.readiness.resources")}
              value={`${overview.activeResourceCount}/${overview.totalResourceCount}`}
            />
            <StatCard
              hint={overview.incompleteResourceCount > 0
                ? t("tenantDetail.readiness.incompleteHint")
                : t("tenantDetail.readiness.completeHint")}
              icon={overview.incompleteResourceCount > 0 ? <CircleAlert /> : <CheckCircle2 />}
              label={t("tenantDetail.readiness.incompleteResources")}
              value={overview.incompleteResourceCount}
            />
          </div>

          <Card>
            <CardHeader>
              <CardTitle>{t("tenantDetail.attention.title")}</CardTitle>
              <CardDescription>{t("tenantDetail.attention.description")}</CardDescription>
            </CardHeader>
            <CardContent className="flex flex-col gap-2">
              {overview.attentionCodes.length === 0 ? <Empty className="border-0 py-4">
                <EmptyHeader>
                  <EmptyMedia className="size-10" variant="icon">
                    <CheckCircle2 className="size-5" />
                  </EmptyMedia>
                  <EmptyTitle>{t("tenantDetail.attention.emptyTitle")}</EmptyTitle>
                  <EmptyDescription>{t("tenantDetail.attention.emptyDescription")}</EmptyDescription>
                </EmptyHeader>
              </Empty> : overview.attentionCodes.map((code) => {
                const isTenantIssue = code === "inactive_tenant";
                const isDomainIssue = code === "missing_domain";
                const count = code === "inactive_resources"
                  ? overview.inactiveResourceCount
                  : code === "incomplete_resources"
                    ? overview.incompleteResourceCount
                    : undefined;
                return <Alert key={code} variant={isTenantIssue ? "destructive" : "default"}>
                  <TriangleAlert />
                  <AlertTitle>{t(`tenantDetail.attention.${code}.title`, { count: count ?? 0 })}</AlertTitle>
                  <AlertDescription>{t(`tenantDetail.attention.${code}.description`, { count: count ?? 0 })}</AlertDescription>
                  <AlertAction>
                    <Button
                      onClick={isTenantIssue ? openEdit : () => setTab(isDomainIssue ? "domains" : "resources")}
                      size="sm"
                      variant="outline"
                    >
                      {t(isTenantIssue ? "common.edit" : "common.review")}
                    </Button>
                  </AlertAction>
                </Alert>;
              })}
            </CardContent>
          </Card>

          {can("tenant.hard_delete") && <Card className="border-destructive/30">
            <CardHeader>
              <CardTitle className="text-base text-destructive">{t("tenantDetail.dangerZone.title")}</CardTitle>
              <p className="text-sm text-muted-foreground">{t("tenantDetail.dangerZone.description")}</p>
            </CardHeader>
            <CardContent>
              <Button
                onClick={() => {
                  setDeleteConfirmation("");
                  setDeleteOpen(true);
                }}
                variant="destructiveOutline"
              >
                <Trash2 data-icon="inline-start" /> {t("tenantDetail.deleteAction")}
              </Button>
            </CardContent>
          </Card>}
        </TabsContent>

        <TabsContent className="flex flex-col gap-4" value="resources">
          <DataTable
            labels={resourceTableLabels}
            onRowClick={(resource) => setSelectedResourceId(resource.id)}
            table={resourceTable}
          >
            <DataTableToolbar
              clearLabel={t("dataTable.clearFilters")}
              columnsLabel={t("dataTable.columns")}
              emptyLabel={t("dataTable.noResults")}
              resetLabel={t("dataTable.resetPreferences")}
              searchLabel={t("tenantDetail.resourcesSearch")}
              table={resourceTable}
              trailing={can("resource.write") ? <Button onClick={() => { setResourceError(""); setPick(""); setResOpen(true); }}>
                <Plus data-icon="inline-start" /> {t("tenantDetail.addResource")}
              </Button> : undefined}
            />
          </DataTable>
        </TabsContent>

        <TabsContent className="flex flex-col gap-4" value="domains">
          <DataTable labels={domainTableLabels} table={domainTable}>
            <DataTableToolbar
              clearLabel={t("dataTable.clearFilters")}
              columnsLabel={t("dataTable.columns")}
              emptyLabel={t("dataTable.noResults")}
              resetLabel={t("dataTable.resetPreferences")}
              searchLabel={t("tenantDetail.domainsSearch")}
              table={domainTable}
              trailing={can("tenant.write") ? <Button onClick={() => { setHostname(""); setDomainError(""); setNewDomainOpen(true); }}>
                <Plus data-icon="inline-start" /> {t("common.add")}
              </Button> : undefined}
            />
          </DataTable>
        </TabsContent>
      </Tabs>

      <Dialog
        open={Boolean(selectedResource)}
        onOpenChange={(open) => {
          if (!open && !editingField) setSelectedResourceId("");
        }}
      >
        <DialogContent className="flex max-h-[90dvh] flex-col overflow-hidden sm:max-w-3xl">
          {selectedResource && <>
            <DialogHeader className="shrink-0">
              <div className="flex items-center gap-2">
                <DialogTitle>{selectedResource.name}</DialogTitle>
                <DomainStatus label={formatStatus(selectedResource.status, t)} value={selectedResource.status} />
              </div>
              <DialogDescription>
                {selectedResource.linked
                  ? t("tenantDetail.resourceLinkedDescription", { source: selectedResource.sourceAlias ?? "" })
                  : t("tenantDetail.resourceDetailsDescription", { type: selectedResource.definitionKey, alias: selectedResource.alias })}
              </DialogDescription>
            </DialogHeader>
            <div className="min-h-0 overflow-y-auto rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("common.field")}</TableHead>
                    <TableHead>{t("common.value")}</TableHead>
                    <TableHead className="w-20 text-right">{t("common.actions")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {selectedResource.fields.map((field) => (
                    <TableRow key={field.key}>
                      <TableCell className="font-medium">{field.label || field.key}</TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2"><code className="text-xs">
                          {displaySecretValue({
                            isSecret: field.isSecret,
                            revealed: reveal.isRevealed,
                            value: field.value,
                          }) || "—"}
                        </code>{field.origin === "inherited" && <span className="text-xs text-muted-foreground">{t("tenantDetail.inherited")}</span>}</div>
                      </TableCell>
                      <TableCell className="text-right">
                        {can("resource.write") && <Button
                          aria-label={t("common.edit")}
                          onClick={() => openFieldEdit(field)}
                          size="icon-sm"
                          title={t("common.edit")}
                          variant="ghost"
                        >
                          <Pencil />
                        </Button>}
                        {can("resource.write") && selectedResource.linked && field.isOverride && <Button
                          aria-label={t("tenantDetail.useSource")}
                          onClick={() => { void clearFieldOverride(field); }}
                          size="icon-sm"
                          title={t("tenantDetail.useSource")}
                          variant="ghost"
                        ><RotateCcw /></Button>}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
            <DialogFooter className="flex-wrap sm:justify-between">
              <div className="flex flex-wrap gap-2">
                {can("resource.write") && <Button
                  onClick={() => { void toggleResource(selectedResource); }}
                  variant="outline"
                >
                  {selectedResource.status === "active"
                    ? <PowerOff data-icon="inline-start" />
                    : <Power data-icon="inline-start" />}
                  {selectedResource.status === "active" ? t("tenantDetail.deactivate") : t("tenantDetail.reactivate")}
                </Button>}
                {can("resource.write") && <Button
                  onClick={() => requestResourceRemoval(selectedResource)}
                  variant="destructive"
                >
                  <Trash2 data-icon="inline-start" /> {t("common.remove")}
                </Button>}
                {can("resource.write") && <Button
                  onClick={() => openDuplicateResource(selectedResource)}
                  variant="outline"
                >
                  <Copy data-icon="inline-start" /> {t("tenantDetail.duplicate")}
                </Button>}
              </div>
              <div className="flex flex-wrap gap-2">
                {can("secret.reveal") && <Button disabled={reveal.isLoading} onClick={() => { void toggleSecrets(); }} variant="outline">
                  {reveal.isRevealed
                    ? <EyeOff data-icon="inline-start" />
                    : <Eye data-icon="inline-start" />}
                  {reveal.isRevealed ? t("common.hide") : t("common.reveal")}
                </Button>}
                <DialogClose render={<Button variant="outline" />}>{t("common.close")}</DialogClose>
              </div>
            </DialogFooter>
          </>}
        </DialogContent>
      </Dialog>

      <Dialog
        open={Boolean(editingResource)}
        onOpenChange={(open) => {
          if (!open) {
            setEditingResource(null);
            setEditResourceError("");
          }
        }}
      >
        {editingResource && <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("tenantDetail.editResourceIdentityTitle")}</DialogTitle>
            <DialogDescription>{t("tenantDetail.editResourceIdentityDescription")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="resource-identity-name">{t("common.name")}</label>
              <Input
                aria-invalid={!editingResourceName.trim()}
                id="resource-identity-name"
                maxLength={120}
                onChange={(event) => setEditingResourceName(event.target.value)}
                value={editingResourceName}
              />
              <p className="text-xs text-muted-foreground">{t("tenantDetail.resourceNameHint")}</p>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="resource-identity-alias">{t("tenantDetail.resourceAlias")}</label>
              <Input
                aria-invalid={!editingResourceAliasValid}
                autoComplete="off"
                id="resource-identity-alias"
                onChange={(event) => setEditingResourceAlias(event.target.value)}
                value={editingResourceAlias}
              />
              <p className="text-xs text-muted-foreground">{t("tenantDetail.resourceAliasHint")}</p>
            </div>
            {editResourceError && <p className="text-sm text-destructive">{editResourceError}</p>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button
              disabled={!editingResourceName.trim() || !editingResourceAliasValid || resourceIdentityMutation.isPending}
              onClick={() => { void saveResourceIdentity(); }}
            >
              {t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>}
      </Dialog>

      <Dialog
        open={duplicateOpen}
        onOpenChange={(open) => {
          setDuplicateOpen(open);
          if (!open) {
            setDuplicateTarget(null);
            setDuplicateAlias("");
            setDuplicateError("");
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("tenantDetail.duplicateTitle")}</DialogTitle>
            <DialogDescription>{t("tenantDetail.duplicateDescription")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-1.5">
            <label className="text-sm font-medium" htmlFor="resource-duplicate-alias">
              {t("tenantDetail.resourceAlias")}
            </label>
            <Input
              aria-invalid={Boolean(duplicateError) || (!duplicateAliasValid && duplicateAlias.length > 0)}
              autoComplete="off"
              id="resource-duplicate-alias"
              onChange={(event) => setDuplicateAlias(event.target.value)}
              value={duplicateAlias}
            />
            <p className="text-xs text-muted-foreground">{t("tenantDetail.duplicateHint")}</p>
            {duplicateError && <p className="text-sm text-destructive">{duplicateError}</p>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button
              disabled={!duplicateAliasValid || duplicateResourceMutation.isPending}
              onClick={() => { void duplicateSelectedResource(); }}
            >
              {t("tenantDetail.duplicate")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={Boolean(editingField)}
        onOpenChange={(open) => {
          if (!open) {
            setEditingField(null);
            setEditingFieldValue("");
            setFieldEditError("");
          }
        }}
      >
        {editingField && <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("tenantDetail.editFieldTitle")}</DialogTitle>
            <DialogDescription>
              {t("tenantDetail.editFieldDescription", { field: editingField.label || editingField.key })}
            </DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-1.5">
            <label className="text-sm font-medium" htmlFor="resource-field-value">
              {editingField.label || editingField.key}
            </label>
            <Input
              aria-invalid={Boolean(fieldEditError) || !editingFieldValueValid}
              autoComplete="off"
              id="resource-field-value"
              onChange={(event) => setEditingFieldValue(event.target.value)}
              type={editingField.isSecret ? "password" : editingField.dataType === "int" ? "number" : "text"}
              value={editingFieldValue}
            />
            {!editingFieldValueValid && <p className="text-xs text-destructive">
              {t(editingField.dataType === "int" ? "validation.integer" : "validation.boolean")}
            </p>}
            {fieldEditError && <p className="text-sm text-destructive">{fieldEditError}</p>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button
              disabled={isUpdatingField || !editingFieldValueValid || (editingField.required && !editingFieldValue)}
              onClick={() => { void saveFieldEdit(); }}
            >
              {t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>}
      </Dialog>

      {/* Edit tenant */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="flex max-h-[90dvh] flex-col overflow-hidden">
          <DialogHeader className="shrink-0">
            <DialogTitle>{t("tenantDetail.editTitle")}</DialogTitle>
            <DialogDescription>{t("tenantDetail.editDescription")}</DialogDescription>
          </DialogHeader>
          <div className="min-h-0 space-y-4 overflow-y-auto px-1">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.name")}</label>
              <Input value={edit.name} onChange={(e) => setEdit({ ...edit, name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.slug")}</label>
              <Input
                aria-describedby="tenant-edit-slug-hint"
                aria-invalid={edit.slug.length > 0 && !editSlugValid}
                autoComplete="off"
                maxLength={63}
                value={edit.slug}
                onChange={(e) => setEdit({ ...edit, slug: e.target.value })}
              />
              <p className={edit.slug.length > 0 && !editSlugValid ? "text-xs text-destructive" : "text-xs text-muted-foreground"} id="tenant-edit-slug-hint">
                {t("validation.tenantSlug")}
              </p>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.status")}</label>
              <Select
                items={{
                  active: formatStatus("active", t),
                  inactive: formatStatus("inactive", t),
                }}
                value={edit.status}
                onValueChange={(value) => setEdit({ ...edit, status: String(value) })}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="active">{formatStatus("active", t)}</SelectItem>
                    <SelectItem value="inactive">{formatStatus("inactive", t)}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
            {editError && <div className="text-sm text-destructive">{editError}</div>}
          </div>
          <DialogFooter className="shrink-0">
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!edit.name.trim() || !editSlugValid} onClick={() => { void saveEdit(); }}>{t("common.save")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add resource */}
      <Dialog
        open={resOpen}
        onOpenChange={(open) => {
          setResOpen(open);
          if (!open) {
            setPick("");
            setResourceName("");
            setResourceAlias("");
            setSourceResourceId("");
            setValues({});
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("tenantDetail.addResourceDialog.title")}</DialogTitle>
            <DialogDescription>{t("tenantDetail.addResourceDialog.description")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {available.length === 0 ? (
              <div className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">
                {t("tenantDetail.addResourceDialog.empty")}
              </div>
            ) : (
              <>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">{t("tenantDetail.resourceType")}</label>
                  <Select
                    items={available.map((definition) => ({ label: definition.name, value: definition.key }))}
                    value={pick || null}
                    onValueChange={(value) => choose(value ? String(value) : "")}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder={t("tenantDetail.selectResource")} />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        {available.map((d) => (
                          <SelectItem key={d.key} value={d.key}>{d.name}</SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                </div>
                {picked && <div className="space-y-1.5">
                  <label className="text-sm font-medium">{t("common.name")}</label>
                  <Input
                    maxLength={120}
                    onChange={(event) => setResourceName(event.target.value)}
                    value={resourceName}
                  />
                  <p className="text-xs text-muted-foreground">{t("tenantDetail.resourceNameHint")}</p>
                </div>}
                {picked && <div className="space-y-1.5">
                  <label className="text-sm font-medium">{t("tenantDetail.resourceAlias")}</label>
                  <Input
                    aria-invalid={!resourceAliasValid && resourceAlias.length > 0}
                    onChange={(event) => setResourceAlias(event.target.value)}
                    placeholder="postgres.agility"
                    value={resourceAlias}
                  />
                  <p className="text-xs text-muted-foreground">{t("tenantDetail.resourceAliasHint")}</p>
                </div>}
                {picked && <div className="space-y-1.5">
                  <label className="text-sm font-medium">{t("tenantDetail.resourceOrigin")}</label>
                  <Select
                    items={[
                      { label: t("tenantDetail.independent"), value: "independent" },
                      ...sourceCandidates.map((resource) => ({
                        label: t("tenantDetail.linkTo", { alias: resource.alias }),
                        value: resource.id,
                      })),
                    ]}
                    value={sourceResourceId || "independent"}
                    onValueChange={(value) => setSourceResourceId(value === "independent" ? "" : String(value))}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent><SelectGroup>
                      <SelectItem value="independent">{t("tenantDetail.independent")}</SelectItem>
                      {sourceCandidates.map((resource) => <SelectItem key={resource.id} value={resource.id}>
                        {t("tenantDetail.linkTo", { alias: resource.alias })}
                      </SelectItem>)}
                    </SelectGroup></SelectContent>
                  </Select>
                  <p className="text-xs text-muted-foreground">{sourceResourceId
                    ? t("tenantDetail.linkedHint")
                    : t("tenantDetail.independentHint")}</p>
                </div>}
                {pickedFields.map((f) => (
                  <div key={f.key} className="space-y-1.5">
                    <label className="text-sm font-medium">
                      {f.label || f.key} {f.required && <span className="text-destructive">*</span>}
                    </label>
                    <Input
                      aria-invalid={!isValidTypedResourceValue(f.data_type, values[f.key] ?? "")}
                      type={f.is_secret ? "password" : f.data_type === "int" ? "number" : "text"}
                      placeholder={sourceResourceId ? t("tenantDetail.inheritPlaceholder") : f.key}
                      value={values[f.key] ?? ""}
                      onChange={(e) => setValues({ ...values, [f.key]: e.target.value })}
                    />
                    {!isValidTypedResourceValue(f.data_type, values[f.key] ?? "") && <p className="text-xs text-destructive">
                      {t(f.data_type === "int" ? "validation.integer" : "validation.boolean")}
                    </p>}
                  </div>
                ))}
              </>
            )}
            {visibleResourceError && <div className="text-sm text-destructive">{visibleResourceError}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button
              disabled={!picked || !resourceName.trim() || !resourceAliasValid || !requiredFilled || !resourceValuesValid || isCreatingResource}
              onClick={() => { void saveResource(); }}
            >
              {t("tenantDetail.saveResource")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add domain */}
      <Dialog open={newDomainOpen} onOpenChange={setNewDomainOpen}>
        {newDomainOpen && <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("tenantDetail.newDomainTitle")}</DialogTitle>
            <DialogDescription>{t("tenantDetail.newDomainDescription")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-1.5">
            <label className="text-sm font-medium">{t("common.hostname")}</label>
            <Input
              aria-describedby="tenant-domain-new-hint"
              aria-invalid={hostname.length > 0 && !hostnameValid}
              autoComplete="off"
              maxLength={253}
              placeholder="app.cliente.com"
              value={hostname}
              onChange={(e) => setHostname(e.target.value)}
            />
            <p className={hostname.length > 0 && !hostnameValid ? "text-xs text-destructive" : "text-xs text-muted-foreground"} id="tenant-domain-new-hint">
              {t("validation.hostname")}
            </p>
            {domainError && <div className="text-sm text-destructive">{domainError}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!hostnameValid} onClick={() => { void saveDomain(); }}>{t("common.add")}</Button>
          </DialogFooter>
        </DialogContent>}
      </Dialog>

      {/* Rename domain */}
      <Dialog open={Boolean(editingDomain)} onOpenChange={(open) => {
        if (!open && !updateDomainMutation.isPending) {
          setEditingDomain(null);
          setEditingHostname("");
          setEditDomainError("");
        }
      }}>
        {editingDomain && <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("tenantDetail.editDomainTitle")}</DialogTitle>
            <DialogDescription>{t("tenantDetail.editDomainDescription")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-1.5">
            <label className="text-sm font-medium" htmlFor="tenant-domain-edit-hostname">
              {t("common.hostname")}
            </label>
            <Input
              aria-describedby="tenant-domain-edit-hint"
              aria-invalid={editingHostname.length > 0 && !editingHostnameValid}
              autoComplete="off"
              id="tenant-domain-edit-hostname"
              maxLength={253}
              placeholder="app.cliente.com"
              value={editingHostname}
              onChange={(event) => setEditingHostname(event.target.value)}
            />
            <p className={editingHostname.length > 0 && !editingHostnameValid ? "text-xs text-destructive" : "text-xs text-muted-foreground"} id="tenant-domain-edit-hint">
              {t("validation.hostname")}
            </p>
            {editDomainError && <div className="text-sm text-destructive">{editDomainError}</div>}
          </div>
          <DialogFooter>
            <Button disabled={updateDomainMutation.isPending} onClick={() => {
              setEditingDomain(null);
              setEditingHostname("");
              setEditDomainError("");
            }} variant="outline">
              {t("common.cancel")}
            </Button>
            <Button
              disabled={!editingHostnameValid || updateDomainMutation.isPending}
              onClick={() => { void saveDomainEdit(); }}
            >
              {t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>}
      </Dialog>

      <ConfirmDialog
        cancelLabel={t("common.cancel")}
        confirmLabel={t("common.removeConfirm")}
        description={t("tenantDetail.confirmRemoveResourceDescription", { resourceName: pendingResource?.name ?? "" })}
        onConfirm={() => { void removeResource(); }}
        onOpenChange={(open) => {
          if (!open) setPendingResource(null);
        }}
        open={Boolean(pendingResource)}
        title={t("tenantDetail.confirmRemoveResourceTitle")}
      />

      <ConfirmDialog
        cancelLabel={t("common.cancel")}
        confirmLabel={t("common.removeConfirm")}
        description={t("tenantDetail.confirmRemoveDomainDescription", { hostname: pendingDomain?.hostname ?? "" })}
        onConfirm={() => { void removeDomain(); }}
        onOpenChange={(open) => {
          if (!open) setPendingDomain(null);
        }}
        open={Boolean(pendingDomain)}
        title={t("tenantDetail.confirmRemoveDomainTitle")}
      />

      <ConfirmDialog
        cancelLabel={t("common.cancel")}
        confirmDisabled={
          deleteTenantMutation.isPending || !matchesTenantSlug(deleteConfirmation, tenant.slug)
        }
        confirmLabel={t("tenantDetail.deleteConfirm")}
        description={t("tenantDetail.deleteDescription", { slug: tenant.slug })}
        onConfirm={() => { void deleteTenantPermanently(); }}
        onOpenChange={(open) => {
          setDeleteOpen(open);
          if (!open && !deleteTenantMutation.isPending) setDeleteConfirmation("");
        }}
        open={deleteOpen}
        title={t("tenantDetail.deleteTitle")}
      >
        <div className="space-y-1.5">
          <label className="text-sm font-medium" htmlFor="tenant-delete-confirmation">
            {t("tenantDetail.deleteInputLabelPrefix")}{" "}
            <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs font-semibold text-foreground">
              {tenant.slug}
            </code>{" "}
            {t("tenantDetail.deleteInputLabelSuffix")}
          </label>
          <Input
            autoComplete="off"
            disabled={deleteTenantMutation.isPending}
            id="tenant-delete-confirmation"
            onChange={(event) => setDeleteConfirmation(event.target.value)}
            placeholder={t("tenantDetail.deleteInputPlaceholder")}
            value={deleteConfirmation}
          />
        </div>
      </ConfirmDialog>
    </div>
  );
}
