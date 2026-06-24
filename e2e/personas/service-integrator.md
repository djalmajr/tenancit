---
id: service-integrator
nome: Pessoa integradora de serviço consumidor
---

# Persona — Integradora de serviço consumidor

Quem a validação encarna ao testar a entrega de API client e a resolução server-to-server por hostname.

## Perfil

- Pessoa desenvolvedora ou operadora de um serviço consumidor que precisa obter recursos de runtime por hostname via API.
- Quer validar rapidamente que `/v1/resolve` retorna os recursos esperados.
- Precisa entender que tokens de API são exibidos uma única vez.
- Dá valor a respostas HTTP previsíveis: 401 sem token, 404 para hostname desconhecido e 200 para hostname configurado.

## Como essa persona julga

- "Consigo copiar o token agora?" — o token completo precisa estar claro e copiável no único momento em que aparece.
- "O que esse token consegue acessar?" — a UI deve explicar que `/v1/resolve` retorna secrets em claro sobre TLS.
- "O tenant está pronto para consumo?" — o caminho mínimo precisa ser evidente: tenant ativo, domínio, definition ativa, resource ativo e API client ativo.
- "Se falhar, sei por quê?" — erros de token, hostname e resource precisam ser distinguíveis.

## Fricções que deve procurar

- Token completo sumir antes de ser copiado.
- Lista exibir token real depois da criação.
- Falta de orientação sobre uso de `Authorization: Bearer`.
- Painel não deixar claro que o consumer path é uma API, não uma tela.
