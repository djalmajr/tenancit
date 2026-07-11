# Design: rewrap de chaves AES

- **Status:** Aceito e implementado; validação local concluída
- **Data:** 2026-07-09
- **Plano:** `plans/022-aes-key-rewrap-spike.md`
- **Escopo:** recriptografar secrets existentes sob a chave AES corrente sem
  alterar o valor lógico entregue aos consumidores

## Resultado da decisão

A implementação é um comando operacional offline e explícito,
executado separadamente do servidor HTTP. O comando processa lotes curtos em
transações independentes, seleciona linhas com `FOR UPDATE SKIP LOCKED`, abre o
ciphertext com a chave indicada por `key_version`, cifra o mesmo plaintext com a
versão corrente e persiste o novo `value_cipher`, `nonce` e `key_version` por
compare-and-swap (CAS).

O job não deve rodar no boot, em requests nem por endpoint administrativo. Uma
API disparadora amplia a superfície de ataque e depende antes de identidade
humana, RBAC e audit log.

## Base existente

O desenho parte destes contratos já implementados:

- `crypto.FromEnv` carrega a chave corrente de `TENANCIT_AES_KEY`, a versão de
  `TENANCIT_AES_KEY_VERSION` e todas as entradas adicionais válidas de
  `TENANCIT_AES_KEY_V<n>`, inclusive uma versão futura staged;
- `Cryptor.Decrypt` escolhe a chave registrada em `key_version`;
- `Cryptor.Encrypt` usa AES-256-GCM, nonce aleatório e sempre grava a versão
  corrente;
- `tenant_resource_values` separa `value_plain` de
  `value_cipher`/`nonce`/`key_version`;
- o AES-GCM atual usa `nil` como additional authenticated data (AAD). O rewrap
  precisa manter esse contrato até uma migração separada.

Somente linhas com `value_cipher IS NOT NULL` são candidatas. Valores em claro
não podem ser modificados. Para um ciphertext válido, `nonce` e `key_version`
também precisam existir; dados fora dessa invariante bloqueiam a campanha.

## Contrato do comando

Interface implementada:

```text
server/bin/tenancit-rewrap \
  --target-version <n> \
  --batch-size <n> \
  [--dry-run] \
  [--confirm-write --job-id <uuid>] \
  [--max-duration <duration>]
```

Regras:

1. `--target-version` deve ser igual à versão corrente do `Cryptor`.
2. A nova versão deve ser positiva, inédita e maior que todas as versões já
   usadas. A progressão é uma política operacional que evita reutilização e
   downgrade, não uma limitação do loader.
3. `--batch-size` tem limite mínimo e máximo definidos no binário. O default
   inicial recomendado é 100, ajustável após medir locks e latência.
4. `--dry-run` percorre e autentica todos os ciphertexts, mas não cifra nem
   grava dados.
5. O comando exige confirmação explícita para escrita e recusa execução se o
   preflight, o backup ou a configuração das chaves não tiver evidência.
6. O processo adquire um advisory lock de campanha. A implementação aceita
   um único coordenador; `SKIP LOCKED` mantém lotes seguros diante de writers e
   permite paralelismo futuro sem redesenhar a consulta.

## Preflight fail-closed

Antes de iniciar qualquer lote de escrita, o comando deve:

1. conectar ao mesmo banco e schema do serviço com TLS conforme o ambiente;
2. adquirir o advisory lock exclusivo da campanha;
3. verificar que a versão alvo é a versão corrente e que todas as versões
   encontradas no banco têm uma chave carregada;
4. inventariar contagens por `key_version`, sem imprimir cipher, nonce ou
   plaintext;
5. rejeitar linhas cifradas com `key_version` nulo ou não positivo, nonce nulo
   ou de tamanho incompatível, ou ciphertext curto demais para a tag GCM;
6. executar um dry-run completo: `Decrypt` de cada linha em páginas limitadas,
   descartando o plaintext imediatamente e contabilizando falhas;
7. confirmar que existe backup recente e que seu restore foi ensaiado;
8. registrar `job_id`, alvo, tamanho de lote, total estimado e identidade do
   operador, sem material criptográfico.

Qualquer chave ausente, falha de autenticação GCM ou linha malformada termina o
preflight com código de saída não zero e **zero updates**.

## Algoritmo transacional por lote

Cada lote usa uma nova transação:

1. selecionar, de forma determinística, até `batch_size` linhas cifradas cuja
   versão seja diferente da versão alvo;
2. bloquear as candidatas com `FOR UPDATE SKIP LOCKED`;
3. para cada candidata, descriptografar usando seu `key_version` original;
4. cifrar o plaintext com a chave corrente, obtendo nonce novo;
5. descriptografar o resultado recém-gerado e comparar com o plaintext em
   memória antes de persistir;
6. atualizar por CAS, exigindo id, versão, cipher e nonce originais;
7. exigir exatamente uma linha atualizada por candidata;
8. fazer commit somente após todas as linhas do lote passarem;
9. apagar referências aos buffers assim que possível, registrar apenas
   contagens e avançar para o próximo lote.

