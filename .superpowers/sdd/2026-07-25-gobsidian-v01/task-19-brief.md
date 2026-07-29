### Task 19: Índice em memória e construção paralela

#### CORRECAO OBRIGATORIA — offset de BOM (auditoria 2026-07-28)

A implementacao que entrou chama `vault.StripBOM` e guarda o resultado em `Note.BOM`, mas **nunca soma os tres bytes de volta aos offsets**. O parser mede sobre o corpo sem BOM; `vault.ReadRange` le o arquivo, que tem o BOM. Toda leitura de secao numa nota com BOM sai deslocada.

Provado com a mesma nota, variando so o BOM:

```
sem BOM:  "## Alvo\n\nCONTEUDO-ESPERADO\n\n"     correto
com BOM:  "o\n\n## Alvo\n\nCONTEUDO-ESPERAD"     tres bytes antes
```

O conteudo comeca no fim da secao anterior e corta o fim da secao pedida. Hoje e leitura errada; quando M4 chegar, `note_patch` escreve no lugar errado e vira perda de dados.

`TestBuildBOM` existe e passa. Ele afirma que o heading **existe** e que `Note.BOM` e `true` — nunca afirma o offset, que e a metade que importa. Uma asercao de presenca nao pega deslocamento.

**Onde corrigir.** No ponto em que os offsets do parser, medidos sobre o buffer, viram offsets do indice, que precisam ser do arquivo. Isso e logo depois do `StripBOM`, adjacente a linha que criou a discrepancia — que e onde alguem procura.

`internal/vault/eol.go`, exportar o tamanho:

```go
const bomPrefix = "\xEF\xBB\xBF"

// BOMLen e o tamanho do marcador UTF-8 que StripBOM remove. Quem guarda
// offsets medidos sobre o corpo sem BOM precisa somar isto para obter
// posicao no arquivo — sem isso toda leitura de secao numa nota com BOM
// sai deslocada em tres bytes, em silencio.
const BOMLen = len(bomPrefix)

var bom = []byte(bomPrefix)
```

`internal/parser/types.go`, deslocar tudo de uma vez:

```go
// ShiftOffsets soma delta a todo offset da nota.
//
// Existe para uma coisa so: os offsets saem do Parse medidos sobre o buffer
// recebido, e quem tem o arquivo precisa deles medidos sobre o arquivo. A
// diferenca e o BOM, que vault.StripBOM removeu antes. Deslocar aqui, num
// lugar, e o que impede cada consumidor de lembrar de somar por conta.
func (n *ParsedNote) ShiftOffsets(delta int64) {
	if delta == 0 {
		return
	}
	for i := range n.Headings {
		n.Headings[i].Start += delta
		n.Headings[i].BodyStart += delta
		n.Headings[i].End += delta
	}
	for i := range n.Blocks {
		n.Blocks[i].Start += delta
		n.Blocks[i].End += delta
	}
	for i := range n.Links {
		// offsetUnknown marca posicao que o parser nao determinou; deslocar
		// um sentinela o transformaria num offset plausivel e errado.
		if n.Links[i].Start == offsetUnknown {
			continue
		}
		n.Links[i].Start += delta
		n.Links[i].End += delta
	}
}
```

`internal/index/build.go` e `internal/index/update.go`, nos dois caminhos que chamam `Parse`:

```go
body, hadBOM := vault.StripBOM(data)
note, err := parser.Parse(body)
if err != nil {
	// ... tratamento existente
}
if hadBOM {
	note.ShiftOffsets(vault.BOMLen)
}
```

**O teste precisa afirmar o conteudo devolvido, nao a presenca.** Indexe a mesma nota duas vezes, uma com BOM e outra sem, leia a mesma secao pelo servico, e exija que os dois conteudos sejam **iguais**. Uma asercao sobre `Headings[0].Text` passa com o bug; uma sobre os bytes devolvidos, nao.

Prove por mutacao: remova o `if hadBOM`, confirme que o teste novo reprova, restaure.

#### Onde isto encaixa

`internal/parser` está pronto e congelado por golden files: frontmatter com offset de corpo, slugs, headings com offsets de seção, e quatro extensões goldmark — wikilinks e embeds, block ids, tags hierárquicas, campos do Dataview. `internal/vault` está pronto: caminho canônico com confinamento, varredura com exclusões, EOL e BOM, caminho longo, placeholder de nuvem.

