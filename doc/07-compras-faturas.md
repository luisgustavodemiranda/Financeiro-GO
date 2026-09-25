# Fase 4 — Compras e faturas com datas explícitas

Branch: `feature/fase-4-compras-faturas`, criada de develop após o merge do PR #4. Implementação e testes locais concluídos; aplicação da migration 0003 e integração real pendentes. Este documento distingue a implementação preparada da validação em banco.

## Regras acordadas e comportamento

- Cada fatura pertence a um cartão existente e recebe `start_date`, `closing_date` e `due_date`. Datas civis usam AAAA-MM-DD, anos 0001 a 9999, sem horário. Início <= fechamento < vencimento. Um período de um dia é válido.
- Início e fechamento são inclusivos. Períodos do mesmo cartão não podem se sobrepor, inclusive entre faturas fechadas. Cartões diferentes podem usar as mesmas datas. Lacunas entre períodos são permitidas; não há geração automática de calendário.
- O cliente escolhe a fatura da compra explicitamente. A compra exige descrição de 1 a 200 caracteres Unicode após trim, sem NUL, data válida dentro do período e `amount_cents` positivo em int64. JSON usa número inteiro, nunca float ou string monetária.
- Compras iguais são permitidas, com IDs diferentes. Ainda não há chave de idempotência para POST de compra: repetir o pedido cria outra compra. Em resposta incerta, conferir o histórico antes de repetir.
- A compra é a despesa e obrigação do cartão. Sua data é a referência da despesa; não há lançamento em conta bancária nem alteração de saldo. O total da fatura é a soma das compras, protegida contra overflow antes da gravação. Não somar compras e total da fatura como despesas distintas.
- Faturas começam `open`, com total zero e `purchases: []`. O fechamento explícito muda para `closed`, mantendo compras e total. Fechar novamente é idempotente. Pode fechar vazia ou antes da data de fechamento: a ação é manual, sem dependência do relógio.
- Depois de fechar, novas compras são rejeitadas. Não há edição, exclusão, reabertura, estorno, parcelamento, pagamento, juros ou limite de crédito neste incremento.
- Fatura fechada ainda é obrigação em aberto: `closed` não significa paga. Vencimento é uma data planejada, não prova pagamento nem altera saldo automaticamente.

Exemplo fictício: comprar por 10000 centavos aumenta a despesa do cartão e a obrigação em 10000, mantendo o saldo bancário. O pagamento futuro deverá diminuir o saldo e quitar a obrigação sem criar outra despesa. O serviço atual de despesas de contas não atende a esse pagamento sem adaptação; ele ainda não é chamado pelo módulo de cartões.

## Contrato HTTP

Base: `/api/v1/cards/{card}/invoices`. IDs são strings geradas pelo repositório; consumidores não devem interpretar seu formato.

| Método e caminho | Entrada | Resposta |
| --- | --- | --- |
| POST na base | `start_date`, `closing_date`, `due_date` | 201, fatura criada |
| GET na base | Sem corpo | 200, array de faturas, em ordem de cadastro |
| GET `/{invoice}` | Sem corpo | 200, fatura com total e compras |
| POST `/{invoice}/purchases` | `description`, `amount_cents`, `date` | 201, fatura atualizada com a nova compra |
| POST `/{invoice}/close` | `{}` | 200, fatura fechada |

As consultas incluem `id`, `card_id`, datas, `status`, `total_cents` e `purchases`. Cada compra inclui `id`, `description`, `amount_cents` e `date`; seu vínculo é a fatura que a contém. Listas vazias são `[]`. Não há paginação ainda.

400: JSON inválido, campos desconhecidos, datas ou valores inválidos. 404: cartão/fatura inexistente ou fatura de outro cartão. 409: período sobreposto, compra em fatura fechada ou overflow. 500: erro interno sem expor detalhes de banco. A validação da entrada ocorre antes da consulta ao repositório.

## Executar e experimentar no terminal do VS Code

No primeiro terminal PowerShell:

```powershell
Set-Location D:\GO\Financeiro-GO
$env:PERSISTENCE = 'memory'
go run ./cmd/financeiro
```

Em outro terminal, com valores exclusivamente fictícios:

