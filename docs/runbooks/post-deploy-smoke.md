# Smoke pós-deploy

**Status:** modo legado mutável VALIDADO em Compose dev local em 2026-07-09;
contrato OIDC não mutável validado localmente em 2026-07-11 e pendente no alvo.

O smoke legado comprova mais que liveness: rejeição sem credencial, acesso admin,
criação de tenant/domínio, `identify`, resolve por `tenantId`, ETag/304,
headers de segurança e cleanup.

Ele não é autorizado em produção OIDC: não se promove bearer administrativo
para automatizar mutações. O release usa `post-deploy-production-smoke.sh`, que
valida OIDC e identifica um tenant sentinela sem modificar domínio.

## Compose local legado

### Pré-condições

- `curl` e `jq`;
- um token admin válido;
- um API client ativo dedicado a smoke ou operação (o script nunca o imprime).

### Execução

```bash
TENANCIT_BASE_URL=https://tenancit.example.com \
TENANCIT_ADMIN_TOKEN='...' \
TENANCIT_SMOKE_API_TOKEN='...' \
make smoke
```

O script usa slug/hostname únicos, registra cleanup em `trap` e exclui o tenant
ao final. Ele não cria definitions nem deixa recursos permanentes.

### Resultado esperado

```text
smoke ok: health, auth boundaries, create, identify, resolve, ETag, cleanup
```

Falha em qualquer oráculo interrompe o procedimento com exit diferente de zero.
Não use `set -x`: isso exporia headers de autorização.

## Produção OIDC não mutável

Pré-provisione um tenant/hostname sentinela e uma chave dedicada contendo apenas
`tenant:identify`. Depois:

```bash
TENANCIT_BASE_URL='https://tenancit.example.com' \
TENANCIT_SMOKE_API_TOKEN='...' \
TENANCIT_SMOKE_HOSTNAME='smoke.example.com' \
TENANCIT_SMOKE_TENANT_SLUG='smoke' \
./scripts/post-deploy-production-smoke.sh
```

O script valida `/healthz`, `/readyz`, modo e redirect OIDC, rejeições 401 e o
slug retornado por `identify`. O token é passado ao `curl` por arquivo de header
temporário com permissão restrita, não por argumento visível. Nenhuma rota admin
mutável é chamada.
