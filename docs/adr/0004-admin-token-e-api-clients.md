# ADR 0004 — Separação entre admin token e API clients de consumo

- **Status:** Aceito
- **Data:** 2026-06-23

## Contexto

O serviço tem duas superfícies com riscos diferentes:

- admin: cria tenants, definitions, API clients e pode revelar secrets;
- consumo: resolve recursos ativos por hostname para aplicações.

Usar a mesma credencial para ambos misturaria bootstrap operacional com acesso
server-to-server. Deixar admin sem autenticação torna `?reveal=true` e CRUD
administrativo públicos para qualquer agente que alcance a porta.

## Decisão

Separar credenciais:

- `/v1/admin/*` exige `Authorization: Bearer <RT_ADMIN_TOKEN>`;
- `/v1/resolve*` exige API client persistido em `api_clients`;
- API clients guardam apenas hash SHA-256;
- o token bruto de API client aparece uma única vez na criação.

`RT_ADMIN_TOKEN` é obrigatório no boot e deve vir de secret manager ou variável
de ambiente segura.

## Consequências

Positivas:

- admin deixa de ser uma superfície aberta;
- bootstrap de API clients não depende de um client já existente;
- revogação de consumers não afeta o acesso admin;
- fica claro que painel humano e consumo server-to-server são fronteiras
  distintas.

Custos:

- operadores precisam configurar o token no painel ou em scripts;
- login humano completo continua fora do escopo atual;
- `RT_ADMIN_TOKEN` precisa ser rotacionado por processo externo.

## Status

Aceito em 2026-06-23. Fonte de verdade: `server/internal/httpapi/auth.go`,
`server/internal/httpapi/server.go` e `web/src/lib/api.ts`.
