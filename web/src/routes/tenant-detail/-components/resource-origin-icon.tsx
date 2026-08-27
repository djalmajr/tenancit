import { Box, Link2 } from "lucide-react";

interface ResourceOriginIconProps {
  label: string;
  linked: boolean;
}

export function ResourceOriginIcon({ label, linked }: ResourceOriginIconProps) {
  const Icon = linked ? Link2 : Box;

  return <Icon aria-label={label} className="size-5 shrink-0 stroke-[2.25]" />;
}
