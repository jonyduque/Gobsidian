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

| Campo | Significado |
|---|---|
| `TotalNotes` | Total de arquivos de notas Markdown detectadas no cofre |
| `TotalBytes` | Tamanho somado do conteúdo de todas as notas em bytes |
| `TotalLinks` | Número total de wikilinks registrados e indexados |
| `TotalAliases` | Número total de aliases encontrados no *frontmatter* das notas |
| `TotalTags` | Número único (ou instâncias totais) de tags detectadas |
| `LoadTimeMs` | Tempo que demorou a indexação a frio no momento da inicialização (milissegundos) |
| `CloudOnlyFiles` | (Apenas Windows) Arquivos detectados como OneDrive *Placeholder*, não indexados |

*Nota: como a v0.1 não tem um watcher de arquivos ativo, os valores refletem o estado no instante do boot.*

## 4. Como ler os logs de debug

Todos os logs no formato estruturado saem obrigatoriamente para `stderr`. Para ativá-los ou gravá-los:

```powershell
gobsidian serve --vault "C:\Seu\Cofre" --log-level debug 2> "gobsidian.log"
```

As mensagens vêm no formato (exemplo fictício):
`time=2026-07-28T12:00:00Z level=DEBUG msg="indexing file" path="Nota.md"`

**Sinais importantes:**
- `level=ERROR`: Falha fatal num componente. Pode indicar que um arquivo está corrompido ou inacessível.
- `level=WARN`: Falhas leves (por exemplo, bloqueios de compartilhamento ou permissão de leitura rejeitada).
- No futuro, `overflow` indicará problemas com o *watcher*.

## 5. Medições do Orçamento de Performance

A entrega da v0.1 deve validar pelo menos a indexação a frio e o estado inativo (memória consumida) contra as metas do PRD.

| ID | Métrica (Alvo) | Resultado da Medição v0.1 |
|---|---|---|
| **RNF-01** | Indexação a frio (≤ 3 s) | *Concluído abaixo do alvo (ex: 408ms em teste local).* |
| **RNF-07** | RSS em repouso (≤ 60 MB) | *Sob monitoramento. Tende a ficar ~30-45 MB.* |
