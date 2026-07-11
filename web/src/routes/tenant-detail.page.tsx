import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getRouteApi, Link, useNavigate } from "@tanstack/react-router";
import { useRef, useState } from "react";
import {
  CheckCircle2,
  Building2, ChevronRight, Database, Eye, EyeOff, Globe, Plus, Trash2, Lock,
  CircleAlert, Power, PowerOff, Settings2,
} from "lucide-react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
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
import { DomainStatus } from "@/components/domain-status";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { toast } from "sonner";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { RevealValue } from "@/components/ui/reveal-value";
import { apiErrorMessage, formatStatus, useI18n } from "@/lib/i18n";
import { displaySecretValue } from "@/lib/secret-display";
import { matchesTenantSlug } from "@/lib/tenant-delete";
import {
  api, type TenantDomain, type TenantResource,
} from "@/lib/api";
import { stableIdempotencyKey, type IdempotencyAttempt } from "@/lib/idempotency";
import {
  invalidateTenant,
  invalidateTenantDomains,
  invalidateTenantResources,
  removeTenantQueries,
} from "@/lib/query-invalidation";
import { adminQueryOptions } from "@/lib/query-options";
import { useTransientResourceReveal } from "@/lib/use-transient-resource-reveal";

const routeApi = getRouteApi("/tenants/$id");

