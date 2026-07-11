# Contribuindo com o Tenancit

Obrigado pelo interesse. Antes de começar uma mudança grande, abra uma issue
descrevendo o problema e a fronteira proposta. Correções pequenas e melhorias
documentais podem seguir diretamente para pull request.

## Desenvolvimento

1. Instale Go 1.25, Bun 1.3 e Docker com Compose.
2. Copie `.env.example` para `.env` apenas localmente.
3. Use `make dev-compose` para Vite com proxy e API local.
4. Não inclua tokens, dumps, screenshots com secrets nem caminhos pessoais.

Antes de enviar:

```bash
make test
make build
make e2e-catalog
```

Mudanças de schema devem seguir expand/contract, incluir migration reversível
na fase expand e manter consultas sqlc sincronizadas. Mudanças de segurança
devem incluir um teste negativo que reproduza o limite protegido.

## Pull requests

- mantenha o escopo pequeno e explique o porquê, não apenas o que mudou;
- atualize contratos, ADRs e runbooks quando o comportamento público mudar;
- preserve pt-BR, en-US e es-ES nas superfícies de UI;
- indique gates executados e limitações conhecidas;
- não misture refactors não relacionados.

Ao contribuir, você concorda que sua contribuição será disponibilizada sob a
mesma [O'Saasy License](LICENSE) do projeto.

Relatos de vulnerabilidade seguem o processo privado em [SECURITY.md](SECURITY.md).
