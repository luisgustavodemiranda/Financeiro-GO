# Fase 2 — Contas, lançamentos e saldo em memória

Implementado com a biblioteca padrão de Go: cadastrar/listar contas, registrar receitas/despesas e consultar lançamentos e saldo por conta. Todos os dados desaparecem ao encerrar ou reiniciar o processo. Cada `NewHandler` cria seu próprio repositório vazio.

## Decisões e regras

`Account` representa a conta e `Entry` um lançamento realizado. Valores monetários são `int64` em centavos tanto no Go quanto no JSON: `1250` significa 12,50. Números fracionários, strings monetárias e valores fora de `int64` são rejeitados. Não existe conversão por `float64`.

O saldo é `saldo inicial + receitas - despesas`, na ordem de registro. Cada passo deve caber em `int64`. A verificação ocorre antes de somar ou subtrair, evitando que o próprio cálculo de validação transborde. Não há total acumulado de receitas/despesas separado.

Saldo inicial pode ser positivo, zero ou negativo e não gera lançamento. Sua data representa a abertura do dia; lançamentos nessa data são permitidos, anteriores não. O saldo consultado inclui todos os registros aceitos, inclusive datas futuras, e não é uma projeção nem uma consulta histórica. Datas devem usar `AAAA-MM-DD`. Valores de lançamentos devem ser positivos; `kind` aceita somente `income` (receita) ou `expense` (despesa). Saldo negativo é permitido, sem regra de limite bancário.

Nome e descrição são obrigatórios após remover espaços das extremidades. O saldo inicial precisa estar presente no JSON, mesmo quando zero: o handler usa `*int64` para distinguir zero de campo ausente ou `null`. Descrição, tipo, valor e data são obrigatórios no lançamento. A conta precisa existir.

## Contrato HTTP

| Método e rota | Resultado |
| --- | --- |
| `POST /api/v1/accounts` | Cria conta; 201 com objeto da conta |
| `GET /api/v1/accounts` | 200 com array de contas |
| `POST /api/v1/accounts/{id}/entries` | Cria lançamento; 201 com objeto do lançamento |
| `GET /api/v1/accounts/{id}/entries` | 200 com array de lançamentos |
| `GET /api/v1/accounts/{id}/balance` | 200 com objeto da conta e saldo atual |

Conta retornada: `id` e `name` (strings), `initial_balance_cents` e `balance_cents` (inteiros int64), `initial_balance_date` (string). Lançamento retornado: `id`, `account_id`, `kind`, `description`, `date` (strings) e `amount_cents` (inteiro int64). IDs são gerados pelo repositório/serviço, são locais ao processo e podem repetir após reiniciar. Arrays vazios retornam `[]`. Listas seguem a ordem de cadastro, sem paginação.

Erros das operações financeiras usam `{"error":"mensagem"}`: 400 para entrada inválida, 404 para conta inexistente, 409 para overflow e 500 para falhas internas. O roteador padrão responde 404/405 para rotas inexistentes/métodos não permitidos. Corpos aceitam um único objeto JSON, rejeitam campos desconhecidos e têm limite de 1 MiB. `GET /api/v1/status` informa explicitamente a persistência em memória.

## Executar pelo terminal do VS Code

Requer Go 1.26 ou superior. No primeiro terminal PowerShell:

```powershell
Set-Location D:\GO\Financeiro-GO
go run ./cmd/financeiro
```

Em outro terminal, com o servidor rodando, use apenas os dados fictícios abaixo:

