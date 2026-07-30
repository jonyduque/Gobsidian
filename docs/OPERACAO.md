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
time=2026-07-28T16:11:37.316-03:00 level=INFO msg="servidor pronto" vault="C:\Cofre" read_only=false notes=7 assets=1 index_ms=10
```

`index_ms` é a duração da construção do índice — **é o número que RNF-01 nomeia**. Ele recorta só a indexação: não inclui o boot do runtime do Go, a leitura da configuração nem o handshake do MCP. Cronometrar o processo por fora mede as quatro coisas juntas e responde outra pergunta.

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

| ID | Métrica (Alvo) | Medição |
|---|---|---|
| **RNF-01** | Indexação a frio (≤ 3 s) | 5–8 ms (7 notas) |
| **RNF-02** | Boot com cache válido (≤ 300 ms) | **26,96 ms** (500 notas distintas, 2026-07-29, Task 52) |
| **RNF-04** | Latência de `vault_search` p95 (≤ 100 ms) | **atingido em todos os 8 formatos** (5–72 ms). **Frase exata otimizada na Task 61 (2026-07-30): p95 22,1 ms.** |
| **RNF-07** | RSS em repouso (≤ 60 MB) | 18,9–19,3 MB |

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
  | `limit: 200` (máximo do schema) | 62,8 ms | 71,9 ms | OK |

  **RNF-04 está 100% atingido em todos os formatos.** A otimização da Task 61 introduziu o método $O(1)$ `Inverted.Positions(term, path)` sobre o mapa interno `terms[term][path]`, eliminando a busca linear $O(N)$ que percorria todas as postings do termo a cada candidato de posição. O p95 da frase exata caiu de **174,2 ms para 22,1 ms** (redução de ~87%, bem abaixo do teto de 100 ms).

### O que falta

Rodar contra o cofre de referência do PRD: **5.000 notas, 50 MB**. Até lá, RNF-01 e RNF-07 em escala de 5.000 notas seguem **não validados** — medidos em escala pequena, o que é diferente de medidos.

Registre o número real, a data e a máquina. Se estourar o alvo, registre assim mesmo: o valor de a tabela existir é dizer onde o produto está, não onde se gostaria que estivesse.

Uma versão anterior desta tabela trazia *"Concluído abaixo do alvo (ex: 408ms em teste local)"* e *"Sob monitoramento. Tende a ficar ~30-45 MB"*. Nenhum dos dois era medição: o primeiro é um exemplo ilustrativo, o segundo uma expectativa. Alvo não atingido e registrado é informação; alvo não medido apresentado como resultado é ficção com aparência de tabela.
