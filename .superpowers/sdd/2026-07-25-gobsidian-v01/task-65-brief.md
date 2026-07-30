### Task 65: Tool `note_move`

R2 e RF-35. **É a tarefa mais perigosa do marco:** toca notas que o usuário não pediu para tocar.

#### O que vincula esta tarefa

- **Não há transação entre arquivos, e o arquivo move POR ÚLTIMO.** Falha antes do move deixa links reescritos apontando para um arquivo que **ainda existe no lugar antigo** — inconsistente, mas nada quebrado, e uma nova execução conserta. O relatório de falha parcial diz **exatamente** quais notas foram reescritas.
- **`--read-only` remove `note_move` do `ListTools`**, não a rejeita depois. A Task 60 já tem o padrão do teste; o teste afirma a **lista**, nome por nome.
- **Confinamento vale para `to` também.** Mover para fora do cofre é a forma mais direta de exfiltrar arquivo, e `to` vem do chamador.
- **Schema que promete e código que ignora é pior que parâmetro ausente.** Cada parâmetro de `docs/TOOLS.md` §`note_move` precisa de teste que prove que ele age.

#### O contrato

`docs/TOOLS.md` §`note_move` já define tudo: `from`, `to`, `update_links` (default `true`), `create_folders` (default `true`), `dry_run`. Retorno: caminho novo, lista de notas reescritas, contagem de links atualizados, e em `dry_run` o diff de cada nota afetada.

#### A ordem, que é decisão fechada

1. Resolver e confinar `from` **e** `to`.
2. Levantar as notas afetadas por `idx.Backlinks(from)`, e para cada uma pegar os `Links` de `idx.Get(bl.From)`.
3. Se `dry_run`: montar os diffs e **retornar sem tocar o disco**.
4. Reescrever cada nota afetada, uma a uma, com `WriteAtomic`.
5. **Mover o arquivo por último.**
6. Se algo falhar no meio, o retorno diz **exatamente** quais notas foram reescritas.

#### O teste que sustenta a tarefa

```go
// TestNoteMovePartialFailureReportsWhatWasApplied e o teste que separa esta
// tool de uma que destroi o cofre em silencio. Sem transacao entre arquivos, a
// unica garantia possivel e o relatorio ser exato — e um relatorio exato so
// vale se alguem o testou com uma falha de verdade no meio.
func TestNoteMovePartialFailureReportsWhatWasApplied(t *testing.T) {
	// Cofre com tres notas apontando para o alvo. A do meio e tornada
	// impossivel de escrever (somente leitura no Windows, permissao no Unix)
	// para a falha cair no meio do lote, nao na primeira nem na ultima.
	// ... montar cofre e indice ...

	res, err := svc.MoveNote(ctx, service.MoveOptions{From: "alvo.md", To: "Novo/alvo.md"})

	if err == nil {
		t.Fatal("esperava falha parcial; o cenario tornou uma nota inescrivel")
	}
	// O relatorio tem de listar as que FORAM reescritas, e so elas.
	if len(res.Rewritten) != 1 || res.Rewritten[0] != "a.md" {
		t.Errorf("Rewritten = %v; quer exatamente [a.md] — a que foi escrita antes da falha", res.Rewritten)
	}
	// E o arquivo NAO pode ter se movido: o move e o ultimo passo, e ele nao
	// chegou a acontecer.
	if _, err := os.Stat(filepath.Join(root, "alvo.md")); err != nil {
		t.Errorf("o alvo saiu do lugar apesar da falha: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "Novo", "alvo.md")); err == nil {
		t.Error("o alvo foi movido apesar da falha antes do passo de move")
	}
	// E o que foi reescrito aponta para o caminho NOVO — o cofre fica
	// inconsistente, mas nada quebrado, e uma nova execucao conserta.
	// ... afirmar o conteudo de a.md ...
}
```

#### Verificações além dos passos

- `dry_run: true` deixa o mtime de **todas** as notas afetadas inalterado, e o do alvo também? Compare, não confie.
- `update_links: false` move o arquivo e **não** toca nenhuma outra nota?
- `create_folders: false` com pasta inexistente recusa com erro que nomeia a pasta?
- `to` fora do cofre é recusado? E `to` que já existe?
- Mover uma nota **sem** backlinks funciona, e reporta zero reescritas?
- Alias e âncora preservados no cofre real, não só no teste unitário do 64.
- Ponta a ponta: mover com o servidor rodando e confirmar que `vault_search` e `link_graph` refletem o caminho novo — o watcher e o `index.MoveNote` já existem, mas a composição não foi testada.
- Mover uma nota para dentro de `.trash/`: o que acontece? **Decida e diga.**

**Prova de mutação obrigatória:** troque a ordem para mover o arquivo **antes** de reescrever, e confirme que `TestNoteMovePartialFailureReportsWhatWasApplied` reprova.

#### Regras de execução

Valem para toda tarefa deste plano e não são negociáveis.

- **O plano é a fonte.** Transcreva o desenho desta seção; não improvise uma variante. Se um teste falhar por motivo que a seção não explica, **pare e reporte** — não ajuste a expectativa para o código passar.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** **`go mod tidy` está proibido.**
- **Toda escrita no cofre usa `writer.WriteAtomic`.** Nunca abra o arquivo do usuário para escrita direta. EOL e BOM preservados byte a byte (RF-38).
- **Confinamento vale para origem E destino.** `vault.Resolve` e `Canonicalize` são as mesmas duas camadas; escrita não ganha caminho novo.
- **`dry_run` não toca o disco.** Prove por mtime, não por intenção.
- **Verde obrigatório antes do commit:** `pwsh -File scripts/verify.ps1`, que agora inclui o `golangci-lint` como oitava etapa.
- Commits em Conventional Commits, em inglês. Sem `helpers.go`, `utils.go` ou `common.go`.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-65-report.md`: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a prova de mutação com a **saída real colada**, nomeando o teste que reprovou; a tabela de verificações acima com o resultado **real** de cada uma, inclusive as que deram certo; arquivos alterados; o que ficou de fora e por quê; e `git status --porcelain`.

**Não escreva número que você não mediu** — escreva "não medido". **Prova de mutação no condicional não é prova:** use `scripts/mutate.ps1` e cole o que ele imprimiu. Rode `pwsh -File scripts/audit_reports.ps1 65` antes de entregar.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, e preocupações.

**Files:** Modify `internal/service/`, `internal/mcpsrv/`, testes
**Commit:** `feat(mcpsrv): note_move with faithful link rewriting`

---

