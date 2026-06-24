import { createRoute } from "@tanstack/react-router";
import { useEffect, useState } from "react";
import { Plus, ShieldAlert, Copy, Check, Ban, RotateCcw } from "lucide-react";
import { Route as rootRoute } from "./__root";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Table, THead, TBody, TR, TH, TD } from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { formatStatus, useI18n } from "@/lib/i18n";
import { api, type ApiClient } from "@/lib/api";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/api-clients",
  component: ApiClients,
});

function ApiClients() {
  const { t } = useI18n();
  const [clients, setClients] = useState<ApiClient[]>([]);
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [token, setToken] = useState("");
  const [copied, setCopied] = useState(false);
  const [error, setError] = useState("");

  const load = () => api.listAPIClients().then((c) => {
    setClients(c ?? []);
    setError("");
  }).catch((e) => setError(String(e)));
  useEffect(() => void load(), []);

  function start() {
    setName("");
    setToken("");
    setCopied(false);
    setError("");
    setOpen(true);
  }
  async function create() {
    if (!name.trim()) return;
    try {
      const res = await api.createAPIClient(name.trim());
      setError("");
      setToken(res.token);
      load();
    } catch (e) {
      setError(String(e));
    }
  }
  function copy() {
    navigator.clipboard?.writeText(token).then(() => setCopied(true)).catch(() => setCopied(true));
  }
  async function toggle(c: ApiClient) {
    try {
      await api.setAPIClientStatus(c.id, c.status === "active" ? "revoked" : "active");
      load();
    } catch (e) {
      setError(String(e));
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("apiClients.title")}</h1>
          <p className="text-muted-foreground">{t("apiClients.description")}</p>
        </div>
        <Button onClick={start}><Plus className="size-4" /> {t("apiClients.new")}</Button>
      </div>

      <Card className="flex items-start gap-3 p-4">
        <ShieldAlert className="mt-0.5 size-4 text-amber-500" />
        <div className="text-sm">
          <div className="font-medium">{t("apiClients.clearSecretAccess.title")}</div>
          <p className="text-muted-foreground">
            {t("apiClients.clearSecretAccess.description")}
          </p>
        </div>
      </Card>

      {error && <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">{error}</div>}

      <Card>
        <Table>
          <THead>
            <TR>
              <TH>{t("apiClients.name")}</TH>
              <TH>{t("apiClients.token")}</TH>
              <TH>{t("apiClients.createdAt")}</TH>
              <TH>{t("common.status")}</TH>
              <TH className="w-16 text-right">{t("apiClients.actions")}</TH>
            </TR>
          </THead>
          <TBody>
            {clients.map((c) => (
              <TR key={c.id}>
                <TD className="font-medium">{c.name}</TD>
                <TD><code className="text-xs text-muted-foreground">rt_live_••••••••</code></TD>
                <TD className="text-muted-foreground">{(c.created_at ?? "").slice(0, 10) || "—"}</TD>
                <TD><Badge variant={c.status === "active" ? "default" : "destructive"}>{formatStatus(c.status, t)}</Badge></TD>
                <TD className="text-right">
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    title={c.status === "active" ? t("apiClients.revoke") : t("apiClients.reactivate")}
                    onClick={() => toggle(c)}
                  >
                    {c.status === "active" ? <Ban className="size-4" /> : <RotateCcw className="size-4" />}
                  </Button>
                </TD>
              </TR>
            ))}
          </TBody>
        </Table>
      </Card>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          {!token ? (
            <>
              <DialogHeader>
                <DialogTitle>{t("apiClients.newDialog.title")}</DialogTitle>
                <DialogDescription>{t("apiClients.newDialog.description")}</DialogDescription>
              </DialogHeader>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">{t("apiClients.name")}</label>
                <Input placeholder="billing-service" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <DialogFooter>
                <DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} />
                <Button disabled={!name.trim()} onClick={create}>{t("apiClients.generateToken")}</Button>
              </DialogFooter>
            </>
          ) : (
            <>
              <DialogHeader>
                <DialogTitle>{t("apiClients.tokenGenerated.title")}</DialogTitle>
                <DialogDescription>{t("apiClients.tokenGenerated.description")}</DialogDescription>
              </DialogHeader>
              <div className="flex items-center gap-2 rounded-md border bg-muted/50 p-3">
                <code className="flex-1 break-all text-xs">{token}</code>
                <Button variant="outline" size="sm" onClick={copy}>
                  {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                  {copied ? t("apiClients.copied") : t("apiClients.copy")}
                </Button>
              </div>
              <DialogFooter>
                <DialogClose render={<Button>{t("apiClients.done")}</Button>} />
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
