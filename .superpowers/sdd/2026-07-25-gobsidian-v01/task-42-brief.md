### Task 42: Relatórios e ledger — fechar o M2.1 com o que aconteceu de verdade

Última do bloco. Só roda depois das Tasks 33–41 fechadas.

#### Onde isto encaixa

O ledger é o que a próxima sessão tem no lugar do contexto desta. Ele está com um SHA que não existe e com cinco tarefas marcadas `Approved` sem revisão. Quatro relatórios violam o contrato de entrega deste projeto, e um deles afirma como medida uma coisa que a revisão mediu e encontrou falsa.

#### O que corrigir, item a item

**1. `.superpowers/sdd/task-29-report.md`** — tem 1.148 bytes, nenhum RED, nenhum GREEN, e responde "`os.Stat` falhando com permissão negada" com *"covered implicitly by the stat checks"*. Coberto implicitamente é o mesmo que não coberto. Reescreva com a evidência que existir hoje, depois das Tasks 33–41.

**2. `.superpowers/sdd/task-30-report.md`** — a prova de mutação é hipotética e **falsa**; a nota de correção já foi acrescentada na Fase 1. Substitua a seção de mutação pela prova real que a Task 34 produziu.

**3. `.superpowers/sdd/task-31-report.md`** — prova de mutação hipotética (*"Se alterarmos..."*), e a garantia central da tarefa — nenhum arquivo do cofre escrito — foi respondida com *"o teste e o código usam apenas `idx.MoveNote` e `idx.Replace`"*, que é argumento, não medição. Substitua pela prova da Task 33.

**4. `.superpowers/sdd/task-32-report.md`** — tem cinco provas de mutação legítimas com saída real colada, e **só** isso: sem o que foi implementado, sem RED/GREEN, sem tabela de verificações, sem arquivos alterados, sem preocupações. Complete as seções que faltam.

**5. O ledger**, em `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`:
- A Task 31 está registrada como commit `14210ee`. Esse objeto **não existe** — `git cat-file -t 14210ee` devolve `fatal: Not a valid object name`. O commit real é `d5d1bf0`. Corrija.
- As Tasks 27–32 estão marcadas `review Approved` sem que exista pacote de revisão para 27, 28, 30, 31 ou 32. Marque-as como revisadas em 2026-07-28 com o resultado real, e registre as Tasks 33–42 conforme forem fechando.
- Acrescente à seção de lacunas carregadas: **reconciliação por overflow não existe em darwin/BSD**, porque o backend kqueue do `fsnotify` v1.10.1 nunca emite `ErrEventOverflow`.

#### A regra que governa esta tarefa

**Não escreva número que você não mediu, e não afirme estado que você não verificou.** Um relatório que resume em vez de mostrar é um relatório escrito sem rodar. Toda prova de mutação tem o formato: *o que mutei / qual teste reprovou, por nome e linha / confirmação de que restaurei*, com a saída real do `go test` colada. Onde não houve medição, escreva **"não medido"** — ninguém vai brigar com isso. O que não pode é hedge (*tende a*, *aproximadamente*, *e.g.*, *deveria*) ao lado de algo apresentado como resultado.

#### Verificações além dos passos

- `pwsh -File scripts/sdd.ps1 status` reflete a realidade? Cole a saída.
- `git cat-file -t <sha>` para **cada** SHA citado no ledger. Cole a saída. Nenhum pode falhar.
- `grep -niE "tende a|aproximadamente|e\.g\.|deveria|should be|implicitly" .superpowers/sdd/task-*-report.md` — cole a saída e justifique cada ocorrência que sobrar.
- Cada relatório tem as seis seções do contrato (status, commit, TDD, mutação, verificações, o que ficou de fora)? Confira um por um.

#### Regras de execução

- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.**
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.** E ponha `assert` do texto-âncora antes de substituir: `str.replace` que não casa não falha, segue em silêncio — duas edições de plano já "deram certo" sem editar nada neste projeto.
- Depois de qualquer ferramenta que reescreva um `.md`, confira a codificação: `python -c "open('ARQUIVO',encoding='utf-8').read()" && echo "[OK] UTF-8 valido"`.
- Commits em Conventional Commits, em inglês.

#### Contrato de relatório

Não há relatório separado desta tarefa: o entregável **são** os relatórios corrigidos e o ledger. Responda com no máximo 15 linhas listando o que mudou em cada arquivo e colando a saída de `sdd.ps1 status`.

