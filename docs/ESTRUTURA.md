# Estrutura do projeto — gobsidian

---

## 1. Árvore

```
gobsidian/
├── cmd/
│   └── gobsidian/
│       ├── main.go               entrypoint; só constrói o context raiz e delega
│       ├── serve.go              subcomando serve (stdio)
│       ├── doctor.go             subcomando doctor (diagnóstico de ambiente)
│       ├── index.go              subcomando index (indexar e sair)
│       ├── search.go             subcomando search (busca via CLI)
│       └── inspect.go            subcomando inspect (dump do parse de uma nota)
│
├── internal/
│   ├── config/
│   │   ├── config.go             struct de configuração, precedência de fontes
│   │   ├── flags.go              mapeamento de flags do cobra
│   │   └── defaults.go           valores padrão em um único lugar
│   │
│   ├── lifecycle/
│   │   ├── lifecycle.go          composição dos três mecanismos → context raiz
│   │   ├── stdin.go              detecção de EOF em stdin
│   │   ├── signals.go            handlers de SIGINT/SIGTERM
│   │   ├── parent_windows.go     vigília do PID pai (build tag windows)
│   │   ├── parent_unix.go        vigília do PID pai (build tag !windows)
│   │   └── shutdown.go           sequência de encerramento com timeout
│   │
│   ├── vault/
│   │   ├── vault.go              tipo Vault; raiz, abertura, validação
│   │   ├── path.go               caminho canônico, resolução, confinamento
│   │   ├── walk.go               varredura com exclusões
│   │   ├── ignore.go             .gitignore e .gobsidianignore
│   │   ├── eol.go                detecção e preservação de CRLF/LF
│   │   └── cloud_windows.go      detecção de arquivo somente-nuvem (OneDrive)
│   │
│   ├── parser/
│   │   ├── parser.go             fachada: []byte → ParsedNote
│   │   ├── frontmatter.go        separação e decodificação do bloco YAML
│   │   ├── ast.go                travessia da AST do goldmark
│   │   ├── ext_wikilink.go       extensão goldmark: [[...]] e ![[...]]
│   │   ├── ext_blockid.go        extensão goldmark: ^id em fim de linha
│   │   ├── ext_tag.go            extensão goldmark: #tag/aninhada
│   │   ├── ext_inline_field.go   extensão goldmark: chave:: valor
│   │   ├── headings.go           extração da hierarquia com offsets de seção
│   │   ├── slug.go               normalização de heading para casar com âncoras
│   │   └── types.go              ParsedNote, Heading, Block, Link, LinkKind
│   │
│   ├── index/
│   │   ├── index.go              struct Index, RWMutex, ciclo de vida
│   │   ├── note.go               tipo Note e derivações
│   │   ├── build.go              varredura inicial com worker pool
│   │   ├── update.go             substituição incremental de uma nota
│   │   ├── backlinks.go          construção e manutenção do grafo reverso
│   │   ├── resolve.go            resolução de wikilink → caminho canônico
│   │   ├── alias.go              mapa alias → caminhos, do frontmatter
│   │   ├── assets.go             registro de anexos (nome, tamanho, mtime)
│   │   ├── anchors.go            validação de âncora de heading e de bloco
│   │   ├── query.go              consultas por metadados (tag, pasta, frontmatter)
│   │   ├── cache.go              serialização e carga do índice em disco
│   │   └── reconcile.go          varredura de reconciliação pós-overflow
│   │
│   ├── search/
│   │   ├── search.go             fachada de busca
│   │   ├── analyzer.go           tokenização, case folding, diacríticos, stemming
│   │   ├── inverted.go           dicionário de termos e listas de postagem
│   │   ├── bm25.go               ranking
│   │   ├── snippet.go            extração de trecho com destaque
│   │   └── persist.go            serialização do índice de busca
│   │
│   ├── watcher/
│   │   ├── watcher.go            fachada; encapsula fsnotify
│   │   ├── debounce.go           janela de debounce e coalescência
│   │   ├── filter.go             filtro de relevância de evento
│   │   └── overflow.go           tratamento de ErrEventOverflow
│   │
│   ├── writer/
│   │   ├── writer.go             fachada de escrita
│   │   ├── lock.go               mutex por caminho canônico; serializa escritas
│   │   ├── atomic.go             temporário + sync + rename, com retry
│   │   ├── section.go            inserir e substituir sob heading
│   │   ├── block.go              substituir bloco por ^id
│   │   ├── linkrewrite.go        reescrita de links em note_move
│   │   └── diff.go               diff unificado para dry-run
│   │
│   ├── service/
│   │   ├── service.go            struct Service; injeção dos subsistemas
│   │   ├── read.go               métodos de leitura
│   │   ├── write.go              métodos de escrita
│   │   ├── graph.go              link_graph, tag_list, vault_stats
│   │   └── errors.go             taxonomia de erros de domínio
│   │
│   ├── mcpsrv/
│   │   ├── server.go             construção do servidor, registro de tools
│   │   ├── tools_read.go         handlers e schemas das tools de leitura
│   │   ├── tools_write.go        handlers e schemas das tools de escrita
│   │   ├── resources.go          exposição de notas como resources gobsidian://
│   │   ├── recover.go            middleware de recuperação de panic
│   │   └── convert.go            erros de domínio → resultados MCP
│   │
│   └── doctor/
│       ├── doctor.go             orquestração das verificações
│       ├── checks.go             verificações independentes de plataforma
│       └── checks_windows.go     OneDrive, MAX_PATH, colisão de casing
│
├── docs/
│   ├── PRD.md
│   ├── ARCHITECTURE.md
│   ├── ESTRUTURA.md
│   ├── TOOLS.md
│   └── WINDOWS.md
│
├── testdata/
│   ├── parser/                   golden files: entrada .md + saída .json esperada
│   │   ├── wikilinks/
│   │   ├── codeblocks/           casos que NÃO devem produzir links
│   │   ├── frontmatter/
│   │   ├── headings/
│   │   ├── blocks/
│   │   └── edge/                 vazio, sem newline final, CRLF misto, BOM
│   ├── vault_small/              cofre de 50 notas para testes de integração
│   └── parity/                   corpus + metadata cache do Obsidian de referência
│
├── tools/
│   ├── parity-dumper/            plugin de dev do Obsidian; serializa app.metadataCache
│   │                             não é parte do produto, não é distribuído
│   └── netcheck/                 analisador go/analysis: proíbe rede em internal/ e cmd/
│
├── scripts/
│   ├── build.ps1                 build local com informação de versão
│   ├── gen_vault.ps1             gera cofre sintético para benchmark
│   ├── check_net.ps1             verificação de RNF-30 (ver §4)
│   └── test_orphans.ps1          100 ciclos de encerramento abrupto
│
├── .github/workflows/
│   ├── ci.yml                    vet, lint, test, race
│   └── bench.yml                 benchmark com verificação de regressão
│
├── go.mod
├── go.sum
├── Makefile
├── .golangci.yml
├── LICENSE
└── README.md
```

