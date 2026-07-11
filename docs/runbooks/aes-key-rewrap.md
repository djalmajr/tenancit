# Rewrap de chave AES

**Status:** VALIDADO localmente em 2026-07-11; primeira campanha de produção
continua condicionada ao alvo, janela e aprovadores.

Este runbook define os controles para uma campanha futura de rotação. O desenho
técnico está em
[`docs/developers/design/aes-key-rewrap.md`](../developers/design/aes-key-rewrap.md).
Não copie chaves reais para comandos, tickets, arquivos ou logs.

## Quando usar

- rotação programada da chave AES corrente;
- retirada de uma versão antiga após mudança de política;
- resposta à suspeita ou confirmação de comprometimento de chave.

O procedimento recriptografa apenas secrets em `tenant_resource_values`. Ele
não muda valores em claro, contratos HTTP nem credenciais de API clients.

## Papéis

- **Operador:** executa inventário, backup, dry-run e job;
- **Aprovador:** valida janela, evidências e retirada da chave antiga;
- **Responsável por secrets:** cria/publica versões e controla revogação;
- **Observador:** acompanha banco, aplicação e alertas durante a janela.

Em produção, operador e aprovador devem ser pessoas diferentes.

## Pré-condições

- [x] versão do Tenancit com o job de rewrap e seus testes aprovada;
- [ ] mudança/janela registrada com `job_id`, ambiente e versão alvo;
- [ ] nova chave AES-256 gerada no secret manager, nunca no repositório;
- [ ] nova versão é positiva, inédita e maior que as anteriores;
- [ ] todas as chaves históricas ainda usadas pelo banco estão disponíveis;
- [ ] todas as réplicas executam o loader que aceita versões adicionais staged;
- [ ] o orquestrador permite provar rollout por instância ou há freeze de writers;
- [ ] dump recente criado e restore validado em banco isolado;
- [ ] backup das chaves coberto pela política do secret manager;
- [ ] espaço, conexões, replica lag e alertas do PostgreSQL verificados;
- [ ] smoke de resolução disponível para a validação final;
- [ ] plano de parada, responsáveis e canal de incidente definidos.

Se qualquer item falhar, não iniciar escrita.

## 1. Inventário somente leitura

Capture apenas contagens:

```sql
SELECT key_version, count(*) AS rows
FROM tenant_resource_values
WHERE value_cipher IS NOT NULL
GROUP BY key_version
ORDER BY key_version;
```

Verifique invariantes malformadas:

```sql
SELECT count(*) AS malformed_rows
FROM tenant_resource_values
WHERE value_cipher IS NOT NULL
  AND (
    nonce IS NULL
    OR key_version IS NULL
    OR key_version <= 0
  );
```

Não exporte `value_cipher`, `nonce` nem valores de resources. Registre no ticket
somente contagens por versão, `malformed_rows`, timestamp e ambiente.

**Gate:** `malformed_rows = 0` e cada versão inventariada possui uma chave no
secret manager. Caso contrário, parar e abrir triagem.

## 2. Preparar a nova configuração

O cutover usa dois rollouts. Não os combine.

**Fase A — stage somente para leitura:**

1. mantenha `TENANCIT_AES_KEY` e `TENANCIT_AES_KEY_VERSION` na origem;
2. publique a nova chave em `TENANCIT_AES_KEY_V<alvo>`;
3. recicle todas as réplicas e prove, por instância, que carregaram a versão
   alvo sem mudar a versão usada por `Encrypt`;
4. valide health e um resolve controlado; nenhum write deve usar o alvo ainda.

**Fase B — trocar novos writes:**

1. publique a nova chave também como `TENANCIT_AES_KEY` e aumente
   `TENANCIT_AES_KEY_VERSION` para o alvo;
2. preserve a chave origem em `TENANCIT_AES_KEY_V<origem>` e todas as demais
   versões ainda presentes no inventário;
3. faça o segundo rolling deploy. Nós ainda na fase A conseguem ler ciphertexts
   alvo, evitando incompatibilidade durante a transição;
4. prove que todas as réplicas agora escrevem no alvo; só então inicie rewrap;
5. valide health e um resolve controlado novamente.

Se não for possível provar o stage em todas as réplicas, congele writers e faça
um restart coordenado com a configuração completa. Um rolling deploy monofásico
é um STOP gate.

Não reduza a versão corrente após qualquer linha ser migrada: isso retomaria
writes sob a chave antiga, mesmo que o keyring ainda consiga ler ambas.

## 3. Backup e restore

Siga [`postgres-backup-restore.md`](postgres-backup-restore.md):

1. crie dump custom com permissão restrita;
2. valide a lista do dump;
3. restaure em banco isolado;
4. confirme schema e contagens do inventário;
5. remova o banco de ensaio;
6. registre checksum, localização protegida, retenção e evidência do restore.

O dump não substitui o backup das chaves. Ciphertexts restaurados só são úteis
se suas versões de chave continuarem disponíveis.

## 4. Dry-run obrigatório

Execute `--dry-run` com a mesma versão alvo e tamanho de lote planejados para a
campanha. Chaves e DSN vêm exclusivamente do ambiente injetado pelo secret
manager:

```text
server/bin/tenancit-rewrap --dry-run --target-version <n> --batch-size <n> --max-duration <duração>
```

O dry-run deve:

- adquirir o advisory lock da campanha;
- verificar configuração e versões;
- percorrer e autenticar 100% dos ciphertexts;
- produzir somente contagens e métricas;
- terminar sem updates.

