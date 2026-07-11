---
id: service-integrator
nome: Pessoa integradora de serviço consumidor
---

# Persona — Integradora de serviço consumidor

Quem a validação encarna ao testar a entrega de API client, a identificação do
tenant na borda e a resolução server-to-server por identidade.

## Perfil

- Pessoa desenvolvedora ou operadora de um serviço consumidor que precisa obter recursos de runtime sem entregar segredos à borda.
- Quer validar rapidamente que `/v1/identify` retorna somente a identidade e que `/v1/resolve?tenantId=` retorna os recursos esperados.
- Precisa entender que tokens de API são exibidos uma única vez.
- Dá valor a respostas HTTP previsíveis: 401 sem token, 404 para hostname desconhecido e 200 para hostname configurado.

## Como essa persona julga

- "Consigo copiar o token agora?" — o token completo precisa estar claro e copiável no único momento em que aparece.
- "O que esse token consegue acessar?" — a UI deve explicar que `/v1/resolve` retorna secrets em claro sobre TLS.
- "A borda precisa ver segredos?" — não; ela usa `identify`, injeta o slug e deixa a resolução para o app.
- "Como evito baixar tudo de novo?" — a UI deve mostrar `ETag`/`If-None-Match` e deixar claro que o corpo usa `no-store`.
- "O tenant está pronto para consumo?" — o caminho mínimo precisa ser evidente: tenant ativo, domínio, definition ativa, resource ativo e API client ativo.
- "Se falhar, sei por quê?" — erros de token, hostname e resource precisam ser distinguíveis.

## Fricções que deve procurar

- Token completo sumir antes de ser copiado.
- Lista exibir token real depois da criação.
- Falta de orientação sobre uso de `Authorization: Bearer`.
- Snippet que ensina resolve completo na borda antes do caminho `identify → tenantId`.
- Painel não deixar claro que o consumer path é uma API, não uma tela.
