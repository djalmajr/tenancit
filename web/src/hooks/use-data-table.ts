import {
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type ColumnDef,
  type GlobalFilterTableState,
  type PaginationState,
  type SortingState,
  useReactTable,
} from "@tanstack/react-table";
import * as React from "react";
import { PAGE_SIZE_OPTIONS } from "@/components/data-table/data-table-pagination";

interface UseDataTableProps<TData> {
  columns: ColumnDef<TData>[];
  data: TData[];
  globalFilterFn?: (row: TData, filterValue: string) => boolean;
  initialState?: {
    globalFilter?: GlobalFilterTableState["globalFilter"];
    pagination?: PaginationState;
    sorting?: SortingState;
  };
}

export function useDataTable<TData>({
  columns,
  data,
  globalFilterFn,
  initialState,
}: UseDataTableProps<TData>) {
  const [globalFilter, setGlobalFilter] = React.useState(initialState?.globalFilter ?? "");
  const [pagination, setPagination] = React.useState<PaginationState>(
    initialState?.pagination ?? {
      pageIndex: 0,
      pageSize: PAGE_SIZE_OPTIONS[0] ?? 5,
    },
  );
  const [sorting, setSorting] = React.useState<SortingState>(initialState?.sorting ?? []);

  const table = useReactTable({
    columns,
    data,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    globalFilterFn: (row, _columnId, filterValue) => {
      if (globalFilterFn) return globalFilterFn(row.original, String(filterValue ?? ""));
      const searchable = Object.values(row.original as Record<string, unknown>)
        .map((value) => String(value ?? ""))
        .join(" ")
        .toLowerCase();
      return searchable.includes(String(filterValue ?? "").toLowerCase());
    },
    onGlobalFilterChange: setGlobalFilter,
    onPaginationChange: setPagination,
    onSortingChange: setSorting,
    state: {
      globalFilter,
      pagination,
      sorting,
    },
  });

  return { globalFilter, setGlobalFilter, table };
}
