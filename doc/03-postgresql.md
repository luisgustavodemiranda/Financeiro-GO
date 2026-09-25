# Fase 3 — Preparação da persistência PostgreSQL

## Estado da entrega

Implementados em código: adaptador pgx v5.11.0, pool de conexões, configuração de persistência, propagação de contexto, transações, migration inicial com executor separado e testes de integração opt-in. O contrato HTTP e os centavos `int64` da fase 2 foram mantidos.

O usuário informou que ainda não existe banco exclusivo e pediu preparar configuração e migrations. **Nenhum banco foi criado, acessado ou alterado; nenhuma migration foi executada. A integração PostgreSQL ainda não foi validada contra um servidor real.** A fase 3 permanece aberta no plano. Não foram acessados arquivos, configurações ou banco do Fiscal-GO nesta fase.

## Como o fluxo mudou

`cmd/financeiro` escolhe o repositório e o entrega ao handler. `PERSISTENCE=memory` mantém o modo anterior. `PERSISTENCE=postgres` abre o pool, verifica conectividade, nome do banco, usuário sem superusuário e versão/checksum da migration antes de iniciar HTTP. Se isso falha, o processo encerra; não troca para memória.

`r.Context()` passa pelo serviço até o repositório. O contexto carrega cancelamento e prazo, não dados financeiros. O adaptador acrescenta `DATABASE_TIMEOUT` a cada operação. Isso permite interromper uma consulta ou espera por bloqueio quando o cliente cancela ou o prazo termina. O repositório em memória também verifica cancelamento.

Em `PostgresRepository.Update`:

1. Abre uma transação e lê a conta com `SELECT ... FOR UPDATE`.
2. Lê os lançamentos e entrega uma cópia do estado ao serviço.
3. O serviço aplica as mesmas validações e verificações de overflow da fase 2.
4. O adaptador insere somente o novo lançamento e atualiza o saldo.
5. `Commit` confirma ambos; erro anterior ao commit provoca rollback.

O bloqueio pertence ao PostgreSQL e coordena escritores da mesma conta mesmo em processos diferentes. Para ler conta e histórico, uma transação `REPEATABLE READ` evita combinar saldos e lançamentos de instantes diferentes. `List` usa uma consulta única. A interface `Update` permite apenas acrescentar um lançamento e alterar o saldo; mudanças no cadastro ou histórico são rejeitadas.

