# Arquitetura — gobsidian

---

## 1. Visão geral

```
┌──────────────────────────────────────────────────────────┐
│  Host MCP  (Claude Desktop / Claude Code / VS Code)       │
└────────────────────────┬─────────────────────────────────┘
                         │  JSON-RPC 2.0 sobre stdio
                         │  stdout = protocolo  |  stderr = log
┌────────────────────────▼─────────────────────────────────┐
│  cmd/gobsidian — entrypoint, flags, subcomandos               │
├──────────────────────────────────────────────────────────┤
│  internal/lifecycle                                       │
│    EOF em stdin · sinais do SO · vigília do PID pai       │
│    → cancela o context raiz                               │
├──────────────────────────────────────────────────────────┤
│  internal/mcpsrv — adaptação do SDK                       │
│    registro de tools · schemas · tradução de erros        │
├──────────────────────────────────────────────────────────┤
│  internal/service — orquestração (fachada única)          │
│    valida caminhos · coordena índice, busca e escrita     │
├───────────┬───────────┬───────────┬──────────┬───────────┤
│  index    │  search   │  writer   │  watcher │  parser   │
│  memória  │  BM25     │  atômico  │  fsnotify│  goldmark │
├───────────┴───────────┴───────────┴──────────┴───────────┤
│  internal/vault — abstração de caminho e I/O              │
│    resolução, canonicalização, confinamento               │
├──────────────────────────────────────────────────────────┤
│  Sistema de arquivos — o cofre                            │
└──────────────────────────────────────────────────────────┘
```

Duas propriedades estruturais governam o desenho.

**O índice é derivado e descartável.** Nenhuma informação existe apenas em memória. Perder o índice custa uma reindexação, nunca dados. Isso permite que qualquer inconsistência detectada seja resolvida com a estratégia mais simples possível: reconstruir.

**`internal/service` é o único ponto que enxerga todos os subsistemas.** O parser não sabe que existe um índice; o escritor não sabe que existe uma busca. Isso mantém cada uma testável isoladamente e impede que a lógica de orquestração se espalhe.

**Duas exceções, e são de projeto, não descuido.** §5.3 põe `search.Update(nota)` no pipeline do watcher, logo depois de `index.Replace(nota)`: atualizar a busca é parte da reindexação incremental, e mandar isso pelo serviço acrescentaria um salto sem acrescentar isolamento. E §6.2 dá pesos por campo à busca — título 3×, headings 2×, corpo 1× —, o que exige conhecer o título e os headings da nota, isto é, o tipo do índice. Portanto `internal/watcher` importa `internal/search`, e `internal/search` importa `internal/index`, das duas de propósito.

O que **não** vale: `internal/parser` continua folha e não importa ninguém; `internal/index` não conhece a busca (a direção é só de search para index); e nada abaixo do serviço importa `internal/service` — um subsistema que importa a fachada que deveria consumi-lo inverte a dependência, e isso já aconteceu uma vez em `watcher/counters.go`. Dentro de `internal/search`, `analyzer.go` e `persist.go` são folhas e devem continuar sendo: são o que se testa sem construir cofre nenhum.

A redação anterior deste parágrafo — "as camadas abaixo dele não se conhecem", sem exceção — contradizia §5.3 e §6.2, que são mais específicas e mais novas. Corrigida em 2026-07-29, depois de a revisão do M3 acusar como violação algo que a própria arquitetura especifica.

---

## 2. Camadas

### 2.1 `cmd/gobsidian`

Analisa flags, resolve o caminho do cofre, constrói o *context* raiz e despacha para o subcomando. Não contém lógica de domínio.

### 2.2 `internal/lifecycle`

Responsável por uma única coisa: decidir quando o processo deve morrer, e garantir que ele morra. Detalhado em §7.

### 2.3 `internal/mcpsrv`

Camada de adaptação sobre `github.com/modelcontextprotocol/go-sdk/mcp`. Registra as tools, declara os schemas de entrada e saída, traduz erros de domínio em resultados MCP.

Esta camada existe para isolar a instabilidade do SDK. O protocolo MCP evoluiu várias vezes — 2024-11-05, 2025-03-26, 2025-06-18, 2025-11-25, 2026-07-28 — com depreciações reais entre versões. Concentrar o contato com o SDK em um pacote significa que uma quebra de API se resolve em um arquivo, não espalhada pelo código.

**Versão alvo: `2025-11-25`** (RNF-24, D6). É a última revisão estável com suporte pleno no SDK Go oficial, e a que os hosts instalados negociam. O SDK mantém compatibilidade retroativa até 2024-11-05, e a negociação de fallback é dele, não nossa.

A revisão `2026-07-28` é uma mudança estrutural, não incremental: remove o handshake `initialize` e a sessão de protocolo, tornando cada requisição autocontida, e deprecia Roots, Sampling e Logging. Para um servidor stdio local, a statelessness não traz benefício — ela existe para deploys remotos com balanceamento de carga. Migrar cedo custaria compatibilidade com o host instalado em troca de nada.

O contrato desta camada com o resto do sistema é o que torna a migração barata: `internal/service` expõe métodos Go com tipos de domínio e não conhece `mcp.CallToolRequest`, `mcp.CallToolResult` nem versão de protocolo. Se a v2 do SDK inverter a assinatura dos handlers, o dano fica em `mcpsrv/`.

### 2.4 `internal/service`

Fachada. Cada tool MCP corresponde a um método aqui. Responsabilidades:

- Validar e canonicalizar caminhos recebidos (§3.2)
- Consultar o índice ou a busca conforme o tipo de consulta
- Coordenar escritas: obter o estado atual, aplicar a transformação, gravar atomicamente, invalidar a entrada de índice
- Aplicar limites de resultado e paginação

### 2.5 `internal/vault`

Abstração de caminho e I/O bruto. Traduz entre o caminho absoluto do sistema de arquivos e o **caminho canônico** interno.

### 2.6 `internal/parser`

Transforma bytes de um arquivo em uma `ParsedNote`. Puro: sem I/O, sem estado. Recebe `[]byte`, devolve estrutura. Isso o torna trivialmente testável por *golden files* e trivialmente paralelizável.

### 2.7 `internal/index`

Guarda o resultado do parse de todas as notas, mais os grafos derivados (backlinks, tags). Protegido por `sync.RWMutex`. Ver §4.

### 2.8 `internal/search`

Índice invertido para busca full-text. Ver §6.

### 2.9 `internal/watcher`

`fsnotify` mais debounce, coalescência e recuperação de overflow. Ver §5.3.

### 2.10 `internal/writer`

Escritas atômicas e transformações estruturais de conteúdo (inserir sob heading, substituir seção, reescrever links).

### 2.11 `internal/ipc`

Transporte local entre a ponte (`cmd/gobsidian`) e o daemon (`internal/daemon`): chave do socket, handshake de versão e de configuração, `Dial`/`Listen`, proxy de bytes. `runtimeDir` e `restrictPermission` vivem atrás de build tag em `ipc_unix.go`/`ipc_windows.go`; o resto é compartilhado. Ver §7.5.

### 2.12 `internal/daemon`

