# Relatório Task 69: Os quatro parâmetros de schema que o código ignora

- **Status**: DONE
- **Commit**: `fix(mcpsrv,service): honour the four schema params the handlers dropped`

## O Que Foi Implementado
- Passados e honrados os quatro parâmetros declarados no schema MCP que eram ignorados nos handlers:
  1. `note_metadata.include`: em [internal/mcpsrv/tools_read.go](file:///C:/Users/jonyd/Projetos/Gobsidian/internal/mcpsrv/tools_read.go) e [internal/service/graph.go](file:///C:/Users/jonyd/Projetos/Gobsidian/internal/service/graph.go). Permite filtrar o conjunto de campos retornado pela tool (`frontmatter`, `tags`, `headings`, `blocks`, `links`, `backlinks`), mantendo `path`, `title` e `hash` sempre presentes.
  2. `link_graph.direction`: enums `"outgoing"`, `"incoming"`, `"both"` (default `"both"`).
  3. `link_graph.include_broken`: booleano (default `true`). Permite incluir ou excluir arestas e nós de alvos de links quebrados.
  4. `link_graph.include_embeds`: booleano (default `true`). Permite incluir ou excluir links do tipo embed (`![[...]`).
- Adicionada a etapa `check_tool_params` no script [scripts/verify.ps1](file:///C:/Users/jonyd/Projetos/Gobsidian/scripts/verify.ps1) para garantir que nenhum parâmetro de schema fique sem leitura em handlers futuros.
- Criada a suíte de testes em [internal/mcpsrv/schema_params_test.go](file:///C:/Users/jonyd/Projetos/Gobsidian/internal/mcpsrv/schema_params_test.go).

## Evidência de TDD

### Comando do RED
`pwsh -File scripts/check_tool_params.ps1` (antes de passar os parâmetros nos handlers)
```
[i] 12 structs de entrada, 68 parametros declarados.
[!] 4 parametro(s) declarado(s) e nunca lido(s):
    tools_read.go:253  noteMetadataInput.Include  (json: "include")
    tools_read.go:258  linkGraphInput.Direction  (json: "direction")
    tools_read.go:260  linkGraphInput.IncludeBroken  (json: "include_broken")
    tools_read.go:261  linkGraphInput.IncludeEmbeds  (json: "include_embeds")
```

### Comando do GREEN
`pwsh -File scripts/check_tool_params.ps1` (após implementar a passagem de parâmetros)
```
[i] 12 structs de entrada, 68 parametros declarados.
[OK] todo parametro declarado e lido em algum lugar.
```

## Saída do `check_tool_params.ps1`

### Antes da alteração
```
[i] 12 structs de entrada, 68 parametros declarados.
[!] 4 parametro(s) declarado(s) e nunca lido(s):
    tools_read.go:253  noteMetadataInput.Include  (json: "include")
    tools_read.go:258  linkGraphInput.Direction  (json: "direction")
    tools_read.go:260  linkGraphInput.IncludeBroken  (json: "include_broken")
    tools_read.go:261  linkGraphInput.IncludeEmbeds  (json: "include_embeds")
```

### Depois da alteração
```
[i] 12 structs de entrada, 68 parametros declarados.
[OK] todo parametro declarado e lido em algum lugar.
```

## Provas de Mutação

### 1. `note_metadata.include`
Comando: `pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/tools_read.go -Anchor 'Include: in.Include,' -Replacement 'Include: nil,' -Test TestNoteMetadata_IncludeParameter -Package ./internal/mcpsrv/`
Saída real:
```
[...] Mutando internal/mcpsrv/tools_read.go
      - Include: in.Include,
      + Include: nil,

[...] go test -race -run TestNoteMetadata_IncludeParameter ./internal/mcpsrv/
----------------------------------------------------------------------
--- FAIL: TestNoteMetadata_IncludeParameter (0.08s)
    schema_params_test.go:134: headings deveria ser vazio quando include=['tags'], obteve: [Heading A]
    schema_params_test.go:137: links deveria ser vazio quando include=['tags'], obteve: [map[Resolved:note_b.md State:0 Via:2 end:66 kind:0 raw:[[note_b]] start:56 target:note_b] map[Resolved:note_c.md State:0 Via:2 end:88 kind:1 raw:![[note_c]] start:77 target:note_c] map[Resolved: State:1 Via:0 end:116 kind:0 raw:[[missing_note]] start:100 target:missing_note]]
    schema_params_test.go:140: frontmatter deveria ser vazio quando include=['tags'], obteve: map[tags:[tagA] title:Note A]
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	1.426s
FAIL
----------------------------------------------------------------------
[OK] internal/mcpsrv/tools_read.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### 2. `link_graph.direction`
Comando: `pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/tools_read.go -Anchor 'direction = in.Direction' -Replacement 'direction = "both"' -Test TestLinkGraph_DirectionParameter -Package ./internal/mcpsrv/`
Saída real:
```
[...] Mutando internal/mcpsrv/tools_read.go
      - direction = in.Direction
      + direction = "both"

[...] go test -race -run TestLinkGraph_DirectionParameter ./internal/mcpsrv/
----------------------------------------------------------------------
--- FAIL: TestLinkGraph_DirectionParameter (0.06s)
    schema_params_test.go:173: nao esperava incoming edge quando direction='outgoing', encontrou: {Source:note_b.md Target:note_a.md}
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	1.354s
FAIL
----------------------------------------------------------------------
[OK] internal/mcpsrv/tools_read.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### 3. `link_graph.include_broken`
Comando: `pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/tools_read.go -Anchor 'includeBroken = *in.IncludeBroken' -Replacement 'includeBroken = true' -Test TestLinkGraph_IncludeBrokenParameter -Package ./internal/mcpsrv/`
Saída real:
```
[...] Mutando internal/mcpsrv/tools_read.go
      - includeBroken = *in.IncludeBroken
      + includeBroken = true

[...] go test -race -run TestLinkGraph_IncludeBrokenParameter ./internal/mcpsrv/
----------------------------------------------------------------------
--- FAIL: TestLinkGraph_IncludeBrokenParameter (0.06s)
    schema_params_test.go:206: nao esperava aresta quebrada quando include_broken=false, obteve: {Target:missing_note}
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	1.373s
FAIL
----------------------------------------------------------------------
[OK] internal/mcpsrv/tools_read.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

### 4. `link_graph.include_embeds`
Comando: `pwsh -File scripts/mutate.ps1 -Path internal/mcpsrv/tools_read.go -Anchor 'includeEmbeds = *in.IncludeEmbeds' -Replacement 'includeEmbeds = true' -Test TestLinkGraph_IncludeEmbedsParameter -Package ./internal/mcpsrv/`
Saída real:
```
[...] Mutando internal/mcpsrv/tools_read.go
      - includeEmbeds = *in.IncludeEmbeds
      + includeEmbeds = true

[...] go test -race -run TestLinkGraph_IncludeEmbedsParameter ./internal/mcpsrv/
----------------------------------------------------------------------
--- FAIL: TestLinkGraph_IncludeEmbedsParameter (0.06s)
    schema_params_test.go:239: nao esperava aresta de embed quando include_embeds=false, obteve: {Target:note_c.md}
FAIL
FAIL	github.com/jonyd/gobsidian/internal/mcpsrv	1.354s
FAIL
----------------------------------------------------------------------
[OK] internal/mcpsrv/tools_read.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

## Diff em `scripts/verify.ps1`
```diff
@@ -148,6 +148,8 @@ if (-not $SkipNet) {
     Write-Output "[i] check_net pulado (-SkipNet)"
 }

+Invoke-Step "check_tool_params" { & (Join-Path $PSScriptRoot "check_tool_params.ps1") }
+
 Pop-Location
```

## Bateria de Verificação
`pwsh -File scripts/verify.ps1`: **10/10 etapas VERDES**.

## Arquivos Alterados
- `internal/mcpsrv/tools_read.go`
- `internal/service/graph.go`
- `scripts/verify.ps1`
- `internal/mcpsrv/schema_params_test.go`
- `.superpowers/sdd/task-69-report.md`

## O Que Ficou de Fora
Nada.

## git status --porcelain
```
?? .superpowers/sdd/2026-07-25-gobsidian-v01/task-69-base.txt
?? .superpowers/sdd/task-69-report.md
 M internal/mcpsrv/schema_params_test.go
 M internal/mcpsrv/tools_read.go
 M internal/service/graph.go
 M scripts/verify.ps1
```
