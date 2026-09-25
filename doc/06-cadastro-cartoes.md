# Fase 4 — Primeiro incremento: cadastro de cartões

Branch: `feature/fase-4-cadastro-cartoes`, criada a partir de develop após a integração da fase 3. Escopo: cadastrar/listar nomes de cartões e preparar a evolução incremental do banco. Sem compras, faturas, limites de crédito, datas de fechamento/vencimento, parcelas ou pagamentos nesta entrega.

## Contrato e comportamento

`POST /api/v1/cards` recebe `{"name":"Cartao ficticio"}` e responde 201 com `{"id":"1","name":"Cartao ficticio"}`. `GET /api/v1/cards` responde 200 com array na ordem de cadastro, ou `[]` quando vazio. IDs são strings, gerados pelo repositório; não são números de cartão. Nomes iguais são permitidos para cartões distintos.

O nome é obrigatório após remover espaços nas extremidades, possui no máximo 100 caracteres Unicode e não pode conter NUL. Nome vazio, tipo JSON incorreto, corpo inválido ou campos desconhecidos retornam 400. A API não aceita número, CVV ou validade. Erros de repositório retornam 500 sem detalhes internos. Não há GET individual, edição ou exclusão neste incremento.

Cadastrar um cartão não cria conta, receita, despesa nem altera saldo. O repositório em memória protege operações simultâneas e devolve cópias. Reiniciar o processo perde esses cadastros. O adaptador PostgreSQL usa timeout e consultas parametrizadas; sua integração real foi aprovada após a autorização para aplicar 0002.

## Caminho da requisição e estudo

1. `internal/server/cards.go` decodifica JSON e passa `r.Context()` ao serviço.
2. `internal/cartoes/cartoes.go` normaliza e valida o nome, chamando a interface Repository.
3. `memory.go` guarda o cadastro com mutex; `postgres.go` usa INSERT/SELECT na tabela cards.
4. `cmd/financeiro/main.go` injeta os repositórios de contas e cartões no mesmo modo de persistência. O servidor exige ambos explicitamente, evitando misturar contas persistentes com cartões temporários.

Leia primeiro o serviço e seus testes, depois o handler e o adaptador. Nenhuma regra de dinheiro nova foi introduzida.

## Migration aplicada após autorização

`migrations/0002_cards.sql` cria somente `financeiro.cards`, com ID bigint identity, nome obrigatório e CHECK do limite de caracteres. Não altera contas ou lançamentos. A migration 0001 permanece byte a byte sem alterações no Git e mantém seu checksum normalizado.

Inspeção somente leitura em 25/09/2026 confirmou nos dois bancos: versão 0001, checksum esperado, usuário sem superusuário e ausência de cards. Essa evidência precisa ser revalidada antes de aplicar 0002.

O executor agora incorpora os arquivos SQL da raiz de migrations, em ordem NNNN_, exigindo versões consecutivas desde 0001. Bootstrap administrativo permanece separado. O histórico aplicado deve ser um prefixo exato das versões conhecidas; checksum alterado, versão desconhecida, lacuna ou schema existente sem histórico válido interrompem a operação.

Apply adquire bloqueio consultivo, verifica o histórico e executa somente versões pendentes numa transação única com seus registros de versão. Num banco existente com 0001, aplica apenas 0002. Num banco sem schema, aplica 0001 e 0002. Falha no lote impede seu commit; falha de conexão no commit exige inspecionar o estado antes de repetir. O executor suporta SQL transacional; operações como CREATE DATABASE continuam fora dele.

Check é somente leitura e exige todas as versões do binário. Esta versão da API recusa iniciar em PostgreSQL enquanto 0002 estiver pendente; essa recusa foi verificada antes da aplicação. Depois da autorização, 0002 foi aplicada em financeiro_go e financeiro_go_test e a inicialização da API PostgreSQL foi confirmada. Binários antigos que reconhecem somente 0001 recusam o histórico novo; não há downgrade automático.

## Experimentar agora em PowerShell

```powershell
Set-Location D:\GO\Financeiro-GO
$env:PERSISTENCE = 'memory'
go run ./cmd/financeiro
```

Em outro terminal:

```powershell
$base = 'http://127.0.0.1:8081/api/v1'
Invoke-RestMethod -Method Post -Uri "$base/cards" -ContentType 'application/json' -Body '{"name":"Cartao ficticio"}'
Invoke-RestMethod "$base/cards"
```

## Validação concluída

Executados: gofmt nos arquivos Go alterados, `go test ./...`, `go vet ./...` e `go build ./...`, todos aprovados. Testes cobrem nomes, limite Unicode, cancelamento, cópias, IDs distintos, cadastros concorrentes, contrato HTTP, saldo de conta preservado e erro interno protegido. Testes do executor cobrem planejamento incremental, ausência de reaplicação de 0001, histórico divergente e estabilidade do checksum.

`TestPostgresCardsIntegration` passou contra financeiro_go_test: cadastro via HTTP, persistência por outra conexão e constraint SQL. Ele limpa apenas seus próprios IDs e não aplica migrations. Os testes existentes de contas também passaram no banco atualizado, incluindo rollback, limites de saldo, concorrência e cancelamento.

Em 25/09/2026, após autorização explícita, a inspeção confirmou versão 0001 e ausência de cards nos dois bancos. O executor aplicou somente 0002, preservando o histórico anterior. A verificação posterior confirmou a tabela e duas versões registradas com checksums válidos. A API iniciou no modo PostgreSQL e GET /api/v1/cards funcionou no banco principal, sem inserir dados nele.

A primeira execução completa identificou que o teste de configuração sem banco herdava PGDATABASE usado na inspeção. O teste foi isolado com t.Setenv; a configuração da aplicação não foi alterada. A suíte completa foi repetida com PGDATABASE definido e integração habilitada: aprovada sem SKIP, assim como vet e build. O CI remoto não acessa esse banco local; sua evidência é distinta da integração real executada aqui.

O primeiro incremento está concluído. A fase 4 completa continua aberta até compras, faturas, parcelas e pagamentos. SQL 0001 e 0002 aplicados devem permanecer imutáveis.
