-- Etapa administrativa autorizada após 0000_create_databases.sql.
-- Executar em postgres, numa transação, após conferir roles e proprietários.
-- Os parâmetros psql abaixo recebem verificadores SCRAM-SHA-256 gerados
-- localmente; nunca substituí-los por credenciais em arquivos versionados.
-- Um executor pode substituir os dois parâmetros por literais SQL escapados.
ALTER ROLE financeiro_go_app LOGIN PASSWORD :'app_scram';
ALTER ROLE financeiro_go_test_app LOGIN PASSWORD :'test_scram';
