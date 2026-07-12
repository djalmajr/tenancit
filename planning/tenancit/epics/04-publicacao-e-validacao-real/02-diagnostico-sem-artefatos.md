# História 02 — Diagnóstico sem depender de artefatos

**Origin:** `planning/tenancit/epics/04-publicacao-e-validacao-real/00-overview.md`

## Contexto

Problema: quando o E2E falha, `actions/upload-artifact` também falha porque a
cota está esgotada, adicionando ruído e podendo ocultar o erro primário. O
objetivo é manter diagnóstico sanitizado no próprio job e tornar artefatos um
complemento opcional. O ganho é uma CI investigável independentemente do plano
ou da cota da conta GitHub.

## Rastreabilidade

- `.github/workflows/ci.yml` e política de artefatos Playwright.
- Não envolve UI, protótipos ou regras de negócio.

## Arquivos

| Caminho | Ação | Motivo |
|---|---|---|
| `.github/workflows/ci.yml` | Modificar | Separar resultado do teste do upload opcional |
| `scripts/e2e.sh` | Modificar se necessário | Emitir resumo sanitizado e localizar evidências |
| `web/playwright.config.ts` | Revisar/modificar | Controlar retenção local e conteúdo sensível |
| `docs/developers/testing.adoc` ou documento equivalente | Atualizar | Explicar diagnóstico com e sem artefatos |

## Detalhe

AS-IS: o upload falha com `Artifact storage quota has been hit`. TO-BE: a saída
do job identifica spec, etapa e error-context sanitizado; upload roda apenas
quando viável e sua indisponibilidade não substitui a conclusão do teste.

### Aceite

- O exit code final reflete o Playwright, não o upload.
- Nenhum screenshot, trace ou error-context contém token, cookie, DSN ou secret.
- A política evita gerar artefatos em sucesso e usa retenção mínima em falha.
- Existe procedimento documentado para baixar/ver evidência quando disponível e
  reproduzir localmente quando não estiver.

### Dependências

Pode avançar em paralelo à história 01, mas deve usar a falha real como prova.

## Tarefas

- [x] Mapear comportamento atual e limites do GitHub Actions.
- [x] Tornar upload opcional/não mascarador sem silenciar o teste.
- [x] Imprimir resumo sanitizado na saída e no job summary.
- [x] Restringir o resumo a caminhos/tamanhos e manter traces/context removidos.
- [ ] Confirmar em workflow remoto que upload desabilitado não interfere.
- [x] Atualizar documentação e overview.

## Verificação

```bash
make e2e
git diff --check
```

Executar também um workflow remoto com o upload indisponível ou explicitamente
desabilitado e confirmar que o resultado continua fiel ao Playwright.

## Evidência

Implementado em 2026-07-11: o line reporter permanece como diagnóstico
primário; o job summary lista apenas caminhos relativos/tamanhos. Upload exige a
variável `TENANCIT_UPLOAD_PLAYWRIGHT_ARTIFACTS=true`, usa retenção de um dia e
possui `continue-on-error`. A confirmação remota será registrada após o push.
