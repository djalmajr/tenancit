import { ChevronsUpDown } from "lucide-react";
import * as React from "react";
import { Button } from "@/components/ui/button";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { cn } from "@/lib/utils";

export type ComboboxOption = { value: string; label: string; keywords?: string; disabled?: boolean };

type ComboboxProps = {
  options: ComboboxOption[];
  value: string;
  onValueChange: (value: string) => void;
  searchable?: boolean;
  searchPlaceholder?: string;
  emptyText?: string;
  placeholder?: string;
  disabled?: boolean;
  className?: string;
  triggerClassName?: string;
  "aria-label"?: string;
};

/** Popover + Command combobox adapted from the reference implementation. */
export function Combobox({
  options, value, onValueChange, searchable = false, searchPlaceholder,
  emptyText = "—", placeholder = "…", disabled, className, triggerClassName,
  "aria-label": ariaLabel,
}: ComboboxProps) {
  const [open, setOpen] = React.useState(false);
  const selected = options.find((option) => option.value === value);
  return (
    <div className={cn("inline-flex", className)}>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger render={<Button aria-expanded={open} aria-label={ariaLabel} className={cn("h-9 w-full min-w-0 justify-between border-input font-normal", !selected && "text-muted-foreground", triggerClassName)} disabled={disabled} role="combobox" type="button" variant="outline" />}>
          <span className="truncate">{selected?.label ?? placeholder}</span>
          <ChevronsUpDown className="shrink-0 opacity-50" data-icon="inline-end" />
        </PopoverTrigger>
        <PopoverContent align="start" className="w-(--anchor-width)! min-w-(--anchor-width)! max-w-(--anchor-width)! p-0">
          <Command>
            {searchable && <CommandInput placeholder={searchPlaceholder} />}
            <CommandList>
              {searchable && <CommandEmpty>{emptyText}</CommandEmpty>}
              <CommandGroup>
                {options.map((option) => (
                  <CommandItem key={option.value} data-checked={value === option.value || undefined} disabled={option.disabled} value={`${option.label} ${option.keywords ?? option.value}`} onSelect={() => { onValueChange(option.value); setOpen(false); }}>
                    <span className="truncate">{option.label}</span>
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
}
