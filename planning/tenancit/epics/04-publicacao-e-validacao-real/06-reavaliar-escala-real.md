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
| `docs/HANDOFF.md` | Atualizar | Expor decisão operacional atual |
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

Telemetria real ou projeção aprovada. Pode começar após a história 03 e deve ser
reconfirmada depois da história 05 quando houver ambiente real.

## Tarefas

- [x] Identificar fontes e harness de cardinalidade por superfície.
- [ ] Coletar contagens, payload, p95/p99 e custo de renderização sanitizados.
- [x] Reexecutar toda a curva sintética em duas rodadas; repetir com volume real quando disponível.
- [ ] Aplicar thresholds e registrar a decisão.
- [ ] Manter listas ou criar epic de paginação completo.
- [ ] Atualizar overview, roadmap e handoff.

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

Pendente por gate de volume: informar uma cardinalidade observada ou uma
projeção formal, repetir com `TENANCIT_SCALE_OBSERVED_VOLUME` e registrar a
decisão final.
