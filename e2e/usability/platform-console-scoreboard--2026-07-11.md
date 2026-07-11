# Scoreboard — console operacional por persona

Data: 2026-07-11

## Escopo e critério

Rodada assistida sobre o console empacotado, complementada pelos fluxos
Playwright e pela matriz OIDC. Critério de aceite: nenhum blocker ou high aberto;
ações escondidas pela UI também precisam ser negadas pelo backend.

| Persona | Jornada observada | Evidência | Veredito |
| --- | --- | --- | --- |
| Administrador | criar tenant/API client, rotacionar, provisionar e repetir após timeout | `admin-idempotent-retry`, `admin-to-consumer-golden-path` | aprovado |
| Operador | acompanhar uso/saúde, localizar falha e evitar ações sem permissão | `operational-health-report`, matriz OIDC e teste de navegação | aprovado |
| Auditor | filtrar activity, paginar, criar hold e exportar janela controlada | `audit-operations`, `audit-export-lifecycle` | aprovado |
| Integrador | configurar webhook, observar retry/DLQ e consumir credencial sem reveal desnecessário | `webhook-delivery-lifecycle`, `api-client-token-lifecycle` | aprovado |

## Achados e resolução

| Severidade | Achado | Resolução |
| --- | --- | --- |
| medium | Navegação anterior misturava gestão, operação e segurança. | Grupos por capacidade e destinos filtrados pelas permissões da sessão. |
| medium | Preferências guardavam apenas visibilidade de colunas. | Visibilidade, ordenação e page size agora persistem por tabela, com reset explícito. |
| low | A persona de operador ainda descrevia OIDC como futuro. | Contexto atualizado para o RBAC OIDC entregue. |

## Resultado

- Blocker aberto: **0**
- High aberto: **0**
- Medium aberto: **0**
- Low aberto: **0**

A rodada não atribui ao console autoridade de autorização, não afirma monitorar
recursos cadastrados e não substitui homologação de um alvo de produção.
