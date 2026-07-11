# Deploy em host

**Status:** PLANO — não executar como receita pronta.

Ainda não existe um primeiro host, domínio, proxy/TLS, secret manager ou modelo
de observabilidade definidos. Por isso o repositório não copia a automação
Ansible, CIDRs, `network_mode: host` ou unidades específicas do `reference implementation`.

Invariantes para o futuro playbook:

- dry-run/check e diff antes da aplicação;
- deploy cotidiano separado de hardening e de ações destrutivas;
- volume PostgreSQL com ownership/retention explícitos;
- secrets fora do Compose versionado;
- TLS antes de expor `/v1/resolve`;
- smoke pós-deploy obrigatório;
- rollback por imagem anterior + DSN preservado.

Quando o ambiente-alvo for escolhido, registrar topologia, trusted proxies,
CIDRs administrativos, backup externo e SLOs antes de criar automação.
