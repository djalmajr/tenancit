# Decisões — Resource Tenant Service

Registro das decisões tomadas na sessão de grilling (2026-05-30). Cada uma com a
opção escolhida e o porquê. Detalhe técnico em [design.md](./design.md).

| # | Tema | Decisão | Racional |
|---|------|---------|----------|
| 1 | **Fronteira de dados** | Serviço **independente** com tabelas próprias | Pivot: deixa de ser port do front-manager-api; vira serviço de configuração multi-tenant autônomo |
| 2 | **Modelo de domínio** | **Tenant + Template/Definition + ResourceTenant + Values** (dinâmico) | Flexível — novo tipo de recurso = novo template, sem migration. Encaixa no caso DB/MinIO/etc por tenant |
| 3 | **Distribuição** | **REST síncrona + cache no cliente** | Simplicidade; Kafka/MinIO eram só p/ compatibilidade legada. Reintroduzível depois |
| 4 | **Criptografia** | **AES-256-GCM**, chave externalizada | Compatibilidade caiu → sem motivo p/ manter DES/PBE/MD5 quebrado e senha embutida |
| 5 | **Fronteira de decrypt** | **Decrypt server-side**, retorna em claro via TLS | Chave fica só no serviço; consumidor não lida com cripto; viabiliza rotação |
| 6 | **Stack** | **Go** | Argumento de reuso da cripto legada evaporou; Go ganha em robustez/footprint no hot path |
| 7 | **Tooling Go** | **chi + pgx + sqlc + goose** | Idiomático, SQL explícito (sem ORM mágico), type-safe |
| 8 | **Resolução de tenant** | **Hostname exato → 1 tenant** | Determinístico e simples; tabela `tenant_domains` deixa porta aberta p/ N domínios |
| 9 | **Auth** | **API key / token de serviço + TLS** | Simples de operar, sem dependência de IdP no hot path |
| 10 | **Contrato de leitura** | **Ambos** (todos os recursos + atalho por tipo) | Endpoint geral serve cache no cliente; atalho serve uso direto (DB connection) |
| 11 | **Unicidade** | **Só 1 ativo por tenant+template** | Resolução determinística; índice único parcial garante invariante |
| 12 | **Admin/validação** | **CRUD completo + validação contra template** | Mantém integridade (required/secret); novo tipo sem deploy |

## Recomendações registradas (não bloqueantes)

- **Lifecycle**: soft-delete via `status` (espelha `disable` legado).
- **Mascaramento admin**: `?reveal=true` com escopo de auth; default mascarado.
- **Tokens de serviço**: tabela `api_clients` com hash (rotação/auditoria) em vez de
  token fixo em env.
- **Rotação de chave AES**: `key_version` já previsto no schema.

## Próximo passo

`/agile-epic` — decompor em stories: bootstrap do serviço Go (chi/pgx/sqlc/goose),
modelo de dados + migrations, módulo de criptografia (AES-GCM), API de consumo
(resolve por hostname), API admin (CRUD + validação), auth (API key + TLS),
observabilidade/deploy.
