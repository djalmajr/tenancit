---
name: colocated-routes
description: >
  Colocate route-private UI, hooks, and queries next to the TanStack
  route that owns them (`-components/`, `-hooks/`, `-queries.ts`). Use
  when adding a screen, splitting a route file, moving files out of
  src/components or src/hooks, running lint:routes, migrating an
  existing CIT SPA (Agility, Apigate, or a create-app product), or
  when the user runs /colocated-routes.
---

# Colocated routes

Route-private code lives next to its route. Shared code stays in `src/components/` (not `ui/`) and `src/hooks/`. The oracle is `node scripts/lint-colocated-routes.mjs` (or `npm run lint:routes`). Do not invent a `--fix` / jscodeshift pass; move files and rewrite imports yourself.

## Layout

Apps use an **explicit** `createRoute` tree (`src/router.tsx` or `src/routeTree.tsx`). A route file is either one that exports `Route` / calls `createRoute`, or a page module under `src/routes/` that the tree imports. Folders starting with `-` are not routes (TanStack private-folder convention). Keep that prefix even without the file-routing plugin.

```
src/routes/users/index.tsx          ← route module (exports Route)
src/routes/users/-components/       ← UI only this route renders
src/routes/users/-hooks/            ← hooks / mutations only this route uses
src/routes/users/-queries.ts        ← query options / loaders for this route
src/components/ui/                  ← registry primitives; never colocate here
src/components/                     ← used by two or more routes (or by __root)
src/hooks/                          ← same, for hooks
src/lib/                            ← no UI; leave in place
```

A flat `src/routes/users.tsx` is valid until it gains private files. Then convert it to `src/routes/users/index.tsx` so `-components/` can sit beside it. Dotted names (`workspaces.$workspaceId.tsx`) become a folder of the same stem: `workspaces.$workspaceId/index.tsx`. Nested folders (`workspaces/$workspaceId/`) are better when the team is already there.

`__root` is the shell. Things only the root imports (`app-shell`, `locale-switcher`) stay in `src/components/`.

## When to colocate vs share

| Importers | Where it goes |
| --- | --- |
| One page route | That route's `-components/` / `-hooks/` / `-queries.ts` |
| Two or more page routes | `src/components/` or `src/hooks/` |
| Only `__root` | `src/components/` (shell) |
| `src/components/ui/*` | Leave it. Install via `npx shadcn add @cit/<name>` |

If two *sibling* routes of one feature share a module, prefer the parent folder's `-components/` over a global dump. The linter treats 2+ page routes as `LIFT` to `src/components/`; override toward the parent when the ownership is obvious.

## New screen

1. Add `src/routes/<name>/index.tsx` with `createRoute({ getParentRoute, path, component })`.
2. Register it in the explicit route tree.
3. Put local UI/hooks/queries in `-components/`, `-hooks/`, `-queries.ts`.
4. Import with relative paths (`./-components/user-form`) or `@/routes/<name>/-components/user-form`.
5. Reuse `src/components/ui/*` and existing shared pieces. Do not copy a primitive into `-components/`.

## Migrate an existing app

1. Install the convention if missing (from the shadcn-cit clone, catalog host up):

```bash
node scripts/install-colocated-routes.mjs --cwd /path/to/app
# or, in the app with @cit already registered:
npx shadcn add @cit/colocated-routes --yes
```

2. Inventory (does not fail the process):

```bash
node scripts/lint-colocated-routes.mjs --cwd /path/to/app --report
# JSON for a large move:
node scripts/lint-colocated-routes.mjs --cwd /path/to/app --json --report
```

3. Work **one route at a time**. For each `COLLOCATE` finding:

   - If `convertRouteToFolder` is true, `git mv src/routes/foo.tsx src/routes/foo/index.tsx`.
   - `git mv` the file to the `suggested` path.
   - Rewrite imports in the route and in the moved file. Keep `@/components/ui/*` and `@/lib/*` as they are.
   - Update the route-tree import only if it pointed at `foo.tsx` by filename; `@/routes/foo` still resolves to the folder.

4. For each `LIFT` finding, move to `src/components/` or `src/hooks/` and fix imports.

5. Re-run the linter on that route's files. Do not swallow new shared usage by leaving a copy behind.

6. Do not move `ui/`, i18n catalogs, API clients, or test setup.

## Lint

- Exit 1 when there are findings (CI / new apps). `--report` always exits 0.
- `--cwd` is the app root; `--src` defaults to `src`.
- Candidate roots: `src/components` except `ui/`, `src/hooks`, and non-route files under `src/routes`.
- Route consumers are counted through non-route importers (a drawer imported only by `users` counts as one). Tests do not count.
- No `--fix`. This skill *is* the migration.

## Out of scope

Registry tokens, `add @cit/app`, i18n, and icons. Those have their own docs. This skill only places product code relative to routes.
