import { existsSync, readdirSync, readFileSync } from "node:fs";
import { basename, join, resolve } from "node:path";

const ROOT = resolve(import.meta.dir, "..");
const FLOWS_DIR = join(ROOT, "e2e", "flows");
const SPECS_DIR = join(ROOT, "web", "e2e");
const README_PATH = join(ROOT, "e2e", "README.md");
const E2E_RUNNER_PATH = join(ROOT, "scripts", "e2e.sh");
const PLAYWRIGHT_CONFIG_PATH = join(ROOT, "web", "playwright.config.ts");
const EXPECTED_FLOWS = 20;
const EXPECTED_STEPS = 147;

type Flow = {
  id: string;
  path: string;
  steps: number[];
};

type SpecStep = {
  flowId: string;
  number: number;
  path: string;
};

type ReadmeRow = {
  flowId: string;
  spec: string;
  stepPattern: string;
  tier: string;
  firstStep: number;
  lastStep: number;
};

const errors: string[] = [];

function readFlows(): Map<string, Flow> {
  const flows = new Map<string, Flow>();
  for (const name of readdirSync(FLOWS_DIR).filter((file) => file.endsWith(".md")).sort()) {
    const path = join(FLOWS_DIR, name);
    const source = readFileSync(path, "utf8");
    const id = source.match(/^id:\s*([^\s]+)\s*$/m)?.[1];
    if (!id) {
      errors.push(`${path}: frontmatter sem id`);
      continue;
    }
    if (id !== basename(name, ".md")) {
      errors.push(`${path}: id ${id} diverge do nome do arquivo`);
    }
    if (flows.has(id)) errors.push(`${path}: flow-id duplicado ${id}`);
    const steps = [...source.matchAll(/^(\d+)\.\s+\(`[^`]+`\)\s+/gm)].map((match) => Number(match[1]));
    if (steps.length === 0) errors.push(`${path}: nenhum passo numerado encontrado`);
    steps.forEach((number, index) => {
      const expected = index + 1;
      if (number !== expected) errors.push(`${path}: passo ${number} fora da sequência; esperado ${expected}`);
    });
    flows.set(id, { id, path, steps });
  }
  return flows;
}

function readSpecSteps(): SpecStep[] {
  const steps: SpecStep[] = [];
  for (const name of readdirSync(SPECS_DIR).filter((file) => file.endsWith(".e2e.test.ts")).sort()) {
    const path = join(SPECS_DIR, name);
    const source = readFileSync(path, "utf8");
    if (/\.to(?:HaveValue|Equal|Be)\([^\n)]*\btoken\b/i.test(source)) {
      errors.push(`${path}: token bruto não pode ser argumento de matcher (vaza no reporter)`);
    }
    const calls = source.matchAll(/\bflowStep\(\s*["']([^"']+)["']\s*,\s*(\d+)\s*,\s*["'`][^"'`]+["'`]\s*,/g);
    for (const match of calls) {
      steps.push({ flowId: match[1], number: Number(match[2]), path });
    }
  }
  return steps;
}

function readReadme(): ReadmeRow[] {
  if (!existsSync(README_PATH)) {
    errors.push(`${README_PATH}: arquivo ausente`);
    return [];
  }
  const source = readFileSync(README_PATH, "utf8");
  const rows: ReadmeRow[] = [];
  const pattern = /^\|\s*`([^`]+)`\s*\|\s*`([^`]+\.e2e\.test\.ts)`\s*\|\s*`([^`]+)`\s*\|\s*`(pr-critical|full)`\s*\|\s*(\d+)\s*[–-]\s*(\d+)\s*\|$/gm;
  for (const match of source.matchAll(pattern)) {
    rows.push({
      flowId: match[1],
      spec: match[2],
      stepPattern: match[3],
      tier: match[4],
      firstStep: Number(match[5]),
      lastStep: Number(match[6]),
    });
  }
  if (rows.length === 0) errors.push(`${README_PATH}: nenhuma linha de catálogo reconhecida`);
  return rows;
}

const flows = readFlows();
const markdownStepTotal = [...flows.values()].reduce((total, flow) => total + flow.steps.length, 0);
if (flows.size !== EXPECTED_FLOWS) errors.push(`Markdown: ${flows.size}/${EXPECTED_FLOWS} flows`);
if (markdownStepTotal !== EXPECTED_STEPS) errors.push(`Markdown: ${markdownStepTotal}/${EXPECTED_STEPS} passos`);

const specSteps = readSpecSteps();
const stepOwners = new Map<string, string>();
for (const step of specSteps) {
  const flow = flows.get(step.flowId);
  if (!flow) {
    errors.push(`${step.path}: flow-id inexistente ${step.flowId}`);
    continue;
  }
  if (step.number < 1 || step.number > flow.steps.length) {
    errors.push(`${step.path}: ${step.flowId}#${step.number} fora do intervalo 1-${flow.steps.length}`);
  }
  const key = `${step.flowId}#${step.number}`;
  const previousOwner = stepOwners.get(key);
  if (previousOwner) errors.push(`${step.path}: passo duplicado ${key}; já mapeado em ${previousOwner}`);
  else stepOwners.set(key, step.path);
}

