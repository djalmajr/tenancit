import {
  getCoreRowModel,
  getFilteredRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  type ColumnDef,
  type ColumnFiltersState,
  type VisibilityState,
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
    globalFilter?: string;
    pagination?: PaginationState;
    sorting?: SortingState;
    columnVisibility?: VisibilityState;
  };
  visibilityStorageKey?: string;
}

function isGlobalFilterUpdater(value: unknown): value is (current: string) => unknown {
  return typeof value === "function";
}

export function useDataTable<TData>({
  columns,
  data,
  globalFilterFn,
  initialState,
  visibilityStorageKey,
}: UseDataTableProps<TData>) {
  const [globalFilter, setGlobalFilter] = React.useState(initialState?.globalFilter ?? "");
  const [pagination, setPagination] = React.useState<PaginationState>(
    initialState?.pagination ?? {
      pageIndex: 0,
      pageSize: PAGE_SIZE_OPTIONS[0] ?? 5,
    },
  );
  const [sorting, setSorting] = React.useState<SortingState>(initialState?.sorting ?? []);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>(() => {
    if (!visibilityStorageKey || typeof window === "undefined") return initialState?.columnVisibility ?? {};
    try {
      const stored = window.localStorage.getItem(visibilityStorageKey);
      return stored ? { ...initialState?.columnVisibility, ...JSON.parse(stored) as VisibilityState } : initialState?.columnVisibility ?? {};
    } catch {
      return {};
    }
  });

  React.useEffect(() => {
    if (visibilityStorageKey) window.localStorage.setItem(visibilityStorageKey, JSON.stringify(columnVisibility));
  }, [columnVisibility, visibilityStorageKey]);

  const table = useReactTable({
    columns,
    data,
    getCoreRowModel: getCoreRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    getSortedRowModel: getSortedRowModel(),
    globalFilterFn: (row, _columnId, filterValue) => {
      const normalizedFilter = typeof filterValue === "string" ? filterValue : "";
      if (globalFilterFn) return globalFilterFn(row.original, normalizedFilter);
      const searchable = Object.values(row.original as Record<string, unknown>)
        .map((value) => {
          if (typeof value === "string") return value;
          if (typeof value === "number" || typeof value === "boolean" || typeof value === "bigint") {
            return String(value);
          }
          return "";
        })
        .join(" ")
        .toLowerCase();
      return searchable.includes(normalizedFilter.toLowerCase());
    },
    onGlobalFilterChange: (unsafeUpdater) => {
      const updater: unknown = unsafeUpdater;
      setGlobalFilter((current) => {
        const next = isGlobalFilterUpdater(updater) ? updater(current) : updater;
        return typeof next === "string" ? next : "";
      });
    },
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onPaginationChange: setPagination,
    onSortingChange: setSorting,
    state: {
      globalFilter,
      columnFilters,
      columnVisibility,
      pagination,
      sorting,
    },
  });

  return { globalFilter, setGlobalFilter, table };
}
