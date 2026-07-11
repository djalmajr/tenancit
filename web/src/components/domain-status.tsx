import { Status, StatusIndicator, StatusLabel } from "@/components/ui/status";

function statusVariant(value: string): "default" | "error" | "success" | "warning" {
  if (value === "active" || value === "success") return "success";
  if (value === "revoked" || value === "error" || value === "denied") return "error";
  if (value === "expired" || value === "rate_limited") return "warning";
  return "default";
}

export function DomainStatus({ label, value }: { label: string; value: string }) {
  return <Status variant={statusVariant(value)}><StatusIndicator /><StatusLabel>{label}</StatusLabel></Status>;
}
