# Task 21 Report

## O que foi implementado
- `Paths() []vault.CanonicalPath` no índice para ter ordem determinística.
- `backlinks_test.go` com `TestBacklinkInvariantUnderMutation` testando a propriedade na direção 1 (link -> backlink) e 2 (backlink -> link) e sofrendo alterações no cofre simulado.
- `backlinks.go` com `buildBacklinks()` e `Backlinks()` para obter e preencher os backlinks inicialmente.
- `update.go` com `Replace` e `Remove`. Estas operações corretamente removem a contribuição velha (inclusive de backlinks) e executam a reavaliação global de links (`reprocessLinksLocked`), porque um alvo removido quebra links, e um criado pode consertá-los.

## Evidência TDD
- **RED**: `go test ./internal/index/ -run TestBacklink -v` acusou vários erros `link X -> Y sem backlink correspondente`.
- **GREEN**: `go test -race ./internal/index/ -v` e todas as validações de gates. Todos passaram sem warning ou erro.

## Respostas às verificações extras (Resultados reais)
- *Depois de editar uma nota para remover um link, o backlink correspondente desaparece?* 
  Sim. `TestBacklinkInvariantUnderMutation` altera `n05.md` para um arquivo sem links, e o teste na direção 2 valida que nenhum backlink fantasma resta no índice, pois `Replace` apaga a contribuição anterior.
- *Depois de remover uma nota, os links que apontavam para ela viram alvo inexistente?*
  Sim. `Remove` reavalia todos os links, e eles passam a ter o estado `LinkTargetMissing`, e os backlinks associados são varridos do sistema.
- *Depois de recriar essa nota, eles voltam a resolver?*
  Sim. `Replace` para uma nova nota reavalia os links de todo o cofre e restaura a referência para a recém-criada nota.
- *Duas notas apontando para a mesma terceira produzem dois backlinks distinguíveis pela origem?*
  Sim. Ambas as entradas entram no slice `ix.backlinks[target]`, distinguíveis pelo campo `From`.
- *Uma nota que aponta para si mesma — o que acontece?*
  Isso resulta em um backlink onde `From` e `target` são o mesmo. É mantido e atualizado perfeitamente, já que é idêntico a qualquer outra conexão.
- *Prove que o teste de propriedade pega o defeito:*
  Ao comentar a limpeza de backlinks em `removeContributionsLocked` de `update.go`, a invariante 2 encontrou o erro, produzindo saídas falhas do tipo:
  `backlinks_test.go:41: backlink fantasma: n05.md -> n06.md`
  Após descomentar a limpeza, os testes voltaram ao verde.

## Arquivos Alterados
- `internal/index/index.go`
- `internal/index/update.go` (create)
- `internal/index/backlinks.go` (create)
- `internal/index/backlinks_test.go` (create)

## Correções Mecânicas
- Nenhuma além de importar o pacote nativo `slices` pois `Index.Paths()` o requisitava para ordenar `paths`.
- Em `update.go`, como `v.Stat` não existia no cofre, utilizei o pacote local `os.Stat(v.Abs(path))` combinando com `vault.IsCloudOnly(abs)` para recriar localmente os dados que deveriam ser da struct Entry.
