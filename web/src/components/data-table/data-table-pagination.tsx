import type { Table } from "@tanstack/react-table";
import { ChevronLeft, ChevronRight, ChevronsLeft, ChevronsRight } from "lucide-react";
import * as React from "react";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";

export interface DataTablePaginationLabels {
  goToFirstPage?: string;
  goToLastPage?: string;
  goToNextPage?: string;
  goToPreviousPage?: string;
  item?: string;
  items?: string;
  page?: string;
  pageOf?: string;
  rowsPerPage?: string;
}

const defaultLabels: Required<DataTablePaginationLabels> = {
  goToFirstPage: "Go to first page",
  goToLastPage: "Go to last page",
  goToNextPage: "Go to next page",
  goToPreviousPage: "Go to previous page",
  item: "item",
  items: "items",
  page: "Page",
  pageOf: "of",
  rowsPerPage: "Rows per page",
};

interface DataTablePaginationProps<TData> extends React.ComponentProps<"div"> {
  labels?: DataTablePaginationLabels;
  pageSizeOptions?: number[];
  table: Table<TData>;
}

export const PAGE_SIZE_OPTIONS = [5, 10, 25, 50];

export function DataTablePagination<TData>({
  className,
  labels: labelsProp,
  pageSizeOptions = PAGE_SIZE_OPTIONS,
  table,
  ...props
}: DataTablePaginationProps<TData>) {
  const labels = { ...defaultLabels, ...labelsProp };
  const pageCount = table.getPageCount();
  const currentPage = pageCount === 0 ? 0 : table.getState().pagination.pageIndex + 1;

  return (
    <div
      className={cn(
        "flex w-full flex-col-reverse items-center justify-between gap-3 overflow-auto p-1 sm:flex-row",
        className,
      )}
      {...props}
    >
      <div className="flex-1 text-sm text-muted-foreground">
        {table.getFilteredRowModel().rows.length}{" "}
        {table.getFilteredRowModel().rows.length === 1 ? labels.item : labels.items}
      </div>
      <div className="flex flex-col-reverse items-center gap-3 sm:flex-row sm:gap-5">
        <div className="flex items-center justify-center text-sm font-medium">
          {labels.page} {currentPage} {labels.pageOf} {pageCount}
        </div>
        <div className="flex items-center gap-2">
          <Button
            aria-label={labels.goToFirstPage}
            className="hidden lg:flex"
            disabled={!table.getCanPreviousPage()}
            onClick={() => table.setPageIndex(0)}
            size="icon"
            type="button"
            variant="outline"
          >
            <ChevronsLeft />
          </Button>
          <Button
            aria-label={labels.goToPreviousPage}
            disabled={!table.getCanPreviousPage()}
            onClick={() => table.previousPage()}
            size="icon"
            type="button"
            variant="outline"
          >
            <ChevronLeft />
          </Button>
          <Button
            aria-label={labels.goToNextPage}
            disabled={!table.getCanNextPage()}
            onClick={() => table.nextPage()}
            size="icon"
            type="button"
            variant="outline"
          >
            <ChevronRight />
          </Button>
          <Button
            aria-label={labels.goToLastPage}
            className="hidden lg:flex"
            disabled={!table.getCanNextPage()}
            onClick={() => table.setPageIndex(pageCount - 1)}
            size="icon"
            type="button"
            variant="outline"
          >
            <ChevronsRight />
          </Button>
        </div>
        <Select
          items={pageSizeOptions.map((pageSize) => ({ label: String(pageSize), value: String(pageSize) }))}
          onValueChange={(value) => table.setPageSize(Number(value))}
          value={`${table.getState().pagination.pageSize}`}
        >
          <SelectTrigger aria-label={labels.rowsPerPage} className="w-16">
            <SelectValue placeholder={table.getState().pagination.pageSize} />
          </SelectTrigger>
          <SelectContent side="top">
            <SelectGroup>
              {pageSizeOptions.map((pageSize) => (
                <SelectItem key={pageSize} value={`${pageSize}`}>
                  {pageSize}
                </SelectItem>
              ))}
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>
    </div>
  );
}
