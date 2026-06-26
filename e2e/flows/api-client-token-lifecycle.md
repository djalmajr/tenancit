---
id: api-client-token-lifecycle
name: Gerar, copiar, revogar e reativar chave de API
reference: docs/produto/03-jornadas-operacionais.adoc#entregar-credencial-para-consumidor
persona: service-integrator
entry: "http://localhost:5180/"
preconditions:
  - app no ar em modo desenvolvimento
  - token administrativo válido disponível: `tenancit_admin_dev`
  - clipboard do browser permitido ou fallback visual aceito para copiar manualmente
design_refs:
  api-clients: "planning/tenancit/proto/routes/api-clients.js"
---

## Objetivo do usuário

Criar uma chave de serviço, copiar o token exibido uma única vez e controlar o status da chave.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`api-clients`) Autenticar-se se necessário e clicar em **Chaves de API** na sidebar → a tela exibe alerta sobre segredos em claro, snippet copiável de `/v1/resolve` e a tabela de chaves.
2. (`api-clients`) Clicar em **Copiar snippet** → o botão indica **Snippet copiado** e a tela mostra feedback.
3. (`api-clients`) Clicar em **Nova chave** → o diálogo **"Nova chave de API"** abre.
4. (`api-clients`) Preencher Nome e clicar em **Gerar token** → o diálogo muda para **"Token gerado"** e mostra o token completo.
5. (`api-clients`) Clicar em **Copiar** → o botão indica **"Copiado"**.
6. (`api-clients`) Clicar em **Concluir** → o diálogo fecha e a chave aparece na tabela com token mascarado.
7. (`api-clients`) Clicar em **Revogar** na chave criada → o status muda para revoked, a ação passa a reativar e a tela mostra feedback.
8. (`api-clients`) Clicar em **Reativar** → o status volta para active e a tela mostra feedback.

## Resultado esperado

O token completo aparece somente no momento da criação, pode ser copiado, e a chave fica gerenciável por status sem expor novamente o segredo.

## Estado atual × design

- A tela existe em `web/src/routes/api-clients.tsx`.
- O token completo vem somente da resposta de criação e a lista mostra apenas máscara.
- O alerta da tela deve deixar claro que `/v1/resolve` retorna secrets em claro sobre TLS.
