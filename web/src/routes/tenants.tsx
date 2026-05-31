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
import { api, type Tenant } from "@/lib/api";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tenants",
  component: Tenants,
});

function Tenants() {
  const navigate = useNavigate();
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [q, setQ] = useState("");
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ slug: "", name: "" });
  const [error, setError] = useState("");

  const load = () => api.listTenants().then((t) => setTenants(t ?? [])).catch(() => {});
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
          <h1 className="text-2xl font-semibold tracking-tight">Tenants</h1>
          <p className="text-muted-foreground">Clientes e seus domínios.</p>
        </div>
        <Button onClick={() => { setError(""); setOpen(true); }}><Plus className="size-4" /> Novo tenant</Button>
      </div>

      <div className="relative max-w-sm">
        <Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
        <Input className="pl-8" placeholder="Buscar por nome ou slug..." value={q} onChange={(e) => setQ(e.target.value)} />
      </div>

      <Card>
        <Table>
          <THead>
            <TR>
              <TH>Nome</TH>
              <TH>Slug</TH>
              <TH>Status</TH>
              <TH className="w-10"></TH>
            </TR>
          </THead>
          <TBody>
            {filtered.map((t) => (
              <TR
                key={t.id}
                className="cursor-pointer"
                onClick={() => navigate({ to: "/tenants/$id", params: { id: t.id } })}
              >
                <TD className="font-medium">{t.name}</TD>
                <TD><code className="text-xs">{t.slug}</code></TD>
                <TD><Badge variant={t.status === "active" ? "default" : "secondary"}>{t.status}</Badge></TD>
                <TD><ChevronRight className="size-4 text-muted-foreground" /></TD>
              </TR>
            ))}
          </TBody>
        </Table>
      </Card>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Novo tenant</DialogTitle>
            <DialogDescription>Cadastre um cliente. Domínios e recursos são adicionados em seguida.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Nome</label>
              <Input placeholder="Acme Corp" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Slug</label>
              <Input placeholder="acme" value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value })} />
            </div>
            {error && <div className="text-sm text-destructive">{error}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">Cancelar</Button>} />
            <Button disabled={!form.slug || !form.name} onClick={create}>Criar tenant</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