for (const flow of flows.values()) {
  for (const number of flow.steps) {
    const key = `${flow.id}#${number}`;
    if (!stepOwners.has(key)) errors.push(`Specs: passo ausente ${key}`);
  }
}
if (specSteps.length !== EXPECTED_STEPS) errors.push(`Specs: ${specSteps.length}/${EXPECTED_STEPS} chamadas flowStep`);

const readmeRows = readReadme();
const readmeByFlow = new Map<string, ReadmeRow>();
for (const row of readmeRows) {
  if (readmeByFlow.has(row.flowId)) errors.push(`${README_PATH}: flow-id duplicado ${row.flowId}`);
  readmeByFlow.set(row.flowId, row);
  const flow = flows.get(row.flowId);
  if (!flow) {
    errors.push(`${README_PATH}: flow-id inexistente ${row.flowId}`);
    continue;
  }
  if (row.firstStep !== 1 || row.lastStep !== flow.steps.length) {
    errors.push(`${README_PATH}: intervalo de ${row.flowId} é ${row.firstStep}-${row.lastStep}; esperado 1-${flow.steps.length}`);
  }
  if (row.stepPattern !== `[${row.flowId}#N] título`) {
    errors.push(`${README_PATH}: padrão test.step inválido para ${row.flowId}: ${row.stepPattern}`);
  }
  if (!existsSync(join(SPECS_DIR, row.spec))) errors.push(`${README_PATH}: spec ausente ${row.spec}`);
  else {
    const specSource = readFileSync(join(SPECS_DIR, row.spec), "utf8");
    const executableTiers = [...specSource.matchAll(/\btag:\s*["']@(pr-critical|full)["']/g)].map(
      (match) => match[1],
    );
    if (executableTiers.length !== 1 || executableTiers[0] !== row.tier) {
      errors.push(
        `${README_PATH}: tier ${row.tier} de ${row.flowId} diverge das tags executáveis ` +
          `de ${row.spec}: ${executableTiers.join(", ") || "nenhuma"}`,
      );
    }
  }
  const mappedSpecs = new Set(
    specSteps.filter((step) => step.flowId === row.flowId).map((step) => basename(step.path)),
  );
  if (mappedSpecs.size !== 1 || !mappedSpecs.has(row.spec)) {
    errors.push(`${README_PATH}: ${row.flowId} aponta para ${row.spec}, mas flowStep aparece em ${[...mappedSpecs].join(", ") || "nenhuma spec"}`);
  }
}
for (const flow of flows.values()) {
  if (!readmeByFlow.has(flow.id)) errors.push(`${README_PATH}: flow ausente ${flow.id}`);
}
if (readmeRows.length !== EXPECTED_FLOWS) errors.push(`README: ${readmeRows.length}/${EXPECTED_FLOWS} flows`);

const runnerSource = readFileSync(E2E_RUNNER_PATH, "utf8");
if (!runnerSource.includes("find \"$output_dir\" -type f -name 'error-context.md' -delete")) {
  errors.push(`${E2E_RUNNER_PATH}: cleanup deve remover error-context.md antes do upload`);
}
const playwrightConfigSource = readFileSync(PLAYWRIGHT_CONFIG_PATH, "utf8");
if (!/\btrace:\s*["']off["']/.test(playwrightConfigSource)) {
  errors.push(`${PLAYWRIGHT_CONFIG_PATH}: traces devem ficar desabilitados para não reter credenciais`);
}

if (errors.length > 0) {
  console.error(`Catálogo E2E inválido (${errors.length} erro(s)):\n- ${errors.join("\n- ")}`);
  process.exit(1);
}

const tiers = readmeRows.reduce<Record<string, number>>((counts, row) => {
  counts[row.tier] = (counts[row.tier] ?? 0) + 1;
  return counts;
}, {});
console.log(
  `Catálogo E2E válido: ${flows.size}/${EXPECTED_FLOWS} flows, ${specSteps.length}/${EXPECTED_STEPS} passos, ` +
    `${tiers["pr-critical"] ?? 0} pr-critical e ${tiers.full ?? 0} full.`,
);
