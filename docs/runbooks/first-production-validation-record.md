# Registro do primeiro ambiente representativo

**Status:** AMBIENTE PESSOAL VALIDADO; PRODUÇÃO DE CLIENTE REQUER GATE PRÓPRIO

Este documento é o registro sanitizado da História 05 do epic 04. Ele só muda
para **VALIDADO** depois que todos os gates abaixo forem executados no mesmo
ambiente e digest. Não registre credenciais, DSNs, cookies, tokens, chaves,
CIDRs privados, nomes internos ou payloads operacionais.

## Identificação não secreta

| Campo | Evidência |
| --- | --- |
| Ambiente e owner operacional | laboratório pessoal; mantenedor do projeto |
| Plataforma/orquestrador | K3s single-node em VM Hetzner |
| Data/janela de validação | 2026-07-11 |
| Commit e digest imutável | commit `b9e37e2`; digest `sha256:b9571461...57f1fa1` |
| URL pública sanitizada | `https://tenancit.djalmajr.dev` |
| Número de réplicas | 2 réplicas do app + 1 worker |
| PostgreSQL/Valkey/IdP gerenciados ou próprios | PostgreSQL, Valkey e Dex isolados no namespace de teste |

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
| Preflight | chart lint + render + dry-run server | digest e objetos aceitos | Aprovado |
| Migrations | `/migrate` com login owner | runtime permanece sem DDL e ready | Aprovado |
| OIDC/RBAC | login real no Dex e sessão | claim `email` para `security_admin` | Aprovado no laboratório |
| HTTP/TLS | certificados e origins HTTPS | TLS público e issuer coerentes | Aprovado |
| Duas réplicas | 30 probes durante remoção de pod | nenhuma resposta diferente de `200` | Aprovado |
| Valkey | 5 RPM + indisponibilidade controlada | 5x `200`, 5x `429`; readiness `503/200` | Aprovado |
| PostgreSQL | indisponibilidade e recuperação | readiness `503/200`; dados preservados | Aprovado |
| Backup/restore | dump, checksum e restore isolado | 130.306 B; 30 tabelas; 1 tenant | Aprovado para teste |
| Rollback | revisão Helm 5 -> 4 -> imagem atual | readiness `200`; 1 tenant preservado; roll-forward concluído | Aprovado |
| Rewrap | `--dry-run --target-version 1` | nenhuma escrita | Aprovado |
| Observabilidade | health/readiness e logs JSON | sem secrets na evidência | Aprovado no laboratório |
| Retenção | workers ativos e settings versionados | execução disponível | Aprovado funcionalmente |

## Resultado e desvios

- Resultado geral: **VALIDADO COMO LABORATÓRIO REAL**, não homologação de uma
  topologia de cliente.
- Desvios aceitos: single-node, dependências single-instance, Dex estático e
  backup no host do operador.
- Ações para clientes: IdP corporativo, HA por nó/zona, backup off-site,
  observabilidade e SLO/RPO/RTO próprios.
- Evidência executável: `docs/runbooks/kubernetes-personal-validation.md`.

## Aprovação

O registro só pode ser fechado quando owner técnico e owner operacional
confirmarem os gates e este documento registrar ambiente, digest, data,
resultados e eventuais riscos residuais sem material confidencial.
