# Política de segurança

## Versões suportadas

Enquanto o Tenancit não publicar releases estáveis, somente o commit mais
recente de `main` recebe correções de segurança. Após o primeiro release, esta
seção passará a listar explicitamente as linhas suportadas.

## Como reportar uma vulnerabilidade

Não abra uma issue pública com tokens, segredos, dados pessoais, detalhes de
exploração ou evidências de ambiente. Prefira o formulário privado **Report a
vulnerability** na aba Security do GitHub. Se ele não estiver disponível,
envie o relato para `djalmajr@gmail.com` com o assunto `[Tenancit Security]`.

Inclua, quando possível:

- versão ou commit afetado;
- pré-condições e impacto observado;
- passos mínimos para reprodução;
- sugestão de correção, se houver;
- qualquer prazo de divulgação coordenada relevante.

O recebimento será confirmado em até 3 dias úteis. A triagem inicial deve
ocorrer em até 7 dias úteis. Prazos de correção e divulgação serão combinados
conforme impacto, complexidade e disponibilidade de mitigação.

## Escopo e pesquisa responsável

São especialmente relevantes falhas de autenticação/autorização, isolamento
entre tenants, exposição de secrets, SSRF em integrações, bypass de rate limit,
adulteração de auditoria e vulnerabilidades na cadeia de build/deploy.

Ao pesquisar, não acesse dados de terceiros, não interrompa serviços, não
persista acesso e use somente ambientes e credenciais sob seu controle. A
manutenção avaliará de boa-fé pesquisas que respeitem esses limites.

Implantações são responsáveis por TLS, IdP, secret manager, backups, rede e
atualização de dependências conforme os runbooks do projeto.
