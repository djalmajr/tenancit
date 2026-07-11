import { chromium } from "@playwright/test";
import { mkdirSync, writeFileSync } from "node:fs";
import os from "node:os";
import { dirname } from "node:path";
import { performance } from "node:perf_hooks";
import { nearestRank } from "../src/lib/scale-gate.ts";

const baseURL = process.env.TENANCIT_SCALE_BASE_URL;
const adminToken = process.env.TENANCIT_E2E_ADMIN_TOKEN;
const cardinality = Number(process.env.TENANCIT_SCALE_CARDINALITY);
const run = Number(process.env.TENANCIT_SCALE_RUN);
const outputPath = process.argv[2];

if (!baseURL || !adminToken || !cardinality || !run || !outputPath) {
  throw new Error("base URL, admin token, cardinality, run, and output path are required");
}

const endpoints = [
  { id: "tenants", path: "/v1/admin/tenants" },
  { id: "definitions", path: "/v1/admin/resource-definitions" },
  { id: "api-clients", path: "/v1/admin/api-clients" },
  { id: "overview", path: "/v1/admin/overview" },
];

function summary(samples) {
  return {
    count: samples.length,
    p50Ms: nearestRank(samples, 0.5),
    p95Ms: nearestRank(samples, 0.95),
    maxMs: Math.max(...samples),
  };
}

async function timedFetch(path) {
  const started = performance.now();
  const response = await fetch(`${baseURL}${path}`, {
    headers: {
      Accept: "application/json",
      "Accept-Encoding": "identity",
      Authorization: `Bearer ${adminToken}`,
    },
  });
  const body = new Uint8Array(await response.arrayBuffer());
  const durationMs = performance.now() - started;
  if (!response.ok) throw new Error(`${path} returned ${response.status}`);
  const decoded = JSON.parse(new TextDecoder().decode(body));
  const responseCardinality = Array.isArray(decoded) ? decoded.length : decoded.tenantCards?.length;
  if (responseCardinality !== cardinality) {
    throw new Error(`${path} returned ${responseCardinality} records, expected ${cardinality}`);
  }
  return { bytes: body.byteLength, durationMs };
}

async function measureHTTP(endpoint, concurrency) {
  for (let i = 0; i < 10; i += 1) await timedFetch(endpoint.path);

  const samples = [];
  let bytes = 0;
  const batches = [];
  while (samples.length < 100) {
    const batchSize = Math.min(concurrency, 100 - samples.length);
    const batchStarted = performance.now();
    const batch = await Promise.all(
      Array.from({ length: batchSize }, () => timedFetch(endpoint.path)),
    );
    for (const sample of batch) {
      samples.push(sample.durationMs);
      bytes = sample.bytes;
    }
    const batchDurationMs = performance.now() - batchStarted;
    batches.push({
      durationMs: batchDurationMs,
      requests: batchSize,
      throughputPerSecond: (batchSize * 1000) / batchDurationMs,
    });
  }
  return { batches, bytes, concurrency, samples, summary: summary(samples) };
}

async function afterRender(page) {
  await page.evaluate(
    () => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))),
  );
}

async function measureBrowser() {
  const browser = await chromium.launch(
    process.env.CI ? { headless: true } : { channel: "chrome", headless: true },
  );
  const context = await browser.newContext({
    deviceScaleFactor: 1,
    locale: "pt-BR",
    viewport: { width: 1440, height: 900 },
  });
  await context.addInitScript(
    ({ token }) => window.localStorage.setItem("tenancitAdminToken", token),
    { token: adminToken },
  );
  const page = await context.newPage();
  const result = {};

  try {
    for (const surface of [
      {
        apiPath: "/v1/admin/overview",
        id: "overview",
        path: "/",
        heading: "Visão geral",
        kind: "render",
      },
      {
        apiPath: "/v1/admin/resource-definitions",
        id: "definitions",
        path: "/resource-definitions",
        heading: "Definições de recurso",
        kind: "render",
      },
      {
        apiPath: "/v1/admin/tenants",
        filterValue: "Benchmark Tenant 00001",
        id: "tenants",
        path: "/tenants",
        heading: "Tenants",
        kind: "table",
      },
      {
        apiPath: "/v1/admin/api-clients",
        filterValue: "Benchmark Client 00001",
        id: "api-clients",
        path: "/api-clients",
        heading: "Chaves de API",
        kind: "table",
      },
    ]) {
      const renderSamples = [];
      for (let i = 0; i < 35; i += 1) {
        const started = performance.now();
        const responsePromise = page.waitForResponse(
          (response) => new URL(response.url()).pathname === surface.apiPath,
        );
        await page.goto(`${baseURL}${surface.path}`);
        const response = await responsePromise;
        await response.finished();
        await page.getByRole("heading", { name: surface.heading, exact: true }).waitFor();
        await afterRender(page);
        if (i >= 5) renderSamples.push(performance.now() - started);
      }

      const surfaceResult = { render: { samples: renderSamples, summary: summary(renderSamples) } };
      if (surface.kind === "table") {
        const search = page.locator('input[aria-label][class*="max-w-xs"]').first();
        const filterSamples = [];
        const sortSamples = [];
        for (let i = 0; i < 35; i += 1) {
          let started = performance.now();
          await search.fill(i % 2 === 0 ? surface.filterValue : "");
          await afterRender(page);
          if (i >= 5) filterSamples.push(performance.now() - started);

          started = performance.now();
          await page.getByRole("button", { name: /Nome/ }).first().click();
          await afterRender(page);
          if (i >= 5) sortSamples.push(performance.now() - started);
        }
        surfaceResult.filter = { samples: filterSamples, summary: summary(filterSamples) };
        surfaceResult.sort = { samples: sortSamples, summary: summary(sortSamples) };
      }
      result[surface.id] = surfaceResult;
    }
  } finally {
    await browser.close();
  }
  return result;
}

const http = {};
for (const endpoint of endpoints) {
  http[endpoint.id] = {
    concurrency1: await measureHTTP(endpoint, 1),
    concurrency10: await measureHTTP(endpoint, 10),
  };
}

const payload = {
  schemaVersion: 1,
  measuredAt: new Date().toISOString(),
  cardinality,
  run,
  environment: {
    commit: process.env.TENANCIT_SCALE_COMMIT ?? "unknown",
    dirty: process.env.TENANCIT_SCALE_DIRTY === "1",
    platform: os.platform(),
    release: os.release(),
    arch: os.arch(),
    cpu: os.cpus()[0]?.model ?? "unknown",
    cpuCount: os.cpus().length,
    postgres: process.env.TENANCIT_SCALE_POSTGRES_VERSION ?? "unknown",
    chromium: process.env.TENANCIT_SCALE_CHROMIUM_VERSION ?? "unknown",
    viewport: "1440x900",
  },
  http,
  browser: await measureBrowser(),
};

mkdirSync(dirname(outputPath), { recursive: true, mode: 0o700 });
writeFileSync(outputPath, `${JSON.stringify(payload, null, 2)}\n`, { mode: 0o600 });
console.log(`scale benchmark ${cardinality} run ${run}: ${outputPath}`);
