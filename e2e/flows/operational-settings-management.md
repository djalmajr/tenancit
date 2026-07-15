---
id: operational-settings-management
name: Ajustar políticas operacionais por seção
reference: web/src/routes/settings.page.tsx
persona: administrator
entry: "http://localhost:5180/"
status: draft
preconditions:
  - app no ar e pessoa administradora autenticada com `settings.manage`
---

## Objetivo do usuário

Alterar defaults e janelas operacionais não secretas com limites claros, salvamento isolado por seção e revisão otimista protegida contra sobrescrita concorrente.

## Passos (cada passo é uma AÇÃO de UI + o resultado esperado)

1. (`settings`) Na entrada, clicar em **Configurações** na sidebar → as políticas aparecem separadas em Segurança, Chaves de API, Retenção e Console.
2. (`security`) Alterar duração absoluta ou inatividade da sessão e salvar somente **Segurança** → a seção confirma o novo valor sem enviar alterações pendentes de outras seções.
3. (`api-clients`) Alterar RPM e validade padrão para novas chaves e salvar → a UI explica que a mudança sugere defaults futuros e não reescreve credenciais existentes.
4. (`retention`) Ajustar uma retenção fora do limite permitido → o input e a API impedem salvar uma política inválida com mensagem contextual.
5. (`retention`) Corrigir o valor e salvar **Retenção** → uso, auditoria, entregas e change feed preservam unidades e responsabilidades distintas.
6. (`console`) Alterar o idioma padrão e salvar → a preferência vale para browsers que ainda não escolheram um idioma, sem sobrescrever uma escolha local existente.
7. (`concurrency`) Simular uma revisão desatualizada e tentar salvar → a UI informa o conflito/precondition e exige recarregar antes de substituir mudanças de outra pessoa.

## Resultado esperado

As políticas são compreensíveis, validadas e salvas por domínio; alterações concorrentes não são sobrescritas silenciosamente e a revisão técnica não é apresentada como histórico navegável inexistente.
