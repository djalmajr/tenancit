# Tenancit

Tenancit é um control plane para configurar recursos de aplicações multi-tenant,
com resolução autenticada, criptografia de segredos e um painel administrativo.

O código é disponibilizado sob a [O'Saasy License](LICENSE): pode ser estudado,
usado, modificado e distribuído, mas não oferecido por terceiros como um SaaS
concorrente cuja proposta principal seja a funcionalidade do Tenancit. Por isso,
o projeto é **source available**, não open source segundo a definição da OSI.

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
| `TENANCIT_ADMIN_AUTH_MODE` | `oidc` em operação real; `legacy_shared_token` apenas em dev explícito |
| `TENANCIT_ADMIN_ORIGIN` | Origin público usado por callback e defesa CSRF no modo OIDC |
| `TENANCIT_OIDC_*` | Issuer, client confidencial, claim e mapeamento fechado de roles |
| `TENANCIT_ADMIN_TOKEN` | Token somente de dev legado ou break-glass OIDC explicitamente habilitado |
| `TENANCIT_AES_KEY` | Chave AES-256 atual, 32 bytes em base64 |
| `TENANCIT_AES_KEY_VERSION` | Versão numérica da chave atual |
| `TENANCIT_AES_KEY_V<n>` | Chave adicional do keyring para decrypt/cutover |

## Comandos usuais

```bash
make test          # Go, ESLint, typecheck e Vitest
make test-db       # mesmo gate Go, exigindo os testes PostgreSQL
make build         # SPA embutida no binário Go
make e2e-oidc      # Dex descartável: OIDC, sessão, CSRF, logout e break-glass
make sqlc          # regenera acesso tipado ao banco
make smoke         # smoke autenticado pós-deploy, com criação e cleanup
```

A documentação detalhada está em [`docs/README.adoc`](docs/README.adoc), e as
decisões arquiteturais em [`docs/adr/`](docs/adr/). O estado executável da
rodada está em [`docs/HANDOFF.md`](docs/HANDOFF.md).

Contribuições são bem-vindas conforme [CONTRIBUTING.md](CONTRIBUTING.md).
Vulnerabilidades devem ser relatadas de forma privada seguindo
[SECURITY.md](SECURITY.md).
Mudanças relevantes são registradas no [CHANGELOG.md](CHANGELOG.md), e releases
seguem o checklist reproduzível em [RELEASING.md](RELEASING.md).
