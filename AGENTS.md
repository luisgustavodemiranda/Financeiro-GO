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
- Usar main para versões estáveis e develop para integração. Cada etapa usa feature/<etapa>, criada a partir de develop; PRs de feature têm develop como destino.
- Para iniciar ou retomar etapas, seguir .agents/skills/financeiro-etapa/SKILL.md. Não implementar novas etapas diretamente em main ou develop.
- Antes de entregar código: gofmt nos arquivos alterados, go test ./..., go vet ./... e go build ./.... Reportar limitações reais.
- Documentar cada fase em doc/. Não declarar componentes planejados como implementados.
