# História 07 — Rewrap AES executável

**Origin:** `planning/tenancit/epics/03-plataforma-operacional/00-overview.md`

## Contexto

Implementar integralmente o design existente como comando offline separado do
servidor HTTP.

## Responsabilidade, motivação e valor

Como o Tenancit cifra secrets que ele próprio armazena, também precisa permitir
retirar uma chave AES antiga. Apenas trocar a chave usada por novos writes
deixaria todo o histórico dependente do material anterior.

**Ganho:** rotação real após política periódica ou suspeita de comprometimento,
sem downtime, plaintext persistido ou perda de dados. É um job offline raro e
controlado, não um serviço permanente ou plataforma genérica de gestão de chaves.

## Arquivos

- `server/cmd/tenancit-rewrap`, serviço de campanha, queries e testes.
- Makefile/container job, métricas/reports e atualização do runbook/design.

## Detalhe

Advisory lock, inventário, dry-run completo, lotes `SKIP LOCKED`, decrypt/
encrypt/verify, CAS, retomada pelo estado do banco e zero alteração de ETag
funcional. Nenhum material criptográfico vai a argumento/log/telemetria.

## Tarefas

- [x] Implementar preflight fail-closed e inventário por versão.
- [x] Implementar dry-run que autentica todos os ciphertexts sem updates.
- [x] Implementar lotes transacionais, CAS, cancelamento e no-progress timeout.
- [x] Integrar métricas e system report sem IDs/material sensível.
- [x] Cobrir writers concorrentes, crash, chave ausente e linha malformada.
- [x] Ensaiar stage/cutover/rewrap/restore/retirada da chave antiga.

## Verificação

Property/integration tests, interrupção e rerun idempotente, restore
representativo e canários de vazamento.

## Evidência local — 2026-07-11

- `Cryptor` ganhou API de bytes apagáveis e metadados de versão sem expor keys.
- `internal/rewrap` autentica inventário paginado antes de qualquer update,
  exige reports fresh de backup e restore e usa advisory lock exclusivo.
- Lotes usam `FOR UPDATE SKIP LOCKED`, verificação decrypt-after-encrypt e CAS;
  cada commit é independente e rerun continua pelas versões no banco.
- Testes cobrem bytes idênticos no dry-run, tamper, nonce malformado, chave
  ausente, rollback injetado, lock concorrente, writer que prevalece, timeout,
  clone restaurado e leitura final usando somente a chave nova.
- A CLI exige change UUID/confirm-write, recebe keys/DSN só pelo ambiente,
  publica report autenticado e emite OTLP/logs/JSON apenas com contagens.
- A imagem inclui `/tenancit-rewrap`; Compose operacional separa dry-run de
  escrita. O role `tenancit_rewrap` só lê colunas cifradas/reports e só atualiza
  cipher, nonce e versão.

A primeira campanha com chave real permanece um gate externo, não uma lacuna de
implementação: exige alvo, secret manager, aprovadores e janela fornecidos pelo
operador.
