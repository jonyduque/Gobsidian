# gobsidian

> Servidor MCP de alta performance para cofres locais do Obsidian, escrito em Go.

`gobsidian` expõe um cofre (*vault*) local do Obsidian a clientes MCP — Claude Desktop, Claude Code, VS Code, qualquer host compatível — através de um binário único, sem runtime externo, sem processos órfãos e com índice em memória construído no boot.

O projeto nasceu de três frustrações concretas com os servidores MCP de Obsidian existentes:

1. **Processos zumbi.** Servidores em Node deixam `node.exe` pendurados quando o host MCP é fechado abruptamente.
2. **Lentidão em cofres grandes.** Reindexação completa a cada evento de arquivo, ou pior, *polling*.
3. **Parsing incompleto.** Wikilinks, embeds, block-ids, propriedades e campos inline do Dataview não são Markdown padrão, e parsers genéricos quebram em casos de borda.

`gobsidian` ataca os três diretamente. Ver [`docs/PRD.md`](docs/PRD.md) para o racional completo.

---

## Estado

**Pré-alfa — documentação de projeto.** Nada implementado ainda. Este repositório contém a especificação que precede a primeira linha de código.

A primeira entrega utilizável é a **v0.1**: ciclo de vida completo, parser, índice e as cinco tools de leitura, sem watcher e sem busca. Watcher, busca full-text e a superfície de escrita vêm depois, nessa ordem. Ver [`docs/PRD.md`](docs/PRD.md) §9.

Protocolo MCP alvo: `2025-11-25`.

| Documento | Conteúdo |
|---|---|
| [`docs/PRD.md`](docs/PRD.md) | Problema, objetivos, requisitos funcionais e não-funcionais, métricas, riscos, marcos |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Camadas, fluxos, modelo de dados, concorrência, ciclo de vida, decisões arquiteturais |
| [`docs/ESTRUTURA.md`](docs/ESTRUTURA.md) | Árvore de diretórios, responsabilidade de cada pacote, convenções de código |
| [`docs/TOOLS.md`](docs/TOOLS.md) | Superfície MCP: contrato de cada *tool*, schemas de entrada e saída, erros |
| [`docs/WINDOWS.md`](docs/WINDOWS.md) | Particularidades de Windows: OneDrive, MAX_PATH, casing, fsnotify, build e registro |

---

## Princípios de projeto

**Um binário, zero dependências de runtime.** Sem Node, sem Python, sem .NET. `gobsidian.exe` e nada mais. Isso elimina de saída toda a classe de falhas de ambiente.

**O cofre é a fonte da verdade, nunca o índice.** Toda escrita vai ao disco de forma atômica e completa. O índice é derivado e descartável — se corromper, reconstrói em segundos. `gobsidian` jamais mantém estado que só exista em memória.

**Nunca decidir conteúdo pelo usuário.** As *tools* de escrita preenchem estrutura (seções, blocos, frontmatter) e reescrevem links quando arquivos se movem. Não reformatam, não "melhoram", não normalizam texto que o usuário escreveu.

**Coexistência pacífica com o Obsidian aberto.** Escritas atômicas via arquivo temporário + rename, para que o Obsidian veja sempre um arquivo completo e válido, nunca um estado intermediário.

**Encerrar quando o pai encerrar.** Detecção de EOF em stdin, sinais do SO e vigilância do PID pai. Três mecanismos independentes, porque o problema do processo órfão é o mais irritante de todos.

---

## Instalação

### A partir do código-fonte

Requer Go 1.25+.

```powershell
go install github.com/jonyd/gobsidian/cmd/gobsidian@latest
```

O binário vai para `$env:GOPATH\bin\gobsidian.exe` (por padrão, `$env:USERPROFILE\go\bin\gobsidian.exe`). Confirme:

```powershell
$GobsidianPath = Join-Path $env:USERPROFILE "go\bin\gobsidian.exe"
if (Test-Path $GobsidianPath) { & $GobsidianPath version } else { Write-Warning "[!] Binario nao encontrado" }
```

### Binário pré-compilado

Baixe de *Releases* e coloque em qualquer diretório do `PATH`. Não há instalador, não há registro no sistema, não há serviço.

---

## Configuração no Claude Desktop

