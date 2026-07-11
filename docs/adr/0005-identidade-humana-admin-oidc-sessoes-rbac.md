# ADR 0005 — Identidade humana administrativa com OIDC, sessões e RBAC

- **Status:** Aceito; implementação em andamento no epic 03
- **Data:** 2026-07-09
- **Relacionados:** [ADR 0004](./0004-admin-token-e-api-clients.md) e
  [design de auditoria administrativa](../developers/design/admin-audit-log.md)

## Contexto

Hoje toda a superfície `/v1/admin/*` usa um único
`TENANCIT_ADMIN_TOKEN`. O servidor compara seu hash em memória e a SPA guarda o
token bruto em `localStorage` para enviá-lo como Bearer. Essa fronteira impede
acesso anônimo, mas tem limitações estruturais para operação humana:

- todas as pessoas compartilham a mesma identidade técnica;
- não há logout ou revogação de uma sessão individual;
- não há autorização por papel ou por ação sensível;
- a presença prolongada do token no browser amplia o impacto de XSS;
- a auditoria só consegue atribuir ações à credencial compartilhada, não a uma
  pessoa autenticada.

O token continua útil para recuperação quando o provedor de identidade está
indisponível. Ele não deve, porém, ser o login cotidiano nem virar fallback
silencioso diante de uma configuração OIDC incompleta.

Também não há topologia de produção fixada para ingress, proxies ou redes
administrativas. Portanto, IP recebido por header, allowlist de CIDR e regras
de trusted proxy ainda não têm uma fonte confiável.

## Decisão

Adotar autenticação humana administrativa por OIDC, com Authorization Code +
PKCE no backend, sessão opaca server-side, proteção CSRF e RBAC deny-by-default.
`TENANCIT_ADMIN_TOKEN` permanece somente como credencial de break-glass.

A separação da ADR 0004 continua válida: API clients de consumo não autenticam
administradores, e sessões humanas não autenticam `/v1/identify` ou
`/v1/resolve*`. Esta ADR evolui apenas a autenticação da superfície admin.

### Fluxo OIDC e validação do IdP

O binário Go será o cliente OIDC confidencial e implementará endpoints de
login, callback e logout no mesmo origin da SPA. O browser não receberá nem
persistirá access token, ID token, refresh token ou client secret do IdP.

O login seguirá este fluxo:

1. o backend gera `state`, `nonce` e o par PKCE e os associa a uma tentativa de
   login curta, de uso único;
2. o browser é redirecionado ao authorization endpoint do issuer configurado;
3. no callback, o backend valida `state`, troca o code e valida assinatura,
   `iss`, `aud`, `exp`, `iat`, `nonce` e PKCE;
4. o backend deriva um principal estável de `iss` + `sub`, aplica o mapeamento
   explícito de grupos/claims para roles e cria a sessão;
5. a SPA recebe somente o cookie de sessão e pode consultar um endpoint como
   `/v1/admin/session` para obter nome de exibição, roles, permissões e o token
   CSRF da sessão, sem receber credencial de autenticação reutilizável.

E-mail e nome são apenas labels de exibição. Nunca substituem `iss` + `sub`
como identidade, pois podem mudar ou ser reciclados. Claims, issuer, audience,
algoritmos permitidos e mapeamentos de grupo serão allowlists de configuração;
um usuário apenas autenticado, mas sem role reconhecida, recebe acesso negado.

Discovery e JWKS terão timeouts e cache com atualização controlada. Falha de
refresh de chaves não permite ignorar assinatura, aceitar algoritmo diferente
ou reaproveitar chave expirada além da política declarada.

### Sessão server-side

A sessão será persistida no PostgreSQL, que já é a dependência de estado do
serviço. O cookie conterá somente um identificador aleatório opaco; o banco
guardará seu hash, nunca o valor reutilizável. O registro de sessão conterá no
máximo o principal validado, roles/permissões calculadas, criação, último uso,
expiração absoluta, expiração por inatividade e estado de revogação.