Esta tarefa junta os dois. O índice transforma um diretório de arquivos em algo consultável, e **todas as tarefas seguintes leem dele** — resolução (20), backlinks (21), consultas (22), métodos de leitura (23) e tools (24). Erro aqui não aparece como erro: aparece como nota que sumiu, backlink que não existe, contagem errada em `vault_stats`.

#### O que já está fechado e vincula esta tarefa

- **O índice guarda offsets de byte, nunca conteúdo.** É o que sustenta o orçamento de 60 MB de RSS e o que faz ler uma seção de 2 KB numa nota de 500 KB custar 2 KB. Não armazene o corpo.
- **`vault.Walk` já resolve exclusões, classificação nota/anexo e a grafia real do disco.** Não reimplemente varredura; consuma `vault.Entry`.
- **Anexo é indexado por nome, nunca lido.** Sem isso todo `![[imagem.png]]` vira link quebrado, e a contagem de links quebrados — principal sinal de saúde do cofre — afoga em falso positivo.
- **Arquivo somente-nuvem não é aberto.** Abrir dispara download síncrono, e indexar o cofre inteiro assim trava por minutos. `vault.Entry.CloudOnly` já traz a informação.
- **`ctx` só onde bloqueia**, e respeitado de verdade: um `ctx` cancelado tem que parar a varredura, não apenas ser aceito como parâmetro.

#### Armadilhas já pagas neste projeto que se aplicam aqui

- **Determinismo sob paralelismo.** O worker pool faz a ordem de conclusão variar; o resultado não pode variar com ela. Construa o mesmo cofre duas vezes e compare — se `Hash` ou a contagem de links divergir, há estado compartilhado onde não devia.
- **Falha na raiz virando sucesso vazio.** A Task 8 chegou a ter uma varredura que, com o cofre desmontado, devolvia `nil` e zero entradas: o servidor afirmava com confiança que o cofre estava vazio. Cofre inacessível e cofre vazio não podem produzir a mesma resposta. `vault.Walk` já devolve erro nesse caso — propague, não engula.
- **Entrada descartada em silêncio.** `vault.SkippedEntries()` existe porque uma nota que está no disco e não entra no índice fica inalcançável *e* indiagnosticável. Se o índice descartar algo por conta própria, conte e reporte igual.

#### Verificações além dos passos

Faça e **reporte o resultado real de cada uma**, inclusive quando estiver correto:

- Construir o mesmo cofre duas vezes produz índices idênticos? Compare `Hash` e contagem de links por nota.
- `ctx` cancelado antes do `Build` para a varredura cedo, ou ela percorre tudo e só então devolve erro? Meça quantas entradas foram visitadas.
- Um arquivo ilegível no meio da varredura derruba a construção inteira ou é pulado?
- Uma nota com frontmatter tem offsets de heading corretos em relação ao buffer? Fatie e confira.
- Anexo entra em `assets` e não em `notes`? Somente-nuvem entra sem ser lido?
- **Costura do BOM, achada na revisão da Task 18 e roteada para cá.** `Parse` exige entrada já sem BOM, e quem remove é `vault.StripBOM`. Hoje as duas pontas são testadas isoladamente e **nada testa a composição**: o golden `testdata/parser/edge/bom.md` prova que uma nota com BOM parseada direto produz `{}` — nenhum heading, nenhum título. Se esta tarefa esquecer a chamada, todo heading de toda nota com BOM desaparece, sem erro. Escreva um teste que leve uma nota com BOM real do disco até o índice e afirme que o heading está lá.

#### Regras de execução

Valem para toda tarefa deste plano e não são negociáveis.