```powershell
$base = 'http://127.0.0.1:8081/api/v1'
$accountBody = @{
    name = 'Conta de estudo'
    initial_balance_cents = [long]10000
    initial_balance_date = '2026-01-01'
} | ConvertTo-Json
$account = Invoke-RestMethod -Method Post -Uri "$base/accounts" -ContentType 'application/json' -Body $accountBody

$incomeBody = @{
    kind = 'income'
    description = 'Receita ficticia'
    amount_cents = [long]2500
    date = '2026-01-02'
} | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$base/accounts/$($account.id)/entries" -ContentType 'application/json' -Body $incomeBody

$expenseBody = @{
    kind = 'expense'
    description = 'Despesa ficticia'
    amount_cents = [long]1200
    date = '2026-01-03'
} | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$base/accounts/$($account.id)/entries" -ContentType 'application/json' -Body $expenseBody

Invoke-RestMethod "$base/accounts"
Invoke-RestMethod "$base/accounts/$($account.id)/entries"
Invoke-RestMethod "$base/accounts/$($account.id)/balance"
```

Saldo esperado: `11300` centavos; dois lançamentos. Encerre com Ctrl+C no primeiro terminal. Ao executar novamente, a lista estará vazia. Se configurar `HTTP_PORT`, ajuste `$base`.

## Caminho de uma requisição

1. `server.NewHandler` monta o serviço e injeta `MemoryRepository` como implementação da interface `Repository`.
2. O handler em `internal/server/accounts.go` decodifica JSON tipado e extrai o ID da rota.
3. `Service.Register`, em `internal/contas/contas.go`, valida os dados e solicita `Repository.Update`.
4. O repositório adquire um mutex, encontra a conta e entrega uma cópia do estado à função do serviço. O serviço verifica a data e o limite numérico, acrescenta o lançamento e atualiza o saldo nessa cópia.
5. Se a função retorna erro, a cópia é descartada. Se tem sucesso, o repositório guarda o novo estado. O handler converte o resultado em status HTTP e JSON.

A interface expressa o contrato, sem depender da memória. A função passada a `Update` mantém a regra financeira no serviço e permite ao repositório controlar a exclusão mútua durante toda a operação. O futuro adaptador PostgreSQL deverá garantir essa mesma atomicidade com transação e controle de concorrência. Não basta executar leitura e gravação independentes.

## Estudo e validação

Leia primeiro `contas.go`, depois `contas_test.go`, `memory.go` e por fim `accounts.go`/`accounts_test.go`. Os testes de serviço chamam as regras diretamente; os de HTTP usam `httptest`, sem abrir porta e sem depender de servidor em execução.

```powershell
Set-Location D:\GO\Financeiro-GO
go test ./internal/contas -v
go test ./internal/server -v
go test ./...
go vet ./...
go build ./...
```

Executados com sucesso nesta entrega: `gofmt` nos arquivos Go alterados, `go test ./...`, `go vet ./...` e `go build ./...`, usando Go 1.26.5 Windows/amd64. Testes cobrem saldo inicial separado, receita/despesa, saldo negativo, isolamento entre contas, campos inválidos, conta inexistente, limites superior/inferior de int64, ausência de escrita após erro, cópias independentes e 100 registros concorrentes. Os testes HTTP verificam contratos, precisão de int64, JSON inválido e repositório vazio em uma nova instância.

## Limitações e próximo incremento

Sem PostgreSQL, frontend, cartões, transferências, importação, autenticação, exclusão/edição, paginação ou idempotência. Repetir um POST válido registra outra operação. Armazenamento cresce em memória, faz busca linear e usa um mutex global para escritas; adequado a esta etapa de estudo local. Clientes futuros em JavaScript precisarão tratar inteiros acima de `Number.MAX_SAFE_INTEGER` sem perder precisão.

Próximo incremento: planejar a persistência PostgreSQL exclusiva, implementar um adaptador com transações e testes de integração que preservem as regras atuais. Executar migrations somente após autorização específica.

O Fiscal-GO foi consultado apenas como referência de arquitetura e serviço/repositório. Seus arquivos e banco não foram alterados. Nenhum banco foi acessado, nenhuma migration foi executada e nenhum commit ou push foi feito. A pasta Financeiro-GO não continha `.git`; `git status` retornou que não é um repositório, portanto não foi possível comparar com histórico Git. A fundação existente foi mantida e estendida.
