# Prompt de delegação — M6, tarefas ao modelo barato

Copie do bloco abaixo, inteiro, para o agente orquestrador.

---

Você vai executar seis tarefas do marco M6 do projeto **gobsidian**, um servidor MCP em Go que expõe um cofre Obsidian local sobre stdio, em Windows.

## Antes de tocar em qualquer coisa

Leia, nesta ordem, e não pule:

1. `AGENTS.md` — é o contrato do ambiente. Tudo que vale para toda tarefa está lá: comandos, proibições, ferramentas de Go disponíveis, e como usar o MCP do gopls.
2. `CLAUDE.md` — as regras não negociáveis e as armadilhas que já custaram caro neste repositório. Cada armadilha listada lá aconteceu de verdade.
3. `docs/PRD.md` e `docs/TOOLS.md` — só as seções que a tarefa da vez citar.

**Não leia o plano inteiro.** Cada tarefa tem um brief autocontido; é ele que você executa.

## As seis tarefas, na ordem obrigatória

Execute **uma de cada vez**, na ordem abaixo. Não comece a seguinte antes de a anterior estar commitada e com relatório gravado.

| Ordem | Tarefa | Motivo de estar nesta posição |
|---|---|---|
| 1 | **75** subcomandos `index`, `search`, `inspect` | Isolada em `cmd/gobsidian`; não toca arquivo de nenhuma outra |
| 2 | **69** os quatro parâmetros de schema ignorados | Acrescenta a primeira etapa nova ao `scripts/verify.ps1` |
| 3 | **74** `netcheck` como analisador de `go vet` | Acrescenta a segunda etapa ao `verify.ps1` — nunca junto com a 69 |
| 4 | **70** gerador determinístico de cofre de 5.000 notas | Destrava a 71 |
| 5 | **71** RNF-01, RNF-02, RNF-04 e RNF-07 medidos a 5.000 | Precisa do cofre da 70 |
| 6 | **72** a folga fina do RNF-04 em `limit: 200` | Precisa do número de escala da 71 para mirar |

A ordem não é sugestão. A 71 depende de dados da 70; a 72 depende de números da 71; e 69 e 74 editam o mesmo arquivo.

## O laço, tarefa a tarefa

Para cada N na ordem acima:

```bash
pwsh -File scripts/sdd.ps1 status        # veja o ledger antes de supor qualquer coisa
pwsh -File scripts/sdd.ps1 base N        # ANTES de começar. Grava o commit anterior ao início.
pwsh -File scripts/sdd.ps1 brief N       # o brief é o que você executa
```

Execute o que o brief manda. Depois:

```bash
pwsh -File scripts/verify.ps1            # bateria inteira, para no primeiro erro
pwsh -File scripts/mutate.ps1 -Path <arquivo> -Anchor '<texto>' -Replacement '<texto>' -Test <Nome> -Package ./internal/x/
pwsh -File scripts/audit_reports.ps1 N   # antes de entregar
pwsh -File scripts/sdd.ps1 review N      # empacota o diff desde a base gravada
```

Commite, grave o relatório em `.superpowers/sdd/task-N-report.md`, e **registre no ledger** em `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` antes de dizer que acabou.

`base` existe porque a revisão precisa do commit **anterior ao início** da tarefa. `HEAD~1` descarta em silêncio tudo menos o último commit de uma tarefa com vários commits, e a revisão passa a olhar meio diff sem avisar.

## `mutate.ps1` tem código de saída invertido, de propósito

- `0` = o teste **reprovou** sob mutação → **a regra está verificada**. É o que você quer.
- `1` = o teste **passou** → a regra está escrita, não verificada. **A tarefa não está pronta.**
- `2` = inconclusivo: âncora ambígua, ou a mutação quebrou o build. Falha de compilação não é cobertura.

Onde o brief pedir prova de mutação, rode o comando e **cole a saída que ele imprimiu**. Prova escrita no condicional — "se removermos X, o teste falha" — não conta. Duas dessas apareceram neste projeto e uma estava factualmente errada.

Alguns briefs dizem que `mutate.ps1` não serve para eles, porque o alvo é script PowerShell, workflow de CI ou analisador, não teste Go. Nesses, a prova é a remoção descrita no próprio brief, com a saída colada. Não force a ferramenta errada.

## Proibições absolutas

- **Nunca** `git checkout`, `git restore`, `git stash`, `git clean`, `git reset`. Há sempre trabalho não commitado nesta árvore, e um agente já destruiu trabalho assim aqui.
- **Nunca** `go mod tidy`. Várias dependências estão fixadas sem importador de propósito; `tidy` removeria elas junto com o pin do SDK MCP, que é decisão fechada (PRD D6).
- **Nunca** `| tail` antes de `&&` num gate. `tail` não propaga a falha, e um commit já entrou sobre gate vermelho por causa disso.
- **Nunca** `git add -A`. Adicione por caminho explícito. `git add -A` já varreu trabalho em curso de outra pessoa para dentro de um commit aqui.
- Nunca pule hooks (`--no-verify`) nem assinatura.

## Regras do código que não são negociáveis