Um `*mcpsrv.Server` compartilhado por N conexões sobre o socket de `internal/ipc`, para o mesmo cofre. Resolve a corrida de inicialização por arquivo de lock e sai por ociosidade reusando `internal/lifecycle`. Ver §7.5.

### 2.13 `internal/console`

Formata a saída dos comandos de CLI: os marcadores de estado, o realce do texto de ajuda do cobra, e a decisão de usar cor. Sem dependência de terceiros — o `SetConsoleMode` do Windows fica atrás de build tag, em `vt_windows.go`.

Duas regras moldam o pacote. **Os marcadores continuam em ASCII** (`[OK]`, `[!]`, `[i]`, `[*]`, `[...]`) e a cor apenas os reforça, porque um console em CP-850 renderiza o resto como lixo e a informação não pode depender do que se perde. E **a decisão de cor sai do destino**, não de `os.Stdout` global: `doctor > relatorio.txt` grava um arquivo limpo enquanto os erros no `stderr` do terminal continuam coloridos.

`serve` não passa por aqui. Seu stdout pertence ao JSON-RPC, e uma sequência ANSI ali corrompe a sessão do mesmo jeito que um `fmt.Println`.

---

## 3. Modelo de caminhos

### 3.1 Caminho canônico

Todo caminho de nota é representado internamente por uma única forma, o **caminho canônico**:

- Relativo à raiz do cofre
- Separador `/`, sempre, em todas as plataformas
- Sem `./` inicial
- Com a **grafia exata do disco**, incluindo maiúsculas e minúsculas

Exemplo: `Civil/PONTO 03.md`.

A grafia exata do disco importa. Cofres reais acumulam inconsistência de casing entre pastas — `PENAL` ao lado de `Civil` ao lado de `CIVIL` — e o índice precisa refletir o que está no disco, não uma normalização inventada.

### 3.2 Resolução e confinamento

Toda entrada de caminho vinda de uma tool passa por:

1. Normalização de separadores para `/`
2. `path.Clean` para eliminar `..` e `.`
3. Junção com a raiz do cofre e resolução para caminho absoluto
4. **Verificação de confinamento**: o resultado precisa ser prefixo-compatível com a raiz do cofre, comparando componente a componente, nunca por prefixo de string
5. Verificação de link simbólico: se o alvo resolvido sair do cofre, rejeitar

O passo 4 merece atenção. Comparar prefixo de string é a falha clássica: `/cofre-outro` tem `/cofre` como prefixo textual e não é interno a ele. A comparação precisa ser feita por componente de caminho.

### 3.3 Resolução insensível a maiúsculas

Consultas podem chegar com casing divergente do disco. O índice mantém um mapa auxiliar de caminho em minúsculas para caminho canônico. A resolução tenta correspondência exata primeiro; se falhar, tenta insensível; se a busca insensível encontrar mais de um resultado, retorna erro de ambiguidade em vez de escolher arbitrariamente.

### 3.4 Resolução de wikilink

Wikilinks não são caminhos. `[[PONTO 03]]` pode se referir a qualquer nota chamada `PONTO 03.md` em qualquer pasta. A resolução segue a ordem do Obsidian:

1. Correspondência exata do caminho relativo, se o link contiver `/`
2. Correspondência de nome de arquivo, com extensão `.md` implícita
3. Correspondência de nome de arquivo de **anexo**, com a extensão explícita — `![[diagrama.png]]` resolve para o anexo, não para uma nota (RF-60, RF-61)
4. Correspondência por **alias**: o campo `aliases` do frontmatter de alguma nota contém o alvo do link (RF-62)
5. Se houver mais de um candidato, o mais próximo da nota de origem na árvore de diretórios
6. Não encontrado → link quebrado, registrado como tal no grafo

A ordem importa. Nome de arquivo real tem precedência sobre alias: se existe `P3.md` e outra nota declara `aliases: [P3]`, `[[P3]]` aponta para o arquivo. Alias é fallback, nunca override — é assim que o Obsidian se comporta, e inverter produziria um grafo que diverge de forma invisível.

Colisão entre aliases de notas diferentes não tem resposta correta. O índice registra ambos os candidatos, resolve pelo mais próximo na árvore como no passo 5, e contabiliza a colisão em `vault_stats` para que ela seja diagnosticável em vez de silenciosa.

### 3.5 Anexos

Anexos — imagens, PDFs, áudio, `.canvas` — entram no índice como entradas de diretório: caminho canônico, tamanho, mtime. Nunca são lidos, nunca são parseados, nunca entram no índice de texto.

O motivo é estreito e suficiente: sem eles, todo `![[imagem.png]]` do cofre vira link quebrado, e a contagem de links quebrados em `vault_stats` — que é o principal sinal de saúde do cofre — fica dominada por falsos positivos e deixa de ter uso.

### 3.6 Âncoras

Um link pode resolver para uma nota existente e ainda assim estar quebrado, se a âncora não existir: `[[nota#Capítulo 9]]` para uma nota que não tem esse heading. A resolução de âncora acontece depois que todas as notas estão indexadas, porque depende da lista de headings e blocos do alvo.

O estado do link tem então três valores, não dois: resolvido, alvo inexistente, alvo existente com âncora inexistente. O terceiro é o que aparece depois de renomear um heading, e é invisível até alguém clicar no link.

---

## 4. Índice em memória

### 4.1 Estruturas

```go
type Index struct {
    mu        sync.RWMutex
    notes     map[string]*Note       // caminho canônico -> nota
    assets    map[string]*Asset      // caminho canônico -> anexo (RF-60)
    lowerPath map[string]string      // caminho minúsculo -> canônico
    byName    map[string][]string    // nome base -> caminhos candidatos
    byAlias   map[string][]string    // alias minúsculo -> caminhos (RF-62)
    backlinks map[string][]Backlink  // alvo -> origens
    tags      map[string][]string    // tag -> caminhos
    generation uint64                // incrementa a cada mutação
}

type Asset struct {
    Path    string    // canônico
    Size    int64
    ModTime time.Time
}

type Note struct {
    Path     string            // canônico
    Size     int64
    ModTime  time.Time
    Hash     uint64            // xxhash do conteúdo
    EOL      EOLStyle          // CRLF ou LF, preservado nas escritas

    Frontmatter map[string]any
    Tags        []string       // inline + frontmatter, deduplicadas
    Headings    []Heading
    Blocks      []Block
    Links       []Link
    Inline      map[string][]string // campos Dataview
    CloudOnly   bool           // arquivo não hidratado (OneDrive)
}

type Heading struct {
    Level   int
    Text    string
    Slug    string  // normalizado para casar com âncoras de wikilink
    Start   int64   // offset do byte do '#'
    End     int64   // offset do fim da seção (início do próximo heading de nível <=)
}

type Block struct {
    ID    string  // sem o '^'
    Start int64
    End   int64
}

type Link struct {
    Raw     string   // texto original, para reescrita fiel
    Target  string   // alvo bruto, antes da resolução
    Alias   string
    Anchor  string   // heading ou ^bloco
    Kind    LinkKind // wikilink, embed, markdown
    Start   int64
    End     int64
    Resolved string     // caminho canônico, ou "" se quebrado
    Via      ResolveVia // como resolveu: path, name, asset, alias
    State    LinkState  // ok, target_missing, anchor_missing
}
```

