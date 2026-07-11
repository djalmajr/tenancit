---
id: directory-search-sort-pagination
name: Busca, ordenação e paginação das tabelas de tenants e chaves
reference: web/src/components/data-table/data-table.tsx
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `tenancit_admin_dev`
  - há registros suficientes para paginar (idealmente >= 12 tenants e >= 12 API clients ativos); se não houver, criar alguns durante o fluxo (custo de seed assumido)
design_refs:
  tenants-list: "planning/tenancit/proto/routes/tenants-list.js"
  api-clients: "planning/tenancit/proto/routes/api-clients.js"
---

## Objetivo do usuário

Localizar rapidamente um registro específico em listas grandes usando busca, ordenação por coluna e paginação, sem perder dados nem quebrar a tabela.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`auth`) Autenticar-se com `tenancit_admin_dev` → painel acessível.
2. (`tenants-list`) Abrir **Tenants**; se a lista for curta, criar tenants suficientes para passar de uma página → a tabela exibe controles de paginação e o indicador de página/itens.
3. (`tenants-list`) Usar os controles **próxima / última / anterior / primeira página** → a navegação respeita os limites (não passa do início/fim) e o indicador atualiza.
4. (`tenants-list`) Alterar **linhas por página**, recarregar e usar **Restaurar tabela** → a preferência sobrevive ao reload e o reset recupera o padrão explícito.
5. (`tenants-list`) Clicar no cabeçalho **Nome** para ordenar asc, desc e resetar; repetir em **Slug** e **Status** → a ordenação tri-state funciona por coluna.
6. (`tenants-list`) Digitar um termo na **busca** (nome, slug ou status) → a lista filtra pelos campos correspondentes; limpar a busca → a lista volta ao conjunto completo.
7. (`api-clients`) Abrir **Chaves de API** → a tabela já vem ordenada por data de criação (desc); ordenar por **Nome**, **Criado em** e **Status**, paginar e buscar por nome/data/status → todos os controles funcionam de forma equivalente.

## Resultado esperado

As tabelas suportam busca multi-campo, ordenação tri-state por coluna e paginação com limites corretos; preferências persistem por tabela e possuem reset, mantendo a integridade dos dados exibidos.

## Estado atual × design

- As tabelas usam `web/src/components/data-table/data-table.tsx` + `web/src/hooks/use-data-table.ts`.
- Tenants vem ordenada por **Nome** asc por padrão; API Clients por **Criado em** desc; a tabela de API Clients mantém chaves **ativas e revogadas** para permitir auditoria e reativação.
- A busca usa `globalFilterFn` específico de cada tela (nome/slug/status em tenants; nome/status/data em chaves). A API não devolve `key_preview`; por isso o fallback visual mascarado `tnc_••••••••` não é apresentado como filtro pesquisável.