---

## 2. Racional da organização

### `cmd/` fino, `internal/` grosso

`cmd/gobsidian` não contém lógica de domínio. Analisa flags, monta configuração, constrói o `Service` e delega. Isso mantém toda a lógica testável sem passar pela CLI.

### Tudo em `internal/`

Nada em `pkg/`. `internal/` impede importação externa em nível de compilador, o que significa que nenhuma API interna vira compromisso público por acidente.

Se o parser vier a ter valor independente, ele pode ser extraído para `pkg/obsidian/` mais tarde — mas essa decisão deve ser deliberada, quando houver demanda real, não presumida na criação do projeto.

### Um pacote por preocupação, sem hierarquia profunda

Todos os pacotes são filhos diretos de `internal/`. A profundidade de aninhamento não comunica nada útil e complica os caminhos de importação. A dependência entre pacotes é dirigida e acíclica:

```
mcpsrv → service → {index, search, writer, vault, parser}
                    index → {parser, vault}
                    search → parser
                    writer → vault
                    watcher → {vault, index, search}
```

`parser` e `vault` são folhas — não dependem de nenhum outro pacote interno. É por isso que ambos são triviais de testar.

### Arquivos por operação, não por tipo

Dentro de cada pacote, os arquivos correspondem a operações (`build.go`, `update.go`, `resolve.go`), não a categorias sintáticas (`types.go`, `interfaces.go`, `helpers.go`). A exceção é `types.go` em `parser/`, onde as estruturas de dados são o contrato central do pacote e ficam melhor concentradas.

Um arquivo chamado `helpers.go` ou `utils.go` é sinal de que uma preocupação não foi nomeada. Não existem no projeto.

### Separação por build tag para código de plataforma

`parent_windows.go` e `parent_unix.go`, `cloud_windows.go`, `checks_windows.go`. Código específico de sistema operacional fica isolado atrás de build tags, nunca dentro de um `if runtime.GOOS == "windows"` no meio de lógica compartilhada.

---

## 3. Convenções de código

### Contexto