**Files:**
- Modify: `.superpowers/sdd/task-29-report.md`, `task-30-report.md`, `task-31-report.md`, `task-32-report.md`
- Modify: `.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`

**Commit:** `docs(sdd): rewrite reports with real evidence; fix ledger`

---

### Portão de saída do M2.1

Nada deste bloco está pronto até estes comandos terem rodado e a saída ter sido **lida**:

```bash
pwsh -File scripts/verify.ps1          # build, -race, vet x3, gofmt, check_net
golangci-lint version                  # exige v2.12.2; v1.x nem carrega o config
golangci-lint run ./...
pwsh -File scripts/build.ps1
pwsh -File scripts/test_orphans.ps1 -Cycles 100
pwsh -File scripts/sdd.ps1 review 33   # diff desde a base gravada antes da Task 33
```

O gate de órfãos precisa de **zero órfãos em 100 ciclos**. O watcher segura handles de diretório, e é exatamente o tipo de recurso que impede encerramento limpo — uma rodada verde antes do M2.1 não diz nada sobre uma rodada depois dele. Se falhar, esse é o resultado mais valioso disponível: diagnostique com `--log-level debug` e reporte qual mecanismo disparou ou deixou de disparar. **Nunca enfraqueça o teste para fazê-lo passar.**

Depois do pacote de revisão, uma re-revisão do diff inteiro 33→42 pelo modelo principal, com foco nos três Critical — não em ler o código, mas em **rodar a mutação de cada um** e confirmar que agora um teste nomeia a falha.

---

## M3 a M6 — escopo em altitude de tarefa

Cada marco recebe seu próprio plano detalhado quando chegar a vez, escrito contra o código que existir então. Planejá-los agora em nível de passo produziria detalhe que envelhece antes de ser usado.

### M2 — Watcher

**Detalhado acima, como Tasks 27–32.** Deixou a altitude de tarefa porque chegou a vez: o plano por passo agora é escrito contra o código que existe, que era a condição para escrevê-lo.

### M3 — Busca (1 semana)

| Tarefa | Entregável | Critério |
|---|---|---|
| S1 | `search/analyzer.go` | Indexação dupla: forma crua e reduzida; teste com corpus jurídico real |
| S2 | `search/inverted.go` | Dicionário de termos e posting lists |
| S3 | `search/bm25.go` | k1=1.2, b=0.75; pesos título 3x, headings 2x, corpo 1x |
| S4 | `search/snippet.go` | Trecho com destaque, `snippet_chars` respeitado |
| S5 | Tool `vault_search` | Filtros combináveis; query vazia redireciona para o caminho de metadados |
| S6 | `index/cache.go` e `search/persist.go` | Cabeçalho com `format_version`, `parser_version`, `analyzer_version`; RNF-02 medido |

Q3 do PRD §11 fecha aqui: medir o custo real de reconstruir a busca a partir do índice de metadados já carregado decide se o índice invertido também vai para o cache.

### M4 — Escrita (2 semanas)

| Tarefa | Entregável | Critério |
|---|---|---|
| E1 | `writer/lock.go` | Mutex por caminho; teste com escritas concorrentes na mesma nota |
| E2 | `writer/atomic.go` | Temporário no mesmo diretório, `Sync`, rename com retry; teste de crash injetado, 1.000 iterações |
| E3 | `writer/diff.go` | Myers sobre linhas, ~150 linhas, sem dependência |
| E4 | `writer/section.go` | `note_append` e `note_patch` por heading; EOL e BOM preservados |
| E5 | `writer/block.go` | `replace_block` por `^id` |
| E6 | Tools `note_create`, `note_append`, `note_patch` | `dry_run` e `expected_hash` em todas |
| E7 | `--read-only` verificado | Tools de escrita ausentes de `ListTools`, não apenas rejeitadas |

RNF-11 é o critério de bloqueio deste marco: zero notas corrompidas em 1.000 iterações de crash injetado. É a única parte do produto que pode destruir dados, e vem depois de tudo por isso.

### M5 — Refatoração do cofre (1 semana)

| Tarefa | Entregável | Critério |
|---|---|---|
| R1 | `writer/linkrewrite.go` | Alias, âncora e forma original preservados; `Link.Raw` é o insumo |
| R2 | Tool `note_move` | Falha parcial reporta com precisão o que foi aplicado |
| R3 | Tool `note_delete` | `to_trash` padrão; relatório prévio de links que quebrarão |
| R4 | Âncoras quebradas em `vault_stats` | Já implementado em M1; exposto aqui no relatório de impacto |

