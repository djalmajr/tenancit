# História 03 — Confirmar a CI final

**Origin:** `planning/tenancit/epics/04-publicacao-e-validacao-real/00-overview.md`

## Contexto

Problema: os gates locais e jobs isolados não bastam para declarar a branch
publicável. O objetivo é provar o commit candidato completo em runners remotos,
sem rerun seletivo. O ganho é estabelecer uma baseline verificável para a futura
release pública.

## Rastreabilidade

- `.github/workflows/ci.yml`, `.github/workflows/security.yml`.
- Baseline e comandos canônicos em `docs/HANDOFF.md`.

## Arquivos

| Caminho | Ação | Motivo |
|---|---|---|
| `.github/workflows/ci.yml` | Modificar apenas se o run revelar defeito real | Manter gate obrigatório determinístico |
| `.github/workflows/security.yml` | Revisar | Confirmar scan sem upload obrigatório |
| `docs/HANDOFF.md` | Atualizar | Registrar SHA e evidências remotas finais |
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

- [ ] Rodar todos os gates canônicos localmente.
- [ ] Publicar o commit candidato.
- [ ] Acompanhar o workflow completo e investigar qualquer primeira falha.
- [ ] Obter três runs verdes consecutivos sem retry manual.
- [ ] Atualizar handoff, overview e evidência da história.

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

Pendente. Registrar SHA, três IDs/URLs de runs e resultados por job.
