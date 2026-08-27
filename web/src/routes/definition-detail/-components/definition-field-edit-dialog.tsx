import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type { Field } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

export interface DefinitionFieldEditForm {
  label: string;
  required: boolean;
}

interface DefinitionFieldEditDialogProps {
  error: string;
  field: Field | null;
  form: DefinitionFieldEditForm;
  isSaving: boolean;
  onFormChange: (form: DefinitionFieldEditForm) => void;
  onOpenChange: (open: boolean) => void;
  onSave: () => void;
}

export function DefinitionFieldEditDialog({
  error,
  field,
  form,
  isSaving,
  onFormChange,
  onOpenChange,
  onSave,
}: DefinitionFieldEditDialogProps) {
  const { t } = useI18n();
  if (!field) return null;

  const dataTypeLabel = t(field.data_type === "bool"
    ? "definitionDetail.dataTypeBool"
    : field.data_type === "int"
      ? "definitionDetail.dataTypeInt"
      : "definitionDetail.dataTypeString");

  return <Dialog open onOpenChange={onOpenChange}>
    <DialogContent className="sm:max-w-lg">
      <DialogHeader>
        <DialogTitle>{t("definitionDetail.editFieldTitle")}</DialogTitle>
        <DialogDescription>{t("definitionDetail.editFieldDescription", { key: field.key })}</DialogDescription>
      </DialogHeader>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="min-w-0 space-y-1.5">
          <label className="text-sm font-medium" htmlFor="definition-field-edit-key">{t("common.key")}</label>
          <Input disabled id="definition-field-edit-key" value={field.key} />
        </div>
        <div className="min-w-0 space-y-1.5">
          <label className="text-sm font-medium" htmlFor="definition-field-edit-label">{t("common.label")}</label>
          <Input
            id="definition-field-edit-label"
            maxLength={120}
            value={form.label}
            onChange={(event) => onFormChange({ ...form, label: event.target.value })}
          />
        </div>
        <div className="min-w-0 space-y-1.5">
          <label className="text-sm font-medium" htmlFor="definition-field-edit-type">{t("common.type")}</label>
          <Input disabled id="definition-field-edit-type" value={dataTypeLabel} />
        </div>
        <div className="flex min-w-0 items-center gap-2">
          <Checkbox
            checked={form.required}
            id="definition-field-edit-required"
            onCheckedChange={(checked) => onFormChange({ ...form, required: Boolean(checked) })}
          />
          <label className="text-sm font-medium" htmlFor="definition-field-edit-required">{t("definitionDetail.requiredTitle")}</label>
        </div>
        <div className="flex min-w-0 items-center gap-2">
          <Checkbox checked={field.is_secret} disabled id="definition-field-edit-secret" />
          <label className="text-sm font-medium" htmlFor="definition-field-edit-secret">{t("common.secret")}</label>
        </div>
      </div>
      {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
      <DialogFooter>
        <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
        <Button disabled={isSaving} onClick={onSave}>{t("common.save")}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>;
}
