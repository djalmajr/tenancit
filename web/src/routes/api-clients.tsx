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
import { api, type ApiClient } from "@/lib/api";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/api-clients",
  component: ApiClients,
});

function ApiClients() {
  const [clients, setClients] = useState<ApiClient[]>([]);
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [token, setToken] = useState("");
  const [copied, setCopied] = useState(false);

  const load = () => api.listAPIClients().then((c) => setClients(c ?? [])).catch(() => {});
  useEffect(() => void load(), []);

  function start() {
    setName("");
    setToken("");
    setCopied(false);
    setOpen(true);
  }
  async function create() {
    if (!name.trim()) return;
    const res = await api.createAPIClient(name.trim()).catch(() => null);
    if (res) {
      setToken(res.token);
      load();
    }
  }
  function copy() {
    navigator.clipboard?.writeText(token).then(() => setCopied(true)).catch(() => setCopied(true));
  }
  async function toggle(c: ApiClient) {
    await api.setAPIClientStatus(c.id, c.status === "active" ? "revoked" : "active").catch(() => {});
    load();
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">API Clients</h1>
          <p className="text-muted-foreground">Tokens de serviço para consumo server-to-server.</p>
        </div>
        <Button onClick={start}><Plus className="size-4" /> Novo client</Button>
      </div>

      <Card className="flex items-start gap-3 p-4">
        <ShieldAlert className="mt-0.5 size-4 text-amber-500" />
        <div className="text-sm">
          <div className="font-medium">Acesso a segredos em claro</div>
          <p className="text-muted-foreground">
            Estes clients chamam <code className="text-xs">/v1/resolve</code>, que retorna valores secret
            descriptografados sobre TLS. O token só é exibido uma vez, na criação.
          </p>
        </div>
      </Card>

      <Card>
        <Table>
          <THead>
            <TR>
              <TH>Nome</TH>
              <TH>Token</TH>
              <TH>Criado em</TH>
              <TH>Status</TH>
              <TH className="w-16 text-right">Ações</TH>
            </TR>
          </THead>
          <TBody>
            {clients.map((c) => (
              <TR key={c.id}>
                <TD className="font-medium">{c.name}</TD>
                <TD><code className="text-xs text-muted-foreground">rt_live_••••••••</code></TD>
                <TD className="text-muted-foreground">{(c.created_at ?? "").slice(0, 10) || "—"}</TD>
                <TD><Badge variant={c.status === "active" ? "default" : "destructive"}>{c.status}</Badge></TD>
                <TD className="text-right">
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    title={c.status === "active" ? "Revogar" : "Reativar"}
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
                <DialogTitle>Novo API client</DialogTitle>
                <DialogDescription>O token será exibido uma única vez após a criação.</DialogDescription>
              </DialogHeader>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Nome</label>
                <Input placeholder="billing-service" value={name} onChange={(e) => setName(e.target.value)} />
              </div>
              <DialogFooter>
                <DialogClose render={<Button variant="outline">Cancelar</Button>} />
                <Button disabled={!name.trim()} onClick={create}>Gerar token</Button>
              </DialogFooter>
            </>
          ) : (
            <>
              <DialogHeader>
                <DialogTitle>Token gerado</DialogTitle>
                <DialogDescription>Copie agora — não será possível visualizá-lo novamente.</DialogDescription>
              </DialogHeader>
              <div className="flex items-center gap-2 rounded-md border bg-muted/50 p-3">
                <code className="flex-1 break-all text-xs">{token}</code>
                <Button variant="outline" size="sm" onClick={copy}>
                  {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                  {copied ? "Copiado" : "Copiar"}
                </Button>
              </div>
              <DialogFooter>
                <DialogClose render={<Button>Concluir</Button>} />
              </DialogFooter>
            </>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
