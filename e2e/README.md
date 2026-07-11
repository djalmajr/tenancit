# Catálogo E2E automatizado

O catálogo executável preserva correspondência entre os 19 flows
de `e2e/flows`, as specs e seus 143 passos numerados. Cada passo Markdown é
um `test.step` produzido por `flowStep(flowId, número, título, ação + assert)`;
não há consolidações de passos.

| flow-id | spec | test.step | tier | passos |
|---|---|---|---|---|
| `admin-auth-overview` | `admin-auth-overview.e2e.test.ts` | `[admin-auth-overview#N] título` | `pr-critical` | 1–11 |
| `admin-form-validation-and-errors` | `admin-form-validation-and-errors.e2e.test.ts` | `[admin-form-validation-and-errors#N] título` | `full` | 1–8 |
| `admin-invalid-token-recovery` | `admin-invalid-token-recovery.e2e.test.ts` | `[admin-invalid-token-recovery#N] título` | `pr-critical` | 1–5 |
| `admin-to-consumer-golden-path` | `admin-to-consumer-golden-path.e2e.test.ts` | `[admin-to-consumer-golden-path#N] título` | `pr-critical` | 1–11 |
| `api-client-token-lifecycle` | `api-client-token-lifecycle.e2e.test.ts` | `[api-client-token-lifecycle#N] título` | `pr-critical` | 1–8 |
| `api-client-governance` | `api-client-operational-governance.e2e.test.ts` | `[api-client-governance#N] título` | `full` | 1–5 |
| `api-client-rate-limit` | `api-client-operational-governance.e2e.test.ts` | `[api-client-rate-limit#N] título` | `full` | 1–4 |
| `api-client-usage-audit` | `api-client-operational-governance.e2e.test.ts` | `[api-client-usage-audit#N] título` | `full` | 1–4 |
| `audit-operations` | `audit-operations.e2e.test.ts` | `[audit-operations#N] título` | `full` | 1–4 |
| `consumer-specific-resource-resolution` | `consumer-specific-resource-resolution.e2e.test.ts` | `[consumer-specific-resource-resolution#N] título` | `pr-critical` | 1–9 |
| `definition-deactivation-provisioning` | `definition-deactivation-provisioning.e2e.test.ts` | `[definition-deactivation-provisioning#N] título` | `full` | 1–8 |
| `directory-search-sort-pagination` | `directory-search-sort-pagination.e2e.test.ts` | `[directory-search-sort-pagination#N] título` | `full` | 1–7 |
| `first-run-empty-states` | `00-first-run-empty-states.e2e.test.ts` | `[first-run-empty-states#N] título` | `pr-critical` | 1–9 |
| `i18n-and-preferences-persistence` | `i18n-and-preferences-persistence.e2e.test.ts` | `[i18n-and-preferences-persistence#N] título` | `full` | 1–7 |
| `keyboard-accessibility-core` | `keyboard-accessibility-core.e2e.test.ts` | `[keyboard-accessibility-core#N] título` | `full` | 1–5 |
| `resource-definition-management` | `resource-definition-management.e2e.test.ts` | `[resource-definition-management#N] título` | `full` | 1–10 |
| `responsive-mobile-navigation` | `responsive-mobile-navigation.e2e.test.ts` | `[responsive-mobile-navigation#N] título` | `full` | 1–7 |
| `tenant-management` | `tenant-management.e2e.test.ts` | `[tenant-management#N] título` | `pr-critical` | 1–10 |
| `tenant-resource-lifecycle` | `tenant-resource-lifecycle.e2e.test.ts` | `[tenant-resource-lifecycle#N] título` | `pr-critical` | 1–11 |

## Convenções de execução

- A suíte roda com um worker. O flow `first-run-empty-states` usa prefixo `00-`
  para observar o PostgreSQL efêmero antes dos flows que criam dados.
- Ações descritas como UI são realizadas por seletores semânticos na UI. Setup
  não pertencente ao percurso pode usar a Admin API; Consumer API é exercitada
  somente pelo fixture `request`, sem navegar o browser para endpoints internos.
- Todos os registros recebem nomes únicos. Cada spec registra cleanup no início
  e o executa em `finally`: tenants são excluídos, definitions são desativadas e
  chaves são revogadas. O teardown da stack remove o banco efêmero inteiro.
- Traces ficam desativados globalmente porque registram valores preenchidos e
  headers de autorização. Na stack efêmera, screenshots de falha continuam
  disponíveis para flows sem segredo; specs que exibem token ou segredo também
  os desativam.
- Cada execução usa um diretório de artefatos próprio. O runner remove
  `error-context.md` antes do upload porque matchers podem capturar o snapshot
  textual da página. Em modo externo, screenshots ficam desligados e a suíte
  exige confirmação explícita por variável de ambiente.
- O tier `pr-critical` cobre bootstrap, autenticação, ciclo principal de tenant,
  secrets e contratos Consumer. O tier `full` cobre profundidade funcional,
  responsividade, acessibilidade, i18n e operações de tabela.

## Gates

```sh
make e2e-catalog
make e2e-smoke
make e2e-pr
make e2e
make e2e-stability
```

Os targets `make` são a interface canônica: criam banco e credenciais efêmeros,
aplicam a política de artefatos e sempre removem containers, volumes e redes.
O modo `TENANCIT_E2E_EXTERNAL=1` só é suportado contra ambiente dedicado,
descartável e vazio; ele muta e remove dados. Para reduzir uso acidental, exige
também `TENANCIT_E2E_EXTERNAL_MUTATIONS_ACK=1`.

O checker compara frontmatter e passos Markdown, chamadas literais de
`flowStep` nas specs e a matriz acima. Ele também protege a política de
artefatos e rejeita matchers que recebem tokens crus. Falha para flow/passo
ausente, duplicado, fora do intervalo, spec divergente ou drift dos totais
19/143.
