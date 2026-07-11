# ADR 0007 — Roles PostgreSQL e deploy expand/contract

- **Status:** Aceito
- **Data:** 2026-07-11

## Contexto

Executar migrations no boot entrega DDL ao processo HTTP, mistura autoridade de
owner com tráfego e torna incompatibilidade visível somente durante o rollout.
Um único DSN também concede escrita aos agentes de backup e amplia workers além
das filas que precisam manipular.

## Decisão

O Tenancit terá um binário `/migrate` separado e grupos NOLOGIN para
`tenancit_runtime`, `tenancit_jobs`, `tenancit_backup` e `tenancit_rewrap`, além
do owner/migration provisionado pelo operador. O servidor HTTP não importa o pacote
`internal/migration`; o lint transforma essa fronteira em gate.

Deploys de upgrade seguem expand/contract. A migration expand roda enquanto o
digest anterior continua atendendo. O release consulta a readiness anterior
antes de substituir réplicas. Contract ocorre em release posterior, depois de
inventário zerado de versões antigas. Imagens são identificadas exclusivamente
por digest; rollback preserva DSN/schema.

## Consequências

- owner/migration nunca atende HTTP;
- runtime não cria schema nem altera trilhas append-only;
- jobs escrevem somente filas/agregados operacionais allowlisted;
- backup é read-only e seus dumps permanecem fora do host/check-out;
- primeiro deploy exige provisionamento explícito; rollout cotidiano falha se
  não houver backup fresh ou se o binário anterior quebrar após expand;
- o alvo real ainda precisa definir ingress/TLS, secrets e trusted proxies.

## Alternativas rejeitadas

- migration automática no boot: privilégio excessivo e rollout ambíguo;
- um DSN compartilhado: não prova menor privilégio;
- rollback de schema automático: pode destruir dados e mascarar contract
  incompatível;
- tag de imagem: não identifica conteúdo imutável.
