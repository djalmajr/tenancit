import { cva, type VariantProps } from "class-variance-authority";
import type * as React from "react";
import { Slot as SlotPrimitive } from "@/components/ui/slot";
import { cn } from "@/lib/utils";

const statusVariants = cva("inline-flex w-fit shrink-0 items-center gap-1.5 overflow-hidden whitespace-nowrap rounded-full border px-2.5 py-1 text-xs font-medium transition-colors", {
  variants: { variant: {
    default: "border-transparent bg-muted text-muted-foreground",
    success: "border-green-500/20 bg-green-500/10 text-green-600 dark:text-green-400",
    error: "border-destructive/20 bg-destructive/10 text-destructive",
    warning: "border-orange-500/20 bg-orange-500/10 text-orange-600 dark:text-orange-400",
    info: "border-blue-500/20 bg-blue-500/10 text-blue-600 dark:text-blue-400",
  } }, defaultVariants: { variant: "default" },
});

function Status({ asChild, className, variant = "default", ...props }: React.ComponentProps<"div"> & VariantProps<typeof statusVariants> & { asChild?: boolean }) {
  const Root = asChild ? SlotPrimitive : "div";
  return <Root className={cn(statusVariants({ variant }), className)} data-slot="status" data-variant={variant} {...props} />;
}
function StatusLabel({ className, ...props }: React.ComponentProps<"div">) { return <div className={cn("leading-none", className)} data-slot="status-label" {...props} />; }

export { Status, StatusLabel };
