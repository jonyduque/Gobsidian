### Task 33: Correlação de rename — não abrir o que não é nota local, e não duplicar saída

Reescrita de `internal/watcher/rename.go` em passagem única. Corrige um Critical e dois Important da revisão.

#### Onde isto encaixa

`CorrelateRenames` roda no começo de cada lote do `Apply`, antes de qualquer `Replace` ou `Remove`. É o primeiro código do pipeline que **abre arquivo**, e por isso é o ponto onde as regras de "não abrir" precisam valer. Hoje ele fura duas delas e devolve caminhos duplicados que fazem o resto do lote trabalhar em dobro.

#### A evidência medida do defeito

**Critical — anexos e arquivos somente-nuvem são lidos.** `internal/watcher/rename.go:58` chama `v.ReadAll(context.Background(), p)` para **todo** caminho do lote que exista no disco. Não há filtro de classe nem de nuvem antes. O lote vem do debouncer, que vem do `filter`, que deixa passar `ClassNote` **e** `ClassAsset`. E `vault.ReadAll` (`internal/vault/vault.go:162-171`) é um `os.ReadFile` puro, cujo próprio comentário diz:

```go
// ReadAll le a nota inteira. Recebe ctx porque leitura de arquivo bloqueia
// de verdade — em cofre sincronizado na nuvem, indefinidamente enquanto o
// cliente hidrata o placeholder.
```

Consequência concreta: num cofre OneDrive com anexos grandes, cada janela de debounce lê para memória todo `.png` tocado pelo sincronizador, e hidrata todo placeholder não baixado, dentro da goroutine que serializa as escritas do índice. `index.Replace` respeita as duas regras (`internal/index/update.go:47` e `:50` desviam para o ramo de anexo sem ler); `CorrelateRenames` roda antes dele e as fura.

**Important — caminho duplicado na saída.** Medido com uma sonda: um único `empty.md` criado sai como `nonRenames = [empty.md empty.md]`. Causa: o passo 3 (`rename.go:59`) manda o arquivo vazio para `nonRenames` pelo gate `len(data) > 0`, e o passo 5 (`rename.go:113-127`) recalcula e manda de novo. O mesmo acontece em `missing` quando `n.Size == 0 && n.Hash != 0` (`rename.go:47` versus `:106`). Os blocos `if` vazios com comentário `// Já foi adicionado acima` são a marca do defeito. Efeito: `Replace`/`Remove` rodam duas vezes no mesmo caminho e o contador `processed` infla.

**Important — `ctx` descartado.** `rename.go:58` e `:115` usam `context.Background()` para uma chamada que bloqueia. O `ctx` do laço do `Apply` está disponível e não é repassado.

#### O que já está fechado e vincula esta tarefa

- **Anexo é indexado por nome, nunca lido** (PRD RF-60, repetido na Task 29). **Arquivo somente-nuvem não é aberto**, porque abrir dispara download síncrono; `vault.IsCloudOnly` existe e lê o atributo sem abrir.
- **`ctx` onde há espera real, e respeitado de verdade.** Leitura de arquivo bloqueia. `ctx` que nenhum corpo verifica ensina revisor a ignorar `ctx`.
- **Rename é reportado, nunca aplicado ao cofre.** Esta tarefa não escreve arquivo nenhum.
- **A regra de cardinalidade não muda:** correlaciona apenas quando há exatamente uma remoção e exatamente uma criação com aquele hash na janela. Qualquer outra cardinalidade é `log.Debug` de recusa por ambiguidade, com o hash e as duas contagens.
- **O `xxhash` é calculado sobre os bytes crus, antes de `vault.StripBOM`.** Confirmado em `internal/index/update.go:84` e `internal/index/build.go:88`. Não mude o lado do cálculo.

#### O que implementar

Reescreva `CorrelateRenames` em **uma** passagem. Assinatura nova:

```go
func CorrelateRenames(ctx context.Context, batch []vault.CanonicalPath, v *vault.Vault, idx *index.Index, log *slog.Logger) (renames []RenameCandidate, nonRenames []vault.CanonicalPath)
```

Estrutura obrigatória:

1. Um tipo local `candidate struct { path vault.CanonicalPath; hash uint64; eligible bool }`.
2. **Um** laço sobre `batch`. Para cada caminho:
   - `os.Stat(v.Abs(p))`.
   - Se o erro é `IsNotExist`: buscar `idx.Get(p)`; elegível apenas se `ok && n.Hash != 0 && n.Size > 0`. O hash vem do índice, não do disco — o arquivo já não está lá.
   - Se o erro **não** é `IsNotExist` (permissão, share caído): **não** elegível, e nunca entra na correlação.
   - Se existe no disco: elegível apenas se `vault.Classify(p) == vault.ClassNote` **e** `!vault.IsCloudOnly(v.Abs(p))`. Só então `v.ReadAll(ctx, p)`, **uma vez**, guardando o hash calculado. Elegível apenas se `err == nil && len(data) > 0`.
