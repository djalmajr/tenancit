import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { getRouteApi, Link, useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import { Boxes, ChevronRight, Plus, Check, Pencil, Trash2 } from "lucide-react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { DomainStatus } from "@/components/domain-status";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { DefinitionActions } from "@/components/definition-actions";
import {
  DefinitionFieldEditDialog,
  type DefinitionFieldEditForm,
} from "./definition-field-edit-dialog";
import { toast } from "sonner";
import { apiErrorMessage, formatStatus, useI18n } from "@/lib/i18n";
import { api, type DefinitionDetail } from "@/lib/api";
import {
  invalidateAllTenantResources,
  invalidateDefinition,
} from "@/lib/query-invalidation";
import { adminQueryOptions } from "@/lib/query-options";
import { useAdminCapabilities } from "@/hooks/use-admin-capabilities";
import { isValidDefinitionKey } from "@/lib/validation";

const routeApi = getRouteApi("/resource-definitions/$id");

const EMPTY = { key: "", label: "", dataType: "string", required: false, isSecret: false };

export default function DefinitionDetailPage() {
  const { can } = useAdminCapabilities();
  const { t } = useI18n();
  const { id } = routeApi.useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [error, setError] = useState("");
  const [fieldError, setFieldError] = useState("");
  const [open, setOpen] = useState(false);
  const [f, setF] = useState({ ...EMPTY });
  const fieldKeyValid = isValidDefinitionKey(f.key);
  const [editingField, setEditingField] = useState<DefinitionDetail["fields"][number] | null>(null);
  const [fieldEditForm, setFieldEditForm] = useState<DefinitionFieldEditForm>({ label: "", required: false });
  const [fieldEditError, setFieldEditError] = useState("");
  const [pendingField, setPendingField] = useState<DefinitionDetail["fields"][number] | null>(null);
  const [editOpen, setEditOpen] = useState(false);
  const [removeOpen, setRemoveOpen] = useState(false);
  const [editForm, setEditForm] = useState({ name: "", description: "" });
  const detailQuery = useQuery(adminQueryOptions.definition(id));
  const detail = detailQuery.data;
  const addFieldMutation = useMutation({
    mutationFn: (field: typeof EMPTY) => api.addField(id, field),
    onSuccess: () => Promise.all([
      invalidateDefinition(queryClient, id),
      invalidateAllTenantResources(queryClient),
    ]),
  });
  const deleteFieldMutation = useMutation({
    mutationFn: (fieldId: string) => api.deleteField(id, fieldId),
    onSuccess: () => Promise.all([
      invalidateDefinition(queryClient, id),
      invalidateAllTenantResources(queryClient),
    ]),
  });
  const updateFieldMutation = useMutation({
    mutationFn: ({ fieldId, form }: { fieldId: string; form: DefinitionFieldEditForm }) =>
      api.updateDefinitionField(id, fieldId, form),
    onSuccess: () => Promise.all([
      invalidateDefinition(queryClient, id),
      invalidateAllTenantResources(queryClient),
    ]),
  });
  const statusMutation = useMutation({
    mutationFn: (status: string) => api.setDefinitionStatus(id, status),
    onSuccess: () => Promise.all([
      invalidateDefinition(queryClient, id),
      invalidateAllTenantResources(queryClient),
    ]),
  });
  const updateDefinitionMutation = useMutation({
    mutationFn: (body: { name: string; description: string }) => api.updateDefinition(id, body),
    onSuccess: () => Promise.all([
      invalidateDefinition(queryClient, id),
      invalidateAllTenantResources(queryClient),
    ]),
  });
  const deleteDefinitionMutation = useMutation({ mutationFn: () => api.deleteDefinition(id) });
  const visibleError = error || (detailQuery.error ? apiErrorMessage(detailQuery.error, t) : "");

  async function addField() {
    if (!fieldKeyValid) return;
    if (!f.key.trim()) return;
    try {
      await addFieldMutation.mutateAsync(f);
      setF({ ...EMPTY });
      setFieldError("");
      setOpen(false);
      toast.success(t("definitionDetail.fieldAdded"));
    } catch (e) {
      setFieldError(apiErrorMessage(e, t));
    }
  }
  async function removeField() {
    if (!pendingField) return;
    try {
      await deleteFieldMutation.mutateAsync(pendingField.id);
      setError("");
      setPendingField(null);
      toast.success(t("definitionDetail.fieldRemoved"));
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }
  function openFieldEditor(field: DefinitionDetail["fields"][number]) {
    setFieldEditError("");
    setFieldEditForm({ label: field.label, required: field.required });
    setEditingField(field);
  }
  async function saveField() {
    if (!editingField) return;
    try {
      await updateFieldMutation.mutateAsync({ fieldId: editingField.id, form: fieldEditForm });
      setEditingField(null);
      setFieldEditError("");
      toast.success(t("definitionDetail.fieldUpdated"));
    } catch (e) {
      setFieldEditError(apiErrorMessage(e, t));
    }
  }
  async function toggleStatus() {
    if (!detail) return;
    const next = detail.definition.status === "active" ? "inactive" : "active";
    try {
      await statusMutation.mutateAsync(next);
      setError("");
      toast.success(next === "active" ? t("definitionDetail.statusActivated") : t("definitionDetail.statusDeactivated"));
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }

  function openEditor() {
    if (!detail) return;
    setError("");
    setEditForm({ name: detail.definition.name, description: detail.definition.description });
    setEditOpen(true);
  }

  async function saveDefinition() {
    if (!editForm.name.trim()) return;
    try {
      await updateDefinitionMutation.mutateAsync(editForm);
      setEditOpen(false);
      setError("");
      toast.success(t("definitionDetail.updated"));
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }

  async function removeDefinition() {
    try {
      await deleteDefinitionMutation.mutateAsync();
      await invalidateAllTenantResources(queryClient);
      setRemoveOpen(false);
      toast.success(t("definitionDetail.removed"));
      await navigate({ to: "/resource-definitions" });
    } catch (e) {
      setRemoveOpen(false);
      setError(apiErrorMessage(e, t));
    }
  }

  if (!detail) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16">
        {visibleError && <Alert variant="destructive"><AlertDescription>{visibleError}</AlertDescription></Alert>}
        <p className="text-muted-foreground">
          {detailQuery.isPending
            ? t("common.loading")
            : visibleError
              ? t("definitionDetail.loadError")
              : t("definitionDetail.notFound")}
        </p>
        <Link to="/resource-definitions"><Button variant="outline">{t("definitionDetail.back")}</Button></Link>
      </div>
    );
  }

  const d = detail.definition;
  return (
    <div className="space-y-6">
      {visibleError && <Alert variant="destructive"><AlertDescription>{visibleError}</AlertDescription></Alert>}

      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link to="/resource-definitions" className="hover:text-foreground">{t("definitions.title")}</Link>
        <ChevronRight className="size-3.5" />
        <span className="text-foreground">{d.name}</span>
      </div>

      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex items-center gap-3">
          <div className="flex size-11 items-center justify-center rounded-lg bg-muted">
            <Boxes className="size-5" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{d.name}</h1>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <code>{d.key}</code>
              <DomainStatus label={formatStatus(d.status, t)} value={d.status} />
            </div>
          </div>
        </div>
        {can("resource.write") && <div className="flex items-end gap-2">
          <Button onClick={() => { setF({ ...EMPTY }); setFieldError(""); setOpen(true); }}>
            <Plus className="size-4" /> {t("definitionDetail.newField")}
          </Button>
          <DefinitionActions
            definition={d}
            onEdit={openEditor}
            onRemove={() => setRemoveOpen(true)}
            onToggleStatus={() => { void toggleStatus(); }}
          />
        </div>}
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("definitionDetail.fieldsTitle")}</CardTitle>
          <p className="text-sm text-muted-foreground">{t("definitionDetail.fieldsDescription")}</p>
        </CardHeader>
        <CardContent>
          {detail.fields.length === 0 ? (
            <p className="text-sm text-muted-foreground">{t("definitionDetail.emptyDescription")}</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("common.key")}</TableHead>
                  <TableHead>{t("common.label")}</TableHead>
                  <TableHead>{t("common.type")}</TableHead>
                  <TableHead>{t("definitionDetail.requiredTitle")}</TableHead>
                  <TableHead>{t("common.secret")}</TableHead>
                  <TableHead className="w-16 text-right">{t("common.actions")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {detail.fields.map((field) => (
                  <TableRow key={field.id}>
                    <TableCell><code className="text-xs">{field.key}</code></TableCell>
                    <TableCell>{field.label || "—"}</TableCell>
                    <TableCell><code className="text-xs text-muted-foreground">{field.data_type}</code></TableCell>
                    <TableCell>{field.required ? <Check className="size-4 text-green-600" /> : <span className="text-muted-foreground">—</span>}</TableCell>
                    <TableCell>{field.is_secret ? <Check className="size-4 text-amber-600" /> : <span className="text-muted-foreground">—</span>}</TableCell>
                    <TableCell className="text-right">
                      {can("resource.write") && <div className="flex justify-end gap-1">
                        <Button
                          aria-label={t("definitionDetail.editFieldAction", { key: field.key })}
                          onClick={() => openFieldEditor(field)}
                          size="icon-sm"
                          title={t("common.edit")}
                          variant="ghost"
                        >
                          <Pencil className="size-4" />
                        </Button>
                        <Button variant="ghost" size="icon-sm" title={t("common.remove")} onClick={() => setPendingField(field)}>
                          <Trash2 className="size-4" />
                        </Button>
                      </div>}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("definitionDetail.newField")}</DialogTitle>
            <DialogDescription>{t("definitionDetail.newFieldDescription")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-sm font-medium" htmlFor="definition-field-key">{t("common.key")}</label>
                <Input
                  aria-describedby="definition-field-key-hint"
                  aria-invalid={f.key.length > 0 && !fieldKeyValid}
                  autoComplete="off"
                  id="definition-field-key"
                  maxLength={63}
                  placeholder="host"
                  value={f.key}
                  onChange={(e) => setF({ ...f, key: e.target.value })}
                />
                <p className={f.key.length > 0 && !fieldKeyValid ? "text-xs text-destructive" : "text-xs text-muted-foreground"} id="definition-field-key-hint">
                  {t("validation.definitionKey")}
                </p>
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium" htmlFor="definition-field-label">{t("common.label")}</label>
                <Input id="definition-field-label" placeholder="Host" value={f.label} onChange={(e) => setF({ ...f, label: e.target.value })} />
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.type")}</label>
              <Select
                items={{
                  bool: t("definitionDetail.dataTypeBool"),
                  int: t("definitionDetail.dataTypeInt"),
                  string: t("definitionDetail.dataTypeString"),
                }}
                value={f.dataType}
                onValueChange={(value) => setF({ ...f, dataType: String(value) })}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="string">{t("definitionDetail.dataTypeString")}</SelectItem>
                    <SelectItem value="int">{t("definitionDetail.dataTypeInt")}</SelectItem>
                    <SelectItem value="bool">{t("definitionDetail.dataTypeBool")}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center justify-between rounded-md border p-3">
              <div>
                <label className="text-sm font-medium" htmlFor="definition-field-required">{t("definitionDetail.requiredTitle")}</label>
                <div className="text-xs text-muted-foreground">{t("definitionDetail.requiredDescription")}</div>
              </div>
              <Checkbox id="definition-field-required" checked={f.required} onCheckedChange={(checked) => setF({ ...f, required: checked })} />
            </div>
            <div className="flex items-center justify-between rounded-md border p-3">
              <div>
                <label className="text-sm font-medium" htmlFor="definition-field-secret">{t("common.secret")}</label>
                <div className="text-xs text-muted-foreground">{t("definitionDetail.secretDescription")}</div>
              </div>
              <Checkbox id="definition-field-secret" checked={f.isSecret} onCheckedChange={(checked) => setF({ ...f, isSecret: checked })} />
            </div>
            {fieldError && <div className="text-sm text-destructive">{fieldError}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!fieldKeyValid || addFieldMutation.isPending} onClick={() => { void addField(); }}>{t("definitionDetail.addField")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <DefinitionFieldEditDialog
        error={fieldEditError}
        field={editingField}
        form={fieldEditForm}
        isSaving={updateFieldMutation.isPending}
        onFormChange={setFieldEditForm}
        onOpenChange={(nextOpen) => {
          if (!nextOpen) {
            setEditingField(null);
            setFieldEditError("");
          }
        }}
        onSave={() => { void saveField(); }}
      />

      <Dialog open={editOpen} onOpenChange={setEditOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("definitionDetail.editTitle")}</DialogTitle>
            <DialogDescription>{t("definitionDetail.editDescription", { key: d.key })}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="detail-definition-name">{t("common.name")}</label>
              <Input id="detail-definition-name" maxLength={120} value={editForm.name} onChange={(event) => setEditForm({ ...editForm, name: event.target.value })} />
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium" htmlFor="detail-definition-description">{t("common.description")}</label>
              <Input id="detail-definition-description" maxLength={500} value={editForm.description} onChange={(event) => setEditForm({ ...editForm, description: event.target.value })} />
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
        confirmLabel={t("common.removeConfirm")}
        description={t("definitionDetail.removeFieldDescription", { fieldKey: pendingField?.key ?? "" })}
        onConfirm={() => { void removeField(); }}
        onOpenChange={(open) => {
          if (!open) setPendingField(null);
        }}
        open={Boolean(pendingField)}
        title={t("definitionDetail.removeFieldTitle")}
      />
      <ConfirmDialog
        cancelLabel={t("common.cancel")}
        confirmDisabled={deleteDefinitionMutation.isPending}
        confirmLabel={t("common.removeConfirm")}
        description={t("definitionDetail.removeDescription", { name: d.name })}
        onConfirm={() => { void removeDefinition(); }}
        onOpenChange={setRemoveOpen}
        open={removeOpen}
        title={t("definitionDetail.removeTitle")}
      />
    </div>
  );
}
