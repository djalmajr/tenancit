# História 02 — OIDC, sessões, CSRF e RBAC

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Implementar o ADR 0005 sem acoplar o produto a um IdP específico. Dex entra
somente no Compose/E2E. O token compartilhado vira break-glass explícito.

## Rastreabilidade

- `docs/adr/0005-identidade-humana-admin-oidc-sessoes-rbac.md`.
- Admin middleware, principal, auditoria, SPA auth e todas as rotas admin.

## Arquivos

- Novas migrations/queries de login attempts e sessões hash-only.
- Pacotes de OIDC, session store, CSRF, authorization e config.
- Endpoints login/callback/session/logout e UI cookie-based.
- Compose Dex, fixtures, contratos, runbooks e testes negativos.

## Detalhe

Authorization Code + PKCE, validação completa de issuer/audience/nonce, cookie
opaco `HttpOnly`, expiração absoluta/inativa, rotação, origin + CSRF e RBAC por
permissão. `iss` + `sub` é identidade durável; e-mail é somente label.

Aceite: nenhuma credencial OIDC chega à SPA; ausência de role falha fechado;
mudança de privilégio revoga/reavalia sessões; break-glass não liga sozinho.

## Tarefas

- [ ] Aprovar ADR e congelar contratos/config.
- [ ] Criar schema expand e store transacional de attempts/sessions.
- [ ] Implementar discovery/JWKS/PKCE/nonce/state e timeouts.
- [ ] Implementar cookie, CSRF/origin, logout e revogação imediata.
- [ ] Mapear roles para permissões deny-by-default em todas as rotas.
- [ ] Propagar principal OIDC/break-glass à auditoria.
- [ ] Migrar SPA e remover token/localStorage.
- [ ] Cobrir replay, fixation, spoofing, clock boundary e IdP indisponível.

## Verificação

Unitários criptográficos/config; integração com Dex; matriz rota/permissão;
Playwright login/logout/expiração/CSRF/break-glass; canários de logs e cookies.

