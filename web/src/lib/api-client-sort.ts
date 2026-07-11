export function apiClientCreatedAtSortValue(value: string | undefined): number {
  const timestamp = Date.parse(value ?? "");
  return Number.isNaN(timestamp) ? 0 : timestamp;
}
