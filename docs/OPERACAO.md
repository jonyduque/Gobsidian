# Guia de Operação e Diagnóstico

Este documento detalha as etapas de registro, monitoramento e diagnóstico do `gobsidian` v0.1 no ambiente real.

## 1. Registro no Claude Desktop

O `gobsidian` pode ser configurado manualmente no Claude Desktop ou usando o script PowerShell incluído.
Para detalhes específicos sobre Windows (OneDrive, caminhos com espaços, etc.), **consulte sempre `docs/WINDOWS.md` §8.**

### Edição direta do `claude_desktop_config.json`

O arquivo fica localizado em:
- **Windows:** `%APPDATA%\Claude\claude_desktop_config.json`
- **macOS:** `~/Library/Application Support/Claude/claude_desktop_config.json`

Exemplo de configuração para o servidor MCP:

```json
{
  "mcpServers": {
    "gobsidian": {
      "command": "C:\\Users\\jonyd\\go\\bin\\gobsidian.exe",
      "args": [
        "serve",
        "--vault",
        "C:\\Caminho\\Absoluto\\Para\\O\\Cofre"
      ]
    }
  }
}
```

**Dicas para evitar erros comuns:**
- Sempre use o **caminho absoluto** para o executável em `"command"`.
- Assegure-se de escapar corretamente as barras (`\\`) no Windows.
- O caminho do cofre com espaços **não deve** ter aspas adicionais incluídas na string, pois o JSON as tratará literalmente.
- Nunca termine o caminho do `--vault` com uma barra (`\`), pois ela escapará a aspa do JSON.

## 2. Diagnóstico quando o servidor não carrega

Se o servidor não for listado no Claude Desktop após a reinicialização:

1. **Rode o `doctor` primeiro.** No terminal, chame o comando para verificar as permissões e detecções:
   ```powershell
   gobsidian doctor --vault "C:\Seu\Cofre"
   ```
2. **Execute um teste MCP manual.** Tente inicializar a ferramenta manualmente em seu console para certificar que ela retorna um JSON válido em `stdout`, sem avisos (isso prova que `stdout` só envia JSON-RPC):
   ```powershell
   $InitRequest = '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"manual","version":"1.0"}}}'
   & { $InitRequest; Start-Sleep 2 } | gobsidian serve --vault "C:\Seu\Cofre"
   ```
3. Verifique se alguma saída de erro não-JSON poluiu o `stdout`. Se sim, esta é a causa da falha na conexão.

## 3. Explicação dos campos de `vault_stats`

A tool `vault_stats` retorna estatísticas internas do índice em memória, importantes para debug de carga.

| Campo JSON | Significado |
|---|---|
| `notes` | Notas Markdown indexadas |
| `assets` | Anexos indexados por nome (nunca lidos) |
| `total_size` | Soma dos tamanhos de notas e anexos, em bytes |
| `orphans` | Notas sem nenhum backlink |
| `broken_links` | Links que deveriam resolver para uma nota do cofre e não resolvem. URL externa **não** conta |
| `broken_anchors` | Links que resolvem a nota mas não ao heading ou bloco citado |
| `alias_collisions` | Aliases declarados por mais de uma nota |
| `generation` | Contador de mutações do índice desde o boot |
| `runtime` | Presente só com `include_runtime`: `num_goroutine`, `alloc`, `total_alloc`, `sys`, `num_gc` |
| `watcher` | Presente só com `include_runtime`, e **ausente se o watcher estiver desligado**. Campos na tabela abaixo |

### Campos de `watcher`

São a instrumentação principal para diagnosticar cofre em pasta sincronizada.

| Campo JSON | Significado |
|---|---|
| `active` | `false` significa que o subsistema existe e **caiu** — o `Run` retornou. Objeto ausente significa watcher desligado; os dois casos são distinguíveis de propósito |
| `events_received` | Eventos brutos do `fsnotify`, **antes** do filtro de relevância |
| `events_dropped` | Soma de `events_dropped_by_reason` |
| `events_dropped_by_reason` | Desdobrado porque as causas pedem ações diferentes: `chmod` alto é OneDrive em operação normal; `outside_vault` alto indica que a raiz é um link e o confinamento está recusando eventos; `excluded` alto indica atividade em `.obsidian/` ou `.git/`; `unknown_op` alto merece `--log-level debug` |
| `events_coalesced` | Eventos absorvidos pela janela de debounce — é o trabalho evitado |
| `events_processed` | Mudanças que chegaram ao índice |
| `events_skipped` | Descartadas por mtime e tamanho iguais aos indexados |
| `reconciliations` | Overflows tratados por varredura completa. Uma vez por overflow, não por arquivo |
| `reconciled_updated` / `reconciled_removed` | Quantos arquivos cada reconciliação corrigiu. Um overflow que corrige 3 e um que corrige 3.000 são situações diferentes |

Razão alta entre `events_received` e `events_processed` é o comportamento esperado e saudável — é para isso que existem as três camadas de filtragem.

`orphans`, `broken_links` e `broken_anchors` só vêm com `include_health`.

Uma versão anterior desta tabela listava `TotalNotes`, `TotalBytes`, `TotalLinks`, `TotalAliases`, `TotalTags`, `LoadTimeMs` e `CloudOnlyFiles`. **Nenhum desses campos existe.** Documentar campo que o servidor não emite é pior que não documentar: quem lê acredita, escreve o consumidor em cima, e descobre no primeiro uso. A tabela acima foi conferida campo a campo contra `service.StatsResult`.

A duração da indexação a frio não está em `vault_stats` — está no log de boot, como `index_ms`. Ver §4.

Os valores refletem o estado **corrente**, não o do boot: o watcher está ativo desde o M2 e mantém o índice em dia incrementalmente. `generation` é o contador que sobe a cada mutação — se ele não sobe depois de você editar uma nota, o watcher não está enxergando o cofre, e os campos de `watch` abaixo dizem por quê.

A nota anterior aqui dizia que *"a v0.1 não tem um watcher de arquivos ativo"*. Ficou para trás do M2 inteiro; corrigida em 2026-07-29.

## 4. Como ler os logs de debug

Todos os logs no formato estruturado saem obrigatoriamente para `stderr`. Para ativá-los ou gravá-los:

```powershell
gobsidian serve --vault "C:\Seu\Cofre" --log-level debug 2> "gobsidian.log"
```

As mensagens vêm em `chave=valor`. A primeira que importa é a do boot, emitida assim que o índice fica pronto:

```
time=2026-08-04T12:41:29.427-03:00 level=INFO msg="servidor pronto" vault="C:\Cofre" read_only=false notes=3149 assets=0 index_ms=905
```

`index_ms` é a duração da construção do índice de **metadados** — é o número que RNF-01 nomeia. Ele recorta só essa etapa: não inclui o boot do runtime do Go, a leitura da configuração nem o handshake do MCP.

**`search_ready` saiu desta linha em 2026-08-03.** Desde que o cache passou a ser carregado em segundo plano, a busca nunca está pronta neste ponto — o campo só poderia valer `false`, e um campo que só pode ter um valor não informa nada; informa errado, porque parece medir algo. Quando a busca fica pronta, uma segunda linha diz quando e por qual caminho:

```
level=INFO msg="indice de busca pronto" origem=cache notas=3149 duracao_ms=842
```

`origem=cache` é o caminho normal: o cache cobria o cofre inteiro. `origem=construcao` significa que o cofre foi tokenizado, e aí duas outras linhas aparecem antes:

```
level=INFO msg="construindo indice de busca em segundo plano" notas=3149
level=INFO msg="indice de busca pronto" origem=construcao notas=3149 reaproveitadas_do_cache=399 duracao_ms=206267
```

Enquanto isso, `vault_search` responde `INDEX_BUILDING`; as outras onze tools funcionam normalmente.

**Até 2026-08-03 este log enganava.** `index_ms` aparecia ao lado de "servidor pronto" e parecia ser o tempo de boot, mas o servidor só anunciava as tools depois de tokenizar o cofre inteiro — 219 s a mais num cofre de 109 MB. Quem lesse `index_ms=1275` concluiria que o boot levou 1,3 s.

A do encerramento diz qual mecanismo disparou:

```
time=2026-07-28T16:11:37.341-03:00 level=INFO msg="encerramento solicitado" reason=stdin-eof
```

Os valores possíveis são `stdin-eof`, `signal` e `parent-gone` no servidor stdio, e `idle` no daemon — que não tem stdin de host nem pai vigiável, e por isso encerra por ociosidade.

No boot, se escritas anteriores foram interrompidas por queda do processo, aparece:

```
level=WARN msg="temporarios de escritas interrompidas removidos" count=3
```

Toda escrita no cofre é atômica — temporário `.gobsidian-tmp-*` no mesmo diretório, `Sync`, `rename` —, e uma escrita que falha por motivo normal remove o próprio temporário. O único caso que sobra é o processo morto, que não roda `defer`: o temporário fica no disco e é varrido no boot seguinte. O filtro de ruído do `vault` o esconde do índice, então ele não conta como nota; se você o vir no Explorer antes de reiniciar, é isso, e não corrupção. Contagem alta e recorrente aqui significa que o processo está morrendo no meio de escritas, e vale olhar `reason=` do encerramento anterior.

`reason=` vale `stdin-eof`, `signal` ou `parent-gone`. Um encerramento sem `reason=` nenhum significa que nenhum mecanismo disparou e o processo morreu por outro motivo — é o que o gate de órfãos verifica.

**Sinais importantes:**
- `level=ERROR`: Falha fatal num componente. Pode indicar que um arquivo está corrompido ou inacessível.
- `level=WARN`: Falhas leves (por exemplo, bloqueios de compartilhamento ou permissão de leitura rejeitada).
- Overflow do buffer do sistema operacional aparece como `msg="Overflow de fsnotify detectado, reconciliação agendada"` em `WARN`, e o contador `reconciliations` sobe. Recorrência indica ampliar `--debounce-ms`. **Em macOS e BSD isso nunca aparece**: o backend kqueue do `fsnotify` v1.10.1 não emite `ErrEventOverflow`, e lá o único anteparo contra evento perdido é a reindexação no boot (`ARCHITECTURE.md` §5.3).

## 5. Medições do Orçamento de Performance

Medido com `scripts/measure.ps1`, que lê `index_ms` do próprio log de boot e amostra o `WorkingSet64` do processo depois do handshake MCP e de um período de acomodação. O script reporta o **maior** RSS observado, não o último: um pico mascarado por uma amostra tardia seria ficção.

```powershell
pwsh -File scripts/measure.ps1 -Vault <caminho-do-cofre>
```

### O que foi medido até agora

**Cofre pequeno, 2026-07-28.** 7 notas, 1 anexo, 180 KB. maquina de referencia, 12 núcleos, Windows 11. Três execuções.

### Tabela completa dos RNFs — estado apos o fechamento da Parte I do M7 (2026-08-09)

Atualizada pela Task 87. As linhas RNF-04, RNF-06 e RNF-07 mudaram desde o
fechamento do M6 (2026-08-02): RNF-06 foi corrigido pela Task 86 e RNF-07
melhorou como efeito colateral das otimizacoes de busca das Tasks 78-85 — a
tabela abaixo nao tinha sido atualizada ate agora, e por isso RNF-06 ainda
aparecia como NAO ATINGIDO aqui apesar de a Task 86 ja ter corrigido.
Detalhe de cada remedicao na secao "Fechamento da Parte I do M7", ao fim deste
documento.

Os 22 RNFs do PRD, cada um com **número medido** ou **"não medido"**. Não há
terceira coluna de opinião: alvo não atingido e registrado é informação; alvo
não medido apresentado como resultado é ficção com aparência de tabela.

Onde a escala importa, o número é o de **5.000 notas** — é a escala em que o
produto tem de valer, e vários destes RNFs saem abaixo da resolução do relógio
num cofre de 7 notas.

| ID | Métrica (alvo) | Medição | Estado |
|---|---|---|---|
| **RNF-01** | Indexação a frio (≤ 3 s) | 500,11 ms no cofre sintético; **1,1 s** num cofre real de 109 MB | **Atingido** |
| **RNF-02** | Boot com cache válido (≤ 300 ms) | 208–282 ms no cofre sintético; **371–472 ms** num cofre real de 109 MB (2026-08-06, com `index_cache`); **810–1079 ms** num cofre real de 5.686 notas (2026-08-26, Task 135) | **NÃO ATINGIDO** |
| **RNF-03** | `note_read` p95 (≤ 15 ms) | p95 **344,97 µs**, mediana 206,47 µs (5.000 notas) | **Atingido** |
| **RNF-04** | `vault_search` p95 (≤ 100 ms) | 5.000 notas: **8 de 8** (2026-08-12, Task 94). `limit: 200` em **43,1 / 28,8 / 27,6 ms** em três rodadas; era 119–123 ms. Medido a frio e no índice **vindo do cache**, que é o ramo que o servidor executa — ver "Recorte de trecho" ao fim | **Atingido** |
| **RNF-05** | `note_list` com filtro de metadados p95 (≤ 10 ms) | p95 **533,68 µs**, mediana 249,24 µs (5.000 notas) | **Atingido** |
| **RNF-06** | Reindexação de arquivo único (≤ 20 ms) | mediana **334,87 µs**, p95 544,87 µs (5.000 notas, lote=20; Task 86, 2026-08-06). Era 20,35 ms | **Atingido** |
| **RNF-07** | Heap vivo em repouso (≤ 8 MB + 32 KB × notas) | **Redefinido em 2026-08-28** — era `RSS ≤ 60 MB`. Cinco cofres reais, `scripts/measure.ps1`, estado `servindo`: **7 / 17 / 40 / 58 / 126 MB** contra tetos de 10,4 / 47,8 / 47,2 / 87,9 / 185,7 MB | **Atingido**, folga de 15% a 64% |
| **RNF-08** | CPU em repouso (< 0,5 %) | **não medido** | — |
| **RNF-09** | Escalabilidade linear até 20.000 notas | **não medido** (medido até 5.000) | — |
| **RNF-10** | Zero órfãos em 100 ciclos de start/kill do host | **100/100 em quatro cenários** — `stdin-eof`, `parent-death`, `signal` e `daemon-idle` —, cada um com o `reason=` do seu mecanismo, 400 ciclos no total | **Atingido** |
| **RNF-11** | Zero notas corrompidas em 1.000 crashes injetados | **0 / 1.000**, com 381 temporários órfãos varridos | **Atingido** |
| **RNF-12** | Índice degradado nunca devolve resultado incorreto | **não medido**; verificado por teste (reconciliação por overflow, `internal/watcher`) | — |
| **RNF-13** | Falha de tool não derruba o servidor | **não medido**; verificado por teste (`internal/mcpsrv`, erros como resultado MCP) | — |
| **RNF-20** | Windows 10+ primeira classe; macOS 13+ e Linux suportados | **não medido**; CI roda build, vet e `go test -race` nos três | — |
| **RNF-21** | Cofres em OneDrive/Dropbox/Drive, incluindo somente-nuvem | **não medido**; verificado por teste (`vault.CloudOnly`, `internal/vault`) | — |
| **RNF-22** | Caminhos acima de 260 caracteres no Windows | **não medido**; verificado por teste (`longpath_windows_test.go`) | — |
| **RNF-23** | Nomes com acentuação e espaços | **não medido**; verificado por teste (corpus dos golden files) | — |
| **RNF-24** | Protocolo MCP fixado em `2025-11-25` com fallback | **não medido**; fixado no SDK e verificado por teste | — |
| **RNF-30** | Nenhum socket que saia da máquina (reformulado em 2026-08-05, Task 90 — texto anterior: "nenhuma requisição de rede") | **não medido**; verificado por gate — `check_net.ps1` com o analisador `netcheck` em `go vet -vettool`, nos três GOOS, mais checagem textual de `net/*` | **Atingido** |
| **RNF-31** | Todo caminho de tool confinado ao cofre | **não medido**; verificado por teste (`validateLocal` + `Canonicalize`) | — |
| **RNF-32** | Links simbólicos para fora do cofre não são seguidos | **não medido**; verificado por teste (`TestWalkNaoSegueSymlink`, `TestWalkPulaSymlinkDeArquivo`, `TestLeituraRecusaSymlinkPorPadrao`, executados com privilégio) | **Atingido desde 2026-08-26.** Antes disso valia só para symlink de DIRETÓRIO: o único teste que existia era o de diretório, e um symlink de ARQUIVO chamado `nota.md` passava nas duas camadas léxicas de confinamento, entrava no índice e era lido. O requisito estava publicado como atingido enquanto metade dele era violada. `--follow-symlinks` religa o comportamento antigo, por decisão do dono |
| **RNF-33** | `--read-only` desabilita toda a superfície de escrita | **não medido**; verificado por teste (tools de escrita ausentes de `ListTools`) | — |

### O que RNF-01 e RNF-02 medem, e o que eles NÃO mediam

**Os dois mediam sub-etapas do boot e foram apresentados como se medissem o
boot.** A correção é de 2026-08-03 e veio de um relato de uso, não de um teste:
o Claude Code recusava a conexão com `connection timed out after 30000ms`.

`index_ms`, que é o número do RNF-01, cobre **só o índice de metadados**. O boot
completo também tokenizava o cofre inteiro para o índice invertido, e num cofre
real de 109 MB isso levava **219 s** — o servidor só anunciava as tools depois.
O log dizia "servidor pronto" carimbando `index_ms=1275` ao lado, o que fazia o
número parecer o tempo de boot.

Pior: o host desiste em 30 s e mata o processo **antes** de o cache ser gravado,
então a tentativa seguinte recomeçava do zero. Toda tentativa falhava pelo mesmo
motivo, indefinidamente.

O RNF-02 tinha o mesmo vício em escala menor: os 96,94 ms medem
`LoadInvertedCache` isolado, e o boot com cache quente no cofre real levava
**8,6 s** contra um teto de 300 ms.

> [!IMPORTANT]
> **As tabelas desta seção e das seguintes são por ETAPA, em ordem cronológica.**
> Foram quatro mudanças no mesmo dia, cada uma medida contra a anterior, e o
> "depois" de uma é o "antes" da próxima. Nenhum número intermediário descreve o
> estado atual.
>
> **Estado corrente, medido em cinco partidas:** servidor anuncia as tools em
> **832–1183 ms**, busca utilizável em **603–821 ms** depois disso, RSS em
> repouso de **381 MB**. Está na tabela do fim de *Estrutura em memória: base
> achatada e delta*.

**Medições no cofre real de 109 MB / 3.153 notas, 2026-08-03:**

| Momento | Antes | Depois |
|---|---|---|
| Servidor anuncia as tools (cache frio) | **220 s** | **2,18 s** |
| Busca fica utilizável (cache frio) | 220 s | 206 s, em segundo plano |
| Servidor anuncia as tools (cache quente) | 8,6 s | ~7 s |

A construção do índice invertido passou a rodar em segundo plano
(`cmd/gobsidian/serve.go`). Enquanto ela corre, `vault_search` devolve
`INDEX_BUILDING` em vez de uma lista curta — "ainda não sei" e "não achei nada"
pedem ações diferentes de quem chama. As outras onze tools funcionam desde o
primeiro segundo, porque dependem só do índice de metadados.

**RNF-02 seguia não atingido a ~7 s contra 300 ms**, num cofre real de 109 MB
cujo cache tinha 472 MB para ler do disco. Ver a seção seguinte, que registra
onde ele está depois do trabalho de 2026-08-03 sobre o formato do cache.

### Formato do cache de busca e boot com cache quente (2026-08-03)

O formato do cache era `gob` sobre `map[string]map[string][]TokenPosition`, e o
carregamento dele era 86% do boot com cache quente. Ele foi substituído por um
codec binário próprio (`internal/search/persist_codec.go`), e o carregamento
saiu do caminho de boot.

Todos os números abaixo são medidos, no mesmo cofre real e na mesma máquina.
O antes vem de decodificar o arquivo `gob` de 505.643.791 bytes que estava em
disco; o depois, de `BenchmarkLoadInvertedCacheReal` sobre o arquivo novo gerado
do mesmo cofre.

| Medida | Formato `gob` | Formato binário | Fator |
|---|---|---|---|
| Tamanho do arquivo | 505.643.791 B (482,2 MB) | 70.084.435 B (66,8 MB) | 7,2× menor |
| Carregamento | 5,59 s | **1,59 s** ± 9% (n=6) | 3,5× |
| Bytes alocados | 3,69 GB | **737,2 MiB** (n=6) | 5,0× |
| Alocações | 13.035.004 | **643.413** (n=6) | 20,3× |

Cofre de referência: 3.152 notas, 109 MB, 126.342 termos, 3.020.792 postings,
18.229.295 posições.

O que cada peça do formato paga:

- **tabela de caminhos** — o caminho da nota era repetido em cada posting.
  286,3 MB dos 471,6 MB eram a mesma string escrita 2,96 milhões de vezes.
- **posições em varint sobre delta** — 16 bytes fixos por posição viraram 2 a 4.
- **codec manual** — o perfil do `gob` era 63,9% de `decodeStruct` cumulativo,
  gasto em reflexão.
- **arena contígua** — as 2,96 milhões de fatias de posição viraram subfatias de
  um bloco só, dimensionado por totais gravados no próprio arquivo.
- **decodificação sobre `[]byte`** — `bufio` + `binary.ReadUvarint` custavam uma
  chamada de interface por byte de varint: 30% do tempo restante. Medido em
  −17,77% (p=0,002, n=6).

**Boot com cache quente, medido em três partidas:**

| Momento | Antes | Depois |
|---|---|---|
| Servidor anuncia as tools | ~7 s | **1,02–1,13 s** |
| Busca fica utilizável | ~7 s | 1,58–1,66 s |

O carregamento do cache saiu do caminho de boot pelo mesmo raciocínio que já
havia tirado a construção. Enquanto ele corre, `vault_search` devolve
`INDEX_BUILDING`.

**RNF-02 segue não atingido: ~1,05 s contra 300 ms.** Está 6,7× mais perto do
que estava, e o que sobra é dominado pela construção de mapas Go — 142 mil mapas
internos e 3 milhões de entradas —, não mais por serialização. `search_ready`
saiu do log de "servidor pronto": com o cache carregado em segundo plano ele só
poderia valer `false`, e um campo que só pode ter um valor não informa nada.

### Estrutura em memória: base achatada e delta (2026-08-03)

Resolvido o formato do arquivo, o que sobrava do carregamento era montar
`map[string]map[string][]TokenPosition`: **126.342 mapas internos e 3.020.792
entradas**, medidos em 35% do tempo em `aeshashbody`, `mapassign_faststr` e
`matchH2`.

Os mapas existiam porque o índice muda em tempo de execução — o watcher chama
`Add`, `Remove` e `Update` a cada arquivo mudado. A saída foi separar as duas
naturezas em vez de fazer uma estrutura servir às duas:

| Camada | O que é | Muda? |
|---|---|---|
| `base` | arrays achatados vindos do cache (`internal/search/soa.go`) | não |
| `delta` | mapas, com o que mudou desde a partida | sim |

Toda leitura consulta as duas e o delta ganha. Toda escrita vai só para o
delta e marca o caminho como substituído no base. Construção do zero — sem
cache — deixa `base` nil e usa exatamente o caminho de antes.

O formato do arquivo passou a gravar tudo em ordem crescente, e é isso que
permite busca binária no lugar do mapa. O leitor **confere** as três ordens: um
arquivo fora de ordem não falharia sozinho, responderia "termo não existe" para
termos que existem, e a busca deixaria de achar notas sem erro nenhum no log.

**Carregamento do cache, mesmo cofre e mesma máquina, `benchstat` n=6:**

| Medida | Mapas (formato 4) | Base achatada (formato 5) | Delta |
|---|---|---|---|
| Tempo | 1,592 s ± 9% | **659,2 ms ± 23%** | **−58,58%** (p=0,002) |
| Bytes alocados | 737,2 MiB | **389,8 MiB** | −47,12% (p=0,002) |
| Alocações | 643,4 k | **291,1 k** | −54,76% (p=0,002) |

**Boot com cache quente, cinco partidas:** busca utilizável em 603, 656, 724,
821 e 708 ms — mediana 708 ms, contra 1217 ms de mediana medida em 12 partidas
com a versão em mapas.

**RSS em repouso, três partidas:** 382,2 / 381,5 / 380,9 MB, contra 587,0 /
587,2 / 587,4 MB da versão em mapas. **−205 MB.**

Um ganho colateral que não estava no plano: `removeLocked` varria os 126.342
termos a **cada arquivo mudado**, para apagar o caminho de cada um. A base
carrega um índice direto (documento → termos), construído em duas passadas
lineares sem ordenação, e a operação passou a custar os termos daquela nota.

**O que continua em mapa, de propósito:** a tabela caminho → id, com uma
entrada por documento (3.152), porque `DocLength` é chamado dentro do laço de
postings do BM25 e ali a busca binária sobre caminhos de prefixo longo custa
mais que um hash. O que a estrutura achatada eliminou foram os 126 mil mapas e
os 3 milhões de entradas, não uma tabela de 3 mil.

**GOGC não foi alterado, e a memória transitória passou a ser devolvida.**

`GOGC=off` sobre o carregamento mediu `~ (p=0,093, n=6)`. `GOGC=400` durante a
carga mediu **−28,51% (p=0,002, n=6)** no benchmark — mas o benchmark não é o
boot: nele a estrutura de ~500 MB da iteração anterior ainda está viva quando a
seguinte começa, então o heap vivo é o dobro do de uma partida real, que é
justamente o regime onde afrouxar o GC ajuda. No boot real, 12 partidas por
braço deram mediana 1217 → 1147 ms (−5,8%), com U de Mann-Whitney = 88 contra
região crítica de 37/107: **não significativo**. E o RSS em repouso ficou igual
ou pior — 829,5 / 794,2 / 789,1 MB contra 782,8 / 783,0 / 779,5 MB da base,
porque um alvo de heap maior é exatamente isso. GOGC ficou como estava.

O que pagou foi `debug.FreeOSMemory()` depois de o índice ficar pronto. A
montagem aloca 737 MB para deixar ~500 MB vivos; a diferença — a fatia com o
arquivo inteiro (70 MB) e a tabela de faixas da arena (~120 MB) — vira lixo
assim que o índice fica montado, mas o Go devolve essas páginas no tempo dele,
e até lá elas aparecem no RSS de um servidor que ficará horas em repouso.

**RSS em repouso, 22 s depois da partida, três partidas por braço:**

| Braço | Medidas | Mediana |
|---|---|---|
| Base | 782,8 / 783,0 / 779,5 MB | 782,8 MB |
| `GOGC=400` só | 829,5 / 794,2 / 789,1 MB | 794,2 MB |
| `FreeOSMemory` só | 627,4 / 587,1 / 588,6 MB | **588,6 MB** |
| Ambos | 587,0 / 587,2 / 587,4 MB | 587,2 MB |

Cerca de **−195 MB**, com as distribuições da base e do adotado sem
sobreposição. Custo medido da chamada: 67, 70 e 73 ms, pagos depois de o
servidor já estar respondendo e de a busca já estar utilizável.

Estes números são do cofre real de 109 MB, cujo índice vivo passa de 500 MB —
não são o cenário de RNF-07, que foi medido num cofre sintético de 5.000 notas.
**RNF-07 não foi remedido**, então segue registrado nos 67,08 MB de antes.

#### Dois defeitos que este trabalho expôs

Nenhum dos dois era hipotético; os dois estavam em produção e nenhum teste os
via.

**`DocLength` divergia entre índice construído e índice recarregado.** Ele era
derivado na leitura, somando o número de posições de cada termo — e um token
cuja forma reduzida difere da raiz entra em duas postings. Medido: um documento
de 5 tokens que todos reduzem dava `DocLength` 5 recém-construído e **10**
recarregado do cache. `DocLength` é o divisor da normalização por tamanho do
BM25, então o mesmo cofre ranqueava diferente conforme o servidor tivesse
acabado de indexar ou de ler o cache. `docLengths` passou a ser gravado no
arquivo.

**Nota sem token nenhum nunca contava como coberta.** Ela não entrava em
`docLengths`, logo não entrava em `DocCount`, logo o cabeçalho do cache
declarava menos notas do que o índice de metadados enxergava — e o boot
concluía "cache parcial" em **toda** partida. No cofre de referência, 4 notas
vazias em 3.152 custavam uma reconstrução e a regravação do cache inteiro a cada
partida, medidas em 3,3 a 3,9 s por boot.

**Quatro RNFs não estão atingidos**, e nenhum deles é ambíguo:

- **RNF-04**, só para `limit: 200` a 5.000 notas: 181,25 ms contra 100 ms. Caiu
  68% na Task 72 e continua 81% acima. O que resta atacar é o custo por
  resultado, não a concorrência.
- **RNF-06**: 20,35 ms de mediana contra alvo de 20 ms — margem de 2%, dentro da
  linha de degradado de 100 ms que o próprio PRD define.
- **RNF-07**: 67,08 MB contra 60 MB com cache quente, 112,96 MB a frio.

**RNF-08 e RNF-09 não foram medidos** e estão escritos como tal. Medir RNF-09
exigiria um cofre de 20.000 notas que não foi gerado; RNF-08 exige amostragem de
CPU do processo em repouso, que nenhum harness deste projeto faz hoje.

Onde a linha diz "verificado por teste", o RNF é uma garantia funcional e não um
número: não há o que medir, há o que provar, e a prova é o teste nomeado.

### Índice de metadados persistido, `index_cache.gob` (2026-08-06, Task 85)

Até aqui só o índice invertido (busca) tinha cache em disco. O índice de
metadados — o que sustenta `vault_stats`, `note_read`, `note_list` e a
resolução de link — era reconstruído por varredura e parse do cofre inteiro em
**toda** partida, mesmo com o cache de busca quente. Era o que faltava para
fechar `docs/PRD.md` Q3, que já tinha decidido persistir os dois caches desde
2026-07-29.

`internal/index/persist.go` e `persist_codec.go` implementam
`index_cache.gob`: mesma técnica do codec de busca (varint, tabela de string
por comprimento+bytes, portão de versão antes de qualquer campo de layout),
mais um CRC32 (IEEE) como rodapé — divergência deliberada, justificada no
comentário do arquivo: um cache de metadados errado serve nota errada para uma
tool, não busca lenta, então um byte corrompido no meio do payload de uma
string (não num comprimento) precisa ser pego de forma determinística, não
só quando por acaso estoura um limite.

`byAlias`, backlinks e a resolução de cada link **não são gravados**: são
recalculados no load chamando as mesmas três funções que `Build` chama depois
de indexar (`buildAliasMap`, `resolveAllLinks`, `buildBacklinks`). Persistir
esses três separadamente seria uma segunda forma de calcular o mesmo dado a
partir das notas — a lição já paga neste projeto (`byAlias` minúsculo numa via
e cru na outra) é que a forma menos usada é a que diverge.

`cmd/gobsidian/serve.go` só aceita o cache quando `Index.VerifyFreshness`
confirma, por uma varredura leve (sem parse, sem leitura de conteúdo), que
todo arquivo do disco bate em tamanho e mtime com o que o cache tem — e que a
contagem de arquivos é a mesma dos dois lados, o que pega adição pura. Qualquer
divergência cai para `idx.Build` como antes, e o cache é regravado no fim.
`Index.LoadIndexCache` também confere **cobertura**, não só versão: o
cabeçalho declara `NoteCount`/`AssetCount`, e um corpo que traga menos é
recusado — a regra que o cache de busca aprendeu na marra (`LoadInvertedCache`
conferia versão e não contagem, e um cache parcial passava por completo).

**Não foi possível confirmar o cofre real exato citado nas seções acima**
dentro desta tarefa (3.149 notas / 109 MB), então a medição abaixo é num cofre
**sintético** gerado com `scripts/gen_vault.ps1 -Notes 3149 -BodyKB 35 -Seed
42`, deliberadamente na mesma escala: 3.149 notas, 50 anexos, **107,93 MB**.
Não é o mesmo cofre; é a melhor aproximação disponível, e fica registrado como
tal em vez de apresentado como o mesmo dado.

**`index_ms`, cinco partidas sem cache (equivalente ao comportamento anterior
a esta tarefa — cache apagado antes de cada partida):**

| Partida | 1 | 2 | 3 | 4 | 5 |
|---|---|---|---|---|---|
| `index_ms` | 7736 | 886 | 877 | 901 | 852 |

A primeira partida paga leitura fria do cache de disco do SO para os 108 MB do
cofre — as quatro seguintes, com o mesmo disco já quente, ficam em 852–901 ms,
a mesma faixa do que já estava registrado para o cofre real (832–1183 ms).

**`index_ms`, cinco partidas com cache presente e válido:**

| Partida | 1 | 2 | 3 | 4 | 5 |
|---|---|---|---|---|---|
| `index_ms` | 208 | 236 | 213 | 213 | 282 |

**No cofre sintético, as cinco partidas ficaram abaixo do teto de 300 ms** — de
3 a 4 vezes mais rápido que sem cache.

**No cofre real, NÃO.** Remedido pelo revisor em 2026-08-06, seis partidas com
`index_origin=cache`:

| Partida | 1 | 2 | 3 | 4 | 5 | 6 |
|---|---|---|---|---|---|---|
| `index_ms` | 451 | 472 | 408 | 411 | 371 | 458 |

Contra a baseline de 1267, 1396 e 1192 ms medida no mesmo cofre antes da
tarefa: **queda de ~66%**. Mas a faixa é 371–472 ms contra um teto de 300 ms, e
**RNF-02 segue NÃO ATINGIDO**.

A diferença entre os dois cofres é o ponto que importa: o sintético é local, o
real está em OneDrive. `VerifyFreshness` percorre todo arquivo do cofre para
conferir mtime e tamanho, e é essa varredura — não a decodificação do cache —
que sobra. O cofre sintético diria "requisito atingido"; o real diz "2,7 vezes
mais perto, ainda fora".

Registrado assim de propósito. Alvo não atingido e medido é informação; alvo
medido no cofre errado e apresentado como atingido é o defeito que esta seção
de OPERACAO.md existe para não repetir.

**Os três cenários exigidos pela tarefa, verificados contra o binário real
(não só a suíte de testes):**

1. **Com cache:** `index_ms` cai de ~870 ms para ~230 ms — tabelas acima.
2. **Cache apagado e servidor reiniciado:** reconstrói sem erro, `notes=3149
   assets=50` batendo com a varredura, e regrava `index_cache.gob` no fim.
3. **Um byte corrompido no meio do arquivo:** o servidor recusa o cache e
   reconstrói, sem decodificar lixo. Log real da partida:
   `msg="cache de indice de metadados descartado" err="index cache file
   corrupted"`, seguido de `index_ms=868` — a reconstrução, não um índice com
   metadado corrompido servido em silêncio.

#### Remedição no cofre real (2026-08-06, mesmo dia)

O caminho do cofre real foi confirmado depois da medição sintética acima.
Por pedido de quem é dono dele, o caminho não entra em nenhum artefato
versionado — as tabelas abaixo citam só a contagem de notas e o tamanho, como
já era o padrão nesta seção. O cofre cresceu desde a medição anterior: **4.165
notas** hoje, não mais 3.149.

Medido com o binário compilado direto (não `scripts/measure.ps1`): stdin
mantido aberto por um `sleep 10 |` até o boot terminar, `index_ms` lido do
próprio log.

**`index_ms`, cinco partidas sem cache** (arquivo `index_cache.gob` apagado
antes de cada uma; `inverted_cache.gob`, do índice de busca, preservado —
apagá-lo custaria minutos de reconstrução e não mede nada desta tarefa):

| Partida | 1 | 2 | 3 | 4 | 5 |
|---|---|---|---|---|---|
| `index_ms` | 1389 | 1230 | 1237 | 1287 | 1269 |

Sem o pico de disco frio que apareceu no cofre sintético — nenhuma partida
tomou a penalidade de OneDrive frio que a primeira execução do dia às vezes
paga. Faixa consistente com a já registrada antes desta tarefa (832–1183 ms
para 3.149 notas; a faixa um pouco mais alta aqui é esperada, com mais notas).

**`index_ms`, cinco partidas com cache presente e válido:**

| Partida | 1 | 2 | 3 | 4 | 5 |
|---|---|---|---|---|---|
| `index_ms` | 547 | 397 | 494 | 443 | 438 |

**RNF-02 (≤ 300 ms) segue NÃO ATINGIDO no cofre real** — a correção do
parágrafo anterior (que dizia "atingido nesta escala" sobre o cofre
sintético) fica registrada aqui, não escondida: ~3× mais rápido do que sem
cache (1230–1389 ms → 397–547 ms), e ainda acima do teto.

**Por que o cofre real não bate com o sintético, apesar da mesma escala de
notas:** a hipótese mais provável é `Index.VerifyFreshness`
(`internal/index/persist.go`) — a varredura que confere mtime e tamanho de
cada arquivo antes de aceitar o cache é sequencial e só faz `Stat`, sem
paralelismo. Num disco local esse custo é pequeno; num cofre sincronizado
por nuvem, cada `Stat` paga uma latência que o disco local não tem, e a
mesma varredura dentro de `Build` fica menos visível porque roda em paralelo
com o parse dos arquivos, não sozinha. Não investigado a fundo dentro desta
tarefa — os testes desta seção cobrem correção (cache não pode mentir), não
a latência de `VerifyFreshness` em disco de nuvem. Fica registrado como o
próximo alvo de otimização se RNF-02 continuar sendo prioridade nesta escala.

### Medições anteriores, por escala

**Cofre pequeno, 2026-07-28.** 7 notas, 1 anexo, 180 KB. Três execuções.

| ID | Métrica (Alvo) | Medição |
|---|---|---|
| **RNF-01** | Indexação a frio (≤ 3 s) | 5–8 ms (7 notas) |
| **RNF-02** | Boot com cache válido (≤ 300 ms) | **26,96 ms** (500 notas distintas, 2026-07-29, Task 52) |
| **RNF-07** | RSS em repouso (≤ 60 MB) | 18,9–19,3 MB (7 notas) |
| **RNF-11** | Zero notas corrompidas em 1.000 crashes injetados | **0 corrompidas / 1.000**, com **381 temporários órfãos** varridos (2026-08-01). Ver a nota abaixo: até esta data o teste não escrevia. |

**Medições do M3.1 e M4 (Task 52 em 2026-07-29, Task 61 em 2026-07-30).** Em corpus sintético de 500 notas distintas (`idx.NoteCount() == 500`):

- **RNF-02 (Boot com cache):** `LoadInvertedCache` do disco levou **26,96 ms**. Reconstruir o índice invertido a partir do `index` de metadados na memória levou **106,58 ms**. A leitura do cache serializado do disco é ~4x mais rápida, economizando ~80 ms no boot frio.
- **RNF-04 (Latência de busca p95):** **remedido em 2026-07-30 pela Task 61.** A medição passa por `service.Search`, com trecho ligado, e é **por formato de consulta** — não um p95 único sobre a mistura. 30 consultas por formato, 500 notas distintas, `TestRNF04VaultSearchLatencyP95`:

  **Estes números foram medidos no ramo do DELTA** (índice construído do zero),
  que é o que `createSearchService` produzia até a Task 98. Não são comparáveis
  com medições posteriores a 2026-08-13, que saem do ramo do cache.

  | Formato | Mediana | p95 | Status RNF-04 |
  |---|---|---|---|
  | termo amplo, `limit` default | 10,3 ms | 12,9 ms | OK |
  | dois termos | 12,0 ms | 21,6 ms | OK |
  | termo seletivo | 3,4 ms | 8,3 ms | OK |
  | filtro de pasta | 9,6 ms | 13,6 ms | OK |
  | filtro de tag | 9,8 ms | 14,9 ms | OK |
  | **frase exata** | **17,3 ms** | **22,1 ms** | **OK (otimizado na Task 61)** |
  | trecho de 1000 chars | 11,9 ms | 15,8 ms | OK |
  | `limit: 200` (máximo do schema) | 62,8 ms | 71,9 ms | OK, **folga fina** |

  **RNF-04 está atingido em todos os formatos, com uma ressalva medida.** A otimização da Task 61 introduziu o método $O(1)$ `Inverted.Positions(term, path)` sobre o mapa interno `terms[term][path]`, eliminando a busca linear $O(N)$ que percorria todas as postings do termo a cada candidato de posição. O p95 da frase exata caiu de **174,2 ms para 22,1 ms** (redução de ~87%, bem abaixo do teto de 100 ms).

  **A ressalva: `limit: 200` tem ~20% de folga, e ela some sob carga.** Remedido em 2026-08-01 na mesma máquina: p95 de **81,4 ms** com a máquina ociosa. Com **quatro cópias** do binário de teste rodando ao mesmo tempo (12 núcleos), o p95 foi a **100,6 / 102,9 / 107,4 ms** em três das quatro — uma delas estourando o teto por 0,6%. Com **oito cópias**, as oito estouraram, entre 111 e 126 ms.

  O custo é a geração de trecho, que lê do disco uma vez por resultado: com `limit: 200` são 200 leituras, e a mediana sozinha já fica em 61–76 ms. Os outros sete formatos têm 3x a 20x de folga e não são afetados.

  Isso é **lacuna registrada, não alvo atingido com conforto**. Reduzir o custo — leitura de trecho concorrente ou cache — é trabalho de endurecimento (M6), não está feito. O teste `TestRNF04VaultSearchLatencyP95` repete a medição de um formato até 3 vezes antes de reprovar, para separar pico de carga de regressão de código: pico não sobrevive a três rodadas, regressão sobrevive a todas. **Repetir não cria folga; a folga medida continua sendo de ~20%.**

  **O teto do RNF-04 passou a ser cobrado pelo gate.** Ele só vale sem `-race` (o detector multiplica a latência por 2 a 6), e a única etapa que rodava testes usava `-race` — de modo que o teto não era cobrado em lugar nenhum, ficando por conta de quem rodasse `go test` na mão. `verify.ps1` ganhou a etapa `go test (RNF-04, sem -race)`.

**RNF-11 (crash injetado durante escrita), 2026-08-01.** `TestRNF11NoCorruptionUnder1000Crashes`, 1.000 iterações, 12 trabalhadores: **0 notas corrompidas**, **381 temporários órfãos** varridos com `-race` (280 sem), em ~20 s.

O número de órfãos é o que dá sentido ao zero: ele conta as iterações que morreram **depois** de criar o temporário e **antes** do rename, que é a janela exata que a escrita atômica existe para cobrir.

**Até 2026-08-01 esse número era zero, e o teste não verificava nada.** Ele matava o processo filho num ponto fixo do relógio, 0 a 39 ms contados do `cmd.Start()`. Mas o filho é o próprio binário de teste: antes de chegar em `WriteAtomic` ele paga criação de processo e init do runtime. Medido: uma escrita **não interrompida** leva mediana de **47,2 ms sem `-race` e 1,077 s com**. A janela de 39 ms cabia inteira dentro do init — as mortes caíam antes de o temporário existir, e sob `-race` (o único modo em que o gate roda) isso era 100% das 1.000 iterações. O teste relatava "0 corrompidas em 1.000 iterações" sem ter escrito um byte, e esse zero é o critério de bloqueio do M4.

Corrigido sincronizando com a escrita em vez do relógio: o filho avisa em stdout imediatamente antes de `WriteAtomic`, e o pai mata de 0 a 9,95 ms **depois do aviso**. Ajustar a constante não resolveria — o init domina o relógio e muda com máquina, modo e versão do Go.

**Prova de mutação (2026-08-01).** Trocando `os.CreateTemp` por `os.OpenFile(targetPath, O_TRUNC)` — isto é, escrita in-place em vez de temp+rename — o teste reprova com **7 de 1.000 iterações corrompidas**, todas truncadas a 0 bytes. Antes da correção essa mesma mutação passava verde.

Quem denunciou a lacuna foi a guarda `orfaos == 0`, escrita junto com o teste e correta desde então: ela se recusa a reportar cobertura que não houve.

**Medições do M5 (Tasks 63 a 67 em 2026-07-30).** `note_move` e `note_delete` validados funcionalmente com 100% de cobertura nos testes de mutação. Latências de movimentação e exclusão em lote no cofre de 5.000 notas: **não medido** (agendado para o endurecimento M6/H1).

### Medições no cofre sintético de 5.000 notas (2026-08-01)

Executado contra o cofre sintético gerado deterministicamente (`scripts/gen_vault.ps1 -Notes 5000 -Seed 42`: 5.000 notas, 50 anexos, 1.27 MB, 10.101 links, 1.518 quebrados).

Todos os números abaixo são de **2026-08-01, depois da revisão da Task 72**, com
`maxSnippetWorkers = 8`, na maquina de referencia (12 núcleos, Windows 11). A medição de
latência roda **sem `-race`**: o detector multiplica o tempo por 2 a 6 e o número
deixa de ser comparável com o teto.

| ID | Métrica (Alvo) | Mínima | Mediana (5 rodadas) | Máxima | Status RNF |
|---|---|---|---|---|---|
| **RNF-01** | Indexação a frio (≤ 3 s) | 486,53 ms | **500,11 ms** | 529,20 ms | **Atingido** (6x abaixo do teto) |
| **RNF-02** | Boot com cache válido (≤ 300 ms) | 77,29 ms | **96,94 ms** | 106,27 ms | **Atingido** (3x abaixo do teto) |
| **RNF-07** | RSS em repouso (≤ 60 MB) | 66,33 MB | **67,08 MB** (cache quente) | 112,96 MB (frio) | **NÃO ATINGIDO** |

**RNF-07 não é atingido a 5.000 notas, e a versão anterior desta tabela dizia o
contrário por medir a grandeza errada.** Ela registrava `Alloc: 29,12 MB` de
`runtime.MemStats` e marcava "OK no Heap Alloc". RNF-07 é **RSS** — o working set
do processo —, que inclui o runtime do Go, os stacks das goroutines, o binário
mapeado e os spans já devolvidos pelo alocador mas ainda residentes. `Alloc` é
uma fração disso. Medido no processo real (`gobsidian serve` contra o cofre de
5.000, `Process.WorkingSet64`, cinco amostras a 500 ms depois de 8 s de repouso):

| Regime | RSS observado | Alloc / Sys do Go |
|---|---|---|
| Cache quente | 66,33 / 67,08 / 67,44 MB | Alloc 29,17 MB, Sys 132,83 MB |
| Cache frio (índice do zero) | 107,58 / 108,90 / 112,96 MB | — |

O teto são 60 MB. O melhor caso está **12% acima**, e a subida a frio, **80%
acima**. O instrumento foi conferido contra um cofre de 100 notas, onde deu
20,97 MB — coerente com os 18,9–19,3 MB históricos do cofre de 7 notas, o que
descarta erro de escala na medição. **Fechar RNF-07 a 5.000 notas é trabalho em
aberto**, não uma linha verde.

#### RNF-04: latência de `vault_search` p95 a 5.000 notas (30 consultas por formato)

A coluna "antes" é o caminho sequencial, isto é, o estado anterior à Task 72,
medido no mesmo cofre e na mesma máquina com `maxSnippetWorkers = 1`.

| Formato | p95 antes | Mediana depois | **p95 depois** | Status (alvo ≤ 100 ms) |
|---|---|---|---|---|
| `termo amplo, limit default` | 140,48 ms | 71,66 ms | **94,54 ms** | **Atingido** |
| `dois termos` | — | 17,05 ms | **20,60 ms** | **Atingido** |
| `termo seletivo` | — | 10,87 ms | **16,40 ms** | **Atingido** |
| `filtro de pasta` | — | 75,41 ms | **92,40 ms** | **Atingido** |
| `filtro de tag` | — | 76,66 ms | **92,55 ms** | **Atingido** |
| `frase exata` | — | 53,15 ms | **64,90 ms** | **Atingido** |
| `trecho maximo` | — | 21,66 ms | **30,98 ms** | **Atingido** |
| `limit maximo do schema (200)` | 561,81 ms | 164,23 ms | **181,25 ms** | **NÃO ATINGIDO** (81% acima) |

**Sete formatos de oito passaram a caber no teto; `limit: 200` não.** Ele caiu de
561,81 ms para 181,25 ms — uma redução de 68% —, e continua a 81% acima dos
100 ms. Registrar 181 ms é a resposta certa; chamar de vitória de 68% e parar
seria descrever o delta e omitir o requisito.

#### RNF-04 a 500 notas, e o que a carga de quatro cópias passou a medir

No corpus de 500 notas do gate, ociosa, os oito formatos ficam entre 7,3 e
25,5 ms de p95, e `limit: 200` sai de 81,40 ms (sequencial) para **25,48 ms** —
a meta de 50 ms da Task 72 está atingida com o dobro de folga.

Sob **quatro cópias simultâneas do binário de teste**, `limit: 200` mediu, em
cinco rodadas de quatro, entre **73,0 e 136,3 ms**. O sequencial, no mesmo
harness, media 79,8–90,8 ms. Lido como está, o número diz que a otimização
piorou o caso sob carga — mas **o harness deixou de medir o que media**: quatro
cópias com pool de 8 põem 32 leitores em 12 núcleos, e um servidor só nunca faz
isso. Quando o código era sequencial, quatro cópias eram quatro leitores e o
harness era um proxy honesto de "máquina ocupada"; hoje ele é proxy de "quatro
vezes a nossa própria concorrência".

Quem escolheu `maxSnippetWorkers = 8` foi a coluna de processo único, medida em
`internal/service/search.go`: 16 trabalhadores são piores que 8 nas duas escalas,
e a 500 notas não são melhores que 4. **Fica registrado como lacuna** que não há,
hoje, um harness de carga que estresse a máquina sem multiplicar o pool do
servidor.

## Fechamento da Parte I do M7 — Task 87 (2026-08-09)

Task 87 não envia código — mede o efeito das Tasks 78 a 86 (seis delas mudam o
calculo de score de busca; as demais tocam `note_read` em lote e o cache de
metadados) e fecha a documentação. **Esta tarefa não tem prova de mutação: não
altera nenhuma regra de código, então não há regra para provar por mutação.**

### O que já tinha número e não foi remedido

Por instrução explícita do lote: "não remeça o que já tem número, a menos que
suspeite dele". Nenhuma das duas rasuras abaixo deu motivo para suspeita.

- **RNF-02** (boot com cache válido): **371–472 ms** num cofre real de 4.165
  notas (2026-08-06), contra baseline de 1192–1396 ms sem `index_cache`.
  Medido pela Task 85 (commit `4d97943`) e corroborado independentemente
  (commit `6f5a842`), ambos `git cat-file -t` confirmando `commit`. **NÃO
  ATINGIDO** contra o teto de 300 ms. Nenhuma das Tasks 78-86 tocou
  `Index.VerifyFreshness` nem `LoadIndexCache`, os dois pontos que a análise
  anterior aponta como o custo residual (varredura sequencial de `Stat` por
  arquivo, agravada por sincronização de nuvem) — não há razão técnica para
  esperar que o número tenha mudado, e por isso não foi remedido aqui.
- **RNF-06** (reindexação de arquivo único): **334,87 µs** de mediana, p95
  544,87 µs (5.000 notas, lote=20), commit `d6fb7d0`, duas provas de mutação
  coladas no ledger (`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`,
  seção "Task 86"). **Atingido**, folga de ~60x sobre o teto de 20 ms.

### O que esta tarefa remediu: RNF-04 e RNF-07, no cofre sintético de 5.000 notas

**Por que o cofre sintético e não o real desta vez.** A tabela de RNF-04 que
esta tarefa atualiza (`limit: 200` a 5.000 notas, oito formatos de consulta)
e a de RNF-07 (RSS a 5.000 notas, cache quente/frio) foram estabelecidas no
cofre sintético gerado por `scripts/gen_vault.ps1 -Notes 5000 -Seed 42`
(2026-08-01), não no cofre real — ao contrário de RNF-02, que só tem número no
cofre real. Remedir a mesma tabela exige o mesmo cofre; usar o real aqui
trocaria a variável errada (cofre, não código) e tornaria o antes/depois
incomparável.

**Achado incidental antes de remedir.** O diretório `%TEMP%\vault_5000` já
existia na máquina, mas com 5.000 notas, 3 anexos e 6,7 MB — não bate com o
cofre documentado (50 anexos, 1,27 MB, 10.101 links, 1.518 quebrados), então
não era o corpus de referência: provavelmente um cofre gerado por uma tarefa
anterior do mesmo lote, com parâmetros diferentes de `-BodyKB`. Regenerado com
o comando exato documentado:

```
pwsh -File scripts/gen_vault.ps1 -Out "$env:TEMP\vault_5000" -Notes 5000 -Seed 42
```

```
[OK] Cofre sintético gerado em C:\Users\jonyd\AppData\Local\Temp\vault_5000
[*] Notas: 5000
[*] Anexos: 50
[*] Tamanho total: 1.27 MB (1329475 bytes)
[*] Links totais: 10101
[*] Links quebrados: 1518
```

Confere byte a byte com a geração de 2026-08-01 (mesma semente, mesmo script).

#### RNF-04 — `TestScale5000_RNF01_RNF02_RNF07_RNF04`, `internal/service/rnf5000_test.go`

Mede através de `svc.Search`, a mesma chamada que `internal/mcpsrv/tools_read.go`
faz para a tool `vault_search` (`s.svc.Search(ctx, service.SearchOptions{...})`)
— é a "pilha inteira" que o RNF nomeia: parsing da consulta, filtros,
limit/offset e leitura de disco por trecho. `BenchmarkSearchLimit200` (que caiu
de 218,5 ms para 115,0 ms segundo o revisor) mede só o kernel de busca, sem
passar por essa camada — por isso não substitui esta medição.

```
go test ./internal/service/ -run TestScale5000_RNF01_RNF02_RNF07_RNF04 -v -count=1
```

Três rodadas independentes (a suíte já tolera até 3 antes de reprovar, para
separar pico de carga de regressão real — aqui todas as três concordam):

| Formato | Rodada 1 p95 | Rodada 2 p95 | Rodada 3 p95 | Status (alvo ≤ 100 ms) |
|---|---|---|---|---|
| termo amplo, limit default | 29,52 ms | 31,80 ms | 33,37 ms | **Atingido** |
| dois termos | 14,28 ms | 10,67 ms | 13,97 ms | **Atingido** |
| termo seletivo | 11,99 ms | 11,27 ms | 8,35 ms | **Atingido** |
| filtro de pasta | 25,78 ms | 32,94 ms | 29,94 ms | **Atingido** |
| filtro de tag | 31,37 ms | 34,24 ms | 33,07 ms | **Atingido** |
| frase exata | 19,70 ms | 20,82 ms | 17,51 ms | **Atingido** |
| trecho máximo | 13,23 ms | 12,31 ms | 10,53 ms | **Atingido** |
| `limit: 200` | **122,55 ms** | **120,06 ms** | **119,18 ms** | **NÃO ATINGIDO** |

Saída bruta da primeira rodada:

```
=== RNF-04 (Latencia vault_search p95 5.000 notas) ===
  termo amplo, limit default     mediana 19.75ms    p95 29.517ms
  dois termos                    mediana 9.9605ms   p95 14.2779ms
  termo seletivo                 mediana 5.7207ms   p95 11.989ms
  filtro de pasta                mediana 20.5042ms  p95 25.7847ms
  filtro de tag                  mediana 20.5481ms  p95 31.3671ms
  frase exata                    mediana 13.0583ms  p95 19.7018ms
  trecho maximo                  mediana 7.9574ms   p95 13.2298ms
  limit maximo do schema         mediana 107.7772ms p95 122.5453ms
--- PASS: TestScale5000_RNF01_RNF02_RNF07_RNF04 (11.19s)
```

**Sete formatos de oito seguem atingidos, com folga maior que antes** (o pior
deles, "termo amplo", tinha p95 de 94,54 ms nas Tasks 78-86 anteriores; hoje
está em 29–33 ms). `limit: 200` caiu de 181,25 ms para a faixa 119–123 ms —
**queda de ~33%, e segue NÃO ATINGIDO** contra o teto de 100 ms. Registrar essa
faixa é a resposta certa; chamar de "quase lá" e parar omitiria os ~20 ms que
faltam. Medido sem `-race` (o teto do RNF-04 só vale sem o detector, que
multiplica a latência por 2 a 6 — mesma regra já cobrada por `verify.ps1`).

#### RNF-07 — `WorkingSet64` do processo real, cofre sintético de 5.000 notas

`scripts/measure.ps1` não aceita `--cache-dir`, e medir "frio" vs "quente"
exige controlar se o cache já existe antes do boot. Usado um script local
equivalente (mesma sequência de handshake MCP, mesmo `SettleMs`, mesmo "reporta
o pico, não a última amostra"), não commitado — só chama o binário compilado
com `--cache-dir` explícito, a mesma superfície que `measure.ps1` já expõe por
outro caminho.

```
pwsh -File scripts/build.ps1
# depois, por partida: gobsidian.exe serve --vault <vault_5000> --cache-dir <dir vazio ou já preenchido>
# handshake MCP, sleep de acomodacao (8s), 5 amostras de WorkingSet64 a 200ms, reporta o pico
```

**Cache quente** (cache já presente e válido; 3 partidas, settle 8 s, 5
amostras cada):

| Partida | `index_ms` | `index_origin` | RSS pico |
|---|---|---|---|
| 1 | 280 | cache | 38,10 MB |
| 2 | 283 | cache | 37,95 MB |
| 3 | 267 | cache | 38,10 MB |

**Cache frio** (diretório de cache vazio, servidor reconstrói e grava; 3
partidas, settle 8 s, cache apagado antes de cada uma):

| Partida | `index_ms` | `index_origin` | RSS pico |
|---|---|---|---|
| 1 | 508 | build | 54,82 MB |
| 2 | 513 | build | 54,76 MB |
| 3 | 538 | build | 54,69 MB |

**RNF-07 estava NÃO ATINGIDO (67,08 MB quente / 112,96 MB frio, 2026-08-01) e
agora está ATINGIDO nos dois regimes**: 37,95–38,10 MB quente (43% abaixo do
teto de 60 MB) e 54,69–54,82 MB frio (9% abaixo). Consistente com a redução de
alocação de 89% relatada pelo revisor para o lote — a fração que chega a RSS é
menor que 89% porque RSS inclui runtime do Go, stacks e páginas ainda
residentes que a redução de alocação no heap não toca diretamente, mas a
direção e a magnitude batem.

O instrumento foi conferido antes de aceitar o resultado: o cofre regenerado
bate nota por nota, anexo por anexo e link por link com a geração de
2026-08-01 (mesma semente), então a comparação antes/depois é sobre o mesmo
corpus, não sobre um corpus parecido.

### Tabela de fechamento — os quatro RNFs, antes e depois

| RNF | Métrica (alvo) | Antes (M7, pré-Task 78) | Depois (2026-08-09) | Estado |
|---|---|---|---|---|
| **RNF-02** | Boot com cache válido (≤ 300 ms) | 1192–1396 ms sem `index_cache` | **371–472 ms** (cofre real, 4.165 notas; não remedido nesta tarefa — número já estabelecido pela Task 85, 2026-08-06) | **NÃO ATINGIDO** |
| **RNF-04** | `vault_search` p95 (≤ 100 ms) | `limit: 200` em 181,25 ms; outros 7 formatos já atingidos | 7 de 8 formatos atingidos (7–33 ms); `limit: 200` em **119–123 ms** | **Parcial (NÃO ATINGIDO em 1 de 8 formatos)** |
| **RNF-06** | Reindexação de arquivo único (≤ 20 ms) | mediana 20,35 ms, p95 30,14 ms | **334,87 µs** mediana, p95 544,87 µs (Task 86, não remedido nesta tarefa) | **Atingido** |
| **RNF-07** | RSS em repouso (≤ 60 MB) | 67,08 MB quente / 112,96 MB frio | **37,95–38,10 MB** quente / **54,69–54,82 MB** frio | **Atingido** |

Dos quatro RNFs que fechavam o M6 como não atingidos, **dois seguem não
totalmente atingidos hoje** (RNF-02, e RNF-04 num único formato de oito) e
**dois passaram a ser atingidos** (RNF-06 pela Task 86, RNF-07 como efeito
colateral das otimizações de busca das Tasks 78-85). Nenhum teto foi
afrouxado para chegar a esse resultado — decisão fechada do lote (D-M7,
"nenhum teto de RNF é afrouxado nesta batelada"), e os números acima medem
contra os mesmos alvos do PRD.

## Parte II do M7 — carga sob demanda e arena mapeada (2026-08-10)

### Task 88 — índice de busca carregado só na primeira `vault_search`

`prepararIndiceDeBusca` deixou de rodar incondicionalmente no boot. A carga só
dispara na primeira chamada de `vault_search`; até lá a tool devolve
`INDEX_BUILDING`, nunca lista vazia. `--eager-search` liga o comportamento
antigo.

Medido no cofre real de 4.490 notas, três partidas, sem nenhuma chamada a
`vault_search`:

```
RSS 125,2 MB   index_ms=622
RSS 125,0 MB   index_ms=507
```

Baseline no mesmo cofre e máquina, cache quente, antes da mudança: **501,9 /
502,1 / 501,9 MB**. **~502 MB → ~125 MB numa instância que nunca busca — queda
de 75%.** Detalhe completo, inclusive a prova de mutação, no ledger
(`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`, "Task 88 (M7)").

Isto reduz o público da Task 89 seguinte: só instância que **busca** carrega o
array de posições, então só ela se beneficia de mapeá-lo.

### Task 89 — arena de posições mapeada do arquivo

O array de posições era ~291 MB (na época do brief) dos ~500 MB de uma
instância com índice carregado — cada processo alocava a própria cópia no
heap. Mapeado do arquivo em modo leitura, essas páginas passam a ser
file-backed e reclamáveis pelo sistema operacional em vez de heap privado
sempre residente.

**Formato do cache: 5 → 6.** `escreveCache` passou a gravar, depois do corpo em
varint de sempre (que continua sendo o que a decodificação integral usa), uma
seção fixa de posições em 16 bytes cada — alinhada em 8 — e um rodapé de 24
bytes no fim do arquivo. `LoadInvertedCache` tenta mapear essa seção primeiro;
qualquer recusa (cache dentro do cofre, arquivo pequeno demais, rodapé
ausente/corrompido, mmap falhou na plataforma) cai para `os.ReadFile` + decodificação
integral de sempre — nunca mapeia lixo.

**Toda troca de formato reconstrói o cache de busca no boot seguinte** — regra
já registrada no `CLAUDE.md` para a troca 1→5, e que vale de novo aqui.
Medido no cofre real (formato 5 → formato 6, tokenização completa das 4.490
notas): **157,3 s** (binário anterior à Task 89, grava o formato 5) contra
**157,9 s** (binário desta tarefa, grava o formato 6) — a diferença entre os
dois é ruído; escrever a seção fixa adicional não mede no tempo de
construção, que é dominado pela tokenização. Os dois rodam em segundo plano,
com as outras onze tools respondendo desde o primeiro segundo (mesmo
mecanismo de sempre).

**Custo em disco.** `inverted_cache.gob` do cofre real:

| Formato | Tamanho | 
|---|---|
| 5 (antes) | 108.492.193 B (103,5 MB) |
| 6 (depois) | 573.554.656 B (546,9 MB) |

**5,3× maior — 443,5 MB a mais**, não os ~291 MB estimados no brief: o cofre
real cresceu desde que aquele número foi escrito (é um cofre de uso diário do
usuário), então o array de posições hoje tem mais entradas do que as
18.229.295 registradas antes. É o troco pré-decidido — disco por memória
compartilhada — só que o disco pago é maior do que a estimativa.

**RSS, cofre real, três partidas com `--eager-search` (equivalente a ter
buscado — força a mesma carga sob demanda da Task 88), cache aquecido por uma
partida anterior ignorada na medição:**

| Cenário | Working Set (soma) | Working Set - Private (soma) |
|---|---|---|
| 1 instância, formato 5 (antes) | 584,9 MB | 574,0 MB |
| 1 instância, formato 6 + mmap (depois) | 244,3 MB | 129,9 MB |
| 3 instâncias, formato 5 (antes) | 1.754,5 MB | 1.721,9 MB |
| 3 instâncias, formato 6 + mmap (depois) | 732,7 MB | 389,6 MB |

Queda no agregado de três instâncias: **58,2% em Working Set total, 77,4% em
Working Set-Private** — a métrica certa aqui, porque Working Set total conta
página mapeada residente inteira em CADA processo que a tem residente, mesmo
quando o cache de páginas do SO a compartilha; Working Set-Private é o que
sai do cômputo de cada processo quando a memória deixa de ser heap privado.
**Acima do critério de parada de 30% definido no brief — não foi preciso
considerar a opção (b) (mapear o arquivo comprimido e decodificar sob
demanda).**

Ressalva que o relatório desta tarefa também registra: a queda percentual por
instância é **igual** com uma ou com três instâncias (77,4% nos dois casos), e
3 × (Working Set-Private de uma instância) bate com o Working Set-Private
agregado de três — não há queda ADICIONAL mensurável por instância extra
além da primeira. Isso é esperado: no Windows, página mapeada de arquivo em
modo leitura sai da contagem de "Private" por ser file-backed e reclamável,
**independente de quantos processos a têm mapeada** — os contadores de
processo do Windows não expõem uma métrica de "páginas de quadro físico
únicas entre N processos" (o equivalente do PSS do Linux). O que está medido
e provado é que o array deixou de ser heap privado sempre residente; que as
páginas residentes são efetivamente as MESMAS entre os processos (e não só
"igualmente baratas cada uma") seria preciso uma ferramenta como RAMMap/VMMap
para confirmar — não medido nesta tarefa.

**Rename atômico sobre arquivo mapeado.** `os.Rename` do salvamento atômico
falha no Windows se o processo ainda tem o arquivo de destino mapeado — só
importa no caminho raro de cache PARCIAL retomado e depois regravado
(`buildInvertedIndex` continuando depois de `AdotarDe` um cache incompleto
carregado via arena). Confirmado experimentalmente antes de confiar na
correção: com o arquivo ainda mapeado, `os.Remove` nele falha de verdade
(`TestSaveOverwritesMappedCache`, `internal/search/persist_test.go`).
`promoverArenaSePresente` copia as posições para o heap e desmapeia ANTES do
`os.Rename`; no caminho comum (cache completo, "pronta") `SaveInvertedCache`
nunca roda de novo no mesmo processo, então isto não dispara ali.

Ver ledger (`.superpowers/sdd/2026-07-25-gobsidian-v01/progress.md`, "Task 89
(M7)") para a prova de mutação e a saída de `verify.ps1`.

### Task 90 — RNF-30 reformulado, antes de abrir o primeiro socket

Reabre uma decisão fechada, com autorização explícita do dono do projeto em
2026-08-05: de "nenhuma requisição de rede... nenhum socket de saída em
nenhuma circunstância" para **"nenhum socket que saia da máquina"**. Um socket
de domínio Unix não atravessa rede — o kernel resolve um caminho de arquivo
especial —, então a garantia contra exfiltração se mantém sob a formulação
nova; o que muda é o mecanismo de IPC permitido, não a superfície de
exposição. Texto completo, com data e autorização, em `docs/PRD.md` §6.4.

A regra passou a ter duas camadas: nenhum pacote sob `internal/` ou `cmd/`
importa `net/http` ou qualquer `net/*` (o pacote `net` em si passou a ser
permitido); e dentro de `net`, só `net.Dial`/`net.Listen` com a constante
literal `"unix"` são aceitos — rede vinda de variável é recusada estaticamente
mesmo que o valor em tempo de execução seja `"unix"`. Verificado por
`tools/netcheck` via `go/analysis` + `go/types`, com as três formas de
violação plantadas e removidas para confirmar disparo (`net.Dial("tcp", ...)`,
`net.Dial` com rede vinda de variável, `http.Get`). Detalhe e saída completa
no ledger, seção "Task 90".

### Task 91 — transporte IPC e o processo-ponte

Implementa `internal/ipc` (o transporte AF_UNIX, chave do socket, handshake de
versão, `Dial`/`Listen`, proxy de bytes) e a ponte burra em
`cmd/gobsidian/ponte.go`: depois do handshake ela só copia bytes, não
interpreta JSON-RPC. RSS medido conectada a um daemon de teste: **13,77 MB**
de Working Set, contra a dezenas ou centenas de MB de uma instância completa.
Fallback para o modo em processo é obrigatório sempre que a discagem falha —
socket ausente, conexão recusada ou versão incompatível levam ao mesmo lugar.

Esta tarefa ainda não tinha o daemon (Task 92); só exercitou o fallback em
produção e o handshake/proxy contra um par de teste. Latência de uma chamada
de tool completa através da ponte não foi medida aqui por depender do daemon
real — ver Task 92 e a tabela desta seção.

### Task 92 — daemon: uma instância por cofre, com ciclo de vida próprio

`internal/daemon`: um único `*mcpsrv.Server` compartilhado por N conexões
sobre o socket de `internal/ipc`, resolvido por cofre via a mesma
`config.VaultKey` que deriva o caminho do socket. `EnsureStarted` resolve a
corrida de inicialização por arquivo de lock; o handshake carrega `ReadOnly` e
`VaultKey` e recusa (`ErrConfigMismatch`) duas pontes do mesmo cofre com
configuração divergente conectadas ao mesmo daemon. Ociosidade (padrão: 900 s
/ 15 min) reusa `lifecycle.Trigger`. `GOBSIDIAN_NO_DAEMON=1` pula a decisão
inteira — é como desligar o daemon sem reverter código, e é o mecanismo que
`README.md` documenta.

**Duas descobertas reais, medindo — não hipóteses.** Primeiro: o lock por
arquivo sozinho não bastava contra um cofre que demora para responder — dez
pontes lançadas juntas contra o cofre real, sob carga pesada da máquina,
produziram **dois** daemons vivos ao mesmo tempo, com quase um minuto de
intervalo entre os dois "daemon iniciado" no log. Corrigido com uma segunda
checagem (um dial) logo após adquirir o lock, antes de iniciar; ver "Risco
residual conhecido" abaixo — a correção reduz a janela, não a elimina por
construção. Segundo: um script de medição que esperava o sinal errado (da
ponte, não do daemon) ficou "travado" por 30 minutos sem bug nenhum de
produto — o daemon tinha terminado de construir o índice e ficado ocioso,
correto.

**RSS agregado, cofre real de 5.619 notas, três instâncias:**

```
ANTES (sem daemon, GOBSIDIAN_NO_DAEMON=1):
  memória física consumida: 520,3 MB
  Working Set total: 736,8 MB   Working Set-Private total: 389,8 MB

DEPOIS (3 pontes + 1 daemon compartilhado):
  memória física consumida: 260,0 MB
  Working Set total: 295,1 MB   Working Set-Private total: 141,6 MB
```

Working Set agregado caiu 59,9%, Working Set-Private 63,7%, e a **memória
física realmente consumida** (via `Win32_OperatingSystem.FreePhysicalMemory`
antes/depois — a métrica que não confunde página compartilhada com página
duplicada, porque o Windows não expõe um equivalente do PSS do Linux por
processo) caiu **50,0%**. Cada ponte custa ~16 MB contra ~246 MB de uma
instância independente; o daemon sozinho custa essencialmente o mesmo que uma
instância independente — o ganho agregado vem inteiro de não repetir esse
custo por sessão.

Ver ledger, seção "Task 92 (M7)", para as provas de mutação, o gate de
órfãos com o quarto cenário (`daemon-idle`) e o teste do daemon morto no meio
de uma chamada.

### Task 93 — tabela de 1, 3 e 5 sessões, nas três configurações

Fecha a Parte II. Não envia código; mede o que faltava para a recomendação
final. As Tasks 88, 89 e 92 já tinham medido pedaços desta tabela em
snapshots diferentes do cofre real (4.490 notas na Task 89, 5.619 na Task
92 — o cofre cresce com o uso diário). Para que as nove células fossem
comparáveis **entre si**, esta tarefa remediu as três colunas inteiras
**no mesmo cofre real, na mesma sessão de medição, 2026-08-10**: 4.513
notas, 71 anexos — o número que o próprio servidor reporta no log de boot
(`notes=`), não uma contagem bruta de arquivos `.md` no disco, que conta
entradas dentro de `.obsidian/` e outras que `vault.Walk` exclui.

**Nome da coluna do meio corrigido.** "Com a Task 88" nomeava mal a coluna:
88 (carga sob demanda) e 89 (arena mapeada) vieram juntas, e não dá para
isolar o efeito de uma sem um terceiro binário. A coluna mede o efeito das
duas somadas, sem daemon, e passa a se chamar **"Parte II, sem daemon"**.

**Duas rodadas de correção de método, as duas descobertas medindo, não
supondo:**

1. **`FreePhysicalMemory` (chamada de "física" na primeira versão desta
   tabela) é ruidosa demais para decidir a célula que importa.** Repetindo
   a MESMA configuração (1 sessão, sem daemon) três vezes seguidas, a queda
   de memória livre do sistema saiu **223,8 / 252,8 / 317,2 MB** — variação
   de 93,4 MB **dentro da mesma configuração**, maior que o efeito de
   ~16 MB que a comparação com/sem daemon precisa resolver em uma sessão
   só. É uma métrica de sistema inteiro, e qualquer outra coisa rodando na
   máquina no instante da amostra entra na conta. `WorkingSet` e
   `WorkingSet-Private`, por processo, saíram estáveis nas mesmas três
   repetições (244,4 / 244,4 / 244,2 MB) — são a métrica que decide daqui
   pra frente; física fica só como contexto, nunca como prova, e só onde o
   efeito (centenas de MB) é grande demais para o ruído observado
   (dezenas de MB) mudar a direção.
2. **A primeira tentativa de reproduzir "dois daemons vivos" (a corrida da
   Task 92) foi falso positivo do próprio script de medição, não uma
   segunda ocorrência do bug.** Contando "todo processo `gobsidian.exe`
   vivo" sem filtrar pelo cofre, uma bateria de N=5 apareceu com 7
   processos em vez de 6. O sétimo era um daemon de
   `testdata\vault_small` — outro processo, de outro agente, rodando no
   MESMO worktree ao mesmo tempo, não relacionado a esta medição. Corrigido
   filtrando por linha de comando conter o caminho do cofre alvo antes de
   somar OU de matar qualquer processo; refeita a medição com o filtro
   certo, o resultado voltou a ser exatamente 1 daemon (a mesma contagem
   que as baterias anteriores, feitas antes desse processo concorrente
   aparecer, já mostravam). **Ressalva honesta e sem meio-termo:** a
   limpeza anterior deste script matava `gobsidian.exe` por nome, sem
   filtrar — se algum processo concorrente de outro agente estava vivo
   durante essa janela, pode ter sido encerrado sem querer. Corrigido antes
   de qualquer medição nova ser aceita.

**Método**, com as correções acima: `Win32_PerfFormattedData_PerfProc_Process`
(`WorkingSet`, `WorkingSetPrivate`) por processo, **filtrado pela linha de
comando conter o caminho do cofre desta medição**, somado; a célula
decisiva (N=1) repetida **3 vezes por configuração**, N=5 repetida 2 vezes,
N=3 medida uma vez (diferença grande o bastante para não depender de
repetição). Todas as três colunas usam `--cache-dir` dedicado (fora do
cofre e fora do padrão), aquecido por uma partida ignorada antes de cada
bateria, e a coluna "com o daemon" soma a(s) ponte(s) **mais** o processo
`daemon` detached que elas iniciam. **As sessões de fato buscam**:
`--eager-search` força a mesma carga que `vault_search` dispararia de
qualquer forma — sem isso as três colunas mediriam a mesma coisa (Task 88
já garante que sessão que só lê/escreve nunca carrega o índice de busca).

**Coluna "hoje"**: binário compilado no commit `782e813` (o commit
imediatamente anterior à Task 88 — `git cat-file -t 782e813` confirma
`commit`), a única forma de reproduzir fielmente "sempre carrega o índice
inteiro, sem mmap, sem daemon" já que o binário atual não tem mais esse
caminho. **1 e 3 sessões medidas no cofre de hoje** (4.513 notas) para
ficarem comparáveis com as outras duas colunas; **5 sessões também
medidas** (não foi preciso reaproveitar número antigo de outro tamanho de
cofre — o binário já estava compilado e o cofre já estava com cache
aquecido, então a medição de 5 custou só mais uma rodada, não um worktree
novo).

| Sessões simultâneas | hoje (pré-Parte II) | Parte II, sem daemon | com o daemon |
|---|---|---|---|
| **1** | WS 585,0 MB · WS-Priv 574,1 MB | WS 244,3 MB · WS-Priv 129,8 MB (média de 3) | WS 260,3 MB · WS-Priv 134,3 MB (média de 3) |
| **3** | WS 1.754,4 MB · WS-Priv 1.721,7 MB | WS 733,3 MB · WS-Priv 389,6 MB | WS 288,7 MB · WS-Priv 141,1 MB |
| **5** | WS 2.923,1 MB · WS-Priv 2.868,9 MB | WS 1.221,3 MB · WS-Priv 648,6 MB (média de 2) | WS 319,4 MB · WS-Priv 148,3 MB (média de 2) |

`WS` = Working Set total. `WS-Priv` = Working Set-Private (sai da conta
quando a página é file-backed e compartilhável — ver a ressalva da
Task 89 sobre não confirmar compartilhamento físico estrito sem
RAMMap/VMMap). Os três números de "hoje" e a linha N=3 das outras duas
colunas são medição única; onde há "média de N", as N amostras
individuais estão na seção de verificação abaixo.

**A célula decisiva — 1 sessão, sem daemon contra com daemon:**

| Métrica | sem daemon (3 amostras) | com daemon (3 amostras) | diferença |
|---|---|---|---|
| WS | 244,4 / 244,4 / 244,2 | 260,8 / 260,1 / 260,0 | **+16,0 MB (+6,5%)** |
| WS-Private | 129,9 / 129,9 / 129,6 | 134,8 / 134,1 / 134,0 | **+4,5 MB (+3,5%)** |

**A expectativa registrada antes de medir — bridge (~15 MB) mais o processo
do daemon (~246 MB) custam mais que uma instância independente sozinha —
bateu.** As três amostras de cada lado não se sobrepõem (o pior caso "sem
daemon" é 244,4; o melhor caso "com daemon" é 260,0), então a diferença não
é ruído de uma medição só — é o custo real de manter um segundo processo
vivo quando não há segunda sessão para compartilhar com ele.

**3 e 5 sessões, sem e com daemon:**

| Sessões | WS sem → com | WS-Priv sem → com |
|---|---|---|
| 3 | 733,3 → 288,7 MB (**−60,6%**) | 389,6 → 141,1 MB (**−63,8%**) |
| 5 | 1.221,3 → 319,4 MB (**−73,8%**) | 648,6 → 148,3 MB (**−77,1%**) |

### Recomendação — o daemon deve vir ligado por padrão?

**Não — os dados invertem a leitura inicial.** A regra que o brief desta
tarefa já tinha fixado antes de qualquer medição — "se a coluna de uma
sessão mostrar ganho zero ou negativo, isso decide a questão" — se aplica
diretamente: a sessão única **perde** 16,0 MB de Working Set (6,5%) e 4,5 MB
de Working Set-Private (3,5%) com o daemon ligado, medido com três
repetições que não se sobrepõem. Não é ruído, e não é zero: é negativo.

**As duas pontas, com número:**

- **A favor:** a partir de três sessões simultâneas que buscam de verdade
  contra o mesmo cofre, o ganho é grande (−60,6% de Working Set em 3,
  −73,8% em 5) e cresce com N. Para quem sabe que vai abrir sessões
  concorrentes, o daemon é uma vitória clara.
- **Contra:** a sessão única — o caso mais comum, um host MCP por vez — paga
  um custo real, ainda que pequeno em termos absolutos (16 MB), só para
  manter viva uma capacidade que ela não usa. Some a isso o processo de
  vida longa, o arquivo de lock com a corrida residual (abaixo) e o modo de
  falha novo, e o argumento "compartilhamento sem downside" que sustentava
  a recomendação anterior não sobrevive à célula que faltava medir.

**A recomendação técnica desta medição era desligar por padrão**, com
opt-in para quem sabe que vai rodar sessões concorrentes. O critério tinha
sido fixado antes de medir — ganho zero ou negativo na sessão única decide
a questão — e a sessão única deu negativo.

**O padrão embarcado na v1.1.0 é o contrário: o daemon sai ligado**
(decisão de projeto de 2026-08-10, tomada com esta tabela à vista). O que a
decisão troca é explícito e está inteiro nos números acima — 16 MB a mais
na sessão única, para todo mundo, em troca de os −60% e −74% de três e
cinco sessões chegarem sem depender de o usuário encontrar uma variável de
ambiente na documentação.

Quem quer o comportamento que os números recomendam liga
`GOBSIDIAN_NO_DAEMON=1` e paga zero. **Esta seção fica como está, com a
recomendação técnica contrária ao padrão embarcado, de propósito:** quem
reabrir a questão precisa do número que a contraria, não de uma
justificativa retroativa do que foi decidido.

**O que isto NÃO muda:** a Task 88 continua sendo o maior ganho por linha
mexida do marco — ela é quem faz a MAIORIA das sessões (leitura e escrita,
sem busca) nunca pagar custo nenhum, e essa tabela nem chega a exercitar
esse caso, porque as três colunas forçam `--eager-search` de propósito. O
daemon continua tendo valor real — só não de graça, e não para o caso
comum.

### Risco residual conhecido — corrida de inicialização do daemon

`EnsureStarted` (`internal/daemon/lock.go`) usa um arquivo de lock
(`O_CREATE|O_EXCL`) para serializar quem tenta iniciar o daemon do mesmo
cofre, mas libera o lock assim que a própria chamada termina — o que inclui
esperar o socket responder. Isso serializa quem disputa o **mesmo instante**,
não "existe um daemon rodando". **Medido, não hipotético:** dez pontes
lançadas juntas contra o cofre real, sob carga pesada da máquina (o gate de
órfãos rodando em paralelo), produziram dois daemons vivos simultaneamente,
uma vez, antes da correção — quase um minuto de intervalo entre os dois logs
de "daemon iniciado". A correção adiciona uma segunda checagem (um dial)
logo após adquirir o lock, antes de chamar `iniciar`: se outro processo já
respondeu enquanto esta chamada esperava a vez, usa esse e nunca inicia um
segundo.

**Isso reduz a janela para milissegundos; não é exclusão mútua entre
processos por construção.** É um *check-then-act*: entre o dial de
confirmação e o `SpawnDetached` de fato ainda existe uma janela teórica, só
que agora medida em milissegundos em vez de "a duração inteira do lock,
incluindo o tempo de boot do daemon". Reconfirmado sem daemon duplicado em
duas rodadas de dez pontes simultâneas depois da correção, em máquina sem
carga concorrente — a janela teórica continua existindo sob a mesma condição
que a expôs uma vez (contenção pesada de CPU no instante exato da corrida).
Não observado em produção fora da condição de teste que a expôs. Se dois
daemons chegarem a coexistir, o pior caso é dois processos com índice
próprio competindo pelo mesmo socket — não corrupção do cofre, que continua
protegida pelas escritas atômicas e pelo mutex por caminho (`internal/writer`).

## Recorte de trecho: a busca que pedia a lista inteira de postings (2026-08-12)

Três mudanças, medidas em sequência, no caminho de `vault_search`. A primeira
respondeu por quase todo o ganho; as outras duas existem porque sem elas os
números da primeira seriam lidos errado.

### O que o perfil mostrou

`GenerateSnippet` chamava `Inverted.Postings(termo)` — que materializa a lista
inteira de postings do termo e, no índice construído do zero, ainda a **ordena**
— e depois varria essa lista linearmente para achar **uma** nota. Uma vez por
resultado, por termo. Com `limit: 200` num cofre de 5.000 notas isso são 200
ordenações de milhares de elementos e uma varredura de 2 milhões de comparações
de string, para extrair 200 offsets.

Perfil de `BenchmarkSearchLimit200`, filtrado em `service.Search`:

```
search.(*Inverted).Postings          29,31 s   40,23% do CPU total
└─ sort.Slice                        26,55 s   90,58% de Postings
search.CalculateBM25                  0,53 s    0,73%
vault.ReadRange                       1,25 s    1,72%

alloc_space: Postings via GenerateSnippet   1.429,98 MB   79,63%
```

A pontuação BM25 custava menos de 1%. O gargalo era a montagem do trecho.

`Inverted.Positions(termo, caminho)` já existia desde a Task 61, faz busca
binária, e resolvia exatamente esta pergunta — mas tinha sido aplicada só ao
casamento de frase, não ao recorte de trecho.

### Medição de `Postings` para `Positions`

`benchstat`, n=6 por braço (a linha `TermoAmploCache` deu `~` com a média
subindo na primeira passada e foi remedida isolada com n=10 — era ruído):

```
                          antes           depois         delta
sec/op
SearchLimit200-12       175,43m ± 29%   22,89m ± 31%   -86,95% (p=0,002 n=6)
SearchLimit200Cache-12   40,70m ±  8%   27,67m ± 32%   -32,02% (p=0,002 n=6)
SearchTermoAmplo-12      30,46m ±  6%   17,85m ±  9%   -41,38% (p=0,002 n=6)
SearchTermoAmploCache-12 16,26m ±  4%   14,98m ±  5%    -7,87% (p=0,003 n=10)

B/op
SearchLimit200-12       50,404Mi ± 0%   3,476Mi ± 0%   -93,10% (p=0,002 n=6)
SearchTermoAmplo-12      7,820Mi ± 0%   3,126Mi ± 0%   -60,02% (p=0,002 n=6)
```

Depois da troca, o perfil do mesmo benchmark põe `Positions` em **0,01 s**, e o
que sobra no recorte é `vault.ReadRange` com 97% do custo de `GenerateSnippet`
— dos quais 68% em `vault.Open`, isto é, `CreateFile` do Windows.

Uma alternativa mais ambiciosa foi **descartada por medição**: passar a âncora
do trecho de dentro do BM25, eliminando a consulta ao índice. O que ela removeria
custa 0,1% do perfil depois desta mudança. Mudança de assinatura pública por 0,1%
é dívida.

### O benchmark mirava o ramo que o servidor não executa

`Inverted.Postings` tem dois ramos. Índice construído do zero: `base == nil`,
tudo vive no delta em mapas, e a função **ordena**. Índice vindo do cache:
`base != nil` e delta vazio, e ela devolve a fatia do base sem ordenar. O
servidor em produção sempre carrega do cache.

`benchServico` construía do zero. Medido no mesmo cofre, mesma consulta:

```
BenchmarkSearchLimit200-12        174.791.983 ns/op   (ramo delta)
BenchmarkSearchLimit200Cache-12    39.565.533 ns/op   (ramo base)
```

**4,4 vezes entre os dois ramos.** `internal/service/bench_cache_test.go` entrou
por causa disso: `BenchmarkSearchLimit200Cache` e `BenchmarkSearchTermoAmploCache`
montam a pilha com o índice **carregado do cache**, e uma guarda reprova se o
cache for recusado em silêncio — sem ela o benchmark novo mediria exatamente o
ramo que ele existe para não medir.

### Cache de trecho: paga na consulta repetida, não na primeira

`internal/search/snippet_cache.go`. LRU por número de entradas, teto padrão de
1.024, chave `{caminho, hash, início, fim, maxChars}`. O hash é o `index.Note.Hash`
— xxhash do conteúdo bruto —, que `GenerateSnippet` já obtinha do `idx.Get` que
ele fazia de qualquer jeito: a invalidação não custa syscall nenhum. Não há
`Invalidate()`: nota editada muda de hash, logo muda de chave, e a entrada velha
morre por LRU. Nota que não está no índice de metadados **não é cacheada**, porque
sem hash não existe chave de invalidação.

Medição própria, os dois braços numa invocação só de `go test -count=8`, o que os
intercala:

```
                       frio (cache off)   quente (mesma consulta)   delta
sec/op                 18,18m ± 7%        14,24m ± 10%              -21,63% (p=0,000 n=8)
B/op                   3,479Mi ± 0%       3,289Mi ±  0%              -5,45% (p=0,000 n=8)
allocs/op              13,45k ± 0%        11,88k ±  0%             -11,64% (p=0,000 n=8)
```

**O caminho frio não mudou** (`~`, e `B/op`/`allocs/op` idênticos — evidência mais
forte que o `~`, porque são determinísticos). Quem melhorou o frio foi a troca de
`Postings` por `Positions`, não o cache.

Uma medição anterior desta mesma comparação, em máquina mais carregada, deu
-30,37% (n=20, ±25% nos dois braços). Fica registrada a divergência: o número
válido é o de cima, medido com ±7% e ±10%.

**Todo harness que mede latência de busca desliga este cache**
(`semCacheDeTrecho`, em `internal/service/bench_test.go`). Não é preferência:
`b.Loop` e os laços de RNF-04 repetem a MESMA consulta 30 vezes, e com o cache
ligado 29 dessas 30 são acertos — o p95 passaria a descrever a consulta repetida.
Isso foi pego na revisão: a primeira medição de RNF-04 depois do cache deu
21,49 ms para `limit: 200` e estava quente. Quem mede repetição é
`BenchmarkSearchLimit200CacheTrechoRepetido`, e o nome dele diz isso.

### `maxSnippetWorkers` re-aferido, e mantido em 8

A varredura que escolheu 8 foi feita quando cada resultado custava ~1 ms de CPU
em `Postings`. Com esse custo apagado, o trabalho por resultado virou quase só
espera de abertura de arquivo — regime diferente. Remedido em
`BenchmarkSearchLimit200Cache` com o cache de trecho desligado, seis configurações
intercaladas, n=6 por braço, base 8:

| workers | sec/op | vs 8 |
|---|---|---|
| 1 | 36,82m ± 312% | +80,29% (p=0,002) |
| 4 | 22,52m ± 16% | ~ (p=0,132) |
| 8 | 20,42m ± 60% | — |
| 16 | 18,73m ± 7% | ~ (p=0,240) |
| 24 | 19,55m ± 17% | ~ (p=0,699) |
| 32 | 20,23m ± 21% | ~ (p=1,000) |

16 foi o único candidato próximo e levou segunda passada só contra 8, com a regra
de decisão declarada antes de medir (muda se p < 0,05, n=16 por braço): 18,92m ± 9%
contra 17,97m ± 7%, `~ (p=0,254)`. **A constante fica em 8.** O que a varredura
acrescenta é que serializar ficou *mais* caro neste regime: sem custo de CPU por
resultado, o que sobra é latência de abertura de arquivo, e ela só some com
concorrência. Esta varredura foi medida uma vez e não foi reproduzida de forma
independente; ela sustenta a decisão de **não** mudar nada.

### RNF-04 remedido, a frio, `TestScale5000_RNF01_RNF02_RNF07_RNF04`

Mesmo teste e mesmo harness que registraram 119–123 ms, com o cache de trecho
desligado. Três rodadas independentes, sem `-race`:

| Formato | R1 p95 | R2 p95 | R3 p95 | Alvo ≤ 100 ms |
|---|---|---|---|---|
| termo amplo, limit default | 22,74 ms | 28,71 ms | 36,73 ms | Atingido |
| dois termos | 11,54 ms | 25,94 ms | 13,98 ms | Atingido |
| termo seletivo | 10,21 ms | 18,38 ms | 10,47 ms | Atingido |
| filtro de pasta | 21,62 ms | 54,35 ms | 32,37 ms | Atingido |
| filtro de tag | 28,19 ms | 47,22 ms | 23,25 ms | Atingido |
| frase exata | 29,59 ms | 35,28 ms | 27,78 ms | Atingido |
| trecho máximo | 15,14 ms | 20,67 ms | 11,86 ms | Atingido |
| `limit: 200` | **25,98 ms** | **82,01 ms** | **29,04 ms** | Atingido |

**Os oito formatos cabem no teto nas três rodadas, e `limit: 200` era o único que
faltava desde 2026-08-01.** A rodada 2 saiu carregada — a mediana de `limit: 200`
foi de 18,8 ms na R1 para 59,2 ms nela, sem mudança de código — e está registrada
como saiu. A margem contra os 100 ms sobrevive à pior das três, que é o que
autoriza a linha "Atingido"; se a folga fosse só na melhor rodada, a resposta
seria "parcial".

**Estas três rodadas mediram o ramo do delta**, não o que o servidor executa. O
harness carregava o cache para o RNF-02 e montava o serviço com o índice
construído do zero. A Task 94 corrigiu isso; a tabela acima fica como registro
do que foi medido, e a de baixo é a que vale.

### RNF-04 no ramo que o servidor executa (Task 94, 2026-08-12)

`internal/service/rnf5000_test.go` passou a montar o serviço com o índice
**vindo de `LoadInvertedCache`**, não com o `inv` construído do zero.
`Inverted.Postings` tem dois ramos, e o servidor em produção sempre carrega do
cache; até aqui todos os números de RNF-04 do projeto saíam do outro.

A correção carrega uma instância própria, viva até o fim do teste — a do laço
de RNF-02 é fechada a cada volta de propósito, porque cinco mapeamentos abertos
sobre o mesmo arquivo fazem o `t.TempDir()` recusar apagar no Windows. Junto
entrou uma **guarda de ramo**, no espírito da de `bench_cache_test.go`: cache
recusado em silêncio (troca de formato, versão de analisador, caminho de cofre
diferente) faz o teste **reprovar**, não medir. Sem ela a correção poderia se
desfazer sozinha numa mudança futura de formato e ninguém perceberia.

Cache de trecho continua desligado (`semCacheDeTrecho`): os laços repetem a
mesma consulta 30 vezes. Três rodadas independentes, sem `-race`:

| Formato | R1 p95 | R2 p95 | R3 p95 | Alvo ≤ 100 ms |
|---|---|---|---|---|
| termo amplo, limit default | 21,84 ms | 23,59 ms | 25,83 ms | Atingido |
| dois termos | 10,26 ms | 11,64 ms | 11,86 ms | Atingido |
| termo seletivo | 9,17 ms | 9,98 ms | 9,88 ms | Atingido |
| filtro de pasta | 28,36 ms | 24,86 ms | 26,49 ms | Atingido |
| filtro de tag | 32,24 ms | 26,07 ms | 32,17 ms | Atingido |
| frase exata | 35,45 ms | 29,44 ms | 28,78 ms | Atingido |
| trecho máximo | 13,75 ms | 12,13 ms | 13,23 ms | Atingido |
| `limit: 200` | **43,09 ms** | **28,79 ms** | **27,56 ms** | Atingido |

**Os oito formatos cabem no teto nas três rodadas também neste ramo**, e a pior
rodada de `limit: 200` (43,09 ms) deixa 57% de folga.

O que a troca de ramo **não** produziu foi um salto de desempenho, e a razão
está medida: os 4,4× entre os dois ramos foram medidos com `Postings`, e a
troca por `Positions` desta mesma sessão apagou justamente o `sort` que os
separava — depois dela, `SearchLimit200` deu 22,89m contra 27,67m de
`SearchLimit200Cache`. Os dois ramos convergiram. **O valor da correção é o
harness passar a exercitar o caminho do servidor**, não um número melhor; se
os ramos voltarem a divergir, agora é este que aparece.

**~~`TestRNF04VaultSearchLatencyP95` (500 notas) tem o mesmo defeito, e não foi
corrigido.~~ Fechado pela Task 98 em 2026-08-13**, junto com
`TestRNF04SnippetConcurrencyLimit200` e, pela Task 101,
`TestBM25KernelLatency` — os três que restavam. O texto abaixo fica como
registro do que era verdade quando foi escrito.** `createSearchService` monta o índice
com `search.NewInverted()` mais `Add` por nota, nunca grava nem lê cache, então
`base == nil` e toda leitura cai no ramo que ordena. Vale para todos os testes
que passam por esse construtor, `TestRNF04SnippetConcurrencyLimit200` inclusive.
500 notas é outra escala e a decisão é de quem é dono do requisito.

### Lacunas que continuam abertas

- ~~**`rnf5000_test.go` mede o ramo errado do índice.**~~ **Fechada pela Task 94
  em 2026-08-12**, e o resto a 500 notas pelas Tasks 98 e 101 em 2026-08-13.
  **Nenhum teste com teto mede mais o ramo do delta.** O harness passou a montar o serviço com o índice vindo do
  cache, com guarda de ramo que reprova se o cache for recusado em silêncio. A
  medição anterior fica registrada acima: ela saiu do ramo do delta, e saber
  disso é a informação que importa. Continua aberto o mesmo defeito em
  `TestRNF04VaultSearchLatencyP95`, a 500 notas — relatado, não corrigido.
- **`docs/bench-baseline.json` ficou obsoleto.** A referência de
  `BenchmarkSearchLimit200` é 373.350.430 ns/op, medida no runner do CI antes
  destas mudanças. O gate só reprova regressão, então nada quebra — mas uma
  referência dez vezes acima do real deixa de pegar regressão. Precisa ser regerada
  pelo `bench.yml` com `-UpdateBaseline`, no runner do CI; número local não é
  comparável.
- ~~**Efeito do cache de trecho no RSS não foi medido**~~ **Medido pela Task 96
  em 2026-08-12**, na seção abaixo. O custo fica em torno de 1 MB e **não é
  distinguível do ruído do instrumento**; a conta de ~1,2 MB do comentário de
  `DefaultSnippetCacheEntries` não é contrariada, e o padrão de 1.024 não ameaça
  o RNF-07.

### Custo do cache de trecho no RSS (Task 96, 2026-08-12)

**O que foi medido:** `WorkingSet64` de pico do processo real, cofre sintético de
5.000 notas (`vault_5000`), depois de uma carga de 33 consultas `limit: 200` que
enche o cache de trecho. Dois braços: teto padrão (1.024 entradas) e desligado.
Seis partidas por braço, **intercaladas** (A, B, A, B, …), mesmo diretório de
cache quente para os dois, 15 s de acomodação e 8 amostras a 200 ms por partida.

**O instrumento não foi `scripts/measure.ps1`**, e o motivo é o mesmo da seção de
RNF-07 acima: ele mede repouso logo após o boot e não emite busca nenhuma, então
o cache de trecho ficaria vazio nos dois braços e a comparação mediria zero. Foi
usado um script local equivalente (mesma sequência de handshake MCP, mesmo
"reporta o pico e não a última amostra"), com a fase de carga no meio, não
commitado. Duas coisas que ele precisou fazer e que ficam registradas:

- **`GOBSIDIAN_NO_DAEMON=1` é obrigatório.** Sem ela, `serve` encontrou um daemon
  vivo para esse cofre e virou ponte — o `WorkingSet64` medido seria o da ponte,
  não o do processo que guarda o índice e o cache. A primeira tentativa mediu
  exatamente isso antes de o log `conectado ao daemon via socket` denunciar; o
  script hoje **reprova** se essa linha aparecer.
- **`StreamReader.ReadLine()` no stderr trava.** A primeira versão ficou pendurada
  porque o prazo só é conferido entre linhas, e o servidor havia parado de
  escrever. `ReadLineAsync` com `Wait` limitado, como já registrado para o harness
  de órfãos.

**Não existe flag de CLI para o teto do cache de trecho.** O braço desligado
exigiu um segundo binário, compilado com `SnippetCacheEntries` apontando para
zero em `cmd/gobsidian/servico.go`; a edição foi aplicada e revertida em bytes
crus, com SHA-256 conferindo o restauro, e o arquivo versionado está intacto.

**Como se sabe que o cache estava cheio.** A chave é
`{caminho, hash, início, fim, maxChars}`, e dois pares `(caminho, texto)`
distintos só podem vir de chaves distintas — logo o número de pares distintos nas
respostas é um limite inferior das entradas postas. A carga produziu **4.574
pares distintos** em todas as doze partidas, contra um teto de 1.024. Como
`Len = min(chaves distintas, teto)`, o cache estava no teto.

| Partida | A: padrão (1.024) | B: desligado (0) |
|---|---|---|
| 1 | 64,49 MB | 104,62 MB |
| 2 | 64,45 MB | 62,80 MB |
| 3 | 63,17 MB | 64,25 MB |
| 4 | 63,68 MB | 64,01 MB |
| 5 | 64,16 MB | 62,88 MB |
| 6 | 65,20 MB | 62,45 MB |
| **mediana** | **64,31 MB** | **63,45 MB** |

**A diferença não é significativa.** U de Mann-Whitney com n=6 por braço dá
U = 11, contra região crítica U ≤ 5 para α = 0,05 bicaudal: `~`. O ponto estimado
é **+0,86 MB** para o braço com cache (+1,43 MB descartando a partida 1 de B), o
que é compatível com a conta de ~1,2 MB do comentário de
`DefaultSnippetCacheEntries` e não a contraria. **O cache custa cerca de 1 MB e
esse 1 MB está abaixo da resolução deste instrumento**; dizer mais que isso seria
escrever número que a medição não sustenta.

A partida 1 do braço B deu 104,62 MB, e **o mesmo aconteceu numa bateria
anterior de três partidas** (104,09 MB, também na primeira do braço B) — é a
primeira execução daquele binário em cada bateria. A causa não foi identificada.
Fica registrada como saiu; é por isso que a comparação usa mediana.

O `alloc` do runtime, colhido por `vault_stats` com `include_runtime`, ficou em
21,84 MB (mediana) no braço A contra 22,62 MB no B — **na direção oposta**, o que
é a mesma conclusão dita de outro jeito: a variação entre partidas é maior que o
efeito procurado. `alloc` não é colhido depois de um GC forçado, então ele também
mede quando o coletor passou.

**Nenhum dos dois braços ameaça o RNF-07, e o padrão fica em 1.024.** Vale
registrar o que a tabela mostra e a de RNF-07 não: sob esta carga os dois braços
ficam em 62–65 MB, acima do alvo de 60 MB — **mas RNF-07 cobra RSS em repouso**, e
o número de repouso continua sendo 37,95–38,10 MB. O que empurra o RSS para além
do alvo aqui é a carga de 6.600 resultados, presente igualmente nos dois braços,
não o cache. Medir o repouso sob carga seria responder outra pergunta; medir o
custo do cache sem carga mediria zero.

### Achado fora do escopo: um placeholder do OneDrive derrubava a indexação

Ao montar o teste que prova que o cache não abre arquivo somente-nuvem, o
`index.Build` entrou em pânico:

```
panic: runtime error: invalid memory address or nil pointer dereference
github.com/jonyd/gobsidian/internal/index.(*Index).insert(...)
	internal/index/index.go:111
github.com/jonyd/gobsidian/internal/index.(*Index).Build.func4()
	internal/index/build.go:107
```

`build.go` manda placeholder de nuvem ao coletor com `parsed{entry: e}` e `note`
**nil**, de propósito — ler o arquivo dispararia download síncrono. `insert`
entrava no ramo `r.entry.IsNote` (verdadeiro: é `.md`) e desreferenciava
`r.note.Title`. **Um único `.md` não hidratado do OneDrive derrubava a indexação
inteira no boot**, e cofre em OneDrive é cenário suportado (`docs/WINDOWS.md`).

Sobreviveu porque nenhum teste montava a condição: o único que tentava usa
`FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS`, que não é gravável, e está pulado.
`FILE_ATTRIBUTE_OFFLINE` é gravável por `SetFileAttributes` e `vault.IsCloudOnly`
também o aceita — e é por isso que `TestBuildNotaSomenteNuvemNaoDerrubaIndexacao`
consegue existir.

**Divergência relacionada, corrigida pela Task 95 em 2026-08-12:** `Build`
registrava o placeholder como `Note` com `CloudOnly = true` e `index.Replace` o
registrava como `Asset`, com `idx.Get` devolvendo `false`. As duas construções do
índice respondiam diferente para a mesma entrada — mesma família da divergência de
chave do `byAlias`, que já custou um link resolvendo para nota deletada. Detalhe
na seção seguinte.

### Uma classificação só, e o que ela mudou nas tools (Task 95, 2026-08-12)

`Note` com `CloudOnly = true` é o comportamento certo — um `.md` é uma nota, e
não ser lido é outra coisa, que é o que o comentário de `Note.CloudOnly` já
prometia. Quem estava errado era `Replace`.

Consertar só `Replace` deixaria a próxima divergência nascer no mesmo lugar. A
decisão que vira o que uma entrada é — Asset, Note lida, ou Note registrada sem
abrir o arquivo — saiu para `classificar`, em `internal/index/classify.go`, e as
**duas** construções passam por ela: `Build` (pelo worker de leitura e por
`insert`) e `Replace`. É a disciplina de `aliasKey` e de `nomeChave`: não é para
consertar o errado, é para tornar a próxima divergência impossível sem tocar
naquela função. Junto saiu a derivação de `IsNote`, que `Replace` fazia por
sufixo `".md"` enquanto `vault.Walk` usava `vault.Classify` — duas cópias da
mesma regra, que discordavam em ruído de editor (`~$nota.md`).

O teste que sustenta isso compara as **duas construções campo a campo**, por
reflexão sobre `index.Note` e não por lista escrita à mão, de modo que campo
novo entra na comparação sozinho. Não afirma valores: afirma igualdade. Um teste
que conferisse valores em cada lado passaria com as duas erradas do mesmo jeito.

**Mudança de comportamento das tools.** Antes, um evento do watcher sobre um
`.md` somente-nuvem **rebaixava** a nota que o boot tinha indexado: ela virava
`Asset` e sumia de `idx.Get`. O que muda para quem chama:

| Tool | Antes do evento do watcher | Depois do evento, antes da Task 95 | Depois da Task 95 |
|---|---|---|---|
| `note_read` | `CLOUD_ONLY_FILE` | `NOTE_NOT_FOUND` | `CLOUD_ONLY_FILE` |
| `note_metadata` | metadados, sem título | `NOTE_NOT_FOUND` | metadados, sem título |
| `note_list` | presente | ausente | presente |
| `link_graph` | nó presente | nó ausente | nó presente |
| `[[nuvem]]` | resolve para a nota | **link quebrado** (`target_missing`) | resolve para a nota |
| `vault_stats` | conta em `notes` | conta em `assets` | conta em `notes` |

O `NOTE_NOT_FOUND` era falso: o arquivo existe. O link quebrado apareceu na
própria prova de mutação — `comum.md` citando `[[nuvem]]` resolvia por `Build` e
ficava `target_missing` depois de `Replace`.

**Nenhuma tool passa a abrir o arquivo.** `note_read` sai em `CLOUD_ONLY_FILE`
antes de qualquer leitura, `note_metadata` e `note_list` respondem do índice em
memória, e `GenerateSnippet` já tinha a saída antecipada. Os testes seguram um
handle exclusivo sobre o placeholder e conferem que o handle barra uma leitura
**antes** de afirmar que nada foi aberto — a prova é mecânica, não inferida da
ausência de conteúdo.

**Achado que a Task 95 levantou e a Task 97 fechou:** `search.Inverted.Update`
chamava `os.ReadFile` sem consultar `vault.IsCloudOnly`. Ver a seção seguinte —
e ler ali por que "pré-existente" era o enquadramento errado.

### O caminho que a correção do pânico tornou alcançável (Task 97, 2026-08-12)

`Inverted.Update` abria o arquivo sem consultar `vault.IsCloudOnly`, e ela é
chamada para **toda nota do cofre** por `buildInvertedIndex` no boot (via
`idx.NotePaths()`), pelo watcher em `Apply` e pela reconciliação por overflow.

A tentação é registrar isso como defeito antigo, e o registro estaria errado:
**antes da correção do pânico desta mesma sessão, o caminho era inalcançável.**
Um único `.md` não hidratado derrubava `index.Build`, e o processo morria antes
de chegar a `NotePaths`. Consertar o pânico — que era um P0 — tornou o boot
possível e, com ele, tornou alcançável um download síncrono de **todo**
placeholder do cofre, em segundo plano, na primeira partida. Trocar um crash por
uma violação da regra "placeholder de nuvem nunca é aberto" não é conserto
completo; é a mesma armadilha já paga por este projeto, do outro lado. **Quem
roda depois do guarda precisa do mesmo guarda.**

**A guarda mora em `Inverted.Update`, não nos chamadores.** É a função que abre o
arquivo; são três pontos de chamada hoje, e guarda em chamador é a próxima
divergência esperando acontecer — a mesma lição de `aliasKey`, de `nomeChave` e
da `classificar` da Task 95.

**A nota entra coberta e vazia, não pulada.** `Add(path, nil)` registra
`docLengths` com zero, sem postings, sem abrir o arquivo. Um `return` seco
deixaria o caminho fora de `docLengths`, logo fora de `DocCount`, logo o
cabeçalho do cache declararia menos notas do que o índice de metadados enxerga,
logo `invertedCacheState` concluiria "cache parcial" em **todo** boot e
regravaria o cache inteiro. É a armadilha que a nota sem token nenhum já custou —
4 notas vazias em 3.152 custavam uma reconstrução e uma regravação por partida —
e é por isso que `HasDoc` existe separada de `DocLength`: o laço de retomada de
`buildInvertedIndex` conta por `HasDoc`, e uma nota de tamanho zero seria relida
para sempre sem nunca passar a contar como coberta.

Quatro testes, em `internal/search/cloudonly_update_windows_test.go`:

- **Não abre o arquivo**, por prova mecânica: um handle exclusivo
  (`GENERIC_READ|GENERIC_WRITE`) sobre o placeholder, com a guarda que confere
  que ele barra um `os.ReadFile` **antes** de qualquer asserção depender disso.
- **Entra coberta**: `HasDoc` verdadeiro, `DocLength` zero, e `Postings` vazio
  para termos que só existem no conteúdo do disco.
- **O boot não declara cache parcial**: compara `DocCount` do índice invertido
  com `NoteCount` do índice de metadados, e depois `hdr.NoteCount` contra
  `idx.NoteCount()` depois de uma ida ao disco — que é literalmente o que
  `invertedCacheState` confronta. Conferir `DocLength == 0` não pega essa
  regressão; comparar as duas contagens que o boot compara, pega.
- **Ida e volta pelo cache**: índice recém-construído e índice recarregado
  respondem igual, campo a campo, num cofre com placeholder.

## Teto de concorrência reapertado e custo da guarda de nuvem medido (2026-08-13)

Duas lacunas que o lote anterior deixou escritas como abertas.

### O teto de 60 ms tinha deixado de conseguir falhar

`TestRNF04SnippetConcurrencyLimit200` existe para detectar uma coisa só: que o
recorte concorrente de trechos **está ativo**. O teto de 60 ms foi escolhido
quando o caminho sequencial media 82–113 ms.

Depois da troca `Postings` → `Positions`, o sequencial mede outra coisa. Medido
com `maxSnippetWorkers = 1`, cinco execuções, sem `-race`:

```
32,36  27,60  30,14  31,56  29,65 ms
```

**Desligar a concorrência passava num teto de 60 ms.** O teste continuava verde
afirmando algo que ele não verificava mais — e ninguém tinha como notar, porque
teste que passa não chama atenção. O código ficou rápido e o teto ficou parado.

A banda foi medida dos dois lados antes de escolher o número:

| | faixa | execuções |
|---|---|---|
| concorrente | 8,39 – 12,73 ms | 12 limpas (pior já visto na sessão: 17,87) |
| sequencial | 27,60 – 32,36 ms | 5, com `maxSnippetWorkers = 1` |

**Teto novo: 22 ms** — 1,2× acima do pior concorrente já observado e 20% abaixo
do **melhor** sequencial. Mais alto e o sequencial começa a caber; mais baixo e
o teto passa a cobrar ruído. Quem absorve carga transitória continua sendo a
repetição de três rodadas, não a altura do teto.

Verificado por mutação, e a prova exigiu `-NoRace`:

```
      - const maxSnippetWorkers = 8
      + const maxSnippetWorkers = 1
--- FAIL: TestRNF04SnippetConcurrencyLimit200 (3.04s)
    rodada 1/3 estourou (32.9465ms > 22ms); repetindo
    rodada 2/3 estourou (28.9213ms > 22ms); repetindo
    p95 de limit: 200 = 27.5913ms excede o teto de 22ms em 3 rodadas seguidas
MUT_TETO_EXIT=0
```

**Com `-race` a mutação sai `EXIT=1`, e isso não é falha do teto:** o teste tem
`if raceEnabled || p95 <= teto`, então sob o detector ele passa sempre — a
latência com `-race` é 2 a 6× maior e o número deixaria de ser comparável.
`scripts/mutate.ps1 -NoRace` existe para este caso, e o comentário dele já
citava a Task 72 pelo mesmo motivo.

### Custo da guarda de placeholder de nuvem em `Inverted.Update`

A Task 97 pôs `vault.IsCloudOnly(abs)` antes do `os.ReadFile`, e o custo ficou
no ledger como **não medido** porque nenhum benchmark cobria `Update` em lote.
Entrou `BenchmarkInvertedUpdateLote`, que percorre `index.NotePaths()` chamando
`Update` — o mesmo laço de `buildInvertedIndex`.

A/B com a guarda desligada por curto-circuito (`false && vault.IsCloudOnly(abs)`,
que o compilador não chega a chamar), `benchstat`, n=8 por braço, cofre de 5.000
notas:

```
                      sem guarda      com guarda      delta
sec/op                2,071 ± 12%     2,305 ±  4%     +11,27% (p=0,038 n=8)
B/op                108,3Mi ±  0%   109,1Mi ±  0%      +0,68% (p=0,000 n=8)
allocs/op            542,4k ±  0%    547,4k ±  0%      +0,92% (p=0,000 n=8)
```

**+234 ms sobre 5.000 notas, ou ~47 µs por nota.** O `p=0,038` é marginal e o
braço sem guarda tem ±12%, mas as alocações não deixam dúvida sobre o mecanismo:
**+5,0 mil allocs, exatamente uma por nota** — a conversão UTF-16 do caminho que
`vault.IsCloudOnly` faz antes do `GetFileAttributes`.

**Nada a fazer, e a decisão é essa por medição.** São 234 ms uma vez por cofre,
numa construção em segundo plano que já leva 2,3 s e cujo resultado é cacheado;
as outras onze tools respondem desde o primeiro segundo. A alternativa óbvia —
passar o `CloudOnly` que `vault.Walk` já calculou, em vez de reconsultar — moveria
a guarda para os três chamadores, que é exatamente o que a Task 97 recusou. Um
ganho de 234 ms por cofre não paga trocar guarda única por três cópias.

### O gate de benchmark decidia por uma amostra (2026-08-13)

Registrado porque errei duas vezes antes de achar a causa, e as duas correções
erradas eram plausíveis.

`docs/bench-baseline.json` estava dez vezes acima do real depois da troca
`Postings` → `Positions`. Regenerei da **mediana de 3 rodadas**, como a nota
antiga do arquivo mandava. **O gate reprovou no push seguinte, sem mudança de
código:** `SearchDoisTermos` 3.568.100 → 5.192.208 ns/op, +45,5%.

Medi o espalhamento de quatro rodadas do mesmo código e regravei a referência no
**pior caso**, não na mediana:

| benchmark | min | max | espalhamento |
|---|---|---|---|
| `SearchDoisTermos` | 3.446.682 | 5.192.208 | **50,6%** |
| `SearchLimit200` | 9.093.936 | 11.801.909 | **29,8%** |
| `IndexBuild` | 150.144.662 | 162.847.931 | 8,5% |

**Reprovou de novo no push seguinte:** `SearchTermoAmploCache` +39,1%, num
caminho que nenhum dos commits daquele push tocava.

Duas correções, os dois sintomas, nenhuma causa. **A causa estava acima do
limiar:** `bench.yml` tirava **uma amostra** por benchmark, e num benchmark de
poucos milissegundos o ruído de runner compartilhado é maior que qualquer
tolerância que valha a pena ter.

E o conserto óbvio sozinho teria **piorado**: `bench_compare.ps1` fazia
`$Medidos[$nome] = $valor`, então com `-count > 1` a **última** amostra vencia em
silêncio. Cinco amostras decidindo pela quinta é pior que uma, porque parece
robusto.

As duas metades:

- `bench.yml` roda `-count 5`.
- `bench_compare.ps1` usa a **mediana** das amostras, imprime quantas
  sustentaram cada uma, e **avisa abaixo de três** — gate decidido por uma
  amostra sem ninguém ver que foi uma amostra é exatamente a falha consertada.

Parser conferido com entrada sintética: amostras 100, 900, 300, 200, 500
reportam 300 de cinco amostras; arquivo de amostra única dispara o aviso.

Referência regravada da mediana de 5 (rodada 31697933573). **Três rodadas de
validação depois, todas verdes**, e a primeira delas sem nenhum aviso de melhora
— a comparação caiu dentro de ±20% nos nove benchmarks, o que nenhuma das duas
referências anteriores conseguiu.

**O que isto custa:** `-count 5` multiplicou a etapa por ~5 (de ~45 s para
~3 min). É o preço de o gate significar alguma coisa. Gate que reprova
aleatoriamente ensina a re-rodar até ficar verde, e aí ele para de valer.

## `vault_search` com filtro de frontmatter travava por minutos (2026-08-13)

Defeito de complexidade, não escolha de otimização. Identificado no brainstorm
de performance como item 3, e ficou aberto por três lotes porque **nenhum
benchmark cobria esse formato de consulta** — não aparecia em perfil nenhum.

`matchesSearchFilters` resolvia o filtro de frontmatter chamando
`s.index.List(q)` **dentro** do laço de filtragem. E o laço não percorre a
página: percorre **todos** os `rawHits` que o BM25 pontuou. Para a consulta
`nota` num cofre de 5.000 notas são ~5.000 hits, cada um disparando uma
varredura do índice inteiro.

Medido, `BenchmarkSearchFiltroFrontmatter`, cofre de 5.000 notas, `limit: 200`:

```
antes    192.800.034.800 ns/op   e   181.935.343.300 ns/op
depois        53.936.180 ns/op   ...      48.035.000 ns/op
```

**192,8 s e 181,9 s por chamada.** Uma única `vault_search` com filtro de
frontmatter levava mais de três minutos — 640× acima do limite **degradado** de
300 ms que o PRD define para o RNF-04, e três ordens de grandeza acima do alvo.
A terceira amostra do braço antigo foi morta por timeout de 8 minutos.

O conserto resolve o filtro **uma vez**, em `casamFrontmatter`, e passa o
conjunto para o laço.

**A distinção que o conserto introduz é onde ele poderia estragar tudo:** `nil`
significa "sem filtro de frontmatter" e conjunto **vazio** significa "filtro
existe e nada casou". Confundir os dois faz a busca devolver o cofre inteiro
exatamente quando o usuário pediu um filtro que não casa nada — e o teste do
caminho feliz passaria igual. As duas direções estão fixadas por teste e
provadas por mutação:

```
      - if len(opts.Frontmatter) == 0 { return nil }
      + if len(opts.Frontmatter) == 0 { return map[vault.CanonicalPath]bool{} }
--- FAIL: TestVaultSearchQuery — res = {Results:[] Total:0}
MUT_NIL_EXIT=0

      - if porFrontmatter != nil {
      + if len(porFrontmatter) > 0 {
--- FAIL: TestVaultSearchFrontmatterSemCasarNaoDevolveTudo
    filtro que nao casa ninguem devolveu 2 resultados
MUT_VAZIO_EXIT=0
```

A primeira tentativa de provar o caso `nil` usou `if false && ...` e saiu
`EXIT=1`. **Não era regra não verificada:** curto-circuitar aquele `if` faz
`List` receber filtro vazio e devolver tudo, o que é inócuo. Mutação que não
muda comportamento não prova nada, e `EXIT=1` é o script dizendo isso
corretamente.

**Por que ficou escondido:** `TestVaultSearchFrontmatter` existe desde sempre e
usa duas notas. Com N=2 o custo quadrático é invisível. Benchmark que não cobre
um formato de consulta é um formato de consulta sem gate.

## O lock de inicialização do daemon desligou o daemon por três dias (2026-08-13)

Diagnosticado numa máquina real, testando o servidor MCP contra um cofre de
6.037 notas e 1,37 GB dentro do OneDrive.

### O sintoma

Três processos `serve` para o mesmo cofre, um deles com **1.378 MB** de working
set, cada um com o próprio índice. O daemon — que existe desde a v1.1.0 e sai
ligado por padrão — não estava em uso.

### A cadeia, cada elo verificado

1. Config MCP limpa: `serve --vault`, sem `GOBSIDIAN_NO_DAEMON`, `env` vazio.
2. O daemon **funciona** quando lançado à mão: indexa, serve e encerra por
   ociosidade.
3. Nenhum processo de daemon aparece quando o `serve` tenta — **zero em oito
   amostragens** ao longo dos 10 s de espera.
4. O log diz `socket do daemon nao respondeu em 10s`, e **não**
   `nao foi possivel iniciar o daemon`. Ou seja: `iniciar()` devolveu nil sem
   ter sido chamado.
5. `EnsureStarted` só pula o spawn num caso — `adquiriu == false`, o ramo
   "outra ponte já está subindo, só espero".
6. `adquirirLock` usa `O_CREATE|O_EXCL`, grava o PID no arquivo e **nunca o lê
   de volta**. Não há checagem de obsolescência.
7. O lock era de **11/08 09:14**, PID **18240**, e esse PID estava **morto**. A
   primeira falha no log é de 11/08 09:16.

A única proteção prevista é o `defer liberar()`, e o comentário do próprio
código já antecipa o risco: *"um lock que sobrevive a uma tentativa fracassada
de iniciar o daemon travaria toda ponte seguinte atrás de um arquivo que ninguém
mais vai liberar."* Mas `defer` **não roda quando o processo é morto** — e matar
processos `gobsidian` é exatamente o que o instalador faz para poder substituir o
binário.

### O custo

Desde 11/08, toda sessão MCP esperava 10 s em vão e depois construía o índice
inteiro só para si. A Parte II do M7 (Tasks 88–93), com −60% e −74% de memória
medidos para três e cinco sessões, estava **desligada em produção sem ninguém
perceber** — o fallback em processo é silencioso por desenho, e o log que explica
está em `INFO` no stderr do servidor.

### O que foi feito, e o que NÃO foi

Os dois instaladores (`install.ps1` e `installer/install.js`) passaram a:

- **perguntar antes de matar**, listando PID e cofre de cada processo, e abortar
  a instalação se o usuário recusar — encerrar sessão MCP sem avisar derruba
  trabalho em curso e o usuário não tem como saber que foi o instalador;
- **remover locks obsoletos**, e **só** os que têm PID morto. Lock com processo
  vivo é uma ponte legítima subindo o daemon agora, e apagá-lo abriria a corrida
  que o lock existe para fechar. A limpeza roda mesmo quando não há processo
  algum — que era exatamente o caso desta máquina.

Verificado nos dois, com PID morto e PID vivo no mesmo diretório: o morto é
removido, o vivo é mantido com a mensagem dizendo por quê.

**~~A causa raiz continua no código Go e não foi corrigida.~~ Corrigida em
2026-08-14** (`d8ff710`, v1.2.1). `adquirirLock` passou a ler o PID que ele mesmo
grava e a recuperar o lock quando o dono está morto. O instalador continua
limpando no momento da instalação, e as duas camadas não são redundância inútil:
o instalador cobre binários antigos já espalhados, que continuam com o defeito.

Duas decisões do conserto, ambas comentadas onde vivem:

- **Conteúdo ilegível ou vazio conta como obsoleto.** O processo pode morrer
  entre o `O_EXCL` e a escrita do PID, deixando um arquivo de zero byte para
  sempre; tratá-lo como vivo devolveria o travamento permanente.
- **PID reciclado conta como vivo**, conservador de propósito: para esse caso a
  ponte espera, que é o comportamento de hoje. Degradar para o estado anterior é
  aceitável; roubar o lock de um dono legítimo não é.

**A regra do Windows quase ficou escrita sem estar verificada.** A primeira
versão do teste chamava `cmd.Wait()`, o que faz o Go fechar o handle do
processo; o PID some, `OpenProcess` falha sozinho e `pidVivo` devolvia false
pelo motivo errado. A mutação que apagava a checagem de `exitTime` **passou**. O
teste passou a adiar o `Wait` e a esperar num handle próprio, montando a
condição real — processo morto que ainda responde a `OpenProcess` — e a conferir
que ela se montou antes de afirmar.

---

## RNF-02 numa escala maior, e o que o formato 3 do cache custou (2026-08-26, Task 135)

O formato do cache de metadados subiu para 3 para carregar o `Context` de cada
link. Medir o efeito no boot era obrigatório: o RNF-02 é publicado, e o cache
cresceu 67%.

Cofre real do dono: **5.686 notas, 42.329 links, 109 MB, em OneDrive.** Medido
com `index_ms` da linha `servidor pronto` — que é o recorte que RNF-01 e RNF-02
nomeiam — em processo (`GOBSIDIAN_NO_DAEMON=1`), somente-leitura, e com
`--cache-dir` próprio, **para não gravar formato 3 no cache que as sessões vivas
do dono leem**: o binário instalado é anterior e recusaria, forçando reconstrução
em todas elas.

| | formato 2 | formato 3 |
|---|---|---|
| arquivo de cache | 19,53 MB | 32,62 MB (+67%) |
| boot quente, mediana de 5 | **891 ms** | **921 ms** |
| amostras | 810 / 871 / 891 / 930 / 1079 ms | 872 / 887 / 921 / 988 / 1034 ms |
| boot frio, n=1 | 1741 ms | 2326 ms |

**RNF-02 segue NÃO ATINGIDO, e nas duas versões.** 891 ms contra o teto de
300 ms, com o formato ANTIGO — é condição preexistente nesta escala, não regressão
desta tarefa. Está também acima do limite de falha de 1 s em uma das cinco
amostras do formato 2 (1079 ms) e em nenhuma do formato 3, o que sozinho já
mostra o tamanho do ruído.

**O delta de +30 ms na mediana não é distinguível de ruído**: as faixas se
sobrepõem quase inteiras, e a do formato 2 é a **mais larga** das duas (269 ms
contra 162 ms). Remover o contexto não devolveria o RNF-02.

**O boot frio tem uma amostra só de cada e não sustenta conclusão.** A diferença
está na direção que se esperaria — `index_ms` inclui `SaveIndexCache`, e o
formato 3 grava 13 MB a mais —, mas com n=1 isso é hipótese. As duas passam no
alvo de 3 s do RNF-01.

**Onde o tempo está, não foi medido.** `LoadIndexCache` isolado neste mesmo cofre
mede 275–282 ms, e o boot mede ~900 ms: os ~600 ms restantes estão fora do codec.
`VerifyFreshness` faz `Stat` em cada um dos 5.686 arquivos, num cofre em OneDrive,
e é o suspeito óbvio — **mas suspeito não é medida**, e nenhuma medição foi feita
para confirmá-lo. É o próximo alvo se RNF-02 continuar prioridade nesta escala.

### Correção: fora do OneDrive o formato 3 CUSTA, e cruza a linha do RNF-02

A seção acima concluiu que o delta do formato 3 "não é distinguível de ruído" e
que "remover o contexto não devolveria o RNF-02". **Isso vale para o cofre em
OneDrive e não generaliza.** O dono mandou medir num cofre fora do OneDrive, e lá
o sinal aparece limpo.

`C:\Users\jonyd\Obsidian\Jurisprudência` — **1.254 notas, 90 anexos**, disco
local. Mesmo protocolo. Duas bateladas de cada formato, agrupadas: **n=13 cada**,
alternando a ordem para que o cache de arquivo do SO não favoreça um dos lados.

| | formato 2 | formato 3 |
|---|---|---|
| arquivo de cache | 9,76 MB | **19,05 MB (+95%)** |
| boot quente, mediana de 13 | **243 ms** | **323 ms** |
| faixa | 214–433 ms | 284–415 ms |
| amostras acima do teto de 300 ms | **3 de 13** | **10 de 13** |

**O formato 3 empurra o RNF-02 de atingido para não atingido neste cofre.**
Mediana 243 ms contra 323 ms, teto de 300 ms. As caudas se sobrepõem, mas a
separação é forte onde importa: **9 das 13 amostras do formato 2 ficam abaixo da
MENOR amostra do formato 3** (284 ms), e nenhuma do formato 3 desce abaixo disso.

Por que o outro cofre escondeu isto: lá o ruído é de ~270 ms de largura e o efeito
é de ~90 ms, então o efeito some dentro dele — e o RNF-02 já estava 3× estourado
pelos dois formatos, o que tornava a pergunta acadêmica. Aqui o cofre é local,
quieto, e a métrica vive **em cima da linha** — é exatamente onde 80 ms decidem.

**A ordem das bateladas importa e quase produziu um número errado.** A primeira
rodada do formato 3 tocou 1,3 GB de arquivos com o cache de arquivo do SO frio
(boot frio de 4.534 ms; a repetição com o SO quente deu 765 ms). Se eu tivesse
comparado aquela batelada contra a do formato 2, que rodou depois com tudo quente,
o custo apareceria muito maior do que é. Cada formato foi medido duas vezes, em
ordens diferentes, e as bateladas foram agrupadas.

**O boot frio continua sem conclusão:** uma amostra por batelada, e a primeira de
todas está contaminada pelo cache de arquivo do SO. 521 e 597 ms no formato 2
contra 765 ms no formato 3 (a de 4.534 ms é descartada pelo motivo acima). A
direção bate com gravar 9 MB a mais — `index_ms` inclui `SaveIndexCache` — mas com
n=1 por batelada isso é hipótese.

### Retratação: o custo de boot do contexto não sobrevive a medição intercalada

A seção anterior afirmou que o formato 3 "empurra o RNF-02 de atingido para não
atingido" no cofre local. **Está errada, e a causa foi o método, não o cofre.**

Aquelas bateladas rodaram em sequência — todas as amostras de um formato, depois
todas do outro — enquanto a máquina estava carregada pelas próprias medições. A
deriva de carga entre bateladas foi lida como diferença entre formatos. Alternar a
ORDEM das bateladas, que foi o cuidado que tomei, **não basta**: as bateladas
continuam separadas no tempo.

Refeito com três binários construídos lado a lado e **uma rodada de cada por vez**,
dez vezes, cada um com seu `--cache-dir`. Mesmo cofre local
(`Obsidian\Jurisprudência`, 1.254 notas), máquina em repouso:

| variante | cache | mediana de 10 | faixa | acima do teto de 300 ms |
|---|---|---|---|---|
| sem contexto (formato 2) | 9,76 MB | 179 ms | 163–237 ms | **0 de 10** |
| contexto de 80 (formato 3) | 19,05 MB | 193 ms | 159–231 ms | **0 de 10** |
| contexto de 40 + heading (formato 4) | 16,95 MB | 191 ms | 147–245 ms | **0 de 10** |

**Os três passam no RNF-02 neste cofre, com folga.** A diferença entre as medianas
é de 14 ms, **menor que a variação dentro de uma única variante** — a faixa do
formato 2 sozinha tem 74 ms de largura. O efeito de +80 ms relatado antes não
existe; era carga de máquina.

**O que continua verdadeiro e não depende de tempo:** o tamanho do cache. 9,76 →
19,05 → 16,95 MB são valores determinísticos, medidos, e não flutuam.

**A lição de método, que é o que sobra:** neste projeto, comparar duas variantes de
desempenho exige **intercalar as execuções**, não apenas alternar a ordem das
bateladas. Qualquer deriva de máquina — cache de arquivo do SO aquecendo, outro
processo, throttling térmico — atinge bateladas sequenciais de forma desigual e
aparece como sinal. Foi o segundo número errado que este método produziu na mesma
sessão; o primeiro foi corrigido por acaso, ao trocar de cofre.

**Decisão do dono, com o número na mão (2026-08-26):** ficar com `contextoBytes =
80`, mantendo o `heading`. Formato do cache 4 → 5. Os 2,1 MB de diferença para o
corte de 40 compram o dobro de contexto e não custam tempo mensurável — as três
variantes ficaram dentro de 14 ms uma da outra, com zero amostras acima do teto.
O corte para 40 tinha sido decidido sobre a medição retratada acima.

---

## Achados de desempenho de 2026-08-27, medidos e intercalados

Todos com binários construídos lado a lado e **uma rodada de cada por vez** —
bateladas sequenciais mediram deriva de máquina duas vezes nesta base, e a lição
está em [`ARMADILHAS.md`](ARMADILHAS.md).

| Achado | O que mudou | Medida | Veredito |
|---|---|---|---|
| **P6** | `TotalSize` memorizado contra `generation`, em vez de varrer os dois mapas sob lock a cada chamada | **47.155 → 68,55 ns/op** (n=2000 por rodada) | **688×** |
| **P8** | `registrarCitantesLocked` sem `slices.Contains` na lista de citantes do alvo | 4,905 → 4,865 s no Build de 8.000 notas com alvo-hub, n=5 intercalado | **−0,8%, dentro do ruído** |
| **P10 + P13** | Análise dos termos do trecho uma vez, e não por hit; `index.Get` uma vez por resultado, e não duas | 22,28 → 22,10 ms de mediana em `BenchmarkSearchLimit200`, n=8 intercalado | **dentro do ruído** |
| **P14** | Varredura lê os atributos de nuvem do `fs.FileInfo` que já tem, em vez de um `GetFileAttributes` por entrada | **não medido em relógio** | trabalho removido é certo pelo código |

**P8 e P10/P13 removem trabalho que existia, e o número não os sustenta como
otimização.** Ficam pela forma — o laço quadrático era real, e o tipo
`TermosDeTrecho` impede um chamador de perder a expansão por forma reduzida em
silêncio —, não pela velocidade. Escrever "otimizamos a busca" aqui seria a
mesma classe de mentira que esta base já corrigiu duas vezes nesta sessão.

**P14 não foi medido em relógio**, e por isso não há número. O que é certo pelo
código é o trabalho removido: um `GetFileAttributes` por entrada da varredura. O
efeito onde importa — cofre em OneDrive, onde o syscall é caro — não foi medido.

### Dois achados REJEITADOS depois de verificados

**M1** estava prescrito ao contrário. Ele pedia que `note_delete` adotasse o
critério de `note_move`, que reporta só âncoras **já** ausentes. No delete a nota
some, então toda referência ancorada quebra — inclusive as que apontam para
heading existente. `TOOLS.md` diz que a tool lista o que "passará a ter" links
quebrados. Aplicar M1 esconderia exatamente as âncoras que quebram por causa do
delete.

**P11** dizia que a varredura de temporários pulava, em silêncio, subárvores além
de MAX_PATH por falta do prefixo `\?\`. Sondado: **o pacote `os` do Go aplica o
prefixo sozinho** (`fixLongPath`) — `MkdirAll`, `WriteFile` e `WalkDir`
alcançaram 318 caracteres sem ele. A correção chegou a ser escrita e a prova de
mutação a reprovou. O que P11 acertou, e ficou: a varredura **descartava erro de
subárvore em silêncio**, então "varri e não achei nada" e "não consegui entrar em
trinta diretórios" davam a mesma resposta. `SweepResult` passou a carregar as
três contagens, e o boot as loga.

---

## RNF-07 redefinido, e medido pelo instrumento novo (2026-08-28)

A definição anterior — `RSS em repouso ≤ 60 MB` — foi substituída. O que motivou
está na seção seguinte, que é a investigação; esta é a medição sob a regra nova.

**Alvo: `≤ 8 MB + 32 KB × notas` de heap vivo, no estado `servindo`.** A
redação normativa, com os três motivos da troca, está em `docs/PRD.md` §6.1.

Medido por `pwsh -File scripts/measure.ps1 -Vault <x>`, que passou a medir isto:

| Cofre | notas | `pronto` | `servindo` | teto | folga | RSS `servindo` |
|---|---|---|---|---|---|---|
| Oral | 78 | 5 MB | 7 MB | 10,4 MB | 33% | 31,6 MB |
| Revisão | 1.275 | 9 MB | 17 MB | 47,8 MB | 64% | 46,7 MB |
| Jurisprudência | 1.254 | 32 MB | 40 MB | 47,2 MB | **15%** | 96,3 MB |
| Estudo | 2.557 | 16 MB | 58 MB | 87,9 MB | 34% | 161,6 MB |
| TJSP 192 | 5.686 | 53 MB | 126 MB | 185,7 MB | 32% | 292,8 MB |

**Os cinco passam.** A folga mais apertada é a de Jurisprudência, 15%, e ela é
explicada: o cofre é denso em links e o `Context` do backlink acrescenta ali ~6 MB
de heap vivo.

**O RSS continua na saída como diagnóstico**, e a coluna acima mostra por que ele
foi despromovido: ele é 2,3× a 2,8× o heap vivo, e a razão não é constante.

### Duas correções no próprio instrumento

**`measure.ps1` media a ponte, não o servidor.** Sem `GOBSIDIAN_NO_DAEMON`, o
`serve` tenta o daemon; quando havia um, o processo medido era uma **ponte de
~15 MB**, não o processo que segura o índice — e a ponte nem imprime "servidor
pronto". O número dependia de haver ou não daemon vivo naquele instante. O script
passou a forçar o modo em processo.

**Ele media um estado só.** Emitia `initialize` + `vault_stats` e parava, o que
desde a carga preguiçosa descreve um servidor que nunca buscou. Agora faz **duas
partidas** — e são partidas, não dois momentos da mesma, porque separar linhas de
`gctrace` por timestamp atribuiria o ciclo errado ao estado errado.

**E confere `index_origin`.** Uma medição publicada saiu de uma rodada com
`build`: 57,1 MB onde o valor com cache era 49,5. O script agora avisa quando a
origem não é `cache`.

---

## A investigação que levou à redefinição (2026-08-27)

O achado da auditoria dizia que `scripts/measure.ps1` emite `initialize` +
`vault_stats` e mede — e que, desde a carga preguiçosa da Task 88, uma sessão que
nunca buscou **nunca carregou o índice invertido**. O número publicado descreve um
servidor que ainda não buscou. A auditoria mediu **dois** cofres; isto mede os
cinco, com o binário atual.

Protocolo, com uma variável só entre os braços:

- **A** — `initialize` + `vault_stats`. É o que `measure.ps1` faz hoje.
- **B** — A, mais **uma** `vault_search`, que é o que dispara a carga preguiçosa.

Em processo (`GOBSIDIAN_NO_DAEMON=1`), somente-leitura, `--cache-dir` próprio por
cofre — para não encostar nas sessões vivas do dono nem gravar cache de formato 5
onde o binário instalado lê. **Os dois caches quentes**: metadados e invertido.
RSS é o **pico** de seis amostras após 3 s de acomodação; `alloc` é o campo do
`vault_stats`. **`alloc` NÃO é heap vivo** — é `HeapAlloc`, que inclui lixo ainda
não coletado; a seção sobre o `Context` mostra por que isso importa. Toda rodada
foi conferida com `index_origin=cache`: uma medição anterior de Estudo saiu com
`build` e publicou 57,1 MB onde o valor com cache é 49,5.

| Cofre | notas | A: RSS | A: `alloc` | B: RSS | B: `alloc` | busca |
|---|---|---|---|---|---|---|
| Oral | 78 | 24,8 MB | 5,3 MB | 30,9 MB | 8,4 MB | 65 hits |
| Revisão | 1.275 | 35,4 MB | 10,0 MB | 45,5 MB | 18,2 MB | 512 |
| Jurisprudência | 1.254 | **78,3 MB** | 59,6 MB | **91,3 MB** | 41,3 MB | 401 |
| Estudo | 2.557 | 49,5 MB | 27,6 MB | **151,0 MB** | 60,4 MB | 1.426 |
| TJSP 192 | 5.686 | **130,1 MB** | 87,7 MB | **275,6 MB** | 129,5 MB | 2.733 |

Alvo: **≤ 60 MB**. Limite de falha: **150 MB**.

**Pelo protocolo publicado (A), o RNF-07 já falha em 2 dos 5 cofres reais** —
Jurisprudência a 78,5 MB e TJSP 192 a 129,3 MB, sem que nenhuma busca tenha
acontecido. Os 37,95 MB publicados são de um cofre **sintético** de 5.000 notas;
o cofre real de 5.686 notas dá 129,3 MB pelo mesmo protocolo.

**Pelo protocolo B — o estado de qualquer sessão que já buscou uma vez — falha em
3 dos 5, e dois deles estouram o LIMITE DE FALHA**: Estudo a 151,5 MB e TJSP 192 a
276,2 MB, este último **1,8× o limite** e 4,6× o alvo.

### O custo a frio, que não é RNF-07 mas assusta mais

TJSP 192 com os dois caches vazios: **pico de 1.467,7 MB** durante a construção,
gravando 32,6 MB de cache de metadados e **564,6 MB** de cache invertido. Não é
repouso e não é o que o RNF-07 nomeia — mas é memória que a máquina do dono
precisa ter no primeiro boot depois de toda troca de formato de cache.

### O que o `heap` acrescenta, e por que ele foi medido

Em Jurisprudência o heap vivo **cai** de 59,7 para 41,3 MB entre A e B, enquanto o
RSS **sobe** de 78,5 para 90,7. Não é contradição: **RSS não é heap vivo**. O GC
rodou entre as duas amostras. Isso importa porque mede o tamanho do ruído: em
comparações onde a diferença de dados é de poucos MB, o momento do GC decide o
número.

### O `Context` do backlink custa RSS, e o custo não é uniforme

O campo acrescentado em 2026-08-26 mora no índice de metadados, que é exatamente o
que o protocolo A mede. Medido com dois binários — o atual e um com
`Context: ""` —, cinco execuções intercaladas de cada, protocolo A:

| Cofre | com contexto | sem contexto | delta |
|---|---|---|---|
| Jurisprudência | **78,4 MB** [78,2–78,8] | **56,8 MB** [52,5–57,1] | **+21,6 MB** |
| Estudo | 49,0 MB [48,6–49,2] | 52,6 MB [52,3–53,5] | **−3,6 MB** |

**As faixas não se sobrepõem em nenhum dos dois, e os sinais são opostos.**

Em Jurisprudência o efeito é grande e inequívoco: **é o `Context` que leva o cofre
de 56,8 MB, dentro do alvo, para 78,4 MB, fora dele.** Faz sentido pela forma do
cofre — o cache de metadados dele vai de 9,8 para 19,1 MB com o campo, e ele é
denso em links.

Em Estudo o cache cresce só 2,8 MB e o resultado é **negativo e reproduzível**.
Isso foi investigado com `GODEBUG=gctrace=1,scavtrace=1`, e a explicação está
abaixo — **não é dado, é o alvo de heap do GC**.

### Por que o cofre SEM contexto consome mais RSS em Estudo

Traces de GC, protocolo A, os dois binários no mesmo cofre:

| | ciclos de GC | último ciclo | heap **vivo** | **meta de heap** | RSS |
|---|---|---|---|---|---|
| **Estudo**, com contexto | 3 | `30->31->15 MB` | 15 MB | **31 MB** | 49,2 MB |
| **Estudo**, sem contexto | 4 | `32->32->16 MB` | 16 MB | **33 MB** | 52,7 MB |
| **Jurisprudência**, com contexto | 2 | `38->39->32 MB` | **32 MB** | **39 MB** | 78,4 MB |
| **Jurisprudência**, sem contexto | 3 | `31->32->26 MB` | **26 MB** | **33 MB** | 56,9 MB |

**RSS acompanha a META de heap, não o volume de dados.** O Go fixa a meta ao fim
de cada ciclo em ~2× o heap vivo daquele instante (GOGC=100). Então o RSS de um
processo em repouso é, aproximadamente, a meta que vigorava quando o último ciclo
terminou.

Em **Jurisprudência** o contexto acrescenta ~6 MB de heap vivo (32 contra 26). Isso
é grande o bastante para mandar na meta — 39 contra 33 — e o RSS segue, com
amplificação: +21,5 MB.

Em **Estudo** o contexto acrescenta pouco demais para atravessar a granularidade de
*qual* ciclo foi o último. Os dois terminaram com heap vivo praticamente igual (15
contra 16 MB), e o braço **sem** contexto rodou um ciclo a mais, disparado mais
tarde (`@0.275s`, já depois da chamada de `vault_stats`) e a partir de um heap vivo
marginalmente maior — fixando meta de **33 MB** contra os 31 MB do outro. O RSS
seguiu a meta, e o sinal saiu invertido.

**O binário com contexto roda sistematicamente UM ciclo de GC a menos** — 3 contra
4 em Estudo, 2 contra 3 em Jurisprudência — que é precisamente o que um heap vivo
maior produz: meta maior, menos ciclos para a mesma alocação.

**Consequência para o requisito:** o RNF-07 é especificado contra RSS, e RSS aqui é
um artefato do alvo de GC. Uma inversão de 3,6 MB por causa de qual ciclo terminou
por último mostra que RSS é instrumento ruidoso para um orçamento de 60 MB. Isso
reforça o caso de re-especificar o RNF-07, e não só o número dele.

**O que continua sem número:** quanto de heap **vivo** o `Context` custa em Estudo.
Sessenta chamadas seguidas de `vault_stats` produziram só 4–5 ciclos de GC, então
`alloc` nunca convergiu para o heap vivo — ele continua sendo heap vivo mais lixo
do momento. Fechar isso exigiria um `runtime.GC()` seguido de `ReadMemStats`, que o
servidor não expõe. **A leitura anterior de "heap vivo 9 MB maior sem contexto" era
esse artefato, e está retratada.**

### O que isto pede de decisão

O RNF-07 não pode continuar publicado como **Atingido**. A tabela acima já foi
corrigida. O que falta decidir é o alvo:

1. **Re-negociar o número** com base no cofre real, e nomear o estado medido
   ("depois da primeira busca"), que é o único que descreve uma sessão de verdade.
2. **Manter 60 MB e tratar como dívida aberta**, com o caminho de redução nomeado —
   o índice invertido é o que domina o protocolo B, e os IDs densos da Oportunidade 1
   atacam exatamente a estrutura que o infla.
3. **Reduzir `contextoBytes`**, que devolve ~21 MB em cofre denso em links e não
   custa tempo mensurável — mas não resolve o problema maior, que é o invertido.

---

## Oportunidade 1 implementada: BM25 em IDs densos (2026-08-28)

Era "o maior ganho de desempenho pendente" da auditoria, e estava congelada até
existir um perfil do caminho BM25. O perfil foi feito, e ele mudou a decisão.

### O perfil que destravou

`BenchmarkSearchLimit200Cache`, filtrado na subárvore de `service.Search`:

| | CPU | alocação |
|---|---|---|
| syscall de arquivo (`cgocall`) | 81,5% | — |
| **`CalculateBM25`** | 16,2% | **79%** (107,9 de 137,1 MB) |

Dentro do `CalculateBM25`: `index.Paths` + `DocLength` 33% (P1), operações de
mapa 31% (P3), `getFieldWeight` 12% (P2).

**O perfil anterior punha o BM25 em 0,73%** e por isso ninguém mexia nele — mas
era anterior à troca de `Postings` por `Positions`, que cortou 87% do tempo
total. A fração mudou e ninguém remediu.

**Em CPU o teto é 16%.** O argumento da Oportunidade 1 é **alocação**: o BM25
respondia por 79% dos bytes da busca.

### O resultado, medido intercalado

Sete rodadas de cada braço, binários construídos lado a lado:

| | tempo | alocação |
|---|---|---|
| `SearchLimit200Cache` | 16,91 → **11,67 ms** (−31,0%) | 3,78 → **2,21 MB/op** (−41,6%) |
| `SearchTermoAmploCache` | 11,26 → **6,20 ms** (−45,0%) | 3,37 → **1,80 MB/op** (−46,6%) |

**É o maior ganho de desempenho da série.** P1, P2 e P3 saem juntos: a auditoria
previa que a Oportunidade 1 os subsumiria, e subsumiu.

### Paridade de ranking, verificada e não suposta

A reescrita muda a ordem de acumulação em ponto flutuante — soma os pesos de
campo por posting e multiplica pelo multiplicador uma vez, onde antes
multiplicava por ocorrência. Seis consultas contra o cofre real de 1.254 notas,
50 resultados cada, comparando o binário anterior com este:

- **ordem idêntica nas seis**;
- maior diferença absoluta de score: **3,55e-15**.

Isso é associatividade, e nada mais.

### O que ficou de fora, e por quê

O `baseSoA` já trabalha em IDs `int32` internamente; o que se perde é na volta,
porque `Postings` reconverte para string. Fechar isso de ponta a ponta mudaria a
API de `Inverted`, o que atinge trecho, serviço e testes. **O ganho medido veio
sem isso**, então a mudança maior segue disponível e não foi gasta.

---

## Fechamento dos achados restantes (2026-08-31)

Os 61 achados da auditoria estão fechados ou rejeitados. As medições dos que
tinham custo, todas intercaladas com binários construídos lado a lado:

| Achado | Medida | Veredito |
|---|---|---|
| **P9** — pré-filtro por tamanho na correlação de renames | 70,11 → **21,07 ms** e 744,5 → **247,4 KB/op** (lote de 300 notas, 1 remoção) | **−69,9% / −66,8%** |
| **P12** — `Analyze` com pré-alocação e caminho rápido ASCII | 3,46 → **3,12 ms**, 3,15 → **2,70 MB/op** | −9,7% / −14,3% |
| **P15** — varredura de temporários sobreposta à carga do índice | boot até servir: 770 → **570 ms** (TJSP 192) | **−26%** |
| **B2** — cópia das posições quando a arena está mapeada | 10,86 → **11,48 ms**, +11,1% de alocação | **+5,7%, pago de propósito** |
| **P4** — `ToLower` içado do comparador, `cmp.Compare`, sort fora do lock | `note_list` ordenada: 2,26 → **1,72 ms** (−24,1%); filtrada por tag: 806 → 765 µs (−5,0%) | **−24,1%** |
| **P7** — seção fixa escrita em blocos | 7,48 → **1,09 MB/op**, e 16,4 → 20,7 ms | **−85,4% de alocação, +25,9% de tempo** |

### P15 mediu a coisa errada na primeira tentativa

A primeira leitura disse que a sobreposição era **regressão**: `index_ms` foi de
225 para 277 ms no TJSP 192. O número estava certo e a métrica errada — a
varredura rodava **depois** da janela que o `index_ms` cobre, e passou a rodar
**dentro** dela. O que o P15 promete é o tempo até o servidor responder, e esse
caiu 26%.

Consequência para o RNF-01: `index_ms` agora inclui a disputa por disco da
varredura concorrente. O alvo é 3.000 ms e o cofre maior mede 277 ms, então a
folga cobre a mudança — mas o número deixou de ser só a carga do índice.

### B2 é a única troca que piora um número de propósito

`Postings` devolvia uma janela **para dentro da arena mapeada**, e quem chama a
itera depois de soltar o RLock, enquanto `promoverArenaSePresente` desmapeia fora
do lock. Um leitor com a janela antiga lê memória desmapeada: **falha de proteção
de página, que mata o processo sem log** — não dado errado. Isso era barrado só
pelo gate `Building()`, uma dependência entre dois arquivos que nada testava.

Os +5,7% compram a remoção disso. A busca continua ~27% mais rápida que no início
desta série, mesmo com a cópia.

### P4 e P7, medidos (2026-08-31)

Os dois estavam sem medição isolada porque não havia benchmark que os cobrisse.
Agora há: `BenchmarkListOrdenada`/`BenchmarkListPorTag` em `internal/index` e
`BenchmarkEscreveCache` em `internal/search`.

**P4 é ganho limpo.** `note_list` ordenada por tamanho sobre 3.000 notas: 2,26 →
1,72 ms, **−24,1%**, alocação inalterada — o que era esperado, porque o que saiu
do comparador era CPU (`ToLower` da mesma string a cada comparação), não memória.
A variante filtrada por tag ganha 5,0%: ali o `ToLower` por chave do mapa de tags
continua, e ele não foi hasteado porque exigiria um índice de tags em minúsculas
memorizado — estado novo para um ganho que este número não justifica.

**P7 troca CPU por memória, de propósito.** A alocação por salvamento cai
**85,4%** — 7,48 para 1,09 MB no benchmark, e no cofre de referência é a
diferença entre materializar 291 MB e escrever em blocos de 8 KB. O tempo sobe
**25,9%**.

O benchmark escreve em `io.Discard`, então ele isola a CPU: num salvamento real,
que grava 564 MB de cache invertido no cofre principal, o disco domina e a
fração de 25,9% incide sobre uma parcela menor do total. **Não foi medido em
disco real.**

**Uma segunda otimização do P7 foi tentada e descartada por medição.** A hipótese
era que o custo estivesse no `sort` dos ids, refeito na segunda passada;
gravar a ordem na primeira e apenas reproduzi-la deu **23,79 ms contra 23,65 ms**
— nenhum ganho — e alocou 1,83 MB contra 1,09. O custo é a segunda travessia do
mapa, não a ordenação. A versão simples ficou.

### As três rejeições, com o que as derrubou

**M1** estava prescrito ao contrário — mandava `note_delete` usar o critério do
`note_move`, escondendo as âncoras que quebram por causa do delete.

**P11** partia de premissa falsa: o pacote `os` do Go já aplica o prefixo de
caminho longo, e a correção foi reprovada pela própria prova de mutação.

**B15** não tem correção melhor disponível. Um rename no nível do sistema de
arquivos **é** um delete mais um create com bytes idênticos — não há sinal nessa
camada que os separe. A guarda de cardinalidade 1-para-1 já exclui o caso comum
(N arquivos iguais de um modelo), e `apply.go` diz na própria linha que renames
são processados *sem reescrever conteúdo*: uma inferência errada custa uma
entrada de índice, corrigida na próxima reindexação.
