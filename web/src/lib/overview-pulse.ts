import type { APIClientUsageRecord, ApiClient } from "@/lib/api";

export function currentUTCMonthRange(now = new Date()) {
  const year = now.getUTCFullYear();
  const month = now.getUTCMonth();
  return {
    from: new Date(Date.UTC(year, month, 1)).toISOString().slice(0, 10),
    to: new Date(Date.UTC(year, month + 1, 0)).toISOString().slice(0, 10),
  };
}

export function countExpiringAPIClients(clients: ApiClient[], now = new Date(), windowDays = 30) {
  const from = now.getTime();
  const through = from + windowDays * 24 * 60 * 60 * 1000;
  return clients.filter((client) => {
    if (client.status !== "active" || !client.expires_at) return false;
    const expiresAt = new Date(client.expires_at).getTime();
    return Number.isFinite(expiresAt) && expiresAt >= from && expiresAt <= through;
  }).length;
}

export function totalUsageRequests(records: APIClientUsageRecord[]) {
  return records.reduce((total, record) => total + record.request_count, 0);
}
