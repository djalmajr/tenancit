import type { Table as TanstackTable } from "@tanstack/react-table";
import { flexRender } from "@tanstack/react-table";
import type * as React from "react";
import { cn } from "@/lib/utils";
import { DataTablePagination, type DataTablePaginationLabels } from "./data-table-pagination";

export interface DataTableLabels extends DataTablePaginationLabels {
  noResults?: string;
}

interface DataTableProps<TData> extends React.ComponentProps<"div"> {
  labels?: DataTableLabels;
  onRowClick?: (row: TData) => void;
  table: TanstackTable<TData>;
  withPagination?: boolean;
}

function columnAlignment(meta: unknown) {
  const align = (meta as { align?: "center" | "left" | "right" } | undefined)?.align;
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
      <div className="overflow-hidden rounded-md border">
        <div className="overflow-auto">
          <table className="w-full caption-bottom text-sm">
            <thead className="bg-background">
              {table.getHeaderGroups().map((headerGroup) => (
                <tr key={headerGroup.id}>
                  {headerGroup.headers.map((header) => {
                    const size = header.column.columnDef.size;
                    const hasFixedSize = size !== undefined && size > 0;

                    return (
                      <th
                        className={cn(
                          "h-9 border-b border-border px-2 align-middle font-medium whitespace-nowrap text-muted-foreground",
                          columnAlignment(header.column.columnDef.meta),
                        )}
                        colSpan={header.colSpan}
                        key={header.id}
                        style={hasFixedSize ? { maxWidth: size, minWidth: size, width: size } : undefined}
                      >
                        {header.isPlaceholder
                          ? null
                          : flexRender(header.column.columnDef.header, header.getContext())}
                      </th>
                    );
                  })}
                </tr>
              ))}
            </thead>
            <tbody className="[&_tr:last-child]:border-0">
              {table.getRowModel().rows.length ? (
                table.getRowModel().rows.map((row) => (
                  <tr
                    className={cn(
                      "border-b transition-colors hover:bg-muted/50",
                      onRowClick && "cursor-pointer",
                    )}
                    key={row.id}
                    onClick={onRowClick ? () => onRowClick(row.original) : undefined}
                  >
                    {row.getVisibleCells().map((cell) => {
                      const size = cell.column.columnDef.size;
                      const hasFixedSize = size !== undefined && size > 0;

                      return (
                        <td
                          className={cn(
                            "px-2 py-2 align-middle whitespace-nowrap",
                            columnAlignment(cell.column.columnDef.meta),
                          )}
                          key={cell.id}
                          style={hasFixedSize ? { maxWidth: size, minWidth: size, width: size } : undefined}
                        >
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </td>
                      );
                    })}
                  </tr>
                ))
              ) : (
                <tr className="border-b">
                  <td className="h-24 px-2 py-2 text-center text-muted-foreground" colSpan={columns.length}>
                    {labels?.noResults ?? "No results."}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>
      {withPagination && <DataTablePagination labels={labels} table={table} />}
    </div>
  );
}
