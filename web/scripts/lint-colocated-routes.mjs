#!/usr/bin/env node
/**
 * Reports product modules that belong next to a single route
 * (`-components/`, `-hooks/`, `-queries.ts`) or that should be lifted
 * out of a private folder because more than one route imports them.
 *
 *   node scripts/lint-colocated-routes.mjs
 *   node scripts/lint-colocated-routes.mjs --cwd ../app --report
 *   node scripts/lint-colocated-routes.mjs --src ./src --json
 */
import { existsSync, readdirSync, readFileSync, statSync } from "node:fs"
import {
  basename,
  dirname,
  extname,
  isAbsolute,
  join,
  relative,
  resolve,
  sep,
} from "node:path"
import { pathToFileURL } from "node:url"

const SOURCE_EXT = new Set([".ts", ".tsx", ".js", ".jsx"])
const SKIP_DIR_NAMES = new Set([
  "node_modules",
  "dist",
  "build",
  "coverage",
  "paraglide",
  ".git",
])
const TEST_RE = /\.(?:test|spec)\./
const D_TS_RE = /\.d\.ts$/
const IMPORT_FROM_RE = /\b(?:from|import)\s*\(\s*['"]([^'"]+)['"]\s*\)|\bfrom\s+['"]([^'"]+)['"]/g
const ROUTE_FACTORY_RE =
  /\b(?:createRoute|createFileRoute|createLazyRoute|createRootRoute|createRootRouteWithContext)\s*[<(]/
const ROUTE_EXPORT_RE = /\bexport\s+const\s+Route\b/

export function parseArgs(argv) {
  const opts = {
    cwd: process.cwd(),
    src: null,
    json: false,
    report: false,
    help: false,
  }
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i]
    if (arg === "--help" || arg === "-h") opts.help = true
    else if (arg === "--json") opts.json = true
    else if (arg === "--report") opts.report = true
    else if (arg === "--cwd") opts.cwd = resolve(argv[++i] ?? "")
    else if (arg === "--src") opts.src = argv[++i]
    else if (arg.startsWith("--cwd=")) opts.cwd = resolve(arg.slice(6))
    else if (arg.startsWith("--src=")) opts.src = arg.slice(6)
    else throw new Error(`unknown flag: ${arg}`)
  }
  return opts
}

export function lintColocatedRoutes({
  cwd = process.cwd(),
  src = null,
} = {}) {
  const appRoot = resolve(cwd)
  const srcDir = resolve(appRoot, src ?? "src")
  const routesDir = join(srcDir, "routes")
  if (!existsSync(srcDir)) {
    throw new Error(`src not found: ${srcDir}`)
  }
  if (!existsSync(routesDir)) {
    throw new Error(`routes not found: ${routesDir}`)
  }

  const files = listSourceFiles(srcDir)
  const contents = new Map()
  for (const file of files) {
    contents.set(file, readFileSync(file, "utf8"))
  }

  const resolvedImports = new Map()
  const reverse = new Map()
  for (const file of files) {
    const specs = importSpecifiers(contents.get(file) ?? "")
    const targets = []
    for (const spec of specs) {
      const target = resolveImport(file, spec, srcDir)
      if (target && files.includes(target)) targets.push(target)
    }
    resolvedImports.set(file, targets)
    for (const target of targets) {
      const importers = reverse.get(target)
      if (importers) importers.push(file)
      else reverse.set(target, [file])
    }
  }

  const treeImported = routeFilesImportedByTree(srcDir, routesDir, files, contents)
  const routeFiles = files.filter((file) =>
    isRouteModule(file, routesDir, contents.get(file) ?? "", treeImported)
  )
  const routeSet = new Set(routeFiles)
  const findings = []

  for (const file of files) {
    if (TEST_RE.test(file) || D_TS_RE.test(file)) continue
    if (routeSet.has(file)) continue
    if (!isCandidate(file, srcDir, routesDir)) continue

    const consumers = routeConsumers(file, reverse, routeSet)
    if (consumers.length === 0) continue

    const pageConsumers = consumers.filter((route) => !isRootRoute(route))
    const already = colocatedTarget(file, routesDir)

    if (pageConsumers.length === 1) {
      const route = pageConsumers[0]
      const dest = suggestedPath(file, route, routesDir)
      if (pathsEqual(file, dest)) continue
      findings.push({
        kind: "colocate",
        file: rel(appRoot, file),
        consumers: pageConsumers.map((item) => rel(appRoot, item)),
        suggested: rel(appRoot, dest),
        convertRouteToFolder: isFlatRouteFile(route, routesDir),
      })
      continue
    }

    if (already && pageConsumers.length > 1) {
      findings.push({
        kind: "lift",
        file: rel(appRoot, file),
        consumers: pageConsumers.map((item) => rel(appRoot, item)),
        suggested: rel(appRoot, liftDestination(file, srcDir)),
        convertRouteToFolder: false,
      })
    }
  }

  findings.sort((a, b) => a.file.localeCompare(b.file))
  return { appRoot, srcDir, routesDir, routeCount: routeFiles.length, findings }
}

