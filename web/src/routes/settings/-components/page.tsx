import { useMemo, useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Combobox } from "@/components/ui/combobox";
import { Input } from "@/components/ui/input";
import { api } from "@/lib/api";
import { apiErrorMessage, LOCALE_OPTIONS, useI18n, type TranslationKey } from "@/lib/i18n";
import { adminQueryKeys } from "@/lib/query-keys";
import { useAdminCapabilities } from "@/hooks/use-admin-capabilities";
import { isIntegerInRange } from "@/lib/validation";

const SECTION_KEYS = {
  security: ["session_absolute_hours", "session_idle_minutes"],
  retention: ["usage_retention_months", "audit_retention_days", "webhook_delivery_retention_days", "outbox_event_retention_days"],
  apiClients: ["api_client_default_rpm", "api_client_default_ttl_days"],
  console: ["default_locale"],
} as const;

export default function SettingsPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const { can } = useAdminCapabilities();
  const settingsQuery = useQuery({ queryKey: adminQueryKeys.settings(), queryFn: ({ signal }) => api.getSettings(signal) });
  const [draft, setDraft] = useState<Record<string, string>>();
  const [savingSection, setSavingSection] = useState<string>();
  const values = draft ?? settingsQuery.data?.values ?? {};
  const definitions = useMemo(() => new Map(settingsQuery.data?.definitions.map((definition) => [definition.key, definition]) ?? []), [settingsQuery.data]);
  const error = settingsQuery.error ? apiErrorMessage(settingsQuery.error, t) : "";

  async function saveSection(section: string, keys: readonly string[]) {
    if (!settingsQuery.data || !keys.every((key) => settingValid(key))) return;
    setSavingSection(section);
    try {
      const updates = Object.fromEntries(keys.map((key) => [key, values[key]]));
      const updated = await api.updateSettings(updates, settingsQuery.data.revision);
      queryClient.setQueryData(adminQueryKeys.settings(), updated);
      setDraft((current) => {
        if (!current) return undefined;
        const next = { ...current };
        for (const key of keys) next[key] = updated.values[key];
        return next;
      });
      toast.success(t("settings.saved"));
    } catch (cause) {
      toast.error(apiErrorMessage(cause, t));
      await queryClient.invalidateQueries({ queryKey: adminQueryKeys.settings() });
    } finally {
      setSavingSection(undefined);
    }
  }

  function settingValid(key: string) {
    const definition = definitions.get(key);
    const value = values[key] ?? "";
    if (!definition) return false;
    if (definition.type === "enum") return (definition.options ?? []).includes(value);
    return definition.minimum !== undefined
      && definition.maximum !== undefined
      && isIntegerInRange(value, definition.minimum, definition.maximum);
  }

  function field(key: string) {
    const definition = definitions.get(key);
    if (!definition) return null;
    const label = t(`settings.field.${key}` as TranslationKey);
    return <div className="flex flex-col gap-1.5" key={key}>
      <label className="text-sm font-medium" htmlFor={`setting-${key}`}>{label}</label>
      {definition.type === "enum" ? <Combobox aria-label={label} disabled={!can("settings.manage")} options={(definition.options ?? []).map((option) => {
        const locale = key === "default_locale" ? LOCALE_OPTIONS.find((item) => item.value === option) : undefined;
        return { label: locale ? `${locale.flag} ${locale.label}` : option, value: option };
      })} searchable={false} triggerClassName="w-full" value={values[key]} onValueChange={(value) => setDraft({ ...values, [key]: value })} /> : <Input aria-invalid={!settingValid(key)} disabled={!can("settings.manage")} id={`setting-${key}`} inputMode="numeric" max={definition.maximum} min={definition.minimum} onChange={(event) => setDraft({ ...values, [key]: event.target.value })} step="1" type="number" value={values[key] ?? ""} />}
      <p className="text-xs text-muted-foreground">{t(`settings.hint.${key}` as TranslationKey)}</p>
    </div>;
  }

  if (settingsQuery.isPending) return <div className="rounded-md border p-4 text-sm text-muted-foreground" role="status">{t("common.loading")}</div>;

  return <div className="flex flex-col gap-4">
    {error && <Alert variant="destructive"><AlertDescription>{error}</AlertDescription></Alert>}
    {Object.entries(SECTION_KEYS).map(([section, keys]) => <Card key={section}>
      <CardHeader>
        <CardTitle>{t(`settings.section.${section}` as TranslationKey)}</CardTitle>
        <CardDescription>{t(`settings.section.${section}Description` as TranslationKey)}</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-4">
        <div className="grid gap-4 md:grid-cols-2">{keys.map((key) => field(key))}</div>
        {can("settings.manage") && <div>
          <Button
            disabled={Boolean(savingSection) || !settingsQuery.data || !keys.every((key) => settingValid(key))}
            onClick={() => void saveSection(section, keys)}
          >
            {savingSection === section ? t("settings.saving") : t("common.save")}
          </Button>
        </div>}
      </CardContent>
    </Card>)}
    <p className="text-xs text-muted-foreground">{t("settings.revision", { revision: settingsQuery.data?.revision ?? 0 })}</p>
  </div>;
}
