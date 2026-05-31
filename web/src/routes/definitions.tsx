import { createRoute, useNavigate } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { Plus, Database, HardDrive, Mail, Box } from "lucide-react";
import { Route as rootRoute } from "./__root";
import { Card, CardHeader, CardTitle, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { api, type Definition } from "@/lib/api";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/resource-definitions",
  component: Definitions,
});

function defIcon(key: string) {
  if (key === "minio" || key.includes("s3") || key.includes("storage")) return <HardDrive className="size-4" />;
  if (key === "smtp" || key.includes("mail")) return <Mail className="size-4" />;
  if (key === "postgres" || key.includes("db") || key.includes("sql")) return <Database className="size-4" />;
  return <Box className="size-4" />;
}

function Definitions() {
  const navigate = useNavigate();
  const [defs, setDefs] = useState<Definition[]>([]);
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ key: "", name: "", description: "" });

  const load = () => api.listDefinitions().then((d) => setDefs(d ?? [])).catch(() => {});
  useEffect(() => void load(), []);

  async function create() {
    if (!form.key || !form.name) return;
    const created = await api.createDefinition(form).catch(() => null);
    setForm({ key: "", name: "", description: "" });
    setOpen(false);
    if (created) navigate({ to: "/resource-definitions/$id", params: { id: created.id } });
    else load();
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Resource Definitions</h1>
          <p className="text-muted-foreground">Catálogo de tipos de recurso e seus campos.</p>
        </div>
        <Button onClick={() => setOpen(true)}><Plus className="size-4" /> Nova definition</Button>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {defs.map((d) => (
          <button
            key={d.id}
            className="text-left"
            onClick={() => navigate({ to: "/resource-definitions/$id", params: { id: d.id } })}
          >
            <Card className="h-full transition-colors hover:border-primary/50">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div className="flex size-9 items-center justify-center rounded-md bg-muted">
                    {defIcon(d.key)}
                  </div>
                  <Badge variant={d.status === "active" ? "default" : "secondary"}>{d.status}</Badge>
                </div>
                <CardTitle className="mt-2 text-base">{d.name}</CardTitle>
                <p className="text-sm text-muted-foreground">{d.description || "Sem descrição."}</p>
                <code className="text-xs text-muted-foreground">{d.key}</code>
              </CardHeader>
              <CardFooter className="text-xs text-muted-foreground">
                {d.fieldCount ?? 0} campos · {d.secretCount ?? 0} secret
              </CardFooter>
            </Card>
          </button>
        ))}
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Nova definition</DialogTitle>
            <DialogDescription>Defina um tipo de recurso. Os campos são adicionados em seguida.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Key</label>
              <Input placeholder="postgres" value={form.key} onChange={(e) => setForm({ ...form, key: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Nome</label>
              <Input placeholder="PostgreSQL Connection" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Descrição</label>
              <Input placeholder="Conexão de banco por tenant" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </div>
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">Cancelar</Button>} />
            <Button disabled={!form.key || !form.name} onClick={create}>Criar definition</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
