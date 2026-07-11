import type { Column } from "@tanstack/react-table";
import { Check, PlusCircle, XCircle } from "lucide-react";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList, CommandSeparator } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { cn } from "@/lib/utils";

export interface FacetOption { label: string; value: string }

export function DataTableFacetedFilter<TData>({
  clearLabel,
  column,
  emptyLabel,
  multiple = true,
  options,
  title,
}: {
  clearLabel: string;
  column: Column<TData>;
  emptyLabel: string;
  multiple?: boolean;
  options: FacetOption[];
  title: string;
}) {
  const [open, setOpen] = useState(false);
  const filterValue = column.getFilterValue();
  const selected = new Set(Array.isArray(filterValue) ? filterValue as string[] : []);
  const reset = () => column.setFilterValue(undefined);

  return <Popover open={open} onOpenChange={setOpen}>
    <PopoverTrigger render={<Button className="border-dashed font-normal" variant="outline" />}>
      {selected.size ? <XCircle onClick={(event) => { event.stopPropagation(); reset(); }} /> : <PlusCircle />}
      {title}
      {selected.size > 0 && <><Separator className="mx-0.5 data-[orientation=vertical]:h-4" orientation="vertical" />
        <Badge className="rounded-sm px-1 font-normal lg:hidden" variant="secondary">{selected.size}</Badge>
        <div className="hidden gap-1 lg:flex">{selected.size > 2 ? <Badge className="rounded-sm px-1 font-normal" variant="secondary">{selected.size}</Badge> : options.filter((option) => selected.has(option.value)).map((option) => <Badge className="rounded-sm px-1 font-normal" key={option.value} variant="secondary">{option.label}</Badge>)}</div>
      </>}
    </PopoverTrigger>
    <PopoverContent align="start" className="w-56 p-0">
      <Command><CommandInput placeholder={title} /><CommandList><CommandEmpty>{emptyLabel}</CommandEmpty><CommandGroup>
        {options.map((option) => { const active = selected.has(option.value); return <CommandItem key={option.value} onSelect={() => {
          const next = new Set(selected);
          if (multiple) {
            if (active) next.delete(option.value);
            else next.add(option.value);
          }
          else { next.clear(); if (!active) next.add(option.value); setOpen(false); }
          column.setFilterValue(next.size ? [...next] : undefined);
        }}><span className={cn("flex size-4 items-center justify-center rounded-sm border border-primary", active ? "bg-primary text-primary-foreground" : "opacity-50 [&_svg]:invisible")}><Check /></span>{option.label}</CommandItem>; })}
      </CommandGroup>{selected.size > 0 && <><CommandSeparator /><CommandGroup><CommandItem className="justify-center" onSelect={reset}>{clearLabel}</CommandItem></CommandGroup></>}</CommandList></Command>
    </PopoverContent>
  </Popover>;
}
