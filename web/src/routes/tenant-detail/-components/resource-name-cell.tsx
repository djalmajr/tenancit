import { ResourceOriginIcon } from "./resource-origin-icon";

interface ResourceNameCellProps {
  label: string;
  linked: boolean;
  name: string;
}

export function ResourceNameCell({ label, linked, name }: ResourceNameCellProps) {
  return <span className="flex min-w-0 items-center gap-2 font-medium">
    <ResourceOriginIcon label={label} linked={linked} />
    <span className="min-w-0 flex-1 truncate" title={name}>{name}</span>
  </span>;
}
