# Bootstrap local

**Status:** VALIDADO em 2026-07-09 com Docker Desktop.

## Pré-condições

Go 1.25, Bun 1.3, Docker/Compose e portas 5432, 8081 e 5180 livres.

## Procedimento

```bash
make dev-compose-up
curl -fsS http://localhost:5180/healthz
```

Abra `http://localhost:5180/`. O Vite entrega a SPA e encaminha `/v1` e
`/healthz` para a API Go. Os valores do Compose são somente de demonstração.

## Verificação

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml ps
curl -fsS http://localhost:5180/healthz | jq -e '.status == "ok"'
```

## Encerramento e rollback

```bash
make dev-compose-down
```

Esse comando preserva o volume nomeado `tenancit-postgres-data`. Para apagar
deliberadamente todos os dados locais:

```bash
make docker-reset CONFIRM=destroy-local-data
```