### 4.2 O que o índice não guarda

**O corpo das notas.** Guardar 50 MB de texto em memória para relê-lo do disco de qualquer forma na hora de exibir é desperdício. O índice guarda deslocamentos de byte; `note_read` faz `ReadAt` na região pedida.

Essa decisão é o que sustenta RNF-07 (60 MB de RSS): o índice de um cofre de 5.000 notas fica na casa de poucas dezenas de megabytes porque armazena estrutura, não conteúdo.

### 4.3 Backlinks

Backlinks são derivados, reconstruídos a partir dos links de saída. Quando uma nota é reindexada, suas contribuições anteriores ao mapa de backlinks são removidas antes que as novas sejam inseridas. Manter isso correto exige que cada entrada de backlink carregue o caminho da origem, o que a estrutura `Backlink` faz.

Links quebrados são mantidos no grafo, não descartados. Isso é o que permite `vault_stats` reportá-los e o que permite que um link passe a resolver automaticamente quando a nota alvo for criada depois.

### 4.4 Concorrência

`sync.RWMutex`. Leituras — a esmagadora maioria das operações — são concorrentes. Escritas no índice acontecem apenas na thread do watcher, serializadas.

O padrão de uso é fortemente assimétrico: dezenas de leituras por escrita. `RWMutex` é a escolha certa; estruturas *lock-free* seriam complexidade sem retorno mensurável nessa faixa.

O campo `generation` incrementa a cada mutação e permite que uma consulta longa detecte que o índice mudou sob ela.

### 4.5 Serialização de escrita

O `RWMutex` do índice protege a estrutura de dados. Ele não protege o arquivo em disco, e essa distinção é onde mora um bug fácil de introduzir.

Um host MCP pode emitir chamadas de tool concorrentes. Duas chamadas de `note_append` na mesma nota executam a sequência ler-transformar-gravar em paralelo, cada uma partindo do mesmo conteúdo inicial, e a segunda gravação apaga a primeira. Nenhuma nota é corrompida — a escrita atômica garante isso — e mesmo assim uma inserção desaparece sem erro. Silencioso é o pior tipo de perda.

A defesa é um mapa de mutexes por caminho canônico, mantido em `internal/writer`, adquirido antes da leitura e liberado após o rename. Escritas em notas diferentes seguem paralelas; escritas na mesma nota serializam.

Isso não resolve — e não pretende resolver — a concorrência com o Obsidian, que é outro processo (N5). Para essa, o instrumento é `expected_hash`, e ele é complementar: o mutex ordena o que está dentro do nosso processo, o hash detecta o que veio de fora.

O campo `Note.Hash` é o valor exposto como `hash` nos retornos de `note_read`, `note_metadata` e `note_list`, e é o que o cliente devolve em `expected_hash`. É xxhash do conteúdo bruto do arquivo, incluindo frontmatter e BOM — qualquer byte diferente produz hash diferente.

### 4.6 Índice Invertido e Analisador Morfológico (`internal/search`)

O motor de busca em texto completo implementa ranking BM25 com indexação dupla para o idioma português:

- **Analisador Morfológico (`analyzer.go`)**: Normalização de maiúsculas/acentos via `golang.org/x/text/unicode/norm` e `transform.Chain`. Indexação dupla de cada token: forma crua (preservando distinção exata de vocabulário) e forma reduzida por 8 regras morfológicas conservadoras para o português (flexões nominais/verbais sem colidir radicais jurídicos distintos).
- **Índice Invertido (`inverted.go`, `soa.go`)**: Estrutura thread-safe em **duas camadas**, sob um `sync.RWMutex`. A `base` é imutável e vem do cache, em arrays achatados com busca binária (`soa.go`); o `delta` é um `map[termo]map[caminho][]TokenPosition` pequeno, com o que mudou desde a partida. Toda leitura consulta as duas e o delta ganha; toda escrita vai só ao delta e marca o caminho como substituído na base. Construído do zero — sem cache — a base é nil e o delta é o índice inteiro, que é o caminho da construção em segundo plano.

  A separação existe porque as duas metades têm naturezas opostas: o que vem do cache é grande e não muda (126 mil termos e 3 milhões de postings no cofre de referência), o que muda é minúsculo. Uma estrutura única obrigava a grande a ser mutável, e mutável significava um mapa por termo — 35% do tempo de carga e a maior parte da memória. A base carrega ainda um índice direto (documento → termos), que faz substituir uma nota custar os termos daquela nota em vez de uma varredura dos 126 mil.
- **Ranking BM25 (`bm25.go`)**: $k_1 = 1.2$, $b = 0.75$. Pesos de campo: Título 3.0x, Headings 2.0x, Corpo 1.0x. Pontuação em forma dupla: $Score = Score_{raw} \times 1.5 + Score_{reduced} \times 1.0$. Desempate determinístico por caminho de nota.
- **Recorte de Snippets (`snippet.go`)**: Extração do trecho com maior densidade de termos casados, com compensação de offset de BOM (+3 bytes no disco) e truncamento UTF-8 seguro em limite configurável (`snippet_chars`).

### 4.7 Persistência de Cache em Disco (`internal/search/persist.go`)

O cache do índice fica armazenado fora do cofre em `%LOCALAPPDATA%\gobsidian\<hash-do-caminho-do-cofre>\inverted_cache.gob` (decisão D1):

- **Codec binário próprio (`persist_codec.go`), formato 5.** A extensão do arquivo continua `.gob` por compatibilidade de caminho, mas **o conteúdo não é `gob` desde 2026-08-03**. O formato 1 era `encoding/gob` sobre os mapas, e a reflexão dele custava 63,9% cumulativos do tempo de carga. O formato corrente grava: assinatura `GBS5`, as três versões, uma **tabela de caminhos** escrita uma vez (o caminho repetido por posting era metade do arquivo), os totais adiantados de postings e posições, `docLengths`, e as posições como **varint sobre o delta** da anterior. Tudo em **ordem crescente** — é o que permite ao leitor montar os arrays achatados com busca binária, e o leitor confere as três ordens, porque um arquivo fora de ordem não falharia sozinho: responderia "termo não existe" para termos que existem.
- **Cabeçalho de Integridade (`CacheHeader`)**: Guarda `FormatVersion`, `ParserVersion`, `AnalyzerVersion`, `VaultPath` e `NoteCount`. A versão de formato é conferida **logo após a assinatura**, antes de qualquer campo de layout: o layout já mudou dentro da mesma família, e decodificar primeiro faria os varints casarem com campos trocados, produzindo lixo estruturalmente válido ou um erro de corrupção que culpa o disco por uma troca de formato. Qualquer divergência rejeita o arquivo e força reindexação limpa, sem migração.
- **Escrita Atômica**: Gravado primeiro em um temporário no mesmo diretório com o prefixo `.gobsidian-tmp-cache-*.gob` (ignorado pelo filtro de watcher) e renomeado atomicamente para `inverted_cache.gob` via `os.Rename`.
- **Diferenciação de Estado**: Cofres vazios (0 notas) geram cache válido com `NoteCount = 0`, distinguível de cache ausente (`ErrCacheNotFound`).
- **Medição.** No cofre real de 3.152 notas / 109 MB: arquivo de 505.643.791 B → 70.084.435 B, carregamento de 5,59 s → **659,2 ms** (±23%, n=6), 13.035.004 → 291.104 alocações. **RNF-02 segue não atingido**: 832–1183 ms contra teto de 300 ms. As medições completas e o que cada peça do formato paga estão em [`OPERACAO.md`](OPERACAO.md).

