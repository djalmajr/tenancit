---
id: operational-health-review
name: Revisar saúde, dependências e evidências operacionais
reference: web/src/routes/operations-health.page.tsx
persona: future-operator
entry: "http://localhost:5180/"
status: draft
preconditions:
  - app no ar e operador autenticado com permissão de leitura administrativa
  - PostgreSQL e Valkey configurados no ambiente
---

## Objetivo do usuário

Entender se o próprio Tenancit e suas dependências internas estão prontos, sem interpretar a tela como monitoramento dos recursos cadastrados pelos tenants.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`overview`) Na entrada, revisar **Saúde operacional** e **Dead letters** → o dashboard oferece um sinal resumido e direciona a investigação.
2. (`health`) Clicar em **Saúde** na sidebar → a tela mostra estado geral, quantidade de dependências, entregas em trânsito e dead letters.
3. (`dependencies`) Revisar a tabela **Dependências** → PostgreSQL, Valkey e outras dependências do runtime exibem status e latência, sem listar recursos de tenants.
4. (`reports`) Revisar **Reports operacionais** → evidências de backup/restore exibem origem, estado efetivo, execução e validade.
5. (`degraded`) Observar um componente ou report degradado/desatualizado → badge e estado geral deixam a condição explícita e preservam a diferença entre indisponível e stale.
6. (`refresh`) Manter a aba visível até o próximo polling → dados são atualizados sem recarregar a página; ao ocultar a aba, o polling deixa de gerar atividade desnecessária.

## Resultado esperado

O operador identifica degradação do runtime e evidências vencidas, sabe qual dependência investigar e não confunde saúde do Tenancit com disponibilidade de bancos, buckets ou filas cadastrados como configuração.
