import { html } from "htm/preact";
import { useLocation } from "preact-iso";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "~/components/ui/card.js";
import { Badge } from "~/components/ui/badge.js";
import { Button } from "~/components/ui/button.js";
import { Icon } from "~/components/ui/icon.js";
import { useStore, resetState } from "~/routes/store.js";

function Stat({ icon, label, value, hint }) {
  return html`
    <${Card}>
      <${CardHeader} className="flex flex-row items-center justify-between space-y-0 pb-2">
        <${CardTitle} className="text-sm font-medium text-muted-foreground">${label}<//>
        <${Icon} icon=${icon} className="text-muted-foreground" />
      <//>
      <${CardContent}>
        <div class="text-2xl font-bold">${value}</div>
        <p class="text-xs text-muted-foreground">${hint}</p>
      <//>
    <//>
  `;
}

export function OverviewPage() {
  const { route } = useLocation();
  const { tenants, definitions } = useStore();
  const activeTenants = tenants.filter((t) => t.status === "active").length;
  const totalResources = tenants.reduce((acc, t) => acc + t.resources.length, 0);
  const activeDefs = definitions.filter((d) => d.status === "active").length;
  const totalDomains = tenants.reduce((acc, t) => acc + t.domains.length, 0);

  return html`
    <div class="flex flex-col flex-1 w-full h-full overflow-y-auto p-6 space-y-6">
      <div class="flex items-start justify-between">
        <div>
          <h1 class="text-2xl font-semibold tracking-tight">Visão geral</h1>
          <p class="text-muted-foreground">Configuração multi-tenant de recursos por cliente.</p>
        </div>
        <${Button} variant="outline" size="sm" onClick=${() => resetState()} title="Restaura os dados de exemplo">
          <${Icon} icon="lucide:rotate-ccw" /> Resetar dados
        <//>
      </div>

      <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <${Stat} icon="lucide:building-2" label="Tenants ativos" value=${activeTenants} hint=${`${tenants.length} no total`} />
        <${Stat} icon="lucide:globe" label="Domínios" value=${totalDomains} hint="hostnames mapeados" />
        <${Stat} icon="lucide:database" label="Recursos provisionados" value=${totalResources} hint="instâncias" />
        <${Stat} icon="lucide:boxes" label="Definitions ativas" value=${activeDefs} hint="tipos de recurso" />
      </div>

      <${Card}>
        <${CardHeader}>
          <${CardTitle}>Tenants<//>
          <${CardDescription}>Atalho para os clientes cadastrados.<//>
        <//>
        <${CardContent} className="space-y-2">
          ${tenants.length === 0
            ? html`<p class="text-sm text-muted-foreground">Nenhum tenant ainda.</p>`
            : tenants.map(
                (t) => html`
                  <a
                    href=${`/tenants/${t.id}`}
                    onClick=${(e) => { e.preventDefault(); route(`/tenants/${t.id}`); }}
                    class="flex items-center justify-between rounded-md border p-3 hover:bg-accent transition-colors cursor-pointer"
                  >
                    <div class="flex items-center gap-3">
                      <div class="flex size-9 items-center justify-center rounded-md bg-muted">
                        <${Icon} icon="lucide:building-2" />
                      </div>
                      <div>
                        <div class="font-medium">${t.name}</div>
                        <div class="text-xs text-muted-foreground">${t.domains[0]?.hostname || "sem domínio"}</div>
                      </div>
                    </div>
                    <div class="flex items-center gap-3">
                      <span class="text-xs text-muted-foreground">${t.resources.length} recursos</span>
                      <${Badge} variant=${t.status === "active" ? "default" : "secondary"}>${t.status}<//>
                    </div>
                  </a>
                `
              )}
        <//>
      <//>
    </div>
  `;
}