---

## 5. Fluxos

### 5.1 Boot

```
1. Analisar flags, resolver e validar o caminho do cofre
2. Iniciar lifecycle: instalar handlers de sinal, monitor de stdin, vigília do PID pai
3. Tentar carregar cache de índice do disco
   3a. Cache válido  → validar por mtime+tamanho de amostra; reparsear divergências
   3b. Cache ausente → varredura completa
4. Varredura completa:
   - filepath.WalkDir aplicando as exclusões (.obsidian, .git, .trash, ignore)
   - enfileirar caminhos em um canal
   - worker pool de runtime.NumCPU() workers lendo e parseando
   - resultados coletados em um único goroutine que popula o índice
5. Construir backlinks e mapas auxiliares
6. Construir ou carregar o índice de busca
7. Iniciar o watcher
8. Iniciar o servidor MCP em stdio
```

O passo 8 é o último. O servidor só anuncia disponibilidade quando pode responder. Um servidor que aceita `initialize` antes de estar indexado responde a primeira chamada com erro ou com dados incompletos — pior que um boot 200 ms mais lento.

### 5.2 Leitura (`note_read` de uma seção)

```
service.ReadNote(path, heading)
  → vault.Resolve(path)            valida confinamento, resolve casing
  → index.Get(canonical)           RLock; devolve *Note
  → localizar Heading por slug     comparação normalizada
  → os.OpenFile + ReadAt(start, end-start)
  → devolver bytes + metadados da seção
```

Nenhuma alocação do arquivo inteiro. Uma leitura de seção de 2 KB em uma nota de 500 KB custa 2 KB.

### 5.3 Watch e reindexação incremental

```
fsnotify.Watcher
  → canal bruto de eventos
  → filtro de relevância (extensão .md, fora dos diretórios excluídos)
  → debouncer: tique único e conjunto sujo, janela da configuração (padrão 250 ms)
  → coalescência: N eventos no mesmo arquivo dentro da janela = 1 reparse
  → verificação de mudança real: mtime e tamanho iguais aos indexados? descartar
  → reparse do arquivo
  → index.Replace(nota): remove contribuições antigas, insere novas
  → search.Update(nota)
  → generation++
```

Três camadas de filtragem antes de qualquer trabalho real. Isso é o que torna o cofre em OneDrive utilizável: o sincronizador gera rajadas de eventos, muitos deles sobre arquivos que não mudaram de conteúdo, e a verificação de mtime e tamanho descarta a maioria antes que custem um parse.

O debouncer utiliza um **tique único e um conjunto sujo** (`map[vault.CanonicalPath]struct{}`), e não um timer por arquivo. Isso resolve dois problemas graves sob carga:
1. **Rajada do OneDrive:** Uma sincronização inicial toca milhares de arquivos. Um timer por arquivo exigiria criar e destruir milhares de objetos de runtime exatamente no pico de pressão. Um único `time.Ticker` mantém o custo constante.
2. **Inanição (Starvation):** Um timer por arquivo, se reiniciado a cada novo evento, nunca dispararia enquanto o arquivo estivesse sendo escrito continuamente (ex: download grande). O tique único garante que, a cada janela, o conjunto sujo é esvaziado e o reparse acontece.

**Tratamento de overflow.** `ReadDirectoryChangesW` no Windows tem buffer finito. Sob rajada intensa, `fsnotify` emite `ErrEventOverflow`, e eventos foram perdidos — não se sabe quais. A única resposta correta é uma varredura completa de reconciliação: percorrer o cofre, comparar mtime e tamanho com o índice, reparsear divergências, remover notas que sumiram. Ignorar o overflow deixa o índice silenciosamente incorreto, que é o pior estado possível.

A reconciliação repara **os dois índices**, o de metadados e o de busca. Até 2026-08-04 ela reparava só o de metadados, e isso não era uma lacuna cosmética: `service.Search` descarta a posting cujo caminho não existe no índice de metadados, então uma nota movida durante o overflow ficava com os metadados no caminho novo e a posting no antigo, e `vault_search` devolvia **zero resultados** para ela, para sempre. Pela mesma razão o atalho de "mtime e tamanho iguais, pular" exige também presença no índice de busca: divergência entre os dois é a premissa desta varredura, e pular por "já está atualizada" fazia o anteparo confirmar o defeito.

**Diretório novo é varrido.** Quando um diretório é criado dentro do cofre, o watcher registra o watch nele **e varre o que já está lá dentro**, emitindo criação para cada arquivo e registrando watch nos subdiretórios. Sem isso, uma pasta que chega ao cofre já com arquivos entrega exatamente um evento — a criação do próprio diretório — e nenhum arquivo, porque eles existiam antes de o watch existir. Não é caso de borda: é o usuário arrastando uma pasta para o cofre, e era também o que fazia `note_move` perder a nota quando o rename chegava antes do registro do watch. A varredura roda no laço de eventos, de propósito: numa pasta grande ela segura o consumo e pode estourar o buffer do sistema operacional, o que dispara justamente a reconciliação acima.

**Correlação de Renames.** `fsnotify` reporta rename como um par de eventos (remoção na origem, criação no destino) sem correlação explícita. O debouncer entrega um lote de caminhos ao `Apply`. O `Apply` separa esses caminhos entre deletados (não existem mais no disco) e modificados/criados. Arquivos deletados e criados no *mesmo lote* que possuam exato o mesmo hash `xxhash` (calculado sobre o conteúdo bruto antes da remoção de BOM) e não sejam vazios são correlacionados como um rename em nível lógico, preservando o hash e marcando backlinks pendentes. Limitações documentadas:
1. Cópias seguidas de remoções do original caem como rename se acontecerem na mesma janela de debounce.
2. Remoção e criação separadas por tempo maior que o debounce não correlacionam.
3. Apenas notas são hasheadas e correlacionadas, anexos e mídias não. Anexo e arquivo somente-nuvem **nunca são abertos** para correlacionar: ler um placeholder dispara download síncrono, e ler um anexo viola a regra de que anexo é indexado por nome. Os dois são classificados antes de qualquer leitura e caem direto no caminho comum.
4. Qualquer modificação de conteúdo simultânea ao move anula o rename lógico e ele decai para um remover e adicionar comum (remove+replace).
5. Notas vazias nunca correlacionam. O gate `Size > 0` existe porque arquivos vazios compartilham hash e a cardinalidade 1-para-1 sozinha não distingue rename de coincidência. Um rename de nota vazia decai para remove + create.
6. macOS e BSD não sinalizam overflow: o backend kqueue do `fsnotify` v1.10.1 nunca emite `ErrEventOverflow` — só `backend_inotify.go` e `backend_windows.go` emitem. Nessas plataformas a reconciliação descrita acima nunca dispara, e o único anteparo contra evento perdido é a reindexação no boot. Lacuna registrada, não resolvida por heurística. <!-- check-doc-refs: ignore backend_inotify.go, backend_windows.go -- arquivos do fsnotify v1.10.1, dependencia externa -->

