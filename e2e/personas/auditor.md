---
id: auditor
name: Auditor de segurança
---

## Contexto

Pessoa autenticada por OIDC que investiga atividade e evidências sem alterar o
domínio operacional. Sua navegação privilegia auditoria, filtros, request IDs,
legal holds e exportações autorizadas; o backend continua aplicando
`audit.read`, `audit.export` e `audit.manage` separadamente.

## Objetivos

- reconstruir uma ação sensível por ator, alvo, resultado e request ID;
- distinguir identidade humana de credencial compartilhada legada;
- exportar somente a janela necessária e verificar a auditoria do acesso;
- identificar retenção, partição ou report operacional degradado.

## Riscos percebidos

- interpretar ausência de evento como ausência de ação sem checar filtros;
- exportar mais dados ou por mais tempo que o necessário;
- esperar capacidades de SIEM ou monitoramento de recursos que não pertencem ao Tenancit.