Toda função que pode **bloquear** recebe `ctx context.Context` como primeiro parâmetro e o respeita: leitura e escrita de arquivo, varredura do cofre, worker pool, watcher, chamadas MCP. O context raiz vem de `lifecycle`; cancelá-lo derruba tudo em cascata. Isso é o que faz o shutdown funcionar de verdade em vez de apenas parecer funcionar.

O critério é espera real, não contato com o sistema operacional. Consulta de variável de ambiente, resolução de caminho e cálculo em memória não recebem `ctx`, porque não há nada a cancelar. Um `ctx` que nenhum corpo de função verifica é pior que ausente: ensina o revisor a não olhar para `ctx`, e o parâmetro perde o significado justamente onde ele importa.

### Erros

Erros são embrulhados com `fmt.Errorf("...: %w", err)` em cada camada, acrescentando contexto sem perder a cadeia. Erros de domínio são valores sentinela declarados em `service/errors.go`, verificáveis com `errors.Is`.

Erro que o usuário verá inclui o dado que permite corrigi-lo — o caminho, o heading procurado, as alternativas disponíveis. Ver ARCHITECTURE §8.3.

### Logging

`log/slog`, sempre em **stderr**. Nunca em stdout, sem exceção: stdout carrega o JSON-RPC e um único byte estranho corrompe a sessão.

Log estruturado com campos, não interpolação: `slog.Info("reindexed", "path", p, "duration", d)`.

Níveis: `Debug` para eventos do watcher e detalhe de parse; `Info` para mudanças de estado (boot completo, reindexação de lote); `Warn` para condições recuperadas (retry de rename, overflow tratado); `Error` para falhas devolvidas ao cliente.

### Concorrência

Mutexes protegem estruturas de dados; canais coordenam goroutines. Não misturar os dois papéis.

Toda goroutine tem dono explícito e caminho de encerramento. `sync.WaitGroup` para esperar por elas no shutdown. Goroutine sem caminho de saída é vazamento, e em um processo de vida longa vazamento é falha.

Todos os testes rodam com `-race` no CI.

### Nomenclatura

Nomes de pacote em minúsculas, uma palavra, sem underscores. Sem `util`, sem `common`, sem `base`.

Tipos exportados carregam o significado sem repetir o pacote: `index.Note`, não `index.IndexNote`.

Nomes de arquivo em `snake_case.go`.

### Testes

Arquivo de teste ao lado do arquivo testado. Testes tabelados como padrão. `testdata/` para fixtures, nunca strings longas embutidas no código de teste.

Testes de parser comparam contra golden files em JSON, regeneráveis com `go test ./internal/parser -update`. Isso torna aceitar uma mudança intencional de comportamento uma operação de um comando, e torna uma regressão acidental imediatamente visível no diff.

---

## 4. Build

### go.mod

```
module github.com/jonyd/gobsidian

go 1.25

require (
    github.com/modelcontextprotocol/go-sdk v1.5.0
    github.com/yuin/goldmark            v1.7.8
    github.com/fsnotify/fsnotify         v1.8.0
    github.com/cespare/xxhash/v2         v2.3.0
    github.com/spf13/cobra               v1.8.1
    golang.org/x/text                    v0.21.0
    golang.org/x/sys                     v0.28.0
    gopkg.in/yaml.v3                     v3.0.1
)
```

A diretiva `go 1.25` é o piso mínimo, não a versão do toolchain instalado — ela é imposta pelo próprio `go-sdk@v1.5.0`, cujo `go.mod` declara `go 1.25.0`, e as regras de grafo de módulos do Go exigem que o módulo principal declare pelo menos essa versão; não é uma escolha por recurso de linguagem que precisemos.

As versões acima são o ponto de partida; fixe o que `go mod tidy` resolver e não use `latest` em nenhuma delas. O SDK de MCP em particular fica em `v1.5.0`, que é a versão com suporte pleno ao protocolo `2025-11-25` (PRD D6).

Versões das dependências fixadas exatamente. O SDK do MCP em particular: o protocolo evoluiu com quebras (2025-06-18 → 2025-11-25 → 2026-07-28), e uma atualização automática pode quebrar a compatibilidade com o host instalado.

### Informação de versão

Injetada via linker, não hardcoded:

```powershell
$Version = git describe --tags --always --dirty
$Commit = git rev-parse --short HEAD
$BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")

$LdFlags = "-s -w " +
    "-X main.version=$Version " +
    "-X main.commit=$Commit " +
    "-X main.buildDate=$BuildDate"

go build -ldflags $LdFlags -o "bin\gobsidian.exe" ".\cmd\cofre"
```

`-s -w` remove tabela de símbolos e informação de debug, reduzindo o binário em torno de 30%.

### Comandos de desenvolvimento