O cookie terá, em produção:

- nome com prefixo `__Host-`;
- `Secure`, `HttpOnly`, `SameSite=Lax` e `Path=/`;
- nenhum atributo `Domain`;
- duração limitada pela menor validade entre a política local e a identidade
  validada.

O identificador será rotacionado após login, reautenticação e mudança de
privilégios. Logout e revogação invalidam a sessão server-side imediatamente.
As sessões terão limites absoluto e por inatividade configuráveis; não haverá
sessão sem expiração. Tokens do IdP só serão persistidos se um requisito futuro
de refresh os tornar indispensáveis e, nesse caso, deverão ser cifrados e
rotacionáveis, nunca expostos à SPA ou aos logs.

### CSRF e semântica HTTP

Cookies `SameSite` não serão a única defesa. Cada sessão terá um token CSRF
aleatório, comparado em tempo constante com `X-CSRF-Token` em toda operação
administrativa que possa produzir efeito. O servidor também validará
`Origin` — e `Referer` como fallback controlado — contra o origin canônico.
Ausência ou divergência falha com `403` antes do handler.

Métodos `POST`, `PUT`, `PATCH` e `DELETE` exigem CSRF. O reveal de secrets atual,
embora seja `GET ...?reveal=true`, também produz exposição sensível: durante a
migração ele exigirá a mesma proteção e, no contrato definitivo, deverá virar
uma operação explícita não-GET. Login usa `state`/`nonce`/PKCE; logout por efeito
de estado não será um GET desprotegido.

### Principal e RBAC

O middleware de autenticação produzirá um principal verificado comum para OIDC
e break-glass, com `kind`, `issuer`, `subject`, label e permissões. Handlers não
lerão claims, grupos, cookies ou headers de identidade diretamente.

As permissões serão nomes estáveis de domínio, verificadas antes do handler,
por exemplo:

- `admin.read` para consultas administrativas comuns;
- `tenant.write` e `resource.write` para configuração operacional;
- `secret.reveal` para revelar credenciais;
- `tenant.hard_delete` para exclusão irreversível;
- `api_client.manage` para criar, revogar e reativar credenciais de consumo;
- `audit.read` para consultar/exportar a trilha administrativa.

Roles (`viewer`, `operator`, `security_admin` e futuras roles) são apenas
conjuntos configurados dessas permissões. Rotas declaram a permissão requerida;
não codificam nomes de grupos do IdP. O mapeamento grupo/claim -> role é
explícito, versionado e deny-by-default. Mudança que reduza privilégios revoga
ou reavalia sessões existentes; não espera indefinidamente o próximo login.

O principal autenticado será propagado à auditoria. Eventos OIDC usam
`actor_kind=oidc_user` e a chave durável `actor_issuer` + `actor_subject`.
Headers como `X-Actor-Email` nunca atribuem autoria.

### Break-glass

`TENANCIT_ADMIN_TOKEN` deixa de ser credencial de login da SPA. Seu uso será
reservado a recuperação operacional explícita, preferencialmente por CLI e
somente sobre TLS. O modo break-glass:

- é habilitado por configuração explícita, não automaticamente quando o IdP
  falha;
- recebe um principal técnico identificável, como
  `break_glass/admin-token:primary`, sem fingir identidade humana;
- tem permissões mínimas declaradas para o procedimento de recuperação, em vez
  de herdar acesso ilimitado por acidente;
- tem limite de tentativas mais restritivo e cada sucesso ou negativa é
  auditado sem registrar token, hash ou prefixo derivado;
- exige segredo aleatório de alta entropia vindo de secret manager;
- é rotacionado imediatamente após uso ou suspeita de exposição e de forma
  periódica mesmo sem uso.

Para rotação sem indisponibilidade em múltiplas réplicas, a implementação pode
aceitar `current` e `previous` por uma janela curta e configurada. Cada versão
terá um identificador não secreto para auditoria. O token anterior será
removido após todas as réplicas aceitarem o novo; a janela não pode se tornar
compatibilidade permanente.

