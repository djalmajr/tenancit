import type { Column } from "@tanstack/react-table";
import { ChevronDown, ChevronUp, ChevronsUpDown } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface DataTableColumnHeaderLabels {
  asc?: string;
  desc?: string;
  reset?: string;
}

interface DataTableColumnHeaderProps<TData, TValue> {
  className?: string;
  column: Column<TData, TValue>;
  label: string;
  labels?: DataTableColumnHeaderLabels;
}

export function DataTableColumnHeader<TData, TValue>({
  className,
  column,
  label,
  labels,
}: DataTableColumnHeaderProps<TData, TValue>) {
  const align = (column.columnDef.meta as { align?: "center" | "left" | "right" } | undefined)?.align;
  const alignmentClass =
    align === "right"
      ? "ml-auto justify-end text-right"
      : align === "center"
        ? "mx-auto justify-center text-center"
        : "-ml-2 justify-start text-left";
  const sorted = column.getIsSorted();

  if (!column.getCanSort()) {
    return <div className={cn("w-fit", alignmentClass, className)}>{label}</div>;
  }

  const nextSortLabel =
    sorted === "asc"
      ? labels?.desc
      : sorted === "desc"
        ? labels?.reset
        : labels?.asc;

  return (
    <Button
      aria-label={nextSortLabel ? `${label}: ${nextSortLabel}` : label}
      className={cn("font-medium text-muted-foreground", alignmentClass, className)}
      onClick={column.getToggleSortingHandler()}
      size="sm"
      title={nextSortLabel}
      type="button"
      variant="ghost"
    >
      {label}
      {sorted === "desc" ? <ChevronDown /> : sorted === "asc" ? <ChevronUp /> : <ChevronsUpDown />}
    </Button>
  );
}
