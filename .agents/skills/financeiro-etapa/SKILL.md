---
name: financeiro-etapa
description: Iniciar ou retomar uma etapa do Financeiro-GO em feature/* a partir de develop, implementar, testar, documentar, fazer commit, push e abrir PR para develop. Use para novas etapas ou continuação do plano; nunca faz merge automático.
---

# Etapas do Financeiro-GO

Siga AGENTS.md da raiz e sua política de contexto eficiente. Consulte arquitetura e plano atuais; leia código e documentação adicional apenas do incremento. Não recarregue fontes já disponíveis e inalteradas.

## Escopo e branch

Identifique a próxima pendência e explique brevemente a entrega e o critério de conclusão. Uma fase pode conter vários incrementos. Pedidos de somente análise ou planejamento não iniciam implementação nem publicação.

Inspecione status, branch, remotes e histórico. Para nova etapa, faça fetch quando disponível, confira develop contra origin/develop e crie feature/<etapa> a partir de develop. Atualize develop apenas por fast-forward em árvore limpa. Retome a feature correspondente quando existir; não crie uma nova a cada mensagem de continuação.

Preserve alterações locais/staged. Não use reset, clean, stash, rebase, force push ou commit de trabalho alheio para conseguir trocar de branch. Só carregue mudanças para uma feature se pertencerem à etapa e a base for develop. Se outra feature, divergência ou ausência de develop impedir uma base correta, resolva o contexto antes da troca; não invente histórico.

## Implementação

Implemente em pequenos incrementos e siga as regras e checks do AGENTS.md. Para provisionamento, migrations, schema ou permissões, leia [financeiro-banco](../financeiro-banco/SKILL.md) e retome aqui após a operação; não abra um segundo fluxo Git. Leitura de regra financeira sem alteração de persistência não exige carregar a skill de banco.

Documente resultados no documento da etapa e marque no plano apenas o que foi comprovado. Teste ignorado por falta de banco não aprova integração. Explique execução, arquivos para estudo e próximo passo na medida da mudança.

## Commit e PR automáticos

A autorização permanente do usuário ao invocar financeiro-etapa cobre feature, implementação, testes, documentação, commit, push e abertura/atualização de PR para develop. Não peça confirmação intermediária para essas ações. Restrições explícitas, como trabalho somente local, prevalecem. Essa autorização não inclui alterações de banco, merge ou exclusão de branches.

Revise diff staged/unstaged e arquivos novos; use staging explícito e commits Conventional Commits apenas da etapa. Não crie commit vazio. Envie a feature para o remoto do Financeiro-GO e procure PR aberto da mesma feature/base antes de criar outro. Atualize título e descrição pelo resultado final, com comportamento, testes e limitações; nunca inclua segredos.

Confira CI do SHA publicado, sem confundir checks locais com remotos. Reutilize resultado já concluído desse SHA; acompanhe execução pendente com espera adequada, sem polling excessivo. Corrija falhas da etapa. Se houver bloqueio externo ou validação pendente, publique somente trabalho coerente e deixe o PR em rascunho com o motivo. Com incremento concluído e checks aprovados, deixe pronto para revisão.

Nunca faça merge automaticamente. Informe branch, entrega, checks, pendências e link real do PR. A promoção develop → main é uma decisão de versão separada.
