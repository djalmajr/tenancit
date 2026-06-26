import { CheckCircle2, X } from "lucide-react";
import { Button } from "@/components/ui/button";

export function StatusNotice({
  dismissLabel,
  message,
  onDismiss,
}: {
  dismissLabel: string;
  message: string;
  onDismiss: () => void;
}) {
  if (!message) return null;

  return (
    <div className="flex items-start justify-between gap-3 rounded-md border border-primary/20 bg-primary/10 p-3 text-sm text-primary" role="status">
      <div className="flex items-start gap-2">
        <CheckCircle2 className="mt-0.5 size-4 shrink-0" />
        <span>{message}</span>
      </div>
      <Button aria-label={dismissLabel} className="-my-1 -mr-1" onClick={onDismiss} size="icon-sm" variant="ghost">
        <X className="size-4" />
      </Button>
    </div>
  );
}
