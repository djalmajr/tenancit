import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";

// Masks a secret value until the user clicks the reveal toggle (RN-06 on the
// admin surface). Extracted from tenant-detail for testability.
export function RevealValue({
  hideLabel = "Ocultar",
  showLabel = "Revelar",
  value,
}: {
  hideLabel?: string;
  showLabel?: string;
  value: string;
}) {
  const [shown, setShown] = useState(false);
  const label = shown ? hideLabel : showLabel;

  return (
    <div className="flex items-center gap-2">
      <code className="text-xs">{shown ? value : "••••••••••••"}</code>
      <button
        type="button"
        className="text-muted-foreground hover:text-foreground"
        onClick={() => setShown((s) => !s)}
        aria-label={label}
        title={label}
      >
        {shown ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
      </button>
    </div>
  );
}
