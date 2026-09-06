# Re-revisão do fix round 1 — Task 159

Diff revisado: `review-fix1-d622afd..facfe32.diff` (commits `4c08910` + `facfe32`).
Papel: revisor (somente leitura — nenhum arquivo editado, nenhum comando git que
muda estado executado, nenhum subagente usado).

## Progresso

Ressalva igual à do `review-159.md`: só medi `date +%H:%M` uma vez.

- (não medido) Li `review-159.md` achados 1–9 e o diff completo do fix round 1.
- (não medido) Contei `CreateFile(` em `86f07e5` (`git grep`, 8 sítios) e li o
  corpo de cada um dos 8 (5 helpers nomeados + 3 blocos inline) para classificar
  quais conferiam a trava, comparando com a tabela de `review-159.md`.
- (não medido) `git log -1 --format="%H %ad"` em `4c08910` e `facfe32`: ambos
  `2026-09-05 23:0x`, confirmando a data que o achado 7 cita como medida.
- (não medido) Li `internal/vaulttest/exclusivo_other.go` e `somentenuvem_other.go`
  inteiros: todo caminho é `t.Skip`, sem exceção — confirma a frase nova do
  CLAUDE.md.
- (não medido) `GOOS=linux go list -f '{{.Imports}}' ./internal/vaulttest` ->
  `[testing time]`, bate com a frase nova do CLAUDE.md e com o relatório.
- (não medido) `grep -n "elapsed > tetoDesistencia\|tetoDesistencia ="
  internal/ipc/ipc_test.go` -> linhas 58, 71, 242 — bate com o relatório
  (":71 e :242, antes :64 e :235").
- (não medido) `go build ./...`, `go vet ./...`, `gofmt -l internal cmd`,
  `go test ./internal/ipc/...`, `go test -race ./internal/service/ -run
  'MoveNaoReportaSucessoComNotaDuplicada'`: todos verdes.
- (não medido) `python -c "open(...,encoding='utf-8').read()"` em `CLAUDE.md` e
  `docs/papeis/testador.md`: ambos abrem sem exceção.
- **23:07** Escrevi este relatório.

## Achados

1 — ADDRESSED — internal/ipc/ipc_test.go:58,71,242 — `const tetoDesistencia = 2*time.Second`
substitui `vaulttest.Prazo` nas duas asserções de latência, com comentário
distinguindo teto de latência de prazo de espera. `vaulttest.Prazo` permanece
nos 9 usos restantes do arquivo (orçamento de espera de fato). Compila, `go
test ./internal/ipc/...` ok.

3 — ADDRESSED — docs/papeis/testador.md:142-153, internal/vaulttest/doc.go:6-8 —
A nova contagem ("oito sítios: cinco helpers nomeados e três blocos inline;
quatro conferiam — três com `t.Fatal` [`classify_cloudonly`, `cloudonly_update`,
`cloudonly_replace`], um com `t.Skip` [`walk_raiz`]; entre os cinco nomeados só
`travarExclusivo` conferia e falhava, `travarDiretorioExclusivo` conferia e
pulava, os outros três não conferiam nada") foi reconferida por mim lendo o
corpo dos 8 sítios em `86f07e5` — bate exatamente, nome a nome. `doc.go` ficou
curto e aponta para a seção de `testador.md` em vez de duplicar o número — no
espírito de "uma conta por regra".

5 — ADDRESSED — internal/index/build_descarte_unix_test.go:19-24 — prova
simétrica acrescentada (`os.ReadFile` depois do `chmod 0000`, `t.Fatalf` se não
erro), texto igual ao sugerido no achado. Relatório é honesto sobre não ter
conseguido rodar em Linux/macOS nesta máquina Windows (`//go:build !windows`);
registra "não provado por mutação, só compilado" em vez de fingir que testou —
`GOOS=linux go vet ./internal/index/` aqui também só compila, não executa.

6 — ADDRESSED — internal/service/move_atomico_windows_test.go:86-88 — guarda de
montagem agora imprime `MoveNote err=%v` e a frase "se err também foi nil, o
defeito é do MoveNote, não da montagem", como sugerido. `t.Fatalf`/`t.Fatal`
preservados em toda a função (nenhum viraram `t.Error`). `go test -race -run
TestMoveNaoReportaSucessoComNotaDuplicada` passa.

7 — ADDRESSED — CLAUDE.md:126-131 — qualificador de GOOS acrescentado, e a
alegação nele é verificável: `_other.go` só faz `t.Skip` (li os dois arquivos
inteiros) e `GOOS=linux go list -f '{{.Imports}}' ./internal/vaulttest` devolve
`[testing time]`, batendo com o texto. A data "medido em 2026-09-05" bate com o
timestamp real dos commits (`4c08910`/`facfe32`, ambos `2026-09-05 23:0x`).

8 — ADDRESSED, com ressalva de precisão (não bloqueante) — CLAUDE.md:70-71 — a
linha `vaulttest/` foi inserida na árvore, logo depois de `vault/`, como pedido.
Mas o texto inserido — "condicoes de ambiente do Windows; so _test.go importa" —
quebra a acentuação que toda outra linha do mesmo bloco de código usa
(`configuração`, `canônico`, `exclusões`, `detecção`, `extensões` têm acento; a
linha nova de `vaulttest/` não tem em "condicoes" nem "so"). Não é regressão
funcional — `check_doc_refs.ps1` e a checagem UTF-8 continuam verdes, e o
próprio achado 8 já sugeriu o texto sem acento — mas é o tipo de inconsistência
que `CLAUDE.md` chama de "duas cópias do mesmo fato" quando um documento
normativo tem uma linha fora do padrão das vizinhas. Sugiro só um ajuste de
redação ("condições", "só") num commit futuro; não reabre o achado.

Verdict: APPROVED
