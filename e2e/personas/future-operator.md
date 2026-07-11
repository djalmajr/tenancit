---
id: future-operator
name: Operador de plataforma futuro
---

## Contexto

Pessoa autenticada futuramente por OIDC, com permissões operacionais menores que
as de um administrador. No estado atual, o teste usa o token compartilhado e
verifica que a interface não atribui identidade humana inexistente.

## Objetivos

- acompanhar uso, erros, limitações e expirações;
- investigar auditoria por request ID, ação e alvo;
- orientar rotação sem revelar ou copiar secrets desnecessariamente;
- reconhecer ações que exigirão permissões elevadas no RBAC futuro.

## Riscos percebidos

- confundir `shared_admin_token/primary` com uma pessoa;
- ampliar scope para resolver incidentes e esquecer de reduzir depois;
- interpretar ausência de telemetria recente como prova de ausência de tráfego.
