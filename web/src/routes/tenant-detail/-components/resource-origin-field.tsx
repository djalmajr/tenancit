import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface ResourceOriginCandidate {
  id: string;
  label: string;
}

interface ResourceOriginFieldProps {
  candidates: ResourceOriginCandidate[];
  independentHint: string;
  independentLabel: string;
  label: string;
  linkedHint: string;
  onValueChange: (value: string) => void;
  value: string;
}

export function ResourceOriginField({
  candidates,
  independentHint,
  independentLabel,
  label,
  linkedHint,
  onValueChange,
  value,
}: ResourceOriginFieldProps) {
  if (candidates.length === 0) return null;

  return <div
    className="min-w-0 space-y-1.5"
    data-slot="resource-origin-field"
  >
    <label className="text-sm font-medium">{label}</label>
    <Select
      items={[
        { label: independentLabel, value: "independent" },
        ...candidates.map((candidate) => ({ label: candidate.label, value: candidate.id })),
      ]}
      value={value || "independent"}
      onValueChange={(nextValue) => onValueChange(nextValue === "independent" ? "" : String(nextValue))}
    >
      <SelectTrigger className="w-full">
        <SelectValue />
      </SelectTrigger>
      <SelectContent><SelectGroup>
        <SelectItem value="independent">{independentLabel}</SelectItem>
        {candidates.map((candidate) => <SelectItem key={candidate.id} value={candidate.id}>
          {candidate.label}
        </SelectItem>)}
      </SelectGroup></SelectContent>
    </Select>
    <p className="text-xs text-muted-foreground">{value ? linkedHint : independentHint}</p>
  </div>;
}
