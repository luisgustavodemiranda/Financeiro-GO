# Plano de desenvolvimento

- [x] Fase 1: configuração, HTTP local, testes e documentação.
- [x] Fase 2: dinheiro em centavos, contas, receitas, despesas e saldo com repositório em memória na API e nos testes. Dados perdidos ao reiniciar. Entrega em `doc/02-contas-lancamentos.md`.
- [ ] Fase 3: PostgreSQL exclusivo, migrations, repositórios pgx e testes de integração.
  - [x] Adaptador pgx, transações, bloqueio por conta, contextos e configuração explícita.
  - [x] Suporte a DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD e DB_SSLMODE; nome padrão financeiro_go confirmado pelo usuário.
  - [x] Migration 0001 e comando separado de aplicação com versão/checksum, sem execução automática.
  - [x] Testes de integração preparados com opt-in e banco exclusivo de testes.
  - [ ] Criar/configurar bancos exclusivos e aplicar migration após autorização específica.
  - [ ] Executar e aprovar integração real, incluindo persistência, rollback e concorrência.
- [ ] Fase 4: cartões, faturas, pagamentos e parcelas sem duplicidade.
- [ ] Fase 5: orçamento mensal e projeções com valores previstos e realizados.
- [ ] Fase 6: interface React + TypeScript e dashboard.
- [ ] Fase 7: importação CSV com prévia, validação e controle de reimportação.
- [ ] Fase 8: autenticação, isolamento familiar, backup e preparação para publicação.

Cada incremento termina com testes e documentação. Próximo incremento: provisionar os bancos financeiro_go e financeiro_go_test e aplicar a migration preparada, mediante autorização; então executar a integração real e validar o modo PostgreSQL. A fase 3 permanece aberta até essa verificação. Nenhum banco foi criado, acessado ou alterado nesta entrega. Detalhes em `doc/03-postgresql.md`.
