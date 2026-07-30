# Relatório Task 63: Offsets para `LinkMarkdown` e `LinkEmbed` no parser

- **Status**: DONE
- **Commit**: `feat(parser): byte offsets for markdown links and embeds`

## O Que Foi Implementado
- Implementada a função `findMarkdownLinkSpan` em [internal/parser/ast.go](file:///C:/Users/jonyd/Projetos/Gobsidian/internal/parser/ast.go) para calcular os offsets `Start` e `End` exatos de links (`LinkMarkdown`) e embeds (`LinkEmbed`) em grafia Markdown (`[texto](destino)` e `![alt](imagem)`).
- Atualizada a função `collect` em [internal/parser/ast.go](file:///C:/Users/jonyd/Projetos/Gobsidian/internal/parser/ast.go) para preencher `Start` e `End` adicionando o `bodyOffset` (preservando o suporte a notas com BOM e frontmatter).
- Onde o offset não puder ser determinado com segurança, ele permanece no valor sentinela `offsetUnknown` (`-1`), prevenindo sobreescritas no início do arquivo.
- Criado o teste unitário dedicado [internal/parser/markdown_link_test.go](file:///C:/Users/jonyd/Projetos/Gobsidian/internal/parser/markdown_link_test.go).
- Atualizados os 2 golden files afetados (`wikilinks/embed_imagem.json` e `wikilinks/aninhado_ambiguo.json`), após inspeção e confirmação da exatidão dos offsets.

## Evidência de TDD

### Comando do RED
`go test -v ./internal/parser/...` (antes de preencher `Start` e `End`)
```
--- FAIL: TestLinkMarkdownOffsets (0.00s)
    --- FAIL: TestLinkMarkdownOffsets/link_markdown_simples (0.00s)
        markdown_link_test.go:75: Start ou End e -1: Start=-1, End=-1
FAIL
```

### Comando do GREEN
`go test -v ./internal/parser/...` (após implementar `findMarkdownLinkSpan`)
```
=== RUN   TestLinkMarkdownOffsets
=== RUN   TestLinkMarkdownOffsets/link_markdown_simples
=== RUN   TestLinkMarkdownOffsets/embed_markdown_simples
=== RUN   TestLinkMarkdownOffsets/aninhado_ambiguo
=== RUN   TestLinkMarkdownOffsets/parenteses_no_destino
=== RUN   TestLinkMarkdownOffsets/com_titulo
=== RUN   TestLinkMarkdownOffsets/com_BOM
--- PASS: TestLinkMarkdownOffsets (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/parser	0.624s
```

## Prova de Mutação

### Comando de Mutação
`pwsh -File scripts/mutate.ps1 -Path internal/parser/ast.go -Anchor 'start, end := findMarkdownLinkSpan(node, false, body)' -Replacement 'start, end := offsetUnknown, offsetUnknown' -Test TestLinkMarkdownOffsets -Package ./internal/parser/`

### Saída Real do Script
```
[...] Mutando internal/parser/ast.go
      - start, end := findMarkdownLinkSpan(node, false, body)
      + start, end := offsetUnknown, offsetUnknown

[...] go test -race -run TestLinkMarkdownOffsets ./internal/parser/
----------------------------------------------------------------------
--- FAIL: TestLinkMarkdownOffsets (0.00s)
    --- FAIL: TestLinkMarkdownOffsets/link_markdown_simples (0.00s)
        markdown_link_test.go:75: Start ou End e -1: Start=-1, End=-1
    --- FAIL: TestLinkMarkdownOffsets/aninhado_ambiguo (0.00s)
        markdown_link_test.go:75: Start ou End e -1: Start=-1, End=-1
    --- FAIL: TestLinkMarkdownOffsets/parenteses_no_destino (0.00s)
        markdown_link_test.go:75: Start ou End e -1: Start=-1, End=-1
    --- FAIL: TestLinkMarkdownOffsets/com_titulo (0.00s)
        markdown_link_test.go:75: Start ou End e -1: Start=-1, End=-1
    --- FAIL: TestLinkMarkdownOffsets/com_BOM (0.00s)
        markdown_link_test.go:75: Start ou End e -1: Start=-1, End=-1
FAIL
FAIL	github.com/jonyd/gobsidian/internal/parser	0.713s
FAIL
----------------------------------------------------------------------
[OK] internal/parser/ast.go restaurado byte a byte (SHA-256 confere).
[OK] O teste REPROVOU com a regra mutada — a regra esta verificada.
```

## Verificações Exigidas pelo Brief

| Verificação | Resultado Real |
|---|---|
| `[texto](destino.md)` simples: `raw[Start:End]` devolve link inteiro | OK (`raw[Start:End] == "[texto](destino.md)"`) |
| `![alt](img.png)`: `raw[Start:End]` devolve embed inteiro | OK (`raw[Start:End] == "![alt](img.png)"`) |
| `[[[a]] b](d.md)`: span vai até o `)` certo | OK (`raw[Start:End] == "[[[a]] b](d.md)"`) |
| Link com parênteses no destino `[x](a(b).md)` e com título `[x](a.md "t")` | OK (`raw[Start:End]` confere byte a byte) |
| Nota com BOM: offsets vêm com `bodyOffset` somado | OK (`raw[Start:End] == "[texto](destino.md)"` em buffer com BOM `\xef\xbb\xbf`) |
| Quantos goldens do corpus mudaram? | 2 (`testdata/parser/wikilinks/embed_imagem.json` e `testdata/parser/wikilinks/aninhado_ambiguo.json`). Diffs lidos e validados: anteriormente tinham `start: -1, end: -1` e agora contêm os offsets byte-exatos dos links no buffer. |
| Sobrou algum `Kind` com offset -1? | Nenhum `Kind` de link é deixado em -1 por padrão; caso ocorra sintaxe malformada sem fechamento de `)` ou `]`, o parser reverte com segurança para `offsetUnknown` (-1). |

## Auditoria de Relatório
`pwsh -File scripts/audit_reports.ps1 63` executado (0 achados).

## Bateria de Verificação
`pwsh -File scripts/verify.ps1`: **8/8 etapas VERDES**.
`pwsh -File scripts/build.ps1`: **Binário gerado com sucesso**.

## Arquivos Alterados
- `internal/parser/ast.go`
- `internal/parser/markdown_link_test.go`
- `testdata/parser/wikilinks/aninhado_ambiguo.json`
- `testdata/parser/wikilinks/embed_imagem.json`
- `.superpowers/sdd/task-63-report.md`

## O Que Ficou de Fora
Nada.

## git status --porcelain
```
?? .superpowers/sdd/2026-07-25-gobsidian-v01/task-63-base.txt
?? .superpowers/sdd/task-63-report.md
 M internal/parser/ast.go
?? internal/parser/markdown_link_test.go
 M testdata/parser/wikilinks/aninhado_ambiguo.json
 M testdata/parser/wikilinks/embed_imagem.json
```