Consulta de seleção proposta:

```sql
SELECT id, value_cipher, nonce, key_version
FROM tenant_resource_values
WHERE value_cipher IS NOT NULL
  AND key_version IS DISTINCT FROM $1
ORDER BY id
LIMIT $2
FOR UPDATE SKIP LOCKED;
```

Update CAS proposto:

```sql
UPDATE tenant_resource_values
SET value_cipher = $1,
    nonce = $2,
    key_version = $3
WHERE id = $4
  AND key_version = $5
  AND value_cipher = $6
  AND nonce = $7;
```

O CAS não substitui o row lock; ele detecta regressões se a implementação
futura separar leitura e escrita por engano. Resultado diferente de uma linha
é conflito e causa rollback do lote.

O rewrap **não** toca `tenant_resources.updated_at`. Cipher e nonce mudam, mas
o plaintext e o contrato de resolução não; portanto o ETag funcional não deve
ser invalidado apenas por manutenção criptográfica.

## Concorrência

| Cenário | Comportamento esperado |
| --- | --- |
| Provision/update conclui antes do lock | O job lê e recriptografa o valor mais recente. |
| Provision/update tenta durante o lote | O writer espera o row lock; depois do commit, sua escrita com a chave corrente prevalece. |
| Linha já está na versão alvo | É ignorada, tornando reruns idempotentes. |
| Linha está bloqueada por outro writer | `SKIP LOCKED` a deixa para outro ciclo. |
| Processo encerra no meio do lote | PostgreSQL desfaz o lote inteiro; lotes anteriores permanecem válidos. |
| Dois coordenadores são iniciados | O segundo falha no advisory lock. |

Um lote vazio não significa conclusão: pode haver candidatas temporariamente
bloqueadas. O processo consulta a contagem restante fora da seleção e só encerra
com sucesso quando ela chega a zero. Se não houver progresso por uma janela
configurável, encerra com erro recuperável em vez de fazer loop infinito.

## Cutover seguro em múltiplas réplicas

Mudar chave e versão corrente em um único rolling deploy é inseguro: a primeira
réplica nova pode escrever na versão alvo enquanto uma réplica ainda antiga não
conhece essa chave. O cutover exige duas fases independentes:

1. **Stage de leitura:** primeiro implantar em todas as réplicas a versão do
   loader que aceita keyring independente da corrente. Mantendo writes na versão
   antiga, distribuir `TENANCIT_AES_KEY_V<alvo>` e reciclar todas as réplicas.
   Evidência por instância deve confirmar que a versão alvo está carregada, sem
   registrar a chave.
2. **Troca de escrita:** somente depois, fazer rolling deploy com
   `TENANCIT_AES_KEY=<nova>`, `TENANCIT_AES_KEY_VERSION=<alvo>` e a chave antiga
   preservada em `TENANCIT_AES_KEY_V<origem>`. Réplicas ainda não atualizadas
   nesta segunda fase conseguem decriptar writes novos porque receberam a chave
   alvo na fase de stage.

O job de rewrap só começa quando todas as réplicas escrevem na versão alvo. Se o
orquestrador não consegue provar que stage e reciclagem chegaram a todas as
instâncias, a alternativa segura é congelar writers e fazer um restart
coordenado; nunca prosseguir assumindo que um rolling deploy monofásico basta.

## Idempotência e retomada

O banco é o checkpoint. Não há cursor externo obrigatório:

- linhas na versão alvo já estão concluídas;
- linhas em versão antiga ainda estão pendentes;
- rollback transacional elimina estados pela metade dentro de uma linha;
- executar novamente o mesmo alvo continua do estado persistido;
- um job interrompido não exige desfazer lotes já confirmados.

O job não deve tentar “consertar” automaticamente linhas malformadas. Essas
linhas exigem triagem separada, mantendo a campanha fail-closed.

## Observabilidade sem vazamento

Métricas mínimas:

- `tenancit.rewrap.rows.by_version{from_version,to_version,outcome}`;
- `tenancit.rewrap.rows.remaining.by_version{key_version}`;
- `tenancit.rewrap.batches{outcome}` e `tenancit.rewrap.batch.duration`;
- `tenancit.rewrap.failures{failure_reason}` para autenticação, verificação,
  CAS, locks sem progresso e demais classes fechadas;
- `tenancit.rewrap.campaigns`, linhas processadas/restantes e duração total;
- duração total e timestamp do último progresso.

Logs estruturados podem conter `job_id`, versão origem/destino, número do lote,
contagens, duração e classe do erro. Eles não podem conter plaintext,
ciphertext, nonce, chave, variável de ambiente, payload de resource ou token.
IDs de linha só devem aparecer em canal operacional restrito e, por padrão,
devem ser substituídos por uma referência opaca correlacionável ao `job_id`.

## Controles de segurança