3. Montar `missingHashes` e `addedHashes` apenas com elegíveis.
4. Correlacionar por cardinalidade exata 1-para-1; qualquer outra cardinalidade recusa com `log.Debug`.
5. **Uma única** varredura final monta `nonRenames`: todo caminho do lote que não entrou em nenhum `RenameCandidate`, elegível ou não. Nenhum caminho é lido de novo e nenhum aparece duas vezes — a garantia vem da estrutura, não de um `if` de guarda.

Em `internal/watcher/apply.go`, passe o `ctx` do laço para `CorrelateRenames`. **`context.Background()` não pode sobrar em lugar nenhum do pacote `watcher`** — confirme com `grep -rn "context.Background()" internal/watcher/ --include=*.go | grep -v _test`.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Um teste que não pode falhar é pior que teste ausente.** Um teste de correlação que só afirma o par `From`/`To` passa sem `Backlinks`, sem gate de nuvem e com duplicata na saída.
- **Perguntar o que um valor zero significa.** `Hash` zero é valor legítimo de `xxhash` e também zero-value de `uint64`; `Size` zero é arquivo vazio legítimo. Os gates existentes (`Hash != 0`, `Size > 0`) tratam disso e não devem ser removidos na reescrita.
- **Feature P1 não tem direito de apagar dado P0.** A reescrita não pode reduzir o que já era correlacionado. Compare a saída de `CorrelateRenames` antes e depois num lote de rename simples de nota não vazia.

#### Verificações além dos passos

Faça e **reporte o resultado real de cada uma**, inclusive quando estiver correta:

- `TestCorrelateRenames_NeverReadsAssetsOrCloudOnly` — lote com um `.png` e um arquivo marcado como somente-nuvem; afirmar **zero** leituras deles. Instrumente contando aberturas (arquivo sentinela, wrapper, ou comparação de tempo de acesso — diga qual usou).
- `TestCorrelateRenames_NoDuplicateOutput` — lote com uma nota vazia criada e uma nota vazia removida; afirmar `len(nonRenames) == 2` e elementos distintos. Com o código anterior esse teste devolvia 4.
- `TestCorrelateRenames_SingleReadPerPath` — afirmar que o número de leituras é igual ao número de notas locais elegíveis do lote, não ao dobro.
- `TestCorrelateRenames_WithBOM` — nota gravada com `\xEF\xBB\xBF` na frente, indexada, renomeada; afirmar que correlaciona. É o teste que pega hash calculado sobre os bytes errados.
- `TestCorrelateRenames_ReportsBacklinkCandidates` — `c.md` com `[[origem]]`; renomear `origem.md`; afirmar `len(renames[0].Backlinks) == 1` e `Backlinks[0].From == "c.md"`.
- `TestCorrelateRenames_DoesNotWriteVault` — `filepath.WalkDir` do cofre antes e depois montando `map[path]{mtime,size}`; afirmar igualdade dos dois mapas. É a garantia central da tarefa.
- Dois arquivos vazios removidos e criados na mesma janela continuam **não** correlacionados?
- Uma cópia seguida da remoção do original continua sendo correlacionada como rename? (É o comportamento documentado em `ARCHITECTURE.md` §5.3, limitação 1. Não mude, só confirme.)

**Provas de mutação obrigatórias, com saída colada:**
1. Remover o gate `!vault.IsCloudOnly(...)` → `TestCorrelateRenames_NeverReadsAssetsOrCloudOnly` reprova.
2. Reintroduzir o segundo laço de leitura → `TestCorrelateRenames_NoDuplicateOutput` reprova.
3. Apagar a linha `backlinks := idx.Backlinks(oldPath)` → `TestCorrelateRenames_ReportsBacklinkCandidates` reprova.

#### Regras de execução

Valem para toda tarefa deste plano e não são negociáveis.

- **O plano é a fonte.** Transcreva o desenho desta seção; não improvise uma variante. Se um teste falhar por motivo que a seção não explica, **pare e reporte** — não ajuste a expectativa para o código passar.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.**
- **`go mod tidy` está proibido.**
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.**
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês. Sem `helpers.go`, `utils.go` ou `common.go`.

#### Contrato de relatório

Grave em `.superpowers/sdd/task-33-report.md`: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); as **três** provas de mutação com saída real colada; a tabela de verificações com o resultado real de cada uma; a saída do `grep` de `context.Background()`; arquivos alterados; achados da auto-revisão; e preocupações.

**Não escreva número que você não mediu.** Se não mediu, escreva "não medido".

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, e preocupações.

**Files:**
- Modify: `internal/watcher/rename.go` (reescrita), `internal/watcher/rename_test.go`
- Modify: `internal/watcher/apply.go` (repassar `ctx` na chamada)

**Interfaces:**
- Consumes: `vault.Classify`, `vault.IsCloudOnly`, `vault.ReadAll`, `index.Note.Hash`, `index.Backlinks`, `xxhash`
- Produces: `CorrelateRenames(ctx, batch, v, idx, log)` com saída sem duplicatas

**Commit:** `fix(watcher): never open assets or cloud-only files to correlate a rename`

---