- **stdout pertence ao JSON-RPC.** Todo log vai para stderr via `log/slog`. `fmt.Println` em código alcançável de `serve` corrompe a sessão — o sintoma é o servidor sumir do host sem erro nenhum. `doctor`, `version` e os subcomandos da Task 75 imprimem em stdout **de propósito**: são comandos de CLI, não servidores. Onde isso aparecer, comente por quê.
- **Nenhum pacote sob `internal/` ou `cmd/` importa `net` ou `net/*`.** `net/http` chega transitivamente pelo SDK — isso é esperado.
- **Nenhum tipo do SDK MCP sai de `internal/mcpsrv`.** `internal/service` fala tipos de domínio.
- **`ctx` onde há espera real.** Função que pode bloquear recebe `ctx` e o respeita. Cálculo em memória não recebe. Quando o parâmetro existir só por consistência de assinatura, nomeie-o `_`.
- **Código de plataforma atrás de build tag, em arquivo separado.** Nunca `if runtime.GOOS ==` dentro de lógica compartilhada.
- **Saída de console em ASCII puro:** `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`. Console PowerShell em CP-850 renderiza o resto como lixo.
- **Sem `helpers.go`, `utils.go`, `common.go`.**
- **Flag booleana ou inteira não distingue "omitida" de "definida com zero".** Use `*bool` e `*int`, ou os companheiros `ReadOnlySet` / `DebounceMSSet`.
- Commits em **Conventional Commits, em inglês**. Docs em **português**.
- Depois de qualquer ferramenta que reescreva um `.md`: `python -c "open('ARQUIVO.md',encoding='utf-8').read()"` para conferir que continua UTF-8 válido.

## O que faz uma tarefa estar pronta

Esta seção existe porque oito tarefas deste projeto foram entregues como concluídas sem estarem.

- **Não escreva número que você não mediu.** Se não mediu, escreva **"não medido"**. Ninguém vai brigar com isso. Alvo não atingido e registrado é informação; alvo não medido apresentado como resultado é ficção com aparência de tabela.
- **Não afirme estado que você não verificou.** O README já declarou "v0.1 publicada" sem tag e sem release.
- **Um teste que não pode falhar é pior que teste ausente.** Antes de dizer que testou: apague a regra, rode, confirme que um teste **nomeia** a falha, restaure. Em 2026-08-01 descobriu-se aqui que o critério de bloqueio do M4 matava o processo filho antes de ele escrever, relatando "0 corrompidas em 1.000 iterações" sem ter escrito um byte.
- **Afirme o valor, não a presença do nome do campo.** Um teste que procura a string `"orphans"` no JSON não pode falhar se o campo aparece com ou sem o parâmetro.
- **Schema que promete e código que ignora é pior que parâmetro ausente.** Ou implemente, ou tire do schema e da documentação. Este defeito apareceu nove vezes aqui.
- **Não deixe sua deliberação no código.** Comentário explica por que o código é assim; raciocínio sobre o que fazer não é comentário.
- **Confira todo SHA que escrever no ledger** com `git cat-file -t`. Uma tarefa foi registrada num SHA que não existe no repositório.
- **O relatório é o entregável, não o resumo dele.** Comando rodado, saída real colada, prova de mutação. "Testes passam" não é evidência; a saída do teste é.

## Escopo não encolhe em silêncio

Se alguma parte da tarefa não der para fazer, entregue o resto inteiro e **diga o que ficou de fora e por quê**. Reduzir escopo é decisão de quem pediu. Responder **BLOCKED** com o motivo é resposta melhor que uma entrega que parece completa.

Se você achar um defeito fora do escopo da tarefa da vez, **registre no relatório e siga**. Não conserte sozinho: isso polui o diff da revisão e some no meio dela.

## Ao invocar subagentes

Se você delegar parte do trabalho:

- Dê ao subagente **o brief da tarefa e este bloco de regras**, não um resumo. O brief é autocontido de propósito; um resumo perde exatamente o que evita os erros conhecidos.
- **Um subagente por tarefa.** Não rode dois em paralelo na mesma árvore: `base` e `review` são por árvore de trabalho, e duas tarefas concorrentes fazem cada revisão enxergar o trabalho da outra.
- Repita as proibições de `git` em todo prompt de subagente. Foi um subagente que destruiu trabalho aqui.
- Exija do subagente a **saída colada** dos comandos, não a afirmação de que rodou.
- Confira o que ele devolveu antes de commitar. Especialmente: rode você mesmo `verify.ps1` e `mutate.ps1`; não aceite o relato deles.

## Ferramentas que reduzem erro

O MCP do gopls está disponível (`go_diagnostics`, `go_file_context`, `go_search`, `go_symbol_references`, `go_package_api`, `go_rename_symbol`). Use `go_diagnostics` depois de cada edição em `.go` — pega erro de tipo antes de você gastar um ciclo de build. Use `go_symbol_references` antes de renomear qualquer coisa.

`golangci-lint` precisa ser **v2.12.2**. Confira com `golangci-lint version` antes de confiar num zero: o `go.mod` declara `go 1.25.0`, e um binário compilado com Go mais antigo recusa o config antes de analisar linha nenhuma.

## Comece

Rode `pwsh -File scripts/sdd.ps1 status`, depois `base 75`, depois `brief 75`. Ao terminar cada tarefa, diga qual acabou, cole o SHA do commit e siga para a próxima da tabela.
