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

**Provas de mutação obrigatórias, com saída colada.** Use `scripts/mutate.ps1`, nunca a mão — ele exige âncora única, restaura em `finally` conferindo por SHA-256, e trata falha de build como inconclusivo em vez de contá-la como cobertura. Saída `0` é o que você quer; saída `1` significa que a regra está escrita e não verificada.

```bash
# 1. o gate de classe: sem ele, o anexo passa a ser lido e correlacionado
pwsh -File scripts/mutate.ps1 -Path internal/watcher/rename.go `
  -Anchor 'vault.Classify(p) == vault.ClassNote' -Replacement 'true' `
  -Test TestCorrelateRenames_AssetIsNeverCorrelated -Package ./internal/watcher/

# 3. os candidatos de backlink
pwsh -File scripts/mutate.ps1 -Path internal/watcher/rename.go `
  -Anchor 'backlinks := idx.Backlinks(oldPath)' -Replacement 'var backlinks []index.Backlink' `
  -Test TestCorrelateRenames_ReportsBacklinkCandidates -Package ./internal/watcher/
```

A prova 2 (duplicata) não tem mutação de uma linha só, porque o defeito era estrutural — dois laços de leitura. Prove reintroduzindo o segundo laço à mão, confirmando que `TestCorrelateRenames_NoDuplicateOutput` reprova, e desfazendo.

#### O código dos dois testes que esta tarefa precisa e que não são óbvios

Transcreva. Os outros quatro da lista acima são diretos; estes dois têm uma sutileza cada.

**O teste do anexo não instrumenta leitura — ele afere comportamento.** Não existe seam para contar `os.ReadFile` sem inventar um, e inventar um só para o teste é maquinário que ninguém mais usa. Em vez disso: dê ao anexo **exatamente os mesmos bytes** de uma nota removida. Se o código ler o anexo, ele vai hashear, vai casar 1-para-1, e vai correlacionar. Se o gate de classe funcionar, ele nunca é aberto e a correlação não acontece. O comportamento observável denuncia a leitura.

```go
func TestCorrelateRenames_AssetIsNeverCorrelated(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Bytes identicos nos dois lados. Se o anexo for lido, o hash casa e a
	// correlacao acontece — que e exatamente o defeito que este teste pega.
	conteudo := []byte("# Nota\n\nconteudo qualquer\n")
	if err := os.WriteFile(filepath.Join(tmp, "origem.md"), conteudo, 0644); err != nil {
		t.Fatal(err)
	}
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}

	// origem.md sai do disco; diagrama.png entra com os MESMOS bytes.
	if err := os.Remove(filepath.Join(tmp, "origem.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "diagrama.png"), conteudo, 0644); err != nil {
		t.Fatal(err)
	}

	batch := []vault.CanonicalPath{"origem.md", "diagrama.png"}
	renames, nonRenames := watcher.CorrelateRenames(context.Background(), batch, v, idx, log)

	if len(renames) != 0 {
		t.Fatalf("anexo foi correlacionado como rename: %+v — o gate de classe nao "+
			"impediu a leitura, e um .png de 40 MB tocado pelo OneDrive seria lido "+
			"inteiro dentro do laco que serializa as escritas do indice", renames)
	}
	if len(nonRenames) != 2 {
		t.Errorf("nonRenames = %v, quer os dois caminhos exatamente uma vez", nonRenames)
	}
}
```

**Arquivo somente-nuvem não é testável nesta máquina.** `vault.IsCloudOnly` lê um atributo que só o cliente de sincronização produz, e forjá-lo num `t.TempDir()` testaria o forjador. **Não invente fixture para ele e não marque a verificação como coberta.** No relatório, escreva que o gate de nuvem foi verificado por inspeção de código e pela mutação do gate de classe (que compartilha a mesma condição), e que teste automatizado depende de um cofre OneDrive real — que é um item para o M6, não para esta tarefa.

**O teste de duplicata precisa de arquivo vazio nos dois lados**, porque era exatamente ali que os dois laços se sobrepunham:

```go
func TestCorrelateRenames_NoDuplicateOutput(t *testing.T) {
	tmp := t.TempDir()
	v, err := vault.New(tmp)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}
	idx := index.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Nota vazia indexada, depois removida: Hash != 0 mas Size == 0.
	if err := os.WriteFile(filepath.Join(tmp, "vazia.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("idx.Build: %v", err)
	}
	if err := os.Remove(filepath.Join(tmp, "vazia.md")); err != nil {
		t.Fatal(err)
	}
	// Nota vazia nova: len(data) == 0.
	if err := os.WriteFile(filepath.Join(tmp, "nova.md"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	batch := []vault.CanonicalPath{"vazia.md", "nova.md"}
	renames, nonRenames := watcher.CorrelateRenames(context.Background(), batch, v, idx, log)

	if len(renames) != 0 {
		t.Errorf("arquivos vazios nao podem correlacionar: %+v", renames)
	}
	// A assercao central: cada caminho aparece UMA vez. Com os dois lacos de
	// leitura da entrega original, isto devolvia 4 entradas, e o Apply chamava
	// Replace/Remove duas vezes por caminho.
	visto := map[vault.CanonicalPath]int{}
	for _, p := range nonRenames {
		visto[p]++
	}
	if len(nonRenames) != 2 {
		t.Fatalf("nonRenames = %v (len %d), quer 2 entradas distintas", nonRenames, len(nonRenames))
	}
	for p, n := range visto {
		if n != 1 {
			t.Errorf("%s aparece %d vezes em nonRenames; duplicata faz Replace rodar em dobro", p, n)
		}
	}
}
```

Os dois testes ficam no pacote `watcher_test` (externo), como o `rename_test.go` atual. Importe `context`, `io`, `log/slog`, `os`, `path/filepath`, `testing`, mais `index`, `vault` e `watcher`.

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

