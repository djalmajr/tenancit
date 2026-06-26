# Usability — Acessar painel e revisar visão geral (admin-auth-overview)
- **Persona:** platform-operator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ✅ completável — **nenhum achado** (re-run contra o TenancitIO atual)
- **Ambiente:** stack TenancitIO atual (Vite :5180 + API :8087 + Postgres :5433)

## Walkthrough
1. Sem token salvo → tela **"Acesso administrativo"** em card central, **shell oculto** atrás, autofocus no Token. ✅
2–3. Dropdowns de idioma/tema no login funcionam (pt/en/es, claro/escuro/sistema) — cobertura detalhada em `i18n-and-preferences-persistence`. ✅
4. **Entrar** com `tenancit_admin_dev` → página recarrega e abre a Visão geral. ✅
5. KPIs presentes: **Tenants ativos, Domínios, Recursos provisionados, Definições ativas**. ✅
6. Card **Tenants** lista cada tenant com nome + host primário (ou **"sem domínio"**) + nº de recursos + status (ex.: "Acme Corp · dup.e2e.local · 0 recursos · ativo"). ✅
7–9. Idioma/tema no header e navegação pela sidebar (Visão geral/Tenants/Recursos/Chaves de API) — funcionam (cobertura em flows i18n/responsive). ✅
10. **Colapsar/expandir** a sidebar → `data-state="collapsed"`, navegação segue acessível por ícones, expande de volta. ✅
11. **Sair** → token removido, recarrega, tela "Acesso administrativo" volta a bloquear o painel. ✅

## Findings
Nenhum. Nenhuma regressão das mudanças desta sessão (login/erro/KPIs/nav/logout intactos).
