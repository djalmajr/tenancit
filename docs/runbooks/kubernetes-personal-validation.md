# Validação Kubernetes no ambiente pessoal

**Status:** VALIDADO em 2026-07-11

Este perfil implanta um laboratório real, não uma topologia de produção de
cliente. Ele prova imagem imutável, migrations, OIDC, TLS, duas réplicas,
Valkey, backup/restore e recuperação sem transformar escolhas pessoais em
defaults do produto.

## Topologia

```text
Internet -> TLS/Traefik -> 2 réplicas Tenancit -> PostgreSQL
                         |                    -> Valkey
                         +-> Dex OIDC isolado
```

- chart: `deploy/helm/tenancit`;
- perfil: `deploy/helm/tenancit/values-personal.yaml`;
- namespace/release default: `tenancit`;
- secrets: Secret Kubernetes gerado de
  `~/.config/tenancit/personal.env` com modo `0600`;
- dependências são próprias do laboratório. Ambientes de cliente devem fornecer
  PostgreSQL, Valkey e IdP conforme seus requisitos.

## Publicar e implantar

Publique a imagem pelo workflow `Publish image` e use somente o digest:

```bash
export KUBECONFIG=/caminho/para/kubeconfig
export TENANCIT_IMAGE_DIGEST=sha256:<64-hex>
./scripts/deploy-k8s-personal.sh
```

O script reconcilia secrets sem imprimi-los, instala o chart, executa a
migration como init container e aguarda app/worker. A credencial Dex fica
somente no arquivo local informado ao final.

## Gates executados

Em 2026-07-11 foram comprovados:

- login OIDC completo pelo browser e RBAC `security_admin`;
- certificados públicos válidos e sessão server-side;
- duas réplicas, com 30/30 respostas `200` durante remoção de uma delas;
- bucket global: chave de 5 RPM produziu 5 respostas `200` e 5 `429`;
- readiness `503` durante indisponibilidade controlada de Valkey e PostgreSQL,
  retornando a `200` após recuperação;
- backup custom de 130.306 bytes, checksum SHA-256, restore isolado com 30
  tabelas e 1 tenant, seguido de remoção do banco temporário;
- rewrap `--dry-run --target-version 1` sem escrita;
- token one-shot do teste revogado imediatamente.

## Limites do laboratório

- single-node: duas réplicas não toleram perda de nó ou zona;
- PostgreSQL e Valkey são single-instance com storage local;
- backup copiado para o host do operador, sem retenção off-site automatizada;
- Dex estático valida o protocolo, mas cada cliente precisa integrar seu IdP;
- nenhum SLO/RPO/RTO de produção é inferido deste ensaio.

Esses limites são gates de cada ambiente de cliente, não responsabilidades que
devam ser escondidas no perfil pessoal.

## Rollback verificado

O release foi revertido da revisão 5 para a revisão 4 (digest anterior), com
`/readyz` saudável e o tenant sentinela preservado. Em seguida, o roll-forward
restaurou o digest `sha256:b9571461...57f1fa1`, novamente com readiness saudável.
As imagens de PostgreSQL, Valkey e Dex também são fixadas por digest no perfil.
Jobs one-shot de rewrap são removidos depois da coleta da evidência.

O namespace aplica default-deny e libera somente tráfego interno necessário,
DNS, HTTPS de saída e ingress do Traefik. Os CIDRs de pods/serviços do laboratório
são parâmetros explícitos do perfil; ambientes de clientes devem fornecer os
CIDRs da própria topologia.