Referências técnicas: [transações e bloqueios PostgreSQL](https://www.postgresql.org/docs/current/explicit-locking.html) e [documentação do pool pgx](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool).

## Schema e migration

Arquivo: `migrations/0001_accounts_entries.sql`.

| Tabela no schema financeiro | Responsabilidade |
| --- | --- |
| `accounts` | ID identity, nome, saldo inicial, data de referência e saldo atual |
| `entries` | ID textual, FK da conta, sequência por conta, tipo, descrição, centavos e data |
| `schema_migrations` | Versão, checksum do arquivo e instante da aplicação |

Dinheiro é `bigint`, correspondente a `int64`. O banco exige valores positivos nos lançamentos, tipos válidos, FK e unicidade da sequência por conta. O índice dessa unicidade também atende consultas de histórico por conta. IDs continuam strings no JSON; contas usam IDs numéricos do PostgreSQL e lançamentos usam `conta-sequência`. Lacunas em IDs de contas são normais após falhas de inserção.

O serviço continua responsável pela relação entre saldo inicial, datas, lançamentos e saldo final. Não há triggers que recalculam saldos: escritas financeiras devem passar pelo serviço. Não há SQL de cartões, transferências, importação ou dados reais.

Para manter os adaptadores compatíveis, datas com ano zero e textos com caractere NUL agora são rejeitados pelo serviço. As datas válidas estão entre `0001-01-01` e `9999-12-31`.

O comando de migration usa transação e bloqueio consultivo, verifica o nome do banco e consulta o catálogo para detectar o schema. Se ele já existe, exige a versão/checksum esperados; não adota um schema desconhecido. Repetir a migration já aplicada não recria tabelas. O checksum normaliza CRLF/LF para funcionar em Windows e Linux. A API apenas verifica esse registro; não compara cada objeto do catálogo nem corrige alterações manuais no schema.

Nesta fase o executor conhece somente a migration 0001; não é um gerenciador genérico. Depois de aplicada, ela deve permanecer imutável. Uma próxima mudança exigirá novo arquivo/versionamento e evolução do executor. Não há comando de rollback destrutivo.

## Configuração preparada

Incremento de configuração: o usuário confirmou `financeiro_go` como nome do banco. A API e o comando de migration agora aceitam também `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` e `DB_SSLMODE`. Host, usuário e senha são obrigatórios nesse formato; os padrões são porta `5432`, nome `financeiro_go` e SSL `require`. `DB_SSLMODE=disable` pode ser informado explicitamente. Nenhum valor de credencial fornecido pelo usuário foi gravado nos arquivos.

Se `FINANCEIRO_DATABASE_URL` estiver preenchida, ela tem precedência e as variáveis `DB_*` são ignoradas. A URL é construída com escape de caracteres especiais da senha. `PERSISTENCE` continua sendo `memory` por padrão; definir `DB_*` não ativa nem cria um banco. Os testes de integração continuam exigindo `FINANCEIRO_TEST_DATABASE_URL` separada.

Exemplo de entrada local em PowerShell, sem senha no histórico:

```powershell
Remove-Item Env:FINANCEIRO_DATABASE_URL -ErrorAction SilentlyContinue
$env:DB_HOST = Read-Host 'Host do PostgreSQL'
$env:DB_PORT = '5432'
$env:DB_NAME = 'financeiro_go'
$env:DB_USER = Read-Host 'Usuario do PostgreSQL'
$secretInput = Read-Host 'Senha do PostgreSQL' -AsSecureString
$env:DB_PASSWORD = [System.Net.NetworkCredential]::new('', $secretInput).Password
Remove-Variable secretInput
$env:DB_SSLMODE = 'disable' # Opção informada para o servidor local deste projeto.
```

Esses comandos apenas preparam o ambiente do terminal. A criação do banco e aplicação da migration continuam pendentes de autorização específica. O requisito de usuário sem superusuário permanece; a configuração recebida ainda não foi verificada em conexão real.

| Variável | Uso |
| --- | --- |
| `PERSISTENCE` | `memory` por padrão; `postgres` ativa o adaptador |
| `FINANCEIRO_DATABASE_URL` | Conexão exclusiva da aplicação ou do comando de migration |
| `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE` | Alternativa à URL; usada pela API e pelo comando de migration |
| `DATABASE_TIMEOUT` | Prazo positivo por operação e na inicialização; padrão `10s` |
| `FINANCEIRO_TEST_DATABASE_URL` | Opt-in dos testes; aceita somente `financeiro_go_test` |

O pool usa até cinco conexões. Somente os nomes `financeiro_go` e `financeiro_go_test` são aceitos, para impedir o uso acidental de outro projeto. A conexão confere também `current_database()` e recusa superusuário. Use usuário dedicado e sem credenciais compartilhadas com o Fiscal-GO. Erros de configuração/conexão não imprimem a URL ou senha. Arquivos `.env` não são carregados automaticamente e não devem ser versionados.

Os bancos e usuários precisam ser provisionados separadamente, após autorização. Para aplicar a migration, o usuário dedicado precisa poder criar o schema no banco de destino. Se forem usados usuários separados de migration e aplicação, será necessário conceder ao usuário da aplicação `USAGE` no schema e na sequência, `SELECT` nas três tabelas, `INSERT` em contas/lançamentos e `UPDATE` em contas. O usuário dos testes também precisa de `DELETE` em contas/lançamentos para limpar suas fixtures. A definição e aplicação dessas permissões ficam para o provisionamento autorizado.

## Terminal do VS Code — disponível agora

```powershell
Set-Location D:\GO\Financeiro-GO
$env:PERSISTENCE = 'memory'
go run ./cmd/financeiro
```

Em outro terminal:

```powershell
Invoke-RestMethod 'http://127.0.0.1:8081/api/v1/status'
go test ./...
go vet ./...
go build ./...
```

Os exemplos de cadastro, receita e despesa em `doc/02-contas-lancamentos.md` continuam válidos. Encerre o servidor com Ctrl+C.

## Comandos futuros — somente após provisionamento e autorização

Não executados nesta entrega. Revise a migration e confirme o banco de destino antes de aplicá-la. Insira a URL localmente; não envie credenciais em conversas ou arquivos versionados.

```powershell
# PowerShell 7: a entrada mascarada evita eco da URL com credenciais.
$env:FINANCEIRO_DATABASE_URL = Read-Host 'URL do banco financeiro_go' -MaskInput
# Aplicar somente após autorização específica:
go run ./cmd/migrate -apply
# Só continue se o comando anterior concluir com sucesso.
$env:PERSISTENCE = 'postgres'
go run ./cmd/financeiro
```

No Windows PowerShell 5.1, que não tem `-MaskInput`, use a alternativa:

```powershell
$secureUrl = Read-Host 'URL do banco exclusivo' -AsSecureString
$env:FINANCEIRO_DATABASE_URL = [System.Net.NetworkCredential]::new('', $secureUrl).Password
Remove-Variable secureUrl
```

A URL fica no ambiente do processo, em texto, como exigido pelo driver; a entrada mascarada evita apenas sua exibição e registro como comando no histórico. Ajuste TLS ao servidor; não há desativação automática de TLS no código. `go run ./cmd/migrate` sem `-apply` encerra com mensagem e código não zero, sem conectar ao banco.

## Integração preparada, ainda não executada

Provisionar `financeiro_go_test` separadamente e aplicar a mesma migration com autorização, apontando `FINANCEIRO_DATABASE_URL` para esse banco. Depois configurar `FINANCEIRO_TEST_DATABASE_URL` com a conexão de testes. O teste não cria schema e não executa migrations.

```powershell
# PowerShell 7; no 5.1 use a alternativa de entrada acima.
$env:FINANCEIRO_TEST_DATABASE_URL = Read-Host 'URL do banco financeiro_go_test' -MaskInput
go test ./internal/contas -run '^TestPostgresIntegration$' -count=1 -v
```

Sem a variável, o teste é explicitamente ignorado (`SKIP`). Com variável inválida, banco incorreto, conexão indisponível ou migration ausente, ele falha. Executar a integração grava apenas fixtures fictícias e tenta remover somente os IDs criados pela própria execução; não usa `TRUNCATE`, não reinicia sequências e não apaga schemas. Uma interrupção abrupta pode deixar fixtures no banco de testes.

Cenários escritos: criação/listagem, HTTP com adaptador real, consulta por outro pool, saldo inicial separado, limites de `int64`, erro do callback, erro SQL com rollback, recuperação da conexão, isolamento entre contas, 24 registros concorrentes entre pools, consistência entre saldo/histórico e cancelamento durante espera por bloqueio. Esses cenários só poderão ser considerados aprovados após rodar contra o banco.

## Validação e limitações

Os testes existentes foram adaptados para contexto e acrescentados testes unitários de cancelamento, proteção do histórico, configuração do PostgreSQL e resposta HTTP sem exposição de erro interno.

Validação executada nesta entrega com Go 1.26.5, Windows/amd64:

- `gofmt` aplicado nos arquivos Go alterados; verificação final sem arquivos pendentes.
- `go test ./...`: aprovado para os testes disponíveis; integração PostgreSQL ignorada.
- `go vet ./...`: aprovado.
- `go build ./...`: aprovado, incluindo o comando de migration.
- `go test ./internal/contas -run '^TestPostgresIntegration$' -count=1 -v`: confirmou `SKIP` por ausência de conexão de testes. Isso não comprova o comportamento SQL.
- `go run ./cmd/migrate` sem `-apply`: recusou a operação antes de conectar, com código de saída 2 do programa, como esperado.

O cache padrão do Go apresentou acesso negado no ambiente de execução. Os checks finais usaram um cache temporário exclusivo, sem alterar configuração global:

```powershell
$env:GOCACHE = Join-Path $env:TEMP 'financeiro-go-build-cache'
```

Não foi executado `go test -race`; este ambiente está com `CGO_ENABLED=0`.

Ainda sem banco provisionado, frontend, autenticação, paginação, idempotência ou importação de dados da memória para PostgreSQL. Escolher PostgreSQL não migra dados do processo em memória. O adaptador carrega o histórico inteiro em cada registro, preservando a interface didática da fase 2; uma evolução futura pode reduzir esse custo. Falhas ambíguas de rede no commit não têm repetição automática: sem idempotência, reenviar um POST pode duplicar a operação. `/health` continua indicando apenas o processo HTTP, não a saúde contínua do banco.

A pasta segue sem `.git`, portanto não foi possível obter diff contra histórico. Não houve commit, push ou alteração no Fiscal-GO.

## Ordem de estudo e próximo incremento

1. Compare `internal/contas/memory.go` com `internal/contas/postgres.go`: mesmo contrato, mecanismos diferentes para armazenamento e exclusão mútua.
2. Leia `migrations/0001_accounts_entries.sql` e relacione colunas com `Account` e `Entry`.
3. Leia `internal/database/postgres.go` e a montagem em `cmd/financeiro/main.go`.
4. Leia `internal/contas/postgres_integration_test.go` para entender a verificação planejada contra o banco.

O próximo incremento é provisionar os dois bancos exclusivos, aplicar a migration após autorização e executar a integração real. Só então marcar a fase 3 como concluída e avançar para cartões.
