# Revisão da Task 159 — `internal/vaulttest`

Pacote revisado: `review-86f07e5..5377559.diff` (commits `c81b2b8` + `5377559`).
Papel: revisor (somente leitura — nenhum arquivo de código foi editado, nenhum
comando git que muda estado foi executado).

---

## Verificações do brief, rodadas por mim

Não reaproveitei os greps do relatório; rodei os três, nesta árvore, agora:

```
$ grep -rn "CreateFile(" --include=*_test.go internal/ cmd/
(vazio, exit=1)

$ grep -rn "boundedWait" --include=*.go .
(vazio, exit=1)

$ grep -rn "FILE_ATTRIBUTE_OFFLINE" --include=*_test.go internal/ cmd/
internal/index/build_cloudonly_windows_test.go:27:// FILE_ATTRIBUTE_OFFLINE e gravavel por SetFileAttributes e vault.IsCloudOnly
internal/search/snippet_cache_windows_test.go:21:// A prova é possível porque vault.IsCloudOnly aceita FILE_ATTRIBUTE_OFFLINE, e
internal/vault/cloudonly_info_windows_test.go:25:// FILE_ATTRIBUTE_OFFLINE é gravável por SetFileAttributes e entra na mesma
internal/vault/cloudonly_info_windows_test.go:41:	if err := windows.SetFileAttributes(pw, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
internal/vault/cloudonly_info_windows_test.go:42:		t.Skipf("nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE: %v", err)
internal/vault/cloudonly_info_windows_test.go:49:		t.Fatal("IsCloudOnly nao viu o FILE_ATTRIBUTE_OFFLINE: o cenario nao se montou")
internal/vault/cloudonly_info_windows_test.go:81:	if err := windows.SetFileAttributes(pw, windows.FILE_ATTRIBUTE_OFFLINE); err != nil {
internal/vault/cloudonly_info_windows_test.go:82:		t.Skipf("nao foi possivel marcar FILE_ATTRIBUTE_OFFLINE: %v", err)
```

Zero `CreateFile(`, zero `boundedWait`. As oito ocorrências de
`FILE_ATTRIBUTE_OFFLINE` são as que o brief manda manter: duas em comentário e
seis em `internal/vault/cloudonly_info_windows_test.go`, que é `package vault`
(interno) e fecharia ciclo se importasse `vaulttest`. Bate exatamente com o que o
relatório declarou.

Grafo de imports, rodado por mim:

```
$ go list -deps ./internal/vaulttest/ | grep gobsidian
github.com/jonyd/gobsidian/internal/vault
github.com/jonyd/gobsidian/internal/vaulttest

$ go list -f '{{.Imports}}' ./internal/vaulttest/
[github.com/jonyd/gobsidian/internal/vault golang.org/x/sys/windows os testing time]
```

Nenhum `index`, `search`, `service`, `parser`, `mcpsrv` — a restrição de ciclo é
cumprida. E nenhum arquivo de produção importa `vaulttest`:

```
$ grep -rln "internal/vaulttest" --include=*.go . | grep -v "_test.go"
(nenhum)
```

Build tags, arquivo por arquivo: `doc.go` e `prazo.go` sem tag (corretos, são
cross-platform), `exclusivo_windows.go` / `somentenuvem_windows.go` /
`exclusivo_windows_test.go` com `//go:build windows`, `exclusivo_other.go` /
`somentenuvem_other.go` com `//go:build !windows`. Nenhum `if runtime.GOOS ==`
introduzido. Nenhum `helpers.go`/`utils.go`/`common.go`. Nenhum `net/*`
(`check_net.ps1` verde, ver abaixo). Saída de console não muda.

Rodado por mim, além disso:

```
$ gofmt -l internal/vaulttest/          -> (vazio)
$ GOOS=linux  go vet ./...              -> exit 0, saida vazia
$ GOOS=darwin go vet ./...              -> exit 0, saida vazia
$ golangci-lint run ./internal/vaulttest/...
0 issues.
$ pwsh -File scripts/check_doc_refs.ps1
[OK] nenhum token entre crases parece citar artefato ausente do codigo.
$ pwsh -File scripts/check_net.ps1
[OK] Nenhum pacote de internal/ ou cmd/ importa net/* ou abre socket que saia da
     maquina (verificado via netcheck vettool em windows, linux, darwin)
```

