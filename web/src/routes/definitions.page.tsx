import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { Plus, Database, HardDrive, Mail, Box, ChevronRight } from "lucide-react";
import { Card, CardHeader, CardTitle, CardFooter, CardContent, CardDescription, CardAction } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DomainStatus } from "@/components/domain-status";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { apiErrorMessage, formatStatus, useI18n } from "@/lib/i18n";
import { api, type Definition } from "@/lib/api";
import { invalidateAllTenantResources, invalidateDefinitions } from "@/lib/query-invalidation";
import { adminQueryOptions } from "@/lib/query-options";

function defIcon(key: string) {
  if (key === "minio" || key.includes("s3") || key.includes("storage")) return <HardDrive className="size-4" />;
  if (key === "smtp" || key.includes("mail")) return <Mail className="size-4" />;
  if (key === "postgres" || key.includes("db") || key.includes("sql")) return <Database className="size-4" />;
  return <Box className="size-4" />;
}

export default function Definitions() {
  const { t } = useI18n();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ key: "", name: "", description: "" });

  const definitionsQuery = useQuery(adminQueryOptions.definitions());
  const defs: Definition[] = definitionsQuery.data ?? [];
  const createDefinitionMutation = useMutation({ mutationFn: api.createDefinition });
  const visibleError = error || (definitionsQuery.error ? apiErrorMessage(definitionsQuery.error, t) : "");

  async function create() {
    if (!form.key || !form.name) return;
    try {
      const created = await createDefinitionMutation.mutateAsync(form);
      await Promise.all([
        invalidateDefinitions(queryClient),
        invalidateAllTenantResources(queryClient),
      ]);
      setError("");
      setForm({ key: "", name: "", description: "" });
      setOpen(false);
      await navigate({ to: "/resource-definitions/$id", params: { id: created.id } });
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("definitions.title")}</h1>
          <p className="text-muted-foreground">{t("definitions.description")}</p>
        </div>
        <Button onClick={() => { setError(""); setOpen(true); }}><Plus className="size-4" /> {t("definitions.new")}</Button>
      </div>

      {visibleError && <Alert variant="destructive"><AlertDescription>{visibleError}</AlertDescription></Alert>}

      {definitionsQuery.isPending ? (
        <div className="rounded-md border p-4 text-sm text-muted-foreground" role="status">{t("common.loading")}</div>
      ) : visibleError && defs.length === 0 ? null : defs.length === 0 ? (
        <div className="flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed py-16 text-center">
          <div className="flex size-10 items-center justify-center rounded-md bg-muted">
            <Box className="size-5 text-muted-foreground" />
          </div>
          <p className="text-sm text-muted-foreground">{t("definitions.empty")}</p>
          <Button variant="outline" onClick={() => { setError(""); setOpen(true); }}>
            <Plus className="size-4" /> {t("definitions.new")}
          </Button>
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {defs.map((d) => (
            <button
              key={d.id}
              className="h-full text-left"
              onClick={() => { void navigate({ to: "/resource-definitions/$id", params: { id: d.id } }); }}
            >
              <Card className="h-full transition-colors hover:ring-primary/40" size="sm">
                <CardHeader>
                  <div className="mb-2 flex size-9 items-center justify-center rounded-lg bg-muted text-muted-foreground">{defIcon(d.key)}</div>
                  <CardAction><DomainStatus label={formatStatus(d.status, t)} value={d.status} /></CardAction>
                  <CardTitle className="line-clamp-2 pr-16 text-base">{d.name}</CardTitle>
                  <CardDescription className="line-clamp-2">{d.description || t("definitions.emptyDescription")}</CardDescription>
                </CardHeader>
                <CardContent className="mt-auto"><code className="block truncate rounded-md bg-muted px-2 py-1 text-xs text-muted-foreground">{d.key}</code></CardContent>
                <CardFooter className="justify-between text-xs text-muted-foreground">
                  <span>{t("definitions.footerCounts", { fieldCount: d.fieldCount ?? 0, secretCount: d.secretCount ?? 0 })}</span><ChevronRight className="size-4" />
                </CardFooter>
              </Card>
            </button>
          ))}
        </div>
      )}

      <Dialog open={open} onOpenChange={(o) => { if (!o) setError(""); setOpen(o); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("definitions.newDialog.title")}</DialogTitle>
            <DialogDescription>{t("definitions.newDialog.description")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="new-definition-key">{t("common.key")}</label>
              <Input id="new-definition-key" placeholder="postgres" value={form.key} onChange={(e) => setForm({ ...form, key: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="new-definition-name">{t("common.name")}</label>
              <Input id="new-definition-name" placeholder="PostgreSQL Connection" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="new-definition-description">{t("common.description")}</label>
              <Input id="new-definition-description" placeholder="Conexão de banco por tenant" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
            </div>
          </div>
          {visibleError && <Alert className="mt-4" variant="destructive"><AlertDescription>{visibleError}</AlertDescription></Alert>}
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!form.key || !form.name || createDefinitionMutation.isPending} onClick={() => { void create(); }}>{t("definitions.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
