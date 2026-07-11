# Benchmark de escala e gate de paginação

Execute `make benchmark-scale`. O runner sobe uma stack isolada, gera datasets
de 100, 500, 1.000 e 5.000 registros, faz duas medições consecutivas e destrói
containers, banco tmpfs e volumes no final.

Cada medição usa 10 warm-ups e 100 amostras HTTP em concorrência 1 e 10. No
browser, usa viewport 1440×900, 5 warm-ups e 30 amostras de render e, nas
tabelas, filtro e sort. Os JSONs brutos, `summary.json` e `decision.md` são
gravados em `benchmarks/scale/results/<run>/` e ficam ignorados pelo Git.

Os percentis usam nearest-rank. Os limiares binários são 256.000 B (soft) e
512.000 B (hard), além de 300 ms HTTP p95 e 150 ms browser p95, sempre
confirmados nas duas rodadas da mesma escala.

`TENANCIT_SCALE_OBSERVED_VOLUME` declara a cardinalidade operacional observada
ou prevista. A curva sintética revela breakpoints de capacidade, mas não abre
sozinha o epic de paginação: a decisão aplica os triggers do roadmap no ponto
operacional declarado. Sem uma declaração, o valor é zero e o contrato de
listas completas é preservado.

O primeiro run de referência está resumido em
[`report-2026-07-10.md`](./report-2026-07-10.md). Ele preservou listas completas
e registrou 1.000 itens como o próximo checkpoint objetivo para reavaliar o
gate com telemetria operacional real.
