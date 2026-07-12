# História 04 — Preparar a publicação O'Saasy

**Origin:** `planning/tenancit/epics/04-publicacao-e-validacao-real/00-overview.md`

## Contexto

Problema: um repositório tecnicamente pronto ainda pode publicar metadados
contraditórios, referências privadas ou políticas incompletas. O objetivo é
produzir um candidato público coerente com O'Saasy e com a fronteira core versus
extensões. O ganho é permitir colaboração e adoção sem abrir mão da restrição
contra oferta concorrente do software como serviço.

A verificação de licença desta história é interna e documental. Não exige
revisão jurídica externa nem decisão adicional do mantenedor.

## Rastreabilidade

- `LICENSE`, `SECURITY.md`, `CONTRIBUTING.md`.
- `docs/adr/0008-fronteira-core-e-extensoes.md`.
- `docs/business/05-licenciamento-e-sustentabilidade.adoc`.

## Arquivos

| Caminho | Ação | Motivo |
|---|---|---|
| `README.md` e `docs/README.adoc` | Revisar/modificar | Expor licença, suporte, segurança e posicionamento |
| `LICENSE` | Verificar | Confirmar texto O'Saasy escolhido e titular consistente |
| `SECURITY.md`, `CONTRIBUTING.md` | Revisar | Tornar canais e contribuições públicos e acionáveis |
| `.github/*` | Adicionar/revisar | Templates, Dependabot, CodeQL/ruleset quando aplicável |
| Histórico Git e working tree | Escanear | Impedir publicação de secrets e referências privadas |

## Detalhe

AS-IS: licença e documentos básicos existem; repositório ainda é privado.
TO-BE: candidato de release possui metadados, licença, segurança, contribuição,
changelog/tag e configurações públicas coerentes.

### Aceite

- `LICENSE`, README e documentação descrevem O'Saasy sem alegar MIT.
- Scan completo de working tree e histórico não encontra credenciais ou dados
  confidenciais; falsos positivos possuem justificativa versionada.
- Links, exemplos, emails e imagens funcionam para leitor externo.
- Branch protection/ruleset e recursos de segurança são preparados/ativados.
- A visibilidade somente muda após autorização explícita na execução; a história
  pode ficar `READY TO PUBLISH` sem realizar essa mudança.

### Dependências

História 03.

## Tarefas

- [x] Executar inventário público de arquivos, links e metadados.
- [x] Validar consistência O'Saasy e fronteira core/extensões.
- [x] Reexecutar gitleaks no histórico e working tree.
- [x] Preparar description, topics, templates, changelog e primeira tag/release.
- [ ] Ativar ruleset, CodeQL e vulnerability reporting quando a visibilidade permitir.
- [x] Preparar Dependabot e produzir checklist final de publicação.
- [ ] Alterar visibilidade somente com autorização explícita.

## Verificação

```bash
gitleaks git --redact --config .gitleaks.toml
git fsck --full
asciidoctor -o /dev/null docs/README.adoc
make test
```

Também validar links públicos e configurações do GitHub imediatamente antes da
mudança de visibilidade.

## Evidência

Preparação em 2026-07-11:

- texto do `LICENSE` conferido com `https://osaasy.dev/`;
- 47 commits / ~8,9 MB verificados por gitleaks, sem vazamentos; `git fsck` verde;
- working tree (~7,1 MB) novamente verificada, sem vazamentos;
- templates de issue/PR, Dependabot Bun/Go/Docker/Compose/Actions e CodeQL v4
  preparados; CodeQL só executa quando o repositório for público;
- tópicos configurados: `control-plane`, `go`, `multitenancy`, `oidc`,
  `postgresql`, `source-available`;
- `CHANGELOG.md` e `RELEASING.md` preparam `v0.1.0`, sem criar tag ou mudar a
  visibilidade.

Pendente: CI candidata, regras/recursos que só podem ser confirmados após a
publicação e autorização explícita para mudar a visibilidade.