```powershell
$base = 'http://127.0.0.1:8081/api/v1'
$card = Invoke-RestMethod -Method Post -Uri "$base/cards" -ContentType 'application/json' -Body '{"name":"Cartao ficticio"}'
$invoices = "$base/cards/$($card.id)/invoices"
$invoice = Invoke-RestMethod -Method Post -Uri $invoices -ContentType 'application/json' -Body '{"start_date":"2026-01-01","closing_date":"2026-01-31","due_date":"2026-02-10"}'
$invoiceUrl = "$invoices/$($invoice.id)"
Invoke-RestMethod -Method Post -Uri "$invoiceUrl/purchases" -ContentType 'application/json' -Body '{"description":"Compra ficticia","amount_cents":1234,"date":"2026-01-15"}'
Invoke-RestMethod -Uri $invoiceUrl
Invoke-RestMethod -Uri $invoices
Invoke-RestMethod -Method Post -Uri "$invoiceUrl/close" -ContentType 'application/json' -Body '{}'
```

Reiniciar no modo memória perde todos os dados. Para testar sem banco:

```powershell
Remove-Item Env:FINANCEIRO_TEST_DATABASE_URL -ErrorAction SilentlyContinue
go test ./...
go vet ./...
go build ./...
```

## Caminho da requisição e ordem de estudo

1. `internal/cartoes/invoices.go`: modelos, regras de data, valor, total e fechamento. Leia com `invoices_test.go`, que descreve comportamentos com exemplos fictícios.
2. `internal/server/invoices.go`: handler decodifica JSON, passa contexto e IDs ao serviço e traduz resultado em HTTP. O cliente não pode definir status ou total.
3. `internal/cartoes/cartoes.go`: interface Repository. O serviço depende do contrato, sem conhecer SQL ou mutex.
4. `internal/cartoes/invoices_memory.go`: mutex serializa operações, e cópias impedem alteração do estado pelos retornos. O callback de UpdateInvoice permite validar usando o estado já protegido; erro não grava mudanças.
5. `internal/cartoes/invoices_postgres.go`: transação bloqueia o cartão antes de consultar sobreposição e criar a fatura. Compra e fechamento bloqueiam a mesma fatura com FOR UPDATE. INSERT da compra e UPDATE do total são confirmados juntos. Leituras usam snapshot REPEATABLE READ para manter total e compras consistentes.
6. `internal/server/invoices_test.go` e `invoices_postgres_integration_test.go`: contrato HTTP e preparação de integração real, incluindo leitura por outro pool.

O módulo continua o mesmo, sem abstração adicional de unidade de trabalho. A implementação PostgreSQL protege concorrência através do repositório; SQL manual que contorne esse fluxo não recebe todas as regras do serviço. O schema contém constraints básicas, mas a não sobreposição e o total consistente dependem do uso da aplicação. Listagem carrega compras de cada fatura e deverá ganhar paginação antes de volumes maiores.

## Migration e ordem de implantação

`migrations/0003_invoices_purchases.sql` cria `financeiro.invoices` e `financeiro.purchases`, índices por cartão/fatura, referências e constraints de datas, estado e valores. Não altera linhas ou tabelas de contas, lançamentos ou cartões. Não exige backfill, extensão PostgreSQL ou credencial administrativa. 0001 e 0002 permanecem inalteradas, com checksums protegidos por teste.

Inspeção somente leitura em 25/09/2026 confirmou nos dois bancos exclusivos: histórico 0001/0002 com hashes esperados, usuários próprios sem superusuário, permissão CREATE no schema, estrutura de cards esperada e ausência de invoices/purchases. Nenhuma migration ou fixture foi executada nesta etapa. Revalidar o estado imediatamente antes de aplicar.

A execução proposta, após autorização específica, é aplicar somente 0003 em `financeiro_go` e `financeiro_go_test` pelo executor existente e rodar integração apenas no segundo, limpando somente as fixtures criadas pelos próprios testes. O executor valida o histórico e usa transação. Esta versão da API recusa iniciar em PostgreSQL enquanto 0003 estiver pendente. O binário anterior também rejeita histórico com versão desconhecida; planejar parada da API para migration e atualização do binário. Não há downgrade automático.

## Validação e pendências

gofmt, `go test ./...`, `go vet ./...` e `go build ./...` passaram. Testes locais cobrem períodos inclusivos, sobreposição, datas inválidas, cartão/fatura inexistente, vínculo entre cartões, descrição, valores, precisão int64 no JSON, overflow sem gravação, cópias, cancelamento, fechamento repetido/vazio, concorrência e preservação do saldo e lançamentos bancários.

A suíte de comportamento é compartilhada entre memória e o teste opt-in PostgreSQL. A integração nova está preparada, mas não executada: sem FINANCEIRO_TEST_DATABASE_URL, testes de banco ficam SKIP. Isso não comprova funcionamento do SQL. Aplicação de 0003 e integração real permanecem pendentes; o PR deve ficar em rascunho até a validação. Próximo incremento recomendado após essa conclusão: definir e implementar pagamento integral com operação atômica entre conta e fatura, sem nova despesa.