- **O plano é a fonte.** Transcreva o código desta seção; não improvise uma variante. Se ele não compilar, corrija o erro mecânico e **diga exatamente o que mudou**. Se um teste falhar por motivo que a seção não explica, **pare e reporte** — não ajuste a expectativa para o código passar. Teste dobrado para passar é como defeito silencioso chega em produção.
- **Nunca rode `git checkout`, `git restore`, `git stash`, `git clean` ou `git reset`.** Há trabalho não commitado de outras frentes neste repositório, e um subagente já destruiu trabalho exatamente assim. Para desfazer o que você escreveu, edite de volta ou apague o arquivo específico que você criou.
- **`go mod tidy` está proibido.** Várias dependências fixadas ainda não têm importador, e o `tidy` as removeria — inclusive o pin obrigatório do SDK de MCP. Se o build reclamar de entrada faltando em `go.sum`, **pare e reporte**; não rode `go get`.
- **Ao editar arquivo por script, leia *e* grave com `newline=""`.** Escrita em modo texto converte o arquivo inteiro para CRLF no Windows e o `gofmt` rejeita. Já custou dois commits neste projeto.
- **Verde obrigatório antes do commit:** `go test -race ./...`, `go vet ./...`, `gofmt -l .`, `GOOS=linux go vet ./...`, `GOOS=darwin go vet ./...`.
- Commits em Conventional Commits, em inglês. Sem arquivos chamados `helpers.go`, `utils.go` ou `common.go`.

#### Contrato de relatório

Grave o relatório completo em `.superpowers/sdd/task-19-report.md`, com: o que implementou; evidência de TDD (comando e saída do RED, comando e saída do GREEN); a tabela de verificações extras acima com o resultado real de cada uma; arquivos alterados; achados da auto-revisão; correções mecânicas que fez no código do plano; e preocupações.

Responda com no máximo 15 linhas: status (`DONE` | `DONE_WITH_CONCERNS` | `BLOCKED` | `NEEDS_CONTEXT`), commit criado, resumo de teste em uma linha, as respostas diretas pedidas acima, e preocupações. O detalhe mora no arquivo de relatório, não na resposta.

**Files:**
- Create: `internal/index/note.go`, `internal/index/index.go`, `internal/index/build.go`
- Create: `internal/index/build_test.go`

**Interfaces:**
- Consumes: `vault.Vault`, `vault.Entry`, `vault.DetectEOL` (Task 8); `parser.Parse`, `parser.ParsedNote` (Tasks 12–17)
- Produces: `index.Note`, `index.Asset`, `index.Backlink`, `index.LinkState`, `index.ResolveVia`, `index.ResolvedLink`; `index.New() *Index`; `(*Index).Build(ctx, *vault.Vault) error`; `(*Index).Get(vault.CanonicalPath) (*Note, bool)`; `(*Index).NoteCount() int`; `(*Index).AssetCount() int`; `(*Index).TotalSize() int64`; `(*Index).Generation() uint64`

- [ ] **Step 1: Escrever o teste**

`internal/index/build_test.go`:

```go
package index_test

import (
	"context"
	"testing"

	"github.com/jonyd/gobsidian/internal/index"
	"github.com/jonyd/gobsidian/internal/vault"
)

func TestBuildIndexesNotesAndAssets(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "Civil/PONTO 03.md", "---\naliases: [P3]\n---\n# Ponto 3\n\nVer [[Penal/A]].\n")
	writeFile(t, root, "Penal/A.md", "# A\n\n![[diagrama.png]]\n")
	writeFile(t, root, "Anexos/diagrama.png", "\x89PNG")

	v, err := vault.New(root)
	if err != nil {
		t.Fatalf("vault.New: %v", err)
	}

	idx := index.New()
	if err := idx.Build(context.Background(), v); err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got := idx.NoteCount(); got != 2 {
		t.Errorf("NoteCount() = %d, quer 2", got)
	}
	if got := idx.AssetCount(); got != 1 {
		t.Errorf("AssetCount() = %d, quer 1", got)
	}

	note, ok := idx.Get("Civil/PONTO 03.md")
	if !ok {
		t.Fatal("nota nao encontrada pelo caminho canonico")
	}
	if note.Title != "Ponto 3" {
		t.Errorf("Title = %q, quer %q", note.Title, "Ponto 3")
	}
	if note.Hash == 0 {
		t.Error("Hash nao foi calculado")
	}
	if note.EOL != vault.EOLLF {
		t.Errorf("EOL = %v, quer LF", note.EOL)
	}
	if len(note.Aliases) != 1 || note.Aliases[0] != "P3" {
		t.Errorf("Aliases = %v, quer [P3]", note.Aliases)
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	root := t.TempDir()
	for i := range 50 {
		writeFile(t, root, fmt.Sprintf("n%02d.md", i), fmt.Sprintf("# N%d\n\n[[n%02d]]\n", i, (i+1)%50))
	}

	v, _ := vault.New(root)

	first := index.New()
	if err := first.Build(context.Background(), v); err != nil {
		t.Fatalf("Build 1: %v", err)
	}
	second := index.New()
	if err := second.Build(context.Background(), v); err != nil {
		t.Fatalf("Build 2: %v", err)
	}

	// O worker pool paraleliza o parse, e a ordem de conclusao varia. O
	// resultado nao pode variar com ela.
	if first.NoteCount() != second.NoteCount() {
		t.Fatalf("contagens divergem: %d vs %d", first.NoteCount(), second.NoteCount())
	}
	for i := range 50 {
		p := vault.CanonicalPath(fmt.Sprintf("n%02d.md", i))
		a, okA := first.Get(p)
		b, okB := second.Get(p)
		if !okA || !okB {
			t.Fatalf("%s ausente em um dos indices", p)
		}
		if a.Hash != b.Hash || len(a.Links) != len(b.Links) {
			t.Errorf("%s divergiu entre construcoes", p)
		}
	}
}

func TestBuildRespectsContextCancellation(t *testing.T) {
	root := t.TempDir()
	for i := range 200 {
		writeFile(t, root, fmt.Sprintf("n%03d.md", i), "# N\n")
	}

	v, _ := vault.New(root)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	idx := index.New()
	if err := idx.Build(ctx, v); err == nil {
		t.Fatal("Build com context cancelado deveria falhar")
	}
}
```

