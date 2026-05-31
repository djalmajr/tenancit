import { createRoute, Link } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import { Boxes, ChevronRight, Plus, Check, Trash2, Power, PowerOff } from "lucide-react";
import { Route as rootRoute } from "./__root";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Table, THead, TBody, TR, TH, TD } from "@/components/ui/table";
import { Select, SelectItem } from "@/components/ui/select";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter, DialogClose,
} from "@/components/ui/dialog";
import { api, type DefinitionDetail } from "@/lib/api";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/resource-definitions/$id",
  component: DefinitionDetailPage,
});

const EMPTY = { key: "", label: "", dataType: "string", required: false, isSecret: false };

function DefinitionDetailPage() {
  const { id } = Route.useParams();
  const [detail, setDetail] = useState<DefinitionDetail | null>(null);
  const [open, setOpen] = useState(false);
  const [f, setF] = useState({ ...EMPTY });

  const load = useCallback(() => {
    api.getDefinition(id).then(setDetail).catch(() => setDetail(null));
  }, [id]);
  useEffect(() => load(), [load]);

  async function addField() {
    if (!f.key.trim()) return;
    await api.addField(id, f).catch(() => {});
    setF({ ...EMPTY });
    setOpen(false);
    load();
  }
  async function removeField(fieldId: string) {
    await api.deleteField(id, fieldId).catch(() => {});
    load();
  }
  async function toggleStatus() {
    if (!detail) return;
    const next = detail.definition.status === "active" ? "inactive" : "active";
    await api.setDefinitionStatus(id, next).catch(() => {});
    load();
  }

  if (!detail) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16">
        <p className="text-muted-foreground">Definition não encontrada.</p>
        <Link to="/resource-definitions"><Button variant="outline">Voltar</Button></Link>
      </div>
    );
  }

  const d = detail.definition;
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link to="/resource-definitions" className="hover:text-foreground">Resource Definitions</Link>
        <ChevronRight className="size-3.5" />
        <span className="text-foreground">{d.name}</span>
      </div>

      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <div className="flex size-11 items-center justify-center rounded-lg bg-muted">
            <Boxes className="size-5" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">{d.name}</h1>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <code>{d.key}</code>
              <Badge variant={d.status === "active" ? "default" : "secondary"}>{d.status}</Badge>
            </div>
          </div>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={toggleStatus}>
            {d.status === "active" ? <PowerOff className="size-4" /> : <Power className="size-4" />}
            {d.status === "active" ? "Desativar" : "Ativar"}
          </Button>
          <Button onClick={() => { setF({ ...EMPTY }); setOpen(true); }}>
            <Plus className="size-4" /> Novo campo
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Campos</CardTitle>
          <p className="text-sm text-muted-foreground">Definição dinâmica dos campos deste tipo de recurso.</p>
        </CardHeader>
        <CardContent>
          {detail.fields.length === 0 ? (
            <p className="text-sm text-muted-foreground">Nenhum campo. Adicione os campos que compõem este recurso.</p>
          ) : (
            <Table>
              <THead>
                <TR>
                  <TH>Chave</TH>
                  <TH>Label</TH>
                  <TH>Tipo</TH>
                  <TH>Required</TH>
                  <TH>Secret</TH>
                  <TH className="w-16 text-right">Ações</TH>
                </TR>
              </THead>
              <TBody>
                {detail.fields.map((field) => (
                  <TR key={field.id}>
                    <TD><code className="text-xs">{field.key}</code></TD>
                    <TD>{field.label || "—"}</TD>
                    <TD><code className="text-xs text-muted-foreground">{field.data_type}</code></TD>
                    <TD>{field.required ? <Check className="size-4 text-green-600" /> : <span className="text-muted-foreground">—</span>}</TD>
                    <TD>{field.is_secret ? <Check className="size-4 text-amber-600" /> : <span className="text-muted-foreground">—</span>}</TD>
                    <TD className="text-right">
                      <Button variant="ghost" size="icon-sm" title="Remover" onClick={() => removeField(field.id)}>
                        <Trash2 className="size-4" />
                      </Button>
                    </TD>
                  </TR>
                ))}
              </TBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Novo campo</DialogTitle>
            <DialogDescription>Adicione um campo a este tipo de recurso.</DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Chave</label>
                <Input placeholder="host" value={f.key} onChange={(e) => setF({ ...f, key: e.target.value })} />
              </div>
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Label</label>
                <Input placeholder="Host" value={f.label} onChange={(e) => setF({ ...f, label: e.target.value })} />
              </div>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium">Tipo</label>
              <Select value={f.dataType} onChange={(e) => setF({ ...f, dataType: e.target.value })}>
                <SelectItem value="string">string</SelectItem>
                <SelectItem value="int">int</SelectItem>
                <SelectItem value="bool">bool</SelectItem>
              </Select>
            </div>
            <label className="flex items-center justify-between rounded-md border p-3">
              <div>
                <div className="text-sm font-medium">Obrigatório</div>
                <div className="text-xs text-muted-foreground">Exigido na criação do recurso.</div>
              </div>
              <input type="checkbox" checked={f.required} onChange={(e) => setF({ ...f, required: e.target.checked })} />
            </label>
            <label className="flex items-center justify-between rounded-md border p-3">
              <div>
                <div className="text-sm font-medium">Secret</div>
                <div className="text-xs text-muted-foreground">Criptografado com AES-256-GCM.</div>
              </div>
              <input type="checkbox" checked={f.isSecret} onChange={(e) => setF({ ...f, isSecret: e.target.checked })} />
            </label>
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline">Cancelar</Button>} />
            <Button disabled={!f.key.trim()} onClick={addField}>Adicionar campo</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
