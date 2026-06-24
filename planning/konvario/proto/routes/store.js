// Store reativo em memória + persistência em localStorage.
// Garante que TODA ação do protótipo (criar/editar/desativar/remover) tenha efeito
// visível e sobreviva a reloads. Sem backend.

import { useState, useEffect } from "preact/hooks";
import { TENANTS, DEFINITIONS, API_CLIENTS } from "~/routes/_data.js";

const KEY = "rt-proto-state-v2";
const listeners = new Set();

const clone = (x) => JSON.parse(JSON.stringify(x));

function seed() {
  return {
    tenants: clone(TENANTS),
    definitions: clone(DEFINITIONS),
    apiClients: clone(API_CLIENTS),
  };
}

function load() {
  try {
    const raw = localStorage.getItem(KEY);
    if (raw) return JSON.parse(raw);
  } catch (_) {}
  return seed();
}

let state = load();

function commit(next) {
  state = next;
  try {
    localStorage.setItem(KEY, JSON.stringify(state));
  } catch (_) {}
  listeners.forEach((l) => l(state));
}

export function getState() {
  return state;
}

export function resetState() {
  commit(seed());
}

export function useStore() {
  const [, force] = useState(0);
  useEffect(() => {
    const l = () => force((v) => v + 1);
    listeners.add(l);
    return () => listeners.delete(l);
  }, []);
  return state;
}

let counter = 1;
const uid = (prefix) => `${prefix}-${Date.now().toString(36)}-${counter++}`;

// ---------- Tenants ----------

export function addTenant({ name, slug, hostname }) {
  const id = uid("t");
  const tenant = {
    id,
    slug: slug || name.toLowerCase().replace(/\s+/g, "-"),
    name,
    status: "active",
    domains: hostname ? [{ id: uid("d"), hostname }] : [],
    resources: [],
  };
  commit({ ...state, tenants: [...state.tenants, tenant] });
  return id;
}

export function updateTenant(id, patch) {
  commit({
    ...state,
    tenants: state.tenants.map((t) => (t.id === id ? { ...t, ...patch } : t)),
  });
}

function mutateTenant(id, fn) {
  commit({
    ...state,
    tenants: state.tenants.map((t) => (t.id === id ? fn({ ...t }) : t)),
  });
}

export function addDomain(tenantId, hostname) {
  mutateTenant(tenantId, (t) => {
    t.domains = [...t.domains, { id: uid("d"), hostname }];
    return t;
  });
}

export function removeDomain(tenantId, domainId) {
  mutateTenant(tenantId, (t) => {
    t.domains = t.domains.filter((d) => d.id !== domainId);
    return t;
  });
}

export function addResource(tenantId, definition, values) {
  mutateTenant(tenantId, (t) => {
    t.resources = [
      ...t.resources,
      {
        id: uid("r"),
        definitionKey: definition.key,
        definitionName: definition.name,
        status: "active",
        values,
      },
    ];
    return t;
  });
}

export function toggleResourceStatus(tenantId, resourceId) {
  mutateTenant(tenantId, (t) => {
    t.resources = t.resources.map((r) =>
      r.id === resourceId ? { ...r, status: r.status === "active" ? "inactive" : "active" } : r
    );
    return t;
  });
}

export function removeResource(tenantId, resourceId) {
  mutateTenant(tenantId, (t) => {
    t.resources = t.resources.filter((r) => r.id !== resourceId);
    return t;
  });
}

// ---------- Resource Definitions ----------

export function addDefinition({ key, name, description, icon }) {
  const id = uid("def");
  const def = {
    id,
    key,
    name,
    status: "active",
    description: description || "",
    icon: icon || "lucide:box",
    fields: [],
  };
  commit({ ...state, definitions: [...state.definitions, def] });
  return id;
}

export function toggleDefinitionStatus(id) {
  commit({
    ...state,
    definitions: state.definitions.map((d) =>
      d.id === id ? { ...d, status: d.status === "active" ? "inactive" : "active" } : d
    ),
  });
}

export function addField(defId, field) {
  commit({
    ...state,
    definitions: state.definitions.map((d) =>
      d.id === defId ? { ...d, fields: [...d.fields, { id: uid("f"), ...field }] } : d
    ),
  });
}

export function removeField(defId, fieldId) {
  commit({
    ...state,
    definitions: state.definitions.map((d) =>
      d.id === defId ? { ...d, fields: d.fields.filter((f) => f.id !== fieldId) } : d
    ),
  });
}

// ---------- API Clients ----------

export function addApiClient(name) {
  const id = uid("c");
  const token = `rt_live_${Math.random().toString(36).slice(2, 10)}${Math.random().toString(36).slice(2, 10)}`;
  const client = {
    id,
    name,
    status: "active",
    key_preview: `rt_live_••••••${token.slice(-4)}`,
    created_at: new Date().toISOString().slice(0, 10),
  };
  commit({ ...state, apiClients: [...state.apiClients, client] });
  return token;
}

export function toggleApiClient(id) {
  commit({
    ...state,
    apiClients: state.apiClients.map((c) =>
      c.id === id ? { ...c, status: c.status === "active" ? "revoked" : "active" } : c
    ),
  });
}

// ---------- selectors ----------

export const findTenant = (id) => state.tenants.find((t) => t.id === id);
export const findDefinition = (id) => state.definitions.find((d) => d.id === id);
export const findDefinitionByKey = (key) => state.definitions.find((d) => d.key === key);