**O cofre nunca é escrito.** O que o rename correlacionado altera é o índice, in-place, via `index.MoveNote`: a nota muda de caminho preservando hash, tags, aliases e backlinks, e as notas que apontavam para o caminho antigo passam a resolver para o novo. Os links no disco continuam apontando para o caminho antigo e são reportados como candidatos a atualização — reescrevê-los seria decidir conteúdo pelo usuário.

### 5.4 Escrita (`note_patch` sob um heading)

```
service.PatchNote(path, heading, content, expectedHash, dryRun)
  → vault.Resolve(path)
  → writer.LockPath(canonical)            serializa escritas na mesma nota (§4.5)
  → index.Get(canonical)                  localiza Heading{Start,End}
  → ler arquivo inteiro                   (escrita exige o conteúdo completo)
  → expectedHash informado e divergente?  → HASH_MISMATCH, parar
  → verificar hash contra o índice        detecta modificação externa desde o parse
       divergente → reparsear e recalcular offsets antes de prosseguir
  → montar novo conteúdo:
       prefixo [0:heading.End_do_titulo] + conteúdo novo + sufixo [heading.End:]
  → normalizar EOL do conteúdo novo para o estilo do arquivo
  → dryRun? devolver diff unificado e parar
  → writer.AtomicWrite(path, bytes)
  → invalidar entrada de índice (o watcher confirmará)
  → writer.UnlockPath(canonical)
```

A verificação de hash antes de aplicar o patch fecha a janela entre o parse e a escrita. Sem ela, uma edição feita no Obsidian entre as duas operações faria os deslocamentos apontarem para o lugar errado, e o patch cortaria a nota no meio de uma frase.

### 5.5 Escrita atômica

```go
// mesmo diretório do alvo → mesmo volume → rename atômico
tmp := filepath.Join(filepath.Dir(target), ".gobsidian-tmp-"+random())

f, _ := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
f.Write(data)
f.Sync()          // durabilidade antes do rename
f.Close()
os.Rename(tmp, target)   // com retry: OneDrive devolve ERROR_SHARING_VIOLATION transitório
```

O temporário fica no mesmo diretório do alvo, não em `%TEMP%`. Rename entre volumes não é atômico — degrada para cópia mais exclusão, e o ponto todo se perde.

`Sync()` antes do rename garante que o conteúdo esteja em disco antes que o nome passe a apontar para ele. Sem isso, um corte de energia entre rename e flush deixa um arquivo de tamanho correto cheio de zeros.

O retry no rename é específico do Windows com sincronizadores de nuvem, que abrem arquivos brevemente e devolvem violação de compartilhamento. Backoff exponencial, três tentativas, 50 ms iniciais.

### 5.6 `note_move` com reescrita de links

```
1. Resolver origem e destino; destino não pode existir
2. Consultar index.Backlinks(origem) → lista de notas afetadas
3. Para cada nota afetada, calcular a reescrita de cada Link:
     preservar alias, âncora e a forma original (wikilink vs markdown)
     recalcular o alvo mínimo não-ambíguo em relação ao novo caminho
4. dryRun? devolver o conjunto de diffs e parar
5. Mover o arquivo (rename atômico)
6. Escrever cada nota afetada atomicamente
7. Em caso de falha parcial: registrar o que foi aplicado, retornar erro detalhado
```

O passo 7 é honesto sobre um limite real: não há transação entre arquivos no sistema de arquivos. O produto garante que cada arquivo individualmente fica consistente, e reporta com precisão o estado alcançado quando algo falha no meio. Prometer atomicidade multi-arquivo seria mentira.

O campo `Link.Raw` existe exatamente para o passo 3: permite reescrever preservando a forma que o usuário escreveu, em vez de normalizar tudo para uma forma canônica e produzir um diff gigante.

---

## 6. Busca

### 6.1 Duas camadas

**Consultas de metadados** — por tag, por pasta, por campo de frontmatter, por glob de caminho — são servidas direto do índice em memória. Não tocam o índice de texto. Latência na casa do microssegundo.

**Consultas full-text** vão ao índice invertido.

Separar as duas evita o erro comum de mandar tudo para o motor de busca, o que torna `note_list --tag civil` cem vezes mais caro do que precisa ser.

### 6.2 Índice invertido

Índice invertido próprio, não uma dependência de motor de busca completo.

O raciocínio: um motor completo traz *query language*, facetas, agregações, persistência transacional e um custo de dependência considerável. O requisito real é BM25 sobre um corpus de dezenas de megabytes com filtros simples. Um índice invertido com dicionário de termos, listas de postagem e BM25 ocupa algumas centenas de linhas e é integralmente compreensível.

```
Analisador:
  → segmentação por limites de palavra Unicode
  → case folding
  → remoção de diacríticos (NFD + descarte de marcas combinantes)
  → emite a forma crua normalizada
  → emite também a forma reduzida, quando diferente da crua
       redução conservadora: plurais regulares e sufixos verbais comuns
  → sem stopwords (termos jurídicos frequentes não são ruído)

Índice:
  termo → []Posting{noteID, tf, positions}
  BM25 com k1=1.2, b=0.75
  campos com peso: título 3x, headings 2x, corpo 1x
  forma crua pontua acima da forma reduzida no mesmo documento
```

A remoção de diacríticos não é opcional em um cofre em português. Buscar "usucapiao" precisa encontrar "usucapião". Ausência de stopwords é uma escolha de domínio: em um corpus jurídico, remover palavras frequentes remove justamente os termos de arte.

**Indexação dupla** (RF-23, D9). Cada token entra no índice duas vezes quando a redução muda alguma coisa: `prescrições` gera `prescricoes` e `prescric`. A consulta passa pelo mesmo analisador e casa com qualquer uma das duas.

Isso resolve o dilema entre recall e precisão sem escolher um dos lados. Um stemmer agressivo aplicado destrutivamente — Snowball, RSLP — funde termos de arte que só se distinguem pelo sufixo, e o usuário não tem como perceber por que a busca trouxe o resultado errado. Não reduzir nada faz "prescrições" não encontrar "prescrição". Guardar as duas formas custa cerca de 30% no dicionário de termos, número irrelevante nesta escala, e mantém a forma que o usuário digitou pontuando acima da aproximação.

A busca por frase exata (RF-24) percorre apenas as posições da forma crua. Casar frase com stems produziria correspondências que o usuário não pediu e não consegue explicar.

### 6.3 Persistência

