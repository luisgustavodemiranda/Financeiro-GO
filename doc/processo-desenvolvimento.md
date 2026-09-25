# Processo de desenvolvimento

Este projeto é desenvolvido como estudo em Go e portfólio público. O plano distingue código preparado de funcionalidades verificadas em execução real. Somente dados fictícios entram nos exemplos e testes.

## Ciclo de cada incremento

1. Definir o comportamento esperado, os limites e os critérios de conclusão.
2. Explicar os conceitos de Go envolvidos e os arquivos que serão alterados.
3. Implementar uma mudança pequena, mantendo handlers, serviços e interfaces de repositório.
4. Testar regras financeiras, erros e contratos HTTP relevantes.
5. Executar `gofmt`, `go test ./...`, `go vet ./...` e `go build ./...`.
6. Documentar resultados reais, limitações e próximo passo em `doc/`.
7. Ao invocar financeiro-etapa, revisar o diff, fazer commit e push da feature e abrir/atualizar PR para develop automaticamente, deixando a revisão e o merge para o usuário.

## Git e revisão

A publicação inicial reúne a fundação, a fase 2 e a preparação da fase 3, sem inventar commits retroativos. As branches permanentes são `main` (versões estáveis) e `develop` (integração). Cada etapa começa em `feature/<etapa>`, a partir de `develop`; por exemplo, `feature/fase-3-integracao-postgres`. Pull Requests de feature têm `develop` como destino. A promoção de `develop` para `main` ocorre por PR de versão, após validação e autorização de merge.

O primeiro commit estabelece a base compartilhada por main e develop; não representa conclusão da fase 3. Novos incrementos seguem o fluxo de features. O workflow `Backend checks` executa formatação, testes, vet e build. Integração PostgreSQL continua opt-in; CI sem banco não comprova o funcionamento do SQL.

## Skill para cada etapa

A skill versionada [financeiro-etapa](../.agents/skills/financeiro-etapa/SKILL.md) orienta a preparação da branch, implementação, testes e documentação. Exemplo de pedido:

```text
Use $financeiro-etapa para iniciar a etapa fase-3-integracao-postgres.
Verifique as pendências do plano e prepare os próximos passos sem executar migrations.
```

Por autorização permanente do usuário, cada chamada de financeiro-etapa cobre criar/retomar feature, implementar, testar, documentar, fazer commit, push e abrir/atualizar PR com destino develop, sem confirmações intermediárias. Se já houver PR da feature, ele será atualizado. Não criar commits vazios nem PR duplicado. Pedidos de somente análise ou trabalho local prevalecem sobre esse padrão. Uma árvore com mudanças locais deve ser preservada e compreendida antes de trocar de branch.

O PR fica pronto para revisão quando os checks exigidos passam e o incremento está concluído. Se restar bloqueio externo, como autorização para migrations, publicar apenas uma preparação coerente e deixar o PR em rascunho, com a pendência explícita. Merge e alterações de banco continuam exigindo autorização específica; a skill nunca faz merge automático.

Antes de publicar, revisar os arquivos incluídos e o `.gitignore`. Nunca incluir `.env`, senhas, faturas ou dados pessoais. O `.env.example` contém somente estrutura de configuração, sem credenciais reais. A API local não deve ser apresentada como serviço público pronto para produção: autenticação, frontend e validação operacional ainda são pendências do plano.

## Banco de dados

Os bancos são exclusivos do Financeiro-GO. Criação e execução de migrations exigem autorização específica; não fazem parte de uma autorização de commit ou publicação. O Fiscal-GO permanece apenas como referência de arquitetura.

A skill [financeiro-banco](../.agents/skills/financeiro-banco/SKILL.md) concentra inspeção, criação inicial, evolução de schema/permissões, migrations e validação. financeiro-etapa a consulta somente quando necessário e mantém a mesma feature/PR. Chamada isolada da skill de banco permite análise e preparação; uma ordem explícita de executar uma operação definida autoriza essa operação sem confirmação repetida.

```text
Use $financeiro-banco para revisar o provisionamento preparado e suas pendências.
```

Para alterações futuras, a skill exige inspecionar o schema real e preservar migrations aplicadas. O executor atual suporta apenas 0001; a skill orienta evoluí-lo e testá-lo antes de uma migration incremental adicional. Nenhuma nova capacidade do executor foi implementada por criar a skill.

## Revisão de eficiência do processo

As regras comuns de contexto e verificação ficam no AGENTS.md. financeiro-etapa foi encurtada, e detalhes de banco são carregados somente quando o assunto exige. A revisão reduziu repetição entre instruções, não removeu validação financeira ou controle de mudanças no banco.

Na execução, reaproveitar fontes lidas enquanto a versão for válida; buscar trechos com rg; agrupar consultas independentes; evitar imprimir documentos e logs inteiros. Reutilizar checks do mesmo código e CI do mesmo SHA; repetir quando a mudança, falha ou dúvida justificar. Para Markdown apenas, revisar diff, links e skills sem repetir a suíte Go local; o CI continua rodando no push.

Ambas as skills passaram no quick_validate.py. A revisão de instruções cobriu criação inicial sem autorização, autorização já concedida, migration futura com executor ainda limitado a 0001, retomada de feature existente e mudança somente documental. Essa revisão não é um teste executado contra PostgreSQL. Não houve acesso ou alteração de banco nesta revisão.

Não foi medida economia de tokens por sessão: ela depende da tarefa e do contexto. O ganho esperado é reduzir leituras e saídas redundantes. Um resumo de continuidade deve registrar apenas branch/PR, resultado, checks e pendências no documento da etapa, sem criar cópias do plano ou do schema.

## Próximo incremento

A fase 3 foi validada conforme [05-integracao-postgres-validada.md](05-integracao-postgres-validada.md). Cadastro de cartões e executor incremental foram preparados em [06-cadastro-cartoes.md](06-cadastro-cartoes.md). Próximo passo: aplicar a migration 0002 após autorização e validar a integração real antes de avançar para compras/faturas.
