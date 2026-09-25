# Fase 3 — Integração PostgreSQL validada

Em 25/09/2026, o usuário confirmou a criação dos bancos pelo bootstrap e autorizou configurar senhas independentes, habilitar os logins, aplicar a migration 0001 e executar os testes. Trabalho em `feature/fase-3-validacao-postgres`, a partir de develop atualizado, preservando a anotação local da inspeção anterior.

## Operações realizadas

Revalidação somente leitura confirmou ambos os bancos e roles. Antes da alteração, foram conferidos os respectivos proprietários, NOLOGIN e ausência de superusuário, criação de bancos/roles, replicação e bypass de RLS. Não houve recriação dos bancos nem redefinição de usuários externos ao projeto.

`migrations/bootstrap/0001_enable_logins.sql` registra a etapa administrativa de ativação. Os parâmetros foram preenchidos em memória com verificadores SCRAM-SHA-256; o arquivo versionado não contém senhas nem verificadores reais. A alteração dos dois logins foi confirmada em uma transação no banco administrativo postgres.

Credenciais aleatórias independentes foram armazenadas localmente em `.env.postgres.clixml`, com PSCredential/DPAPI do Windows. O arquivo é ignorado por `.gitignore`, não foi publicado e sua descriptografia depende do usuário Windows e computador que o criaram. Não apague esse arquivo sem garantir outra forma de recuperar ou substituir as credenciais. A configuração do servidor e credenciais administrativas não constam deste documento.

Cada usuário exclusivo aplicou a migration de aplicação com `go run ./cmd/migrate -apply` em seu próprio banco. `financeiro_go_app` e `financeiro_go_test_app` não são superusuários. Como proprietários dos respectivos bancos, ainda têm permissões de DDL nesses bancos; separação de usuário de migration e usuário de runtime permanece uma evolução de implantação.

## Evidências

| Verificação | Resultado |
| --- | --- |
| Banco financeiro_go | Migration 0001, três tabelas, 13 constraints e quatro índices |
| Banco financeiro_go_test | Mesma versão e checksum; mesmo conjunto de objetos |
| Conteúdo da aplicação | Zero contas e zero lançamentos; nenhum dado de teste inserido |
| TestPostgresIntegration | Todos os cenários aprovados, sem SKIP |
| Reinício da API | Conta fictícia e saldo 1234 centavos preservados após encerrar e iniciar um novo processo |
| Limpeza | Fixture de reinício removida pelo ID exato; integração limpa seus próprios IDs |
| go test ./... -count=1 | Aprovado com conexão de integração configurada |
| go vet ./... e go build ./... | Aprovados |
| gofmt -l cmd internal migrations | Sem pendências; nenhum arquivo Go alterado nesta etapa |

A integração cobre persistência por outro pool, endpoints HTTP, isolamento entre contas, saldo inicial separado, limites superior/inferior de int64, rollback após erro da regra e após erro SQL, 24 registros concorrentes entre pools, consistência de snapshot e cancelamento durante bloqueio. O teste de reinício foi executado em processo separado, com uma fixture adicional somente em financeiro_go_test. A API temporária foi encerrada ao final.

Não houve alteração ou consulta a tabelas do Fiscal-GO. O CI remoto continua sem conexão ao banco local: seus testes PostgreSQL podem aparecer como SKIP; a aprovação de integração registrada aqui corresponde à execução local autorizada.

## Executar no terminal do VS Code

Abra PowerShell na raiz do projeto, usando o mesmo usuário Windows que possui as credenciais protegidas:

```powershell
Set-Location D:\GO\Financeiro-GO
# Somente se o Windows bloquear o script local; vale apenas neste terminal:
Set-ExecutionPolicy -Scope Process -ExecutionPolicy RemoteSigned
$databaseHost = Read-Host 'Host do PostgreSQL'
& ./scripts/Use-Postgres.ps1 -DatabaseHost $databaseHost -Target Application -SSLMode disable
go run ./cmd/financeiro
```

`disable` reproduz a opção de SSL informada para o servidor local. O padrão do script é `require`; configure TLS conforme o servidor. O script apenas importa a credencial protegida e prepara variáveis do terminal, sem criar banco ou aplicar migrations. Ele remove uma URL de aplicação anterior para que DB_* seja usada e, no modo Application, remove a URL de testes para evitar execução acidental da integração. Encerre o servidor com Ctrl+C.

Em outro terminal, para executar a integração já provisionada:

```powershell
Set-Location D:\GO\Financeiro-GO
Set-ExecutionPolicy -Scope Process -ExecutionPolicy RemoteSigned
$databaseHost = Read-Host 'Host do PostgreSQL'
& ./scripts/Use-Postgres.ps1 -DatabaseHost $databaseHost -Target Test -SSLMode disable
go test ./internal/contas -run '^TestPostgresIntegration$' -count=1 -v
```

Use `-Target Application` novamente antes de rodar a API principal no mesmo terminal. A senha fica no ambiente do processo durante o uso; não imprima DB_PASSWORD nem as URLs. Não copie o CLIXML para o repositório ou para outro computador esperando que funcione. Em outro ambiente, configure DB_* ou FINANCEIRO_DATABASE_URL com credenciais próprias, conforme doc/03-postgresql.md.

## Estudo e próximo passo

Leia `internal/contas/postgres.go`: o serviço calcula o saldo, enquanto o repositório bloqueia a conta e confirma lançamento/saldo numa transação. Depois leia `postgres_integration_test.go` para observar como esses comportamentos são verificados no banco real. `scripts/Use-Postgres.ps1` é apenas configuração local; não muda a arquitetura Go.

A fase 3 está concluída para o escopo definido. Permanecem fora dela autenticação, frontend, paginação, idempotência, backup operacional e cartões. A migration 0001 aplicada deve permanecer imutável. Antes de futuras migrations, evoluir o executor, que ainda reconhece somente 0001. O próximo incremento de produto deve delimitar cartões/faturas e impedir a duplicação de despesas no pagamento.