Edite `claude_desktop_config.json` (em `$env:APPDATA\Claude\`):

```json
{
  "mcpServers": {
    "gobsidian": {
      "command": "C:\\Users\\jonyd\\go\\bin\\gobsidian.exe",
      "args": [
        "serve",
        "--vault",
        "C:\\Users\\jonyd\\OneDrive - Minha Organizacao\\Meu Cofre\\Meu Cofre"
      ]
    }
  }
}
```

Notas importantes sobre esse JSON:

- **Barras invertidas duplicadas.** JSON exige escape. `\\` em cada separador.
- **Caminhos com espaços não precisam de aspas extras** quando passados como elemento separado do array `args`. Aspas adicionais dentro da string são um erro comum e fazem o servidor receber um caminho inválido.
- **Use caminho absoluto para o binário.** O host MCP não herda necessariamente o `PATH` do seu shell.

Para editar o arquivo com segurança em PowerShell:

```powershell
$ConfigPath = Join-Path $env:APPDATA "Claude\claude_desktop_config.json"
$Config = Get-Content $ConfigPath -Raw | ConvertFrom-Json
$Config | ConvertTo-Json -Depth 10 | Out-File $ConfigPath -Encoding UTF8
```

---

## Uso via linha de comando

`gobsidian` também funciona fora do MCP, o que é essencial para diagnóstico.

```powershell
# Servir via stdio (modo usado pelo host MCP)
gobsidian serve --vault "C:\caminho\do\cofre"

# Indexar e sair, imprimindo estatisticas
gobsidian index --vault "C:\caminho\do\cofre" --stats

# Diagnostico do ambiente: permissoes, OneDrive, MAX_PATH, casing
gobsidian doctor --vault "C:\caminho\do\cofre"

# Busca direta, sem MCP
gobsidian search --vault "C:\caminho\do\cofre" --query "usucapiao extraordinaria" --limit 10

# Inspecionar o parse de uma nota
gobsidian inspect --vault "C:\caminho\do\cofre" --note "Civil/PONTO 03.md" --json
```

`gobsidian doctor` é o primeiro comando a rodar quando algo não funciona. Ele verifica o que costuma quebrar antes de qualquer outra coisa — gobsidian inexistente, permissões, arquivos OneDrive somente-nuvem, caminhos acima de 260 caracteres, colisões de casing entre pastas.

---

## Superfície MCP (resumo)

Contratos completos em [`docs/TOOLS.md`](docs/TOOLS.md).

### Leitura

| Tool | Função |
|---|---|
| `vault_search` | Busca full-text com filtros por pasta, tag e frontmatter |
| `note_read` | Lê nota inteira, ou uma seção por heading, ou um bloco por id |
| `note_list` | Lista notas por glob, pasta, tag ou consulta de frontmatter |
| `note_metadata` | Frontmatter, tags, links de saída, backlinks, headings, blocos |
| `link_graph` | Vizinhança de links de uma nota, com profundidade configurável |
| `tag_list` | Todas as tags do cofre com contagem |
| `vault_stats` | Contagem de notas, órfãs, links quebrados, tamanho do índice |

### Escrita

| Tool | Função |
|---|---|
| `note_create` | Cria nota, com frontmatter opcional, falhando se já existir |
| `note_append` | Anexa ao fim da nota ou ao fim de uma seção específica |
| `note_patch` | Substitui o conteúdo sob um heading ou de um bloco identificado |
| `note_move` | Move ou renomeia, reescrevendo todos os wikilinks que apontam para a nota |
| `note_delete` | Remove nota, opcionalmente relatando links que ficarão quebrados |

`note_append` e `note_patch` operando por heading são deliberadamente o centro de gravidade da API de escrita. Inserção delta em posição conhecida é mais segura, mais barata e mais revisável do que reescrever o arquivo inteiro.

### Resources

Notas são expostas também como *resources* MCP sob o esquema `gobsidian://`, permitindo que o host as anexe ao contexto sem chamada de tool. O esquema é próprio de propósito: `obsidian://` já pertence ao aplicativo Obsidian e é registrado no sistema operacional.

---

## Orçamento de performance

Metas medidas em cofre de referência: 5.000 notas, 50 MB, SSD NVMe, Windows 11.

| Métrica | Alvo |
|---|---|
| Indexação a frio | ≤ 3 s |
| Boot com índice em cache | ≤ 300 ms |
| `note_read` (p95) | ≤ 15 ms |
| `vault_search` full-text (p95) | ≤ 100 ms |
| Reindexação de um arquivo alterado | ≤ 20 ms |
| RSS em repouso | ≤ 60 MB |
| CPU em repouso | < 0,5 % |
| Processos órfãos após término do host | 0 |

A última linha é requisito de bloqueio de release, não meta de esforço.

---

## Licença

MIT.
