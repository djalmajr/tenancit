import { html } from "htm/preact";
import { useState } from "preact/hooks";
import { useRoute, useLocation } from "preact-iso";
import { Button } from "~/components/ui/button.js";
import { Badge } from "~/components/ui/badge.js";
import { Icon } from "~/components/ui/icon.js";
import { Input } from "~/components/ui/input.js";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "~/components/ui/card.js";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell,
} from "~/components/ui/table.js";
import { Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription, EmptyContent } from "~/components/ui/empty.js";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from "~/components/ui/dialog.js";
import { Field, FieldLabel, FieldDescription } from "~/components/ui/field.js";
import { Select, SelectItem } from "~/components/ui/select.js";
import {
  useStore, findTenant, findDefinitionByKey,
  updateTenant, addResource, toggleResourceStatus, removeResource,
  addDomain, removeDomain,
} from "~/routes/store.js";

const open = (id) => document.getElementById(id)?.showModal();
const close = (id) => document.getElementById(id)?.close();

function RevealableValue({ value }) {
  const [shown, setShown] = useState(false);
  return html`
    <div class="flex items-center gap-2">
	      <code class="text-xs">${shown ? value || "—" : "••••••••••••"}</code>
      <button class="text-muted-foreground hover:text-foreground" onClick=${() => setShown(!shown)} title=${shown ? "Ocultar" : "Revelar"}>
        <${Icon} icon=${shown ? "lucide:eye-off" : "lucide:eye"} />
      </button>
    </div>
  `;
}

function ResourceCard({ tenantId, resource }) {
  const def = findDefinitionByKey(resource.definitionKey);
  const isActive = resource.status === "active";
  return html`
    <${Card}>
      <${CardHeader} className="flex flex-row items-center justify-between space-y-0">
        <div class="flex items-center gap-2">
          <iconify-icon icon=${def?.icon || "lucide:box"} width="18" class="text-muted-foreground"></iconify-icon>
          <${CardTitle} className="text-base">${resource.definitionName}<//>
          <code class="text-xs text-muted-foreground">${resource.definitionKey}</code>
        </div>
        <div class="flex items-center gap-2">
          <${Badge} variant=${isActive ? "default" : "secondary"}>${resource.status}<//>
          <${Button} variant="ghost" size="icon-sm" title=${isActive ? "Desativar" : "Reativar"} onClick=${() => toggleResourceStatus(tenantId, resource.id)}>
            <${Icon} icon=${isActive ? "lucide:power" : "lucide:power-off"} />
          <//>
          <${Button} variant="ghost" size="icon-sm" title="Remover" onClick=${() => removeResource(tenantId, resource.id)}>
            <${Icon} icon="lucide:trash-2" />
          <//>
        </div>
      <//>
      <${CardContent}>
        <${Table}>
          <${TableHeader}>
            <${TableRow}>
              <${TableHead}>Campo<//>
              <${TableHead}>Valor<//>
              <${TableHead} className="w-24"><//>
            <//>
          <//>
          <${TableBody}>
            ${(def?.fields || []).map((f) => {
              const val = resource.values[f.key];
              return html`
                <${TableRow}>
                  <${TableCell} className="font-medium">
                    ${f.label}
                    ${f.is_secret && html`<${Icon} icon="lucide:lock" className="ml-1 inline text-amber-500" />`}
                  <//>
                  <${TableCell}>
                    ${f.is_secret ? html`<${RevealableValue} value=${val} />` : html`<code class="text-xs">${val || "—"}</code>`}
                  <//>
                  <${TableCell}>
                    ${f.is_secret && html`<${Badge} variant="outline" className="text-amber-600 border-amber-300">secret<//>`}
                  <//>
                <//>
              `;
            })}
          <//>
        <//>
      <//>
    <//>
  `;
}