Adicione `"fmt"` e um `writeFile` local idêntico ao de `internal/vault/walk_test.go`.

- [ ] **Step 2: Rodar para confirmar que falha**

Run: `go test ./internal/index/ -v`
Esperado: FAIL — `undefined: index.New`.

- [ ] **Step 3: Implementar os tipos**

`internal/index/note.go`:

```go
package index

import (
	"time"

	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
)

type LinkState int

const (
	LinkOK LinkState = iota
	LinkTargetMissing
	LinkAnchorMissing
)

func (s LinkState) String() string {
	switch s {
	case LinkTargetMissing:
		return "target_missing"
	case LinkAnchorMissing:
		return "anchor_missing"
	default:
		return "ok"
	}
}

type ResolveVia int

const (
	ViaNone ResolveVia = iota
	ViaPath
	ViaName
	ViaAsset
	ViaAlias
)

func (v ResolveVia) String() string {
	switch v {
	case ViaPath:
		return "path"
	case ViaName:
		return "name"
	case ViaAsset:
		return "asset"
	case ViaAlias:
		return "alias"
	default:
		return ""
	}
}

// ResolvedLink e um parser.Link mais o resultado da resolucao, que depende do
// cofre inteiro e por isso nao pode ser feita no parser.
type ResolvedLink struct {
	parser.Link
	Resolved vault.CanonicalPath
	Via      ResolveVia
	State    LinkState
}

type Note struct {
	Path    vault.CanonicalPath
	Title   string
	Size    int64
	ModTime time.Time
	// Hash e xxhash do conteudo BRUTO do arquivo, com frontmatter e BOM.
	// E o valor exposto como "hash" e aceito em expected_hash.
	Hash uint64
	EOL  vault.EOLStyle
	BOM  bool
	// CloudOnly marca placeholder do OneDrive: indexado por metadados de
	// diretorio, sem leitura de conteudo.
	CloudOnly bool

	Frontmatter map[string]any
	Tags        []string
	Aliases     []string
	Headings    []parser.Heading
	Blocks      []parser.Block
	Links       []ResolvedLink
	Inline      map[string][]string
}

type Asset struct {
	Path    vault.CanonicalPath
	Size    int64
	ModTime time.Time
}

type Backlink struct {
	From    vault.CanonicalPath
	Anchor  string
	Alias   string
	Context string // texto ao redor da referencia
	Kind    parser.LinkKind
}
```

- [ ] **Step 4: Implementar o índice e a construção**

`internal/index/index.go` declara a struct de ARCHITECTURE §4.1 — `notes`, `assets`, `lowerPath`, `byName`, `byAlias`, `backlinks`, `tags`, `generation` — protegida por `sync.RWMutex`, com os acessores `Get`, `NoteCount`, `AssetCount`, `TotalSize`, `Generation`. Leituras usam `RLock`; toda mutação incrementa `generation`.

