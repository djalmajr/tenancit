import type { Table } from "@tanstack/react-table";
import { ArrowDownAZ, ArrowDownUp, ArrowUpAZ, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Command, CommandGroup, CommandItem, CommandList } from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { useI18n } from "@/lib/i18n";

export function DataTableSortList<TData>({ table }: { table: Table<TData> }) {
  const { t } = useI18n();
  const sorting = table.getState().sorting;
  const columns = table.getAllLeafColumns().filter((column) => column.getCanSort());

  if (columns.length === 0) return null;

  return (
    <Popover>
      <PopoverTrigger render={<Button className="h-8 font-normal" variant="outline" />}>
        <ArrowDownUp className="text-muted-foreground" />
        {t("dataTable.sort")}
        {sorting.length > 0 && (
          <span className="flex size-5 items-center justify-center rounded bg-muted text-xs tabular-nums">
            {sorting.length}
          </span>
        )}
      </PopoverTrigger>
      <PopoverContent align="end" className="w-64 p-0">
        <Command>
          <CommandList>
            <CommandGroup heading={t("dataTable.sortBy")}>
              {columns.map((column) => {
                const direction = column.getIsSorted();
                const Icon = direction === "asc" ? ArrowUpAZ : direction === "desc" ? ArrowDownAZ : ArrowDownUp;
                return (
                  <CommandItem
                    key={column.id}
                    onSelect={() => {
                      if (!direction) column.toggleSorting(false, sorting.length > 0);
                      else if (direction === "asc") column.toggleSorting(true, true);
                      else column.clearSorting();
                    }}
                  >
                    <Icon className="text-muted-foreground" />
                    <span className="min-w-0 flex-1 truncate">{column.columnDef.meta?.label ?? column.id}</span>
                    {direction && <span className="text-xs text-muted-foreground">{t(direction === "asc" ? "dataTable.ascending" : "dataTable.descending")}</span>}
                  </CommandItem>
                );
              })}
              {sorting.length > 0 && (
                <CommandItem className="justify-center text-muted-foreground" onSelect={() => table.resetSorting()}>
                  <X />{t("dataTable.sortReset")}
                </CommandItem>
              )}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  );
}