```powershell
# Build local
& (Join-Path $PSScriptRoot "scripts\build.ps1")

# Testes com detector de corrida
go test -race ./...

# Testes com cobertura
go test -coverprofile="coverage.out" ./...
go tool cover -html="coverage.out"

# Benchmarks
go test -bench=. -benchmem ./internal/index ./internal/search ./internal/parser

# Lint
golangci-lint run

# Regenerar golden files apos mudanca intencional do parser
go test ./internal/parser -update

# Verificar que nenhum pacote NOSSO importa rede (RNF-30)
$ModulePath = go list -m 2>$null
if ($LASTEXITCODE -ne 0 -or -not $ModulePath) {
    Write-Warning "[!] 'go list -m' falhou; nao foi possivel determinar o modulo"
    exit 1
}

$Rows = go list -f '{{.ImportPath}}|{{join .Imports ","}}' ./...
if ($LASTEXITCODE -ne 0 -or -not $Rows) {
    Write-Warning "[!] 'go list ./...' falhou ou nao retornou pacotes; verificacao nao executada"
    exit 1
}

$Scoped = $Rows | Where-Object {
    $Pkg = ($_ -split "\|", 2)[0]
    $Pkg -eq "$ModulePath/internal" -or $Pkg -like "$ModulePath/internal/*" -or
    $Pkg -eq "$ModulePath/cmd" -or $Pkg -like "$ModulePath/cmd/*"
}

$Offenders = @()

foreach ($Row in $Scoped) {
    $Parts = $Row -split "\|", 2
    $Pkg = $Parts[0]

    $Imports = @()
    if ($Parts.Count -gt 1 -and $Parts[1]) { $Imports = $Parts[1] -split "," }

    $Net = $Imports | Where-Object { $_ -eq "net" -or $_ -like "net/*" }
    if ($Net) { $Offenders += "$Pkg -> $($Net -join ', ')" }
}

if ($Offenders) {
    Write-Warning "[!] Pacote do produto importando rede:"
    $Offenders | ForEach-Object { Write-Output "    $_" }
    exit 1
}
Write-Output "[OK] Nenhum pacote de internal/ ou cmd/ importa rede"
```

A última verificação vira um passo do CI. RNF-30 só tem valor se for continuamente verificado. O script falha alto (`[!]` e `exit 1`) se `go list` não conseguir rodar ou devolver nada — um gate que não consegue distinguir "nada para checar" de "nada errado" não é gate. A varredura roda sobre todos os pacotes do módulo, mas os candidatos a ofensor são filtrados para os que vivem sob `internal/` e `cmd/`, que é o escopo real da garantia — não todo `./...`.

**Por que `./...` e não `-deps`.** A verificação óbvia — `go list -deps ./cmd/gobsidian` e falhar se aparecer qualquer `net/*` — falharia sempre e em poucos dias estaria comentada. O SDK de MCP importa `net/http` para o transporte HTTP/SSE, e essa importação entra no fechamento transitivo mesmo construindo apenas stdio.

`go list ./...` percorre somente os pacotes do módulo. O que ele verifica é a afirmação que de fato sustentamos: nenhum código nosso fala com a rede. Ver PRD §6.4 para as três partes da garantia e para o segundo passo, o analisador de chamadas.

---

## 5. Onde começar a implementar

A ordem importa, e não é a ordem óbvia.

**Primeiro: `internal/lifecycle`.** Antes de qualquer tool, antes do parser, antes de tudo. É o requisito que define o produto, e é o tipo de coisa que nunca fica boa se for deixada para o fim. Um servidor que só responde `vault_stats` mas encerra corretamente em todos os cenários já é mais útil que um servidor completo que deixa órfãos.

**Segundo: `internal/vault` e `internal/parser`.** São folhas do grafo de dependências, puros, e integralmente testáveis por golden file. Todo o resto se apoia neles, e um erro aqui contamina tudo acima silenciosamente.

**Terceiro: `internal/index`.** Com o parser correto, o índice é essencialmente contabilidade — cuidadosa, mas sem sutileza.

**Quarto: `internal/mcpsrv` e as tools de leitura.** Neste ponto o produto já é usável e já substitui parte do fluxo de trabalho. **É o corte da v0.1** (PRD §9): passa a ser usado todos os dias, sem watcher e sem busca, reindexando no boot. Tudo o que vem depois é construído sobre uma fundação sob uso real.

**Quinto: `internal/watcher`.** Até aqui, reindexar no boot basta para validar tudo.

**Sexto: `internal/search`.**

**Sétimo: `internal/writer` e as tools de escrita.** Por último, porque é a única parte que pode destruir dados, e deve ser construída sobre fundações já verificadas.
