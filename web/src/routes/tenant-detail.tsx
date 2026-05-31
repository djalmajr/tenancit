import { createRoute, Link } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import {
  Building2, ChevronRight, Database, Globe, Plus, Trash2, Lock,
  Power, PowerOff, Settings2,
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
import { Select, SelectItem } from "@/components/ui/select";
import { RevealValue } from "@/components/ui/reveal-value";
import {
  api, type Tenant, type TenantDomain, type TenantResource, type Definition, type Field,
} from "@/lib/api";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tenants/$id",
  component: TenantDetail,
});

function TenantDetail() {
  const { id } = Route.useParams();
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [tab, setTab] = useState<"resources" | "domains">("resources");
  const [domains, setDomains] = useState<TenantDomain[]>([]);
  const [resources, setResources] = useState<TenantResource[]>([]);
  const [definitions, setDefinitions] = useState<Definition[]>([]);

  const [hostname, setHostname] = useState("");
  const [newDomainOpen, setNewDomainOpen] = useState(false);

  const [editOpen, setEditOpen] = useState(false);
  const [edit, setEdit] = useState({ name: "", slug: "", status: "active" });

  const [resOpen, setResOpen] = useState(false);
  const [pick, setPick] = useState("");
  const [pickedFields, setPickedFields] = useState<Field[]>([]);
  const [values, setValues] = useState<Record<string, string>>({});
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    const [t, d, r] = await Promise.all([
      api.getTenant(id).catch(() => null),
      api.listDomains(id).catch(() => []),
      api.listTenantResources(id, true).catch(() => []),
    ]);
    setTenant(t);
    setDomains(d ?? []);
    setResources(r ?? []);
  }, [id]);

  useEffect(() => {
    void load();
    api.listDefinitions().then((d) => setDefinitions(d ?? [])).catch(() => {});
  }, [load]);

  const activeKeys = new Set(resources.filter((r) => r.status === "active").map((r) => r.definitionKey));
  const available = definitions.filter((d) => d.status === "active" && !activeKeys.has(d.key));
  const picked = available.find((d) => d.key === pick) ?? null;
  const requiredFilled =
    picked && pickedFields.filter((f) => f.required).every((f) => (values[f.key] ?? "").trim());

  function openEdit() {
    if (!tenant) return;
    setEdit({ name: tenant.name, slug: tenant.slug, status: tenant.status });
    setErr("");
    setEditOpen(true);
  }
  async function saveEdit() {
    if (!edit.name.trim() || !edit.slug.trim()) return;
    setErr("");
    try {
      await api.updateTenant(id, edit);
      setEditOpen(false);
      await load();
    } catch (e) {
      setErr(String(e));
    }
  }

  async function choose(key: string) {
    setPick(key);
    setValues({});
    const def = available.find((d) => d.key === key);
    if (!def) return setPickedFields([]);
    const detail = await api.getDefinition(def.id).catch(() => null);
    setPickedFields(detail?.fields ?? []);
  }
  async function saveResource() {
    if (!picked || !requiredFilled) return;
    setErr("");
    try {
      await api.createResource(id, { definitionKey: picked.key, values });
      setResOpen(false);
      setPick("");
      setValues({});
      setPickedFields([]);
      await load();
    } catch (e) {
      setErr(String(e));
    }
  }

  async function toggleResource(r: TenantResource) {
    try {
      await api.setResourceStatus(id, r.id, r.status === "active" ? "inactive" : "active");
      await load();
    } catch {
      /* RN-01 conflict on reactivate — ignore silently */
    }
  }
  async function removeResource(r: TenantResource) {
    await api.deleteResource(id, r.id).catch(() => {});
    await load();
  }

  async function saveDomain() {
    if (!hostname.trim()) return;
    setErr("");
    try {
      await api.addDomain(id, hostname.trim());
      setHostname("");
      setNewDomainOpen(false);
      await load();
    } catch (e) {
      setErr(String(e));
    }
  }
  async function removeDomain(d: TenantDomain) {
    await api.removeDomain(id, d.id).catch(() => {});
    await load();
  }

  if (!tenant) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16">
        <p className="text-muted-foreground">Tenant não encontrado.</p>
        <Link to="/tenants"><Button variant="outline">Voltar para Tenants</Button></Link>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link to="/tenants" className="hover:text-foreground">Tenants</Link>
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
              <Badge variant={tenant.status === "active" ? "default" : "secondary"}>{tenant.status}</Badge>
            </div>
          </div>
        </div>
        <Button variant="outline" onClick={openEdit}>
          <Settings2 className="size-4" /> Editar
        </Button>
      </div>

      <div className="flex gap-1 border-b">
        <TabBtn active={tab === "resources"} onClick={() => setTab("resources")} icon={<Database className="size-4" />}>
          Recursos
        </TabBtn>
        <TabBtn active={tab === "domains"} onClick={() => setTab("domains")} icon={<Globe className="size-4" />}>
          Domínios
        </TabBtn>
      </div>

      {tab === "resources" && (
        <div className="space-y-4">
          <div className="flex justify-end">
            <Button onClick={() => { setErr(""); setPick(""); setPickedFields([]); setResOpen(true); }}>
              <Plus className="size-4" /> Adicionar recurso
            </Button>
          </div>
          {resources.length === 0 ? (
            <Card className="p-10 text-center">
              <Database className="mx-auto size-6 text-muted-foreground" />
              <p className="mt-2 font-medium">Nenhum recurso</p>
              <p className="text-sm text-muted-foreground">Este tenant ainda não tem recursos provisionados.</p>
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
                    <Badge variant={r.status === "active" ? "default" : "secondary"}>{r.status}</Badge>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      title={r.status === "active" ? "Desativar" : "Reativar"}
                      onClick={() => toggleResource(r)}
                    >
                      {r.status === "active" ? <Power className="size-4" /> : <PowerOff className="size-4" />}
                    </Button>
                    <Button variant="ghost" size="icon-sm" title="Remover" onClick={() => removeResource(r)}>
                      <Trash2 className="size-4" />
                    </Button>
                  </div>
                </CardHeader>
                <CardContent>
                  <Table>
                    <THead>
                      <TR>
                        <TH>Campo</TH>
                        <TH>Valor</TH>
                        <TH className="w-20">Secret</TH>
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
                            {f.isSecret ? <RevealValue value={f.value} /> : <code className="text-xs">{f.value || "—"}</code>}
                          </TD>
                          <TD>{f.isSecret && <Badge variant="secondary" className="text-amber-600">secret</Badge>}</TD>
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
              <CardTitle className="text-base">Domínios</CardTitle>
              <p className="text-sm text-muted-foreground">Hostnames que resolvem para este tenant.</p>
            </div>
            <Button variant="outline" size="sm" onClick={() => { setHostname(""); setErr(""); setNewDomainOpen(true); }}>
              <Plus className="size-4" /> Adicionar
            </Button>
          </CardHeader>
          <CardContent>
            {domains.length === 0 ? (
              <p className="text-sm text-muted-foreground">Nenhum domínio. Adicione ao menos um para resolver o tenant.</p>
            ) : (
              <Table>
                <THead>
                  <TR>
                    <TH>Hostname</TH>
                    <TH className="w-16 text-right">Ações</TH>
                  </TR>
                </THead>
                <TBody>
                  {domains.map((d) => (
                    <TR key={d.id}>
                      <TD className="font-medium"><code className="text-xs">{d.hostname}</code></TD>
                      <TD className="text-right">
                        <Button variant="ghost" size="icon-sm" title="Remover" onClick={() => removeDomain(d)}>
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

      {/* Editar tenant */}
      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Editar tenant</DialogTitle>
            <DialogDescription>Atualize os dados do cliente.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Nome</label>
              <Input value={edit.name} onChange={(e) => setEdit({ ...edit, name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Slug</label>
              <Input value={edit.slug} onChange={(e) => setEdit({ ...edit, slug: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Status</label>
              <Select value={edit.status} onChange={(e) => setEdit({ ...edit, status: e.target.value })}>
                <SelectItem value="active">active</SelectItem>
                <SelectItem value="inactive">inactive</SelectItem>
              </Select>
            </div>
            {err && <div className="text-sm text-destructive">{err}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">Cancelar</Button>} />
            <Button disabled={!edit.name.trim() || !edit.slug.trim()} onClick={saveEdit}>Salvar</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Adicionar recurso */}
      <Dialog open={resOpen} onOpenChange={setResOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Adicionar recurso</DialogTitle>
            <DialogDescription>Selecione um tipo. Os campos vêm da definition; só 1 ativo por tipo.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            {available.length === 0 ? (
              <div className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">
                Todos os tipos ativos já estão provisionados (ou não há definitions ativas).
              </div>
            ) : (
              <>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">Tipo de recurso</label>
                  <Select value={pick} onChange={(e) => choose(e.target.value)}>
                    <SelectItem value="">Selecione...</SelectItem>
                    {available.map((d) => (
                      <SelectItem key={d.key} value={d.key}>{d.name}</SelectItem>
                    ))}
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
            {err && <div className="text-sm text-destructive">{err}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">Cancelar</Button>} />
            <Button disabled={!picked || !requiredFilled} onClick={saveResource}>Salvar recurso</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Adicionar domínio */}
      <Dialog open={newDomainOpen} onOpenChange={setNewDomainOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Adicionar domínio</DialogTitle>
            <DialogDescription>Hostname exato que resolve para este tenant.</DialogDescription>
          </DialogHeader>
          <div className="space-y-1.5">
            <label className="text-sm font-medium">Hostname</label>
            <Input placeholder="app.cliente.com" value={hostname} onChange={(e) => setHostname(e.target.value)} />
            {err && <div className="text-sm text-destructive">{err}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">Cancelar</Button>} />
            <Button disabled={!hostname.trim()} onClick={saveDomain}>Adicionar</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
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
