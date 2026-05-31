# Protótipo — Resource Tenant Admin Console

Protótipo interativo (hi-fi) do painel administrativo do serviço Resource Tenant.
Stack CDN-only (z-proto + Tailwind v4 + shadcn + Preact/htm). Sem build.

## Rodar

```bash
cd planning/resource-tenant-service/proto
bunx serve -s .
# abre http://localhost:3000
```

O flag `-s` (SPA mode) é necessário para as rotas client-side (`/tenants/:id` etc).

## Telas (rotas)

| Rota | Cena | O que demonstra |
|------|------|-----------------|
| `/` | Visão geral | KPIs (tenants, domínios, recursos, definitions) + atalho para tenants |
| `/tenants` | Lista de Tenants | Tabela + busca + dialog "Novo tenant" (nome, slug, hostname) |
| `/tenants/:id` | Detalhe do Tenant | Abas **Recursos** e **Domínios**; recursos com valores; **secrets mascarados com botão revelar**; dialog "Adicionar recurso" (escolha do tipo/template) |
| `/resource-definitions` | Catálogo | Cards dos tipos de recurso (postgres, minio, smtp) com contagem de campos/secret |
| `/resource-definitions/:id` | Detalhe da Definition | Tabela de campos (chave, label, tipo, required, secret) + dialog "Novo campo" com toggles required/secret |
| `/api-clients` | API Clients | Tokens de serviço (consumo server-to-server), alerta sobre retorno de secrets em claro |

## Decisões de produto refletidas

- **Secret-aware**: campos `is_secret` aparecem mascarados (`••••`) com botão de revelar
  (espelha o `?reveal=true` previsto no design); ícone de cadeado e badge "secret".
- **Modelo dinâmico**: recursos derivam seus campos da Resource Definition (template).
- **1 ativo por tipo**: dialog de adicionar recurso reforça a invariante.
- **Cadastro via UI + API**: o painel é servido pelo próprio app Go (SPA embutido),
  convivendo com os endpoints REST.

> Dados são mock (`routes/_data.js`). Sem fetch, sem backend.
