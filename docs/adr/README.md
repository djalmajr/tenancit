# ADRs — Architecture Decision Records (Tenancit)

Índice das decisões arquiteturais do **Tenancit**. Comece por
aqui para entender por que o serviço tem esse formato, não apenas como ele
funciona.

## O que é um ADR

Um ADR registra uma decisão estrutural: contexto, decisão tomada e
consequências. ADR aceito não deve ser reescrito para refletir uma nova decisão;
quando a arquitetura mudar, crie um ADR novo que supersede o anterior.

## Convenções

- Numeração sequencial com quatro dígitos: `0001`, `0002`, ...
- Nome do arquivo: `NNNN-titulo-em-kebab-case.md`.
- Status possíveis: `Proposto`, `Aceito`, `Superado por NNNN`, `Descontinuado`.
- Escopo: arquitetura, fronteiras, segurança, modelo de dados e stack.

## Índice de ADRs

| # | Título | Status |
|---|---|---|
| [0001](./0001-servico-autonomo-go-sqlc-spa-embutida.md) | Serviço autônomo em Go, sqlc e SPA embutida | Aceito |
| [0002](./0002-modelo-dinamico-definition-resource-values.md) | Modelo dinâmico com definitions, resources e values | Aceito |
| [0003](./0003-secrets-server-side-aes-gcm.md) | Secrets cifrados com AES-GCM e decrypt server-side | Aceito |
| [0004](./0004-admin-token-e-api-clients.md) | Separação entre admin token e API clients de consumo | Aceito |

## Template

```markdown
# ADR NNNN — Título

- **Status:** Proposto | Aceito | Superado por NNNN | Descontinuado
- **Data:** AAAA-MM-DD

## Contexto

O que motivou a decisão? Quais alternativas foram consideradas?

## Decisão

A decisão tomada, em voz ativa.

## Consequências

Trade-offs, custos e benefícios.

## Status

Histórico de transição do ADR.
```

## Documentos irmãos

- [Documentação geral](../README.adoc)
- [Engenharia — Arquitetura](../engenharia/01-arquitetura.adoc)
- [Engenharia — Segurança e criptografia](../engenharia/04-seguranca-e-criptografia.adoc)
