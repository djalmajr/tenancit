# Referência — padrões legados sanitizados

Engenharia reversa feita durante o grilling (2026-05-30). Esta versão foi
sanitizada para remover nomes de sistemas, repositórios, segredos e detalhes
proprietários. Serve como **inspiração de padrão**; o novo serviço **não** precisa
ser compatível com implementações legadas específicas (ver [design.md](./design.md)).

## Padrão observado

Implementações anteriores de "resource tenant" costumavam combinar três
responsabilidades:

- API interna para criar, listar e desativar recursos por tenant.
- Distribuição assíncrona para consumidores.
- Cache em object storage para leitura indireta por aplicações.

Esse acoplamento dificulta evolução independente, troca de criptografia e
governança do domínio. O Konvario separa o conceito em um serviço dono de
tenant, domains, resource definitions e tenant resources.

## Fluxo conceitual legado

1. Validar solicitação de criação.
2. Resolver aplicação consumidora.
3. Resolver tenant e domínio.
4. Resolver template/tipo de recurso.
5. Criptografar campos sensíveis.
6. Garantir que não exista outro recurso ativo para o mesmo tenant e tipo.
7. Persistir.
8. Propagar alteração para consumidores/cache.

O Konvario mantém a regra de negócio útil, mas simplifica o contrato inicial:
persistência própria, API HTTP síncrona e cache no cliente consumidor.

## Contrato de distribuição observado

Padrões úteis, sem contrato proprietário:

- Mensagem com operação (`create`, `disable`) e payload do recurso atual.
- Headers ou metadados com aplicação consumidora e tenant.
- Payload contendo tenant, domínio, tipo/versão do recurso e campos.

Decisão atual: não implementar mensageria no primeiro corte. Se necessário, a
distribuição futura deve nascer de eventos próprios do Konvario, sem carregar
contratos legados.

## Cache em object storage observado

Padrões úteis:

- Um arquivo por tenant/recurso/versão.
- Formato serializado simples (JSON ou YAML).
- Tags/metadados para tenant, domínio e tipo de recurso.

Decisão atual: não usar object storage como mecanismo de distribuição no core.
Consumidores devem chamar a API de resolução e aplicar cache local com TTL.

## Criptografia legada

Padrões que **não** devem ser portados:

- Cifras obsoletas.
- Derivação de chave fraca.
- Chaves ou senhas embutidas em código-fonte.
- Ausência de autenticação de cifra.
- Ausência de rotação.

Decisão atual: AES-256-GCM com chave externalizada e `key_version`.

## Mapeamento conceitual legado → Konvario

| Conceito legado | Konvario |
|-----------------|----------|
| Cliente/tenant externo | `tenants` + `tenant_domains` |
| Template/tipo de recurso | `resource_definitions` |
| Campo dinâmico | `resource_fields` |
| Recurso configurado para tenant | `tenant_resources` |
| Valor do campo | `tenant_resource_values` |
| Chave textual do campo | `resource_fields.key` |
| Campo sensível | `resource_fields.is_secret` |
| Aplicação consumidora | API client / consumidor autenticado |