O índice de busca é serializado para disco em `%LOCALAPPDATA%gobsidian\<hash-do-caminho-do-cofre>\`, **fora do cofre**. Colocá-lo dentro faria o Obsidian tentar indexá-lo e o OneDrive tentar sincronizá-lo — ambos indesejáveis.

Invalidação por mtime e tamanho por arquivo. Na carga, uma amostra é verificada; se a taxa de divergência passar de um limiar, reconstrói tudo em vez de reconciliar arquivo a arquivo.

**Versão de esquema.** O cabeçalho do cache carrega três números, e qualquer divergência descarta o cache inteiro sem tentar reconciliar:

| Campo | Muda quando |
|---|---|
| `FormatVersion` | O layout serializado muda |
| `ParserVersion` | O parser passa a produzir estrutura diferente para a mesma entrada |
| `AnalyzerVersion` | O analisador de busca muda tokenização, normalização ou redução |

Os nomes são os do código (`internal/index/persist.go`), e não uma grafia separada por sublinhados: o cache não é JSON, é um codec binário próprio, e não existe chave serializada com outro nome.

Sem `ParserVersion`, corrigir um bug de parsing e reiniciar carregaria de volta o índice errado — mtime e tamanho dos arquivos não mudaram, então nada indicaria invalidação. O cache continuaria servindo o resultado do parser antigo, e o bug pareceria não corrigido. É a classe de falha mais confusa que um cache pode produzir, e custa três inteiros evitá-la.

Esses números são constantes no código, incrementadas à mão. Derivá-los de hash do código-fonte invalidaria o cache a cada recompilação, inclusive quando nada relevante mudou.

---

## 7. Ciclo de vida do processo

Esta é a seção que justifica o projeto. Três mecanismos independentes, todos cancelando o mesmo *context* raiz.

### 7.1 EOF em stdin

O mecanismo primário e mais confiável. Quando o host MCP encerra, ele fecha o stdin do filho. Um goroutine lê stdin; ao receber `io.EOF`, cancela o context.

Isso funciona mesmo quando o host morre sem enviar sinal algum, porque o sistema operacional fecha os *handles* do processo morto.

### 7.2 Sinais do sistema operacional

`signal.Notify` para `os.Interrupt` e `syscall.SIGTERM`. No Windows, `os.Interrupt` cobre `CTRL_C_EVENT` e `CTRL_BREAK_EVENT`. Cobre o encerramento cooperativo e o uso via terminal.

Não cobre `taskkill /F` nem `SIGKILL` — nada cobre, e não precisa: nesses casos o processo morre de fato, que é o resultado desejado.

### 7.3 Vigília do PID pai

O mecanismo de rede de segurança, para o caso em que o host morre de forma que deixe o stdin do filho sem fechar — reparentamento, herança de handle por outro processo, comportamento anômalo do host.

```
No startup: capturar os.Getppid()
A cada 5 s: verificar se o processo pai ainda existe E ainda é o mesmo processo
Se não: cancelar o context
```

A verificação de identidade importa. PIDs são reutilizados. Verificar apenas existência produz falso negativo quando o PID do pai morto é reciclado por um processo novo. No Windows, isso se resolve comparando o *creation time* do processo além do PID.

### 7.4 Shutdown

```
context cancelado
  → parar de aceitar novas chamadas MCP
  → aguardar chamadas em voo                       orçamento: 3 s
  → fechar o watcher                               libera handles do sistema de arquivos
  → fechar arquivos abertos
  → persistir o índice em cache                    orçamento: 2 s, best-effort
  → os.Exit(0)