**Gate:** zero falhas de decrypt, zero linhas malformadas e contagem de updates
igual a zero. Capture o sumário sanitizado, nunca a saída com dados sensíveis.

## 5. Executar por lotes

1. confirme novamente health, backup, configuração e responsáveis online;
2. inicie o job com `--confirm-write` e `--job-id <uuid-da-mudança>`;
3. acompanhe batches concluídos, linhas restantes, falhas, locks, latência e
   replica lag;
4. mantenha writers normais ativos, salvo se o teste de carga do ambiente exigir
   uma janela sem escrita;
5. não aumente o lote durante a campanha sem interromper e reaprovar;
6. se não houver progresso dentro da janela definida, cancele e investigue.

O job deve confirmar cada lote em uma transação. Cancelamento gracioso termina o
lote atual ou faz rollback; nunca deve deixar cipher, nonce e versão parcialmente
atualizados.

Na imagem imutável, use `deploy/docker-compose.rewrap.yml`. O arquivo de
ambiente indicado por `TENANCIT_REWRAP_ENV_FILE` deve ter modo `0600`, ser
gerado pelo secret manager e conter `TENANCIT_REWRAP_DATABASE_URL`, keyring AES,
URL/token/source do reporter e OTLP. Execute primeiro `rewrap-dry-run`; somente
após aprovação execute `rewrap`.

## 6. Validar conclusão

Repita o inventário. O resultado esperado é uma única versão para ciphertexts:

```sql
SELECT key_version, count(*) AS rows
FROM tenant_resource_values
WHERE value_cipher IS NOT NULL
GROUP BY key_version
ORDER BY key_version;
```

Gates obrigatórios:

- [ ] zero linhas em versões diferentes do alvo;
- [ ] zero falhas de decrypt/verificação/CAS;
- [ ] health da aplicação verde;
- [ ] smoke de identify/resolve e ETag verde;
- [ ] secrets continuam mascarados no admin por padrão;
- [ ] latência, erros e replica lag retornaram ao baseline;
- [ ] logs revisados sem plaintext, cipher, nonce ou chave;
- [ ] evidências anexadas à mudança usando somente metadados sanitizados.

## 7. Janela de observação e retirada

Mantenha as chaves antigas no secret manager durante a janela de observação. A
retirada exige aprovação e todos os gates da etapa anterior.

Antes de remover uma chave antiga, considere também backups retidos: um dump
feito antes do rewrap ainda depende dela. Escolha explicitamente entre manter a
chave pelo mesmo prazo do dump ou expirar ambos juntos.

Depois da retirada:

1. aplique a configuração sem a versão antiga;
2. reinicie/recicle as instâncias de forma controlada;
3. repita health e smoke;
4. monitore falhas `unknown key version` e autenticação GCM;
5. encerre a mudança somente após a janela definida.

## Parada, retomada e rollback

| Condição | Ação |
| --- | --- |
| Falha no preflight/dry-run | Não escrever; corrigir configuração ou triar dado malformado. |
| Falha de uma linha durante lote | Rollback do lote inteiro, parar job e preservar todas as chaves. |
| CAS conflict | Rollback, investigar concorrência/implementação e retomar somente após correção. |
| Locks sem progresso | Cancelar graciosamente, identificar transações bloqueadoras e retomar depois. |
| Processo/host cai | Manter serviço com todas as chaves; reiniciar o mesmo alvo para retomar. |
| Métricas da aplicação degradam | Parar job; lotes concluídos continuam válidos e o banco misto permanece legível. |
| Nova chave suspeita após writes | Parar e preservar a nova chave; não baixar a versão corrente. Escalar incidente. |
| Restauração necessária | Restaurar dump em banco novo e validar antes de trocar o DSN. |

O rollback preferido é parar e retomar, não recriptografar para trás. Se a chave
antiga estiver comprometida, jamais migrar dados de volta para ela. Restore é
último recurso e exige todas as chaves usadas pelo snapshot.

## Resposta a comprometimento

1. acione o processo de incidente e restrinja acesso à chave exposta;
2. gere uma versão nova e escolha o cutover seguro: execute as Fases A/B acima
   se todas as réplicas já suportam stage, ou congele writers e faça restart
   coordenado; nunca introduza writes novos que algum nó ativo não consiga ler;
3. após a Fase B, bloqueie novos writes sob a chave comprometida. Preserve-a
   temporariamente, com acesso mínimo, nas réplicas ativas e no processo de
   rewrap enquanto ainda houver ciphertexts antigos; removê-la antes disso torna
   dados existentes indisponíveis;
4. execute backup, dry-run e campanha com prioridade, sem remover gates de
   integridade;
5. valide zero linhas antigas e revogue a chave comprometida;
6. avalie exposição de backups, logs, hosts e secret manager;
7. registre linha do tempo e evidências sem material sensível.

## Evidência mínima para encerrar

- identificador e aprovadores da mudança;
- versão origem e alvo;
- contagens antes/depois;
- checksum e restore testado do backup;
- sumário do dry-run e do job;
- métricas de duração, erros e linhas restantes;
- resultado do health/smoke;
- instante de retirada da chave antiga ou justificativa de retenção;
- confirmação de que nenhum segredo foi incluído nas evidências.

## Validação realizada e gate de produção

O ensaio local automatizado cobre restore clonado, falha no meio do lote,
retomada/rerun, writer concorrente, dois processos, chave histórica ausente,
tamper e retirada da chave antiga. O backup/restore custom foi comprovado na
história de continuidade. Antes da primeira produção, repetir tudo no restore
do alvo e anexar as evidências sanitizadas; sem isso, não retirar chave real.
