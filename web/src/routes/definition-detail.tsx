import { createRoute, Link } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import { Boxes, ChevronRight, Plus, Check, Trash2, Power, PowerOff } from "lucide-react";
import { Route as rootRoute } from "./__root";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Table, THead, TBody, TR, TH, TD } from "@/components/ui/table";
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
import { StatusNotice } from "@/components/ui/status-notice";
import { apiErrorMessage, formatStatus, useI18n } from "@/lib/i18n";
import { api, type DefinitionDetail } from "@/lib/api";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/resource-definitions/$id",
  component: DefinitionDetailPage,
});

const EMPTY = { key: "", label: "", dataType: "string", required: false, isSecret: false };

function DefinitionDetailPage() {
  const { t } = useI18n();
  const { id } = Route.useParams();
  const [detail, setDetail] = useState<DefinitionDetail | null>(null);
  const [error, setError] = useState("");
  const [fieldError, setFieldError] = useState("");
  const [open, setOpen] = useState(false);
  const [f, setF] = useState({ ...EMPTY });
  const [notice, setNotice] = useState("");
  const [pendingField, setPendingField] = useState<DefinitionDetail["fields"][number] | null>(null);

  const load = useCallback(() => {
    api.getDefinition(id).then((d) => {
      setDetail(d);
      setError("");
    }).catch((e) => {
      setDetail(null);
      setError(apiErrorMessage(e, t));
    });
  }, [id]);
  useEffect(() => load(), [load]);

  async function addField() {
    if (!f.key.trim()) return;
    try {
      await api.addField(id, f);
      setF({ ...EMPTY });
      setFieldError("");
      setOpen(false);
      setNotice(t("definitionDetail.fieldAdded"));
      load();
    } catch (e) {
      setFieldError(apiErrorMessage(e, t));
    }
  }
  async function removeField() {
    if (!pendingField) return;
    try {
      await api.deleteField(id, pendingField.id);
      setPendingField(null);
      setNotice(t("definitionDetail.fieldRemoved"));
      load();
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }
  async function toggleStatus() {
    if (!detail) return;
    const next = detail.definition.status === "active" ? "inactive" : "active";
    try {
      await api.setDefinitionStatus(id, next);
      setNotice(next === "active" ? t("definitionDetail.statusActivated") : t("definitionDetail.statusDeactivated"));
      load();
    } catch (e) {
      setError(apiErrorMessage(e, t));
    }
  }

  if (!detail) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16">
        {error && <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}
        <p className="text-muted-foreground">{error ? t("definitionDetail.loadError") : t("definitionDetail.notFound")}</p>
        <Link to="/resource-definitions"><Button variant="outline">{t("definitionDetail.back")}</Button></Link>
      </div>
    );
  }

  const d = detail.definition;
  return (
    <div className="space-y-6">
      {error && <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}
      <StatusNotice dismissLabel={t("common.dismiss")} message={notice} onDismiss={() => setNotice("")} />

      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link to="/resource-definitions" className="hover:text-foreground">{t("definitions.title")}</Link>
        <ChevronRight className="size-3.5" />
        <span className="text-foreground">{d.name}</span>
      </div>

      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="flex size-11 items-center justify-center rounded-lg bg-muted">
            <Boxes className="size-5" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{d.name}</h1>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <code>{d.key}</code>
              <Badge variant={d.status === "active" ? "default" : "secondary"}>{formatStatus(d.status, t)}</Badge>
            </div>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={toggleStatus}>
            {d.status === "active" ? <PowerOff className="size-4" /> : <Power className="size-4" />}
            {d.status === "active" ? t("definitionDetail.deactivate") : t("definitionDetail.activate")}
          </Button>
          <Button onClick={() => { setF({ ...EMPTY }); setFieldError(""); setOpen(true); }}>
            <Plus className="size-4" /> {t("definitionDetail.newField")}
          </Button>
        </div>
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
              <THead>
                <TR>
                  <TH>{t("common.key")}</TH>
                  <TH>{t("common.label")}</TH>
                  <TH>{t("common.type")}</TH>
                  <TH>{t("definitionDetail.requiredTitle")}</TH>
                  <TH>{t("common.secret")}</TH>
                  <TH className="w-16 text-right">{t("common.actions")}</TH>
                </TR>
              </THead>
              <TBody>
                {detail.fields.map((field) => (
                  <TR key={field.id}>
                    <TD><code className="text-xs">{field.key}</code></TD>
                    <TD>{field.label || "—"}</TD>
                    <TD><code className="text-xs text-muted-foreground">{field.data_type}</code></TD>
                    <TD>{field.required ? <Check className="size-4 text-green-600" /> : <span className="text-muted-foreground">—</span>}</TD>
                    <TD>{field.is_secret ? <Check className="size-4 text-amber-600" /> : <span className="text-muted-foreground">—</span>}</TD>
                    <TD className="text-right">
                      <Button variant="ghost" size="icon-sm" title={t("common.remove")} onClick={() => setPendingField(field)}>
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

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("definitionDetail.newField")}</DialogTitle>
            <DialogDescription>{t("definitionDetail.newFieldDescription")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-sm font-medium">{t("common.key")}</label>
                <Input placeholder="host" value={f.key} onChange={(e) => setF({ ...f, key: e.target.value })} />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">{t("common.label")}</label>
                <Input placeholder="Host" value={f.label} onChange={(e) => setF({ ...f, label: e.target.value })} />
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">{t("common.type")}</label>
              <Select value={f.dataType} onValueChange={(value) => setF({ ...f, dataType: String(value) })}>
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value="string">string</SelectItem>
                    <SelectItem value="int">int</SelectItem>
                    <SelectItem value="bool">bool</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
            <label className="flex items-center justify-between rounded-md border p-3">
              <div>
                <div className="text-sm font-medium">{t("definitionDetail.requiredTitle")}</div>
                <div className="text-xs text-muted-foreground">{t("definitionDetail.requiredDescription")}</div>
              </div>
              <input type="checkbox" checked={f.required} onChange={(e) => setF({ ...f, required: e.target.checked })} />
            </label>
            <label className="flex items-center justify-between rounded-md border p-3">
              <div>
                <div className="text-sm font-medium">{t("common.secret")}</div>
                <div className="text-xs text-muted-foreground">{t("definitionDetail.secretDescription")}</div>
              </div>
              <input type="checkbox" checked={f.isSecret} onChange={(e) => setF({ ...f, isSecret: e.target.checked })} />
            </label>
            {fieldError && <div className="text-sm text-destructive">{fieldError}</div>}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
            <Button disabled={!f.key.trim()} onClick={addField}>{t("definitionDetail.addField")}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        cancelLabel={t("common.cancel")}
        confirmLabel={t("common.removeConfirm")}
        description={t("definitionDetail.removeFieldDescription", { fieldKey: pendingField?.key ?? "" })}
        onConfirm={removeField}
        onOpenChange={(open) => {
          if (!open) setPendingField(null);
        }}
        open={Boolean(pendingField)}
        title={t("definitionDetail.removeFieldTitle")}
      />
    </div>
  );
}