### Boot fail-loud e configuração

O processo recusará iniciar quando o modo administrativo escolhido não puder
cumprir este contrato. Em especial, são erros fatais:

- configuração OIDC parcial, issuer/audience ausente ou redirect URI
  incompatível;
- discovery/JWKS inválido no bootstrap quando OIDC estiver habilitado;
- ausência de origin HTTPS/cookie seguro fora do modo local explícito;
- ausência ou ambiguidade do mapeamento mínimo de roles;
- break-glass habilitado sem token forte, ou token informado sem habilitação
  explícita;
- configuração de trusted proxy sem lista explícita de proxies confiáveis.

Indisponibilidade do IdP não habilita `TENANCIT_ADMIN_TOKEN`, não reduz RBAC e
não converte claims não validadas em identidade. Health/readiness deve expor
estado operacional sem revelar issuer interno, claims, secrets ou detalhes de
chaves.

Um modo local precisa ser nomeado e ativado explicitamente. Ele pode relaxar o
atributo `Secure` apenas para loopback, mas nesse caso usa um nome de cookie dev
sem o prefixo `__Host-`; browsers exigem `Secure` para esse prefixo. Sessões dev
não são aceitas em produção. O modo local não muda o significado de break-glass
nem pode ser inferido automaticamente por hostname.

### Rate limiting

Aplicar limites separados a início de login, callback inválido, autenticação
break-glass e operações administrativas sensíveis. Respostas limitadas usam
`429` e `Retry-After`; a chave do limiter nunca inclui token bruto, cookie ou
valor secreto.

Antes de autenticar, o limite usa somente o endereço do peer de rede observado
pelo servidor. Depois de autenticar, combina limites por principal e por
instância/deployment. Em uma única réplica, um token bucket em memória é
aceitável; em múltiplas réplicas, o limite efetivo deve estar no ingress ou em
estado compartilhado. Não se anunciará proteção global usando contadores
independentes por processo.

A escrita de negativas no audit log acontece depois do rate limit, evitando
que tráfego hostil amplifique escrita no PostgreSQL.

### CIDR e trusted proxies

Esta ADR **não** define allowlist de CIDR, não confia em `X-Forwarded-For` e não
fixa IPs de nenhum ambiente. Enquanto a topologia não estiver documentada, o
servidor considera apenas o peer direto e ignora headers encaminhados para
decisões de segurança.

Uma restrição futura por rede será apenas defesa em profundidade, nunca
substituto de OIDC, RBAC ou CSRF. Ela só poderá ser habilitada depois de existir:

1. diagrama do caminho cliente -> ingress/proxy -> Tenancit;
2. lista explícita dos proxies confiáveis e da quantidade de hops;
3. regra de rejeição para headers encaminhados vindos de peers não confiáveis;
4. testes de spoofing, IPv4/IPv6 e mudança de topologia;
5. procedimento fail-loud para configuração vazia ou inválida.

Nenhuma senha, IP, CIDR, rede de host ou configuração Ansible de outro projeto
é incorporada a esta decisão.

### Fronteira de produto e extensão

O contrato de principal, autorização, sessão e auditoria deve existir em
interfaces estáveis do serviço. A decisão futura de como empacotar integrações
OIDC/RBAC não pode alterar as garantias acima. Uma distribuição sem integração
OIDC ativa deve se descrever como operação break-glass, não como autenticação
humana multiusuário.

## Migração

A mudança será incremental e sem misturar consumer API clients com identidade
humana:

1. introduzir o principal verificado e a interface de autorização, identificando
   o uso cotidiano atual como `shared_admin_token`;
2. integrar o principal ao audit log sem atribuir pessoa a ações antigas;
3. criar tabelas de sessão/tentativa OIDC, endpoints de login e testes de
   validação negativa;