- chaves vêm exclusivamente do secret manager/ambiente do processo;
- o operador nunca fornece chaves por argumentos de linha de comando;
- o job roda em host controlado, com egress mínimo e logs restritos;
- a credencial de banco tem somente as permissões necessárias para ler e
  atualizar `tenant_resource_values` e adquirir o lock definido;
- plaintext não vai para disco, arquivos temporários, métricas ou logs;
- lotes pequenos limitam tempo de lock e quantidade de plaintext residente;
- o processo aceita cancelamento gracioso: termina o lote atual ou faz rollback;
- o código futuro deve minimizar a vida do plaintext. Como a API atual de
  `Decrypt` retorna `string`, zeroização confiável não é possível; uma API
  baseada em `[]byte` pode ser adicionada como hardening antes de produção;
- a chave antiga permanece disponível durante toda a campanha e a janela de
  observação.

## Backup, rollback e retirada da chave

Antes da escrita, é obrigatório capturar dump PostgreSQL e validar o restore em
banco isolado, seguindo `docs/runbooks/postgres-backup-restore.md`. O backup das
chaves é separado do dump e permanece no secret manager.

O rollback primário é **parar o job e manter as chaves nova e antigas**. Um
banco misto é legível pelo serviço, e executar novamente retoma o progresso.
Não se deve reduzir `TENANCIT_AES_KEY_VERSION` depois do primeiro lote. Mesmo com
o keyring capaz de ler versões staged, um downgrade retomaria novos writes sob a
chave antiga e quebraria o objetivo da campanha. Todas as versões necessárias
permanecem carregadas até o fim da observação.

Restore do dump é contingência final e deve ocorrer em banco novo, mantendo o
banco original intacto. Ele exige conservar todas as chaves correspondentes ao
snapshot. Se a chave antiga estiver comprometida, não se recriptografa de volta
para ela; o objetivo é concluir a migração, validar e revogar a chave exposta.

A chave antiga só pode ser removida quando:

1. a contagem de ciphertexts fora da versão alvo for zero;
2. não houver linhas cifradas malformadas;
3. o serviço e o smoke de resolução passarem com a nova configuração;
4. a janela de observação terminar sem falhas de decrypt;
5. backups que ainda dependem da chave antiga tiverem política explícita de
   retenção ou expiração;
6. exports de auditoria e envelopes de idempotência cifrados na versão antiga
   tiverem expirado e o cleanup confirmado (máximo contratual atual: 24 h).

## Estrutura implementada

A entrega ficou separada em componentes revisáveis:

1. API byte-oriented do `Cryptor` expõe somente metadados de versão e permite
   apagar buffers de plaintext;
2. `internal/rewrap` implementa inventário paginado, autenticação integral,
   advisory lock, `SKIP LOCKED`, CAS e retomada pelo banco;
3. `cmd/tenancit-rewrap` fornece CLI offline, cancelamento por sinal, deadline,
   JSON sanitizado, OTLP e report operacional autenticado;
4. `/tenancit-rewrap` integra a imagem imutável; Compose separado diferencia
   dry-run e escrita confirmada;
5. o login `tenancit_rewrap` possui SELECT/UPDATE apenas nas colunas cifradas e
   leitura dos reports de segurança.

Não é necessária migration de schema para o algoritmo básico. Constraints para
fortalecer a invariante cipher/nonce/key_version podem ser uma mudança separada,
precedida por inventário dos dados existentes.

## Critérios de aceite e evidências

- dry-run percorre todas as linhas e não altera nenhum byte no banco;
- mistura de linhas em versões antigas, alvo e valores em claro termina com
  todos os ciphertexts na versão alvo e valores em claro inalterados;
- o plaintext antes/depois é idêntico, enquanto cipher e nonce mudam;
- chave histórica ausente, ciphertext adulterado ou linha malformada falha antes
  do primeiro update;
- falha injetada no meio do lote faz rollback integral daquele lote;
- reinício retoma e não reprocessa linhas já no alvo;
- atualização concorrente não é sobrescrita por valor obsoleto;
- dois processos não executam a mesma campanha simultaneamente;
- duas fases de cutover provam que uma réplica ainda na corrente antiga consegue
  decriptar um ciphertext da versão staged antes de qualquer write novo;
- conclusão exige `rows_remaining = 0` e smoke de resolução bem-sucedido;
- logs e métricas passam por teste que rejeita plaintext, cipher, nonce e chaves;
- restore do backup é ensaiado antes da primeira campanha de produção.

Os testes de integração cobrem dry-run byte-idêntico, lotes de tamanho um,
rerun idempotente, chave histórica ausente, ciphertext adulterado, nonce
malformado, evidência ausente, rollback do lote por falha injetada, advisory
lock, timeout sem progresso, writer concorrente, clone restaurado e leitura sem
a chave antiga após conclusão. O CLI completo publica report por credencial
dedicada e não aceita key material em argumentos.

## Fora da implementação atual

- execução de uma rotação real;
- HSM, Vault Transit ou envelope encryption;
- rotação multi-região;
- disparo por API administrativa;
- mudança do formato criptográfico ou introdução de AAD.