export function formatFindings(result, { json = false } = {}) {
  if (json) return JSON.stringify(result, null, 2)
  const { findings } = result
  if (findings.length === 0) {
    return `colocated-routes: 0 findings (${result.routeCount} routes)\n`
  }
  const lines = [
    `colocated-routes: ${findings.length} finding(s) (${result.routeCount} routes)`,
    "",
  ]
  for (const finding of findings) {
    const label = finding.kind === "lift" ? "LIFT     " : "COLLOCATE"
    lines.push(`${label}  ${finding.file}`)
    lines.push(`  consumers: ${finding.consumers.join(", ")}`)
    lines.push(`  move to:   ${finding.suggested}`)
    if (finding.convertRouteToFolder) {
      const route = finding.consumers[0]
      const folder = route.replace(/\.(tsx|ts|jsx|js)$/, "")
      lines.push(`  first:     ${route} → ${folder}/index${extname(route)}`)
    }
    lines.push("")
  }
  return lines.join("\n")
}

export function runCli(argv) {
  let opts
  try {
    opts = parseArgs(argv)
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error))
    process.exit(2)
  }
  if (opts.help) {
    console.log(`Usage: lint-colocated-routes [--cwd <app>] [--src <dir>] [--json] [--report]

Flags:
  --cwd <dir>   App root (default: current directory)
  --src <dir>   Source directory, relative to cwd or absolute (default: src)
  --json        Machine-readable output
  --report      Always exit 0 (inventory; do not fail CI)
  --help        Show this help
`)
    process.exit(0)
  }
  const result = lintColocatedRoutes({ cwd: opts.cwd, src: opts.src })
  process.stdout.write(formatFindings(result, { json: opts.json }))
  if (!opts.report && result.findings.length > 0) process.exit(1)
}

function listSourceFiles(root) {
  const out = []
  const walk = (dir) => {
    let entries
    try {
      entries = readdirSync(dir, { withFileTypes: true })
    } catch {
      return
    }
    for (const entry of entries) {
      if (entry.name.startsWith(".") && entry.name !== ".") continue
      const full = join(dir, entry.name)
      if (entry.isDirectory()) {
        if (SKIP_DIR_NAMES.has(entry.name)) continue
        walk(full)
        continue
      }
      if (!entry.isFile()) continue
      if (D_TS_RE.test(entry.name)) continue
      if (!SOURCE_EXT.has(extname(entry.name))) continue
      try {
        if (!statSync(full).isFile()) continue
      } catch {
        continue
      }
      out.push(full)
    }
  }
  walk(root)
  return out.sort()
}

