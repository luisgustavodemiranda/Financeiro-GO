# Orientações do Financeiro-GO

- Documentação e explicações em português do Brasil; identificadores Go conforme convenções da linguagem.
- Ler FINANCEIROGO.md e o plano antes de mudanças relevantes.
- Desenvolver em pequenos incrementos, explicando decisões para quem está aprendendo Go.
- Manter monólito modular, serviços e interfaces de repositório. Não adicionar abstrações sem necessidade.
- Usar centavos inteiros e testar as regras financeiras. Não inventar valores de salários ou regras tributárias.
- Nunca incluir faturas, holerites, dados pessoais ou credenciais no código, fixtures ou histórico Git.
- Não alterar o Fiscal-GO ou seu banco. Banco deste projeto é independente.
- Mudanças no banco usam migrations. Executar migrations somente com autorização do usuário.
- Preservar alterações existentes. Não publicar, fazer push ou merge sem autorização.
- Invocar financeiro-etapa autoriza criar/retomar feature, implementar, testar, documentar, fazer commit, push e abrir/atualizar PR para develop, sem confirmação intermediária. Restrições explícitas do pedido prevalecem; banco e merge continuam exigindo autorização específica.
- Usar main para versões estáveis e develop para integração. Cada etapa usa feature/<etapa>, criada a partir de develop; PRs de feature têm develop como destino.
- Para iniciar ou retomar etapas, seguir .agents/skills/financeiro-etapa/SKILL.md. Não implementar novas etapas diretamente em main ou develop.
- Antes de entregar código: gofmt nos arquivos alterados, go test ./..., go vet ./... e go build ./.... Reportar limitações reais.
- Documentar cada fase em doc/. Não declarar componentes planejados como implementados.
- Para provisionamento, schema, permissões ou migrations, usar .agents/skills/financeiro-banco/SKILL.md; financeiro-etapa mantém o fluxo Git/PR quando já estiver ativo.

## Contexto e verificação eficientes

- Ler arquitetura e plano antes de mudanças relevantes, reaproveitando o conteúdo já disponível se a versão não mudou. Conferir status/diff ou hash; reler apenas arquivos/trechos alterados. Após perda de contexto, recuperar as fontes necessárias.
- Usar rg para localizar o assunto. Não carregar doc/ inteiro, todas as skills ou o Fiscal-GO por padrão. Ler somente a skill aplicável e as referências necessárias à operação.
- Agrupar consultas independentes, limitar a saída a resultados úteis e não imprimir credenciais. Executar checks uma vez por conjunto de mudanças relevante; repetir se houver alterações que os invalidem, falha ou dúvida concreta. Mudanças apenas de Markdown usam revisão de links, diff e validação das skills; não exigem repetir checks Go locais já válidos.
- Separar evidência de código, CI e banco: revisão/commit não invalida um teste do mesmo código; resultado antigo de inspeção não prova estado atual do banco. Antes de executar DDL, revalidar destino, schema e versão.
- Ao pausar, registrar resumo curto no documento da etapa: branch/PR, entrega, checks, pendências e autorização ainda necessária, sem duplicar regras do AGENTS.md. Não afirmar economia exata de tokens sem medição.
