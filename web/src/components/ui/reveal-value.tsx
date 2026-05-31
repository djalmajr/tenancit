import { useState } from "react";
import { Eye, EyeOff } from "lucide-react";

// Masks a secret value until the user clicks the reveal toggle (RN-06 on the
// admin surface). Extracted from tenant-detail for testability.
export function RevealValue({ value }: { value: string }) {
  const [shown, setShown] = useState(false);
  return (
    <div className="flex items-center gap-2">
      <code className="text-xs">{shown ? value : "••••••••••••"}</code>
      <button
        type="button"
        className="text-muted-foreground hover:text-foreground"
        onClick={() => setShown((s) => !s)}
        aria-label={shown ? "Ocultar" : "Revelar"}
        title={shown ? "Ocultar" : "Revelar"}
      >
        {shown ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
      </button>
    </div>
  );
}
