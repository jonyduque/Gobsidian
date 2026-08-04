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
time=2026-08-03T12:41:29.427-03:00 level=INFO msg="servidor pronto" vault="C:\Cofre" read_only=false notes=3153 assets=0 index_ms=1119 search_ready=false
```

`index_ms` é a duração da construção do índice de **metadados** — é o número que RNF-01 nomeia. Ele recorta só essa etapa: não inclui o boot do runtime do Go, a leitura da configuração nem o handshake do MCP.

`search_ready` diz se a busca já funciona. Quando `false`, o índice invertido está sendo construído em segundo plano e duas outras linhas aparecem:

```
level=INFO msg="construindo indice de busca em segundo plano" notas=3153
level=INFO msg="indice de busca pronto" notas=3153 reaproveitadas_do_cache=399 duracao_ms=206267
```

Enquanto isso, `vault_search` responde `INDEX_BUILDING`; as outras onze tools funcionam normalmente.

**Até 2026-08-03 este log enganava.** `index_ms` aparecia ao lado de "servidor pronto" e parecia ser o tempo de boot, mas o servidor só anunciava as tools depois de tokenizar o cofre inteiro — 219 s a mais num cofre de 109 MB. Quem lesse `index_ms=1275` concluiria que o boot levou 1,3 s.

A do encerramento diz qual dos três mecanismos disparou:

```
time=2026-07-28T16:11:37.341-03:00 level=INFO msg="encerramento solicitado" reason=stdin-eof
```

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

### Tabela completa dos RNFs — estado no fechamento do M6 (2026-08-02)

Os 22 RNFs do PRD, cada um com **número medido** ou **"não medido"**. Não há
terceira coluna de opinião: alvo não atingido e registrado é informação; alvo
não medido apresentado como resultado é ficção com aparência de tabela.

Onde a escala importa, o número é o de **5.000 notas** — é a escala em que o
produto tem de valer, e vários destes RNFs saem abaixo da resolução do relógio
num cofre de 7 notas.

| ID | Métrica (alvo) | Medição | Estado |
|---|---|---|---|
| **RNF-01** | Indexação a frio (≤ 3 s) | 500,11 ms no cofre sintético; **1,1 s** num cofre real de 109 MB | **Atingido** |
| **RNF-02** | Boot com cache válido (≤ 300 ms) | 96,94 ms no cofre sintético; **1,02–1,13 s** num cofre real de 109 MB (2026-08-03) | **NÃO ATINGIDO** |
| **RNF-03** | `note_read` p95 (≤ 15 ms) | p95 **344,97 µs**, mediana 206,47 µs (5.000 notas) | **Atingido** |
| **RNF-04** | `vault_search` p95 (≤ 100 ms) | 500 notas: 8 de 8, 7–25 ms. 5.000 notas: 7 de 8; `limit: 200` em **181,25 ms** | **Parcial** |
| **RNF-05** | `note_list` com filtro de metadados p95 (≤ 10 ms) | p95 **533,68 µs**, mediana 249,24 µs (5.000 notas) | **Atingido** |
| **RNF-06** | Reindexação de arquivo único (≤ 20 ms) | mediana **20,35 ms**, p95 30,14 ms (5.000 notas). Degradado do PRD: 100 ms | **NÃO ATINGIDO** |
| **RNF-07** | RSS em repouso (≤ 60 MB) | 5.000 notas: **67,08 MB** com cache quente, **112,96 MB** a frio | **NÃO ATINGIDO** |
| **RNF-08** | CPU em repouso (< 0,5 %) | **não medido** | — |
| **RNF-09** | Escalabilidade linear até 20.000 notas | **não medido** (medido até 5.000) | — |
| **RNF-10** | Zero órfãos em 100 ciclos de start/kill do host | **100/100 em três cenários** — `stdin-eof`, `parent-death`, `signal` —, cada um com o `reason=` do seu mecanismo | **Atingido** |
| **RNF-11** | Zero notas corrompidas em 1.000 crashes injetados | **0 / 1.000**, com 381 temporários órfãos varridos | **Atingido** |
| **RNF-12** | Índice degradado nunca devolve resultado incorreto | **não medido**; verificado por teste (reconciliação por overflow, `internal/watcher`) | — |
| **RNF-13** | Falha de tool não derruba o servidor | **não medido**; verificado por teste (`internal/mcpsrv`, erros como resultado MCP) | — |
| **RNF-20** | Windows 10+ primeira classe; macOS 13+ e Linux suportados | **não medido**; CI roda build, vet e `go test -race` nos três | — |
| **RNF-21** | Cofres em OneDrive/Dropbox/Drive, incluindo somente-nuvem | **não medido**; verificado por teste (`vault.CloudOnly`, `internal/vault`) | — |
| **RNF-22** | Caminhos acima de 260 caracteres no Windows | **não medido**; verificado por teste (`longpath_windows_test.go`) | — |
| **RNF-23** | Nomes com acentuação e espaços | **não medido**; verificado por teste (corpus dos golden files) | — |
| **RNF-24** | Protocolo MCP fixado em `2025-11-25` com fallback | **não medido**; fixado no SDK e verificado por teste | — |
| **RNF-30** | Nenhuma requisição de rede | **não medido**; verificado por gate — `check_net.ps1` com o analisador `netcheck` em `go vet -vettool`, nos três GOOS | **Atingido** |
| **RNF-31** | Todo caminho de tool confinado ao cofre | **não medido**; verificado por teste (`validateLocal` + `Canonicalize`) | — |
| **RNF-32** | Links simbólicos para fora do cofre não são seguidos | **não medido**; verificado por teste (`TestWalkNaoSegueSymlink`, executado com privilégio) | **Atingido** |
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
