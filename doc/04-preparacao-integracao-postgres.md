# Fase 3 — Preparação da integração real

Registro histórico da preparação. O usuário posteriormente criou os bancos e autorizou habilitar os logins, aplicar a migration e executar os testes. Resultado em [05-integracao-postgres-validada.md](05-integracao-postgres-validada.md).

Etapa iniciada em `feature/fase-3-integracao-postgres`, criada a partir de `develop` após fetch e confirmação de sincronização. Não havia alterações locais no início.

## Inspeção realizada

A conexão ao banco administrativo `postgres` funcionou. Foram feitas somente consultas SELECT aos catálogos para verificar os nomes dos dois bancos exclusivos do projeto e os privilégios do usuário fornecido. Nenhuma tabela de aplicação ou banco do Fiscal-GO foi acessado.

Resultado: `financeiro_go` e `financeiro_go_test` ainda não existem. A credencial fornecida pertence a um superusuário com permissão para criar bancos e roles. A API recusa superusuário; portanto, usar diretamente essa credencial na aplicação não é o caminho de configuração.

Revalidação em 25/09/2026, durante a preparação com financeiro-banco: conexão confirmada em postgres com o usuário esperado. Consultas somente leitura confirmaram ausência dos dois bancos e também dos roles financeiro_go_app e financeiro_go_test_app; nenhum conflito de nomes foi encontrado naquele momento. Revalidar novamente antes da execução. O bootstrap existente permanece adequado, sem mudanças no SQL.

Host, usuário administrativo e senha não são registrados neste documento, nas fixtures ou no SQL. Não houve execução de DDL ou migration.

## Provisionamento preparado para revisão

Arquivo: [0000_create_databases.sql](../migrations/bootstrap/0000_create_databases.sql).

| Recurso | Finalidade |
| --- | --- |
| Role `financeiro_go_app` | Proprietário e usuário dedicado ao banco da aplicação |
| Role `financeiro_go_test_app` | Proprietário e usuário dedicado ao banco de testes |
| Banco `financeiro_go` | Dados da aplicação |
| Banco `financeiro_go_test` | Fixtures descartáveis dos testes |

Os roles começam com NOLOGIN e sem superusuário, criação de bancos/roles, replicação ou bypass de RLS. A ativação exige definir credenciais independentes por canal local seguro, sem incluí-las no SQL ou logs. Neste incremento didático cada usuário é dono de seu banco e pode aplicar a migration; separar usuário de migration e de execução com permissões menores permanece uma evolução de implantação.

O bootstrap revoga conexão e criação de temporários para PUBLIC apenas nesses dois bancos novos. Não altera usuários ou bancos existentes. Antes de executá-lo, verificar também se os nomes dos roles já existem. Se houver conflito, interromper e inspecionar; não redefinir senha ou privilégios de um role existente automaticamente.

CREATE DATABASE não pode executar dentro de uma transação explícita: este bootstrap é separado do comando `cmd/migrate`. Um erro pode deixar provisionamento parcial; conferir o catálogo antes de retomar. Não apagar recursos para repetir a execução.

## Sequência após autorização

1. Revalidar nomes e criar somente os roles e bancos descritos no bootstrap.
2. Definir credenciais locais independentes, ativar os dois logins e confirmar conexão sem superusuário.
3. Aplicar `migrations/0001_accounts_entries.sql` com `cmd/migrate -apply` em cada banco usando seu respectivo usuário. Essa migration cria o schema financeiro e as tabelas accounts, entries e schema_migrations.
4. Executar `TestPostgresIntegration` usando exclusivamente o banco de testes. Ele grava e remove suas próprias fixtures fictícias.
5. Verificar a API com persistência real, executar os checks e registrar os resultados. Marcar a fase 3 concluída somente quando os critérios forem comprovados.

Critérios: persistência entre conexões/reinícios, saldo e lançamento atômicos, rollback após erros, ausência de atualizações perdidas e isolamento de fixtures no banco de testes.

## Situação ao preparar a etapa

A conexão administrativa e a inexistência dos bancos foram confirmadas por leitura. O SQL de provisionamento está preparado, mas não foi executado. A integração real permanece pendente. O usuário autorizou posteriormente o fluxo automático de commit, push e PR ao invocar financeiro-etapa; esta preparação será apresentada em PR em rascunho enquanto depender de autorização para o banco. Merge e mudanças no banco permanecem sujeitos a autorização específica.

Verificações locais: `git diff --check`, `go test ./...`, `go vet ./...` e `go build ./...` passaram. A integração PostgreSQL continuou ignorada por falta de banco de testes configurado. Nenhum arquivo Go foi alterado neste incremento; gofmt não foi necessário. Os checks de Go não validam a execução do bootstrap SQL.
