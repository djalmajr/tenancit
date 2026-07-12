# Registro do primeiro ambiente representativo

**Status:** AGUARDANDO ALVO

Este documento é o registro sanitizado da História 05 do epic 04. Ele só muda
para **VALIDADO** depois que todos os gates abaixo forem executados no mesmo
ambiente e digest. Não registre credenciais, DSNs, cookies, tokens, chaves,
CIDRs privados, nomes internos ou payloads operacionais.

## Identificação não secreta

| Campo | Evidência |
| --- | --- |
| Ambiente e owner operacional | Pendente |
| Plataforma/orquestrador | Pendente |
| Data/janela de validação | Pendente |
| Commit e digest imutável | Pendente |
| URL pública sanitizada | Pendente |
| Número de réplicas | Pendente |
| PostgreSQL/Valkey/IdP gerenciados ou próprios | Pendente |

## Entrada mínima necessária

O operador precisa disponibilizar no ambiente, sem copiá-los para este arquivo:

- alvo de deploy, DNS e ingress TLS;
- referências do secret manager para os cinco logins PostgreSQL, Valkey, OIDC,
  keyring AES, OTLP e credenciais operacionais;
- issuer, client, audiences, claim de roles e mappings fechados do IdP;
- owners e valores aprovados de retenção, SLO, RPO e RTO;
- tenant/hostname sentinela e API client de smoke com `tenant:identify`.

## Gates

| Gate | Comando/cenário | Aceite | Evidência |
| --- | --- | --- | --- |
| Preflight | `make deploy-preflight` | digest, TLS, DSNs e backup fresh aceitos | Pendente |
| Migrations | `/migrate` com login owner | runtime permanece sem DDL e ready | Pendente |
| OIDC/RBAC | login/logout, roles, CSRF e sessão revogada | claims e deny-by-default confirmados | Pendente |
| HTTP/TLS | headers, cookies, origins e forwarded headers | negativos bloqueados; proxy confiável explícito | Pendente |
| Duas réplicas | tráfego alternado e parada de uma réplica | atendimento continua e revogação é imediata | Pendente |
| Valkey | bucket combinado, restart e indisponibilidade | limite global; falha fechada `503` | Pendente |
| Backup/restore | backup off-host e restore drill | checksum, smoke, RPO e RTO aprovados | Pendente |
| Rollback | voltar ao digest anterior | dados preservados e readiness restaurada | Pendente |
| Rewrap | inventário + dry-run no restore | versões/contagens batem; nenhuma escrita | Pendente |
| Observabilidade | RED/USE, reports e alertas | sem secrets; SLOs e owners acionáveis | Pendente |
| Retenção | jobs, legal holds e expiração | política aprovada e observável | Pendente |

## Resultado e desvios

- Resultado geral: **PENDENTE**.
- Desvios aceitos: nenhum.
- Ações corretivas: pendentes após a primeira execução.
- Links para logs/métricas sanitizados: pendentes.

## Aprovação

O registro só pode ser fechado quando owner técnico e owner operacional
confirmarem os gates, e `docs/HANDOFF.md` registrar ambiente, digest, data,
resultados e eventuais riscos residuais sem material confidencial.
