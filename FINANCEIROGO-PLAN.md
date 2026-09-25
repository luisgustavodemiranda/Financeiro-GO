# Plano de desenvolvimento

- [x] Fase 1: configuração, HTTP local, testes e documentação.
- [x] Fase 2: dinheiro em centavos, contas, receitas, despesas e saldo com repositório em memória na API e nos testes. Dados perdidos ao reiniciar. Entrega em `doc/02-contas-lancamentos.md`.
- [ ] Fase 3: PostgreSQL exclusivo, migrations, repositórios pgx e testes de integração.
  - [x] Adaptador pgx, transações, bloqueio por conta, contextos e configuração explícita.
  - [x] Suporte a DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD e DB_SSLMODE; nome padrão financeiro_go confirmado pelo usuário.
  - [x] Migration 0001 e comando separado de aplicação com versão/checksum, sem execução automática.
  - [x] Testes de integração preparados com opt-in e banco exclusivo de testes.
  - [x] Conexão administrativa inspecionada somente por leitura; bancos ainda ausentes e bootstrap para usuários/bancos exclusivos preparado em doc/04-preparacao-integracao-postgres.md.
  - [ ] Criar/configurar bancos exclusivos e aplicar migration após autorização específica.
  - [ ] Executar e aprovar integração real, incluindo persistência, rollback e concorrência.
- [ ] Fase 4: cartões, faturas, pagamentos e parcelas sem duplicidade.
- [ ] Fase 5: orçamento mensal e projeções com valores previstos e realizados.
- [ ] Fase 6: interface React + TypeScript e dashboard.
- [ ] Fase 7: importação CSV com prévia, validação e controle de reimportação.
- [ ] Fase 8: autenticação, isolamento familiar, backup e preparação para publicação.

Cada incremento termina com testes e documentação. Próximo incremento: executar o provisionamento revisável em migrations/bootstrap/0000_create_databases.sql e aplicar a migration 0001 nos dois bancos, mediante autorização; então executar a integração real. A fase 3 permanece aberta até essa verificação. A inspeção administrativa foi somente leitura; nenhum banco foi criado ou alterado. Histórico em `doc/03-postgresql.md`; preparação atual em `doc/04-preparacao-integracao-postgres.md`.
