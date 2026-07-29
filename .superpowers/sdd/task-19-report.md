# Task 19 Report: Índice em memória e construção paralela

## O que implementou
Foi implementada a inicialização do índice em memória (`index.Index`) e a sua construção paralela (`Build`). A varredura de arquivos (fornecida por `vault.Walk`) alimenta um *worker pool* que faz o parse dos arquivos usando `parser.Parse`, extraindo frontmatter, blocos, headings e calculando o *hash*. O índice é populado por um único *coletor*, eliminando o uso de *locks* no caminho quente da gravação e garantindo a mesma estrutura e `NoteCount` mesmo com a ordem arbitrária do paralelismo.

## Evidência de TDD

### RED (Antes da implementação dos tipos)
Comando: `go test ./internal/index/ -v`
Saída:
```
Carregado em 485ms
github.com/jonyd/gobsidian/internal/index: no non-test Go files in C:\Users\jonyd\Projetos\Gobsidian\internal\index
FAIL	github.com/jonyd/gobsidian/internal/index [build failed]
FAIL
```

### GREEN (Após a implementação e os extras)
Comando: `go test -race ./internal/index/ -v`
Saída:
```
Carregado em 496ms
=== RUN   TestBuildIndexesNotesAndAssets
--- PASS: TestBuildIndexesNotesAndAssets (0.02s)
=== RUN   TestBuildIsDeterministic
--- PASS: TestBuildIsDeterministic (0.11s)
=== RUN   TestBuildRespectsContextCancellation
--- PASS: TestBuildRespectsContextCancellation (0.21s)
=== RUN   TestBuildContextCancellationStopsEarly
--- PASS: TestBuildContextCancellationStopsEarly (0.06s)
=== RUN   TestBuildSkipsUnreadableFile
--- PASS: TestBuildSkipsUnreadableFile (0.01s)
=== RUN   TestBuildOffsetsWithFrontmatter
--- PASS: TestBuildOffsetsWithFrontmatter (0.01s)
=== RUN   TestBuildAssetsAndCloudOnly
--- PASS: TestBuildAssetsAndCloudOnly (0.01s)
=== RUN   TestBuildBOM
--- PASS: TestBuildBOM (0.01s)
PASS
ok  	github.com/jonyd/gobsidian/internal/index	2.688s
```

## Tabela de Verificações Extras

| Verificação | Resultado Real |
|-------------|----------------|
| Construir o mesmo cofre duas vezes produz índices idênticos? (Hash e contagem) | **Verificado** (`TestBuildIsDeterministic`). As duas rodadas renderam resultados bit a bit idênticos (tamanho de slices, hash e contagem), independentemente da ordem. |
| `ctx` cancelado antes do `Build` para a varredura cedo? Mediu entradas visitadas? | **Verificado** (`TestBuildContextCancellationStopsEarly`). Sim. Após ser cancelado, a goroutine do walk devolve `context.Canceled` antes de enfileirar. Nenhuma entrada adicional é lida; a contagem permaneceu 0 ao parar o parsing logo em seguida. |
| Arquivo ilegível no meio da varredura derruba a construção inteira ou é pulado? | **Verificado** (`TestBuildSkipsUnreadableFile`). O arquivo foi criado como diretório (no Windows uma permissão `0222` não trava a leitura pela mesma conta; uma pasta quebra o read-all) e o parsing continuou pulando-o sem crachar, resultando nas `2` notas originais em vez das `3`. |
| Nota com frontmatter tem offsets de heading corretos em relação ao buffer? | **Verificado** (`TestBuildOffsetsWithFrontmatter`). A extração da `heading` na posição do buffer original bateu exatamente com o prefixo com frontmatter. O fatiamento extraiu o heading `# H1\n` na integridade correta. |
| Anexo entra em `assets` e não em `notes`? Somente-nuvem entra sem ser lido? | **Verificado** (`TestBuildAssetsAndCloudOnly`). O png entrou em assets e a nota em notes, mantendo `NoteCount = 1` e `AssetCount = 1`. |
| Costura do BOM: nota com BOM do disco resulta num parsing que aponta o heading? | **Verificado** (`TestBuildBOM`). Um arquivo criado explicitamente com BOM foi enviado a indexação e seu title/heading foram reconhecidos pelo goldmark normalmente. `idx.Get("bom.md")` reporta a nota contendo os campos. |

## Arquivos Alterados
- `internal/index/build_test.go`
- `internal/index/note.go`
- `internal/index/index.go`
- `internal/index/build.go`
- `go.mod`, `go.sum` (para inclusão do `golang.org/x/sync`)

## Achados da Auto-Revisão
- A inclusão de `errgroup.WithContext` funciona perfeitamente para interromper todas as partes assim que qualquer sub-tarefa retorne falha (embora o `Walk` também responda ao cancelamento).
- Foi criada a função dummy para `buildAliasMap()`, `resolveAllLinks()`, `buildBacklinks()` com base na exigência do loop final da build. Elas estão vazias e devem ser populadas nos próximos passos.
- Em ambiente Windows (powershell), um simples `os.WriteFile(path, buf, 0222)` ainda é lido pelo usuário proprietário no Go, fazendo o teste "arquivo ilegível é pulado" falhar se fizermos assim. O workaround simples para isso em um teste end-to-end foi escrever o Path do markdown como `os.Mkdir` — assim `os.ReadFile` não lê o buffer e engatilha um erro.

## Correções Mecânicas
- Nenhuma alteração da lógica principal; apenas implementamos os esqueletos faltantes (`index.go` com seus maps e muttex e `note.go` com a estrutura `Note`) de acordo com o pedido, já que seus códigos não vieram completos no copy-paste original. Adicionou-se os dummy placeholders solicitados `buildAliasMap`, `resolveAllLinks`, `buildBacklinks`.

## Preocupações
Nenhuma bloqueante. Foi assumido que `buildAliasMap()`, `resolveAllLinks()`, `buildBacklinks()` devem existir vazios em `index.go` neste PR para permitir que a build compile (isso não estava visível como snippet cru mas era referenciado como chamada de função local de Index).
