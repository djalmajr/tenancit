export const SCALE_THRESHOLDS = {
  softItems: 500,
  softPayloadBytes: 256_000,
  hardItems: 2_000,
  hardPayloadBytes: 512_000,
  hardHTTPP95Ms: 300,
  hardBrowserP95Ms: 150,
} as const;

export type CapacityPoint = {
  cardinality: number;
  hardTriggers: string[];
  softItemCount: boolean;
  softPayload: boolean;
};

export type ScaleDecision = "KEEP_FULL_LISTS" | "OPEN_PAGINATION_EPIC";

export function parseObservedOperationalVolume(value: string | undefined): number {
  if (value === undefined) return 0;
  const parsed = Number(value);
  if (!Number.isSafeInteger(parsed) || parsed < 0 || value.trim() === "") {
    throw new Error("TENANCIT_SCALE_OBSERVED_VOLUME must be a non-negative integer");
  }
  return parsed;
}

export function nearestRank(values: number[], percentile: number) {
  if (values.length === 0) throw new Error("nearestRank needs at least one sample");
  if (percentile <= 0 || percentile > 1) throw new Error("percentile must be in (0, 1]");
  const sorted = [...values].sort((a, b) => a - b);
  return sorted[Math.ceil(sorted.length * percentile) - 1];
}

export function decideScaleGate(
  observedOperationalVolume: number,
  capacityPoints: CapacityPoint[],
): ScaleDecision {
  if (observedOperationalVolume >= SCALE_THRESHOLDS.hardItems) {
    return "OPEN_PAGINATION_EPIC";
  }

  const operationalPoint = capacityPoints
    .filter((point) => point.cardinality <= observedOperationalVolume)
    .sort((a, b) => b.cardinality - a.cardinality)[0];

  if (!operationalPoint) return "KEEP_FULL_LISTS";
  if (operationalPoint.hardTriggers.length > 0) return "OPEN_PAGINATION_EPIC";
  if (operationalPoint.softItemCount && operationalPoint.softPayload) {
    return "OPEN_PAGINATION_EPIC";
  }
  return "KEEP_FULL_LISTS";
}
