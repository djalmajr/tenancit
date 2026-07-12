# História 01 — Estabilizar o E2E de webhook

**Origin:** `planning/tenancit/epics/04-publicacao-e-validacao-real/00-overview.md`

## Contexto

Problema: a CI final passa 21/22 testes, mas o fluxo de webhook não encontra o
campo `Signing secret` depois de salvar o target. O objetivo é corrigir a causa,
sem esconder regressão com timeout ou retry. O ganho é recuperar a confiança no
fluxo vertical outbox → assinatura → receiver → status.

## Rastreabilidade

- Flow de webhook no catálogo E2E e história 04 do epic 03.
- Sem alteração de protótipo ou regra de negócio planejada.

## Arquivos

| Caminho | Ação | Motivo |
|---|---|---|
| `web/e2e/webhook-delivery.e2e.test.ts` | Diagnosticar/modificar | Sincronizar o teste com o contrato one-shot real |
| `web/src/routes/integrations.page.tsx` ou equivalente | Modificar somente se a UI violar o contrato | Garantir exibição acessível do secret após sucesso |
| `server/internal/httpapi/*webhook*` e testes | Modificar somente se a resposta estiver incorreta | Preservar persistência, auditoria e resposta one-shot |
| `e2e/flows/*webhook*` | Atualizar se o comportamento contratual mudar | Manter catálogo e automação alinhados |

## Detalhe

AS-IS: após clicar em `Salvar`, `getByRole('textbox', {name: 'Signing secret'})`
não encontra o elemento na CI. TO-BE: o sucesso mostra o secret uma única vez,
permite capturá-lo para validar HMAC e o remove após concluir/recarregar.

### Aceite

- A causa é classificada como produto, seletor/acessibilidade, sincronização ou
  fixture; a correção ataca a causa comprovada.
- Secret aparece somente após criação bem-sucedida e nunca em cache/log/trace.
- Recarregar ou reabrir não revela novamente o valor.
- Execução local passa três vezes com `retries=0` em stacks novas.

### Dependências

Nenhuma. Desbloqueia as histórias 02 e 03.

## Tarefas

- [x] Reproduzir isoladamente e guardar screenshot sanitizado da falha.
- [x] Traçar request, validação SSRF e topologia do receiver.
- [x] Corrigir a menor fronteira responsável sem sleeps arbitrários.
- [x] Preservar os testes unitários do hardening de loopback e validar pelo E2E real.
- [x] Executar o flow em três stacks novas e depois o catálogo completo.
- [x] Registrar a evidência e atualizar o overview.

## Verificação

```bash
make test-web
make e2e
make e2e-stability
```

## Evidência

Causa: após o hardening que restringiu a exceção HTTP de desenvolvimento a
loopback real, a fixture continuava cadastrando
`http://webhook-receiver-e2e:9090`, resolvido para endereço privado do bridge.
O backend rejeitava corretamente o target e o diálogo permanecia no formulário.

Correção: `app-e2e` e `webhook-receiver-e2e` compartilham o namespace de rede;
o target usa `http://127.0.0.1:9090`. O acesso externo ao produto e o proxy Vite
passam pela porta publicada pelo receiver. A defesa SSRF não foi relaxada.

Verificação em 2026-07-11:

- três stacks descartáveis: webhook 1/1 + route smoke 1/1, retry zero;
- `TENANCIT_E2E_RETRIES=0 make e2e`: 22/22 + route smoke 1/1;
- lint, TypeScript e Vitest: 21 arquivos / 82 testes verdes.
