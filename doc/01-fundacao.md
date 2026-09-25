# Fase 1 — Fundação

Entregue: módulo Go, configuração validada, rotas GET /health e GET /api/v1/status, encerramento por sinal, timeouts e testes de configuração/HTTP.

Sem dependências externas. Porta 8081 para facilitar execução paralela ao Fiscal-GO. Bind local por padrão. /health comprova apenas o processo HTTP; não afirma conectividade com banco.

PostgreSQL, autenticação, módulos financeiros, frontend e importação ainda não implementados. Nenhum banco foi criado ou alterado.

Testes: configuração válida/inválida, limites de porta, duração positiva, contratos HTTP, JSON, método não permitido e rota inexistente.
