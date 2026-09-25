---
name: financeiro-etapa
description: Iniciar ou retomar uma etapa do Financeiro-GO em feature/* a partir de develop, com implementação incremental, testes e documentação. Use quando o usuário pedir uma nova etapa, feature ou continuação do plano deste projeto.
---

# Etapas do Financeiro-GO

Localize a raiz do Financeiro-GO pela pasta desta skill ou pelo workspace. Leia `AGENTS.md`, `FINANCEIROGO.md`, `FINANCEIROGO-PLAN.md` e `doc/processo-desenvolvimento.md` na raiz, aproveitando versões já lidas e ainda atuais. Inspecione somente código e documentos relevantes à etapa.

## Delimitar a entrega

Identifique no plano o que foi entregue e o que falta validar. Explique brevemente o incremento e seus critérios de conclusão em português, com contexto para alguém aprendendo Go. Uma etapa pode conter vários incrementos; não marque a fase inteira como concluída porque apenas seu código compila. Se o pedido for somente planejamento ou análise, não implemente nem crie branch sem necessidade.

## Preparar ou retomar a feature

- Inspecione status, branch atual, remotes e histórico antes de qualquer troca. `main` é estável; `develop` integra; `feature/<etapa>` recebe o trabalho.
- Para nova implementação, use um nome curto em minúsculas, com hífens, como `feature/fase-3-integracao-postgres`. Crie-a a partir de `develop`, não de `main` nem de outra feature.
- Se a feature correspondente já existe, examine seu estado e retome-a; não a recrie nem redefina seu ponteiro. Se o usuário já está trabalhando em outra feature, esclareça a relação antes de misturar escopos.
- Preserve mudanças locais e staged. Não faça reset, clean, stash ou commit automático para conseguir trocar de branch. Se elas pertencem à etapa e a origem é develop, carregá-las para uma nova feature pode ser adequado; caso contrário, mantenha-as e resolva o contexto antes da troca.
- Quando houver remoto acessível, faça fetch e confira a relação entre develop local e origin/develop. Atualização local apenas por fast-forward, em árvore limpa. Não faça rebase, force push ou descarte commits para resolver divergência; reporte o conflito e peça a decisão necessária.
- Se develop existir só no remoto, crie a branch local com tracking. Se não existir em nenhum lugar, não invente sua base: proponha a inicialização conforme o estado do projeto. Um repositório ainda sem commits precisa de publicação inicial separada.

## Implementar e verificar

Trabalhe em pequenos incrementos no fluxo handler HTTP → serviço → interface de repositório. Preserve as regras e limites financeiros do projeto, com dinheiro em centavos int64. Inclua testes de comportamento relevantes, usando dados fictícios.

Antes de entregar código, execute gofmt nos arquivos Go alterados, `go test ./...`, `go vet ./...` e `go build ./...`; revise `git diff --check`, o diff staged e unstaged e os arquivos novos. Testes ignorados por falta de banco não equivalem a integração aprovada. Resolva falhas dentro do escopo e reporte limitações reais.

Documente o incremento em doc/ e atualize o plano apenas com resultados comprovados. Explique como executar, o caminho da requisição, os arquivos para estudar e o próximo incremento.

## Publicar quando autorizado

Respeite a autorização já existente na conversa, sem pedir confirmações repetidas para ações cobertas por ela. Invocar esta skill não concede por si só permissão para commit, push, abertura de PR, merge ou execução de migrations. Conclua a preparação revisável antes de solicitar uma autorização que realmente falte.

Quando commit estiver autorizado, revise os arquivos e faça staging explícito, sem `git add .`. Use mensagens curtas em Conventional Commits. Quando publicação da feature e PR estiverem autorizados, envie a feature e abra o PR com base `develop`, descrevendo comportamento, testes e limitações. Verifique os checks do commit publicado; não trate testes locais como CI remoto aprovado.

Merge exige autorização explícita; nunca o execute só porque o CI passou. A promoção de develop para main é uma decisão de versão separada. Não exclua branches remotas sem autorização.

Conclua informando a feature, entrega, validações executadas e pendências. Inclua links de commits/PRs apenas quando realmente existirem.
