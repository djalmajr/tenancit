import type { Table as TanstackTable } from "@tanstack/react-table";
import { flexRender } from "@tanstack/react-table";
import type * as React from "react";
import { cn } from "@/lib/utils";
import { DataTablePagination, type DataTablePaginationLabels } from "./data-table-pagination";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export interface DataTableLabels extends DataTablePaginationLabels {
  noResults?: string;
}

interface DataTableProps<TData> extends React.ComponentProps<"div"> {
  labels?: DataTableLabels;
  onRowClick?: (row: TData) => void;
  table: TanstackTable<TData>;
  withPagination?: boolean;
}

function columnAlignment(meta: { align?: "center" | "left" | "right" } | undefined) {
  const align = meta?.align;
  if (align === "right") return "text-right";
  if (align === "center") return "text-center";
  return "text-left";
}

export function DataTable<TData>({
  children,
  className,
  labels,
  onRowClick,
  table,
  withPagination = true,
  ...props
}: DataTableProps<TData>) {
  const columns = table.getAllColumns();

  return (
    <div className={cn("flex w-full flex-col gap-2.5 overflow-hidden", className)} {...props}>
      {children}
      <div className="overflow-x-auto rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => {
                  const size = header.column.columnDef.size;
                  const hasFixedSize = size !== undefined && size > 0;

                  return (
                    <TableHead
                      className={columnAlignment(header.column.columnDef.meta)}
                      colSpan={header.colSpan}
                      key={header.id}
                      style={hasFixedSize ? { maxWidth: size, minWidth: size, width: size } : undefined}
                    >
                      {header.isPlaceholder
                        ? null
                        : flexRender(header.column.columnDef.header, header.getContext())}
                    </TableHead>
                  );
                })}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow
                  className={cn(onRowClick && "cursor-pointer")}
                  data-state={row.getIsSelected() && "selected"}
                  key={row.id}
                  onClick={onRowClick ? () => onRowClick(row.original) : undefined}
                >
                  {row.getVisibleCells().map((cell) => {
                    const size = cell.column.columnDef.size;
                    const hasFixedSize = size !== undefined && size > 0;

                    return (
                      <TableCell
                        className={cn(
                          "max-w-96 truncate",
                          columnAlignment(cell.column.columnDef.meta),
                        )}
                        key={cell.id}
                        style={hasFixedSize ? { maxWidth: size, minWidth: size, width: size } : undefined}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </TableCell>
                    );
                  })}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell className="h-24 text-center text-muted-foreground" colSpan={columns.length}>
                  {labels?.noResults ?? "No results."}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      {withPagination && <DataTablePagination labels={labels} table={table} />}
    </div>
  );
}