export function TenantDetailPage() {
  const { params } = useRoute();
  const { route } = useLocation();
  useStore();
  const tenant = findTenant(params.id);
  const { definitions } = useStore();

  const [tab, setTab] = useState("resources");
  const [edit, setEdit] = useState({ name: "", slug: "", status: "active" });
  const [hostname, setHostname] = useState("");
  const [pick, setPick] = useState("");
  const [values, setValues] = useState({});

  if (!tenant) {
    return html`
      <div class="flex flex-col items-center justify-center flex-1 gap-3 p-6">
        <p class="text-muted-foreground">Tenant não encontrado.</p>
        <${Button} variant="outline" onClick=${() => route("/tenants")}>Voltar para Tenants<//>
      </div>
    `;
  }

  // Regra: só 1 ativo por tipo -> só ofereço definitions sem recurso ATIVO no tenant.
  const activeKeys = new Set(tenant.resources.filter((r) => r.status === "active").map((r) => r.definitionKey));
  const available = definitions.filter((d) => d.status === "active" && !activeKeys.has(d.key));
  const pickedDef = available.find((d) => d.key === pick);

  function openEdit() {
    setEdit({ name: tenant.name, slug: tenant.slug, status: tenant.status });
    open("edit-tenant");
  }
  function saveEdit() {
    if (!edit.name.trim()) return;
    updateTenant(tenant.id, { name: edit.name, slug: edit.slug, status: edit.status });
    close("edit-tenant");
  }

  function openAddResource() {
    setPick("");
    setValues({});
    open("new-resource");
  }
  function choose(key) {
    setPick(key);
    const def = available.find((d) => d.key === key);
    const init = {};
    (def?.fields || []).forEach((f) => (init[f.key] = ""));
    setValues(init);
  }
  const requiredFilled = pickedDef && pickedDef.fields.filter((f) => f.required).every((f) => (values[f.key] || "").trim());
  function saveResource() {
    if (!pickedDef || !requiredFilled) return;
    addResource(tenant.id, pickedDef, { ...values });
    close("new-resource");
  }

  function saveDomain() {
    if (!hostname.trim()) return;
    addDomain(tenant.id, hostname.trim());
    setHostname("");
    close("new-domain");
  }

  const TabBtn = (id, label, icon) => html`
    <button
      class=${`flex items-center gap-2 border-b-2 px-3 py-2 text-sm font-medium transition-colors ${
        tab === id ? "border-primary text-foreground" : "border-transparent text-muted-foreground hover:text-foreground"
      }`}
      onClick=${() => setTab(id)}>
      <${Icon} icon=${icon} /> ${label}
    </button>
  `;

  return html`
    <div class="flex flex-col flex-1 w-full h-full overflow-y-auto p-6 space-y-6">
      <div class="flex items-center gap-2 text-sm text-muted-foreground">
        <a class="hover:text-foreground cursor-pointer" onClick=${() => route("/tenants")}>Tenants</a>
        <${Icon} icon="lucide:chevron-right" />
        <span class="text-foreground">${tenant.name}</span>
      </div>

      <div class="flex items-start justify-between">
        <div class="flex items-center gap-3">
          <div class="flex size-11 items-center justify-center rounded-lg bg-muted">
            <${Icon} icon="lucide:building-2" />
          </div>
          <div>
            <h1 class="text-2xl font-semibold tracking-tight">${tenant.name}</h1>
            <div class="flex items-center gap-2 text-sm text-muted-foreground">
              <code>${tenant.slug}</code>
              <${Badge} variant=${tenant.status === "active" ? "default" : "secondary"}>${tenant.status}<//>
            </div>
          </div>
        </div>
        <${Button} variant="outline" onClick=${openEdit}><${Icon} icon="lucide:settings-2" /> Editar<//>
      </div>

      <div class="flex gap-1 border-b">
        ${TabBtn("resources", "Recursos", "lucide:database")}
        ${TabBtn("domains", "Domínios", "lucide:globe")}
      </div>

      ${tab === "resources" && html`
        <div class="space-y-4">
          <div class="flex justify-end">
            <${Button} onClick=${openAddResource}>
              <${Icon} icon="lucide:plus" /> Adicionar recurso
            <//>
          </div>
          ${tenant.resources.length === 0
            ? html`
              <${Empty}>
                <${EmptyHeader}>
                  <${EmptyMedia}><${Icon} icon="lucide:database" /><//>
                  <${EmptyTitle}>Nenhum recurso<//>
                  <${EmptyDescription}>Este tenant ainda não tem recursos provisionados.<//>
                <//>
                <${EmptyContent}>
                  <${Button} onClick=${openAddResource}><${Icon} icon="lucide:plus" /> Adicionar recurso<//>
                <//>
              <//>`
            : tenant.resources.map((r) => html`<${ResourceCard} tenantId=${tenant.id} resource=${r} />`)}
        </div>
      `}

      ${tab === "domains" && html`
        <${Card}>
          <${CardHeader} className="flex flex-row items-center justify-between space-y-0">
            <div>
              <${CardTitle} className="text-base">Domínios<//>
              <${CardDescription}>Hostnames que resolvem para este tenant.<//>
            </div>
            <${Button} variant="outline" size="sm" onClick=${() => { setHostname(""); open("new-domain"); }}><${Icon} icon="lucide:plus" /> Adicionar<//>
          <//>
          <${CardContent}>
            ${tenant.domains.length === 0
              ? html`<p class="text-sm text-muted-foreground">Nenhum domínio. Adicione ao menos um para resolver o tenant.</p>`
              : html`
                <${Table}>
                  <${TableHeader}>
                    <${TableRow}>
                      <${TableHead}>Hostname<//>
                      <${TableHead} className="w-24 text-right">Ações<//>
                    <//>
                  <//>
                  <${TableBody}>
                    ${tenant.domains.map((d) => html`
                      <${TableRow}>
                        <${TableCell} className="font-medium"><code class="text-xs">${d.hostname}</code><//>
                        <${TableCell} className="text-right">
                          <${Button} variant="ghost" size="icon-sm" title="Remover" onClick=${() => removeDomain(tenant.id, d.id)}><${Icon} icon="lucide:trash-2" /><//>
                        <//>
                      <//>
                    `)}
                  <//>
                <//>`}
          <//>
        <//>
      `}
    </div>

    <!-- Editar tenant -->
    <${Dialog} id="edit-tenant">
      <${DialogContent}>
        <${DialogHeader}>
          <${DialogTitle}>Editar tenant<//>
          <${DialogDescription}>Atualize os dados do cliente.<//>
        <//>
        <div class="space-y-4 py-2">
          <${Field}><${FieldLabel}>Nome<//><${Input} value=${edit.name} onInput=${(e) => setEdit({ ...edit, name: e.target.value })} /><//>
          <${Field}><${FieldLabel}>Slug<//><${Input} value=${edit.slug} onInput=${(e) => setEdit({ ...edit, slug: e.target.value })} /><//>
          <${Field}>
            <${FieldLabel}>Status<//>
            <${Select} value=${edit.status} onChange=${(e) => setEdit({ ...edit, status: e.target.value })}>
              <${SelectItem} value="active">active<//>
              <${SelectItem} value="inactive">inactive<//>
            <//>
          <//>
        </div>
        <${DialogFooter}>
          <${Button} variant="outline" onClick=${() => close("edit-tenant")}>Cancelar<//>
          <${Button} disabled=${!edit.name.trim()} onClick=${saveEdit}>Salvar<//>
        <//>
      <//>
    <//>

    <!-- Adicionar recurso -->
    <${Dialog} id="new-resource">
      <${DialogContent}>
        <${DialogHeader}>
          <${DialogTitle}>Adicionar recurso<//>
          <${DialogDescription}>Selecione um tipo. Os campos vêm do template; só 1 ativo por tipo.<//>
        <//>
        <div class="space-y-4 py-2">
          ${available.length === 0
            ? html`<div class="rounded-md border border-dashed p-4 text-sm text-muted-foreground">Todos os tipos de recurso ativos já estão provisionados (ou não há definitions ativas). Desative um recurso para trocar, ou crie uma nova definition.</div>`
            : html`
              <${Field}>
                <${FieldLabel}>Tipo de recurso<//>
                <${Select} value=${pick} onChange=${(e) => choose(e.target.value)}>
                  <${SelectItem} value="">Selecione...<//>
                  ${available.map((d) => html`<${SelectItem} value=${d.key}>${d.name}<//>`)}
                <//>
              <//>
              ${pickedDef && pickedDef.fields.length === 0 && html`<p class="text-sm text-muted-foreground">Este template não tem campos definidos.</p>`}
              ${pickedDef && pickedDef.fields.map((f) => html`
                <${Field}>
                  <${FieldLabel}>${f.label} ${f.required && html`<span class="text-destructive">*</span>`} ${f.is_secret && html`<${Icon} icon="lucide:lock" className="inline text-amber-500" />`}<//>
                  <${Input}
                    type=${f.is_secret ? "password" : f.data_type === "int" ? "number" : "text"}
                    placeholder=${f.key}
                    value=${values[f.key] || ""}
                    onInput=${(e) => setValues({ ...values, [f.key]: e.target.value })}
                  />
                <//>
              `)}`}
        </div>
        <${DialogFooter}>
          <${Button} variant="outline" onClick=${() => close("new-resource")}>Cancelar<//>
          <${Button} disabled=${!pickedDef || !requiredFilled} onClick=${saveResource}>Salvar recurso<//>
        <//>
      <//>
    <//>

    <!-- Adicionar domínio -->
    <${Dialog} id="new-domain">
      <${DialogContent}>
        <${DialogHeader}>
          <${DialogTitle}>Adicionar domínio<//>
          <${DialogDescription}>Hostname exato que resolve para este tenant.<//>
        <//>
        <div class="py-2">
          <${Field}>
            <${FieldLabel}>Hostname<//>
            <${Input} placeholder="app.cliente.com" value=${hostname} onInput=${(e) => setHostname(e.target.value)} />
          <//>
        </div>
        <${DialogFooter}>
          <${Button} variant="outline" onClick=${() => close("new-domain")}>Cancelar<//>
          <${Button} disabled=${!hostname.trim()} onClick=${saveDomain}>Adicionar<//>
        <//>
      <//>
    <//>
  `;
}
