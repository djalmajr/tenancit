import { createRoute, Link } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import {
  CheckCircle2,
  Building2, ChevronRight, Database, Eye, EyeOff, Globe, Plus, Trash2, Lock,
  CircleAlert, Power, PowerOff, Settings2,
} from "lucide-react";
import { Route as rootRoute } from "./__root";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Table, THead, TBody, TR, TH, TD } from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
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
import { formatStatus, useI18n } from "@/lib/i18n";
import {
  api, type Tenant, type TenantDomain, type TenantResource, type Definition, type Field, type ApiClient,
} from "@/lib/api";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tenants/$id",
  component: TenantDetail,
});

function TenantDetail() {
  const { t } = useI18n();
  const { id } = Route.useParams();
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [tab, setTab] = useState<"resources" | "domains">("resources");
  const [domains, setDomains] = useState<TenantDomain[]>([]);
  const [resources, setResources] = useState<TenantResource[]>([]);
  const [definitions, setDefinitions] = useState<Definition[]>([]);
  const [apiClients, setApiClients] = useState<ApiClient[]>([]);

  const [hostname, setHostname] = useState("");
  const [newDomainOpen, setNewDomainOpen] = useState(false);

  const [editOpen, setEditOpen] = useState(false);
  const [edit, setEdit] = useState({ name: "", slug: "", status: "active" });

  const [resOpen, setResOpen] = useState(false);
  const [pick, setPick] = useState("");
  const [pickedFields, setPickedFields] = useState<Field[]>([]);
  const [values, setValues] = useState<Record<string, string>>({});
  const [domainError, setDomainError] = useState("");
  const [editError, setEditError] = useState("");
  const [pageError, setPageError] = useState("");
  const [resourceError, setResourceError] = useState("");
  const [secretsRevealed, setSecretsRevealed] = useState(false);
  const [pendingDomain, setPendingDomain] = useState<TenantDomain | null>(null);
  const [pendingResource, setPendingResource] = useState<TenantResource | null>(null);

  const load = useCallback(async (reveal: boolean) => {
    try {
      const [t, d, r] = await Promise.all([
        api.getTenant(id),
        api.listDomains(id),
        api.listTenantResources(id, reveal),
      ]);
      setPageError("");
      setTenant(t);
      setDomains(d ?? []);
      setResources(r ?? []);
    } catch (e) {
      setPageError(String(e));
      setTenant(null);
    }
  }, [id]);

  useEffect(() => {
    void load(false);
    api.listDefinitions().then((d) => setDefinitions(d ?? [])).catch((e) => setPageError(String(e)));
    api.listAPIClients().then((c) => setApiClients(c ?? [])).catch((e) => setPageError(String(e)));
  }, [load]);

  const activeKeys = new Set(resources.filter((r) => r.status === "active").map((r) => r.definitionKey));
  const activeResourceCount = resources.filter((r) => r.status === "active").length;
  const activeApiKeyCount = apiClients.filter((c) => c.status === "active").length;
  const available = definitions.filter((d) => d.status === "active" && !activeKeys.has(d.key));
  const picked = available.find((d) => d.key === pick) ?? null;
  const requiredFilled =
    picked && pickedFields.filter((f) => f.required).every((f) => (values[f.key] ?? "").trim());

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
      await api.updateTenant(id, edit);
      setEditOpen(false);
      toast.success(t("tenantDetail.updated"));
      await load(secretsRevealed);
    } catch (e) {
      setEditError(String(e));
    }
  }

  async function choose(key: string) {
    setPick(key);
    setValues({});
    const def = available.find((d) => d.key === key);
    if (!def) return setPickedFields([]);
    try {
      const detail = await api.getDefinition(def.id);
      setPickedFields(detail?.fields ?? []);
      setResourceError("");
    } catch (e) {
      setPickedFields([]);
      setResourceError(String(e));
    }
  }
  async function saveResource() {
    if (!picked || !requiredFilled) return;
    setResourceError("");
    try {
      await api.createResource(id, { definitionKey: picked.key, values });
      setResOpen(false);
      setPick("");
      setValues({});
      setPickedFields([]);
      toast.success(t("tenantDetail.resourceAdded"));
      await load(secretsRevealed);
    } catch (e) {
      setResourceError(String(e));
    }
  }

  async function toggleResource(r: TenantResource) {
    const next = r.status === "active" ? "inactive" : "active";
    try {
      await api.setResourceStatus(id, r.id, next);
      toast.success(next === "active" ? t("tenantDetail.resourceStatusActivated") : t("tenantDetail.resourceStatusDeactivated"));
      await load(secretsRevealed);
    } catch (e) {
      setPageError(String(e));
    }
  }
  async function removeResource() {
    if (!pendingResource) return;
    try {
      await api.deleteResource(id, pendingResource.id);
      setPendingResource(null);
      toast.success(t("tenantDetail.resourceRemoved"));
      await load(secretsRevealed);
    } catch (e) {
      setPageError(String(e));
    }
  }

  async function saveDomain() {
    if (!hostname.trim()) return;
    setDomainError("");
    try {
      await api.addDomain(id, hostname.trim());
      setHostname("");
      setNewDomainOpen(false);
      toast.success(t("tenantDetail.domainAdded"));
      await load(secretsRevealed);
    } catch (e) {
      setDomainError(String(e));
    }
  }
  async function removeDomain() {
    if (!pendingDomain) return;
    try {
      await api.removeDomain(id, pendingDomain.id);
      setPendingDomain(null);
      toast.success(t("tenantDetail.domainRemoved"));
      await load(secretsRevealed);
    } catch (e) {
      setPageError(String(e));
    }
  }

  async function toggleSecrets() {
    const next = !secretsRevealed;
    setSecretsRevealed(next);
    toast.success(next ? t("tenantDetail.secretsEnabled") : t("tenantDetail.secretsDisabled"));
    await load(next);
  }

  if (!tenant) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16">
        {pageError && <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{pageError}</div>}
        <p className="text-muted-foreground">{pageError ? t("tenantDetail.loadError") : t("tenantDetail.notFound")}</p>
        <Link to="/tenants"><Button variant="outline">{t("tenantDetail.backToTenants")}</Button></Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {pageError && <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{pageError}</div>}
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link to="/tenants" className="hover:text-foreground">{t("tenants.title")}</Link>
        <ChevronRight className="size-3.5" />
        <span className="text-foreground">{tenant.name}</span>
      </div>

      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="flex size-11 items-center justify-center rounded-lg bg-muted">
            <Building2 className="size-5" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{tenant.name}</h1>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <code>{tenant.slug}</code>
              <Badge variant={tenant.status === "active" ? "default" : "secondary"}>{formatStatus(tenant.status, t)}</Badge>
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

      <div className="flex gap-1 border-b">
        <TabBtn active={tab === "resources"} onClick={() => setTab("resources")} icon={<Database className="size-4" />}>
          {t("tenantDetail.resourcesTab")}
        </TabBtn>
        <TabBtn active={tab === "domains"} onClick={() => setTab("domains")} icon={<Globe className="size-4" />}>
          {t("tenantDetail.domainsTab")}
        </TabBtn>
      </div>

      {tab === "resources" && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <div className="flex gap-2">
              <Button variant="outline" onClick={toggleSecrets}>
                {secretsRevealed ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                {secretsRevealed ? t("tenantDetail.hideSecrets") : t("tenantDetail.enableSecretReveal")}
              </Button>
              <Button onClick={() => { setResourceError(""); setPick(""); setPickedFields([]); setResOpen(true); }}>
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
                    <Badge variant={r.status === "active" ? "default" : "secondary"}>{formatStatus(r.status, t)}</Badge>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      title={r.status === "active" ? t("tenantDetail.deactivate") : t("tenantDetail.reactivate")}
                      onClick={() => toggleResource(r)}
                    >
                      {r.status === "active" ? <Power className="size-4" /> : <PowerOff className="size-4" />}
                    </Button>
                    <Button variant="ghost" size="icon-sm" title={t("common.remove")} onClick={() => setPendingResource(r)}>
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                </CardHeader>
                <CardContent>
                  <Table>
                    <THead>
                      <TR>
                        <TH>{t("common.field")}</TH>
                        <TH>{t("common.value")}</TH>
                        <TH className="w-20">{t("common.secret")}</TH>
                      </TR>
                    </THead>
                    <TBody>
                      {r.fields.map((f) => (
                        <TR key={f.key}>
                          <TD className="font-medium">
                            {f.label || f.key}
                            {f.isSecret && <Lock className="ml-1 inline size-3 text-amber-500" />}
                          </TD>
                          <TD>
                            {f.isSecret && secretsRevealed ? (
                              <RevealValue hideLabel={t("common.hide")} showLabel={t("common.reveal")} value={f.value} />
                            ) : (
                              <code className="text-xs">{f.value || "—"}</code>
                            )}
                          </TD>
                          <TD>{f.isSecret && <Badge variant="secondary" className="text-amber-600">{t("common.secret")}</Badge>}</TD>
                        </TR>
                      ))}
                    </TBody>
                  </Table>
                </CardContent>
              </Card>
            ))
          )}
        </div>
      )}

      {tab === "domains" && (
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
                <THead>
                  <TR>
                    <TH>{t("common.hostname")}</TH>
                    <TH className="w-16 text-right">{t("common.actions")}</TH>
                  </TR>
                </THead>
                <TBody>
                  {domains.map((d) => (
                    <TR key={d.id}>
                      <TD className="font-medium"><code className="text-xs">{d.hostname}</code></TD>
                      <TD className="text-right">
                        <Button variant="ghost" size="icon-sm" title={t("common.remove")} onClick={() => setPendingDomain(d)}>
                          <Trash2 className="size-4" />
                        </Button>
                      </TD>
                    </TR>
                  ))}
                </TBody>
              </Table>
            )}
          </CardContent>
        </Card>
      )}

      {/* Edit tenant */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("tenantDetail.editTitle")}</DialogTitle>
            <DialogDescription>{t("tenantDetail.editDescription")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
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
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!edit.name.trim() || !edit.slug.trim()} onClick={saveEdit}>{t("common.save")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add resource */}
      <Dialog open={resOpen} onOpenChange={setResOpen}>
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
            {resourceError && <div className="text-sm text-destructive">{resourceError}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!picked || !requiredFilled} onClick={saveResource}>{t("tenantDetail.saveResource")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Add domain */}
      <Dialog open={newDomainOpen} onOpenChange={setNewDomainOpen}>
        <DialogContent>
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
            <Button disabled={!hostname.trim()} onClick={saveDomain}>{t("common.add")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        cancelLabel={t("common.cancel")}
        confirmLabel={t("common.removeConfirm")}
        description={t("tenantDetail.confirmRemoveResourceDescription", { resourceName: pendingResource?.name ?? "" })}
        onConfirm={removeResource}
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
        onConfirm={removeDomain}
        onOpenChange={(open) => {
          if (!open) setPendingDomain(null);
        }}
        open={Boolean(pendingDomain)}
        title={t("tenantDetail.confirmRemoveDomainTitle")}
      />
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

function TabBtn({
  active, onClick, icon, children,
}: {
  active: boolean; onClick: () => void; icon: React.ReactNode; children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={`flex items-center gap-2 border-b-2 px-3 py-2 text-sm font-medium transition-colors ${
        active ? "border-primary text-foreground" : "border-transparent text-muted-foreground hover:text-foreground"
      }`}
    >
      {icon}
      {children}
    </button>
  );
}
