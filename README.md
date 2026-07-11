# Tenancit

Tenancit é um control plane para configurar recursos de aplicações multi-tenant,
com resolução autenticada, criptografia de segredos e um painel administrativo.

## Requisitos

- Go 1.25
- Bun 1.3
- Docker com Compose

## Início rápido

Para subir o produto empacotado em `http://localhost:8080`:

```bash
docker compose up --build
```

Para desenvolvimento com Vite, HMR e proxy same-origin (`http://localhost:5180`),
com a API em `http://localhost:8081`:

```bash
make dev-compose
```

Os valores presentes nos arquivos Compose são exclusivamente de demonstração.
Substitua tokens e chaves antes de qualquer implantação real.

## Configuração

Copie [`.env.example`](.env.example) para `.env` somente em ambiente local. O
servidor falha de forma explícita quando uma configuração obrigatória não existe.

| Variável | Uso |
| --- | --- |
| `TENANCIT_ADDR` | Endereço do servidor HTTP; default `:8080` |
| `TENANCIT_DATABASE_URL` | DSN PostgreSQL |
| `TENANCIT_ADMIN_TOKEN` | Bearer token da API administrativa |
| `TENANCIT_AES_KEY` | Chave AES-256 atual, 32 bytes em base64 |
| `TENANCIT_AES_KEY_VERSION` | Versão numérica da chave atual |
| `TENANCIT_AES_KEY_V<n>` | Chave adicional do keyring para decrypt/cutover |

## Comandos usuais

```bash
make test          # Go, ESLint, typecheck e Vitest
make test-db       # mesmo gate Go, exigindo os testes PostgreSQL
make build         # SPA embutida no binário Go
make sqlc          # regenera acesso tipado ao banco
make smoke         # smoke autenticado pós-deploy, com criação e cleanup
```

A documentação detalhada está em [`docs/README.adoc`](docs/README.adoc), e as
decisões arquiteturais em [`docs/adr/`](docs/adr/). O estado executável da
rodada está em [`docs/HANDOFF.md`](docs/HANDOFF.md).
