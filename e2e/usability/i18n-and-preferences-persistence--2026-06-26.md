# Usability — Tradução das telas e persistência de idioma/tema (i18n-and-preferences-persistence)
- **Persona:** platform-operator · **Date:** 2026-06-26 · **Entry:** http://localhost:5180/
- **Verdict:** ✅ completável — i18n completo e prefs persistem, **nenhum achado**
- **Ambiente:** stack KonvarIO atual (Vite :5180 + API :8087 + Postgres :5433)

## Walkthrough
2. **shell** — dropdown de idioma (🇧🇷/🇺🇸/🇪🇸) → **English**: nav "Overview/Tenants/Resources/API Keys", KPIs e header em inglês; `konvarioLocale=en-US`. ✅
3. **EN em todas as telas** — varredura de tenants-list, tenant-detail, definitions-list, definition-detail e api-clients: **0 strings residuais em PT**. Infra da DataTable também traduzida ("Page 1 of 3", "Rows per page", "Search by name, token, or status..."); diálogo "New API key / The token will be shown only once after creation / Cancel / Generate token". ✅
4. **Español** — overview "Vista general", nav "Claves de API", "Filas por página", "14 elementos", "activo", placeholder "Buscar por nombre, token o status..."; **0 vazamentos** de EN/PT (atenção: "Página"/"de" são idênticos em PT/ES — falso positivo descartado). ✅
5. **tema** — opções Claro/Oscuro/Sistema; **Oscuro** aplica `.dark` no root, `konvarioTheme=dark`, bom contraste/legibilidade (screenshot). ✅
6. **reload** — após refresh, `es-ES` + `dark` permanecem (localStorage + classe + nav em ES). ✅
7. **logout** — "Salir" → tela "Acceso administrativo" **em ES + dark**, token limpo, controles refletindo a preferência persistida. ✅

## Findings (prioritized)
Nenhum. Cobertura i18n completa nas 3 locales (garantida em tempo de tipo por `MESSAGES: Record<Locale, Record<TranslationKey, string>>` — chave faltante não compila), sem strings hardcoded vazando, tema escuro consistente, e idioma+tema sobrevivem a navegação, reload e logout (reaparecem no login).

## Observações
- O brand **KonvarIO** aparece correto no subtítulo de login em ES também (confirma o rebrand nas 3 locales).
- Botão de login em ES é "Entrar" (válido em espanhol) — não é vazamento de PT.
