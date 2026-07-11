export function RoutePending() {
  return (
    <div className="flex min-h-40 items-center justify-center" role="status" aria-live="polite">
      <span className="text-sm text-muted-foreground">Carregando…</span>
    </div>
  );
}