4. aplicar RBAC e CSRF a todas as rotas, incluindo reveal;
5. trocar a SPA para cookie/session endpoint e remover token administrativo de
   formulário e `localStorage`;
6. ativar rate limits, runbooks de revogação/rotação e métricas;
7. habilitar OIDC em produção; somente depois retirar o uso cotidiano do token
   e reclassificar seu modo explícito como `break_glass`.

Durante a transição, toda rota deve declarar qual autenticador e qual permissão
aceita. Não haverá janela em que ausência de middleware torne uma rota admin
pública.

## Alternativas consideradas

### Continuar com o token compartilhado

Rejeitada para uso cotidiano porque não fornece identidade humana, revogação
individual ou least privilege. Mantida apenas para break-glass.

### Entregar bearer tokens do IdP diretamente à SPA

Rejeitada porque aumenta exposição no browser, acopla handlers ao formato de
claims e dificulta revogação local. A sessão opaca mantém tokens do IdP no
backend.

### Delegar autenticação ao ingress e confiar em headers de usuário

Rejeitada enquanto não houver topologia e contrato de trusted proxies. Mesmo
em uma evolução futura, headers só poderão ser aceitos de peers autenticados e
explicitamente confiáveis; a aplicação continuará responsável por autorização
e auditoria.

### Usuários e senhas locais

Rejeitada porque adiciona armazenamento de senha, recuperação, MFA e lifecycle
de usuários a um serviço cujo domínio não é identidade. O IdP é a fonte da
identidade humana.

## Consequências

Positivas:

- autoria humana verificável por `iss` + `sub`;
- revogação individual, expiração e logout reais;
- permissões específicas para reveal, hard delete, API clients e auditoria;
- tokens do IdP e credencial break-glass deixam de permanecer no
  `localStorage`;
- indisponibilidade ou erro de configuração não degrada silenciosamente a
  segurança;
- auditoria recebe principal estável sem confiar em headers do cliente.

Custos:

- novas tabelas, endpoints e rotinas de limpeza de sessão;
- dependência operacional do IdP para novos logins;
- necessidade de gerir client secret, redirects, grupos e rotação;
- CSRF e sessão adicionam estado e testes ao backend;
- rate limit global exige ingress adequado ou estado compartilhado em HA;
- mudança do reveal para semântica não-GET exige migração do contrato e da SPA.

Riscos a controlar:

- lockout por role mapping incorreto, mitigado por validação fail-loud e
  break-glass testado;
- sessão privilegiada sobreviver a redução de acesso, mitigada por revogação ou
  reavaliação com prazo curto;
- crescimento de sessões e tentativas, mitigado por TTL e limpeza periódica;
- confiar por engano em IP encaminhado, mitigado por trusted proxy desabilitado
  até a topologia ser explícita.

## Critérios para aceitar e implementar

- testes negativos cobrem `state`, nonce, PKCE, assinatura, issuer, audience,
  expiração e role ausente;
- cookie não contém token OIDC nem principal serializado e é revogável no
  servidor;
- todas as mutações e reveal rejeitam CSRF ausente/inválido;
- cada rota admin declara permissão e testes provam deny-by-default;
- uso OIDC e break-glass produz atores distintos no audit log, sem secrets;
- a SPA não lê nem grava `TENANCIT_ADMIN_TOKEN` em storage do browser;
- configuração parcial falha no boot e nunca habilita fallback;
- rate limiting é testado, documentado e coerente com a quantidade de réplicas;
- qualquer trusted proxy/CIDR permanece desligado até existir topologia e testes
  específicos;
- runbooks exercitam logout global, revogação, rotação e recuperação via
  break-glass.

## Status

Proposto em 2026-07-09. A ADR 0004 permanece aceita até esta decisão ser
implementada. Quando aceita, esta ADR substitui somente o uso cotidiano de
`TENANCIT_ADMIN_TOKEN` na superfície administrativa; a separação de API clients
de consumo continua vigente.
