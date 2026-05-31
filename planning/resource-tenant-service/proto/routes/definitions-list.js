import { html } from "htm/preact";
import { useState, useEffect } from "preact/hooks";
import { useLocation } from "preact-iso";
import { Button } from "~/components/ui/button.js";
import { Badge } from "~/components/ui/badge.js";
import { Icon } from "~/components/ui/icon.js";
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "~/components/ui/card.js";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter,
} from "~/components/ui/dialog.js";
import { Field, FieldLabel, FieldDescription } from "~/components/ui/field.js";
import { Input } from "~/components/ui/input.js";
import { Textarea } from "~/components/ui/textarea.js";
import { useStore, addDefinition } from "~/routes/store.js";

const open = (id) => document.getElementById(id)?.showModal();
const close = (id) => document.getElementById(id)?.close();

// Conjunto curado (fallback instantâneo, sem rede) — filtrado localmente.
const COMMON_ICONS = [
  "lucide:database", "lucide:hard-drive", "lucide:server", "lucide:cloud",
  "lucide:mail", "lucide:bell", "lucide:key-round", "lucide:lock", "lucide:shield",
  "lucide:box", "lucide:boxes", "lucide:package", "lucide:container",
  "lucide:globe", "lucide:network", "lucide:wifi", "lucide:link",
  "lucide:credit-card", "lucide:wallet", "lucide:bar-chart", "lucide:activity",
  "lucide:settings", "lucide:cog", "lucide:cpu", "lucide:memory-stick",
  "lucide:file-text", "lucide:folder", "lucide:image", "lucide:video",
  "lucide:message-square", "lucide:phone", "lucide:send", "lucide:webhook",
  "lucide:flame", "lucide:zap", "lucide:bot", "lucide:brain",
  "simple-icons:postgresql", "simple-icons:mysql", "simple-icons:redis",
  "simple-icons:mongodb", "simple-icons:minio", "simple-icons:amazons3",
  "simple-icons:rabbitmq", "simple-icons:apachekafka", "simple-icons:elasticsearch",
];

// Busca ícones na API do Iconify (com timeout) e cai no conjunto curado se falhar.
function IconPicker({ value, onPick }) {
  const [q, setQ] = useState("");
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const term = q.trim().toLowerCase();
    if (!term) { setResults([]); setLoading(false); return; }
    const local = COMMON_ICONS.filter((n) => n.includes(term));
    setResults(local);
    let active = true;
    setLoading(true);
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), 4000);
    const debounce = setTimeout(() => {
      fetch(`https://api.iconify.design/search?query=${encodeURIComponent(term)}&limit=48`, { signal: ctrl.signal })
        .then((r) => r.json())
        .then((d) => { if (active && d.icons && d.icons.length) setResults(d.icons); })
        .catch(() => {})
        .finally(() => { if (active) setLoading(false); clearTimeout(timer); });
    }, 300);
    return () => { active = false; clearTimeout(debounce); clearTimeout(timer); ctrl.abort(); };
  }, [q]);

  const select = (name) => { onPick(name); setQ(""); };

  return html`
    <div class="space-y-2">
      <div class="flex items-center gap-2">
        <div class="flex size-9 items-center justify-center rounded-md border bg-muted shrink-0">
          <iconify-icon icon=${value || "lucide:box"} width="18"></iconify-icon>
        </div>
        <div class="relative flex-1">
          <${Icon} icon="lucide:search" className="absolute left-2.5 top-2.5 text-muted-foreground" />
          <${Input} className="pl-8" placeholder=${value ? value : "Buscar ícone (ex: database, mail, cloud)"} value=${q} onInput=${(e) => setQ(e.target.value)} />
        </div>
      </div>
      ${q.trim() && html`
        <div class="max-h-56 overflow-y-auto rounded-md border">
          ${results.length === 0
            ? html`<div class="p-2 text-xs text-muted-foreground">${loading ? "Buscando..." : "Nenhum ícone."}</div>`
            : results.map((name) => html`
                <button
                  type="button"
                  onClick=${() => select(name)}
                  class=${`flex w-full items-center gap-2 px-2 py-1.5 text-left text-sm hover:bg-accent ${value === name ? "bg-accent" : ""}`}>
                  <iconify-icon icon=${name} width="18" class="shrink-0"></iconify-icon>
                  <span class="truncate font-mono text-xs">${name}</span>
                  ${value === name && html`<${Icon} icon="lucide:check" className="ml-auto text-primary" />`}
                </button>
              `)}
        </div>`}
    </div>
  `;
}

