---
id: future-operator
name: Operador de plataforma
---

## Contexto

Pessoa autenticada por OIDC, com permissões operacionais menores que as de um
administrador. A interface deriva a navegação da sessão, mas o backend é a
autoridade: esconder um botão reduz ruído e não substitui RBAC.

## Objetivos

- acompanhar uso, erros, limitações e expirações;
- investigar auditoria por request ID, ação e alvo;
- orientar rotação sem revelar ou copiar secrets desnecessariamente;
- reconhecer ações que exigem elevação explícita de permissão.

## Riscos percebidos

- confundir `shared_admin_token/primary` com uma pessoa;
- ampliar scope para resolver incidentes e esquecer de reduzir depois;
- interpretar ausência de telemetria recente como prova de ausência de tráfego.
