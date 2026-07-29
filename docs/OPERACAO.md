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

`orphans`, `broken_links` e `broken_anchors` só vêm com `include_health`.

Uma versão anterior desta tabela listava `TotalNotes`, `TotalBytes`, `TotalLinks`, `TotalAliases`, `TotalTags`, `LoadTimeMs` e `CloudOnlyFiles`. **Nenhum desses campos existe.** Documentar campo que o servidor não emite é pior que não documentar: quem lê acredita, escreve o consumidor em cima, e descobre no primeiro uso. A tabela acima foi conferida campo a campo contra `service.StatsResult`.

A duração da indexação a frio não está em `vault_stats` — está no log de boot, como `index_ms`. Ver §4.

*Nota: como a v0.1 não tem um watcher de arquivos ativo, os valores refletem o estado no instante do boot.*

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

`reason=` vale `stdin-eof`, `signal` ou `parent-gone`. Um encerramento sem `reason=` nenhum significa que nenhum mecanismo disparou e o processo morreu por outro motivo — é o que o gate de órfãos verifica.

**Sinais importantes:**
- `level=ERROR`: Falha fatal num componente. Pode indicar que um arquivo está corrompido ou inacessível.
- `level=WARN`: Falhas leves (por exemplo, bloqueios de compartilhamento ou permissão de leitura rejeitada).
- No futuro, `overflow` indicará problemas com o *watcher*.

## 5. Medições do Orçamento de Performance

Medido com `scripts/measure.ps1`, que lê `index_ms` do próprio log de boot e amostra o `WorkingSet64` do processo depois do handshake MCP e de um período de acomodação. O script reporta o **maior** RSS observado, não o último: um pico mascarado por uma amostra tardia seria ficção.

```powershell
pwsh -File scripts/measure.ps1 -Vault <caminho-do-cofre>
```

### O que foi medido até agora

**Cofre pequeno, 2026-07-28.** 7 notas, 1 anexo, 180 KB. maquina de referencia, 12 núcleos, Windows 11. Três execuções.

| ID | Métrica (Alvo) | Medição |
|---|---|---|
| **RNF-01** | Indexação a frio (≤ 3 s) | 5–8 ms |
| **RNF-02** | Boot com cache válido (≤ 300 ms) | **não medido** — ver nota |
| **RNF-04** | Latência de `vault_search` p95 (≤ 100 ms) | **não medido** — ver nota |
| **RNF-07** | RSS em repouso (≤ 60 MB) | 18,9–19,3 MB |

**Isto não é a validação do orçamento completo.** Sete a cem notas não exercitam RNF-01 em escala de 5.000 notas.

**RNF-02 e RNF-04 foram retirados desta tabela em 2026-07-29 pela revisão do M3.** Traziam `15,82 ms` e `0,18 ms`, e nenhum dos dois era o que a legenda dizia:

- O de RNF-02 vinha de `persist_test.go`, cujo laço chamava `inv.Add` cem vezes com **o mesmo caminho** (`folder/note.md`). O cache continha **uma** nota, e o próprio log do teste dizia `Notes: 1` ao lado do rótulo "100 notas". Carregar um cache de uma nota não diz nada sobre boot com cache válido.
- O de RNF-04 era uma chamada única em teste unitário com índice em memória. p95 exige distribuição; um ponto não é percentil.

A medição honesta é a Task 52. Até ela fechar, o valor é **"não medido"** — que é a resposta certa e com a qual ninguém briga.
O que a medição **de fato** estabelece é que tanto a deserialização GOB quanto o motor BM25 operam com folga da ordem de grandeza em relação aos limites de RNF-02 e RNF-04.

### O que falta

Rodar contra o cofre de referência do PRD: **5.000 notas, 50 MB**. Até lá, RNF-01 e RNF-07 em escala de 5.000 notas seguem **não validados** — medidos em escala pequena, o que é diferente de medidos.

Registre o número real, a data e a máquina. Se estourar o alvo, registre assim mesmo: o valor de a tabela existir é dizer onde o produto está, não onde se gostaria que estivesse.

Uma versão anterior desta tabela trazia *"Concluído abaixo do alvo (ex: 408ms em teste local)"* e *"Sob monitoramento. Tende a ficar ~30-45 MB"*. Nenhum dos dois era medição: o primeiro é um exemplo ilustrativo, o segundo uma expectativa. Alvo não atingido e registrado é informação; alvo não medido apresentado como resultado é ficção com aparência de tabela.
