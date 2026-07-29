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

**`internal/service` é o único ponto que enxerga todos os subsistemas.** As camadas abaixo dele não se conhecem. O parser não sabe que existe um índice; o escritor não sabe que existe uma busca. Isso mantém cada uma testável isoladamente e impede que a lógica de orquestração se espalhe.

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
- **Índice Invertido (`inverted.go`)**: Estrutura thread-safe baseada em `sync.RWMutex` mapeando `termo -> (caminho -> []TokenPosition)`. Atualização incremental síncrona aos eventos do watcher.
- **Ranking BM25 (`bm25.go`)**: $k_1 = 1.2$, $b = 0.75$. Pesos de campo: Título 3.0x, Headings 2.0x, Corpo 1.0x. Pontuação em forma dupla: $Score = Score_{raw} \times 1.5 + Score_{reduced} \times 1.0$. Desempate determinístico por caminho de nota.
- **Recorte de Snippets (`snippet.go`)**: Extração do trecho com maior densidade de termos casados, com compensação de offset de BOM (+3 bytes no disco) e truncamento UTF-8 seguro em limite configurável (`snippet_chars`).

### 4.7 Persistência de Cache em Disco (`internal/search/persist.go`)

O cache do índice fica armazenado fora do cofre em `%LOCALAPPDATA%\gobsidian\<hash-do-caminho-do-cofre>\inverted_cache.gob` (decisão D1):

- **Cabeçalho de Integridade (`CacheHeader`)**: Guarda `FormatVersion`, `ParserVersion`, `AnalyzerVersion`, `VaultPath` e `NoteCount`. Qualquer divergência de versão rejeita o arquivo e força reindexação limpa (sem migração).
- **Escrita Atômica**: O arquivo é codificado em GOB primeiro em um arquivo temporário no mesmo diretório com o prefixo `.gobsidian-tmp-cache-*.gob` (ignorado pelo filtro de watcher) e renomeado atomicamente para `inverted_cache.gob` via `os.Rename`.
- **Diferenciação de Estado**: Cofres vazios (0 notas) geram cache válido com `NoteCount = 0`, distinguível de cache ausente (`ErrCacheNotFound`).
- **Medição RNF-02**: Boot com cache válido atinge ~15,8 ms de deserialização em 100 notas, satisfazendo a meta de boot ≤ 300 ms (PRD Q3 fechado).

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

**Correlação de Renames.** `fsnotify` reporta rename como um par de eventos (remoção na origem, criação no destino) sem correlação explícita. O debouncer entrega um lote de caminhos ao `Apply`. O `Apply` separa esses caminhos entre deletados (não existem mais no disco) e modificados/criados. Arquivos deletados e criados no *mesmo lote* que possuam exato o mesmo hash `xxhash` (calculado sobre o conteúdo bruto antes da remoção de BOM) e não sejam vazios são correlacionados como um rename em nível lógico, preservando o hash e marcando backlinks pendentes. Limitações documentadas:
1. Cópias seguidas de remoções do original caem como rename se acontecerem na mesma janela de debounce.
2. Remoção e criação separadas por tempo maior que o debounce não correlacionam.
3. Apenas notas são hasheadas e correlacionadas, anexos e mídias não. Anexo e arquivo somente-nuvem **nunca são abertos** para correlacionar: ler um placeholder dispara download síncrono, e ler um anexo viola a regra de que anexo é indexado por nome. Os dois são classificados antes de qualquer leitura e caem direto no caminho comum.
4. Qualquer modificação de conteúdo simultânea ao move anula o rename lógico e ele decai para um remover e adicionar comum (remove+replace).
5. Notas vazias nunca correlacionam. O gate `Size > 0` existe porque arquivos vazios compartilham hash e a cardinalidade 1-para-1 sozinha não distingue rename de coincidência. Um rename de nota vazia decai para remove + create.
6. macOS e BSD não sinalizam overflow: o backend kqueue do `fsnotify` v1.10.1 nunca emite `ErrEventOverflow` — só `backend_inotify.go` e `backend_windows.go` emitem. Nessas plataformas a reconciliação descrita acima nunca dispara, e o único anteparo contra evento perdido é a reindexação no boot. Lacuna registrada, não resolvida por heurística.

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
| `format_version` | O layout serializado muda |
| `parser_version` | O parser passa a produzir estrutura diferente para a mesma entrada |
| `analyzer_version` | O analisador de busca muda tokenização, normalização ou redução |

Sem `parser_version`, corrigir um bug de parsing e reiniciar carregaria de volta o índice errado — mtime e tamanho dos arquivos não mudaram, então nada indicaria invalidação. O cache continuaria servindo o resultado do parser antigo, e o bug pareceria não corrigido. É a classe de falha mais confusa que um cache pode produzir, e custa três inteiros evitá-la.

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

### AD-07 — Sem rede

RNF-30. Nenhum pacote sob `internal/` ou `cmd/` importa rede, e nenhuma chamada de socket existe no código do produto. Verificável por análise estática no CI. O SDK carrega `net/http` transitivamente para um transporte que a v1 não constrói; a formulação exata da garantia, e o que o CI checa, estão em PRD §6.4. Em um cofre que pode conter material confidencial, essa garantia tem valor de produto, não apenas técnico.

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