function stripComments(source) {
  return source
    .replace(/\/\*[\s\S]*?\*\//g, "")
    .replace(/(^|[^:\\])\/\/.*$/gm, "$1")
}

function importSpecifiers(source) {
  const specs = []
  const body = stripComments(source)
  IMPORT_FROM_RE.lastIndex = 0
  let match
  while ((match = IMPORT_FROM_RE.exec(body))) {
    const spec = match[1] ?? match[2]
    if (spec) specs.push(spec)
  }
  return specs
}

function resolveImport(fromFile, spec, srcDir) {
  if (!spec) return null
  if (spec.startsWith("\0")) return null
  let target
  if (spec.startsWith("@/")) target = join(srcDir, spec.slice(2))
  else if (spec.startsWith("./") || spec.startsWith("../")) {
    target = resolve(dirname(fromFile), spec)
  } else {
    return null
  }
  return existingSource(target)
}

function existingSource(target) {
  const stripped = target.replace(/\.(tsx|ts|jsx|js)$/, "")
  const candidates = [
    target,
    `${stripped}.tsx`,
    `${stripped}.ts`,
    `${stripped}.jsx`,
    `${stripped}.js`,
    join(stripped, "index.tsx"),
    join(stripped, "index.ts"),
    join(stripped, "index.jsx"),
    join(stripped, "index.js"),
  ]
  for (const candidate of candidates) {
    if (existsSync(candidate) && SOURCE_EXT.has(extname(candidate))) {
      return resolve(candidate)
    }
  }
  return null
}

const TREE_FILENAMES = [
  "router.tsx",
  "router.ts",
  "routeTree.tsx",
  "routeTree.ts",
  "route-tree.tsx",
  "route-tree.ts",
]

function routeFilesImportedByTree(srcDir, routesDir, files, contents) {
  const imported = new Set()
  const fileSet = new Set(files)
  for (const name of TREE_FILENAMES) {
    const tree = join(srcDir, name)
    if (!fileSet.has(tree) && !existsSync(tree)) continue
    const source = contents.get(tree) ?? (existsSync(tree) ? readFileSync(tree, "utf8") : "")
    if (!source) continue
    for (const spec of importSpecifiers(source)) {
      const target = resolveImport(tree, spec, srcDir)
      if (target && isUnder(target, routesDir) && fileSet.has(target)) {
        imported.add(target)
      }
    }
  }
  return imported
}

function isRouteModule(file, routesDir, source, treeImported) {
  if (!isUnder(file, routesDir)) return false
  if (TEST_RE.test(file)) return false
  if (hasPrivateSegment(relative(routesDir, file))) return false
  if (treeImported.has(file)) return true
  return ROUTE_FACTORY_RE.test(source) || ROUTE_EXPORT_RE.test(source)
}

function isRootRoute(file) {
  return basename(file).startsWith("__root")
}

function isCandidate(file, srcDir, routesDir) {
  if (isUnder(file, join(srcDir, "components", "ui"))) return false
  if (isUnder(file, join(srcDir, "lib"))) return false
  if (isUnder(file, join(srcDir, "locales"))) return false
  if (isUnder(file, join(srcDir, "config"))) return false
  if (isUnder(file, join(srcDir, "types"))) return false
  if (isUnder(file, join(srcDir, "test"))) return false
  if (isUnder(file, routesDir)) return true
  return (
    isUnder(file, join(srcDir, "components")) ||
    isUnder(file, join(srcDir, "hooks"))
  )
}

function hasPrivateSegment(relPath) {
  return relPath.split(sep).some((part) => part.startsWith("-"))
}

function routeConsumers(file, reverse, routeSet) {
  const seen = new Set([file])
  const routes = new Set()
  const stack = [file]
  while (stack.length > 0) {
    const current = stack.pop()
    for (const importer of reverse.get(current) ?? []) {
      if (seen.has(importer)) continue
      seen.add(importer)
      if (TEST_RE.test(importer)) continue
      if (routeSet.has(importer)) {
        routes.add(importer)
        continue
      }
      stack.push(importer)
    }
  }
  return [...routes].sort()
}

function routeFolder(routeFile, routesDir) {
  const relPath = relative(routesDir, routeFile)
  const dir = dirname(relPath)
  const stem = basename(routeFile, extname(routeFile))
  if (stem === "index" || stem === "_index") {
    return dir === "." ? routesDir : join(routesDir, dir)
  }
  return dir === "." ? join(routesDir, stem) : join(routesDir, dir, stem)
}

function isFlatRouteFile(routeFile, routesDir) {
  const stem = basename(routeFile, extname(routeFile))
  return stem !== "index" && stem !== "_index" && stem !== "__root"
}

function colocatedTarget(file, routesDir) {
  if (!isUnder(file, routesDir)) return false
  return hasPrivateSegment(relative(routesDir, file))
}

function privateKind(file) {
  const base = basename(file)
  const parts = file.split(/[/\\]/)
  if (
    parts.includes("-hooks") ||
    parts.includes("hooks") ||
    /^use-/.test(base) ||
    /^use[A-Z]/.test(base)
  ) {
    return "hooks"
  }
  if (/-queries\./.test(base) || /^queries\./.test(base)) return "queries"
  return "components"
}

function suggestedPath(file, routeFile, routesDir) {
  const folder = routeFolder(routeFile, routesDir)
  const kind = privateKind(file)
  const name = basename(file)
  if (kind === "queries") {
    return join(folder, name.startsWith("-") ? name : `-${name}`)
  }
  if (kind === "hooks") return join(folder, "-hooks", name)
  return join(folder, "-components", name)
}

function liftDestination(file, srcDir) {
  const name = basename(file).replace(/^-/, "")
  const kind = privateKind(file)
  if (kind === "hooks") return join(srcDir, "hooks", name)
  if (kind === "queries") return join(srcDir, "lib", name)
  return join(srcDir, "components", name)
}

function isUnder(file, dir) {
  const relPath = relative(dir, file)
  return relPath !== "" && !relPath.startsWith(`..`) && !isAbsolute(relPath)
}

function pathsEqual(a, b) {
  return resolve(a) === resolve(b)
}

function rel(root, file) {
  return relative(root, file).split(sep).join("/")
}

function isDirectRun() {
  const entry = process.argv[1]
  if (!entry) return false
  try {
    return pathToFileURL(resolve(entry)).href === import.meta.url
  } catch {
    return false
  }
}

if (isDirectRun()) {
  try {
    runCli(process.argv.slice(2))
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error))
    process.exit(2)
  }
}
