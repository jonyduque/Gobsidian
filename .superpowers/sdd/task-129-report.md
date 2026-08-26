# Task 129 — C2: symlink recusado por padrão, com saída

**Status:** DONE_WITH_CONCERNS — **um erro meu destruiu um arquivo de teste
existente antes de ser recuperado.** Ver "O que deu errado no caminho".
**Commit:** `e345578` — `fix(vault,search): close the three critical file-access findings`

---

## O que deu errado no caminho

Criei `internal/vault/symlink_test.go` com a ferramenta `Write` **sem verificar
que o arquivo já existia**. Ele existia, vindo do commit `12ad94f`, e continha
`TestWalkNaoSegueSymlink` e o helper `criaSymlink`. A escrita substituiu o
conteúdo inteiro.

Percebi porque o `git status` mostrou ` M` (modificado) e não `??` (novo). O
original foi recuperado com `git show HEAD:internal/vault/symlink_test.go` e os
testes novos foram **somados** a ele, não colocados no lugar dele.

O helper original é melhor que o que eu havia escrito: `criaSymlink` distingue
recusa por **permissão** (que no Windows é o caso comum, e justifica `t.Skip`)
de qualquer outra falha (que é `t.Fatal`), com a mensagem dizendo qual foi. O
meu usava `t.Skipf` para tudo — cobertura fantasma disfarçada de skip legítimo,
que é exatamente o defeito que o comentário do original nomeia. Os testes novos
passaram a usar o helper original.

**Lição:** ferramenta de escrita que substitui arquivo inteiro precisa de
verificação de existência antes, e `git status` é o que denuncia — `??` contra
` M`.

---

## O achado que a tarefa não previa: RNF-32 estava afirmado além do verificado

`docs/OPERACAO.md:207` publicava:

> **RNF-32** | Links simbólicos para fora do cofre não são seguidos | **não
> medido**; verificado por teste (`TestWalkNaoSegueSymlink`, executado com
> privilégio) | **Atingido**

Esse teste cobre symlink de **diretório** — que `filepath.WalkDir` nunca
atravessou, e que portanto valia "por construção". O symlink de **arquivo**
nunca teve teste e nunca funcionou: um `nota.md` apontando para fora passava nas
duas camadas léxicas, entrava no índice, e `note_read` devolvia conteúdo
arbitrário pelo canal MCP.

Metade do requisito estava publicada como atingida sem nunca ter sido
verificada. A linha foi corrigida para dizer exatamente isso, e agora nomeia os
três testes.

---

## Evidência de TDD

### RED (antes de existir `vault.SeguirSymlinks`)

```
$ go test -run 'TestLeituraRecusaSymlink|TestWalkPulaSymlink|TestSeguirSymlinks' ./internal/vault/

internal\vault\symlink_test.go:131:29: too many arguments in call to vault.New
	have (string, unknown type)
	want (string)
internal\vault\symlink_test.go:131:35: undefined: vault.SeguirSymlinks
FAIL	github.com/jonyd/gobsidian/internal/vault [build failed]
```

### GREEN

```
=== RUN   TestWalkNaoSegueSymlink
--- PASS: TestWalkNaoSegueSymlink (0.01s)
=== RUN   TestLeituraRecusaSymlinkPorPadrao
--- PASS: TestLeituraRecusaSymlinkPorPadrao (0.01s)
=== RUN   TestWalkPulaSymlinkDeArquivo
--- PASS: TestWalkPulaSymlinkDeArquivo (0.01s)
=== RUN   TestSeguirSymlinksPreservaOComportamentoAntigo
--- PASS: TestSeguirSymlinksPreservaOComportamentoAntigo (0.02s)
PASS
ok  	github.com/jonyd/gobsidian/internal/vault	1.198s
```

O pré-existente continua verde: a mudança não regrediu o caso de diretório.

---

## Provas de mutação

### Guarda da varredura

```
pwsh -File scripts/mutate.ps1 -Path internal/vault/walk.go `
  -Anchor 'if !v.seguirSymlinks && d.Type()&fs.ModeSymlink != 0 {' `
  -Replacement 'if false {' `
  -Test TestWalkPulaSymlinkDeArquivo -Package ./internal/vault/
```

```
[OK] internal/vault/walk.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

### Guarda da leitura — a primeira tentativa saiu INCONCLUSIVA

```
-Anchor 'if fi.Mode()&os.ModeSymlink != 0 {' -Replacement 'if false {'
```

```
[!] INCONCLUSIVO: o teste nao chegou a rodar como assercao.
    Uma mutacao que quebra a compilacao nao prova cobertura: ela prova que
    o codigo nao compila.
EXIT=2
```

`if false` deixava `fi` sem uso. Refeito com um mutante que compila:

```
-Replacement 'if fi.Mode()&os.ModeSymlink != 0 && false {'
```

```
FAIL	github.com/jonyd/gobsidian/internal/vault	1.277s
----------------------------------------------------------------------
[OK] internal/vault/vault.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
EXIT=0
```

Registrado aqui porque o `EXIT=2` é a armadilha que o próprio `mutate.ps1`
documenta, e aceitá-lo como prova teria sido cobertura fantasma.

---

## O que entrou

- `vault.Opcao` variádica em `New`, para os chamadores existentes de
  `New(root)` continuarem valendo, e `vault.SeguirSymlinks(bool)`.
- `recusaSymlink` usando **`Lstat`** — `Stat` seguiria o link, que é o que se
  quer evitar — ligada em `Open`, `ReadAll` e `ReadRange`.
- `Walk` pula symlink de arquivo com **`RecordSkip`**, nunca em silêncio. O
  descarte aparece em `vault_stats` e no `doctor`.
- `config.FollowSymlinks` e a flag `--follow-symlinks` nos **seis** subcomandos
  que abrem cofre: `serve`, `daemon`, `index`, `inspect`, `search`, `doctor`.

**Recusar na varredura E na leitura não é redundância.** Indexar e recusar só na
leitura produziria uma nota que aparece em `note_list` e falha em `note_read` —
pior que qualquer um dos dois isolado.

---

## Verificações

1. **Flag nos seis subcomandos, conferido um a um.** Registrar em cinco e
   esquecer o sexto faria a flag virar no-op silencioso ali — a armadilha
   `ReadOnlySet`/`DebounceMSSet`, que este projeto já pagou.
2. **Contrapeso.** `TestSeguirSymlinksPreservaOComportamentoAntigo` existe
   porque a decisão do dono foi recusar por padrão **sem tirar a
   possibilidade**: symlinkar uma pasta externa para dentro do cofre é workflow
   legítimo e suportado pelo Obsidian, e recusar sem alternativa trocaria um
   risco hipotético — o dono do cofre é o "atacante" — por uma regressão certa.
3. `pwsh -File scripts/check_net.ps1`: **EXIT=0**.
4. `pwsh -File scripts/verify.ps1`: **14 de 14 [OK]**.

---

## O que ficou de fora

**A janela TOCTOU continua aberta.** `recusaSymlink` faz `Lstat` e depois
`os.Open`; entre as duas, o caminho pode virar symlink. A correção estrutural é
`os.Root`/`os.OpenRoot`, que torna o escape impossível por construção — Task 130.

**Symlink no meio do caminho** não é checado: a guarda olha o componente final.
Hoje isso não é alcançável porque `WalkDir` não desce em symlink de diretório,
mas é uma propriedade da construção, não da checagem.
