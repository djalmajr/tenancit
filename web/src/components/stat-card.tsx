import type { ReactNode } from "react";
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export function StatCard({ hint, icon, label, value }: { hint: string; icon: ReactNode; label: string; value: number | string }) {
  return <Card className="gap-1" size="sm">
    <CardHeader className="border-b-0 pb-0">
      <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
      <CardAction className="text-muted-foreground">{icon}</CardAction>
    </CardHeader>
    <CardContent className="pt-0">
      <div className="text-3xl font-bold leading-none tabular-nums tracking-tight">{value}</div>
      <p className="mt-1.5 text-xs text-muted-foreground">{hint}</p>
    </CardContent>
  </Card>;
}
