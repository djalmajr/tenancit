import { gzipSync } from "node:zlib";
import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const distDir = new URL("../dist/", import.meta.url);
const manifestPath = new URL(".vite/manifest.json", distDir);
const manifest = JSON.parse(readFileSync(manifestPath, "utf8"));
const entry = Object.values(manifest).find((item) => item.isEntry);

if (!entry) throw new Error("Vite manifest has no JavaScript entry");

const entryPath = new URL(entry.file, distDir);
const entryBytes = readFileSync(entryPath);
const entryRaw = entryBytes.byteLength;
const entryGzip = gzipSync(entryBytes).byteLength;
const limits = { entryRaw: 500_000, entryGzip: 180_000, chunkRaw: 500_000 };
const failures = [];
const expectedDynamicEntries = [
  "src/routes/index.page.tsx",
  "src/routes/tenants.page.tsx",
  "src/routes/tenant-detail.page.tsx",
  "src/routes/definitions.page.tsx",
  "src/routes/definition-detail.page.tsx",
  "src/routes/api-clients.page.tsx",
];

if (entryRaw >= limits.entryRaw) {
  failures.push(`entry raw ${entryRaw} B >= ${limits.entryRaw} B`);
}
if (entryGzip >= limits.entryGzip) {
  failures.push(`entry gzip ${entryGzip} B >= ${limits.entryGzip} B`);
}

const assetsDir = new URL("assets/", distDir);
for (const name of readdirSync(assetsDir)) {
  if (!name.endsWith(".js")) continue;
  const raw = statSync(join(fileURLToPath(assetsDir), name)).size;
  if (raw >= limits.chunkRaw) {
    failures.push(`chunk ${name} raw ${raw} B >= ${limits.chunkRaw} B`);
  }
}

for (const source of expectedDynamicEntries) {
  if (!manifest[source]?.isDynamicEntry) {
    failures.push(`missing dynamic route entry ${source}`);
  }
}

const initialKeys = new Set();
const visitInitialImport = (key) => {
  if (initialKeys.has(key)) return;
  const item = manifest[key];
  if (!item) {
    failures.push(`manifest import ${key} is missing`);
    return;
  }
  initialKeys.add(key);
  for (const importedKey of item.imports ?? []) visitInitialImport(importedKey);
};
const entryKey = Object.entries(manifest).find(([, item]) => item === entry)?.[0];
if (entryKey) visitInitialImport(entryKey);

let initialRaw = 0;
let initialGzip = 0;
for (const key of initialKeys) {
  const item = manifest[key];
  if (!item.file.endsWith(".js")) continue;
  const bytes = readFileSync(new URL(item.file, distDir));
  initialRaw += bytes.byteLength;
  initialGzip += gzipSync(bytes).byteLength;
}

console.log(
  `bundle budget: entry ${entryRaw} B raw / ${entryGzip} B gzip; ` +
    `initial graph ${initialRaw} B raw / ${initialGzip} B gzip; ` +
    `${expectedDynamicEntries.length} lazy route entries`,
);
if (failures.length > 0) throw new Error(failures.join("\n"));
