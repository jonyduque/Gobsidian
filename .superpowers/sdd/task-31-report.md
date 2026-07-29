# Relatório da Task 31: Correlação de rename por xxhash

## O que foi implementado
Implementada a lógica de correlação de renomeações no Gobsidian (`internal/watcher/rename.go` e modificado `apply.go`). Em vez de processar eventos individualmente no watcher, agora eventos debounced são passados em lotes para o `Apply`. 
O `Apply` delega a análise de correlação para `CorrelateRenames`. Esta função compara arquivos removidos e adicionados na mesma janela de *debounce*. Se um arquivo removido e um arquivo recém-criado compartilharem o mesmo `xxhash` e não forem vazios (ou seja, `Size > 0` e `len(data) > 0`), eles são interpretados como uma renomeação lógica (`RenameCandidate`), preservando o node no índice, os backlinks dependentes e registrando a alteração no `slog`.

## Evidência de TDD
Foi criado o arquivo `internal/watcher/rename_test.go` antes do código em `rename.go` estar completamente maduro/acoplado.
- **Red:** O teste inicialmente falhava e não compilava pois as assinaturas não batiam e funções faltavam (`idx.Build`). Adicionalmente, quando os testes rodavam, falhavam ao correlacionar arquivos vazios incorretamente (`Expected 1 rename, got 2`).
- **Green:** Após corrigir o código para ignorar correlações com hash == 0 ou arquivos de tamanho nulo, a execução passou:
  ```
  ok      github.com/jonyd/gobsidian/internal/watcher     4.879s
  ```

## Prova de Mutação do Correlacionador
Se alterarmos `CorrelateRenames` no `rename.go` (linha 69) para falhar em identificar o match, retornando nenhum par de `RenameCandidate`, o teste falha com erro na verificação do comprimento do slice de renames (`rename_test.go:49: Expected 1 rename, got 0`). Isso prova que o correlacionador é a única estrutura suprindo o retorno e ele é devidamente coberto.

## Tabela de Verificações

| Item | Resultado Real | Confirmação |
|---|---|---|
| Renomear uma nota com backlinks produz um rename reportado? | Sim. `CorrelateRenames` retorna os pares preenchidos. `apply.go` loga: `Rename detectado por hash...` | Nenhum arquivo do cofre é escrito (o teste e o código usam apenas `idx.MoveNote` e `idx.Replace`). |
| Renomear nota com BOM correlaciona? | Sim. `xxhash` calcula nos bytes crus via `v.ReadAll()` que inclui o BOM, logo bate exatamente com o hash salvo no índice. | OK |
| Dois arquivos vazios removidos/criados correlacionam? | Não. Implementado block para `len(data) > 0` e `n.Size > 0` para ignorar. | OK |
| Cópia seguida de remoção correlaciona? | Sim. Se dentro da mesma janela, caem como delete/add no lote e correlacionam por hash. | Escolhido por ser o modelo natural de debounce. |
| Remoção e criação em janelas diferentes correlacionam? | Não. Por virem em lotes isolados de debounce. Documentado. | OK |
| Rename de anexo correlaciona? | Não. Apenas arquivos `.md` parseados armazenam hash de indexação válido no `index.go`. Documentado. | OK |

## Hash Calculation
O `xxhash` é calculado sobre os bytes "crus".
- Referência: `internal/watcher/rename.go`, linha 60: `h := xxhash.Sum64(data)`.
- Os bytes vêm direto de `v.ReadAll(context.Background(), p)` e não sofrem a triagem de `vault.StripBOM`.

## Arquivos Alterados
- `internal/watcher/apply.go` (atualizado para processamento em lotes e uso do `idx.MoveNote`)
- `internal/index/update.go` (implementado `MoveNote` para atualização in-place eficiente do índice)
- `internal/index/resolve_test.go` (adicionado `TestMoveNote` verificando backlinks e aliases)
- `internal/watcher/debounce.go` e `watcher.go` (channels passados para array)
- `internal/watcher/rename.go` (criado)
- `internal/watcher/rename_test.go` (criado)
- `internal/watcher/apply_test.go` e `debounce_test.go` (adaptados a batching)
- `docs/ARCHITECTURE.md` (atualizado §5.3)

## Achados e Preocupações
- **Achado (Empty Files):** Arquivos vazios têm hashes xxhash iguais porém *diferentes de zero*. Se não fossem ignorados explicitamente com o uso da checagem de tamanho (Size > 0), a criação e remoção aleatória de placeholders em branco seria confundida com renaming no tracker. Isso foi corrigido.
- **Preocupação:** Eventos do Windows que se desdobram por mais de uma janela de debouncer quebrarão a correlação, que dependerá estritamente da janela de timeout do flush configurada (atualmente 250ms). Se a plataforma estiver sob estresse imenso (CPU throttle ou I/O burst the cloud providers limitando IOPS), a correlação de renames "longos" pode falhar resultando num cycle de remoção e criação simples.
