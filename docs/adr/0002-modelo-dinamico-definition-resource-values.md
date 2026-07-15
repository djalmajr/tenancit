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

Cada `tenant_resource` possui um alias único no tenant. Um tenant pode instanciar
a mesma definition várias vezes. A instância pode ser independente ou apontar
para outra instância independente, da mesma definition e do mesmo tenant:

- campos sem valor local herdam o valor atual da origem;
- campos com valor local são overrides;
- remover um override retoma a herança;
- duplicar cria um snapshot independente, sem vínculo futuro;
- uma origem com dependentes vinculados não pode ser removida.

O vínculo é restrito a uma origem independente para evitar cadeias e ciclos. O
alias, e não a definition, é a identidade usada para resolver uma instância
específica pela Consumer API.

## Consequências

Positivas:

- novo tipo de recurso é dado administrativo, não migration;
- fields obrigatórios e secretos continuam explícitos;
- consultas e constraints seguem relacionais;
- consumidores recebem shape simples com `alias`, `definitionKey` e values efetivos;
- configurações comuns podem ser reaproveitadas sem acoplar o domínio a uma ideia de aplicação.

Custos:

- listagens precisam montar definition + fields + values;
- mudanças incompatíveis de contrato devem virar nova `definition.key`;
- queries agregadas precisam ser desenhadas explicitamente para evitar N+1.
- alterações na origem também precisam invalidar o ETag dos recursos vinculados.

## Status

Aceito em 2026-06-23 e ampliado em 2026-07-15 com aliases e herança explícita.
Fonte de verdade: `server/migrations/00001_init.sql`,
`server/migrations/00012_shared_resources.sql` e
`server/internal/store/queries/*.sql`.