SHA conferido (revisor.md §3): `c81b2b8` existe, e o stat bate com o relatório
linha a linha — `29 files changed, 775 insertions(+), 338 deletions(-)`.

## Provas de mutação — verificadas

`go test -race ./internal/vaulttest/ -v`, rodado por mim:

```
=== RUN   TestTravarExclusivoBarraLeituraEDevolveNoCleanup
=== RUN   TestTravarExclusivoBarraLeituraEDevolveNoCleanup/travado
--- PASS: TestTravarExclusivoBarraLeituraEDevolveNoCleanup (0.00s)
    --- PASS: TestTravarExclusivoBarraLeituraEDevolveNoCleanup/travado (0.00s)
=== RUN   TestTravarDiretorioExclusivoBarraListagem
=== RUN   TestTravarDiretorioExclusivoBarraListagem/travado
--- PASS: TestTravarDiretorioExclusivoBarraListagem (0.00s)
    --- PASS: TestTravarDiretorioExclusivoBarraListagem/travado (0.00s)
PASS
ok  	github.com/jonyd/gobsidian/internal/vaulttest	1.638s
```

`go test -race ./internal/service/ -run 'Move|Engol|Delete' -v` (cauda):

```
--- PASS: TestDeleteToTrashNaoMenteQuandoORemoveFalha (0.02s)
--- PASS: TestDeleteNoteToTrashNaoBaixaPlaceholder (0.01s)
--- PASS: TestMoveDryRunNaoApresentaDiffVazioComoResultado (0.01s)
--- PASS: TestMoveNaoReportaSucessoComNotaDuplicada (0.01s)
--- PASS: TestMoveNaoReescreveCitantesAntesDeMoverOCorpo (0.01s)
...
ok  	github.com/jonyd/gobsidian/internal/service	2.952s
```

Não repeti as mutações do relatório (papel somente-leitura, sem editar fonte),
mas confiro que as saídas coladas são **internamente consistentes** com a árvore
atual, que é o que uma saída fabricada raramente é:

- Prova 2a cita `move_atomico_windows_test.go:78`. O `t.Fatalf` da guarda de
  montagem está hoje em `move_atomico_windows_test.go:82`; remover as cinco
  linhas do `os.Open` (58–62) e acrescentar um marcador dá exatamente 78.
- Prova 2b cita `move_atomico_windows_test.go:121`. O `t.Fatal("MoveNote devolveu
  nil com a origem travada para leitura")` está hoje em `:120`; comentar a
  chamada com um marcador `// MUTACAO` acrescenta uma linha e dá 121.
- A Prova 1 (share 0 → `FILE_SHARE_READ`) derrubar os **dois** testes, inclusive
  o de diretório, é o comportamento correto do Win32 e não o que alguém inventaria.

**Uma checagem que fiz sozinho e que ninguém pediu**: se `MarcarSomenteNuvem` ou
`TravarExclusivo` passassem a `t.Skip` na migração, os testes migrados
sumiriam em silêncio — que é o defeito que esta tarefa existe para remover, com
outra roupa. Rodei os quatro pacotes e contei os pulos:

```
$ go test ./internal/vault/ ./internal/index/ ./internal/search/ ./cmd/gobsidian/ -v \
    | grep -E "^(---|    ---) (SKIP|FAIL)" | sort | uniq -c
      1 --- SKIP: TestPerfilDeHeapServindo (0.00s)
```

Um único SKIP, e ele não tem relação com esta tarefa. Nenhum teste migrado
passou a pular.

---

## Achados

### 1. NON-BLOCKING — `internal/ipc/ipc_test.go:64` e `:235` — `Prazo` usado como teto de latência, não como prazo de espera