`internal/index/build.go`:

```go
package index

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/cespare/xxhash/v2"
	"github.com/jonyd/gobsidian/internal/parser"
	"github.com/jonyd/gobsidian/internal/vault"
	"golang.org/x/sync/errgroup"
)

type parsed struct {
	entry vault.Entry
	note  *parser.ParsedNote
	hash  uint64
	eol   vault.EOLStyle
	bom   bool
}

// Build varre o cofre e constroi o indice do zero.
//
// A varredura enfileira caminhos; um worker pool le e parseia em paralelo; um
// unico coletor popula o indice. O coletor ser unico e o que dispensa lock no
// caminho quente e o que torna o resultado independente da ordem de conclusao.
func (ix *Index) Build(ctx context.Context, v *vault.Vault) error {
	entries := make(chan vault.Entry, 256)
	results := make(chan parsed, 256)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		defer close(entries)
		return v.Walk(gctx, func(e vault.Entry) error {
			select {
			case entries <- e:
				return nil
			case <-gctx.Done():
				return gctx.Err()
			}
		})
	})

	workers := runtime.NumCPU()
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		g.Go(func() error {
			defer wg.Done()
			for e := range entries {
				if err := gctx.Err(); err != nil {
					return err
				}

				// Anexos e placeholders de nuvem nao sao lidos. Ler um
				// placeholder dispararia download sincrono, e indexar o cofre
				// inteiro assim trava por minutos.
				if !e.IsNote || e.CloudOnly {
					select {
					case results <- parsed{entry: e}:
					case <-gctx.Done():
						return gctx.Err()
					}
					continue
				}

				data, err := v.ReadAll(gctx, e.Path)
				if err != nil {
					// Um arquivo ilegivel nao derruba a indexacao inteira.
					continue
				}

				body, hadBOM := vault.StripBOM(data)
				note, err := parser.Parse(body)
				if err != nil {
					continue
				}

				select {
				case results <- parsed{
					entry: e,
					note:  note,
					hash:  xxhash.Sum64(data),
					eol:   vault.DetectEOL(data),
					bom:   hadBOM,
				}:
				case <-gctx.Done():
					return gctx.Err()
				}
			}
			return nil
		})
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	g.Go(func() error {
		for r := range results {
			ix.insert(r)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("construindo indice: %w", err)
	}

	// As tres passadas seguintes dependem do conjunto completo e por isso
	// acontecem depois, nao durante.
	ix.buildAliasMap()
	ix.resolveAllLinks()
	ix.buildBacklinks()

	return nil
}
```

Adicione `golang.org/x/sync` ao `go.mod` (`go get golang.org/x/sync`).

**Sobre `go mod tidy` neste projeto.** Ele fica proibido enquanto houver dependencia fixada sem importador — hoje `goldmark`, `fsnotify`, `yaml.v3` e `x/text`, que so ganham importador em M1 e M3. Rodar `tidy` antes disso as remove do `go.mod`, junto com o pin do SDK.

O que **e** permitido, e necessario quando um pacote passa a importar uma dependencia pela primeira vez, e `go get <modulo>@<versao-fixada>`: ele resolve o grafo transitivo e grava as entradas que faltam no `go.sum` sem remover nada. O sintoma de nao ter feito isso e um erro de build reclamando de entradas ausentes no `go.sum` para modulos que nem aparecem no `go.mod`. É biblioteca oficial estendida, não amplia a superfície de auditoria de forma relevante, e `errgroup` elimina a coordenação manual de erro entre as goroutines.

`insert` grava `Note` ou `Asset` conforme `entry.IsNote`, preenchendo `lowerPath` e `byName`. Note que `insert` é chamado de uma única goroutine — mas ainda assim adquire o lock, porque `Build` pode rodar concorrente a leituras em uma reconstrução.

- [ ] **Step 5: Rodar para confirmar que passa**

Run: `go test -race ./internal/index/ -v`
Esperado: PASS, três testes.

- [ ] **Step 6: Commit**

```bash
git add internal/index go.mod go.sum
git commit -m "feat(index): parallel build with single collector and deterministic result"
```

---

