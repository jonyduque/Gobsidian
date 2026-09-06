# Task 147 — Falha de `ReadDir` na raiz varrida é falha da raiz

## Status

`DONE_WITH_CONCERNS`

Duas ressalvas, ambas de escopo/instrumento, nenhuma de comportamento:

1. O brief chama `internal/watcher/varredura_raiz_test.go` de arquivo **novo**.
   Ele **já existe e é versionado** (commit `499e744`), com dois testes e sem
   build tag. Escrever o conteúdo do brief nele apagaria trabalho existente e —
   por causa do `//go:build !windows` que o brief pede — desligaria esses dois
   testes no Windows. O teste do brief foi para um arquivo irmão,
   `internal/watcher/varredura_raiz_ilegivel_test.go`, com o mesmo build tag e o
   mesmo corpo. Detalhe em "O que ficou de fora".
2. A prova de mutação do consumidor no **watcher** não é obtenível nesta
   máquina: o teste que ela exige é `//go:build !windows`. O comando do brief
   foi rodado assim mesmo e a saída real está colada abaixo (`EXIT=1`, com
   `go test` dizendo `[no tests to run]`). O brief prevê e prescreve o
   substituto — mutação do call site de `Walk` —, que rodou com `EXIT=0`.

## Commit

```
3dfcf8e fix(vault): a ReadDir failure on the scanned root is a root failure, not an unreadable entry
```

Cinco arquivos, exatamente os do brief a menos do arquivo renomeado na ressalva 1:

```
 internal/vault/walk.go                           | 30 +++++++++++---
 internal/vault/walk_raiz_test.go                 | 36 ++++++++++++++++
 internal/vault/walk_raiz_windows_test.go         | 52 ++++++++++++++++++++++++
 internal/watcher/varredura_raiz_ilegivel_test.go | 48 ++++++++++++++++++++++
 internal/watcher/watcher.go                      |  9 ++--
 5 files changed, 167 insertions(+), 8 deletions(-)
```

`go.mod` e `go.sum` **não** mudaram (`golang.org/x/sys v0.47.0` já estava no
`go.sum`), então não entraram no `git add`.

Nota de processo: o commit foi criado uma vez com o assunto corrompido — usei a
sintaxe de here-string do PowerShell (`@'...'@`) dentro da ferramenta **Bash**,
que não a interpreta, e a primeira linha da mensagem virou `@`. Corrigido com
`git commit --amend -F -` e um heredoc de verdade; só a mensagem mudou, o
conteúdo do commit é o mesmo. Nenhum comando proibido foi usado (`--amend` não
está na lista; `checkout`/`restore`/`stash`/`clean`/`reset` não foram tocados).

## Evidência de TDD

### RED 1 — símbolo ausente (Step 2)

`go test ./internal/vault -run TestFalhaNaRaiz`

```
# github.com/jonyd/gobsidian/internal/vault [github.com/jonyd/gobsidian/internal/vault.test]
internal\vault\walk_raiz_test.go:31:14: undefined: FalhaNaRaiz
FAIL	github.com/jonyd/gobsidian/internal/vault [build failed]
FAIL
```

### RED 2 — o teste do Windows reprova ANTES do fix (Step 7)

Para que este RED existisse, `FalhaNaRaiz` foi escrita **sozinha** (Step 3,
primeira metade) e os dois call sites ficaram como estavam. Sem isso o pacote não
compila e o teste do Windows não chega a rodar — o RED seria de build, não do
defeito. Este é o estado exato de antes do fix, com o pacote compilando.

`go test ./internal/vault ./internal/watcher -run 'FalhaNaRaiz|RaizQueExiste|RaizIlegivel' -v`

```
=== RUN   TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir
=== RUN   TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/Lstat_da_raiz_falhou:_d_==_nil
=== RUN   TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/ReadDir_da_raiz_falhou:_d_!=_nil,_caminho_==_raiz
=== RUN   TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/entrada_comum_ilegivel
--- PASS: TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir (0.00s)
    --- PASS: TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/Lstat_da_raiz_falhou:_d_==_nil (0.00s)
    --- PASS: TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/ReadDir_da_raiz_falhou:_d_!=_nil,_caminho_==_raiz (0.00s)
    --- PASS: TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/entrada_comum_ilegivel (0.00s)
=== RUN   TestWalkNaoEngoleRaizQueExisteMasNaoLe
    walk_raiz_windows_test.go:50: Walk devolveu nil com 0 entradas para uma raiz que ReadDir nao le: cofre inacessivel virou cofre vazio
--- FAIL: TestWalkNaoEngoleRaizQueExisteMasNaoLe (0.00s)
FAIL
FAIL	github.com/jonyd/gobsidian/internal/vault	0.732s
testing: warning: no tests to run
PASS
ok  	github.com/jonyd/gobsidian/internal/watcher	0.761s [no tests to run]
FAIL
```

