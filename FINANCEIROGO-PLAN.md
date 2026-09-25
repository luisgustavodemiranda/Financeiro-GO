# Plano de desenvolvimento

- [x] Fase 1: configuração, HTTP local, testes e documentação.
- [x] Fase 2: dinheiro em centavos, contas, receitas, despesas e saldo com repositório em memória na API e nos testes. Dados perdidos ao reiniciar. Entrega em `doc/02-contas-lancamentos.md`.
- [x] Fase 3: PostgreSQL exclusivo, migrations, repositórios pgx e testes de integração.
  - [x] Adaptador pgx, transações, bloqueio por conta, contextos e configuração explícita.
  - [x] Suporte a DB_HOST, DB_PORT, DB_NAME, DB_USER, DB_PASSWORD e DB_SSLMODE; nome padrão financeiro_go confirmado pelo usuário.
  - [x] Migration 0001 e comando separado de aplicação com versão/checksum, sem execução automática.
  - [x] Testes de integração preparados com opt-in e banco exclusivo de testes.
  - [x] Conexão administrativa inspecionada somente por leitura; bancos ainda ausentes e bootstrap para usuários/bancos exclusivos preparado em doc/04-preparacao-integracao-postgres.md.
  - [x] Bancos criados pelo usuário; logins exclusivos configurados e migration 0001 aplicada após autorização específica.
  - [x] Integração real aprovada, incluindo persistência após reinício da API, rollback, limites de int64, concorrência e cancelamento.
- [ ] Fase 4: cartões, faturas, pagamentos e parcelas sem duplicidade.
  - [x] Cadastro e listagem de cartões por nome, serviço, repositórios e testes locais.
  - [x] Executor incremental preparado, preservando o checksum da migration 0001; migration 0002 de cartões criada.
  - [ ] Aplicar 0002 com autorização específica e validar cartões e upgrade em PostgreSQL real.
  - [ ] Compras, faturas, parcelas e pagamentos sem duplicar despesa.
- [ ] Fase 5: orçamento mensal e projeções com valores previstos e realizados.
- [ ] Fase 6: interface React + TypeScript e dashboard.
- [ ] Fase 7: importação CSV com prévia, validação e controle de reimportação.
- [ ] Fase 8: autenticação, isolamento familiar, backup e preparação para publicação.

Cada incremento termina com testes e documentação. Fase 3 validada em `doc/05-integracao-postgres-validada.md`. O primeiro incremento da fase 4 está em `doc/06-cadastro-cartoes.md`: cadastro/listagem e executor incremental implementados, mas aplicação de 0002 e integração real ainda pendentes de autorização. O próximo passo é validar essa evolução antes de iniciar compras e faturas.
