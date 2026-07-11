import { readFileSync, readdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import {
  decideScaleGate,
  parseObservedOperationalVolume,
  SCALE_THRESHOLDS,
} from "../src/lib/scale-gate.ts";

const outputDir = process.argv[2];
const observedVolume = parseObservedOperationalVolume(process.env.TENANCIT_SCALE_OBSERVED_VOLUME);
if (!outputDir) throw new Error("output directory is required");

const measurements = readdirSync(outputDir)
  .filter((name) => /^raw-\d+-\d+\.json$/.test(name))
  .map((name) => JSON.parse(readFileSync(join(outputDir, name), "utf8")))
  .sort((a, b) => a.cardinality - b.cardinality || a.run - b.run);

const bySize = new Map();
for (const measurement of measurements) {
  const list = bySize.get(measurement.cardinality) ?? [];
  list.push(measurement);
  bySize.set(measurement.cardinality, list);
}
for (const [size, runs] of bySize) {
  if (runs.length !== 2) throw new Error(`cardinality ${size} needs exactly two measurements`);
}

function confirmedCapacityTriggers(runs) {
  const triggers = [];
  for (const endpoint of Object.keys(runs[0].http)) {
    const payloadConfirmed = runs.every(
      (run) => run.http[endpoint].concurrency1.bytes >= SCALE_THRESHOLDS.hardPayloadBytes,
    );
    const httpConfirmed = runs.every(
      (run) =>
        Math.max(
          run.http[endpoint].concurrency1.summary.p95Ms,
          run.http[endpoint].concurrency10.summary.p95Ms,
        ) >= SCALE_THRESHOLDS.hardHTTPP95Ms,
    );
    if (payloadConfirmed) triggers.push(`${endpoint}:payload>=500KiB`);
    if (httpConfirmed) triggers.push(`${endpoint}:http-p95>=300ms`);
  }
  for (const surface of Object.keys(runs[0].browser)) {
    const uiConfirmed = runs.every((run) =>
      Object.values(run.browser[surface]).some(
        (metric) => metric.summary.p95Ms >= SCALE_THRESHOLDS.hardBrowserP95Ms,
      ),
    );
    if (uiConfirmed) triggers.push(`${surface}:ui-p95>=150ms`);
  }
  return triggers;
}

const capacity = [...bySize.entries()].map(([cardinality, runs]) => ({
  cardinality,
  triggers: confirmedCapacityTriggers(runs),
  soft: {
    itemCount: cardinality >= SCALE_THRESHOLDS.softItems,
    payload250KiB: runs.every((run) =>
      Object.values(run.http).some(
        (endpoint) => endpoint.concurrency1.bytes >= SCALE_THRESHOLDS.softPayloadBytes,
      ),
    ),
  },
}));

const decision = decideScaleGate(
  observedVolume,
  capacity.map((point) => ({
    cardinality: point.cardinality,
    hardTriggers: point.triggers,
    softItemCount: point.soft.itemCount,
    softPayload: point.soft.payload250KiB,
  })),
);
const gateOpen = decision === "OPEN_PAGINATION_EPIC";
const capacityBreakpoint = capacity.find(
  (point) => point.cardinality >= SCALE_THRESHOLDS.hardItems || point.triggers.length > 0,
)?.cardinality ?? null;
const analysisRecommended = capacity.some(
  (point) => point.soft.itemCount || point.soft.payload250KiB,
);

const summary = {
  schemaVersion: 1,
  observedOperationalVolume: observedVolume,
  analysisRecommended,
  capacityBreakpoint,
  decision,
  rationale: gateOpen
    ? "The declared operational point confirms a hard trigger or two persistent soft triggers."
    : "Synthetic capacity breakpoints do not by themselves prove current operational need; the declared volume remains below the gate.",
  capacity,
  measurements: measurements.map(({ cardinality, run, measuredAt, environment, http, browser }) => ({
    cardinality,
    run,
    measuredAt,
    environment,
    http: Object.fromEntries(
      Object.entries(http).map(([id, value]) => [id, {
        bytes: value.concurrency1.bytes,
        p95Concurrency1Ms: value.concurrency1.summary.p95Ms,
        p95Concurrency10Ms: value.concurrency10.summary.p95Ms,
      }]),
    ),
    browser: Object.fromEntries(
      Object.entries(browser).map(([id, value]) => [id, Object.fromEntries(
        Object.entries(value).map(([metric, samples]) => [metric, samples.summary.p95Ms]),
      )]),
    ),
  })),
};

writeFileSync(join(outputDir, "summary.json"), `${JSON.stringify(summary, null, 2)}\n`);

const rows = summary.measurements.map((measurement) => {
  const maxHTTP = Math.max(
    ...Object.values(measurement.http).flatMap((value) => [
      value.p95Concurrency1Ms,
      value.p95Concurrency10Ms,
    ]),
  );
  const maxUI = Math.max(
    ...Object.values(measurement.browser).flatMap((value) => Object.values(value)),
  );
  const maxBytes = Math.max(...Object.values(measurement.http).map((value) => value.bytes));
  return `| ${measurement.cardinality} | ${measurement.run} | ${maxBytes} | ${maxHTTP.toFixed(1)} | ${maxUI.toFixed(1)} |`;
});
const markdown = `# Scale benchmark decision\n\n` +
  `- Declared operational volume: **${observedVolume}** records.\n` +
  `- Decision: **${decision}**.\n` +
  `- Synthetic capacity runs are diagnostic; only the declared/observed operational point can open the implementation gate.\n\n` +
  `| Cardinality | Run | Max payload (B) | Max HTTP p95 (ms) | Max UI p95 (ms) |\n` +
  `|---:|---:|---:|---:|---:|\n${rows.join("\n")}\n\n` +
  `${summary.rationale}\n`;
writeFileSync(join(outputDir, "decision.md"), markdown);
console.log(`${decision}: ${outputDir}`);