const EMPTY = { key: "", name: "", description: "", icon: "lucide:box" };

export function DefinitionsListPage() {
  const { route } = useLocation();
  const { definitions } = useStore();
  const [form, setForm] = useState({ ...EMPTY });

  const canCreate = form.key.trim() && form.name.trim();
  function start() { setForm({ ...EMPTY }); open("new-def"); }
  function submit() {
    if (!canCreate) return;
    const id = addDefinition(form);
    setForm({ ...EMPTY });
    close("new-def");
    route(`/resource-definitions/${id}`);
  }

  return html`
    <div class="flex flex-col flex-1 w-full h-full overflow-y-auto p-6 space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight">Resource Definitions</h1>
          <p class="text-muted-foreground">Catálogo de tipos de recurso e seus campos.</p>
        </div>
        <${Button} onClick=${start}><${Icon} icon="lucide:plus" /> Nova definition<//>
      </div>

      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        ${definitions.map(
          (d) => html`
            <a onClick=${() => route(`/resource-definitions/${d.id}`)} class="cursor-pointer">
              <${Card} className="h-full transition-colors hover:border-primary/50">
                <${CardHeader}>
                  <div class="flex items-center justify-between">
                    <div class="flex size-9 items-center justify-center rounded-md bg-muted">
                      <iconify-icon icon=${d.icon || "lucide:box"} width="18"></iconify-icon>
                    </div>
                    <${Badge} variant=${d.status === "active" ? "default" : "secondary"}>${d.status}<//>
                  </div>
                  <${CardTitle} className="mt-2 text-base">${d.name}<//>
                  <${CardDescription}>${d.description || "Sem descrição."}<//>
                <//>
                <${CardContent}>
                  <code class="text-xs text-muted-foreground">${d.key}</code>
                <//>
                <${CardFooter} className="text-xs text-muted-foreground">
                  ${d.fields.length} campos · ${d.fields.filter((f) => f.is_secret).length} secret
                <//>
              <//>
            </a>
          `
        )}
      </div>

      <${Dialog} id="new-def">
        <${DialogContent}>
          <${DialogHeader}>
            <${DialogTitle}>Nova definition<//>
            <${DialogDescription}>Defina um tipo de recurso. Os campos são adicionados em seguida.<//>
          <//>
          <div class="space-y-4 py-2">
            <${Field}><${FieldLabel}>Key<//><${Input} placeholder="postgres" value=${form.key} onInput=${(e) => setForm({ ...form, key: e.target.value })} /><//>
            <${Field}><${FieldLabel}>Nome<//><${Input} placeholder="PostgreSQL Connection" value=${form.name} onInput=${(e) => setForm({ ...form, name: e.target.value })} /><//>
            <${Field}>
              <${FieldLabel}>Ícone<//>
              <${IconPicker} value=${form.icon} onPick=${(icon) => setForm({ ...form, icon })} />
            <//>
            <${Field}>
              <${FieldLabel}>Descrição<//>
              <${Textarea} placeholder="Conexão de banco por tenant" value=${form.description} onInput=${(e) => setForm({ ...form, description: e.target.value })} />
              <${FieldDescription}>A key é o identificador estável usado no consumo.<//>
            <//>
          </div>
          <${DialogFooter}>
            <${Button} variant="outline" onClick=${() => close("new-def")}>Cancelar<//>
            <${Button} disabled=${!canCreate} onClick=${submit}>Criar definition<//>
          <//>
        <//>
      <//>
    </div>
  `;
}
