# História 03 — Confirmar a CI final

**Origin:** `planning/tenancit/epics/04-publicacao-e-validacao-real/00-overview.md`

## Contexto

Problema: os gates locais e jobs isolados não bastam para declarar a branch
publicável. O objetivo é provar o commit candidato completo em runners remotos,
sem rerun seletivo. O ganho é estabelecer uma baseline verificável para a futura
release pública.

## Rastreabilidade

- `.github/workflows/ci.yml`, `.github/workflows/security.yml`.
- Baseline e comandos canônicos em `RELEASING.md`.

## Arquivos

| Caminho | Ação | Motivo |
|---|---|---|
| `.github/workflows/ci.yml` | Modificar apenas se o run revelar defeito real | Manter gate obrigatório determinístico |
| `.github/workflows/security.yml` | Revisar | Confirmar scan sem upload obrigatório |
| GitHub Release | Atualizar | Registrar SHA e evidências remotas finais |
| `planning/tenancit/epics/04-publicacao-e-validacao-real/*` | Atualizar | Persistir status entre sessões |

## Detalhe

AS-IS: Go, web e produto passam; E2E falhou em 1/22. TO-BE: todos os jobs
obrigatórios e o scan de segurança passam no mesmo SHA, seguidos por execuções
limpas que não dependem de rerun manual.

### Aceite

- Server, web, produto, E2E completo, OIDC e segurança ficam verdes.
- Pelo menos três execuções consecutivas do gate principal passam sem rerun.
- SHA local, `origin/main` e SHA registrado no handoff coincidem.
- Working tree está limpa e não há run obsoleto confundindo status.

### Dependências

Histórias 01 e 02.

## Tarefas

- [x] Rodar todos os gates canônicos localmente.
- [x] Publicar os commits candidatos.
- [x] Acompanhar os workflows completos e investigar a primeira falha.
- [x] Obter três runs verdes consecutivos sem retry manual.
- [x] Atualizar handoff, overview e evidência da história.

## Verificação

```bash
make test-web
make test-db
make build
make e2e
make e2e-oidc
git status --short --branch
gh run list --branch main --limit 10
```

## Evidência

- `e9614d2`, run `29173148210`: server, web, product e E2E verdes;
- `cc5e337`, run `29173276132`: server, web, product e E2E verdes; Security
  `29173276159` verde; CodeQL corretamente ignorado enquanto privado.
- `e06460f`, run `29173656748`: server, web, product e E2E verdes; Security
  `29173656720` verde.

Três execuções consecutivas passaram sem rerun manual.
