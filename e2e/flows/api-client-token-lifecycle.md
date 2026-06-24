---
id: api-client-token-lifecycle
name: Gerar, copiar, revogar e reativar API client
reference: docs/produto/03-jornadas-operacionais.adoc#entregar-credencial-para-consumidor
persona: service-integrator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `rt_admin_dev`
  - clipboard do browser permitido ou fallback visual aceito para copiar manualmente
design_refs:
  api-clients: "planning/konvario/proto/routes/api-clients.js"
---

## Objetivo do usuário

Criar uma credencial de serviço, copiar o token exibido uma única vez e controlar o status do client.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`api-clients`) Autenticar-se se necessário e clicar em **API Clients** na sidebar → a tela exibe alerta sobre segredos em claro e a tabela de clients.
2. (`api-clients`) Clicar em **Novo client** → o diálogo **"Novo API client"** abre.
3. (`api-clients`) Preencher Nome e clicar em **Gerar token** → o diálogo muda para **"Token gerado"** e mostra o token completo.
4. (`api-clients`) Clicar em **Copiar** → o botão indica **"Copiado"**.
5. (`api-clients`) Clicar em **Concluir** → o diálogo fecha e o client aparece na tabela com token mascarado.
6. (`api-clients`) Clicar em **Revogar** no client criado → o status muda para revoked e a ação passa a reativar.
7. (`api-clients`) Clicar em **Reativar** → o status volta para active.

## Resultado esperado

O token completo aparece somente no momento da criação, pode ser copiado, e o client fica gerenciável por status sem expor novamente o segredo.

## Estado atual × design

- A tela existe em `web/src/routes/api-clients.tsx`.
- O token completo vem somente da resposta de criação e a lista mostra apenas máscara.
- O alerta da tela deve deixar claro que `/v1/resolve` retorna secrets em claro sobre TLS.
