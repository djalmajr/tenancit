import { useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Combobox } from "@/components/ui/combobox";
import { Input } from "@/components/ui/input";
import { api } from "@/lib/api";
import { apiErrorMessage, useI18n, type TranslationKey } from "@/lib/i18n";
import { adminQueryKeys } from "@/lib/query-keys";

const SECTION_KEYS = {
  security: ["session_absolute_hours", "session_idle_minutes"],
  apiClients: ["api_client_default_rpm", "api_client_default_ttl_days"],
  retention: ["usage_retention_months", "audit_retention_days", "webhook_delivery_retention_days", "outbox_event_retention_days"],
  console: ["default_locale"],
} as const;

export default function SettingsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const settingsQuery = useQuery({ queryKey: adminQueryKeys.settings(), queryFn: ({ signal }) => api.getSettings(signal) });
  const [draft, setDraft] = useState<Record<string, string>>();
  const [saving, setSaving] = useState(false);
  const values = draft ?? settingsQuery.data?.values ?? {};
  const definitions = useMemo(() => new Map(settingsQuery.data?.definitions.map((definition) => [definition.key, definition]) ?? []), [settingsQuery.data]);
  const error = settingsQuery.error ? apiErrorMessage(settingsQuery.error, t) : "";

  async function save() {
    if (!settingsQuery.data) return;
    setSaving(true);
    try {
      const updated = await api.updateSettings(values, settingsQuery.data.revision);
      queryClient.setQueryData(adminQueryKeys.settings(), updated);
      setDraft(undefined);
      toast.success(t("settings.saved"));
    } catch (cause) {
      toast.error(apiErrorMessage(cause, t));
      setDraft(undefined);
      await queryClient.invalidateQueries({ queryKey: adminQueryKeys.settings() });
    } finally {
      setSaving(false);
    }
  }

  function field(key: string) {
    const definition = definitions.get(key);
    if (!definition) return null;
    const label = t(`settings.field.${key}` as TranslationKey);
    return <div className="flex flex-col gap-1.5" key={key}>
      <label className="text-sm font-medium" htmlFor={`setting-${key}`}>{label}</label>
      {definition.type === "enum" ? <Combobox aria-label={label} options={(definition.options ?? []).map((option) => ({ label: option, value: option }))} searchable={false} triggerClassName="w-full" value={values[key]} onValueChange={(value) => setDraft({ ...values, [key]: value })} /> : <Input id={`setting-${key}`} inputMode="numeric" max={definition.maximum} min={definition.minimum} onChange={(event) => setDraft({ ...values, [key]: event.target.value })} type="number" value={values[key] ?? ""} />}
      <p className="text-xs text-muted-foreground">{t(`settings.hint.${key}` as TranslationKey)}</p>
      <p className="text-xs text-muted-foreground">{t("settings.owner", { owner: definition.owner })}</p>
    </div>;
  }

  if (settingsQuery.isPending) return <div className="rounded-md border p-4 text-sm text-muted-foreground" role="status">{t("common.loading")}</div>;

  return <div className="flex flex-col gap-8">
    <div><h1 className="text-2xl font-semibold tracking-tight">{t("settings.title")}</h1><p className="text-muted-foreground">{t("settings.description")}</p></div>
    {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
    <div className="grid gap-4 lg:grid-cols-2">
      {Object.entries(SECTION_KEYS).map(([section, keys]) => <Card key={section}>
        <CardHeader><CardTitle>{t(`settings.section.${section}` as TranslationKey)}</CardTitle><CardDescription>{t(`settings.section.${section}Description` as TranslationKey)}</CardDescription></CardHeader>
        <CardContent className="grid gap-4 sm:grid-cols-2">{keys.map((key) => field(key))}</CardContent>
      </Card>)}
    </div>
    <div className="flex items-center justify-between gap-4"><p className="text-sm text-muted-foreground">{t("settings.revision", { revision: settingsQuery.data?.revision ?? 0 })}</p><Button disabled={saving || !settingsQuery.data} onClick={() => void save()}>{saving ? t("settings.saving") : t("common.save")}</Button></div>
  </div>;
}
