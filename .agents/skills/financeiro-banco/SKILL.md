---
name: financeiro-banco
description: Preparar e executar provisionamento PostgreSQL, migrations incrementais, alterações de schema e permissões do Financeiro-GO, com inspeção e validação. Use para criação ou evolução do banco; execução de alterações exige autorização específica.
---

# Banco do Financeiro-GO

Siga AGENTS.md da raiz, reutilizando o contexto válido. Leia plano, arquitetura e somente os arquivos da operação. Esta skill cuida de banco; não duplica o fluxo Git/PR. Se financeiro-etapa já estiver ativa, continue na mesma feature e devolva a ela o resultado. Uma chamada isolada não concede permissão de publicação.

## Definir e inspecionar

Distinga análise, preparação de arquivos e execução. Um pedido explícito para executar uma operação e destino definidos vale como autorização dessa operação; não pergunte de novo. Chamar apenas o nome da skill permite inspecionar e preparar, mas não executar alterações indefinidas ou todas as futuras.

Use o ambiente/configuração local sem imprimir segredos. Confirme servidor, current_database(), usuário e privilégios por leitura. Os destinos da aplicação são financeiro_go e financeiro_go_test; postgres serve apenas para administração do provisionamento. Nunca conectar ao banco Fiscal-GO nem adotar suas tabelas/credenciais.

Antes de alterar banco existente, consulte somente os objetos relevantes em pg_catalog: tabelas, colunas/tipos, nullability/defaults, constraints, índices, dependências, roles/grants e versão/checksum das migrations. Não leia registros pessoais. Histórico documental não substitui inspeção atual. Sem conexão, prepare uma proposta com premissas explícitas e deixe a execução pendente.

## Preparar o incremento

- **Criação inicial:** leia [bootstrap](../../../migrations/bootstrap/0000_create_databases.sql) e [preparação](../../../doc/04-preparacao-integracao-postgres.md). Confira nomes de bancos e roles para evitar colisão; não redefina usuário existente. Proponha usuários exclusivos sem superusuário. Credenciais ficam fora de SQL, fixtures, logs e Git.
- **Evolução:** adicione migration numerada sem editar as já aplicadas. Preserve dados e compatibilidade do adaptador; planeje backfill, validação e ordem de implantação quando necessários. Mudança destrutiva exige escopo explícito e recuperação proporcional ao impacto; nunca apagar/recriar o banco como atalho.
- **Executor:** confira migrations/migrations.go e cmd/migrate/main.go. Na base da fase 3, Apply/Check reconhecem apenas 0001; criar 0002.sql sozinho não a executa. Se isso ainda for verdade, implemente/teste suporte incremental a versões ordenadas, checksum, bloqueio e pendências antes da nova migration. Não modifique 0001 para contornar essa limitação.

Apresente arquivos, banco alvo, efeito esperado e riscos concretos. Havendo autorização anterior suficiente, prossiga. Caso falte, peça uma única autorização para o conjunto exato preparado, incluindo testes que gravem fixtures; continue trabalhos independentes. Não transforme aprovação de criar SQL em autorização para executá-lo.

## Executar e validar

Revalide destino e estado imediatamente antes da escrita. Execute somente o SQL aprovado. Use transação quando suportada; CREATE DATABASE exige execução separada, fora de transação explícita. Em falha parcial, inspecione os objetos criados antes de retomar; não repita às cegas nem remova recursos automaticamente.

Para migrations da aplicação, use o executor do projeto com conexão explícita e usuário sem superusuário. Verifique versão/checksum e catálogo após a execução. Não reaplique migration confirmada nem altere a tabela de versões para esconder divergências.

Rode integração somente em financeiro_go_test: confira FINANCEIRO_TEST_DATABASE_URL e o teste antes de executar. Use fixtures fictícias e limpe somente as próprias fixtures; nunca TRUNCATE ou DROP indiscriminados. Verifique o comportamento afetado, atomicidade/rollback e concorrência quando houver saldo ou transações. SKIP não é aprovação de integração. Siga os checks Go do AGENTS.md quando código mudar.

Registre no documento da etapa o que foi preparado, executado e comprovado, distinguindo banco de teste e aplicação. Atualize o plano sem antecipar conclusão. Informe pendências à financeiro-etapa para que ela prepare o PR; esta skill não faz merge.
