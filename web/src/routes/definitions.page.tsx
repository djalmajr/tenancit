import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { Plus, Database, HardDrive, Mail, Box, ChevronRight } from "lucide-react";
import { Card, CardHeader, CardTitle, CardFooter, CardContent, CardDescription, CardAction } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { DomainStatus } from "@/components/domain-status";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { DefinitionActions } from "@/components/definition-actions";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { apiErrorMessage, formatStatus, useI18n } from "@/lib/i18n";
import { api, type Definition } from "@/lib/api";
import { invalidateAllTenantResources, invalidateDefinitions } from "@/lib/query-invalidation";
import { adminQueryOptions } from "@/lib/query-options";
import { useAdminCapabilities } from "@/hooks/use-admin-capabilities";
import { isValidDefinitionKey } from "@/lib/validation";
import { toast } from "sonner";

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
  const { can } = useAdminCapabilities();
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({ key: "", name: "", description: "" });
  const [editingDefinition, setEditingDefinition] = useState<Definition | null>(null);
  const [removingDefinition, setRemovingDefinition] = useState<Definition | null>(null);
  const [editForm, setEditForm] = useState({ name: "", description: "" });
  const keyValid = isValidDefinitionKey(form.key);

  const definitionsQuery = useQuery(adminQueryOptions.definitions());
  const defs: Definition[] = definitionsQuery.data ?? [];
  const createDefinitionMutation = useMutation({ mutationFn: api.createDefinition });
  const updateDefinitionMutation = useMutation({
    mutationFn: ({ id, body }: { id: string; body: { name: string; description: string } }) => api.updateDefinition(id, body),
  });
  const statusMutation = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) => api.setDefinitionStatus(id, status),
  });
  const deleteDefinitionMutation = useMutation({ mutationFn: api.deleteDefinition });
  const visibleError = error || (definitionsQuery.error ? apiErrorMessage(definitionsQuery.error, t) : "");

  async function create() {
    if (!keyValid || !form.name.trim()) return;
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

  function requestEdit(definition: Definition) {
    setError("");
    setEditForm({ name: definition.name, description: definition.description });
    setEditingDefinition(definition);
  }

  async function saveDefinition() {
    if (!editingDefinition || !editForm.name.trim()) return;
    try {
      await updateDefinitionMutation.mutateAsync({ id: editingDefinition.id, body: editForm });
      await invalidateDefinitions(queryClient);
      setEditingDefinition(null);
      setError("");
      toast.success(t("definitionDetail.updated"));
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }

  async function toggleDefinition(definition: Definition) {
    const status = definition.status === "active" ? "inactive" : "active";
    try {
      await statusMutation.mutateAsync({ id: definition.id, status });
      await Promise.all([invalidateDefinitions(queryClient), invalidateAllTenantResources(queryClient)]);
      setError("");
      toast.success(status === "active" ? t("definitionDetail.statusActivated") : t("definitionDetail.statusDeactivated"));
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }

  async function removeDefinition() {
    if (!removingDefinition) return;
    try {
      await deleteDefinitionMutation.mutateAsync(removingDefinition.id);
      await Promise.all([invalidateDefinitions(queryClient), invalidateAllTenantResources(queryClient)]);
      setRemovingDefinition(null);
      setError("");
      toast.success(t("definitionDetail.removed"));
    } catch (e) {
      setRemovingDefinition(null);
      setError(apiErrorMessage(e, t));
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-end">
        {can("resource.write") && <Button onClick={() => { setError(""); setOpen(true); }}><Plus className="size-4" /> {t("definitions.new")}</Button>}
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
          {can("resource.write") && <Button variant="outline" onClick={() => { setError(""); setOpen(true); }}>
            <Plus className="size-4" /> {t("definitions.new")}
          </Button>}
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {defs.map((d) => (
            <Card
              key={d.id}
              aria-label={t("definitionDetail.open", { name: d.name })}
              className="h-full text-left transition-colors hover:cursor-pointer hover:ring-primary/40"
              onClick={() => { void navigate({ to: "/resource-definitions/$id", params: { id: d.id } }); }}
              onKeyDown={(event) => {
                if (event.key === "Enter" || event.key === " ") {
                  event.preventDefault();
                  void navigate({ to: "/resource-definitions/$id", params: { id: d.id } });
                }
              }}
              role="link"
              size="sm"
              tabIndex={0}
            >
              <CardHeader>
                <div className="mb-2 flex size-9 items-center justify-center rounded-lg bg-muted text-muted-foreground">{defIcon(d.key)}</div>
                <CardAction className="flex items-center gap-1">
                  <DomainStatus label={formatStatus(d.status, t)} value={d.status} />
                  {can("resource.write") && <DefinitionActions
                    definition={d}
                    onEdit={() => requestEdit(d)}
                    onRemove={() => setRemovingDefinition(d)}
                    onToggleStatus={() => { void toggleDefinition(d); }}
                  />}
                </CardAction>
                <CardTitle className="line-clamp-2 pr-24 text-base">{d.name}</CardTitle>
                <CardDescription className="line-clamp-2">{d.description || t("definitions.emptyDescription")}</CardDescription>
              </CardHeader>
              <CardContent className="mt-auto"><code className="block truncate rounded-md bg-muted px-2 py-1 text-xs text-muted-foreground">{d.key}</code></CardContent>
              <CardFooter className="justify-between text-xs text-muted-foreground">
                <span>{t("definitions.footerCounts", { fieldCount: d.fieldCount ?? 0, secretCount: d.secretCount ?? 0 })}</span><ChevronRight className="size-4" />
              </CardFooter>
            </Card>
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
              <Input
                aria-describedby="new-definition-key-hint"
                aria-invalid={form.key.length > 0 && !keyValid}
                autoComplete="off"
                id="new-definition-key"
                maxLength={63}
                placeholder="postgres"
                value={form.key}
                onChange={(e) => setForm({ ...form, key: e.target.value })}
              />
              <p className={form.key.length > 0 && !keyValid ? "text-xs text-destructive" : "text-xs text-muted-foreground"} id="new-definition-key-hint">
                {t("validation.definitionKey")}
              </p>
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
            <Button disabled={!keyValid || !form.name.trim() || createDefinitionMutation.isPending} onClick={() => { void create(); }}>{t("definitions.create")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={Boolean(editingDefinition)} onOpenChange={(nextOpen) => { if (!nextOpen) setEditingDefinition(null); }}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("definitionDetail.editTitle")}</DialogTitle>
            <DialogDescription>{t("definitionDetail.editDescription", { key: editingDefinition?.key ?? "" })}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="edit-definition-name">{t("common.name")}</label>
              <Input id="edit-definition-name" maxLength={120} value={editForm.name} onChange={(event) => setEditForm({ ...editForm, name: event.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="edit-definition-description">{t("common.description")}</label>
              <Input id="edit-definition-description" maxLength={500} value={editForm.description} onChange={(event) => setEditForm({ ...editForm, description: event.target.value })} />
            </div>
            {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!editForm.name.trim() || updateDefinitionMutation.isPending} onClick={() => { void saveDefinition(); }}>{t("common.save")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        cancelLabel={t("common.cancel")}
        confirmDisabled={deleteDefinitionMutation.isPending}
        confirmLabel={t("common.removeConfirm")}
        description={t("definitionDetail.removeDescription", { name: removingDefinition?.name ?? "" })}
        onConfirm={() => { void removeDefinition(); }}
        onOpenChange={(nextOpen) => { if (!nextOpen) setRemovingDefinition(null); }}
        open={Boolean(removingDefinition)}
        title={t("definitionDetail.removeTitle")}
      />
    </div>
  );
}