**O cenário do handle exclusivo REPRODUZ nesta máquina.** Não houve `SKIP`: o
`CreateFile(..., dwShareMode=0, FILE_FLAG_BACKUP_SEMANTICS)` impediu o `ReadDir`,
o `Lstat` passou, e `Walk` devolveu `nil` com **0 entradas** — cofre inacessível
virou cofre vazio, exatamente a evidência que o brief descreve. A Task **não**
está `BLOCKED` por esse motivo.

### GREEN — depois dos Steps 3 e 4

Mesmo comando:

```
=== RUN   TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir
=== RUN   TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/Lstat_da_raiz_falhou:_d_==_nil
=== RUN   TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/ReadDir_da_raiz_falhou:_d_!=_nil,_caminho_==_raiz
=== RUN   TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/entrada_comum_ilegivel
--- PASS: TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir (0.00s)
    --- PASS: TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/Lstat_da_raiz_falhou:_d_==_nil (0.00s)
    --- PASS: TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/ReadDir_da_raiz_falhou:_d_!=_nil,_caminho_==_raiz (0.00s)
    --- PASS: TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/entrada_comum_ilegivel (0.00s)
=== RUN   TestWalkNaoEngoleRaizQueExisteMasNaoLe
--- PASS: TestWalkNaoEngoleRaizQueExisteMasNaoLe (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/vault	0.742s
testing: warning: no tests to run
PASS
ok  	github.com/jonyd/gobsidian/internal/watcher	0.810s [no tests to run]
```

O `no tests to run` no `watcher` **não é o teste passando**: é o
`//go:build !windows` excluindo `TestVarreDiretorioNovoNaoEngoleRaizIlegivel`
desta plataforma. Ele nunca rodou nesta máquina. O que existe aqui é prova de
**compilação** sob Linux — ver "Cobertura do teste `!windows`".

## Prova de mutação

### Mutação 1 — a conta (`FalhaNaRaiz`)

```
pwsh -File scripts/mutate.ps1 -Path internal/vault/walk.go `
  -Anchor 'return d == nil || caminho == raiz' `
  -Replacement 'return d == nil' `
  -Test TestFalhaNaRaiz -Package ./internal/vault/
```

```
[...] Mutando internal/vault/walk.go
      - return d == nil || caminho == raiz
      + return d == nil

[...] go test -race -run TestFalhaNaRaiz ./internal/vault/
----------------------------------------------------------------------
--- FAIL: TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir (0.00s)
    --- FAIL: TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/ReadDir_da_raiz_falhou:_d_!=_nil,_caminho_==_raiz (0.00s)
        walk_raiz_test.go:32: FalhaNaRaiz("C:\\cofre", "C:\\cofre", {cofre}) = false, quer true
FAIL
FAIL	github.com/jonyd/gobsidian/internal/vault	0.574s
FAIL
----------------------------------------------------------------------
[OK] internal/vault/walk.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

Reprovou: `TestFalhaNaRaizReconheceAsDuasFormasDoWalkDir/ReadDir_da_raiz_falhou`,
em `internal/vault/walk_raiz_test.go:32`. Restauro conferido pelo próprio script
(SHA-256).

### Mutação 2 — o consumidor `Walk` (substituto do brief para Windows)

O brief manda, em Windows, mutar à mão e rodar
`TestWalkNaoEngoleRaizQueExisteMasNaoLe`. Usei o `mutate.ps1` para a mesma
mutação, porque ele confere o restauro byte a byte; o diff aplicado está na
saída.

```
pwsh -File scripts/mutate.ps1 -Path internal/vault/walk.go `
  -Anchor 'if FalhaNaRaiz(v.walkRoot, abs, d) {' `
  -Replacement 'if d == nil {' `
  -Test TestWalkNaoEngoleRaizQueExisteMasNaoLe -Package ./internal/vault/
