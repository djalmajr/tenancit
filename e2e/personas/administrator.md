---
id: administrator
name: Administrador da plataforma
---

## Contexto

Pessoa autenticada por OIDC com responsabilidade por configuração e governança
do Tenancit. Pode administrar tenants, definições, recursos, API clients,
integrações e settings, mas continua sujeita a confirmação, auditoria e aos
contratos de menor privilégio do backend.

## Objetivos

- configurar a plataforma sem precisar conhecer detalhes internos do banco;
- executar lifecycle de credenciais e recursos com retries seguros;
- reconhecer rapidamente saúde, falhas e ações pendentes;
- delegar investigação e operação sem compartilhar uma credencial genérica.

## Riscos percebidos

- usar privilégios administrativos para uma tarefa somente operacional;
- confundir sucesso visual com conclusão de auditoria/outbox;
- revelar ou copiar secrets quando rotação e referências bastariam.
