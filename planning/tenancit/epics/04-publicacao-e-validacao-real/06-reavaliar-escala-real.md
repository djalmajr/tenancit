# História 06 — Reavaliar escala com cardinalidade real

**Origin:** `planning/tenancit/epics/04-publicacao-e-validacao-real/00-overview.md`

## Contexto

Problema: implementar paginação server-side sem volume comprovado aumentaria o
contrato e a complexidade operacional prematuramente. O objetivo é medir a
cardinalidade real ou prevista e aplicar o gate existente. O ganho é manter o
produto simples enquanto pequeno e migrar antes que o volume degrade a UX/API.

## Rastreabilidade

- `planning/tenancit/roadmap-quality-scale.md`.
- História 11 do epic 03 e relatório do benchmark de escala.
- Superfícies: tenants, definitions, API clients, audit e usage.

## Arquivos

| Caminho | Ação | Motivo |
|---|---|---|
| Benchmark/scripts de escala existentes | Executar/modificar se necessário | Reproduzir volume observado |
| `planning/tenancit/roadmap-quality-scale.md` | Atualizar | Registrar dados e decisão |
| Roadmap verificável | Atualizar | Expor decisão operacional atual |
| Novo epic de paginação | Criar somente se `MIGRATE` | Separar mudança contratual deste gate |

## Detalhe

AS-IS: benchmark sintético recomenda `KEEP_FULL_LISTS`; primeiro hard trigger
repetível é 500 definições e 1.000 itens em algumas outras superfícies. TO-BE:
volume observado/projetado, latência, payload e comportamento do browser
produzem decisão explícita e reproduzível.

### Aceite

- Cardinalidades são coletadas por superfície sem dados pessoais/segredos.
- Benchmark usa `TENANCIT_SCALE_OBSERVED_VOLUME` e repete ao menos duas rodadas.
- Decisão registra `KEEP_FULL_LISTS` ou `MIGRATE`, thresholds e evidência.
- Se `MIGRATE`, um epic separado cobre filtros, sort, cursor/página, contratos,
  back/front/docs/testes e rollout compatível; não se implementa parcialmente.

### Dependências

Telemetria real, projeção aprovada ou premissa de engenharia explicitamente
marcada. Deve ser reconfirmada por cliente quando houver cardinalidade real.

## Tarefas

- [x] Identificar fontes e harness de cardinalidade por superfície.
- [x] Coletar premissa, payload, p95 e custo de renderização sanitizados.
- [x] Reexecutar toda a curva sintética em duas rodadas; repetir com volume real quando disponível.
- [x] Aplicar thresholds e registrar a decisão.
- [x] Manter listas; nenhum epic de paginação foi aberto.
- [x] Atualizar overview, roadmap e handoff.

## Verificação

```bash
make benchmark-scale
```

Executar também Browser em desktop/mobile no maior volume representativo e
registrar latência/payload sem capturar dados operacionais.

## Evidência

Em 2026-07-11, após adequar o benchmark ao namespace de rede E2E, duas rodadas
de 100, 500, 1.000 e 5.000 registros passaram. O primeiro breakpoint permaneceu
em 500 e a decisão foi `KEEP_FULL_LISTS` porque o volume operacional declarado
continua zero. Esta execução prova o harness, não substitui a medição real.

Premissa de engenharia usada em 2026-07-11: até 50 mil usuários finais por
ambiente e um cenário administrativo sintético de 250 registros por superfície.
Os valores de 100 tenants/definições e 250 API clients não são tratados como
projeção comercial aprovada e devem ser confirmados por cliente. Os
usuários finais aumentam throughput das APIs de consumo, mas não viram linhas
administrativas automaticamente.

`TENANCIT_SCALE_OBSERVED_VOLUME=250 make benchmark-scale` repetiu 100, 250, 500,
1.000 e 5.000 em duas rodadas. O sumário local
`benchmarks/scale/results/20260712t031427z-33947/summary.json` registrou
`KEEP_FULL_LISTS`: volume 250 abaixo do primeiro breakpoint 500. Teste de carga
de consumo/SLO permanece uma trilha diferente e deve ser dimensionado por
ambiente de cliente.

O harness mede toda a curva em `1440x900` e repete o volume operacional de 250
em `390x844`; portanto, a decisão inclui renderização, busca e ordenação em
desktop e mobile sem confundir o ensaio móvel representativo com stress acima
do breakpoint de capacidade.

As duas rodadas mobile em 250 itens ficaram abaixo do hard trigger de 150 ms;
o maior p95 observado no ponto foi 123,8 ms em definições desktop e 28,8 ms em
mobile. Cada métrica usa 5 warm-ups e 30 amostras. As duas rodadas completas
estão consolidadas em `20260712t041000z-combined-250`; nenhum trigger duro ou
soft foi confirmado no ponto operacional. A curva completa anterior preserva
o breakpoint sintético 500.