```

```
[...] Mutando internal/vault/walk.go
      - if FalhaNaRaiz(v.walkRoot, abs, d) {
      + if d == nil {

[...] go test -race -run TestWalkNaoEngoleRaizQueExisteMasNaoLe ./internal/vault/
----------------------------------------------------------------------
--- FAIL: TestWalkNaoEngoleRaizQueExisteMasNaoLe (0.00s)
    walk_raiz_windows_test.go:50: Walk devolveu nil com 0 entradas para uma raiz que ReadDir nao le: cofre inacessivel virou cofre vazio
FAIL
FAIL	github.com/jonyd/gobsidian/internal/vault	0.652s
FAIL
----------------------------------------------------------------------
[OK] internal/vault/walk.go restaurado byte a byte (SHA-256 confere).

[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

Reprovou: `TestWalkNaoEngoleRaizQueExisteMasNaoLe`, em
`internal/vault/walk_raiz_windows_test.go:50`.

### Mutação 3 — o consumidor `watcher`: NÃO obtenível nesta máquina

Comando do brief, rodado como está, com a saída real:

```
pwsh -File scripts/mutate.ps1 -Path internal/watcher/watcher.go `
  -Anchor 'vault.FalhaNaRaiz(dir, caminho, d)' `
  -Replacement 'd == nil' `
  -Test TestVarreDiretorioNovoNaoEngoleRaizIlegivel -Package ./internal/watcher/
```

```
[...] Mutando internal/watcher/watcher.go
      - vault.FalhaNaRaiz(dir, caminho, d)
      + d == nil

[...] go test -race -run TestVarreDiretorioNovoNaoEngoleRaizIlegivel ./internal/watcher/
----------------------------------------------------------------------
ok  	github.com/jonyd/gobsidian/internal/watcher	2.242s [no tests to run]
----------------------------------------------------------------------
[OK] internal/watcher/watcher.go restaurado byte a byte (SHA-256 confere).

[!] O teste PASSOU com a regra mutada.
    TestVarreDiretorioNovoNaoEngoleRaizIlegivel nao consegue reprovar sem essa regra: ela esta escrita, nao verificada.
EXIT=1
```

**Leia o `EXIT=1` com o `[no tests to run]` ao lado.** O teste não sobreviveu à
mutação; ele **não rodou**, porque é `//go:build !windows` e esta máquina é
Windows. O `mutate.ps1` não distingue "passou" de "não existe nesta plataforma",
e é por isso que o brief prescreveu o substituto da Mutação 2.

O que isso deixa em aberto, dito sem maquiagem: **o call site do `watcher` não
tem prova de mutação executada nesta máquina.** Ele tem (a) a conta compartilhada
provada pela Mutação 1, (b) compilação verificada sob Linux, e (c) um teste que
reprovaria em Linux — não verificado por execução. Ver "Concerns".

Restauro dos três arquivos mutados, conferido depois de tudo:

```
$ git diff --stat internal/vault/walk.go internal/watcher/watcher.go
 internal/vault/walk.go      | 30 +++++++++++++++++++++++++-----
 internal/watcher/watcher.go |  9 ++++++---
 2 files changed, 31 insertions(+), 8 deletions(-)
```

Esse diff é a **mudança da Task**, não resíduo de mutação: os três `mutate.ps1`
reportaram restauro com SHA-256 conferido, e o diff acima foi lido linha a linha
antes do commit (é o que está em `git show 3dfcf8e`).

## Cobertura do teste `!windows`

`TestVarreDiretorioNovoNaoEngoleRaizIlegivel` **não roda nesta máquina**. O que
foi verificado sobre ele, e nada além disso:

```
$ GOOS=linux go vet ./internal/watcher
[OK] GOOS=linux go vet ./internal/watcher (inclui _test.go com //go:build !windows)

$ GOOS=linux go list -f '{{.GoFiles}} | TEST: {{.TestGoFiles}}' ./internal/watcher
varredura_raiz_ilegivel_test.go
varredura_raiz_test.go
```

O arquivo entra em `TestGoFiles` sob Linux e passa no `go vet` — ele **compila e
é type-checked**. Não é "o teste passa".

## As verificações do brief

1. **`walk.go` e `watcher.go` chamam a MESMA função; nenhuma decisão de raiz
   feita à mão sobrou.** PASS.

   ```
   $ grep -rn "d == nil" internal/vault internal/watcher
   internal/vault/walk.go:123:// WalkDir tem duas formas de falhar na raiz, e so uma delas vem com d == nil:
   internal/vault/walk.go:125://   - Lstat(raiz) falhou: um unico callback, d == nil.
   internal/vault/walk.go:129:// Tratar so d == nil deixa a segunda forma passar como "entrada ilegivel":
   internal/vault/walk.go:135:	return d == nil || caminho == raiz
   internal/vault/walk_raiz_test.go:25:		{"Lstat da raiz falhou: d == nil", raiz, nil, true},
   internal/watcher/varredura_raiz_ilegivel_test.go:17:// caminho que nao existe, onde Lstat falha e o callback vem com d == nil. A
   internal/watcher/varredura_raiz_test.go:19:// filepath.WalkDir chama o callback com d == nil e erro != nil quando NÃO
   internal/watcher/watcher.go:230:			// d == nil significa falha na PROPRIA RAIZ: o WalkDir nao
   ```

   A única ocorrência **em código** é a linha 135: o corpo da própria conta. As
   demais são comentário ou nome de subteste.

2. **`internal/watcher/watcher.go:63` (varredura inicial, `path == root`) NÃO foi
   tocado.** PASS — segui o brief à risca e **não** o migrei.

   Registro para o orquestrador, porque é a única expressão da mesma distinção
   que continua fora da conta: `path == root` ali é equivalente a
   `FalhaNaRaiz(root, path, d)`, porque `WalkDir` só entrega `d == nil` para a
   raiz — logo `d == nil` implica `path == root`. Está correto hoje e o
   comportamento não muda com a migração. Não migrei por dois motivos: o brief
   diz explicitamente que não é desta Task, e a migração acrescentaria um
   terceiro call site **sem teste que o cubra**, isto é, uma chamada da conta sem
   prova de mutação. Se o orquestrador quiser fechar o "uma conta por regra"
   também ali, vale uma tarefa com o teste junto.

3. **O teste Windows reprova ANTES do fix, com a mensagem que nomeia a raiz.**
   PASS — saída colada em "RED 2". A mensagem é
   `Walk devolveu nil com 0 entradas para uma raiz que ReadDir nao le: cofre inacessivel virou cofre vazio`.

4. **`go list -f '{{.Imports}}' ./internal/watcher` sem aresta nova.** PASS.

   ```
   [context errors fmt github.com/cespare/xxhash/v2 github.com/fsnotify/fsnotify
    github.com/jonyd/gobsidian/internal/index github.com/jonyd/gobsidian/internal/search
    github.com/jonyd/gobsidian/internal/vault io/fs log/slog os path/filepath sync/atomic time]
   ```

   `internal/vault` já estava lá. Nenhum import acrescentado, em produção ou em
   teste (`walk_raiz_windows_test.go` usa `golang.org/x/sys/windows`, que é
   dependência de teste do `vault`, já no `go.mod` como indireta e já no
   `go.sum`).

## `verify.ps1`

`pwsh -File scripts/verify.ps1` — completo, sem `-SkipCross` nem `-SkipNet`.

```
[...] 13. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
GATE_EXIT=0
```

**Contagem real de etapas: 13**, numeradas de `1. go build` a
`13. check_readme_anchors`, todas `[OK]`. O `CLAUDE.md` diz "14 etapas" — a
divergência é do documento, não do gate; não a corrigi porque este commit é de
uma categoria só (ver "O que ficou de fora").

## O que ficou de fora

- **O arquivo `internal/watcher/varredura_raiz_test.go` não foi escrito.** O
  brief o lista como novo; ele existe desde `499e744` com
  `TestVarreDiretorioNovoAcusaFalhaNaPropriaRaiz` e
  `TestVarreDiretorioNovoSegueApesarDeEntradaIlegivel`, sem build tag. Gravar o
  conteúdo do brief nele apagaria os dois e, com o `//go:build !windows` que o
  brief pede, tiraria do Windows a cobertura que eles dão hoje. O teste do brief
  foi para `internal/watcher/varredura_raiz_ilegivel_test.go`, **corpo idêntico
  ao do brief**, mais um comentário dizendo por que ele existe ao lado do irmão
  (o irmão cobre só a primeira forma, o caminho inexistente) e por que fica fora
  do Windows. Nenhuma linha do arquivo pré-existente foi tocada.
- **`internal/watcher/watcher.go:63` não foi migrado para `FalhaNaRaiz`** —
  verificação 2 acima, com o raciocínio.
- **A contagem de etapas do `verify.ps1` no `CLAUDE.md` (14 vs 13 reais) não foi
  corrigida.** É outra categoria de commit (`docs:`), e a regra do brief é anotar
  em vez de corrigir.
- **Uma linha de comentário do brief foi ajustada**, e vale dizer qual: em
  `watcher.go:232` o comentário citava `(walk.go:133)`, número de linha que a
  própria Task invalidava. Trocado por "que vault.Walk ja faz", sem número. O
  resto do comentário :230-244 ficou como estava, como o brief manda, e ganhou a
  linha prescrita sobre a segunda forma.
- **Nada foi medido em desempenho** — a Task não pede número, e não há nenhum
  neste relatório além dos tempos que o `go test` imprime sozinho.

## `git status --porcelain`

```
 M .superpowers/sdd/2026-07-25-gobsidian-v01/progress.md
 M "test-vault/test vault/.obsidian/community-plugins.json"
 D "test-vault/test vault/.obsidian/plugins/parity-dumper/main.js"
 D "test-vault/test vault/.obsidian/plugins/parity-dumper/manifest.json"
 M "test-vault/test vault/.obsidian/workspace.json"
?? .claude/skills/troglodita-commit/
?? .claude/skills/troglodita-help/
?? .claude/skills/troglodita-review/
?? .claude/skills/troglodita/
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-147-base.txt
?? .superpowers/sdd/2026-09-02-topografia-e-limpeza/task-147-brief.md
?? Resume-Claude.ps1
?? "test-vault/test vault/.obsidian/community-plugins (conflito 01M08RSCY410E88M0516VB4K9R 2026-08-17 22h52).json"
?? "test-vault/test vault/.obsidian/core-plugins (conflito 01M08RSCY410E88M0516VB4K9R 2026-08-17 22h52).json"
?? "test-vault/test vault/.obsidian/plugins/gosync/"
?? "test-vault/test vault/A\303\247\303\243o.md"
?? "test-vault/test vault/Pasted image 20260814221454.png"
?? "test-vault/test vault/Sem t\303\255tulo.md"
?? "test-vault/test vault/main.js"
?? "test-vault/test vault/manifest.json"
```

**Byte a byte o mesmo estado de antes da Task**, a menos dos cinco arquivos que
entraram no commit. Nenhum arquivo do usuário foi tocado: `test-vault/`,
`.claude/skills/`, `Resume-Claude.ps1` e
`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md` continuam exatamente como
estavam (as modificações listadas neles já existiam no início — confirmado contra
o `git status` da abertura da sessão). Este relatório é o único arquivo
acrescentado, e é untracked de propósito: não está na lista de `git add` do
brief.

## Concerns para o revisor

1. **O call site do `watcher` não tem prova de mutação executada.** É a lacuna
   real desta entrega, e é de plataforma, não de esforço: o instrumento portátil
   (`chmod 0o000`) não funciona no Windows, e o instrumento do Windows (handle
   exclusivo) está no pacote `vault`, onde a função auxiliar é privada. Provar o
   `watcher` nesta máquina exigiria duplicar o truque do handle no pacote
   `watcher` — arquivo que o brief não lista. Não fiz. Se a revisão quiser essa
   prova, é uma tarefa pequena e eu recomendo que exista, porque hoje o único
   ambiente que exercita esse call site é um que ninguém roda aqui.
2. **O comentário de `watcher.go:230` ainda abre com "d == nil significa falha na
   PROPRIA RAIZ"**, e o código logo abaixo não testa mais `d == nil`. Lido
   inteiro o parágrafo fecha (a linha nova nomeia a segunda forma), mas quem
   grepar `d == nil` não acha o condicional. Mantive porque o brief diz
   explicitamente que o comentário :230-244 fica e que a mudança é acrescentar
   uma linha. Se a revisão preferir, a reescrita da primeira frase é trivial.
3. **`CLAUDE.md` diz que `verify.ps1` tem 14 etapas; ele tem 13.** Não é desta
   Task e não foi corrigido.

---

## Fix round 1 — o comentário de `watcher.go:230`

Achado Minor da revisão, aceito: o parágrafo abria com "d == nil significa falha
na PROPRIA RAIZ" enquanto o guarda logo abaixo já era
`vault.FalhaNaRaiz(dir, caminho, d)`. Era a Concern 2 deste relatório; a decisão
do orquestrador é que "o comentário fica", no brief, não licencia uma primeira
frase que não descreve mais a linha sob ela — comentário que mente sobre o código
é defeito, não preferência. Corrigido.

**Commit:** `a74b766 docs(watcher): the root-failure comment names both forms WalkDir produces`

Separado de `3dfcf8e`, que **não** foi emendado. Um arquivo, só comentário:

```
 internal/watcher/watcher.go | 14 +++++++-------
 1 file changed, 7 insertions(+), 7 deletions(-)
```

### Diff

```diff
@@ -227,10 +227,13 @@ func (w *Watcher) emite(ctx context.Context, e fsnotify.Event) error {
 func (w *Watcher) varreDiretorioNovo(ctx context.Context, dir string) error {
 	err := filepath.WalkDir(dir, func(caminho string, d fs.DirEntry, erro error) error {
 		if erro != nil {
-			// d == nil significa falha na PROPRIA RAIZ: o WalkDir nao
-			// conseguiu nem fazer Lstat no diretorio que acabou de chegar.
-			// E a mesma distincao que vault.Walk ja faz, e que aqui nunca
-			// tinha sido feita.
+			// Falha na PROPRIA RAIZ — o diretorio que acabou de chegar —, nas
+			// duas formas que o WalkDir produz: Lstat(dir) falhou, e o
+			// callback vem com d == nil; ou Lstat passou e ReadDir(dir)
+			// falhou, e o callback vem com d != nil e caminho == dir.
+			// vault.FalhaNaRaiz e a conta unica dessa distincao, a mesma que
+			// vault.Walk usa. Reconhecer so a primeira forma era o defeito que
+			// ficava aqui.
 			//
 			// Engolir isso faz a varredura reportar sucesso com ZERO entradas
 			// — exatamente o estado que esta funcao existe para impedir. O
@@ -242,9 +245,6 @@ func (w *Watcher) varreDiretorioNovo(ctx context.Context, dir string) error {
 			//
 			// Diretorio inacessivel e diretorio vazio nao podem produzir a
 			// mesma resposta.
-			//
-			// A segunda forma — ReadDir da raiz falhou, d != nil — e a que
-			// vault.FalhaNaRaiz cobre.
 			if vault.FalhaNaRaiz(dir, caminho, d) {
 				return fmt.Errorf("varrendo a raiz do diretório novo %q: %w", caminho, erro)
 			}
```

A frase acrescentada na rodada anterior ("A segunda forma — ReadDir da raiz
falhou, d != nil — e a que vault.FalhaNaRaiz cobre") saiu: virou redundante
depois que a abertura passou a nomear as duas formas. Os parágrafos do meio
(:235-244 na numeração antiga — sintoma, antivírus, "diretório inacessível e
diretório vazio não podem produzir a mesma resposta") ficaram intactos.

### Verificações desta rodada

```
$ gofmt -l internal/watcher
[OK] gofmt sem achados
```

```
$ pwsh -File scripts/verify.ps1 -SkipCross -SkipNet
[...] 10. check_readme_anchors
[OK] check_readme_anchors

[OK] Bateria completa. Pode commitar.
GATE_EXIT=0
```

**10 etapas**, todas `[OK]`, com `vet` cruzado e `check_net` pulados pelas flags
— como instruído, porque a mudança é só de comentário e o gate completo (13
etapas) já rodou em `3dfcf8e`. `go test -race` (etapa 2) rodou mesmo assim e
passou.

```
$ git status --porcelain internal/
(vazio)
```

### Situação das Concerns depois desta rodada

1. **Prova de mutação do call site do `watcher`** — continua aberta, inalterada.
   É de plataforma, não de esforço; a recomendação de tarefa pequena segue de pé.
2. **Comentário de `watcher.go:230`** — FECHADA por este commit.
3. **`CLAUDE.md` diz 14 etapas, o gate tem 13** — continua aberta, outra
   categoria de commit. Esta rodada, com as duas flags, imprimiu 10.
