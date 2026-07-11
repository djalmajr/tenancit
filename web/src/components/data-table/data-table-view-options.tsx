import type { Table } from "@tanstack/react-table";
import { Settings2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Command, CommandGroup, CommandItem, CommandList } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

export function DataTableViewOptions<TData>({ label, table }: { label: string; table: Table<TData> }) {
  const columns = table.getAllColumns().filter((column) => typeof column.accessorFn !== "undefined" && column.getCanHide());
  return <Popover>
    <PopoverTrigger render={<Button aria-label={label} className="ml-auto h-8 font-normal" role="combobox" variant="outline" />}>
      <Settings2 className="text-muted-foreground" />{label}
    </PopoverTrigger>
    <PopoverContent align="end" className="w-48 p-0">
      <Command><CommandList><CommandGroup>
        {columns.map((column) => <CommandItem data-checked={column.getIsVisible()} key={column.id} onSelect={() => column.toggleVisibility(!column.getIsVisible())}>
          <span className="truncate">{column.columnDef.meta?.label ?? column.id}</span>
        </CommandItem>)}
      </CommandGroup></CommandList></Command>
    </PopoverContent>
  </Popover>;
}
