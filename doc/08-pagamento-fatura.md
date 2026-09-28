# Fase 4 — Pagamento integral de fatura

Branch: `feature/fase-4-pagamento-fatura`. Implementação e testes em memória concluídos. Adaptador PostgreSQL, migration 0004 e integração preparados, ainda sem execução no banco. A etapa foi retomada em 28/09/2026, preservando o trabalho local após interrupção por limite da revisão automática de permissões.

## Regras e contrato

`POST /api/v1/cards/{card}/invoices/{invoice}/payment` recebe apenas `account_id` (string) e `date` (AAAA-MM-DD). O serviço usa o total inteiro da fatura; o cliente não informa valor. A operação registra um pagamento no sistema, sem enviar dinheiro a um banco.

- Cartão, fatura e conta devem existir; a fatura deve pertencer ao cartão informado.
- Primeiro pagamento exige fatura `closed` e total positivo. Fatura vazia permanece fechada, sem débito de zero.
- Data de pagamento não pode anteceder a data de fechamento nem a referência do saldo inicial da conta. Não é obrigada a coincidir com o vencimento. Não há cálculo automático de juros por atraso; o registro não consulta o relógio e aceita datas futuras explícitas, como os lançamentos atuais.
- O saldo pode ficar negativo, conforme a regra existente de contas. Subtração verifica o limite inferior de int64 antes de gravar.
- O pagamento gera um lançamento `invoice_payment`, com `invoice_id`, e muda a fatura para `paid`. Compras e total permanecem inalterados. Não gera lançamento `expense`: a despesa foi reconhecida na compra.
- Mesma fatura, conta e data retornam o pagamento já salvo, sem novo débito, mesmo sob concorrência. Tentar outra conta ou data depois de pagar retorna conflito. Essa idempotência usa a própria fatura, sem chave adicional.
- Fechar novamente uma fatura paga mantém `paid`. Não é possível adicionar compras, reabrir ou editar o pagamento.

200 devolve a fatura com `payment: {account_id, date, amount_cents, entry_id}`. Faturas não pagas têm `payment: null`. Dinheiro continua int64 em centavos, inclusive no JSON. 400 indica entrada inválida; 404, recurso inexistente; 409, estado incompatível, repetição divergente ou overflow; 500 omite detalhes internos. O endpoint de lançamentos comuns continua aceitando somente income/expense, impedindo criar um pagamento sem quitar a fatura.

Exemplo fictício: conta com 1000 centavos e fatura de 1234. Pagar deixa o saldo em -234, acrescenta um único lançamento invoice_payment de 1234 e marca a fatura como paga. Relatórios futuros devem separar saída de caixa de despesa; somar compras e invoice_payment como despesas duplicaria o valor.

## Fluxo e arquivos para estudar

1. `internal/cartoes/payments.go`: Payment, interface PaymentRepository e serviço. Comece aqui e em `payments_test.go`.
2. `internal/server/payments.go`: decodifica JSON e envia contexto e IDs ao serviço. Não calcula valores.
3. `payments_memory.go`: coordena os repositórios existentes. Bloqueia cartões, prepara uma cópia da fatura e chama Update da conta. Se validação ou cancelamento falhar, a fatura continua intacta. Depois do commit da conta, publica a cópia da fatura sem nova operação que possa retornar erro, mantendo leitores de fatura bloqueados até terminar. A ordem de locks é cartões → conta; operações comuns de conta não adquirem cartões.
4. `payments_postgres.go`: usa uma transação, bloqueando fatura e depois conta. Insere o lançamento, atualiza saldo e marca pagamento; qualquer falha reverte o conjunto. A sequência do lançamento é calculada sob o mesmo bloqueio da conta usado pelos lançamentos comuns.
5. `cmd/financeiro/main.go`: injeta explicitamente o repositório de pagamentos correspondente aos repositórios de conta/cartão. Não há chamadas sequenciais a dois serviços independentes para efetivar o pagamento.

O contrato específico PaymentRepository permite coordenar os dois módulos sem introduzir uma unidade de trabalho genérica. PostgreSQL utiliza o mesmo pool; memória recebe as mesmas instâncias de contas e cartões usadas pelo HTTP.

## Experimentar em PowerShell

No terminal do VS Code:

```powershell
Set-Location D:\GO\Financeiro-GO
$env:PERSISTENCE = 'memory'
go run ./cmd/financeiro
```

Em outro terminal, usando somente dados fictícios:

```powershell
$base = 'http://127.0.0.1:8081/api/v1'
$account = Invoke-RestMethod -Method Post -Uri "$base/accounts" -ContentType 'application/json' -Body '{"name":"Conta ficticia","initial_balance_cents":1000,"initial_balance_date":"2026-01-01"}'
$card = Invoke-RestMethod -Method Post -Uri "$base/cards" -ContentType 'application/json' -Body '{"name":"Cartao ficticio"}'
$invoice = Invoke-RestMethod -Method Post -Uri "$base/cards/$($card.id)/invoices" -ContentType 'application/json' -Body '{"start_date":"2026-01-01","closing_date":"2026-01-31","due_date":"2026-02-10"}'
$invoiceUrl = "$base/cards/$($card.id)/invoices/$($invoice.id)"
Invoke-RestMethod -Method Post -Uri "$invoiceUrl/purchases" -ContentType 'application/json' -Body '{"description":"Compra ficticia","amount_cents":1234,"date":"2026-01-15"}'
Invoke-RestMethod -Method Post -Uri "$invoiceUrl/close" -ContentType 'application/json' -Body '{}'
$payment = @{ account_id = $account.id; date = '2026-02-10' } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$invoiceUrl/payment" -ContentType 'application/json' -Body $payment
# Repetir a mesma chamada não debita novamente.
Invoke-RestMethod -Method Post -Uri "$invoiceUrl/payment" -ContentType 'application/json' -Body $payment
Invoke-RestMethod "$base/accounts/$($account.id)/balance"
Invoke-RestMethod "$base/accounts/$($account.id)/entries"
```

Reiniciar em memória perde todos os dados. Testes locais:

```powershell
Remove-Item Env:FINANCEIRO_TEST_DATABASE_URL -ErrorAction SilentlyContinue
go test ./...
go vet ./...
go build ./...
```

## Banco preparado, não aplicado

`migrations/0004_invoice_payments.sql` amplia as constraints de tipo de lançamento e status, adiciona a referência única de lançamento para fatura e os dados de pagamento na fatura. Colunas novas começam nulas; contas, lançamentos e faturas anteriores são preservados, sem backfill financeiro. 0001/0002/0003 permanecem imutáveis, com checksums protegidos por teste.

Inspeção somente leitura em 28/09/2026 confirmou, nos dois bancos exclusivos, histórico até 0003 e hashes esperados, usuários próprios sem superusuário, colunas atuais e constraints entries_kind_check/invoices_status_check compatíveis com o SQL preparado. Não houve DDL nem execução de fixtures de pagamento.

A próxima execução requer autorização específica para aplicar 0004 em financeiro_go e financeiro_go_test e executar integração somente no segundo, limpando suas próprias fixtures. Revalidar o estado antes da aplicação. ALTER TABLE pode bloquear acesso às tabelas durante a transação; coordenar parada da API para aplicar e atualizar o binário. Esta versão não inicia em PostgreSQL com 0004 pendente; binários anteriores rejeitam a nova versão. Não há downgrade automático.

As constraints complementam os bloqueios e validações da aplicação. Escrita SQL manual que contorne o serviço não recebe todas as regras de consistência entre módulos. Os testes removem apenas suas fixtures, desfazendo primeiro a referência entre pagamento e lançamento; não executam TRUNCATE nem migrations.

## Validação e continuidade

gofmt, go test ./..., go vet ./... e go build ./... aprovados. Cobertura local inclui débito sem nova despesa, saldo negativo, limites de int64, repetição, conta/data divergentes, fatura aberta/vazia, datas, ausência de recursos, cancelamento antes do commit, cópias e concorrência entre pagamento repetido e receitas comuns. Endpoints verificam o contrato e rejeitam valor informado pelo cliente.

O teste opt-in PostgreSQL reutiliza a suíte e acrescenta persistência por outro pool e falha deliberada na última escrita, para comprovar rollback do lançamento e do saldo. Foi preparado, mas ainda não executado; SKIP não comprova integração real. O PR deve permanecer em rascunho enquanto 0004 e essa validação estiverem pendentes. Depois, o próximo incremento recomendado é parcelamento com divisão exata dos centavos.
