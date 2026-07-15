# Scoreboard — catálogo UX no Browser integrado

Data: 2026-07-15

## Escopo

Rodada manual e sequencial no Browser integrado sobre os 25 fluxos catalogados.
Nenhum bloqueio de fixture foi convertido artificialmente em aprovação; ações destrutivas
ou dependentes de infraestrutura externa foram preservadas para ambientes descartáveis.

| Resultado | Quantidade | Fluxos |
| --- | ---: | --- |
| Pass | 2 | `api-client-usage-audit`, `operational-health-review` |
| Needs UX fix | 1 | `admin-invalid-token-recovery` |
| Partial | 13 | auth/overview, formulários, golden path, governança/lifecycle de chaves, auditoria, definitions, i18n, teclado, settings, tenant e recurso independente |
| Blocked por pré-condição | 9 | sessões, retry HTTP, rate limit, consumer API, paginação, first-run, mobile, herança e webhooks |

## Achados priorizados

| Severidade | Fluxo | Evidência | Recomendação |
| --- | --- | --- | --- |
| medium | `admin-invalid-token-recovery` | Após token inválido, o campo foi limpo sem mensagem de erro visível. | Exibir erro localizado, associá-lo ao campo e devolver foco adequadamente. |
| medium | `responsive-mobile-navigation` | A capability solicitou 390×844, mas a página permaneceu em 1280×720. | Revalidar em runtime com emulação funcional; não classificar como defeito do app ainda. |
| low | dados E2E | Definitions antigas permanecem no catálogo e vários fluxos críticos não têm fixtures dedicadas. | Criar seed descartável por jornada e limpeza determinística. |

## Cobertura por persona

| Persona | Resultado da rodada |
| --- | --- |
| Administrador | Login, overview, tenants, recursos, chaves, auditoria e settings percorridos; mutations destrutivas ficaram em fixtures pendentes. |
| Operador | Saúde e uso mensal completáveis e coerentes; relatórios antigos explicam o estado degradado. |
| Auditor | Busca e tabela de eventos completáveis; export/hold requerem fixture isolada. |
| Integrador | Catálogo visual está claro, mas Consumer API, limiter e webhooks precisam de credenciais/receiver descartáveis. |

## Conclusão

A UI principal é navegável e consistente no desktop. O único defeito de UX observado
diretamente foi a ausência de feedback visível para token inválido. A maior pendência
da rodada é operacional: preparar dados descartáveis para provar lifecycle, herança,
multi-página, sessões, webhooks e rate limit sem alterar dados compartilhados.
