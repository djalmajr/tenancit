# Smoke pós-deploy

**Status:** VALIDADO em Compose dev local em 2026-07-09.

O smoke comprova mais que liveness: rejeição sem credencial, acesso admin,
criação de tenant/domínio, `identify`, resolve por `tenantId`, ETag/304,
headers de segurança e cleanup.

## Pré-condições

- `curl` e `jq`;
- um token admin válido;
- um API client ativo dedicado a smoke ou operação (o script nunca o imprime).

## Execução

```bash
TENANCIT_BASE_URL=https://tenancit.example.com \
TENANCIT_ADMIN_TOKEN='...' \
TENANCIT_SMOKE_API_TOKEN='...' \
make smoke
```

O script usa slug/hostname únicos, registra cleanup em `trap` e exclui o tenant
ao final. Ele não cria definitions nem deixa recursos permanentes.

## Resultado esperado

```text
smoke ok: health, auth boundaries, create, identify, resolve, ETag, cleanup
```

Falha em qualquer oráculo interrompe o procedimento com exit diferente de zero.
Não use `set -x`: isso exporia headers de autorização.
