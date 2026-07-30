# Relatório Task 61: A lacuna do RNF-04 — busca por frase

- **Status**: DONE
- **Commit**: `perf(search): bring exact-phrase p95 within RNF-04`

## Resumo das Mudanças e Diagnóstico de Profiling
- **Perfil de CPU (`pprof`)**: Executado `go test -cpuprofile cpu.pprof -run TestRNF04VaultSearchLatencyP95 ./internal/service/`. O relatório indicou `service.(*Service).matchPhraseInNote` consumindo **3,77 s (25,58% de todo o tempo de CPU)**.
- **Causa Raiz Identificada**: A cada candidato a correspondência de frase para um documento, `matchPhraseInNote` chamava `s.inverted.Postings(term)` que alocava e ordenava fatias `[]Posting`, e realizava uma busca linear $O(N)$ sobre todos os documentos do cofre para localizar `p.Path == path`. Com 500 notas e múltiplos termos em 30 iterações, isso gerava ~250.000 iterações de loop e comparações de string desnecessárias.
- **Solução Aplicada**: Criado o método $O(1)$ `Inverted.Positions(term, path []TokenPosition)` em `internal/search/inverted.go`, aproveitando diretamente a tabela hash interna `terms[term][path]`. Atualizada a função `matchPhraseInNote` em `internal/service/search.go` para consultar a lista de posições dos termos em $O(1)$.
- **Resultado Medido**: A latência p95 da busca por frase exata caiu de **174,2 ms (acima do alvo de 100 ms)** para **22,1 ms (redução de 87%)**, colocando o RNF-04 em **100% de conformidade** em todos os 8 formatos.
- Atualizado o teto de teste em `internal/service/search_test.go` para 100 ms.
- Atualizada a documentação em `docs/OPERACAO.md`.

## Tabela Comparativa de Medição RNF-04 (500 notas distintas, 30 amostras por formato)

| Formato | Mediana (Antes) | p95 (Antes) | Mediana (Depois) | p95 (Depois) | Alvo RNF-04 | Status |
|---|---|---|---|---|---|---|
| termo amplo, limit default | 10,2 ms | 14,9 ms | 10,3 ms | 12,9 ms | 100 ms | OK |
| dois termos | 13,4 ms | 17,7 ms | 12,0 ms | 21,6 ms | 100 ms | OK |
| termo seletivo | 3,1 ms | 5,7 ms | 3,4 ms | 8,3 ms | 100 ms | OK |
| filtro de pasta | 10,5 ms | 17,4 ms | 9,6 ms | 13,6 ms | 100 ms | OK |
| filtro de tag | 9,4 ms | 11,9 ms | 9,8 ms | 14,9 ms | 100 ms | OK |
| **frase exata** | **142,7 ms** | **174,2 ms** | **17,3 ms** | **22,1 ms** | **100 ms** | **OK (Redução de 87%)** |
| trecho de 1000 chars | 12,1 ms | 24,5 ms | 11,9 ms | 15,8 ms | 100 ms | OK |
| limit: 200 (máximo schema) | 72,2 ms | 85,0 ms | 62,8 ms | 71,9 ms | 100 ms | OK |

## Evidência de TDD e Prova de Mutação

### RED
Comando:
`go test -v ./internal/service/ -run TestRNF04VaultSearchLatencyP95` (com teto de 100ms no formato frase exata antes da otimização)
Saída:
search_test.go:446: frase exata: p95 = 185.41ms, excede o teto de 100ms deste teste
FAIL

### GREEN
Comando:
`go test -v ./internal/service/ -run TestRNF04VaultSearchLatencyP95` (após introdução do método Positions O(1))
Saída:
search_test.go:436:   frase exata                    mediana 17.2873ms    p95 22.0686ms    teto 100ms    
PASS
ok  	github.com/jonyd/gobsidian/internal/service	5.752s

### Prova de Mutação (Desligar verificação de sequência de frase para simular termos soltos)
Comando:
`pwsh -Command '$a = @"\n\t\tif isPhrase && !s.matchPhraseInNote(hit.Path, queryTokens) {\n\t\t\tcontinue\n\t\t}\n"@; $r = @"\n\t\t// Muted: phrase matching disabled\n"@; pwsh -File scripts/mutate.ps1 -Path internal/service/search.go -Anchor $a -Replacement $r -Test TestVaultSearchPhraseQueryExactSequence -Package ./internal/service/'`
Saída:
--- FAIL: TestVaultSearchPhraseQueryExactSequence (0.01s)
    search_test.go:309: Phrase search failed: res = {Results:[... Total:2 Truncated:false}, quer apenas a.md
FAIL
[OK] internal/service/search.go restaurado byte a byte (SHA-256 confere).

## Arquivos Alterados
- `internal/search/inverted.go`
- `internal/service/search.go`
- `internal/service/search_test.go`
- `docs/OPERACAO.md`
- `.superpowers/sdd/task-61-report.md`

## O Que Ficou de Fora
Nada.
