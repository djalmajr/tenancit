import type { Table } from "@tanstack/react-table";
import { X } from "lucide-react";
import type { ReactNode } from "react";
import { DataTableFacetedFilter, type FacetOption } from "./data-table-faceted-filter";
import { DataTableSortList } from "./data-table-sort-list";
import { DataTableViewOptions } from "./data-table-view-options";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function DataTableToolbar<TData>({
  children,
  clearLabel,
  columnsLabel,
  emptyLabel,
  facets = [],
  resetLabel,
  searchLabel,
  table,
  trailing,
}: {
  children?: ReactNode;
  clearLabel: string;
  columnsLabel: string;
  emptyLabel: string;
  facets?: Array<{ columnId: string; multiple?: boolean; options: FacetOption[]; title: string }>;
  resetLabel: string;
  searchLabel: string;
  table: Table<TData>;
  trailing?: ReactNode;
}) {
  const filtered = table.getState().columnFilters.length > 0 || Boolean(table.getState().globalFilter);
  return <div className="flex w-full flex-wrap items-center gap-2 p-1" role="toolbar">
    <Input className="h-8 w-full sm:w-64" onChange={(event) => table.setGlobalFilter(event.target.value)} placeholder={searchLabel} value={(table.getState().globalFilter as string) ?? ""} />
    {facets.map((facet) => { const column = table.getColumn(facet.columnId); return column ? <DataTableFacetedFilter clearLabel={clearLabel} column={column} emptyLabel={emptyLabel} key={facet.columnId} multiple={facet.multiple} options={facet.options} title={facet.title} /> : null; })}
    {filtered && <Button className="border-dashed" onClick={() => { table.resetColumnFilters(); table.setGlobalFilter(""); }} variant="outline"><X />{clearLabel}</Button>}
    <div className="ml-auto flex items-center gap-2">{children}<DataTableSortList table={table} /><DataTableViewOptions label={columnsLabel} resetLabel={resetLabel} table={table} />{trailing}</div>
  </div>;
}
