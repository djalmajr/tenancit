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

type StoredPreferences = {
  columnVisibility?: VisibilityState;
  pageSize?: number;
  sorting?: SortingState;
  version: 1;
};

function loadPreferences(key?: string): Partial<StoredPreferences> {
  if (!key || typeof window === "undefined") return {};
  try {
    const parsed = JSON.parse(window.localStorage.getItem(key) ?? "null") as StoredPreferences | VisibilityState | null;
    if (!parsed) return {};
    if ("version" in parsed && parsed.version === 1) return parsed;
    return { columnVisibility: parsed as VisibilityState };
  } catch {
    return {};
  }
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
  const preferences = React.useMemo(() => loadPreferences(visibilityStorageKey), [visibilityStorageKey]);
  const [globalFilter, setGlobalFilter] = React.useState(initialState?.globalFilter ?? "");
  const [pagination, setPagination] = React.useState<PaginationState>(
    initialState?.pagination ? {
      ...initialState.pagination,
      pageSize: preferences.pageSize ?? initialState.pagination.pageSize,
    } : {
      pageIndex: 0,
      pageSize: preferences.pageSize ?? PAGE_SIZE_OPTIONS[0] ?? 5,
    },
  );
  const [sorting, setSorting] = React.useState<SortingState>(preferences.sorting ?? initialState?.sorting ?? []);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>(() => {
    return { ...initialState?.columnVisibility, ...preferences.columnVisibility };
  });

  React.useEffect(() => {
    if (visibilityStorageKey) window.localStorage.setItem(visibilityStorageKey, JSON.stringify({
      columnVisibility, pageSize: pagination.pageSize, sorting, version: 1,
    } satisfies StoredPreferences));
  }, [columnVisibility, pagination.pageSize, sorting, visibilityStorageKey]);

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
    initialState: {
      columnVisibility: initialState?.columnVisibility ?? {},
      globalFilter: initialState?.globalFilter ?? "",
      pagination: initialState?.pagination ?? { pageIndex: 0, pageSize: PAGE_SIZE_OPTIONS[0] ?? 5 },
      sorting: initialState?.sorting ?? [],
    },
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
