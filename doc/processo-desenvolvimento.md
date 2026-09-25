# Processo de desenvolvimento

Este projeto é desenvolvido como estudo em Go e portfólio público. O plano distingue código preparado de funcionalidades verificadas em execução real. Somente dados fictícios entram nos exemplos e testes.

## Ciclo de cada incremento

1. Definir o comportamento esperado, os limites e os critérios de conclusão.
2. Explicar os conceitos de Go envolvidos e os arquivos que serão alterados.
3. Implementar uma mudança pequena, mantendo handlers, serviços e interfaces de repositório.
4. Testar regras financeiras, erros e contratos HTTP relevantes.
5. Executar `gofmt`, `go test ./...`, `go vet ./...` e `go build ./...`.
6. Documentar resultados reais, limitações e próximo passo em `doc/`.
7. Revisar o diff e registrar o incremento no Git; commit e publicação dependem do escopo autorizado na conversa.

## Git e revisão

A publicação inicial reúne a fundação, a fase 2 e a preparação da fase 3, sem inventar commits retroativos. As branches permanentes são `main` (versões estáveis) e `develop` (integração). Cada etapa começa em `feature/<etapa>`, a partir de `develop`; por exemplo, `feature/fase-3-integracao-postgres`. Pull Requests de feature têm `develop` como destino. A promoção de `develop` para `main` ocorre por PR de versão, após validação e autorização de merge.

O primeiro commit estabelece a base compartilhada por main e develop; não representa conclusão da fase 3. Novos incrementos seguem o fluxo de features. O workflow `Backend checks` executa formatação, testes, vet e build. Integração PostgreSQL continua opt-in; CI sem banco não comprova o funcionamento do SQL.

## Skill para cada etapa

A skill versionada [financeiro-etapa](../.agents/skills/financeiro-etapa/SKILL.md) orienta a preparação da branch, implementação, testes e documentação. Exemplo de pedido:

```text
Use $financeiro-etapa para iniciar a etapa fase-3-integracao-postgres.
Verifique as pendências do plano e prepare os próximos passos sem executar migrations.
```

A criação da feature local faz parte de iniciar uma etapa. Commit, push, abertura de PR e merge respeitam o escopo autorizado na conversa; invocar a skill não concede autorização automática para essas ações nem para alterações de banco. Não se repete uma confirmação que já tenha sido concedida. Uma árvore com mudanças locais deve ser preservada e compreendida antes de trocar de branch.

Antes de publicar, revisar os arquivos incluídos e o `.gitignore`. Nunca incluir `.env`, senhas, faturas ou dados pessoais. O `.env.example` contém somente estrutura de configuração, sem credenciais reais. A API local não deve ser apresentada como serviço público pronto para produção: autenticação, frontend e validação operacional ainda são pendências do plano.

## Banco de dados

Os bancos são exclusivos do Financeiro-GO. Criação e execução de migrations exigem autorização específica; não fazem parte de uma autorização de commit ou publicação. O Fiscal-GO permanece apenas como referência de arquitetura.

## Próximo incremento

Concluir a fase 3: provisionar os bancos exclusivos, aplicar a migration autorizada e executar os testes de integração real, verificando persistência, rollback e concorrência.
