# Arquitetura e domínio

## Arquitetura

Monólito modular inspirado no Fiscal-GO. Fluxo implementado: HTTP → serviço do módulo → interface de repositório → memória ou adaptador PostgreSQL (pgx). O modo PostgreSQL foi validado com migration aplicada e integração real nos bancos exclusivos do projeto. React + TypeScript permanece futuro. Módulos são adicionados conforme implementados, sem diretórios vazios.

Implementado: `contas` reúne contas e lançamentos, regras, serviço, interface de repositório e adaptadores em memória e PostgreSQL. `cartoes` contém o cadastro por nome, com integração PostgreSQL validada após 0002, e compras/faturas simples validadas em memória e PostgreSQL após aplicar 0003. Lançamentos permanecem no módulo de contas para manter saldo e registro consistentes. `server` traduz HTTP para chamadas dos serviços; `config` concentra configuração; `database` configura e verifica o pool PostgreSQL. `migrations` contém versões SQL e executor incremental explícito. Futuro: pagamentos, parcelas, orçamentos, importações e relatórios.

Worker separado para importações demoradas; Redis, armazenamento de arquivos e Docker Compose só entram em fases justificadas. O banco será exclusivo deste projeto. Não utilizar credenciais, tabelas ou arquivos de dados do Fiscal-GO.

## Regras implementadas na fase 2

- Dinheiro em `int64`, inclusive como número inteiro no JSON, sem `float64`.
- Receitas e despesas têm valor estritamente positivo; o tipo define soma ou subtração.
- Conta existente, nome, descrição e datas válidas são obrigatórios.
- Saldo inicial é informado explicitamente, pode ser zero ou negativo e não gera receita.
- A data do saldo inicial representa a abertura do dia; lançamentos nessa data são permitidos, anteriores são rejeitados.
- Saldo negativo é permitido. Soma e subtração verificam overflow antes de alterar o estado.
- Registro e atualização do saldo são atômicos no repositório em memória; erros não gravam alterações.

Contrato e validações: [fase 2](doc/02-contas-lancamentos.md).

## Persistência validada na fase 3

- `context.Context` é propagado do HTTP até o repositório. Operações PostgreSQL têm timeout.
- Gravação usa transação e `SELECT ... FOR UPDATE` na conta, mantendo lançamento e saldo juntos. Leitura de conta e histórico usa snapshot consistente com `REPEATABLE READ`.
- Dinheiro usa `bigint` no SQL; datas válidas ficam entre 0001 e 9999. Nome/descrição rejeitam caractere NUL para compatibilidade entre adaptadores.
- `PERSISTENCE=memory` é o padrão; `postgres` exige configuração explícita de conexão e migration verificada antes de servir HTTP. Falhas não ativam memória automaticamente.
- São aceitos apenas `financeiro_go` e `financeiro_go_test`. Integração exige o segundo e não aplica migrations.
- Migration é executada somente pelo comando separado com `-apply`, após autorização. A versão e o checksum ficam registrados no banco.

Preparação histórica: [fase 3](doc/03-postgresql.md). Aplicação e validação reais: [conclusão da fase 3](doc/05-integracao-postgres-validada.md). Usuários exclusivos sem superusuário, migration 0001 aplicada em ambos os bancos e integração executada somente no banco de testes.

## Compras e faturas: incremento atual

O módulo `cartoes` também contém compras e faturas com períodos explícitos, vencimento, total em centavos e fechamento manual. Serviços e HTTP validados em memória e PostgreSQL. Migration 0003 aplicada com autorização nos dois bancos, com integração real aprovada. Contrato e regras em [compras e faturas](doc/07-compras-faturas.md).

A compra reconhece despesa no cartão e obrigação pelo mesmo valor, sem chamar o serviço de despesas bancárias. A data da compra é sua referência; vencimento não é data de pagamento. Ainda não há relatório consolidado nem pagamento de fatura. No futuro, o pagamento deverá reduzir a conta e quitar a obrigação sem reconhecer outra despesa.

## Regras para fases futuras

Cadastro e listagem de cartões não movimentam contas, receitas ou despesas. O contrato inicial armazena apenas ID e nome, sem dados de pagamento. Detalhes e limitações em [fase 4, primeiro incremento](doc/06-cadastro-cartoes.md).

- Dinheiro em centavos inteiros (`int64`) na aplicação. Conferir limites e divisão de centavos nas parcelas. Não calcular valores monetários com `float64`.
- Compra no cartão registra despesa e obrigação. Pagamento da fatura reduz a conta e quita a obrigação; não cria nova despesa.
- Transferências entre contas não são receitas nem despesas.
- Benefício alimentar fica separado do dinheiro bancário.
- Separar data de compra, competência, fechamento, vencimento e pagamento efetivo.
- Valores previstos e realizados são distintos. Confirmar uma previsão não pode duplicá-la.
- Saldo inicial tem data de referência e não pode ser somado novamente às receitas anteriores.
- Férias antecipam parte da remuneração. Não somar automaticamente salário integral e pagamento integral das férias.
- Importação precisa de prévia, confirmação e proteção contra reimportação; transações legítimas iguais devem ser preservadas.
- A soma das parcelas precisa coincidir exatamente com o valor total da compra.
- Previsão mensal não garante saldo antes de cada vencimento.

Parcelas, pagamentos, transferências e importação permanecem futuros. Compras e faturas simples estão disponíveis em memória e PostgreSQL, com persistência validada.