```

**Orçamento total de 5 s, repartido.** Um shutdown que trava esperando uma operação lenta é funcionalmente idêntico a um processo órfão. Um timeout global único não basta: se as chamadas em voo consumirem os 5 s inteiros, a persistência do cache começa já estourada e ou é pulada sem critério ou estoura o prazo. Cada etapa tem seu próprio orçamento, e o relógio de cada uma começa quando ela começa.

**Ordem invertida em relação ao intuitivo.** Liberar os handles vem antes de persistir o cache, não depois. Handles abertos são o sintoma que o produto existe para eliminar — travam a pasta e atrapalham o OneDrive. O cache é uma otimização de boot: perdê-lo custa alguns segundos no próximo início e nada mais.

**Persistência é best-effort e nunca bloqueia.** Se estourar os 2 s, é abandonada e o processo encerra. Um cache parcialmente escrito seria pior que nenhum, então a gravação é atômica pelo mesmo mecanismo das notas: temporário, sync, rename. Ou o cache anterior permanece, ou o novo aparece inteiro.

**Um guarda-chuva final.** Um `time.AfterFunc(6*time.Second, func(){ os.Exit(1) })` armado no início da sequência. Se qualquer etapa travar de um jeito que o orçamento dela não previu, o processo morre mesmo assim. Encerrar com código de erro é ruim; sobreviver ao pai é pior.

### 7.5 Daemon e transporte IPC local

Cada sessão de host MCP (Claude Desktop, Claude Code, VS Code) abre um processo `gobsidian serve` novo. Antes desta seção existir, duas sessões contra o mesmo cofre pagavam o índice completo duas vezes — medido em ~1 GB agregado num cofre real antes da carga sob demanda da Tarefa 88. A Tarefa 88 já resolve a maioria dos casos (sessão que só lê e escreve nota nunca carrega o índice de busca); o que sobra depois dela é a sessão que **busca de verdade**, contra o **mesmo cofre**, ao **mesmo tempo** — e é isso que o daemon ataca.

**Um processo por cofre, não um processo global.** `EnsureStarted` deriva a chave do cofre pela mesma `config.VaultKey` que `ipc.SocketPath` usa (uma função, não duas contas do mesmo hash — a classe de defeito que `byAlias` já pagou aqui). O daemon monta `index`, `watcher` e o serviço de domínio uma única vez (`construirServico`, compartilhada com o boot em processo para as duas sequências nunca divergirem) e aceita N conexões sobre o socket.

**A ponte é burra.** `cmd/gobsidian` primeiro tenta discar o socket do cofre; se conectar, vira um proxy de bytes entre o stdio do host e a conexão — não interpreta JSON-RPC. Se a discagem falhar (socket ausente, conexão recusada, versão ou configuração incompatível), tenta `EnsureStarted` e disca de novo; se isso também falhar, cai para o modo em processo de sempre (`serveEmProcesso`). O fallback é obrigatório nos três pontos, não só no primeiro — nenhum caminho deixa a ferramenta inutilizável por um socket quebrado. Medido: a ponte sozinha, conectada, custa ~13,8 MB de Working Set contra ~250 MB de uma instância completa.

**Handshake carrega versão e configuração.** Além da versão de protocolo, `ipc.HandshakeConfig` carrega `ReadOnly`, `VaultKey` e `MaxResults`. Duas pontes do mesmo cofre com `--read-only` divergente conectadas ao mesmo daemon seria bug de segurança, não detalhe — o handshake recusa (`ErrConfigMismatch`) em vez de aceitar.

`MaxResults` entrou em 2026-08-28, com o **protocolo na versão 2** (achado M9). A flag `--max-results` da ponte era no-op silencioso no modo daemon: não ia para o spawn e não entrava no handshake, então valia o `cfg` do PRIMEIRO daemon e a segunda ponte era atendida com um teto que ninguém pediu. O handshake é unidirecional, então a ponte não pode IMPOR o dela; ela recusa e cai para o modo em processo, onde a configuração dela vale. Divergência visível é melhor que silêncio.

**Um segundo lock serializa quem ABRE o socket**, distinto do lock de `EnsureStarted`, que serializa quem LANÇA o daemon. `ipc.Listen` prova que o socket está órfão antes de desvinculá-lo, mas a sonda e o bind não são atômicos entre si: dois daemons lançados no mesmo instante podem ambos sondar "ninguém escuta" antes de qualquer um bindar. Usar o mesmo arquivo para as duas perguntas trava uma na outra — `EnsureStarted` segura o lock até `esperarSocket` devolver, e `esperarSocket` espera o socket que o daemon recém-lançado só pode abrir depois de adquirir o lock.

**Ociosidade, não sinal nem pai.** O daemon não tem stdin de host nem pai vigiável, então dos três mecanismos de §7 só o cancelamento de context se aplica; `lifecycle.Trigger` (exportado para isso) é chamado quando nenhum cliente está conectado por mais que `OciosidadeMax` (padrão de produção: 900 s / 15 min). Reusa a mesma infraestrutura de log e cancelamento que sinal e EOF usam nos outros processos, em vez de um mecanismo paralelo.

**`GOBSIDIAN_NO_DAEMON=1`** pula a decisão inteira e força o caminho em processo de sempre — é como desligar o daemon sem reverter código, e é o que as medições de §-comparação (`docs/OPERACAO.md`) usam para o "antes".

#### A escolha do transporte — D-M7-6

Medida antes de escrever código, não depois. Eco de ida e volta, 20.000 repetições por tamanho, `windows/amd64`, mesmo código nos três sistemas:

| Transporte | 256 B | 4 KB | 64 KB |
|---|---|---|---|
| **AF_UNIX** (`net.Dial("unix")`) | **25,7 µs** | **23,0 µs** | **42,9 µs** |
| named pipe (`go-winio`, config padrão) | 82,9 µs | 93,5 µs | 110,0 µs |

AF_UNIX ganhou em todos os tamanhos, por 3 a 4×, está na biblioteca padrão (sem dependência nova) e usa o mesmo código nos três sistemas — Windows suporta AF_UNIX desde a versão 10 1803 (abril de 2018). A margem quase não importa: a ida e volta custa dezenas de microssegundos contra uma busca real de 90-200 ms (RNF-04), quatro ordens de grandeza de distância. Build tag continua existindo para o caminho do socket e a limpeza dele (Windows deixa um arquivo que precisa ser removido), não para o transporte em si.

O par `net.Dial`/`net.Listen` restrito à constante literal `"unix"` é o que a garantia reformulada do RNF-30 exige e o `netcheck` (PRD §6.4) verifica estaticamente. Ver ali para a regra completa e a razão de produto por trás dela.

#### Risco residual conhecido

`EnsureStarted` resolve a corrida de inicialização por arquivo de lock (`O_CREATE|O_EXCL`), mas o lock é liberado assim que a chamada da própria ponte termina — o que inclui esperar o socket responder. Isso serializa quem disputa o **mesmo instante**, não "existe um daemon rodando". Sob carga pesada da máquina, dez pontes lançadas juntas contra o cofre real produziram, uma vez, dois daemons vivos simultaneamente antes da correção (quase um minuto de intervalo entre os dois "daemon iniciado" no log — uma ponte atrasada pelo agendamento do SO via o lock livre e vencia uma corrida nova depois que o primeiro já tinha subido, sem nunca tê-lo visto). A correção adiciona uma segunda checagem (um dial) logo após adquirir o lock, antes de chamar `iniciar`: se alguém já respondeu enquanto esta chamada esperava a vez, usa esse, nunca inicia um segundo.

**Isso reduz a janela, não a elimina por construção.** É um *check-then-act* de milissegundos (entre o dial de confirmação e o `SpawnDetached` de fato), não exclusão mútua entre processos pelo tempo de vida inteiro do daemon. Reconfirmado sem daemon duplicado em duas rodadas de dez pontes simultâneas depois da correção, em máquina sem carga concorrente — mas a janela teórica continua existindo sob a mesma condição que a expôs uma vez (contenção pesada de CPU no exato instante da corrida). Documentado também como limite conhecido em `docs/OPERACAO.md`, com o número.

---

## 8. Tratamento de erros

### 8.1 Erros não derrubam o servidor

Uma tool que falha devolve um resultado MCP de erro. O servidor continua. `panic` em handler é recuperado por middleware, registrado com stack trace em stderr, e convertido em erro de tool.

Isso é RNF-13, e é o que distingue um servidor robusto de um que exige reinício do Claude Desktop toda vez que um caminho inválido é passado.

### 8.2 Taxonomia

| Categoria | Exemplo | Resposta |
|---|---|---|
| Entrada inválida | caminho fora do cofre, heading inexistente | Erro de tool, mensagem acionável |
| Conflito de estado | arquivo mudou entre parse e escrita | Reparsear e repetir; se persistir, erro |
| Transitório de I/O | violação de compartilhamento do OneDrive | Retry com backoff; erro após esgotar |
| Inconsistência de índice | offset inválido, nota ausente do mapa | Reindexar a nota; se persistir, reindexar o cofre |
| Programação | panic | Recuperar, registrar, erro genérico ao cliente |

### 8.3 Mensagens

Mensagens de erro são lidas por um modelo de linguagem que precisa decidir o que fazer em seguida. Precisam ser acionáveis.

Ruim: `heading not found`.
Bom: `heading "Capítulo 118" não encontrado em "Resumo - Claude - Processo Penal.md"; headings de nível 2 disponíveis: "Capítulo 115", "Capítulo 116", "Capítulo 117"`.

A segunda permite que o cliente se corrija sozinho. A primeira gera uma rodada extra de chamadas.

---

## 9. Decisões arquiteturais

### AD-01 — Go, não Rust

Rust venceria em desempenho bruto e em rigor de gerenciamento de memória. Go vence no que importa para este projeto: `goroutines` e canais tornam o pipeline de indexação paralela e o debouncer do watcher código curto e óbvio; a biblioteca padrão cobre I/O de arquivo, sinais e JSON sem dependências; o tempo de compilação mantém o ciclo de iteração rápido; e o SDK oficial de MCP para Go é mantido em colaboração com o Google.

A diferença de desempenho entre Go e Rust para uma carga dominada por I/O de arquivo e parsing de texto está bem abaixo da margem que separa ambos de Node. O gargalo aqui não é a linguagem.

Ambos entregam o requisito decisivo — binário único, sem runtime, com controle explícito de ciclo de vida.

### AD-02 — Processo externo, não plugin

Um plugin do Obsidian teria acesso gratuito ao *metadata cache* já construído pelo aplicativo. Mas exigiria o Obsidian aberto, competiria pelo *event loop* do Electron, e herdaria o modelo de processo que é a origem do problema de órfãos.

Processo externo custa reimplementar o parsing e ganha independência total.

### AD-03 — goldmark como base do parser

`goldmark` é conforme ao CommonMark, rápido, e — o critério decisivo — tem uma API de extensão que permite registrar *inline parsers* customizados. Wikilinks, embeds e block-ids são implementados como extensões que participam do parse real, o que resolve RF-17 de graça: dentro de um bloco de código, o parser não está em contexto inline, e o wikilink simplesmente não é reconhecido.

A alternativa — regex sobre o texto bruto — exigiria reimplementar o rastreamento de contexto de código, que é precisamente o trabalho que um parser faz.

### AD-04 — Índice invertido próprio

Ver §6.2. Escopo pequeno e bem definido, requisitos de domínio específicos (normalização para português, ausência de stopwords, pesos por campo), e um motor completo traria superfície de dependência desproporcional.

Revisitável se RF-26 (busca semântica) entrar em escopo.

### AD-05 — Offsets de byte, não conteúdo, no índice

Ver §4.2. Sustenta o orçamento de memória e torna a leitura de seção proporcional ao tamanho da seção, não ao da nota.

Custo: os offsets invalidam quando o arquivo muda, exigindo a verificação de hash de §5.4. É um custo pequeno e localizado.

### AD-06 — Três mecanismos de encerramento

Ver §7. Redundância deliberada. Cada mecanismo falha em cenários diferentes e nenhum custa mais do que algumas dezenas de linhas.

### AD-07 — Nenhum socket que saia da máquina

RNF-30. Até 2026-08-05 a regra era "nenhuma chamada de socket no código do produto"; reaberta com autorização explícita do dono do projeto para o IPC local do daemon (§7.5) e reformulada para **nenhum socket que saia da máquina** — um socket de domínio Unix não atravessa rede, então a garantia contra exfiltração se mantém. Nenhum pacote sob `internal/` ou `cmd/` importa `net/http` ou qualquer outro `net/*`; dentro do pacote `net`, só `net.Dial`/`net.Listen` com a constante literal `"unix"` são aceitos, verificado por análise estática (`tools/netcheck`) no CI. O SDK carrega `net/http` transitivamente para um transporte que a v1 não constrói; a formulação exata da garantia, as três partes da regra e o que o CI checa estão em PRD §6.4. Em um cofre que pode conter material confidencial, essa garantia tem valor de produto, não apenas técnico.

### AD-08 — Esquema de URI próprio para resources

Resources são publicados como `gobsidian:///<caminho-canônico>`, não `obsidian://`.

`obsidian://` é o esquema real do aplicativo Obsidian, registrado no sistema operacional, com semântica própria — `obsidian://open?vault=X&file=Y`. Reusá-lo para identificar resources MCP cria colisão com URIs que o sistema já sabe abrir, e um `obsidian://Civil/PONTO 03.md` não é uma URI válida para o aplicativo. Um esquema próprio custa nada e não colide com ninguém.

**São três barras, e o caminho vai escapado.** Este parágrafo já disse `gobsidian://` com duas, e essa forma derrubava o servidor no boot: depois de `//` vem a **autoridade**, então o primeiro segmento do caminho virava nome de host. Espaço é ilegal em host, e pasta com espaço é o caso comum num cofre do Obsidian — o exemplo `Civil/PONTO 03.md` acima é exatamente um deles. A terceira barra declara autoridade vazia; bytes fora de `A-Za-z0-9-._~` viram `%XX`, e a `/` fica crua por separar segmentos.

### AD-09 — Mutex por caminho, não fila global de escrita

Ver §4.5. Uma fila serial de escritas seria mais simples de raciocinar, e transformaria toda reorganização em lote em uma operação sequencial. `note_move` de cinquenta notas escreve dezenas de arquivos distintos que não têm relação entre si; serializá-los seria pagar latência por uma garantia que ninguém pediu.

O mapa de mutexes por caminho dá exatamente a garantia necessária — nenhuma nota é escrita por duas operações ao mesmo tempo — sem custo em operações independentes.

---

## 10. Dependências

Lista deliberadamente curta. Cada dependência é uma superfície de falha e de auditoria.

| Módulo | Uso | Justificativa |
|---|---|---|
| `github.com/modelcontextprotocol/go-sdk` | Protocolo MCP | SDK oficial, mantido com o Google; fixado em `v1.5.0`, que suporta o protocolo `2025-11-25` |
| `github.com/yuin/goldmark` | Parse de Markdown | Conformidade CommonMark; API de extensão |
| `github.com/fsnotify/fsnotify` | Watch do sistema de arquivos | Padrão de fato; abstrai as APIs de cada SO |
| `gopkg.in/yaml.v3` | Frontmatter | — |
| `github.com/cespare/xxhash/v2` | Hash de conteúdo | Rápido; não criptográfico é suficiente |
| `golang.org/x/text` | Normalização Unicode | Remoção de diacríticos para o analisador |
| `golang.org/x/sys` | Chamadas de sistema no Windows | Atributos de arquivo, identidade de processo |
| `github.com/spf13/cobra` | CLI | Subcomandos e ajuda |

Log via `log/slog` da biblioteca padrão. Testes via `testing` da biblioteca padrão.

**Diff unificado sem dependência.** RF-37 exige diff unificado no `dry_run`, e a biblioteca padrão não tem um. A implementação é própria: Myers sobre linhas, em torno de 150 linhas em `writer/diff.go`. O diff aqui é para leitura humana e de modelo, não para aplicar patch — não precisa ser mínimo, precisa ser legível. Trazer uma dependência para isso seria desproporcional.

**`net/http` no grafo.** O SDK importa `net/http` para o transporte HTTP/SSE, e a importação entra no binário mesmo usando apenas stdio. Ver PRD §6.4 para a forma verificável de RNF-30 e o que o CI de fato checa.

---

## 11. Estratégia de testes

**Parser — golden files.** Cada caso de borda documentado vira um par entrada/saída esperada em `testdata/`. O corpus cobre: wikilinks em todas as formas, links dentro de blocos de código, links escapados, colchetes literais, block-ids, headings com caracteres especiais, frontmatter malformado, arquivos vazios, arquivos sem `\n` final, CRLF misturado com LF.

**Índice — testes de propriedade.** Invariante central: para toda nota N e todo link L em N que resolve para M, existe um backlink de N em M. Verificada após sequências aleatórias de criação, modificação, movimentação e exclusão.

**Escrita — crash injetado.** Escritas executadas com abortos em pontos aleatórios do processo (após abrir o temporário, após escrever, antes do sync, entre sync e rename). Invariante: o arquivo alvo está sempre em um estado válido, ou o anterior ou o novo, nunca intermediário. Mil iterações no CI.

**Ciclo de vida — teste de órfãos.** Cem ciclos de iniciar o servidor, executar chamadas, e matar o processo pai com `taskkill /F`. Invariante: nenhum `gobsidian.exe` sobrevive. Critério de bloqueio de release.

**Performance — benchmark no CI.** Cofre sintético de 5.000 notas gerado deterministicamente. Cada alvo de RNF-01 a RNF-08 verificado. Regressão acima de 20% falha o build.

**Paridade com o Obsidian.** Corpus de 500 notas comparado contra o `metadata cache` do próprio Obsidian. Comparação de tags, links, headings e blocos. Divergência zero é o critério.

A referência não é obtenível por leitura de arquivo: o cache vive no IndexedDB do Electron. Ela vem de `tools/parity-dumper/`, um plugin de desenvolvimento descartável que serializa `app.metadataCache` para JSON, rodado uma vez sobre o corpus e versionado em `testdata/parity/`. O plugin não é parte do produto, não é distribuído, e é regenerado à mão quando o Obsidian mudar de comportamento.

A comparação é assimétrica de propósito. Nossa saída precisa conter tudo o que o Obsidian encontrou; o inverso não é exigido, porque RF-63 (âncoras quebradas) e a distinção entre formas de resolução são informação que o Obsidian não expõe. Divergência para menos é falha; para mais, é o produto.
