import { html } from "htm/preact";
import { useState } from "preact/hooks";
import { Button } from "~/components/ui/button.js";
import { Badge } from "~/components/ui/badge.js";
import { Icon } from "~/components/ui/icon.js";
import { Card } from "~/components/ui/card.js";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell,
} from "~/components/ui/table.js";
import { Alert, AlertTitle, AlertDescription } from "~/components/ui/alert.js";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from "~/components/ui/dialog.js";
import { Field, FieldLabel } from "~/components/ui/field.js";
import { Input } from "~/components/ui/input.js";
import { useStore, addApiClient, toggleApiClient } from "~/routes/store.js";

const open = (id) => document.getElementById(id)?.showModal();
const close = (id) => document.getElementById(id)?.close();

export function ApiClientsPage() {
  const { apiClients } = useStore();
  const [name, setName] = useState("");
  const [token, setToken] = useState("");
  const [copied, setCopied] = useState(false);

  function startCreate() {
    setName("");
    setToken("");
    setCopied(false);
    open("new-client");
  }
  function create() {
    if (!name.trim()) return;
    setToken(addApiClient(name.trim()));
  }
  function copy() {
    navigator.clipboard?.writeText(token).then(() => setCopied(true)).catch(() => setCopied(true));
  }

  return html`
    <div class="flex flex-col flex-1 w-full h-full overflow-y-auto p-6 space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight">API Clients</h1>
          <p class="text-muted-foreground">Tokens de serviço para consumo server-to-server.</p>
        </div>
        <${Button} onClick=${startCreate}><${Icon} icon="lucide:plus" /> Novo client<//>
      </div>

      <${Alert}>
        <${Icon} icon="lucide:shield-alert" />
        <${AlertTitle}>Acesso a segredos em claro<//>
        <${AlertDescription}>Estes clients chamam os endpoints de consumo (/v1/resolve), que retornam valores secret descriptografados sobre TLS. O token só é exibido uma vez, na criação.<//>
      <//>

      <${Card}>
        <${Table}>
          <${TableHeader}>
            <${TableRow}>
              <${TableHead}>Nome<//>
              <${TableHead}>Token<//>
              <${TableHead}>Criado em<//>
              <${TableHead}>Status<//>
              <${TableHead} className="w-10 text-right">Ações<//>
            <//>
          <//>
          <${TableBody}>
            ${apiClients.map((c) => html`
              <${TableRow}>
                <${TableCell} className="font-medium">${c.name}<//>
                <${TableCell}><code class="text-xs">${c.key_preview}</code><//>
                <${TableCell} className="text-muted-foreground">${c.created_at}<//>
                <${TableCell}><${Badge} variant=${c.status === "active" ? "default" : "destructive"}>${c.status}<//><//>
                <${TableCell} className="text-right">
                  <${Button} variant="ghost" size="sm" title=${c.status === "active" ? "Revogar" : "Reativar"} onClick=${() => toggleApiClient(c.id)}>
                    <${Icon} icon=${c.status === "active" ? "lucide:ban" : "lucide:rotate-ccw"} />
                    ${c.status === "active" ? "Revogar" : "Reativar"}
                  <//>
                <//>
              <//>
            `)}
          <//>
        <//>
      <//>

      <${Dialog} id="new-client">
        <${DialogContent}>
          ${!token
            ? html`
              <${DialogHeader}>
                <${DialogTitle}>Novo API client<//>
                <${DialogDescription}>O token será exibido uma única vez após a criação.<//>
              <//>
              <div class="py-2">
                <${Field}>
                  <${FieldLabel}>Nome<//>
                  <${Input} placeholder="billing-service" value=${name} onInput=${(e) => setName(e.target.value)} />
                <//>
              </div>
              <${DialogFooter}>
                <${Button} variant="outline" onClick=${() => close("new-client")}>Cancelar<//>
                <${Button} disabled=${!name.trim()} onClick=${create}>Gerar token<//>
              <//>`
            : html`
              <${DialogHeader}>
                <${DialogTitle}>Token gerado<//>
                <${DialogDescription}>Copie agora — não será possível visualizá-lo novamente.<//>
              <//>
              <div class="py-2 space-y-3">
                <div class="flex items-center gap-2 rounded-md border bg-muted/50 p-3">
                  <code class="flex-1 text-xs break-all">${token}</code>
                  <${Button} variant="outline" size="sm" onClick=${copy}>
                    <${Icon} icon=${copied ? "lucide:check" : "lucide:copy"} /> ${copied ? "Copiado" : "Copiar"}
                  <//>
                </div>
              </div>
              <${DialogFooter}>
                <${Button} onClick=${() => close("new-client")}>Concluir<//>
              <//>`}
        <//>
      <//>
    </div>
  `;
}
