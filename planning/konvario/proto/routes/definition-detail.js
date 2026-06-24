import { html } from "htm/preact";
import { useState } from "preact/hooks";
import { useRoute, useLocation } from "preact-iso";
import { Button } from "~/components/ui/button.js";
import { Badge } from "~/components/ui/badge.js";
import { Icon } from "~/components/ui/icon.js";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "~/components/ui/card.js";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell,
} from "~/components/ui/table.js";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription } from "~/components/ui/empty.js";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from "~/components/ui/dialog.js";
import { Field, FieldLabel } from "~/components/ui/field.js";
import { Input } from "~/components/ui/input.js";
import { Select, SelectItem } from "~/components/ui/select.js";
import { Switch } from "~/components/ui/switch.js";
import { useStore, findDefinition, addField, removeField, toggleDefinitionStatus } from "~/routes/store.js";

const open = (id) => document.getElementById(id)?.showModal();
const close = (id) => document.getElementById(id)?.close();

const EMPTY_FIELD = { key: "", label: "", data_type: "string", required: false, is_secret: false };

export function DefinitionDetailPage() {
  const { params } = useRoute();
  const { route } = useLocation();
  useStore();
  const def = findDefinition(params.id);
  const [f, setF] = useState({ ...EMPTY_FIELD });

  if (!def) {
    return html`
      <div class="flex flex-col items-center justify-center flex-1 gap-3 p-6">
        <p class="text-muted-foreground">Definition não encontrada.</p>
        <${Button} variant="outline" onClick=${() => route("/resource-definitions")}>Voltar<//>
      </div>
    `;
  }

  const canAdd = f.key.trim() && f.label.trim();
  function openNew() { setF({ ...EMPTY_FIELD }); open("new-field"); }
  function save() {
    if (!canAdd) return;
    addField(def.id, { ...f });
    close("new-field");
  }

  return html`
    <div class="flex flex-col flex-1 w-full h-full overflow-y-auto p-6 space-y-6">
      <div class="flex items-center gap-2 text-sm text-muted-foreground">
        <a class="hover:text-foreground cursor-pointer" onClick=${() => route("/resource-definitions")}>Resource Definitions</a>
        <${Icon} icon="lucide:chevron-right" />
        <span class="text-foreground">${def.name}</span>
      </div>

      <div class="flex items-start justify-between">
        <div class="flex items-center gap-3">
          <div class="flex size-11 items-center justify-center rounded-lg bg-muted">
            <iconify-icon icon=${def.icon || "lucide:box"} width="20"></iconify-icon>
          </div>
          <div>
            <h1 class="text-2xl font-semibold tracking-tight">${def.name}</h1>
            <div class="flex items-center gap-2 text-sm text-muted-foreground">
              <code>${def.key}</code>
              <${Badge} variant=${def.status === "active" ? "default" : "secondary"}>${def.status}<//>
            </div>
          </div>
        </div>
        <div class="flex gap-2">
          <${Button} variant="outline" onClick=${() => toggleDefinitionStatus(def.id)}>
            <${Icon} icon=${def.status === "active" ? "lucide:power-off" : "lucide:power"} />
            ${def.status === "active" ? "Desativar" : "Ativar"}
          <//>
          <${Button} onClick=${openNew}><${Icon} icon="lucide:plus" /> Novo campo<//>
        </div>
      </div>

      <${Card}>
        <${CardHeader}>
          <${CardTitle} className="text-base">Campos<//>
          <${CardDescription}>Definição dinâmica dos campos deste tipo de recurso.<//>
        <//>
        <${CardContent}>
          ${def.fields.length === 0
            ? html`
              <${Empty}>
                <${EmptyHeader}>
                  <${EmptyMedia}><${Icon} icon="lucide:list" /><//>
                  <${EmptyTitle}>Nenhum campo<//>
                  <${EmptyDescription}>Adicione os campos que compõem este recurso.<//>
                <//>
              <//>`
            : html`
              <${Table}>
                <${TableHeader}>
                  <${TableRow}>
                    <${TableHead}>Chave<//>
                    <${TableHead}>Label<//>
                    <${TableHead}>Tipo<//>
                    <${TableHead}>Required<//>
                    <${TableHead}>Secret<//>
                    <${TableHead} className="w-10 text-right">Ações<//>
                  <//>
                <//>
                <${TableBody}>
                  ${def.fields.map((field) => html`
                    <${TableRow}>
                      <${TableCell}><code class="text-xs">${field.key}</code><//>
                      <${TableCell}>${field.label}<//>
                      <${TableCell}><code class="text-xs text-muted-foreground">${field.data_type}</code><//>
                      <${TableCell}>${field.required ? html`<${Icon} icon="lucide:check" className="text-green-600" />` : html`<span class="text-muted-foreground">—</span>`}<//>
                      <${TableCell}>${field.is_secret ? html`<${Icon} icon="lucide:check" className="text-amber-600" />` : html`<span class="text-muted-foreground">—</span>`}<//>
                      <${TableCell} className="text-right"><${Button} variant="ghost" size="icon-sm" title="Remover" onClick=${() => removeField(def.id, field.id)}><${Icon} icon="lucide:trash-2" /><//><//>
                    <//>
                  `)}
                <//>
              <//>`}
        <//>
      <//>
    </div>

    <${Dialog} id="new-field">
      <${DialogContent}>
        <${DialogHeader}>
          <${DialogTitle}>Novo campo<//>
          <${DialogDescription}>Adicione um campo a este tipo de recurso.<//>
        <//>
        <div class="space-y-4 py-2">
          <div class="grid grid-cols-2 gap-3">
            <${Field}><${FieldLabel}>Chave<//><${Input} placeholder="host" value=${f.key} onInput=${(e) => setF({ ...f, key: e.target.value })} /><//>
            <${Field}><${FieldLabel}>Label<//><${Input} placeholder="Host" value=${f.label} onInput=${(e) => setF({ ...f, label: e.target.value })} /><//>
          </div>
          <${Field}>
            <${FieldLabel}>Tipo<//>
            <${Select} value=${f.data_type} onChange=${(e) => setF({ ...f, data_type: e.target.value })}>
              <${SelectItem} value="string">string<//>
              <${SelectItem} value="int">int<//>
              <${SelectItem} value="bool">bool<//>
            <//>
          <//>
          <div class="flex items-center justify-between rounded-md border p-3">
            <div><div class="text-sm font-medium">Obrigatório</div><div class="text-xs text-muted-foreground">Exigido na criação do recurso.</div></div>
            <${Switch} checked=${f.required} onChange=${(e) => setF({ ...f, required: e.target.checked })} />
          </div>
          <div class="flex items-center justify-between rounded-md border p-3">
            <div><div class="text-sm font-medium">Secret</div><div class="text-xs text-muted-foreground">Criptografado com AES-256-GCM.</div></div>
            <${Switch} checked=${f.is_secret} onChange=${(e) => setF({ ...f, is_secret: e.target.checked })} />
          </div>
        </div>
        <${DialogFooter}>
          <${Button} variant="outline" onClick=${() => close("new-field")}>Cancelar<//>
          <${Button} disabled=${!canAdd} onClick=${save}>Adicionar campo<//>
        <//>
      <//>
    <//>
  `;
}