`Prazo` está documentado em `internal/vaulttest/prazo.go:5` como "o unico limite
de espera dos testes que aguardam algo assincrono". Nestes dois pontos ele não é
um prazo de espera: é o **teto asserido** de quanto a função pode demorar.

```go
// ipc_test.go:57-65
conn, err := ipc.DialAndHandshake(context.Background(), vault, false, 0, 200*time.Millisecond)
elapsed := time.Since(start)
...
if elapsed > vaulttest.Prazo {   // 5 s, contra um orçamento de 200 ms

// ipc_test.go:235
if elapsed > vaulttest.Prazo {
    t.Fatalf("DialAndHandshake com context ja cancelado demorou %s, esperado retorno imediato", elapsed)
```

A asserção de `:64` compara o decorrido contra 5 s quando o orçamento que o
próprio teste passou é 200 ms — 25x de folga. A de `:235` diz "retorno imediato"
e aceita 5 s. As duas ainda reprovam num travamento sem fim, então nenhuma
cobertura se perde de vez; o que se perde é a capacidade de pegar uma regressão
que desrespeite o orçamento por uma ordem de grandeza.

Registro também que a decisão do orquestrador que fecha este ponto ("um limite
maior só alonga um teste que FALHA — custo se errado: rodadas vermelhas mais
lentas") **não vale nestes dois sítios**: aqui o limite maior afrouxa a asserção,
não alonga a espera. Não estou reabrindo a decisão — a troca já está feita e o
custo é pequeno —, só nomeando que o modelo de custo dela não cobria asserção.

**Fix:** trocar as duas por um teto próprio, derivado do que o teste pede.
Em `:64`, `const tetoDesistencia = 2 * time.Second` (ou `10 * 200ms`) local ao
teste, com um comentário dizendo que é teto de latência e não prazo de espera; em
`:235`, o mesmo. `Prazo` fica para os `context.WithTimeout` e `time.After`, que é
o que ele diz ser.

### 2. NON-BLOCKING — `task-159-report.md` D5 — o desvio é subnotificado: são duas asserções, não uma

D5 diz: *"E uma assercao de tempo real, nao so um limite de espera — foi a unica
que encontrei nessa categoria."* Medido por mim com
`grep -n "vaulttest.Prazo" internal/ipc/ipc_test.go`: há **duas** — `:64`
(`TestDialAndHandshakeSocketAusente`, a declarada) e `:235`
(`TestDialAndHandshakeRespeitaContext`, não declarada). A segunda também era 3 s
e também subiu para 5 s.

O desvio continua sendo desvio declarado, não silencioso; o que está errado é a
contagem dentro dele. Importa porque quem for consertar o achado 1 usando D5 como
mapa conserta metade.

**Fix:** corrigir D5 para "duas asserções, `ipc_test.go:64` e `:235`".

### 3. NON-BLOCKING — `docs/papeis/testador.md:142` e `internal/vaulttest/doc.go:8` — "cinco cópias e só uma conferia" é ambíguo, e a contagem do próprio relatório o contradiz

`testador.md:142` diz: *"Isso veio de um defeito medido: existiam cinco cópias do
'handle exclusivo' e **só uma** conferia que a trava travava."* `doc.go:8` repete
a mesma frase. O texto veio literal do brief.

Medido por mim no commit-base `86f07e5`:

```
$ git grep -n "CreateFile(" 86f07e5 -- '*_test.go'   -> 8 sítios
$ (por arquivo, quem conferia a trava)
internal/search/cloudonly_update_windows_test.go   prova=1
internal/service/erro_engolido_windows_test.go     prova=0
internal/index/replace_duas_fases_windows_test.go  prova=0
internal/index/build_descarte_windows_test.go      prova=0
internal/index/classify_cloudonly_windows_test.go  prova=1
internal/service/move_atomico_windows_test.go      prova=0
internal/service/cloudonly_replace_windows_test.go prova=1
internal/vault/walk_raiz_windows_test.go           prova=1
```

**Oito** cópias do padrão, **quatro** com conferência (três reprovando, uma —
walk_raiz — pulando). A frase só fica verdadeira sob a leitura estreita "cinco
helpers **nomeados**, e entre eles só `travarExclusivo` conferia" — que é
defensável, e por isso este achado não bloqueia. Mas o próprio relatório fecha
com *"8 helpers nomeados + 3 constantes boundedWait + 6 blocos inline = 17 copias
colapsadas"*, no mesmo commit que escreve "cinco" num documento normativo de
papel. Duas contas da mesma coisa em dois arquivos do mesmo commit é o que
`CLAUDE.md` chama de duas cópias do mesmo fato.

**Fix:** em `testador.md:142`, trocar por: *"existiam oito sítios de handle
exclusivo — cinco helpers nomeados e três blocos inline — e só quatro conferiam
que a trava travava; entre os cinco nomeados, só um."* Ajustar `doc.go:8` para a
mesma conta ou tirar o número de lá e apontar para o documento.

### 4. NON-BLOCKING — `internal/vaulttest/exclusivo_windows.go:47` — `abrirExclusivo` pula onde dois chamadores antes reprovavam

`abrirExclusivo` faz `t.Skipf` quando o `CreateFile` falha. É o que o brief
prescreve, e é o que quase todas as cópias originais faziam. Duas não:

- `internal/vault/walk_raiz_windows_test.go` (base): `t.Fatalf("CreateFile
  exclusivo em %q: %v", dir, err)`.
- `internal/index/build_descarte_windows_test.go` (base): `t.Fatalf("CreateFile: %v", err)`.

Para esses dois testes, um `CreateFile` que falhe passou de falha nomeada a pulo
silencioso. A prova que importa — o `ReadFile`/`ReadDir` que **passa** apesar da
trava — continua `t.Fatalf`, então a tese da tarefa está intacta; o que mudou é o
caso "nem consegui montar a trava".

Verificado que hoje ninguém pula (ver a contagem de SKIP acima), então isto é
risco de ambiente, não regressão observada.

**Fix (opcional):** deixar como está e registrar, ou dar a `abrirExclusivo` um par
`TravarExclusivoObrigatorio` que reprova em vez de pular. Recomendo registrar e
não complicar o pacote: o pulo é honesto e nomeia o motivo.

### 5. NON-BLOCKING — `internal/index/build_descarte_unix_test.go:13` — o lado POSIX não prova nada

D2 está certo em manter o lado POSIX (`TravarExclusivo` pula fora do Windows, e
migrar perderia a cobertura de `TestBuildRegistraArquivoIlegivel` em Linux e
macOS). Mas o helper POSIX resultante faz `os.Chmod(0000)` e devolve **sem
conferir**:

```go
func lockFileForTest(t *testing.T, path string) {
	t.Helper()
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0644) })
}
```

Rodando como root — que é o padrão em vários contêineres de CI — `chmod 0000` não
impede a leitura, `Build` indexa dois arquivos legíveis e
`TestBuildRegistraArquivoIlegivel` passa sem exercitar nada. É literalmente o
defeito que esta tarefa remove do lado Windows, sobrevivendo do lado POSIX, no
mesmo par de arquivos.

**Fix:** acrescentar a prova simétrica, três linhas:

```go
	if _, err := os.ReadFile(path); err == nil {
		t.Fatalf("chmod 0000 nao barrou a leitura de %s (rodando como root?); "+
			"a prova de 'arquivo ilegivel' seria vazia", path)
	}
```

### 6. NON-BLOCKING — `internal/service/move_atomico_windows_test.go:81` — uma das três asserções originais virou "cenário inválido"

O relatório diz: *"As duas assercoes originais eram a mesma implicacao (`err ==
nil` ⇒ defeito) e ambas sobrevivem."* Eram três condicionais, não duas, e a
terceira não era a mesma implicação:

```go
if err == nil && origemExiste { t.Error("MoveNote devolveu nil mas a origem continua no disco") }
```

Ela cobria o estado `err == nil && origemExiste && !destinoExiste` — que hoje cai
no `t.Fatalf("cenario invalido: …")` da guarda de montagem. O teste **ainda
reprova** nesse estado, então nenhuma cobertura se perde; o que se perde é a
mensagem, que passa a culpar a montagem do cenário em vez de nomear o defeito do
produto. Numa ocorrência futura, quem ler "cenário inválido" investiga o handle,
não o `MoveNote`.

**Fix:** deixar como está (o teste reprova, e reprovar é o que importa) ou
acrescentar ao `t.Fatalf` da guarda uma frase do tipo "…; se `err` também foi nil,
o defeito é do `MoveNote`, não da montagem", passando `err` na mensagem.

### 7. NON-BLOCKING — `CLAUDE.md:126-128` — a afirmação sobre `go list` só vale em Windows

O bloco novo diz que a aresta `vaulttest → vault` *"é do próprio pacote, então
`go list -f '{{.Imports}}' ./internal/vaulttest` a mostra como importaria qualquer
outra"*. Medido por mim:

```
$ go list -f '{{.Imports}}' ./internal/vaulttest/            (windows)
[github.com/jonyd/gobsidian/internal/vault golang.org/x/sys/windows os testing time]

$ GOOS=linux go list -f '{{.Imports}}' ./internal/vaulttest/
[testing time]
```

Fora do Windows a aresta **não aparece** — `exclusivo_other.go`,
`somentenuvem_other.go` e `prazo.go` não importam `vault`. Num arquivo cuja regra
declarada é que "o parágrafo que descreve o grafo não vale mais que os imports", a
receita citada tem de dizer em que GOOS ela vale.

**Fix:** acrescentar "(em `GOOS=windows`; fora dele o pacote não importa `vault`,
porque os arquivos `_other.go` só pulam)" à frase.

### 8. NON-BLOCKING — `CLAUDE.md:60-88` — a árvore de `internal/` do CLAUDE.md não ganhou `vaulttest`

O bloco "Estrutura do projeto" de `CLAUDE.md` enumera todo pacote sob `internal/`,
inclusive folhas como `text/`. `internal/vaulttest/` entrou em
`docs/ESTRUTURA.md` (corretamente, e a entrada está boa) mas não ali, então a
lista do índice ficou incompleta. O brief não pedia — Step 6 só nomeia o bloco do
grafo — e "uma conta por regra" argumenta a favor de não duplicar a árvore. Fica a
critério do orquestrador.

**Fix (se quiser):** uma linha em `CLAUDE.md`, depois de `vault/`:
`vaulttest/  apoio a teste: condições de ambiente do Windows; só _test.go importa`.

### 9. NON-BLOCKING (fora do escopo, carregado do relatório) — `CLAUDE.md:222` diz "14 etapas"; `verify.ps1` tem 13

O relatório levantou e não investigou. Medi:
`grep -nE '^\s*Invoke-Step' scripts/verify.ps1` devolve **13** chamadas (go build,
go test -race, tetos de latência, go vet ×3, gofmt, golangci-lint ×2, check_net,
check_tool_params, check_doc_refs, check_readme_anchors). A saída do gate colada
no relatório numera 13. `CLAUDE.md` é que está errado, e é anterior a esta tarefa.

**Fix:** trocar "14 etapas" por "13 etapas" em `CLAUDE.md:222` e na tabela de
Comandos, num commit de doc à parte.

---

## Julgamento dos pontos que o orquestrador mandou julgar

**(a) D1 — o Step 4 nomeava uma linha que não existe. A prova substituta é
adequada? Manter `os.Open` no cenário da nota duplicada é correto?**

Sim para os dois, e o segundo é a parte que importa.

O brief mandava comentar `vaulttest.TravarExclusivo(t, destino)` em
`move_atomico_windows_test.go`. Não há trava sobre o destino nesse arquivo; as
duas travas são sobre a origem, e com semânticas deliberadamente diferentes.
Manter `os.Open` em `TestMoveNaoReportaSucessoComNotaDuplicada` está **certo, e
migrar teria destruído o teste**: `os.Open` do Go abre com
`FILE_SHARE_READ|FILE_SHARE_WRITE` e sem `FILE_SHARE_DELETE`, então a leitura da
origem passa (a cópia chega ao destino) e o `os.Remove` falha — que é
exatamente a nota nos dois caminhos. Com `TravarExclusivo` (share 0), o
`os.ReadFile` da origem falharia antes da cópia, o destino nunca existiria e o
cenário da duplicação deixaria de existir. O comentário do próprio arquivo
(`move_atomico_windows_test.go:104-108`) já traz a matriz medida em 2026-08-26 que
sustenta isso:

```
GENERIC_READ        + share=0 -> os.ReadFile OK,   os.Remove ERRO
GENERIC_READ|WRITE  + share=0 -> os.ReadFile ERRO, os.Remove ERRO
```

A prova substituta — remover cada trava na vez dela, duas saídas de FAIL coladas —
é **mais forte** que a única que o brief descrevia, não mais fraca: cobre as duas
travas em vez de uma. As duas mensagens de FAIL coladas batem com as linhas da
árvore atual (contas acima). Adequada.

**(b) D4 — `TravarDiretorioExclusivo` reprova onde o helper de `vault` pulava. É
o que o brief manda? `TestWalkNaoEngoleRaizQueExisteMasNaoLe` ainda afirma o
mesmo?**

Sim e sim.

O brief transcreve literalmente `t.Fatalf("vaulttest: o handle exclusivo nao
barrou a listagem de %s", abs)` (brief, linha 91), então a troca de `t.Skip` por
`t.Fatalf` é o brief cumprido, não um desvio do implementador — e a tese da tarefa
("trava que não trava é defeito, não condição de ambiente") sustenta a escolha. O
relatório declara a consequência honestamente: numa máquina onde o `ReadDir`
passasse, o teste ficaria vermelho em vez de pulado.

O teste continua afirmando o mesmo, e o corpo é o mesmo antes e depois:

```go
var vistos int
err = v.Walk(context.Background(), func(vault.Entry) error { vistos++; return nil })
if err == nil {
    t.Fatalf("Walk devolveu nil com %d entradas para uma raiz que ReadDir nao le: cofre inacessivel virou cofre vazio", vistos)
}
```

As únicas mudanças são de pacote (`New` → `vault.New`, `Entry` → `vault.Entry`),
consequência da conversão para `package vault_test` que a decisão 3 do
orquestrador mandou fazer. O comentário de domínio (Lstat passa, ReadDir falha com
`ERROR_SHARING_VIOLATION`, relação com `FalhaNaRaiz`) foi movido para cima do
teste, não perdido. A única regressão de comportamento é a do achado 4, no ramo
"o `CreateFile` falhou", que antes era `Fatalf` e agora é `Skipf`.

**(c) D5 — `TestDialAndHandshakeSocketAusente` compara contra 5 s em vez de 3 s.
A asserção ainda é significativa?**

Significativa, sim; enfraquecida, também. O teste passa `200*time.Millisecond`
como orçamento de dial e afere contra 5 s: uma regressão que travasse sem fim
ainda reprova (que é o propósito declarado do teto, e o que impede o binário de
morrer no timeout de 10 minutos), mas uma que desistisse em 4 s — 20x o orçamento
pedido — passaria, e antes reprovava. Não é asserção vazia; é asserção folgada. É
o achado 1, com o achado 2 registrando que o mesmo aconteceu num segundo ponto que
D5 não declarou. Não bloqueia: a decisão de unificar em 5 s foi tomada, o custo é
folga e não cobertura, e o conserto é local aos dois `if`.

**(d) D2 — `lockFileForTest` mantendo um lado POSIX com `os.Chmod`.**

A decisão de manter é **correta** e bem justificada: `TravarExclusivo` faz
`t.Skip` fora do Windows, então migrar o lado POSIX apagaria a cobertura de
`TestBuildRegistraArquivoIlegivel` em Linux e macOS — trocar uma cópia por um
buraco. A implementação também está limpa: o lado Windows virou três linhas
delegando ao pacote, o `syscall.CreateFile` a mão sumiu, os dois lados moram em
arquivos separados atrás de build tag (nada de `if runtime.GOOS ==`), e a troca de
`func()` de retorno por `t.Cleanup` foi aplicada aos dois lados e ao chamador
(`build_descarte_test.go:26`), que é por que os dois arquivos fora da lista do
brief foram tocados — declarado em D2, não silencioso.

O que falta é a simetria da prova, e é o achado 5: o lado Windows agora prova que
a trava trava e o lado POSIX continua sem provar que o `chmod` tornou o arquivo
ilegível.

---

## Qualidade do código — o que li e o que confirmei

**A prova dentro do helper.** `exclusivo_windows.go:36-43` abre com
`GENERIC_READ|GENERIC_WRITE` e `dwShareMode = 0` — o par que a matriz medida
identifica como o único que barra a leitura —, registra `t.Cleanup` que fecha o
handle **antes** de conferir, e só então afere `os.ReadFile`. A ordem importa e
está certa: se a conferência fosse antes do `Cleanup`, o `t.Fatalf` vazaria o
handle para o resto do binário de teste. `TravarDiretorioExclusivo` acrescenta
`FILE_FLAG_BACKUP_SEMANTICS` (obrigatório para abrir diretório) e afere
`os.ReadDir`. `MarcarSomenteNuvem` grava `FILE_ATTRIBUTE_OFFLINE`, registra o
`Cleanup` que restaura `NORMAL`, e confere `vault.IsCloudOnly(abs)` antes de
devolver — a prova que o brief pedia, com a assinatura que já existia em `vault`
(nenhuma função nova criada lá). Todos os caminhos usam `vault.LongPath`, que é a
conta única de caminho longo; três das cópias originais não usavam.

**O teste do próprio pacote testa o que interessa.** `TestTravarExclusivo…`
prende a trava dentro de um `t.Run` aninhado e afere **depois** do subteste que a
leitura voltou — isto é, que o `Cleanup` fechou o handle e ele não vazou. Um teste
que só afirmasse "com a trava, falha" não pegaria vazamento de handle. Bom.

**Os testes migrados afirmam o mesmo.** Comparei linha removida contra linha
adicionada nos onze arquivos. Em nenhum a mudança passa de "helper local apagado,
chamada trocada, import ajustado": os corpos de asserção são idênticos, com as
duas exceções deliberadas em `internal/service` (as guardas) e a única perda de
mensagem do achado 6. Os comentários de domínio que moravam nos helpers apagados
foram realocados, não descartados — `build_descarte_windows_test.go:5-11`,
`walk_raiz_windows_test.go:11-19` e `classify_cloudonly_windows_test.go:45-47` são
os três casos, e os três explicam por que a trava está ali.

**As guardas de `service` realmente ficaram incondicionais.** Li os dois corpos
resultantes, não o diff:

`move_atomico_windows_test.go:81-88` — `t.Fatalf` se `!origemExiste ||
!destinoExiste` (guarda de montagem, com o estado medido na mensagem), depois
`t.Fatalf` incondicional se `err == nil`. Nenhum `if` composto sobrou.

`erro_engolido_windows_test.go:47-59` — `t.Fatalf` se a origem sumiu, `t.Fatalf`
incondicional se `err == nil`, e a checagem do `.trash` agora fora de qualquer
condicional. A asserção de `.trash` era guardada por `err != nil` e passou a
depender de o `t.Fatalf` anterior ter deixado passar, o que é a mesma coisa com
falha nomeada em vez de silêncio. A nova asserção `err == nil` é **mais forte**
que a original (`errOrigem == nil && err == nil && res.Deleted`): reprova também
o caso `err == nil && !res.Deleted`, que antes passava calado.

**Comentários de doc.** Todo identificador exportado tem doc, inclusive os quatro
dos arquivos `_other.go` que o brief não comentava — o implementador acrescentou e
declarou o acréscimo. `golangci-lint` confirma: 0 issues.

**Nenhuma deliberação commitada.** Procurei os marcadores da armadilha ("Wait,",
"For the sake of", "TODO", "MUTACAO", "XXX") nos arquivos do diff: nada.

**Encoding.** `check_doc_refs.ps1` verde e os três `.md` tocados abrem como UTF-8
(o relatório colou as quatro checagens; `check_doc_refs` passando é confirmação
independente de que os arquivos são legíveis e os tokens entre crases existem).

---

## Veredictos

Os oito primeiros achados são de precisão de documento e de folga de asserção;
nenhum torna um teste incapaz de falhar, nenhum apaga cobertura, nenhum viola as
restrições globais. As três verificações mecânicas do brief devolvem exatamente o
que o brief exige, o grafo de imports é o prescrito, os desvios D1–D6 estão todos
declarados e os quatro que o orquestrador mandou julgar se sustentam — D1 e D2
sustentam-se por razão técnica que confirmei por leitura, D4 é o brief cumprido,
D5 é uma folga declarada (subnotificada pela metade, achado 2).

**Spec: APROVADO**

**Quality: APROVADO**

Blocking: **0**. Non-blocking: **9**.

Recomendo tratar os achados 2, 3 e 5 num commit de acerto — os três são de uma
linha a três linhas e os três tocam a mesma família de "a conta escrita não é a
conta medida", que é o que este projeto paga caro quando escapa. Os achados 1, 4,
6, 7 e 8 podem ser carregados; o 9 é anterior a esta tarefa.

---

## Progresso

Ressalva honesta: colhi `date +%H:%M` uma única vez, ao escrever o relatório
(**02:31**). Não cronometrei cada etapa e **não vou inventar** os horários
intermediários — a decisão do orquestrador de 2026-09-04 proíbe linha do tempo
digitada, e uma linha do tempo reconstruída de memória é exatamente isso. A ordem
abaixo é real; os horários, exceto o último, não foram medidos.

- (não medido) Li o brief da Task 159, o relatório com D1–D6, o pacote de diff
  completo (`-U10`, 2019 linhas, lido em quatro pedaços) e `docs/papeis/revisor.md`
- (não medido) Rodei por mim as três verificações do brief: zero `CreateFile(`,
  zero `boundedWait`, `FILE_ATTRIBUTE_OFFLINE` só nos dois comentários e na exceção
  documentada `internal/vault/cloudonly_info_windows_test.go`
- (não medido) `go list -deps` e `go list -f '{{.Imports}}'` em windows e linux;
  confirmei que nenhum arquivo de produção importa `vaulttest`; conferi build tag
  arquivo por arquivo nos sete do pacote
- (não medido) `go test -race ./internal/vaulttest/ -v`: PASS nos dois; cauda colada
- (não medido) `go test -race ./internal/service/ -run 'Move|Engol|Delete' -v`:
  ok, 2,952 s; cauda colada
- (não medido) Contei SKIP/FAIL em `vault`, `index`, `search`, `cmd/gobsidian`:
  um único SKIP, sem relação com a tarefa — nenhum teste migrado passou a pular
- (não medido) `gofmt -l`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`,
  `golangci-lint run ./internal/vaulttest/...`, `check_doc_refs.ps1`,
  `check_net.ps1`: todos verdes
- (não medido) Conferi o SHA `c81b2b8` e o stat (29 arquivos, +775/-338) contra o
  relatório; conferi a entrada da Task 159 no ledger
  (`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md:5180`)
- (não medido) Medi as oito cópias do handle exclusivo em `86f07e5` e quais
  conferiam a trava (achado 3); medi as duas asserções `elapsed > Prazo` em
  `ipc_test.go` (achados 1 e 2); contei os `Invoke-Step` de `verify.ps1` (achado 9)
- (não medido) Li os corpos resultantes de `move_atomico_windows_test.go` e
  `erro_engolido_windows_test.go` inteiros, não o diff, para julgar as guardas
- **02:31** Escrevi este relatório