### M6 — Endurecimento (1 semana) — v1.0

| Tarefa | Entregável | Critério |
|---|---|---|
| H1 | `bench.yml` no CI | RNF-01 a RNF-08 medidos; regressão acima de 20% falha o build |
| H2 | `gen_vault.ps1` determinístico | Cofre sintético de 5.000 notas reproduzível |
| H3 | Subcomandos `index`, `search`, `inspect` | RF-52 |
| H4 | `go vet -vettool` com `netcheck` no CI | Segunda parte da garantia de RNF-30 |
| H5 | Release v1.0.0 | Binários para os três sistemas |

---

## Auto-revisão do plano

**Cobertura da especificação.** Requisitos do PRD mapeados a tarefas: RF-01, RF-02 e RF-08 → Task 19; RF-03 a RF-05 → M2; RF-06 → M3/S6; RF-07 → M2 (o parsing de `.gitignore` entra com o watcher, que é quem precisa dele); RF-10 a RF-17 → Tasks 12–18; RF-18 → Task 17; RF-19 (callouts, P2) → pós-1.0, não planejado; RF-20 a RF-25 → M3; RF-26 (P2) → Q1 em aberto; RF-30 a RF-38 → M4; RF-35 e RF-36 → M5; RF-40 a RF-44 → Tasks 3–6 e 11; RF-50 e RF-51 → Tasks 9 e 10; RF-52 → M6/H3; RF-53 → Global Constraints e Task 9; RF-54 (P2) → fora da v1 por D7; RF-55 → Tasks 9 e M4/E7; RF-60 a RF-63 → Tasks 8, 19 e 20. RNF-01 e RNF-07 → Task 26/Step 2; RNF-02 a RNF-06, RNF-08, RNF-09 → M6/H1; RNF-10 → Task 11; RNF-11 → M4/E2; RNF-12 → Tasks 19 e 21; RNF-13 → Task 9; RNF-20 a RNF-23 → Tasks 7, 8 e 10; RNF-24 → Task 9; RNF-30 → Task 1 e M6/H4; RNF-31 e RNF-32 → Task 7; RNF-33 → Tasks 9 e M4/E7.

**Lacuna conhecida.** RNF-32 (links simbólicos que apontem para fora do cofre não são seguidos) tem teste apenas indireto na Task 7, porque criar symlink no Windows exige privilégio elevado e o teste falharia em máquina de desenvolvimento comum. `vault.Walk` usa `filepath.WalkDir`, que não segue symlinks por padrão — a propriedade vale por construção. Adicione um teste explícito em M6, marcado com `t.Skip` quando a criação de symlink falhar por permissão.

**Consistência de tipos.** `vault.CanonicalPath` é o tipo de caminho em toda a cadeia — `vault`, `index`, `service` — e nunca vira `string` solto. `parser.Link` é embutido em `index.ResolvedLink`, e não duplicado. `Note.Hash` é `uint64` no índice e string hexadecimal no contrato MCP; a conversão acontece em `service/read.go` e em nenhum outro lugar. `service.Index` é declarado como interface na Task 9 e satisfeito por `*index.Index` na Task 23, o que é o que permite testar o serviço sem construir um índice completo.

**Sem marcadores pendentes.** Nenhum passo contém "TBD", "implementar depois" ou "similar à tarefa N". As Tasks 15 a 17, 20 a 24 descrevem regras em prosa numerada em vez de código completo por escolha: são extensões cujo formato já foi estabelecido integralmente na Task 14, e as regras são o que o implementador precisa saber que os testes não dizem sozinhos. Todo teste está escrito por inteiro.

---

## Referências

| Documento | Papel |
|---|---|
| `docs/PRD.md` | Requisitos, prioridades, decisões fechadas (D1–D13), questões em aberto |
| `docs/ARCHITECTURE.md` | Camadas, fluxos, modelo de dados, decisões arquiteturais (AD-01–AD-09) |
| `docs/ESTRUTURA.md` | Árvore de diretórios, convenções de código, build |
| `docs/TOOLS.md` | Contrato de cada tool: schemas, retornos, códigos de erro |
| `docs/WINDOWS.md` | OneDrive, MAX_PATH, casing, fsnotify, registro no Claude Desktop |
