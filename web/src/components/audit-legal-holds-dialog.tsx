import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Gavel } from "lucide-react";
import { useState } from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { api, getAdminSession } from "@/lib/api";
import { apiErrorMessage, useI18n } from "@/lib/i18n";

function localDateTime(value: Date) { return value.toISOString().slice(0, 16); }

export function AuditLegalHoldsDialog() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [{ from, to }, setWindow] = useState(() => {
    const end = new Date();
    return { from: localDateTime(new Date(end.getTime() - 7 * 24 * 60 * 60 * 1000)), to: localDateTime(end) };
  });
  const [reason, setReason] = useState("");
  const holdsQuery = useQuery({ queryKey: ["admin", "audit-legal-holds"], queryFn: ({ signal }) => api.listAuditLegalHolds(signal), enabled: open });
  const createMutation = useMutation({
    mutationFn: () => api.createAuditLegalHold({ from: new Date(from).toISOString(), to: new Date(to).toISOString(), reason: reason.trim() }),
    onSuccess: async () => { setReason(""); await queryClient.invalidateQueries({ queryKey: ["admin", "audit-legal-holds"] }); await queryClient.invalidateQueries({ queryKey: ["admin", "audit-health"] }); },
  });
  const releaseMutation = useMutation({
    mutationFn: api.releaseAuditLegalHold,
    onSuccess: async () => { await queryClient.invalidateQueries({ queryKey: ["admin", "audit-legal-holds"] }); await queryClient.invalidateQueries({ queryKey: ["admin", "audit-health"] }); },
  });
  const error = holdsQuery.error ?? createMutation.error ?? releaseMutation.error;
  const session = getAdminSession();
  if (session && !session.permissions.includes("audit.manage")) return null;
  return <Dialog onOpenChange={setOpen} open={open}>
    <DialogTrigger render={<Button variant="outline"><Gavel />{t("audit.manageHolds")}</Button>} />
    <DialogContent className="sm:max-w-2xl">
      <DialogHeader><DialogTitle>{t("audit.legalHolds")}</DialogTitle><DialogDescription>{t("audit.legalHoldDescription")}</DialogDescription></DialogHeader>
      {error && <Alert variant="destructive"><AlertDescription>{apiErrorMessage(error, t)}</AlertDescription></Alert>}
      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1.5"><label className="text-sm font-medium" htmlFor="audit-hold-from">{t("audit.holdFrom")}</label><Input id="audit-hold-from" onChange={(event) => setWindow((value) => ({ ...value, from: event.target.value }))} type="datetime-local" value={from} /></div>
        <div className="space-y-1.5"><label className="text-sm font-medium" htmlFor="audit-hold-to">{t("audit.holdTo")}</label><Input id="audit-hold-to" onChange={(event) => setWindow((value) => ({ ...value, to: event.target.value }))} type="datetime-local" value={to} /></div>
      </div>
      <div className="space-y-1.5"><label className="text-sm font-medium" htmlFor="audit-hold-reason">{t("audit.holdReason")}</label><Input id="audit-hold-reason" maxLength={500} onChange={(event) => setReason(event.target.value)} value={reason} /></div>
      <Button disabled={!reason.trim() || !from || !to || createMutation.isPending} onClick={() => createMutation.mutate()}>{t("audit.createHold")}</Button>
      <div className="max-h-64 space-y-2 overflow-y-auto">
        {holdsQuery.data?.items.map((hold) => <div className="flex items-start justify-between gap-4 rounded-lg border p-3" key={hold.id}>
          <div className="min-w-0"><p className="font-medium">{hold.reason}</p><p className="text-xs text-muted-foreground">{new Date(hold.from).toLocaleString()} — {new Date(hold.to).toLocaleString()}</p></div>
          {hold.released_at ? <span className="text-xs text-muted-foreground">{t("audit.holdReleased")}</span> : <Button disabled={releaseMutation.isPending} onClick={() => releaseMutation.mutate(hold.id)} size="sm" variant="outline">{t("audit.releaseHold")}</Button>}
        </div>)}
        {holdsQuery.data?.items.length === 0 && <p className="text-sm text-muted-foreground">{t("audit.noHolds")}</p>}
      </div>
      <DialogFooter><DialogClose render={<Button variant="outline">{t("common.cancel")}</Button>} /></DialogFooter>
    </DialogContent>
  </Dialog>;
}