export default function TenantDetail() {
  const { t } = useI18n();
  const { id } = routeApi.useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<"resources" | "domains">("resources");

  const [hostname, setHostname] = useState("");
  const [newDomainOpen, setNewDomainOpen] = useState(false);

  const [editOpen, setEditOpen] = useState(false);
  const [edit, setEdit] = useState({ name: "", slug: "", status: "active" });

  const [resOpen, setResOpen] = useState(false);
  const [pick, setPick] = useState("");
  const [values, setValues] = useState<Record<string, string>>({});
  const [domainError, setDomainError] = useState("");
  const [editError, setEditError] = useState("");
  const [isCreatingResource, setIsCreatingResource] = useState(false);
  const [pageError, setPageError] = useState("");
  const [resourceError, setResourceError] = useState("");
  const resourceCreateAttempt = useRef<IdempotencyAttempt>(null);
  const [pendingDomain, setPendingDomain] = useState<TenantDomain | null>(null);
  const [pendingResource, setPendingResource] = useState<Pick<TenantResource, "id" | "name"> | null>(null);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [deleteConfirmation, setDeleteConfirmation] = useState("");

  const tenantQuery = useQuery(adminQueryOptions.tenant(id));
  const domainsQuery = useQuery(adminQueryOptions.tenantDomains(id));
  const resourcesQuery = useQuery(adminQueryOptions.tenantResources(id));
  const definitionsQuery = useQuery(adminQueryOptions.definitions());
  const apiClientsQuery = useQuery(adminQueryOptions.apiClients());
  const reveal = useTransientResourceReveal(id);
  const tenant = tenantQuery.data;
  const domains = domainsQuery.data ?? [];
  const resources = reveal.resources ?? resourcesQuery.data ?? [];
  const definitions = definitionsQuery.data ?? [];
  const apiClients = apiClientsQuery.data ?? [];

  const activeKeys = new Set(resources.filter((r) => r.status === "active").map((r) => r.definitionKey));
  const activeResourceCount = resources.filter((r) => r.status === "active").length;
  const activeApiKeyCount = apiClients.filter((c) => c.status === "active").length;
  const available = definitions.filter((d) => d.status === "active" && !activeKeys.has(d.key));
  const picked = available.find((d) => d.key === pick) ?? null;
  const pickedDefinitionQuery = useQuery({
    ...adminQueryOptions.definition(picked?.id ?? ""),
    enabled: Boolean(picked),
  });
  const pickedFields = pickedDefinitionQuery.data?.fields ?? [];
  const requiredFilled =
    picked && pickedFields.filter((f) => f.required).every((f) => (values[f.key] ?? "").trim());
  const queryError = [
    tenantQuery.error,
    domainsQuery.error,
    resourcesQuery.error,
    definitionsQuery.error,
    apiClientsQuery.error,
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
  const deleteResourceMutation = useMutation({
    mutationFn: (resourceId: string) => api.deleteResource(id, resourceId),
    onSuccess: async () => {
      reveal.hide();
      await invalidateTenantResources(queryClient, id);
    },
  });
  const addDomainMutation = useMutation({
    mutationFn: (nextHostname: string) => api.addDomain(id, nextHostname),
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
    if (!edit.name.trim() || !edit.slug.trim()) return;
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
    setValues({});
    setResourceError("");
  }
  async function saveResource() {
    if (!picked || !requiredFilled) return;
    setResourceError("");
    setIsCreatingResource(true);
    reveal.hide();
    try {
      // Secret-bearing values stay out of MutationCache and are erased after use.
      const request = { definitionKey: picked.key, values };
      await api.createResource(id, request, stableIdempotencyKey(resourceCreateAttempt, request));
      await invalidateTenantResources(queryClient, id);
      setResOpen(false);
      setPick("");
      setValues({});
      toast.success(t("tenantDetail.resourceAdded"));
      resourceCreateAttempt.current = null;
    } catch (e) {
      setResourceError(apiErrorMessage(e, t));
    } finally {
      setIsCreatingResource(false);
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

  async function saveDomain() {
    if (!hostname.trim()) return;
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
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link to="/tenants" className="hover:text-foreground">{t("tenants.title")}</Link>
        <ChevronRight className="size-3.5" />
        <span className="text-foreground">{tenant.name}</span>
      </div>

      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex size-11 items-center justify-center rounded-lg bg-muted">
            <Building2 className="size-5" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{tenant.name}</h1>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <code>{tenant.slug}</code>
              <DomainStatus label={formatStatus(tenant.status, t)} value={tenant.status} />
            </div>
          </div>
        </div>
        <Button variant="outline" onClick={openEdit}>
          <Settings2 className="size-4" /> {t("common.edit")}
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("tenantDetail.readiness.title")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("tenantDetail.readiness.description")}</p>
        </CardHeader>
        <CardContent className="grid gap-3 md:grid-cols-4">
          <ReadinessItem
            label={t("tenantDetail.readiness.tenant")}
            ready={tenant.status === "active"}
            value={formatStatus(tenant.status, t)}
          />
          <ReadinessItem
            label={t("tenantDetail.readiness.domains")}
            ready={domains.length > 0}
            value={domains.length > 0 ? String(domains.length) : t("tenantDetail.readiness.missingDomain")}
          />
          <ReadinessItem
            label={t("tenantDetail.readiness.activeResources")}
            ready={activeResourceCount > 0}
            value={activeResourceCount > 0 ? String(activeResourceCount) : t("tenantDetail.readiness.missingResource")}
          />
          <ReadinessItem
            label={t("tenantDetail.readiness.activeApiKeys")}
            ready={activeApiKeyCount > 0}
            value={activeApiKeyCount > 0 ? String(activeApiKeyCount) : t("tenantDetail.readiness.missingApiKey")}
          />
        </CardContent>
      </Card>

      <Tabs value={tab} onValueChange={(value) => setTab(value as "resources" | "domains")}>
        <TabsList className="border-b" variant="line">
          <TabsTrigger value="resources"><Database />{t("tenantDetail.resourcesTab")}</TabsTrigger>
          <TabsTrigger value="domains"><Globe />{t("tenantDetail.domainsTab")}</TabsTrigger>
        </TabsList>

        <TabsContent className="space-y-4" value="resources">
          <div className="flex justify-end">
            <div className="flex gap-2">
              <Button disabled={reveal.isLoading} variant="outline" onClick={() => { void toggleSecrets(); }}>
                {reveal.isRevealed ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                {reveal.isRevealed ? t("tenantDetail.hideSecrets") : t("tenantDetail.enableSecretReveal")}
              </Button>
              <Button onClick={() => { setResourceError(""); setPick(""); setResOpen(true); }}>
                <Plus className="size-4" /> {t("tenantDetail.addResource")}
              </Button>
            </div>
          </div>
          {resources.length === 0 ? (
            <Card className="p-10 text-center">
              <Database className="mx-auto size-6 text-muted-foreground" />
              <p className="mt-2 font-medium">{t("tenantDetail.noResource")}</p>
              <p className="text-sm text-muted-foreground">{t("tenantDetail.noResourceDescription")}</p>
            </Card>
          ) : (
            resources.map((r) => (
              <Card key={r.id}>
                <CardHeader className="flex flex-row items-center justify-between space-y-0">
                  <div className="flex items-center gap-2">
                    <Database className="size-4 text-muted-foreground" />
                    <CardTitle className="text-base">{r.name}</CardTitle>
                    <code className="text-xs text-muted-foreground">{r.definitionKey}</code>
                  </div>
                  <div className="flex items-center gap-2">
                    <DomainStatus label={formatStatus(r.status, t)} value={r.status} />
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      title={r.status === "active" ? t("tenantDetail.deactivate") : t("tenantDetail.reactivate")}
                      onClick={() => { void toggleResource(r); }}
                    >
                      {r.status === "active" ? <Power className="size-4" /> : <PowerOff className="size-4" />}
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      title={t("common.remove")}
                      onClick={() => setPendingResource({ id: r.id, name: r.name })}
                    >
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                </CardHeader>
                <CardContent>
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t("common.field")}</TableHead>
                        <TableHead>{t("common.value")}</TableHead>
                        <TableHead className="w-20">{t("common.secret")}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {r.fields.map((f) => (
                        <TableRow key={f.key}>
                          <TableCell className="font-medium">
                            {f.label || f.key}
                            {f.isSecret && <Lock className="ml-1 inline size-3 text-amber-500" />}
                          </TableCell>
                          <TableCell>
                            {f.isSecret && reveal.isRevealed ? (
                              <RevealValue hideLabel={t("common.hide")} showLabel={t("common.reveal")} value={f.value} />
                            ) : (
                              <code className="text-xs">
                                {displaySecretValue({
                                  isSecret: f.isSecret,
                                  revealed: reveal.isRevealed,
                                  value: f.value,
                                }) || "—"}
                              </code>
                            )}
                          </TableCell>
                          <TableCell>{f.isSecret && <Badge variant="secondary" className="text-amber-600">{t("common.secret")}</Badge>}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            ))
          )}
        </TabsContent>

        <TabsContent value="domains">
          <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <div>
              <CardTitle className="text-base">{t("tenantDetail.domainsTab")}</CardTitle>
              <p className="text-sm text-muted-foreground">{t("tenantDetail.domainsDescription")}</p>
            </div>
            <Button variant="outline" size="sm" onClick={() => { setHostname(""); setDomainError(""); setNewDomainOpen(true); }}>
              <Plus className="size-4" /> {t("common.add")}
            </Button>
          </CardHeader>
          <CardContent>
            {domains.length === 0 ? (
              <p className="text-sm text-muted-foreground">{t("tenantDetail.domainsEmpty")}</p>
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t("common.hostname")}</TableHead>
                    <TableHead className="w-16 text-right">{t("common.actions")}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {domains.map((d) => (
                    <TableRow key={d.id}>
                      <TableCell className="font-medium"><code className="text-xs">{d.hostname}</code></TableCell>
                      <TableCell className="text-right">
                        <Button variant="ghost" size="icon-sm" title={t("common.remove")} onClick={() => setPendingDomain(d)}>
                          <Trash2 className="size-4" />
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <Card className="border-destructive/30">
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
            variant="destructive"
          >
            <Trash2 className="size-4" /> {t("tenantDetail.deleteAction")}
          </Button>
        </CardContent>
      </Card>

      {/* Edit tenant */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent className="flex max-h-[90dvh] flex-col overflow-hidden">
          <DialogHeader className="shrink-0">
            <DialogTitle>{t("tenantDetail.editTitle")}</DialogTitle>
            <DialogDescription>{t("tenantDetail.editDescription")}</DialogDescription>
          </DialogHeader>
          <div className="min-h-0 space-y-4 overflow-y-auto pr-1">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.name")}</label>
              <Input value={edit.name} onChange={(e) => setEdit({ ...edit, name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.slug")}</label>
              <Input value={edit.slug} onChange={(e) => setEdit({ ...edit, slug: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.status")}</label>
              <Select value={edit.status} onValueChange={(value) => setEdit({ ...edit, status: String(value) })}>
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
            <Button disabled={!edit.name.trim() || !edit.slug.trim()} onClick={() => { void saveEdit(); }}>{t("common.save")}</Button>
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
                  <Select value={pick || null} onValueChange={(value) => choose(value ? String(value) : "")}>
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
                {pickedFields.map((f) => (
                  <div key={f.key} className="space-y-1.5">
                    <label className="text-sm font-medium">
                      {f.label || f.key} {f.required && <span className="text-destructive">*</span>}
                      {f.is_secret && <Lock className="ml-1 inline size-3 text-amber-500" />}
                    </label>
                    <Input
                      type={f.is_secret ? "password" : f.data_type === "int" ? "number" : "text"}
                      placeholder={f.key}
                      value={values[f.key] ?? ""}
                      onChange={(e) => setValues({ ...values, [f.key]: e.target.value })}
                    />
                  </div>
                ))}
              </>
            )}
            {visibleResourceError && <div className="text-sm text-destructive">{visibleResourceError}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button
              disabled={!picked || !requiredFilled || isCreatingResource}
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
            <Input placeholder="app.cliente.com" value={hostname} onChange={(e) => setHostname(e.target.value)} />
            {domainError && <div className="text-sm text-destructive">{domainError}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!hostname.trim()} onClick={() => { void saveDomain(); }}>{t("common.add")}</Button>
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
            {t("tenantDetail.deleteInputLabel", { slug: tenant.slug })}
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

function ReadinessItem({
  label,
  ready,
  value,
}: {
  label: string;
  ready: boolean;
  value: string;
}) {
  return (
    <div className="flex min-h-20 items-start gap-3 rounded-md border p-3">
      {ready ? (
        <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-green-600" />
      ) : (
        <CircleAlert className="mt-0.5 size-4 shrink-0 text-amber-600" />
      )}
      <div className="min-w-0">
        <div className="text-sm font-medium">{label}</div>
        <div className="break-words text-sm text-muted-foreground">{value}</div>
      </div>
    </div>
  );
}
