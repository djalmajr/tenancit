import { createRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { Plus, Search, ChevronRight } from "lucide-react";
import { Route as rootRoute } from "./__root";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Table, THead, TBody, TR, TH, TD } from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { formatStatus, useI18n } from "@/lib/i18n";
import { api, type Tenant } from "@/lib/api";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tenants",
  component: Tenants,
});

function Tenants() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ slug: "", name: "" });
  const [error, setError] = useState("");
  const [pageError, setPageError] = useState("");

  const load = () => api.listTenants().then((t) => {
    setPageError("");
    setTenants(t ?? []);
  }).catch((e) => setPageError(String(e)));
  useEffect(() => void load(), []);

  const filtered = tenants.filter(
    (t) => t.name.toLowerCase().includes(q.toLowerCase()) || t.slug.includes(q.toLowerCase()),
  );

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

      <div className="relative max-w-sm">
        <Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
        <Input className="pl-8" placeholder={t("tenants.search")} value={q} onChange={(e) => setQ(e.target.value)} />
      </div>

      <Card>
        <Table>
          <THead>
            <TR>
              <TH>{t("common.name")}</TH>
              <TH>{t("common.slug")}</TH>
              <TH>{t("common.status")}</TH>
              <TH className="w-10"></TH>
            </TR>
          </THead>
          <TBody>
            {filtered.map((tenant) => (
              <TR
                key={tenant.id}
                className="cursor-pointer"
                onClick={() => navigate({ to: "/tenants/$id", params: { id: tenant.id } })}
              >
                <TD className="font-medium">{tenant.name}</TD>
                <TD><code className="text-xs">{tenant.slug}</code></TD>
                <TD><Badge variant={tenant.status === "active" ? "default" : "secondary"}>{formatStatus(tenant.status, t)}</Badge></TD>
                <TD><ChevronRight className="size-4 text-muted-foreground" /></TD>
              </TR>
            ))}
          </TBody>
        </Table>
      </Card>

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
