-- Provisionamento inicial: executar somente após autorização específica.
-- Conectar ao banco administrativo postgres, nunca ao Fiscal-GO.
-- CREATE DATABASE exige execução fora de uma transação explícita.
-- Verificar antes se os dois bancos e os dois roles já existem.
-- Sem IF NOT EXISTS: um conflito exige inspeção, não adoção silenciosa.

CREATE ROLE financeiro_go_app
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;

CREATE ROLE financeiro_go_test_app
    NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;

CREATE DATABASE financeiro_go OWNER financeiro_go_app;
CREATE DATABASE financeiro_go_test OWNER financeiro_go_test_app;

-- Restringir conexão aos respectivos proprietários e administradores.
REVOKE CONNECT, TEMPORARY ON DATABASE financeiro_go FROM PUBLIC;
REVOKE CONNECT, TEMPORARY ON DATABASE financeiro_go_test FROM PUBLIC;

-- Roles começam sem login. Antes de ativá-los, definir senhas independentes
-- por um canal local seguro; não escrever senhas neste arquivo ou no Git.
-- Só então, como parte do provisionamento autorizado:
-- ALTER ROLE financeiro_go_app LOGIN;
-- ALTER ROLE financeiro_go_test_app LOGIN;
