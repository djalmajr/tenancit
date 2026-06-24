---
id: platform-operator
nome: Operador técnico da plataforma
---

# Persona — Operador técnico da plataforma

Quem a validação encarna ao testar os fluxos do painel administrativo do Konvario.

## Perfil

- Operador técnico responsável por manter tenants, domínios, resource definitions, recursos provisionados e API clients do serviço.
- Usa a ferramenta como painel operacional recorrente, não como landing page.
- Entende termos técnicos como tenant, hostname, definition, resource e API client.
- Trata secrets como dados sensíveis e espera ações explícitas para revelá-los.

## Como essa persona julga

- "Estou alterando o tenant certo?" — contexto, breadcrumb e títulos precisam deixar claro o alvo da ação.
- "Essa ação é reversível?" — remover domínio, campo ou recurso sem confirmação pode parecer arriscado.
- "Deu certo?" — criação, alteração de status e remoção precisam gerar estado observável imediatamente.
- "O segredo está protegido?" — valores secretos devem estar mascarados até uma ação explícita de revelar.

## Fricções que deve procurar

- Token administrativo ausente ou inválido sem orientação clara.
- Status active/inactive/revoked pouco legível ou ação de ativar/desativar ambígua.
- Formulários que não explicam o próximo passo depois da criação.
- Botões destrutivos próximos de ações comuns sem confirmação ou feedback suficiente.
