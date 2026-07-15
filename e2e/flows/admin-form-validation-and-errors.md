---
id: admin-form-validation-and-errors
name: Validação de formulários e erros de unicidade no painel admin
reference: docs/developers/03-contratos-http.adoc#erros-esperados
persona: platform-operator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `tenancit_admin_dev`
  - usar valores únicos por rodada para os casos de sucesso e reaproveitar os mesmos para forçar conflito
design_refs:
  tenants-list: "planning/tenancit/proto/routes/tenants-list.js"
  tenant-detail: "planning/tenancit/proto/routes/tenant-detail.js"
  definitions-list: "planning/tenancit/proto/routes/definitions-list.js"
---

## Objetivo do usuário

Receber feedback claro e recuperável quando envia dados inválidos ou em conflito (slug/key/hostname duplicados, campos obrigatórios ausentes), sem ficar com a UI presa ou em estado parcial.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`auth`) Autenticar-se com `tenancit_admin_dev` → painel acessível.
2. (`tenants-list`) Abrir **Novo tenant** e deixar Nome e Slug vazios → o botão **Criar tenant** permanece desabilitado; ao preencher ambos, habilita.
3. (`tenants-list`) Criar um tenant com slug `dup-e2e`; abrir **Novo tenant** de novo e tentar criar outro com o mesmo slug `dup-e2e` → a criação falha (conflito de unicidade, 409), a mensagem de erro aparece dentro do diálogo e o diálogo **não** navega nem fecha.
4. (`tenants-list`) Clicar em **Cancelar** no diálogo → fecha sem criar nada e sem efeito colateral.
5. (`definitions-list`) Criar uma definition com key `dupkey-e2e`; tentar criar outra com a mesma key → o erro aparece no diálogo (conflito/payload inválido) e não há navegação.
6. (`tenant-detail`) Num tenant, abrir aba **Domínios**, adicionar o hostname `dup.e2e.local`; tentar adicionar o **mesmo** hostname novamente (no mesmo tenant ou em outro) → falha por unicidade (409) com mensagem visível no diálogo de domínio.
7. (`tenant-detail`) Abrir **Adicionar recurso**, escolher uma definition já usada pelo tenant e informar um alias existente → o diálogo mantém os dados, sinaliza conflito de alias e não cria uma segunda linha ambígua.
8. (`tenant-detail`) Informar um alias fora do padrão ou deixar um campo obrigatório vazio → **Salvar recurso** permanece desabilitado; campos do tipo `int` aceitam número e campos secretos aparecem mascarados no formulário.

## Resultado esperado

Todo caminho inválido produz um erro visível e em contexto (no próprio diálogo), botões desabilitados evitam submits vazios, e conflitos de unicidade (409) são comunicados sem deixar a UI em estado quebrado ou parcialmente aplicado.

## Estado atual × design

- A SPA propaga erros via `setError(String(e))`, exibindo o texto cru do backend (ex.: `409: ...`). Avaliar se a mensagem é legível o suficiente para o operador — texto excessivamente técnico é um achado de usabilidade.
- O contrato HTTP define 400 (payload/obrigatório), 401 (token), 404 (consumo) e 409 (unicidade de slug, hostname e alias normalizado) em `docs/developers/03-contratos-http.adoc`.
- Um tenant pode ter várias instâncias da mesma definition; o alias único, e não o tipo, diferencia cada configuração.
