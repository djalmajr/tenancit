import * as React from "react";
import { CheckCircle2, X } from "lucide-react";
import { Button } from "@/components/ui/button";

export function Toast({
  dismissLabel,
  message,
  onDismiss,
}: {
  dismissLabel: string;
  message: string;
  onDismiss: () => void;
}) {
  React.useEffect(() => {
    if (!message) return;
    const timeout = window.setTimeout(onDismiss, 4000);
    return () => window.clearTimeout(timeout);
  }, [message, onDismiss]);

  if (!message) return null;

  return (
    <div className="fixed right-4 top-4 z-[70] w-[min(24rem,calc(100vw-2rem))] rounded-md border bg-popover p-3 text-sm text-popover-foreground shadow-md" role="status">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-2">
          <CheckCircle2 className="mt-0.5 size-4 shrink-0 text-primary" />
          <span>{message}</span>
        </div>
        <Button aria-label={dismissLabel} className="-my-1 -mr-1" onClick={onDismiss} size="icon-sm" variant="ghost">
          <X className="size-4" />
        </Button>
      </div>
    </div>
  );
}
