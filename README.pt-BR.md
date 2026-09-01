<div align="center">

# 🪨 gobsidian

**Servidor MCP para cofres locais do Obsidian. Um binário Go, sem runtime, sem processo órfão.**

🌍 **Português** · [English](README.md)

[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![MCP](https://img.shields.io/badge/MCP-2025--11--25-6E56CF)](https://modelcontextprotocol.io)
[![Plataformas](https://img.shields.io/badge/plataformas-Windows%20%7C%20Linux%20%7C%20macOS-informational)](#-compatibilidade)
[![Licença](https://img.shields.io/badge/licen%C3%A7a-MIT-green)](LICENSE)

🔍 [Visão geral](#-visão-geral) ·
📦 [Instalação](#-instalação) ·
⚙️ [Configuração](#-configuração) ·
🧰 [Ferramentas](#-ferramentas-mcp) ·
💻 [CLI](#-linha-de-comando)

🧵 [Daemon](#-daemon) ·
🖥️ [Compatibilidade](#-compatibilidade) ·
📊 [Desempenho](#-desempenho) ·
🛠️ [Desenvolvimento](#-desenvolvimento) ·
📚 [Documentação](#-documentação)

</div>

---

## 🔍 Visão geral

`gobsidian` expõe um cofre local do Obsidian a qualquer host MCP — Claude Desktop, Claude Code, Gemini CLI, Antigravity, VS Code — através de um executável único que fala JSON-RPC sobre stdio.

Nasceu de três problemas concretos dos servidores MCP de Obsidian existentes:

| 🐛 Problema | ✅ O que `gobsidian` faz |
|---|---|
| Processos zumbi depois que o host fecha | Quatro mecanismos independentes de encerramento, verificados em 100 ciclos de morte abrupta cada |
| Reindexação total a cada evento de arquivo | Watcher incremental com coalescência, e índice por deslocamento de byte |
| Parsers genéricos que quebram em wikilink, embed e block-id | Parser próprio, congelado por 48 arquivos golden e conferido contra um dump real do `metadataCache` do Obsidian |

### ✨ Funcionalidades

- ⚡ **Leitura por offset.** Ler uma seção de 2 KB numa nota de 500 KB custa 2 KB de I/O, não 500 KB.
- 🎯 **A busca diz ONDE.** `vault_search` devolve `match_offset`, o deslocamento absoluto do casamento, que alimenta `note_read(offset=…)` direto — achar o termo numa nota de 255 KB e ler só os bytes em volta dele.
- 🗺️ **`note_outline` para nota convertida.** PDF, DOCX e EPUB viram nota sem heading `#` nenhum: o título é parágrafo em negrito. A tool separa os headings reais dos **candidatos** e diz qual é qual — nunca afirma estrutura que o arquivo não tem.
- 📚 **Lote com sobreposição por item.** `note_read` aceita `["a.md", {"path":"b.md","heading":"X"}]` na mesma lista: seis capítulos com seis seções numa chamada, não seis.
- 🔎 **Busca BM25** com pesos por campo, frase exata e filtros de pasta, tag, frontmatter e data.
- 🇧🇷 **Analisador para português**: acentos, *case folding* e indexação dupla — forma crua e reduzida na mesma posting list. As chaves do índice normalizam para NFC, então uma nota gravada num Mac é encontrada por um pedido do Windows.
- ✍️ **Escrita cirúrgica e atômica.** `note_append` e `note_patch` inserem por heading ou block-id; toda escrita passa por temporário + rename, para o Obsidian aberto ao lado nunca ver arquivo pela metade.
- 🔗 **`note_move` reescreve os links**, preservando alias, âncora e forma original.
- 👀 **Watcher incremental** com debounce e reconciliação por varredura quando o `fsnotify` estoura.
- 🔒 **Modo `--read-only`** que remove as tools de escrita de `ListTools` — ausentes, não apenas rejeitadas.
- 🚫 **Nenhum socket que saia da máquina**, cobrado no CI por um analisador de `go vet` nos três sistemas.

> [!NOTE]
> O cofre é a fonte da verdade; o índice é derivado e descartável. Se corromper, reconstrói em segundos.

---

## 📦 Instalação

Qualquer um dos instaladores baixa o binário da release, **confere o SHA-256 e aborta se não bater**, instala, acrescenta ao `PATH`, lê o registro de cofres do próprio Obsidian e pergunta em quais hosts registrar o servidor.

| Você está em | Use | Precisa de |
|---|---|---|
| 🪟 Windows | `install.ps1` da raiz | nada além do PowerShell |
| 🪟 🐧 🍎 Windows, Linux ou macOS | pasta `installer/` | Node.js 18+ |

**Windows, sem dependência:**

```powershell
iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/install.ps1)
```

**Multiplataforma** — mesmo instalador em Node, sem `npm install` nem `node_modules`: <!-- check-doc-refs: ignore node_modules -- diretorio do Node, citado para dizer que o instalador nao cria um -->

```powershell
iex (irm https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.ps1)
```

```bash
curl -fsSL https://raw.githubusercontent.com/jonyduque/Gobsidian/master/installer/install.bash | bash
```

<details>
<summary>⚙️ <b>Opções</b> — instalação silenciosa, host específico, versão fixa</summary>

<br>

`iex` executa o **texto** do script, e `param()` não recebe nada por essa via. Para passar opções, use um scriptblock:

```powershell
& ([scriptblock]::Create((irm .../install.ps1))) -Vault "C:\Meu Cofre" -Hosts claude-desktop -ReadOnly
& ([scriptblock]::Create((irm .../installer/install.ps1))) --vault "C:\Meu Cofre" --hosts claude-desktop
```

```bash
curl -fsSL .../installer/install.bash | bash -s -- --vault "/caminho/do/cofre" --yes
```

| Raiz | `installer/` | Efeito |
|---|---|---|
| `-Vault` | `--vault` | Cofre a servir. Repetível, ou vários por vírgula. |
| `-Hosts` | `--hosts` | Configura só esses hosts, sem menu. |
| `-Version` | `--version` | Release específica. Padrão: a mais recente. |
| `-InstallDir` | `--install-dir` | Onde por o binário. |
| `-ReadOnly` | `--read-only` | Registra com `--read-only`. |
| `-Yes` | `--yes`, `-y` | Não pergunta nada. Exige o cofre. |
| `-NoPath` | `--no-path` | Não mexe no `PATH`. |
| — | `--force` | Reinstala mesmo já estando presente. |

Hosts aceitos: `claude-desktop`, `claude-code`, `gemini-cli`, `antigravity`, `antigravity-ide`, `codex`, `vscode`, `cursor`, `windsurf`.

Também por variável de ambiente, nos dois instaladores: `GOBSIDIAN_VAULT`, `GOBSIDIAN_VERSION`, `GOBSIDIAN_INSTALL_DIR`.

</details>

<details>
<summary>📥 <b>Binário pré-compilado ou código-fonte</b></summary>

<br>

Baixe de [Releases](https://github.com/jonyduque/Gobsidian/releases) e ponha em qualquer diretório do `PATH`. Não há serviço nem registro no sistema.

| Sistema | Arquivo |
|---|---|
| 🪟 Windows x86-64 | `gobsidian-windows-amd64.exe` |
| 🐧 Linux x86-64 | `gobsidian-linux-amd64` |
| 🍎 macOS Apple Silicon | `gobsidian-darwin-arm64` |

```bash
sha256sum -c SHA256SUMS.txt --ignore-missing
```

A partir do código, com **Go 1.25+** (piso vindo do SDK de MCP, não preferência):

```bash
go install github.com/jonyd/gobsidian/cmd/gobsidian@latest
```

</details>

> [!TIP]
> Com elevação vai para `Program Files` e o `PATH` de máquina; sem elevação, para `%LOCALAPPDATA%\Programs\gobsidian` e o `PATH` de usuário. O instalador **não** pede UAC por conta própria.

> [!NOTE]
> **Atualizando de uma versão anterior?** O formato do cache mudou, então a primeira partida reconstrói o índice em segundo plano — as tools respondem desde o primeiro segundo. Não mantenha as duas versões instaladas lado a lado: cada uma invalida o cache da outra a cada troca.

---

## ⚙️ Configuração

O instalador faz isto sozinho. O que segue é para quem prefere configurar à mão.

<details>
<summary>📝 <b>Configuração manual por host</b></summary>

<br>

**Claude Desktop** — `%APPDATA%\Claude\claude_desktop_config.json` no Windows, `~/Library/Application Support/Claude/claude_desktop_config.json` no macOS:

```json
{
  "mcpServers": {
    "gobsidian": {
      "command": "C:\\Program Files\\gobsidian\\gobsidian.exe",
      "args": ["serve", "--vault", "C:\\Meu Cofre"]
    }
  }
}
```

**Claude Code** e **Gemini CLI** registram sozinhos — a ordem dos argumentos difere:

```bash
claude mcp add gobsidian --scope user -- gobsidian serve --vault "C:\Meu Cofre"
gemini mcp add gobsidian gobsidian serve --vault "C:\Meu Cofre" --scope user
```

**VS Code**:

```bash
code --add-mcp '{"name":"gobsidian","command":"gobsidian","args":["serve","--vault","C:\\Meu Cofre"]}'
```

**Antigravity**, **Antigravity IDE**, **Cursor** e **Windsurf** usam o mesmo formato do Claude Desktop, em `~/.gemini/antigravity/mcp_config.json`, `~/.gemini/antigravity-ide/mcp_config.json`, `~/.cursor/mcp.json` e `~/.codeium/windsurf/mcp_config.json`.

> [!IMPORTANT]
> Três erros respondem pela maioria das falhas no Windows: **barra invertida simples** (JSON exige `\\`), **aspas a mais** em volta de caminho com espaço (cada `args` já é uma string), e **caminho relativo** para o binário (o host não herda o `PATH` do seu shell).

</details>

<details>
<summary>🎛️ <b>Opções de <code>serve</code></b></summary>

<br>

| Flag | Efeito |
|---|---|
| `--vault <caminho>` | Raiz do cofre. Obrigatória. |
| `--read-only` | Remove toda a superfície de escrita. |
| `--cache-dir <caminho>` | Diretório do cache. Padrão: hash do caminho do cofre, sempre **fora** dele. |
| `--debounce-ms <n>` | Janela de coalescência do watcher. |
| `--log-level <nível>` | `debug`, `info`, `warn` ou `error`. |
| `--eager-search` | Carrega o índice de busca no boot. Padrão: preguiçoso — a maioria das sessões lê e escreve sem nunca buscar. |

</details>

> [!TIP]
> Comece com `--read-only` até confiar na configuração. Você mantém busca, leitura e grafo de links, sem nenhuma tool capaz de tocar no disco.

---

## 🧰 Ferramentas MCP

Contratos completos, schemas e códigos de erro em [`docs/TOOLS.md`](docs/TOOLS.md).

**📖 Leitura**

| Tool | O que faz |
|---|---|
| `vault_search` | Busca BM25, frase exata, filtros de pasta, tag, frontmatter e data |
| `note_read` | Nota inteira, uma seção por heading, ou um bloco por `^id`; várias notas numa chamada, com sobreposição por item |
| `note_outline` | Mapa da nota: headings reais e candidatos a título de nota convertida de PDF/DOCX/EPUB |
| `note_list` | Lista por glob, pasta, tag ou consulta de frontmatter |
| `note_metadata` | Frontmatter, tags, links, backlinks, headings e blocos |
| `link_graph` | Vizinhança de links, com direção e profundidade |
| `tag_list` | Todas as tags do cofre, com contagem |
| `vault_stats` | Notas, órfãs, links quebrados e contadores do watcher |

**✏️ Escrita** — todas aceitam `dry_run` e `expected_hash`

| Tool | O que faz |
|---|---|
| `note_create` | Cria a nota, falhando se já existir |
| `note_append` | Anexa ao fim da nota ou de uma seção |
| `note_patch` | Substitui o conteúdo sob um heading ou bloco |
| `note_move` | Move ou renomeia, reescrevendo os wikilinks que apontam para ela |
| `note_delete` | Remove, com relatório prévio dos links que vão quebrar |

As notas também são publicadas como resources MCP sob `gobsidian:///<caminho>`. O esquema é próprio porque `obsidian://` pertence ao aplicativo e é registrado no sistema operacional.

---

## 💻 Linha de comando

`gobsidian` funciona fora do MCP, e é isso que torna o diagnóstico possível.

```bash
gobsidian serve   --vault "/caminho/do/cofre"                    # o que o host MCP executa
gobsidian doctor  --vault "/caminho/do/cofre"                    # diagnostica o ambiente
gobsidian index   --vault "/caminho/do/cofre" --json             # indexa e sai, com resumo
gobsidian search  "arquitetura" --vault "/caminho/do/cofre"      # busca sem host MCP
gobsidian inspect "Pasta/Nota.md" --vault "/cofre" --json        # como a nota foi interpretada
```

> [!TIP]
> 🩺 `gobsidian doctor` é o primeiro comando a rodar quando algo não funciona: cofre inacessível, permissões, arquivos OneDrive somente-nuvem, caminhos acima de 260 caracteres e colisões de caixa.

**Saída.** Colorida em terminal, texto puro quando redirecionada — `doctor > relatorio.txt` grava um arquivo limpo. A decisão é por destino, então `stdout` redirecionado não tira a cor do `stderr`. `NO_COLOR` desliga tudo. Os marcadores `[OK]`, `[!]`, `[i]`, `[*]` e `[...]` são ASCII puro e a cor só os reforça.

Em `serve`, **stdout pertence inteiro ao JSON-RPC** e todo log vai para stderr. `doctor` e `version` imprimem em stdout de propósito, porque são comandos de CLI.

---

## 🧵 Daemon

Ligado por padrão. Quando duas ou mais sessões apontam para o **mesmo cofre ao mesmo tempo** e alguma delas busca, o daemon compartilha um índice entre todas em vez de cada uma carregar a própria cópia.

Working Set agregado, cofre real de 4.513 notas:

| Sessões | Sem daemon | Com daemon | |
|---|---|---|---|
| 1 | 244,3 MB | 260,3 MB | +6,5% |
| 3 | 733,3 MB | 288,7 MB | −60,6% |
| 5 | 1.221,3 MB | 319,4 MB | −73,8% |

Uma sessão isolada — o caso mais comum — **custa 16 MB a mais**. A recomendação técnica da medição era desligar por padrão; a decisão de projeto foi embarcá-lo ligado, para que o ganho de três e cinco sessões não dependa de encontrar uma variável de ambiente. Tabela completa e o raciocínio em [`docs/OPERACAO.md`](docs/OPERACAO.md).

```bash
GOBSIDIAN_NO_DAEMON=1 gobsidian serve --vault "/caminho/do/cofre"
```

No `claude_desktop_config.json`, a variável entra no bloco `env` da entrada do servidor.

O transporte é um socket de domínio Unix local, sob o diretório de runtime do usuário, e a posse do lock de inicialização é uma trava do kernel — `flock` no Unix, `LockFileEx` no Windows —, então um daemon que morre nunca deixa o lock preso. **Se o daemon não subir, `serve` cai para o modo em processo de sempre** — o pior caso é perder o compartilhamento de memória, nunca a funcionalidade. Sem nenhuma sessão conectada, o daemon sai sozinho em 15 minutos.

---

## 🖥️ Compatibilidade

| Item | Estado |
|---|---|
| 🪟 Windows 10+ | Plataforma de primeira classe |
| 🐧 Linux (kernel 5.x+) e 🍎 macOS 13+ | Suportados; o CI roda build, `vet` e `go test -race` nos três |
| Protocolo MCP | `2025-11-25`; negociação para versões anteriores pelo SDK oficial, fixado em `v1.5.0` |
| ☁️ OneDrive, Dropbox e Google Drive | Suportados, incluindo arquivos somente-nuvem, que nunca são abertos para não disparar download |
| Caminhos acima de 260 caracteres no Windows | Suportados |

---

## 📊 Desempenho

Cofre sintético determinístico de **5.000 notas**, notebook de 12 núcleos com Windows 11, sem `-race`.

| O quê | Medido |
|---|---|
| Indexação a frio (metadados) | **500 ms** |
| Boot com cache válido | 208–282 ms sintético; 179–193 ms a 1.254 notas; 891 ms a 5.686 |
| `note_read` p95 | **345 µs** |
| `note_list` com filtro p95 | **534 µs** |
| `vault_search` p95 | **8 de 8 formatos**, pior caso 43 ms |
| Reindexação de arquivo único | **335 µs** |
| Heap vivo em repouso | 5 cofres reais, folga de 15% a 64% |
| Processos órfãos após morte do host | **0 em 400 ciclos** |

O boot escala com o tamanho do cofre: a varredura de `Stat` que confere o cache
antes de aceitá-lo é mais cara em cofre sincronizado por nuvem. A memória é
medida como **heap vivo**, e não RSS — o RSS no Go segue a meta do coletor, cerca
de 2× o heap vivo do último ciclo, e mede a política do GC em vez do dado
guardado.

A tabela dos 22 RNFs, cada um com número ou a palavra "não medido", está em
[`docs/OPERACAO.md`](docs/OPERACAO.md).

O CI compara seis benchmarks contra `docs/bench-baseline.json` e **reprova o build** em regressão acima de 20%.

---

## 🛠️ Desenvolvimento

```bash
pwsh -File scripts/verify.ps1        # bateria inteira, para no primeiro erro
pwsh -File scripts/build.ps1         # binário com versão embutida
```

<details>
<summary>🔬 <b>O que a bateria cobre</b></summary>

<br>

Em ordem, parando no primeiro erro: build, `go test -race`, tetos de latência **sem** `-race`, `go vet` nos três alvos, `gofmt`, `golangci-lint`, checagem de rede, de parâmetros de schema, de referências da documentação a artefatos que não existem, e de âncoras do README.

É um comando só porque uma lista solta convida a rodar três dos cinco.

> [!IMPORTANT]
> `golangci-lint` precisa ser **v2.12.2**. Um binário compilado com Go mais antigo que o `go.mod` recusa o config antes de analisar uma linha, e um zero local não diz nada sobre o CI.

</details>

<details>
<summary>👻 <b>O gate de processos órfãos</b> — quatro cenários, um por mecanismo</summary>

<br>

```bash
pwsh -File scripts/test_orphans.ps1 -Cycles 100                      # os quatro, que e o padrao
pwsh -File scripts/test_orphans.ps1 -Cycles 100 -Scenario parent-death
```

| Cenário | O que morre | Único mecanismo possível |
|---|---|---|
| `stdin-eof` | o host | EOF em stdin |
| `parent-death` | o host intermediário, com o pipe segurado por quem **sobrevive** | vigília do pai |
| `signal` | **nada** | `CTRL_BREAK` |
| `daemon-idle` | a única ponte | ociosidade — o daemon não tem pai nem stdin de host |

**Cada cenário desconecta os outros**, e o harness reprova se o motivo registrado não for o do mecanismo que ele nomeia. Sem isso, cair no mecanismo errado pareceria verde — foi assim que a vigília do pai atravessou marcos inteiros sem nunca ter sido exercitada.

**Órfão vazado e ciclo não medido são coisas diferentes.** Um ciclo que não chegou a lançar o processo não observou nada, nem sucesso nem vazamento, e reprovar por causa dele mede a carga da máquina. Até 2% dos ciclos podem não medir — sempre impressos, nunca silenciosos — e `-MaxNaoMedidosPct 0` exige que todos midam. Um vazamento de verdade, ou uma rodada em que **nenhum** ciclo mediu, reprova com qualquer teto.

O script **não compila**: ele recusa um `bin/gobsidian.exe` mais antigo que o código e manda rodar `build.ps1`. Sem essa guarda, um binário obsoleto passa nos cenários que não dependem do código novo e reprova nos que dependem, com uma mensagem que aponta para o lugar errado.

</details>

---

## 📚 Documentação

| Documento | Conteúdo |
|---|---|
| [`docs/PRD.md`](docs/PRD.md) | Problema, requisitos, decisões fechadas, riscos, marcos |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | Camadas, fluxos, modelo de dados, concorrência, ciclo de vida |
| [`docs/ESTRUTURA.md`](docs/ESTRUTURA.md) | Árvore de diretórios, responsabilidade de cada pacote, convenções |
| [`docs/TOOLS.md`](docs/TOOLS.md) | Contrato de cada tool: schemas de entrada e saída, códigos de erro |
| [`docs/WINDOWS.md`](docs/WINDOWS.md) | OneDrive, MAX_PATH, caixa de caminhos, fsnotify |
| [`docs/OPERACAO.md`](docs/OPERACAO.md) | Medições, tabela completa de RNFs, diagnóstico e limites conhecidos |
