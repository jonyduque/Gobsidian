# Relatório da Task 27

## O que foi implementado
1. **Unificação da Classificação**: Extraído e exportado enum `Classification` e `Classify()` de `internal/vault/walk.go`.
2. **Camada de Filtro**: Implementado `internal/watcher/filter.go`, delegando a `vault.Classify()`. Filtra eventos ruidosos, Chmod, e restringe à extensão `.md` e extensões de asset (configuradas no vault).
3. **Watcher**: Encapsulamento de `fsnotify.Watcher` (`v1.10.1`) no `internal/watcher/watcher.go`. Oculta os tipos da biblioteca externa e provê domínio com `Event`, `Op`, e laço `Run(ctx)` sensível a cancelamento.
4. **Subdiretórios Dinâmicos (Recursividade Ausente)**: Constatou-se empiricamente que `fsnotify v1.10.1` **não é recursivo no Windows**. A rotina `filepath.WalkDir` e a interceptação de `OpCreate` de diretórios foram adicionadas para expandir a lista de escuta no fsnotify.
5. **Atualização de Documentação**: `docs/WINDOWS.md` §4.1 foi ajustada para refletir que a recursividade padrão não existe no Windows em `fsnotify` atual.

## TDD e Prova de Mutação
- Executado refactoring no `vault` verificando ausência de quebras (Green original).
- Criados testes `TestFilter` (rejeição de Chmod/desconhecidos) e `TestWatcher` (recebimento e Clean Shutdown liberando Handles).
- **Prova de Mutação**: Excluído `".png"` de `assetExts` no `walk.go` gerou falha tanto no pacote `vault` (`walk_test.go`) quanto no pacote `watcher` (`filter_test.go`). Restabelecido.
- **Canal fechado e Clean Shutdown**: `watcher_test.go` provou que o canal é fechado logo em seguida do shutdown do contexto, sem deixar goroutine presa.
- Verificado diretórios com nome ignorado (`.git`) que não geram trigger de watch pela exclusão da varredura inicial `vault.Classify()`.

## Tabela de Verificações

| Verificação | Status Medido |
|---|---|
| fsnotify 1.10.1 é recursivo no Windows? | **NÃO** medido empiricamente via script independente. Requer iteração explícita. |
| Diretório criado depois gera evento? | Sim, interceptado no loop e adicionado ao fsnotify via stat. |
| Chmod descartado? | Sim. Filtrado em `filter.go`. |
| Fechar o watcher libera handles? | Sim. `TestWatcher` testa fechamento liso via cancel e `Close()`. O OS renomeia e apaga normal depois. |
| Canal de eventos fechado no shutdown? | Sim. Validado em teste (Select com timeout após Close e cancel). |
| Evento fora da raiz (links)? | Sim. O `vault.Canonicalize()` recusa paths acima da raiz (erro) e descarta (logado no Warn). |
| `pwsh check_net.ps1` fica verde? | Sim. Dependência exclusiva de `golang.org/x/sys`. |

## Qualidade e Commit
- Código obteve Green em `go test -race ./...`.
- `go vet` verificado local e cross-platform (`GOOS=linux`, `GOOS=darwin`).
- Formatação normalizada com `gofmt -l .`

## Achados e Preocupações
- Como a recursividade não existe por padrão, pastas recém criadas pelo Windows podem gerar duplos eventos: a pasta criada, e logo depois, os arquivos movidos para ela se ela foi movida. A ausência de recursividade natural exige atenção extra na Task 28 e 31 (Rename de diretórios grandes), que podem demandar re-watch.
