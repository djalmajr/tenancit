import type { MutableRefObject } from "react";

export type IdempotencyAttempt = { fingerprint: string; key: string } | null;

export function stableIdempotencyKey(
  ref: MutableRefObject<IdempotencyAttempt>,
  payload: unknown,
): string {
  const fingerprint = JSON.stringify(payload);
  if (ref.current?.fingerprint === fingerprint) return ref.current.key;
  const key = crypto.randomUUID();
  ref.current = { fingerprint, key };
  return key;
}
