# Task 173 report

## Progresso
- 08:30 Step 1: grep `service\.Index\b\|Index interface\|fakeIndex` -> so a definicao em service.go:11. Nenhum fake. Sem BLOCKED.
- 08:32 Steps 2-3 feitos (service.go: interface apagada, campo/New tipados *index.Index; search.go: assercao removida, s.index direto em CalculateBM25/GenerateSnippet). `go build ./...` falha:
  `internal\service\frontmatter_err_test.go:7:2: "github.com/jonyd/gobsidian/internal/index" imported and not used`
  `internal\service\frontmatter_err_test.go:55:12: invalid operation: svc.index (variable of type *index.Index) is not an interface`
  BLOCKED: internal/service/frontmatter_err_test.go:55 faz `ix, ok := svc.index.(*index.Index)` para acessar `ix.Replace(...)` (metodo fora da interface antiga). Nao e um fake da interface (nao ha `type fakeIndex`, newTestService monta indice real) — e uma assercao de conveniencia para alcancar o tipo concreto, que a mudanca estrutural torna invalida. Aguardando ruling do orquestrador antes de tocar em arquivo de teste.
- 08:41 Ruling do orquestrador aplicada: frontmatter_err_test.go:55 nao e fake, e assercao de conveniencia. Trocado `ix, ok := svc.index.(*index.Index)` + `t.Fatal` por `ix := svc.index`; import `internal/index` removido (nao havia outro uso no arquivo, confirmado por grep). `go build ./...` e `go vet ./...` limpos.
- 08:51 Step 3 test run (rodou em background por exceder 120s foreground): `go test -race ./internal/service/ ./internal/mcpsrv/ ./internal/daemon/ ./cmd/...` -> ok service 43.864s, ok mcpsrv 10.368s, ok daemon 7.227s, ok cmd/gobsidian 6.768s (exit 0).
- 08:51 Step 4: `go test -race ./internal/service/ -run 'Nil|Unavailable|Indisponivel' -v` -> PASS TestErrorIsTypedNilTarget, PASS TestServicoSemIndiceDevolveVaultUnavailable. ok 1.753s.
- 08:55 Step 5: antes173_service.test.exe (pre-edit, buildado as 08:31) x depois173_service.test.exe (pos-edit), `-bench 'SearchDoisTermos|SearchFraseExata|SearchLimit200Cache|SearchTermoAmploCache' -benchmem -count=7` contra %TEMP%\vault_5000, benchstat: `~` em sec/op, B/op e allocs/op em todos os 5 benchmarks (SearchLimit200CacheTrechoRepetido tambem casou no filtro). geomean sec/op +1.40% (ruido, todos p>0.3). IndexBuild: `git diff --stat -- internal/index/` vazio para esta task — internal/index nao mudou nenhuma linha, entao nao ha novo binario para comparar; a comparacao antes172/depois172_index.test.exe ja existente (Task 172) segue documentando que o pacote index e independente desta mudanca.
- 08:56 Step 6: docs/ESTADO.md:157 atualizado (frase da Task 173 acrescentada apos os numeros medidos, numeros marcados "medido antes da Task 173" para nao alegar que a interface ainda existe). UTF-8 validado.
- 09:01 Step 7: `pwsh -File scripts/verify.ps1` completo, 14 etapas, foreground. Ultima linha: `[OK] Bateria completa. Pode commitar.` (6 skips reportados sao os conhecidos/legitimos de plataforma, contagem informativa, nao reprova).
- 09:03 Commit: `git add internal/service/service.go internal/service/search.go internal/service/frontmatter_err_test.go docs/ESTADO.md` (caminhos explicitos, sem -A/.), `git commit -F commit-173.txt`. SHA c1ef235da51893c897721e4ab19053380a120701. `git status --short` pos-commit mostra so o dirty state pre-existente do dono (.superpowers progress.md, test-vault/*), nada da task ficou solto.

## Status
Concluida. SHA: c1ef235da51893c897721e4ab19053380a120701

## TDD/RED, TDD/GREEN
Nao se aplica. Task 173 e estrutural pura (apagar interface, tipar campo/parametro,
remover assercao) — nenhum comportamento novo, nenhum teste novo. O brief (Step 3)
so pede `go build`/`go vet`/suite existente, sem ciclo RED/GREEN.

## Mutacao
O brief e explicito: "Esta tarefa nao tem prova de mutacao: e estrutural, e a prova
e go build, go vet, os testes da Task 158 e o benchstat com ~." As quatro provas:
- `go build ./...` e `go vet ./...`: limpos (Progresso 08:41).
- Testes: service/mcpsrv/daemon/cmd todos ok (Progresso 08:51); os dois testes
  da Task 158 (`TestErrorIsTypedNilTarget`, `TestServicoSemIndiceDevolveVaultUnavailable`)
  PASS com a assinatura nova (Progresso 08:51).
- `benchstat`: `~` em sec/op, B/op, allocs/op nos 5 benchmarks de busca (Progresso 08:55).

## Verificacao
- Step 1 grep (`service\.Index\b\|Index interface\|fakeIndex`): so a definicao, sem fake.
- `go build`/`go vet`: limpos.
- `go test -race` nos 4 pacotes downstream: PASS.
- Testes da Task 158: PASS.
- `benchstat`: `~` em todos os benchmarks.
- `pwsh -File scripts/verify.ps1`: `[OK] Bateria completa. Pode commitar.` (14/14 etapas).
- Grep final (`service\.Index\b\|idxImpl`): vazio.
- Commit por caminho explicito; `git status` pos-commit sem sobra da task.

## Grep final
`grep -rn "service\.Index\b\|idxImpl" --include=*.go .` -> vazio.

## Concerns
- `scripts/audit_reports.ps1 173` reclama de secoes ausentes (TDD/RED, TDD/GREEN,
  Mutacao, Verificacao) porque o script e generico para tarefas com TDD; esta task
  e estrutural e o proprio brief dispensa prova de mutacao. Secoes acrescentadas
  acima explicando o porque em vez de forjar conteudo que nao existe.
- Os demais achados do `audit_reports.ps1` (SHA-NAO-CONFERE e RELATORIO-AUSENTE) sao
  todos no ledger de outro plano (`2026-07-25-gobsidian-v01/progress.md`, tasks 4, 6,
  94-103, 153) e pre-existentes a esta sessao — nao pertencem a Task 173, nao mexidos.
- Nenhum outro ponto em aberto.

## Round 1 (orquestrador) — tabela do benchstat colada (review-173 N1)

Rodada pelo orquestrador as 09:08 sobre os arquivos brutos que o Step 5 gerou (`antes173_service.txt` x `depois173_service.txt`, n=7 intercalados):

```
goos: windows
goarch: amd64
pkg: github.com/jonyd/gobsidian/internal/service
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
                                     │ antes173_service.txt │       depois173_service.txt        │
                                     │        sec/op        │    sec/op     vs base              │
SearchLimit200Cache-12                         11.88m ±  4%   11.99m ±  3%       ~ (p=0.456 n=7)
SearchTermoAmploCache-12                       5.295m ± 25%   5.174m ±  6%       ~ (p=0.456 n=7)
SearchLimit200CacheTrechoRepetido-12           4.659m ± 27%   4.759m ± 14%       ~ (p=0.805 n=7)
SearchDoisTermos-12                            3.119m ±  3%   3.136m ±  8%       ~ (p=0.535 n=7)
SearchFraseExata-12                            14.93m ±  6%   15.79m ± 12%       ~ (p=0.383 n=7)
geomean                                        6.714m         6.808m        +1.40%

                                     │ antes173_service.txt │       depois173_service.txt        │
                                     │         B/op         │     B/op      vs base              │
SearchLimit200Cache-12                         1.921Mi ± 0%   1.921Mi ± 0%       ~ (p=0.318 n=7)
SearchTermoAmploCache-12                       1.493Mi ± 0%   1.493Mi ± 0%       ~ (p=0.535 n=7)
SearchLimit200CacheTrechoRepetido-12           1.641Mi ± 0%   1.641Mi ± 0%       ~ (p=0.535 n=7)
SearchDoisTermos-12                            436.9Ki ± 0%   436.8Ki ± 0%       ~ (p=0.209 n=7)
SearchFraseExata-12                            4.834Mi ± 0%   4.834Mi ± 0%       ~ (p=0.805 n=7)
geomean                                        1.575Mi        1.575Mi       -0.01%

                                     │ antes173_service.txt │       depois173_service.txt       │
                                     │      allocs/op       │  allocs/op   vs base              │
SearchLimit200Cache-12                          10.14k ± 0%   10.14k ± 0%       ~ (p=0.786 n=7)
SearchTermoAmploCache-12                        5.579k ± 0%   5.579k ± 0%       ~ (p=0.700 n=7)
SearchLimit200CacheTrechoRepetido-12            7.496k ± 0%   7.496k ± 0%       ~ (p=1.000 n=7)
SearchDoisTermos-12                              607.0 ± 0%    607.0 ± 0%       ~ (p=0.559 n=7)
SearchFraseExata-12                             21.85k ± 0%   21.85k ± 0%       ~ (p=0.506 n=7)
geomean                                         5.624k        5.624k       -0.00%
```
