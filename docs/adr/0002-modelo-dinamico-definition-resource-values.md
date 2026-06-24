# ADR 0002 — Modelo dinâmico com definitions, resources e values

- **Status:** Aceito
- **Data:** 2026-06-23

## Contexto

O serviço precisa guardar recursos heterogêneos por tenant: banco, storage,
Keycloak, SMTP e outros tipos futuros. Um modelo rígido com colunas por recurso
forçaria migrations para cada novo tipo. Um blob JSON único reduziria
validação, visibilidade e constraints relacionais.

## Decisão

Usar um modelo dinâmico normalizado:

- `resource_definitions` define o tipo de recurso;
- `resource_fields` define os campos esperados;
- `tenant_resources` instancia uma definition para um tenant;
- `tenant_resource_values` guarda os valores dos fields.

O schema garante no máximo um resource ativo por `(tenant, definition)` com
índice único parcial.

## Consequências

Positivas:

- novo tipo de recurso é dado administrativo, não migration;
- fields obrigatórios e secretos continuam explícitos;
- consultas e constraints seguem relacionais;
- consumidores recebem shape simples por `definitionKey`.

Custos:

- listagens precisam montar definition + fields + values;
- mudanças incompatíveis de contrato devem virar nova `definition.key`;
- queries agregadas precisam ser desenhadas explicitamente para evitar N+1.

## Status

Aceito em 2026-06-23. Fonte de verdade: `server/migrations/00001_init.sql` e
`server/internal/store/queries/*.sql`.
