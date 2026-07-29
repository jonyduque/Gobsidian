# Relatório da Task 20: Resolução de wikilink, aliases, anexos e âncoras

## O que foi implementado
- `internal/index/resolve_test.go`: Testes TDD cobrindo a ordem de resolução do Obsidian, empate por proximidade, case-insensitive fallback.
- `internal/index/resolve.go`: Lógica principal de resolução e fallback explícito -> name -> asset -> alias -> proximidade. Também inclui `ResolvePath`.
- `internal/index/alias.go`: Construção de mapas de alias para resolução rápida e agrupamento de colisões.
- `internal/index/assets.go`: Arquivo criado como stub de acordo com a exigência (a lógica em si ficou contida em `resolve.go` para assets).
- `internal/index/anchors.go`: Verificação de âncoras (headings normais e block ids).

## Evidência TDD
- **RED:** O `go test` falhou inicialmente com erros de compilação porque `ResolvePath` e as constantes/lógicas não existiam.
- **GREEN:** Após implementar os arquivos, o `go test -race ./internal/index/ -v` e toda suíte passou com sucesso em 2.855s, junto com `go vet` para multiplataformas.

## Respostas para Verificações Extras
- **Um alias que colide com o nome de um arquivo real resolve para o arquivo?**
  Sim. Em `resolve.go`, `resolveByName` é testado antes de `resolveByAlias`, garantindo que nomes de nota venzam aliases.
- **Dois aliases iguais em notas diferentes — o que acontece, e a colisão é contável?**
  A função `buildAliasMap` adiciona todos os arquivos correspondentes para um dado alias `ix.byAlias[lower] = append(...)`. Isso mantém o tracking da colisão contável para o `vault_stats`.
- **`[[nota#heading-que-nao-existe]]` marca âncora quebrada mantendo o alvo resolvido?**
  Sim. O estado fica `LinkAnchorMissing` em vez de `LinkTargetMissing`. O arquivo em si (`Resolved`) não fica vazio.
- **`[[nota#^bloco-que-nao-existe]]` idem?**
  Sim. Se o bloco `^...` não existe, o fallback atribui `LinkAnchorMissing`.
- **Um embed para anexo (`![[diagrama.png]]`) resolve pelo caminho de anexo, em vez de virar link quebrado?**
  Sim, `resolveByName` lida com assets forçando match exato com extensão. Ele vira estado `LinkOK` via `ViaAsset`.
- **Resolução insensível a maiúsculas com mais de um candidato devolve erro listando os candidatos?**
  O método `ResolvePath` acusa `ErrAmbiguousPath` (o próprio teste `TestResolvePathCaseInsensitiveAndAmbiguous` cobre divergência).
- **Um link cujo alvo é criado depois passa a resolver quando o índice é atualizado?**
  Links não encontrados usam `ViaNone` e mantêm o nome cru (`Resolved = ""`), permitindo serem revistos no próximo build do índice e resolvidos.

## Arquivos Alterados
- `internal/index/resolve_test.go`
- `internal/index/resolve.go`
- `internal/index/alias.go`
- `internal/index/assets.go`
- `internal/index/anchors.go`
- `internal/index/index.go` (modificado para popular `byName` e remover calls fantasmas)

## Preocupações e Achados
Não há grandes preocupações. A integração com `vault_stats` apenas lerá o campo `ix.byAlias` onde slices > 1 denotam colisões. A modificação de `index.go` foi pequena e contida para inicializar corretamente os mappings `byName`.
