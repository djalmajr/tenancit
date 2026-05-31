import { html } from "htm/preact";
import { useState } from "preact/hooks";
import { useLocation } from "preact-iso";
import { Button } from "~/components/ui/button.js";
import { Badge } from "~/components/ui/badge.js";
import { Icon } from "~/components/ui/icon.js";
import { Input } from "~/components/ui/input.js";
import {
  Table, TableHeader, TableBody, TableRow, TableHead, TableCell,
} from "~/components/ui/table.js";
import { Card } from "~/components/ui/card.js";
import {
  Empty, EmptyHeader, EmptyMedia, EmptyTitle, EmptyDescription, EmptyContent,
} from "~/components/ui/empty.js";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from "~/components/ui/dialog.js";
import { Field, FieldLabel, FieldDescription } from "~/components/ui/field.js";
import { useStore, addTenant } from "~/routes/store.js";

const open = (id) => document.getElementById(id)?.showModal();
const close = (id) => document.getElementById(id)?.close();

export function TenantsListPage() {
  const { route } = useLocation();
  const { tenants } = useStore();
  const [query, setQuery] = useState("");
  const [form, setForm] = useState({ name: "", slug: "", hostname: "" });

  const q = query.trim().toLowerCase();
  const filtered = !q
    ? tenants
    : tenants.filter(
        (t) =>
          t.name.toLowerCase().includes(q) ||
          t.slug.toLowerCase().includes(q) ||
          t.domains.some((d) => d.hostname.toLowerCase().includes(q))
      );

  const canCreate = form.name.trim() && form.hostname.trim();

  function submit() {
    if (!canCreate) return;
    const id = addTenant(form);
    setForm({ name: "", slug: "", hostname: "" });
    close("new-tenant");
    route(`/tenants/${id}`);
  }

  return html`
    <div class="flex flex-col flex-1 w-full h-full overflow-y-auto p-6 space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight">Tenants</h1>
          <p class="text-muted-foreground">Clientes e seus domínios.</p>
        </div>
        <${Button} onClick=${() => open("new-tenant")}>
          <${Icon} icon="lucide:plus" /> Novo tenant
        <//>
      </div>

      <div class="relative max-w-sm">
        <${Icon} icon="lucide:search" className="absolute left-2.5 top-2.5 text-muted-foreground" />
        <${Input}
          className="pl-8"
          placeholder="Buscar por nome, slug ou hostname..."
          value=${query}
          onInput=${(e) => setQuery(e.target.value)}
        />
      </div>

      <${Card}>
        ${filtered.length === 0
          ? html`
            <${Empty}>
              <${EmptyHeader}>
                <${EmptyMedia}><${Icon} icon="lucide:building-2" /><//>
                <${EmptyTitle}>${q ? "Nenhum resultado" : "Nenhum tenant"}<//>
                <${EmptyDescription}>${q ? "Ajuste a busca." : "Cadastre o primeiro cliente."}<//>
              <//>
              ${!q && html`<${EmptyContent}><${Button} onClick=${() => open("new-tenant")}><${Icon} icon="lucide:plus" /> Novo tenant<//><//>`}
            <//>`
          : html`
            <${Table}>
              <${TableHeader}>
                <${TableRow}>
                  <${TableHead}>Nome<//>
                  <${TableHead}>Slug<//>
                  <${TableHead}>Domínio<//>
                  <${TableHead}>Recursos<//>
                  <${TableHead}>Status<//>
                  <${TableHead} className="w-10"><//>
                <//>
              <//>
              <${TableBody}>
                ${filtered.map(
                  (t) => html`
                    <${TableRow} className="cursor-pointer" onClick=${() => route(`/tenants/${t.id}`)}>
                      <${TableCell} className="font-medium">${t.name}<//>
                      <${TableCell}><code class="text-xs">${t.slug}</code><//>
                      <${TableCell} className="text-muted-foreground">${t.domains[0]?.hostname || "—"}<//>
                      <${TableCell}>${t.resources.length}<//>
                      <${TableCell}><${Badge} variant=${t.status === "active" ? "default" : "secondary"}>${t.status}<//><//>
                      <${TableCell}><${Icon} icon="lucide:chevron-right" className="text-muted-foreground" /><//>
                    <//>
                  `
                )}
              <//>
            <//>`}
      <//>

      <${Dialog} id="new-tenant">
        <${DialogContent}>
          <${DialogHeader}>
            <${DialogTitle}>Novo tenant<//>
            <${DialogDescription}>Cadastre um cliente. Domínios e recursos são adicionados em seguida.<//>
          <//>
          <div class="space-y-4 py-2">
            <${Field}>
              <${FieldLabel}>Nome<//>
              <${Input} placeholder="Acme Corp" value=${form.name} onInput=${(e) => setForm({ ...form, name: e.target.value })} />
            <//>
            <${Field}>
              <${FieldLabel}>Slug<//>
              <${Input} placeholder="acme" value=${form.slug} onInput=${(e) => setForm({ ...form, slug: e.target.value })} />
              <${FieldDescription}>Opcional — gerado a partir do nome se vazio.<//>
            <//>
            <${Field}>
              <${FieldLabel}>Hostname<//>
              <${Input} placeholder="app.acme.com" value=${form.hostname} onInput=${(e) => setForm({ ...form, hostname: e.target.value })} />
              <${FieldDescription}>Usado para resolver o tenant a partir da URL.<//>
            <//>
          </div>
          <${DialogFooter}>
            <${Button} variant="outline" onClick=${() => close("new-tenant")}>Cancelar<//>
            <${Button} disabled=${!canCreate} onClick=${submit}>Criar tenant<//>
          <//>
        <//>
      <//>
    </div>
  `;
}
