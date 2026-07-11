# ADR 0008 — Fronteira entre core e extensões

- **Status:** Aceito
- **Data:** 2026-07-11

## Contexto

O Tenancit acumulou recursos de governança, operação e integração além do
catálogo inicial de tenants. Manter tudo indivisível aumenta acoplamento e o
custo de adoção; extrair invariantes de segurança como plugins, por outro lado,
permitiria combinações inseguras e contratos incompatíveis.

## Decisão

Uma capacidade pertence ao **core** quando preserva um invariante do produto:
isolamento e identidade de tenant, consistência do catálogo, proteção de
secrets, autenticação/autorização, auditoria de mutações, contrato de consumo,
migrations ou segurança operacional necessária para executar esses contratos.

Uma capacidade é **extensão/adaptador** quando conecta o core a um provedor,
destino ou apresentação opcional e pode falhar ou ser removida sem corromper os
invariantes acima.

### Matriz da implementação atual

| Capacidade | Destino | Justificativa |
|---|---|---|
| tenants, domains, definitions, resources e resolve/identify | Core | Modelo e contrato que definem o produto |
| criptografia, keyring, rewrap e redação de secrets | Core | Invariante de confidencialidade e rotação |
| API clients, scopes, expiração e lifecycle | Core | Fronteira server-to-server |
| principal admin, sessões, RBAC e auditoria transacional | Core | Autoria e autorização das mutações |
| uso básico, health/readiness e interfaces de rate limit | Core | Operabilidade mínima do contrato seguro |
| backend Valkey do rate limiter | Adaptador distribuído incluído | Implementa uma interface core; outro backend pode substituí-lo |
| descoberta OIDC e mapeamento específico de claims | Adaptador de identidade incluído | Sessão/RBAC são core; o provedor é substituível |
| outbox e catálogo de eventos | Core | Garante evento atômico com a mutação |
| entrega HTTP de webhooks, retry e circuit breaker | Extensão incluída | Um transporte opcional sobre o outbox |
| Slack, Teams, Discord, SIEM e filas externas | Extensões futuras | Destinos específicos, sem regra de domínio |
| relatórios operacionais recebidos de backup/deploy | Extensão operacional incluída | Integra jobs externos; não decide disponibilidade do recurso do tenant |
| exportadores de auditoria e archive sinks | Adaptadores | Retenção/auditoria base permanecem no core |
| console administrativo | Distribuição oficial | Cliente do core, substituível pelas APIs |

“Extensão incluída” significa que o código ainda pode viver no monorepo e no
binário oficial, mas deve depender de portas estreitas. Não significa que o
recurso seja obrigatório em toda implantação.

Não será criado agora um runtime genérico de plugins. Plugins Go carregados em
processo não serão usados por causa de portabilidade, isolamento e risco de
compartilhar memória/segredos. Novos provedores usarão, nesta ordem:

1. interfaces Go e composição em build para adaptadores oficiais;
2. webhooks ou APIs HTTP/gRPC versionadas para extensões externas;
3. processos/sidecars separados quando houver código de terceiros.

Extrações físicas só ocorrerão quando existir ao menos um segundo provedor ou
uma necessidade real de distribuição independente. Até lá, a fronteira será
mantida por interfaces e dependências direcionais, evitando uma plataforma de
plugins especulativa.

## Consequências

- segurança, isolamento e consistência não podem ser desabilitados por plugin;
- o binário oficial pode continuar oferecendo integrações úteis por padrão;
- provedores específicos podem evoluir sem contaminar o modelo de domínio;
- algumas implementações atuais serão refatoradas gradualmente para portas,
  sem migração destrutiva nem remoção imediata de funcionalidades;
- extensões de terceiros não recebem acesso direto ao banco ou às chaves AES.
