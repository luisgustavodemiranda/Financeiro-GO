# Financeiro-GO

API de controle financeiro familiar, desenvolvida como projeto de estudo e portfólio em Go.

O projeto explora monólito modular, separação entre handlers HTTP, serviços e repositórios, testes de comportamento e cálculos financeiros em centavos inteiros. A evolução está registrada em [doc/](doc/) e no [plano de desenvolvimento](FINANCEIROGO-PLAN.md).

## Funcionalidades disponíveis

- Cadastro e listagem de contas com saldo inicial separado das receitas.
- Registro de receitas e despesas por conta.
- Consulta de lançamentos e saldo, com proteção contra overflow de `int64`.
- API executável em memória, sem necessidade de banco para experimentar.
- Persistência PostgreSQL com migrations e integração validada em banco real.
- Cadastro e listagem de cartões por nome em memória; persistência de cartões preparada, aguardando migration 0002 e validação real.

## Arquitetura

```text
Requisição HTTP → Handler → Serviço → Interface de repositório
                                        ├─ Memória
                                        └─ PostgreSQL (pgx)
```

O serviço concentra as regras financeiras; cada repositório implementa o armazenamento. Consulte o [processo de desenvolvimento](doc/processo-desenvolvimento.md) para acompanhar os incrementos.

## Estado atual

API HTTP local com configuração validada, encerramento seguro, contas, receitas, despesas e saldo. O padrão continua em memória: **todos os dados desaparecem ao reiniciar**. O modo PostgreSQL foi validado com bancos exclusivos, migration aplicada e testes de integração reais. Frontend permanece futuro.

Arquitetura de referência: Fiscal-GO. Monólito modular em Go, com pgx v5 para PostgreSQL. React + TypeScript estão planejados. Worker e Redis entram quando houver importações assíncronas.

## Executar no Windows

Requer Go 1.26 ou superior. Abra o terminal na pasta do projeto:

```powershell
go run ./cmd/financeiro
```

Acesse http://127.0.0.1:8081/health e http://127.0.0.1:8081/api/v1/status. Encerre com Ctrl+C.

Configuração opcional:

```powershell
$env:HTTP_PORT = "8082"
go run ./cmd/financeiro
```

O arquivo `.env.example` documenta variáveis; arquivos `.env` ainda não são carregados automaticamente. O servidor escuta somente no computador local por padrão.

Para PostgreSQL, consulte [a preparação da fase 3](doc/03-postgresql.md). `PERSISTENCE=postgres` exige `FINANCEIRO_DATABASE_URL` ou variáveis `DB_*`, banco exclusivo e migration previamente aplicada com autorização. A API nunca aplica migrations ao iniciar e nunca troca silenciosamente PostgreSQL por memória quando há erro.

O ambiente local provisionado tem instruções de uso e evidências na [conclusão da fase 3](doc/05-integracao-postgres-validada.md). O arquivo local de credenciais protegidas não acompanha o repositório; quem clonar o projeto deve configurar seu próprio ambiente.

**Fase 4 em preparação:** esta versão exige também a migration 0002 para iniciar com PostgreSQL. Ela ainda não foi aplicada nos bancos locais. O modo em memória continua funcionando; veja [cadastro de cartões](doc/06-cadastro-cartoes.md). A autorização dada anteriormente para 0001 não abrange 0002.

## Validar

```powershell
go test ./...
go vet ./...
go build ./...
```

Sem `FINANCEIRO_TEST_DATABASE_URL`, a integração PostgreSQL aparece como `SKIP`; os testes de serviço e HTTP em memória continuam executando normalmente.

## Como estudar

1. `internal/contas/contas.go`: modelos, interface do repositório e regras do serviço.
2. `internal/contas/contas_test.go`: exemplos executáveis de saldo e validação.
3. `internal/contas/memory.go`: armazenamento, mutex e cópias dos dados.
4. `internal/server/accounts.go` e `accounts_test.go`: contrato HTTP e testes com `httptest`.
5. `internal/server/server.go` e `cmd/financeiro/main.go`: composição e inicialização.

Veja [a fase 2](doc/02-contas-lancamentos.md) para o contrato JSON, exemplos completos de PowerShell, decisões e limitações. O [plano](FINANCEIROGO-PLAN.md) registra os próximos incrementos.

Na fase 3, continue por `internal/contas/postgres.go`, `internal/database/postgres.go`, `migrations/0001_accounts_entries.sql` e `internal/contas/postgres_integration_test.go`. O documento da fase 2 é histórico; a situação atual da persistência está na [fase 3](doc/03-postgresql.md).

Nenhuma informação financeira pessoal foi incluída. Use dados fictícios nos testes. Banco próprio e autenticação serão necessários antes de disponibilizar dados reais em rede.
