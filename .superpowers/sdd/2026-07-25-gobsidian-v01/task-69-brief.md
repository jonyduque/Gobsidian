### Task 69: Os quatro parâmetros de schema que o código ignora

**Onde encaixa.** Fecha, no M6, um defeito que apareceu **nove vezes** neste projeto e nunca foi pego por um gate. `scripts/check_tool_params.ps1` foi escrito em 2026-08-01 justamente para mecanizá-lo, e achou estes quatro na primeira execução.

**A decisão fechada que te vincula (nº 4 do M6):** os quatro são **implementados**, não removidos do schema. Estão documentados em `docs/TOOLS.md` com semântica e default; remover seria encolher contrato publicado.

**O defeito.** Um campo declarado numa struct `*Input` de `internal/mcpsrv` que o handler nunca lê. O modelo do outro lado lê o schema para decidir o que mandar: pede três campos, recebe tudo, e não tem como saber que o pedido não fez nada. Nenhum gate pegava — o compilador de Go não reclama de campo de struct sem uso, e o `golangci-lint` também não, porque o campo **é** usado, pelo decodificador de JSON.

Rode primeiro, para ver o estado de partida:

```bash
pwsh -File scripts/check_tool_params.ps1
```

Saída atual (2026-08-01):

```
[i] 12 structs de entrada, 68 parametros declarados.
[!] 4 parametro(s) declarado(s) e nunca lido(s):
    tools_read.go:253  noteMetadataInput.Include  (json: "include")
    tools_read.go:258  linkGraphInput.Direction  (json: "direction")
    tools_read.go:260  linkGraphInput.IncludeBroken  (json: "include_broken")
    tools_read.go:261  linkGraphInput.IncludeEmbeds  (json: "include_embeds")
```

**O que cada um promete**, conforme `docs/TOOLS.md`:

| Parâmetro | Linha em TOOLS.md | Contrato |
|---|---|---|
| `note_metadata.include` | 141, 151 | "Sempre `path`, `title` e `hash`. Os demais campos **conforme pedido em `include`**." Valores: `headings`, `links`, `backlinks`, `tags`, `frontmatter`, `blocks`. |
| `link_graph.direction` | 183 | enum `outgoing` / `incoming` / `both`, default `both`. |
| `link_graph.include_broken` | 185 | boolean, default `true`. Link não resolvido entra como nó? |
| `link_graph.include_embeds` | 186 | boolean, default `true`. Embed conta como aresta? |

#### Passos

1. Leia `docs/TOOLS.md` linhas 130–200 inteiras antes de escrever código. O contrato é o que está lá, não o que parecer razoável.
2. `internal/service/graph.go`: `GraphRequest` ganha `Direction string`, `IncludeBroken bool`, `IncludeEmbeds bool`. `LinkGraph` os honra. Atenção: `GraphEdge` hoje já não emite link externo nem link quebrado — leia o comentário em `graph.go:27` antes de decidir o que `include_broken` liga.
3. `internal/service`: a requisição de metadados ganha o conjunto pedido. `path`, `title` e `hash` vêm sempre; o resto só se pedido.
4. `internal/mcpsrv/tools_read.go`: os handlers passam os campos adiante, aplicando os defaults documentados. **`*bool` para os booleanos** — `bool` simples não distingue "omitido" de "false", e foi exatamente esse o defeito de `include_health`.
5. Ligue o checador ao gate: em `scripts/verify.ps1`, uma etapa nova `check_tool_params` chamando `scripts/check_tool_params.ps1`. Ela só pode entrar depois de os quatro estarem implementados, senão o gate nasce vermelho.

#### Verificações além dos passos

- Para **cada** um dos quatro, um teste que chama a tool **sem** o parâmetro e afirma o comportamento **default documentado**, e outro que o passa explicitamente com o valor não-default e afirma que a resposta mudou. O modelo é `internal/mcpsrv/defaults_test.go`; leia-o antes.
- **Afirme o valor, não a presença do nome do campo.** A primeira versão do teste de `include_health` procurava a string `"orphans"` no JSON — e não podia falhar, porque o campo aparecia com ou sem o parâmetro. Só passou a valer quando afirmou `orphans == 1`.
- `pwsh -File scripts/check_tool_params.ps1` tem de sair `0`.

#### Prova de mutação obrigatória

Para cada um dos quatro, remova a leitura do campo no handler (volte ao estado de hoje) e confirme que um teste **nomeia** a falha:

```bash
pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/tools_read.go -Anchor '<a linha que lê o campo>' -Replacement '<a linha sem ler>' -Test <NomeDoTeste> -Package ./internal/mcpsrv/
```

`0` = o teste reprovou sob mutação (regra verificada). `1` = o teste passou (regra escrita, não verificada) e a tarefa **não está pronta**. `2` = inconclusivo.

#### Regras de execução

- Rode `pwsh -File scripts/verify.ps1` e só considere pronto com as etapas todas verdes. Não use `| tail` antes de `&&`: `tail` não propaga a falha, e um commit já entrou sobre gate vermelho por causa disso.
- Nunca `git checkout`, `git restore`, `git stash`, `git clean`, `git reset`. Há sempre trabalho não commitado.
- Nunca `go mod tidy`.
- Nenhum tipo do SDK MCP sai de `internal/mcpsrv`. `internal/service` fala tipos de domínio.
- Commits em Conventional Commits, em inglês.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-69-report.md`: a saída de `check_tool_params.ps1` antes e depois; para cada um dos quatro parâmetros, o teste que o cobre e a saída colada de `mutate.ps1`; o diff de `verify.ps1`; e o que ficou de fora. **A saída real colada, não "os testes passam".**

Responda com no máximo 15 linhas.

**Files:** Modify `internal/mcpsrv/tools_read.go`, `internal/service/graph.go`, `internal/service/metadata.go`, `scripts/verify.ps1`; create tests
**Commit:** `fix(mcpsrv,service): honour the four schema params the handlers dropped`

---

