### Task 75: Subcomandos `index`, `search` e `inspect`

**Onde encaixa.** H3 do M6, e é RF-52 do PRD.

**A regra que decide o desenho desta tarefa, e ela não é negociável:** *stdout pertence ao JSON-RPC.* Todo log vai para stderr via `log/slog`. `fmt.Println` em código alcançável de `serve` corrompe a sessão — o sintoma é o servidor sumir do host sem erro nenhum.

**Mas `index`, `search` e `inspect` são comandos de CLI, não servidores**, e por isso imprimem em stdout **de propósito** — exatamente como `doctor` e `version` já fazem. Essa distinção merece um comentário onde aparecer. Leia `cmd/gobsidian/doctor.go` antes de escrever: ele é o modelo.

#### Passos

1. `gobsidian index --vault <path>` — constrói o índice e imprime o resumo. Reaproveita `index.Build`; não escreva um segundo caminho de indexação.
2. `gobsidian search --vault <path> <consulta>` — imprime os resultados. Reaproveita `service.Search`.
3. `gobsidian inspect --vault <path> <nota>` — imprime metadados, links e backlinks de uma nota.
4. Todos com `--json` para saída estruturada, porque é o que torna um subcomando script-ável.

**A armadilha de flag que já custou caro aqui.** Flag booleana ou inteira **não distingue "omitida" de "definida com zero"**. `config.Flags` tem os companheiros `ReadOnlySet` e `DebounceMSSet`, e **toda** chamada a `config.Load` precisa preenchê-los com `cmd.Flags().Changed(nome)`. Esquecer em um subcomando faz a flag virar no-op silencioso. Se você acrescentar subcomando que chama `config.Load`, preencha os dois.

#### Verificações além dos passos

- Um teste por subcomando, em `cmd/gobsidian`. O pacote ficou **sem teste algum** até a revisão do M0 acusar, então não repita.
- Confirme que os três **não** escrevem em stderr o que deveria ir a stdout, nem o contrário. Teste capturando os dois separadamente.
- Confirme que `--json` produz JSON válido — `json.Unmarshal` no teste, não inspeção visual.
- Saída de console em ASCII puro nos modos não-JSON: `[OK]`, `[*]`, `[!]`, `[i]`, `[...]`.
- Confirme que nenhum subcomando novo quebrou `serve`: rode o gate de órfãos, `pwsh -File scripts/test_orphans.ps1 -Cycles 100`.

#### Prova de mutação obrigatória

Para o preenchimento de `ReadOnlySet`/`DebounceMSSet`: remova-o de um subcomando e confirme que um teste nomeia a falha. Se nenhum nomear, a flag pode virar no-op e ninguém saberá. Cole a saída de `scripts/mutate.ps1`.

#### Regras de execução

Idênticas às da Task 69, com ênfase: **stdout pertence ao JSON-RPC em `serve`**, e o comentário que explica por que estes comandos são exceção precisa estar no código.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-75-report.md`: a saída real de cada subcomando, nos dois modos; a saída do gate de órfãos; a prova de mutação das flags; e o que ficou de fora.

Responda com no máximo 15 linhas.

**Files:** Create `cmd/gobsidian/index.go`, `search.go`, `inspect.go` and tests
**Commit:** `feat(cmd): index, search and inspect subcommands (RF-52)`

---

