---
id: first-run-empty-states
name: Primeiro acesso em sistema vazio e estados vazios das telas
reference: docs/produto/03-jornadas-operacionais.adoc#cadastrar-um-tenant
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento com banco vazio (sem tenants, definitions ou API clients)
  - token administrativo válido disponível: `tenancit_admin_dev`
design_refs:
  overview: "planning/tenancit/proto/routes/overview.js"
  tenants-list: "planning/tenancit/proto/routes/tenants-list.js"
  definitions-list: "planning/tenancit/proto/routes/definitions-list.js"
  api-clients: "planning/tenancit/proto/routes/api-clients.js"
  tenant-detail: "planning/tenancit/proto/routes/tenant-detail.js"
---

## Objetivo do usuário

Entrar pela primeira vez num ambiente recém-provisionado e entender, em cada tela, que ainda não há dados e qual é o próximo passo — sem telas quebradas nem áreas em branco sem orientação.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`auth`) Acessar o painel e autenticar com `tenancit_admin_dev` → a visão geral carrega sem erro mesmo sem nenhum dado.
2. (`overview`) Conferir os cards de KPI → **Tenants ativos**, **Domínios**, **Recursos provisionados** e **Definições ativas** exibem `0` sem layout quebrado.
3. (`overview`) Revisar o card **"Tenants"** → aparece a mensagem de estado vazio (nenhum tenant cadastrado) em vez de uma lista vazia silenciosa.
4. (`shell`) Clicar em **Tenants** na sidebar → a tabela mostra o estado **"sem resultados"** e o botão **Novo tenant** continua visível e acionável.
5. (`shell`) Clicar em **Recursos** na sidebar → **ATENÇÃO/ACHADO POTENCIAL**: a grade de definitions pode aparecer totalmente em branco sem nenhum texto orientando criar a primeira definição. Registrar como achado se não houver estado vazio explícito (lista, CTA ou mensagem).
6. (`shell`) Clicar em **Chaves de API** na sidebar → a tabela mostra o estado vazio (nenhuma chave ativa) e o alerta sobre segredos em claro / snippet de `/v1/resolve` continua visível.
7. (`tenants-list`) Criar o primeiro tenant com slug único → a aplicação navega para o detalhe do tenant.
8. (`tenant-detail`) Revisar o card **"Prontidão para consumo"** → tenant ativo aparece satisfeito, enquanto domínios, recursos ativos e chaves de API aparecem como pendências (sinal de atenção), orientando os próximos passos.
9. (`tenant-detail`) Conferir as abas **Recursos** e **Domínios** vazias → cada aba mostra mensagem própria de estado vazio descrevendo o que falta provisionar.

## Resultado esperado

Um operador que nunca usou o sistema consegue, só pela UI, entender que está vazio e o que fazer em seguida: cada listagem tem estado vazio legível e a prontidão do tenant guia a configuração mínima. Qualquer tela que apareça em branco sem orientação é um achado de implementação faltante.

## Estado atual × design

- Overview consome `GET /v1/admin/overview` e tem mensagem de vazio para o card de tenants (`overview.emptyTenants`).
- Tenants e API Clients usam `DataTable` com rótulo de `noResults`.
- `web/src/routes/definitions.tsx` renderizava apenas a grade de cards, **sem estado vazio explícito** quando não havia definitions (área totalmente em branco). **Corrigido em 2026-06-26** — ver "Resultado da execução" abaixo.
- O card de prontidão em `tenant-detail.tsx` é o guia de configuração mínima por tenant.

## Resultado da execução

**Data:** 2026-06-26 · **Persona:** platform-operator · **Resultado:** achado confirmado e corrigido.

**Ambiente:** stack TenancitIO atual — SPA via Vite em `:5180` (preview builtin), API Go atual em `:8087` (lê `TENANCIT_*`), Postgres vazio dedicado em `:5433` (DB `tenancit`, migrations aplicadas no boot). Login com `tenancit_admin_dev`.

**Passos validados (1–6):**

| Passo | Tela | Resultado |
|---|---|---|
| 1 `auth` | Login | ✅ Visão geral carrega sem erro com DB vazio |
| 2 `overview` | KPIs | ✅ Tenants `0`, Domínios `0`, Recursos `0`, Definições `0` — sem layout quebrado |
| 3 `overview` | Card Tenants | ✅ "Nenhum tenant ainda." |
| 4 `shell` | Tenants | ✅ DataTable com "Nenhum tenant encontrado." + "Novo tenant" acionável |
| 5 `shell` | **Recursos/Definições** | ❌→✅ **ACHADO**: grade em branco sem orientação. **Corrigido.** |
| 6 `shell` | Chaves de API | ✅ DataTable com "Nenhuma chave ativa." |

**Passos validados (7–9)** — percorridos após a correção, sem novos achados:

| Passo | Tela | Resultado |
|---|---|---|
| 7 `tenants-list` | Criar tenant | ✅ "Acme Corp" (slug `acme`) criado → navega para `/tenants/{id}` |
| 8 `tenant-detail` | Prontidão para consumo | ✅ **Tenant: ativo** (satisfeito); **Domínios** "adicione um domínio", **Recursos ativos** "adicione um recurso ativo", **Chaves ativas** "crie uma chave em Chaves de API" — pendências sinalizadas |
| 9 `tenant-detail` | Abas vazias | ✅ **Recursos**: "Nenhum recurso — Este tenant ainda não tem recursos provisionados." · **Domínios**: "Nenhum domínio. Adicione ao menos um para resolver o tenant." |

**Conclusão:** flow 100% percorrido. Único achado de implementação foi a ausência de estado vazio em `definitions.tsx` (passo 5), agora corrigida. Todas as demais telas de primeiro uso têm estados vazios legíveis e o card de prontidão guia a configuração mínima do tenant.

**Achado (passo 5):** em [definitions.tsx](../../web/src/routes/definitions.tsx) a lista mapeava `defs.map(...)` numa grade sem branch para `defs.length === 0`; com zero definições o usuário via apenas título + botão "Nova definição" sobre uma área em branco — inconsistente com todas as outras listas (que usam `DataTable` com `noResults`) e até com [definition-detail.tsx](../../web/src/routes/definition-detail.tsx) (que já tratava `length === 0`).

**Correção aplicada:**
- Nova chave i18n `definitions.empty` em pt-BR / en-US / es-ES (`web/src/lib/i18n.tsx`).
- Branch `defs.length === 0` em `definitions.tsx`: container tracejado com ícone, mensagem e CTA "Nova definição" (abre o mesmo diálogo de criação).

**Verificação da correção:** `tsc --noEmit` ✅ · empty state renderiza nos 3 locales (pt "Nenhuma definição ainda." / en "No definitions yet." / es "Aún no hay definiciones.") ✅ · CTA abre o diálogo "Nova definição" e fecha com Esc ✅.
