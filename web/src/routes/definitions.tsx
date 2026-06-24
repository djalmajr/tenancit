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
import { formatStatus, useI18n } from "@/lib/i18n";
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
  const { t } = useI18n();
  const navigate = useNavigate();
  const [defs, setDefs] = useState<Definition[]>([]);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ key: "", name: "", description: "" });

  const load = () => api.listDefinitions().then((d) => {
    setError("");
    setDefs(d ?? []);
  }).catch((e) => setError(String(e)));
  useEffect(() => void load(), []);

  async function create() {
    if (!form.key || !form.name) return;
    try {
      const created = await api.createDefinition(form);
      setError("");
      setForm({ key: "", name: "", description: "" });
      setOpen(false);
      navigate({ to: "/resource-definitions/$id", params: { id: created.id } });
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("definitions.title")}</h1>
          <p className="text-muted-foreground">{t("definitions.description")}</p>
        </div>
        <Button onClick={() => setOpen(true)}><Plus className="size-4" /> {t("definitions.new")}</Button>
      </div>

      {error && <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}

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
                  <Badge variant={d.status === "active" ? "default" : "secondary"}>{formatStatus(d.status, t)}</Badge>
                </div>
                <CardTitle className="mt-2 text-base">{d.name}</CardTitle>
                <p className="text-sm text-muted-foreground">{d.description || t("definitions.emptyDescription")}</p>
                <code className="text-xs text-muted-foreground">{d.key}</code>
              </CardHeader>
              <CardFooter className="text-xs text-muted-foreground">
                {t("definitions.footerCounts", { fieldCount: d.fieldCount ?? 0, secretCount: d.secretCount ?? 0 })}
              </CardFooter>
            </Card>
          </button>
        ))}
      </div>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("definitions.newDialog.title")}</DialogTitle>
            <DialogDescription>{t("definitions.newDialog.description")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.key")}</label>
              <Input placeholder="postgres" value={form.key} onChange={(e) => setForm({ ...form, key: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.name")}</label>
              <Input placeholder="PostgreSQL Connection" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.description")}</label>
              <Input placeholder="Conexão de banco por tenant" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </div>
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!form.key || !form.name} onClick={create}>{t("definitions.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
